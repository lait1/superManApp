package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ---------------------------------------------------------------------------
// manifest.json — the contract between this generator and <CharacterCanvas/>.
// Shape and key names are fixed (see task spec); fields are emitted in the
// documented order so the file diffs cleanly run-to-run.
// ---------------------------------------------------------------------------

type ManifestCanvas struct {
	W int `json:"w"`
	H int `json:"h"`
}

type ManifestItem struct {
	ID     string `json:"id"`
	Slot   string `json:"slot"`
	Rarity string `json:"rarity"`
	File   string `json:"file"`
}

type Manifest struct {
	Canvas     ManifestCanvas    `json:"canvas"`
	Scale      int               `json:"scale"`
	Anchors    map[string]any    `json:"anchors"`
	LayerOrder []string          `json:"layerOrder"`
	Classes    []string          `json:"classes"`
	RankStages []int             `json:"rankStages"`
	Ranks      map[string]string `json:"ranks"`

	// Appearance axes + the look used before onboarding completes.
	BodyTypes  []string          `json:"bodyTypes"`
	SkinTones  []string          `json:"skinTones"`
	Hairstyles []string          `json:"hairstyles"`
	HairColors []string          `json:"hairColors"`
	Defaults   map[string]string `json:"defaults"`

	// Base layers: skin (bodyType → stage → tone), outfit (class → stage →
	// bodyType) and hair (style → color). "bald" has no hair entry.
	Skins   map[string]map[string]map[string]string `json:"skins"`
	Outfits map[string]map[string]map[string]string `json:"outfits"`
	Hair    map[string]map[string]string            `json:"hair"`
	// Blinks: skinTone → eyelid overlay, flashed over the skin layer by the
	// renderer for the idle blink (docs/12 §10).
	Blinks map[string]string `json:"blinks"`

	Scenes map[string]string `json:"scenes"`
	// Auras is the soft back glow (z auraBack); AurasFront holds the sparks
	// drawn over the figure (z auraFront) so the silhouette stays readable.
	Auras      map[string]string `json:"auras"`
	AurasFront map[string]string `json:"aurasFront"`
	Frames     map[string]string `json:"frames"`
	Items      []ManifestItem    `json:"items"`
}

func (g *generator) writeManifest() {
	m := Manifest{
		Canvas: ManifestCanvas{W: canvasW, H: canvasH},
		Scale:  scale,
		Anchors: map[string]any{
			"head_center":   []int{anchorHeadCenter.x, anchorHeadCenter.y},
			"chest_center":  []int{anchorChestCenter.x, anchorChestCenter.y},
			"hand_right":    []int{anchorHandRight.x, anchorHandRight.y},
			"back_mount":    []int{anchorBackMount.x, anchorBackMount.y},
			"feet_baseline": anchorFeetBaseline,
		},
		LayerOrder: layerOrder,
		Classes:    classes,
		RankStages: rankStages,
		Ranks:      ranksMap(),
		BodyTypes:  bodyTypes,
		SkinTones:  skinTones,
		Hairstyles: hairstyles,
		HairColors: hairColors,
		Defaults:   defaultAppearance,
		Skins:      g.skins,
		Outfits:    g.outfits,
		Hair:       g.hair,
		Blinks:     g.blinks,
		Scenes:     g.scenes,
		Auras:      g.auras,
		AurasFront: g.aurasFront,
		Frames:     g.frames,
		Items:      g.items,
	}

	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "marshal manifest:", err)
		os.Exit(1)
	}
	raw = append(raw, '\n')

	path := filepath.Join(g.dir, "manifest.json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write manifest:", err)
		os.Exit(1)
	}
	g.fileCount++
}

func ranksMap() map[string]string {
	out := map[string]string{}
	for k, v := range rankNames {
		out[fmt.Sprintf("%d", k)] = v
	}
	return out
}
