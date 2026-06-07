# assets — character art (placeholders)

Source of truth for superMen's character art pipeline. Right now the PNGs are
**deterministic placeholders** produced by a pure-Go generator (stdlib only) so
that `<CharacterCanvas/>` and the paper-doll renderer work end-to-end before the
real AI sprites land. See [`docs/12-character-design.md`](../docs/12-character-design.md)
for the art spec (style, layers, anchors, palette, palette-swap).

## What's here

| File | Role |
|------|------|
| `palette.json` | Master palette + 6 class ramps (shadow→base→light→highlight) + neutrals + rarity tints. docs/12 §3. |
| `README.md` | This file. |
| `../cmd/genassets/` | The generator (`package main`, stdlib only). |
| `../web/public/assets/character/` | **Output** — generated PNGs + `manifest.json`. Not committed art; regenerate any time. |

`palette.json` is the editable master palette. The generator also embeds an
**identical** fallback palette, so it still runs if the file is missing or
malformed (it logs a warning and uses the fallback).

## Generate

From the repo root:

```sh
go run ./cmd/genassets
```

This writes to `web/public/assets/character/` by default. Flags:

```sh
go run ./cmd/genassets -out web/public/assets/character   # output dir (default)
go run ./cmd/genassets -palette assets/palette.json       # palette source (default)
```

Output is fully **deterministic** (no RNG) — re-running produces byte-identical
files, so regeneration never churns your diff.

### What gets generated

All PNGs are drawn on the logical **128×192** canvas, aligned to the anchors in
docs/12 §5, transparent except the scene (which is the opaque backdrop):

- `body_<class>_r<stage>.png` — 6 classes × 5 rank stages = 30 chibi silhouettes
  (head + torso + arms/legs), colored from the class ramp (palette-swap, §9),
  growing / gaining a cape with the stage (§8). Selective dark outline + dithered
  ground shadow.
- `scene_<class>.png` — 6 class-themed gradient backdrops with Bayer dithering and
  a simple themed silhouette (mountains / bookshelves / arches / trees / stage / road).
- `aura_r<stage>.png` — rank aura overlay. `aura_r1` is intentionally fully
  transparent (recruit has no aura, §7); stages 2–5 add halo → colored aura →
  particle stream → legend bursts.
- `frame_r<stage>.png` — portrait frame per rank (bronze → silver → gold →
  diamond → fiery), thickening + gaining corner ornaments with rank.
- `item_<id>.png` — equipment placeholders across every slot
  (`head`, `armor`, `weapon`, `amulet`, `back`, `boots`), tinted by rarity
  (common grey … legendary orange, docs/04 §4), anchored per slot.
- `manifest.json` — the contract read by `<CharacterCanvas/>` (canvas, scale,
  anchors, layer order, classes, ranks, and the file map for every asset above).

## manifest.json contract

`<CharacterCanvas/>` reads `manifest.json` to know the canvas, anchors, layer
z-order, and which file to load for each (class, stage, slot, rank). The
generator writes it; **do not hand-edit** — change the generator and regenerate.
Shape:

```jsonc
{
  "canvas": { "w": 128, "h": 192 },
  "scale": 3,
  "anchors": { "head_center":[64,40], "chest_center":[64,96], "hand_right":[96,110], "back_mount":[64,88], "feet_baseline":176 },
  "layerOrder": ["scene","auraBack","back","body","boots","armor","arms","head","headgear","weapon","amulet","auraFront","frame"],
  "classes": ["warrior","sage","paladin","druid","bard","adventurer"],
  "rankStages": [1,2,3,4,5],
  "ranks": { "1":"recruit","2":"seeker","3":"veteran","4":"master","5":"legend" },
  "bodies": { "<class>": { "<stage>": "body_<class>_r<stage>.png" } },
  "scenes": { "<class>": "scene_<class>.png" },
  "auras":  { "<stage>": "aura_r<stage>.png" },
  "frames": { "<stage>": "frame_r<stage>.png" },
  "items":  [ { "id":"amulet_owl", "slot":"amulet", "rarity":"rare", "file":"item_amulet_owl.png" } ]
}
```

Render the layers in `layerOrder`, scaling integer-only with
`image-rendering: pixelated` (docs/12 §12).

## Replacing placeholders with real AI sprites (docs/12 §11)

These placeholders follow the same canvas + anchors + palette as the final art,
so you can swap files one at a time without touching the renderer:

0. **Prep (once):** lock the master palette (`palette.json`) and the 128×192
   anchor template (docs/12 §5). Draw the reference base sprite; train a LoRA on
   the locked reference set.
1. **Generate:** fixed style prompt + style/character reference + LoRA → concept/
   draft sprite per item.
2. **Refine (cheap — sprite is tiny):** downscale to 128×192, quantize to the
   master palette (index colors), snap to the pixel grid, drop antialiasing,
   hand-clean in Aseprite (outline, ramps, readability).
3. **Align & export:** align the layer to the anchors on the 128×192 canvas,
   export transparent PNG, **overwrite the matching `item_<id>.png` / `body_*` /
   etc.** in `web/public/assets/character/`, and (for new items) add a row to
   `shop_items` (docs/08) with `slot`/`rarity`/`effect`.

To **add a new equippable item:** run it through 1→3, drop the PNG in the output
dir, and add it to `itemCatalog` in `cmd/genassets/items.go` (or, once real art
fully replaces the generator, just add the file + a `manifest.json` `items` row
and a `shop_items` row). The renderer picks it up by slot — no renderer code
changes.

> Discipline over talent: never break the master palette or the anchor grid.
> That is what keeps the character visually coherent across classes and ranks.
