package main

import (
	"image/color"
)

// ---------------------------------------------------------------------------
// Items — equippable paper-doll layers (docs/04 §3 slots, §4 rarity; docs/12
// §5 z-order + anchors). Each item is drawn on the full 128x192 canvas aligned
// to its slot's anchor, tinted by rarity (common grey ... legendary orange),
// transparent elsewhere. Slots used by the manifest contract:
//   head, armor, weapon, amulet, back, boots.
//
// The id/slot/rarity/file rows here mirror what `shop_items` would carry
// (docs/08); a few are taken verbatim from docs/04 §5 (e.g. amulet_owl rare).
// ---------------------------------------------------------------------------

// itemSpec describes one placeholder equipment piece.
type itemSpec struct {
	ID     string
	Slot   string
	Rarity string
}

// itemCatalog is the deterministic placeholder set, covering every slot and a
// spread of rarities. Order here is the manifest order (kept stable).
var itemCatalog = []itemSpec{
	// head
	{"helm_iron", "head", "common"},
	{"hood_seeker", "head", "uncommon"},
	{"crown_sage", "head", "epic"},
	// armor
	{"vest_padded", "armor", "common"},
	{"armor_titan", "armor", "legendary"}, // docs/04 §5 — quest legendary
	// weapon
	{"sword_short", "weapon", "common"},
	{"blade_focus", "weapon", "epic"}, // docs/04 §7 — "Клинок Фокуса"
	{"staff_arcane", "weapon", "rare"},
	// amulet
	{"amulet_owl", "amulet", "rare"}, // docs/04 §5 — "Амулет Совы" +INT
	{"amulet_sun", "amulet", "legendary"},
	// back
	{"cloak_traveler", "back", "uncommon"},
	{"wings_phoenix", "back", "legendary"},
	// boots
	{"boots_leather", "boots", "common"},
	{"boots_swift", "boots", "rare"},
}

func (g *generator) genItems() {
	for _, it := range itemCatalog {
		c := newCanvas()
		ramp := g.pal.rarity(it.Rarity)
		g.drawItem(c, it.Slot, ramp)
		// rarity glow stud at top so even subtle items read their tier
		c.fillBlock(snap(canvasW-block*2), block, ramp[toneHigh])

		file := "item_" + it.ID + ".png"
		g.writePNG(file, c)
		g.items = append(g.items, ManifestItem{
			ID:     it.ID,
			Slot:   it.Slot,
			Rarity: it.Rarity,
			File:   file,
		})
	}
}

// drawItem paints a slot-appropriate placeholder anchored per docs/12 §5.
func (g *generator) drawItem(c *canvas, slot string, ramp Ramp) {
	switch slot {
	case "head":
		g.drawHeadgear(c, ramp)
	case "armor":
		g.drawArmor(c, ramp)
	case "weapon":
		g.drawWeapon(c, ramp)
	case "amulet":
		g.drawAmulet(c, ramp)
	case "back":
		g.drawBack(c, ramp)
	case "boots":
		g.drawBoots(c, ramp)
	}
	c.selectiveOutline(g.pal.Outline)
}

// head: helmet cap over the top of the head, centered on head_center.
func (g *generator) drawHeadgear(c *canvas, ramp Ramp) {
	hc := anchorHeadCenter
	r := 24
	// upper dome of the head region
	c.ellipseBlocksFunc(hc.x, hc.y-block, r, r, func(bx, by int) (color.RGBA, bool) {
		if by > hc.y-block { // only the top half = a cap, not the face
			return color.RGBA{}, false
		}
		frac := float64(by-(hc.y-block-r)) / float64(r)
		idx := int((1 - frac) * 3.999)
		if idx < 0 {
			idx = 0
		}
		if idx > 3 {
			idx = 3
		}
		return ramp[idx], true
	})
	// brow band
	c.rectBlocks(hc.x-r+block, hc.y-block, hc.x+r-block, hc.y, ramp[toneShadow])
	// crest stud
	c.fillBlock(hc.x-block/2, hc.y-block-r, ramp[toneHigh])
}

