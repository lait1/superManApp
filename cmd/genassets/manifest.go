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
	Canvas     ManifestCanvas               `json:"canvas"`
	Scale      int                          `json:"scale"`
	Anchors    map[string]any               `json:"anchors"`
	LayerOrder []string                     `json:"layerOrder"`
	Classes    []string                     `json:"classes"`
	RankStages []int                        `json:"rankStages"`
	Ranks      map[string]string            `json:"ranks"`
	Bodies     map[string]map[string]string `json:"bodies"`
	Scenes     map[string]string            `json:"scenes"`
	Auras      map[string]string            `json:"auras"`
	Frames     map[string]string            `json:"frames"`
	Items      []ManifestItem               `json:"items"`
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
		Bodies:     g.bodies,
		Scenes:     g.scenes,
		Auras:      g.auras,
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
