// Command genassets generates deterministic placeholder pixel-art sprites for
// superMen's CharacterCanvas (see docs/12-character-design.md).
//
// These are NOT the final AI sprites — they are valid, anchor-aligned PNG
// placeholders so the paper-doll renderer works end-to-end before real art
// lands (replacement pipeline: docs/12 §11, also assets/README.md).
//
// Everything is drawn on the logical 128x192 canvas, aligned to the anchors
// from docs/12 §5, in the master palette / class ramps from docs/12 §3
// (assets/palette.json, with an identical in-code fallback). Output is a set
// of transparent PNGs + a manifest.json under web/public/assets/character/.
//
// Usage:
//
//	go run ./cmd/genassets            # writes to web/public/assets/character
//	go run ./cmd/genassets -out DIR   # custom output dir
//	go run ./cmd/genassets -palette assets/palette.json
//
// stdlib only.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// ---------------------------------------------------------------------------
// Canvas + anchors (docs/12 §2, §5) — these MUST match manifest.json exactly.
// ---------------------------------------------------------------------------

const (
	canvasW = 128
	canvasH = 192
	scale   = 3

	// "Big pixel" block size — we draw on a coarse grid so output reads as
	// pixel-art even though the PNG itself is 128x192. block must divide both
	// canvas dims (128 = 4*32, 192 = 4*48): 4 works.
	block = 4
)

// Anchor points in canvas pixel coordinates (docs/12 §5).
var (
	anchorHeadCenter   = pt{64, 40}
	anchorChestCenter  = pt{64, 96}
	anchorHandRight    = pt{96, 110}
	anchorBackMount    = pt{64, 88}
	anchorFeetBaseline = 176
)

type pt struct{ x, y int }

// Classes, ranks, stages — docs/01 §4, §6 and the manifest contract.
var classes = []string{"warrior", "sage", "paladin", "druid", "bard", "adventurer"}

var rankStages = []int{1, 2, 3, 4, 5}

var rankNames = map[int]string{
	1: "recruit",
	2: "seeker",
	3: "veteran",
	4: "master",
	5: "legend",
}

// Appearance dimensions — the onboarding customization axes. Ids MUST match
// domain.BodyTypes / SkinTones / Hairstyles / HairColors (backend validation).
var (
	bodyTypes  = []string{"a", "b"} // a = sturdy, b = slim
	skinTones  = []string{"s1", "s2", "s3", "s4"}
	hairstyles = []string{"bald", "short", "spiky", "long", "ponytail"} // bald emits no file
	hairColors = []string{"dark", "brown", "blond", "red"}
)

// defaultAppearance mirrors domain.DefaultAppearance for the manifest.
var defaultAppearance = map[string]string{
	"bodyType":  "a",
	"skinTone":  "s2",
	"hairstyle": "short",
	"hairColor": "dark",
}

// Layer z-order — docs/12 §5 / §12. The former monolithic "body" layer is
// decomposed for customization: body (skin: head/arms/legs by bodyType ×
// skinTone), outfit (class costume by stage), hair (style × color).
var layerOrder = []string{
	"scene", "auraBack", "back", "body", "outfit", "boots", "armor",
	"arms", "head", "hair", "headgear", "weapon", "amulet", "auraFront", "frame",
}

func main() {
	outDir := flag.String("out", filepath.FromSlash("web/public/assets/character"), "output directory for generated PNGs + manifest.json")
	paletteFile := flag.String("palette", filepath.FromSlash("assets/palette.json"), "path to palette.json (falls back to embedded palette if missing)")
	flag.Parse()

	pal := loadPalette(*paletteFile)

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "mkdir out:", err)
		os.Exit(1)
	}

	g := &generator{dir: *outDir, pal: pal}

	g.genSkins()
	g.genBlinks()
	g.genOutfits()
	g.genHair()
	g.genScenes()
	g.genAuras()
	g.genFrames()
	g.genItems()
	g.writeManifest()

	fmt.Printf("genassets: wrote %d files to %s\n", g.fileCount, *outDir)
}

// generator carries shared state while emitting assets.
type generator struct {
	dir       string
	pal       Palette
	fileCount int

	// collected for manifest
	skins      map[string]map[string]map[string]string // bodyType -> stage -> tone -> file
	blinks     map[string]string                       // skinTone -> eyelid overlay file
	outfits    map[string]map[string]map[string]string // class -> stage -> bodyType -> file
	hair       map[string]map[string]string            // style -> color -> file
	scenes     map[string]string                       // class -> file
	auras      map[string]string                       // stage -> back-glow file
	aurasFront map[string]string                       // stage -> front sparks file
	frames     map[string]string                       // stage -> file
	items      []ManifestItem
}