// armor: chest plate centered on chest_center.
func (g *generator) drawArmor(c *canvas, ramp Ramp) {
	cc := anchorChestCenter
	w, top, bot := 26, cc.y-30, cc.y+24
	c.rectBlocksRamp(cc.x-w, top, cc.x+w, bot, ramp)
	// pauldrons
	c.rectBlocks(cc.x-w-block, top, cc.x-w+block*2, top+block*3, ramp[toneLight])
	c.rectBlocks(cc.x+w-block*2, top, cc.x+w+block, top+block*3, ramp[toneLight])
	// central emblem at chest center
	c.fillBlock(cc.x-block, cc.y-block, ramp[toneHigh])
	c.fillBlock(cc.x, cc.y-block, ramp[toneHigh])
	c.fillBlock(cc.x-block, cc.y, ramp[toneHigh])
	c.fillBlock(cc.x, cc.y, ramp[toneHigh])
	// belt line
	c.rectBlocks(cc.x-w, bot-block, cc.x+w, bot, ramp[toneShadow])
}

// weapon: a sword/staff held in the right hand at hand_right.
func (g *generator) drawWeapon(c *canvas, ramp Ramp) {
	h := anchorHandRight
	// grip in the hand
	wood := g.pal.neutral("wood")
	c.rectBlocks(h.x-block, h.y-block, h.x+block, h.y+block*4, wood[toneBase])
	// blade/shaft going up
	bladeTop := h.y - 70
	c.rectBlocksRamp(h.x-block, bladeTop, h.x+block, h.y-block, ramp)
	// crossguard
	c.rectBlocks(h.x-block*2, h.y-block*2, h.x+block*2, h.y-block, ramp[toneShadow])
	// pommel gem
	c.fillBlock(h.x-block/2, h.y+block*4, ramp[toneHigh])
	// blade tip highlight
	c.fillBlock(h.x-block/2, bladeTop, ramp[toneHigh])
}

// amulet: pendant on the chest at chest_center.
func (g *generator) drawAmulet(c *canvas, ramp Ramp) {
	cc := anchorChestCenter
	// chain (two short diagonals up toward the neck)
	chain := g.pal.neutral("metal")[toneLight]
	for i := 0; i < 6; i++ {
		c.fillBlock(cc.x-block*3+i*block/2, cc.y-block*4+i*block/2, chain)
		c.fillBlock(cc.x+block*3-i*block/2, cc.y-block*4+i*block/2, chain)
	}
	// gem
	c.ellipseBlocks(cc.x, cc.y, block*3, block*3, ramp[toneBase])
	c.ellipseBlocks(cc.x, cc.y-block, block, block, ramp[toneHigh])                           // sparkle
	c.circleRingBlocks(cc.x, cc.y, block*3+block, block*3, g.pal.neutral("metal")[toneLight]) // bezel
}

// back: cloak/wings mounted at back_mount, hanging behind the body.
func (g *generator) drawBack(c *canvas, ramp Ramp) {
	bm := anchorBackMount
	top := bm.y
	bot := anchorFeetBaseline - block
	for y := snap(top); y < bot; y += block {
		frac := float64(y-top) / float64(bot-top)
		w := int(14 + frac*22)
		idx := toneBase
		if checker(0, y) {
			idx = toneShadow
		}
		c.rectBlocks(bm.x-w, y, bm.x+w, y+block, ramp[idx])
	}
	// collar at the mount
	c.rectBlocks(bm.x-block*3, top-block, bm.x+block*3, top+block, ramp[toneLight])
}

// boots: footwear sitting on the feet baseline, two feet.
func (g *generator) drawBoots(c *canvas, ramp Ramp) {
	base := anchorFeetBaseline
	cx := canvasW / 2
	gap := block
	w := block * 4
	top := base - block*5
	// left boot
	c.rectBlocksRamp(cx-gap-w, top, cx-gap, base, ramp)
	c.rectBlocks(cx-gap-w-block, base-block*2, cx-gap, base, ramp[toneShadow]) // toe
	// right boot
	c.rectBlocksRamp(cx+gap, top, cx+gap+w, base, ramp)
	c.rectBlocks(cx+gap, base-block*2, cx+gap+w+block, base, ramp[toneShadow]) // toe
	// cuffs
	c.rectBlocks(cx-gap-w, top, cx-gap, top+block, ramp[toneHigh])
	c.rectBlocks(cx+gap, top, cx+gap+w, top+block, ramp[toneHigh])
}
