package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
)

// ---------------------------------------------------------------------------
// Palette model — mirrors assets/palette.json (docs/12 §3).
// A "ramp" is 4 tones: shadow -> base -> light -> highlight.
// ---------------------------------------------------------------------------

// Ramp is the 4-tone ladder used everywhere: [0]=shadow [1]=base [2]=light [3]=highlight.
type Ramp [4]color.RGBA

const (
	toneShadow = 0
	toneBase   = 1
	toneLight  = 2
	toneHigh   = 3
)

type classRampJSON struct {
	Shadow    string `json:"shadow"`
	Base      string `json:"base"`
	Light     string `json:"light"`
	Highlight string `json:"highlight"`
}

type paletteJSON struct {
	Neutrals     map[string][]string      `json:"neutrals"`
	ClassRamps   map[string]classRampJSON `json:"classRamps"`
	RarityColors map[string][]string      `json:"rarityColors"`
	SkinTones    map[string][]string      `json:"skinTones"`
	HairColors   map[string][]string      `json:"hairColors"`
}

// Palette is the resolved (parsed) palette used by the generator.
type Palette struct {
	Neutrals   map[string]Ramp
	ClassRamps map[string]Ramp
	Rarity     map[string]Ramp
	SkinTones  map[string]Ramp
	HairColors map[string]Ramp

	Outline      color.RGBA
	ShadowGround color.RGBA
	White        color.RGBA
}

// classRamp returns the ramp for a class (always present for the 6 known classes).
func (p Palette) classRamp(class string) Ramp {
	if r, ok := p.ClassRamps[class]; ok {
		return r
	}
	return p.ClassRamps["adventurer"]
}

func (p Palette) neutral(name string) Ramp {
	if r, ok := p.Neutrals[name]; ok {
		return r
	}
	return p.Neutrals["cloth"]
}

func (p Palette) rarity(name string) Ramp {
	if r, ok := p.Rarity[name]; ok {
		return r
	}
	return p.Rarity["common"]
}

func (p Palette) skinTone(name string) Ramp {
	if r, ok := p.SkinTones[name]; ok {
		return r
	}
	return p.neutral("skin")
}

func (p Palette) hairColor(name string) Ramp {
	if r, ok := p.HairColors[name]; ok {
		return r
	}
	return p.neutral("leather")
}

// loadPalette reads palette.json; on any error it logs and returns the embedded
// fallback so the generator is never blocked.
func loadPalette(path string) Palette {
	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "genassets: palette %q unreadable (%v) — using embedded fallback\n", path, err)
		return embeddedPalette()
	}
	var pj paletteJSON
	if err := json.Unmarshal(raw, &pj); err != nil {
		fmt.Fprintf(os.Stderr, "genassets: palette %q invalid JSON (%v) — using embedded fallback\n", path, err)
		return embeddedPalette()
	}
	p := Palette{
		Neutrals:   map[string]Ramp{},
		ClassRamps: map[string]Ramp{},
		Rarity:     map[string]Ramp{},
		SkinTones:  map[string]Ramp{},
		HairColors: map[string]Ramp{},
	}
	for name, hexes := range pj.Neutrals {
		p.Neutrals[name] = rampFromHexes(hexes)
	}
	for name, hexes := range pj.SkinTones {
		p.SkinTones[name] = rampFromHexes(hexes)
	}
	for name, hexes := range pj.HairColors {
		p.HairColors[name] = rampFromHexes(hexes)
	}
	for class, r := range pj.ClassRamps {
		p.ClassRamps[class] = Ramp{
			mustHex(r.Shadow), mustHex(r.Base), mustHex(r.Light), mustHex(r.Highlight),
		}
	}
	for name, hexes := range pj.RarityColors {
		if name == "_comment" {
			continue
		}
		p.Rarity[name] = rampFromHexes(hexes)
	}
	// Single-color neutrals collapse to tone[0].
	p.Outline = p.neutral("outline")[0]
	p.ShadowGround = p.neutral("shadowGround")[0]
	p.White = p.neutral("white")[0]

	// Backfill anything missing from the embedded palette so we never panic.
	fb := embeddedPalette()
	for name, r := range fb.Neutrals {
		if _, ok := p.Neutrals[name]; !ok {
			p.Neutrals[name] = r
		}
	}
	for c, r := range fb.ClassRamps {
		if _, ok := p.ClassRamps[c]; !ok {
			p.ClassRamps[c] = r
		}
	}
	for name, r := range fb.Rarity {
		if _, ok := p.Rarity[name]; !ok {
			p.Rarity[name] = r
		}
	}
	for name, r := range fb.SkinTones {
		if _, ok := p.SkinTones[name]; !ok {
			p.SkinTones[name] = r
		}
	}
	for name, r := range fb.HairColors {
		if _, ok := p.HairColors[name]; !ok {
			p.HairColors[name] = r
		}
	}
	if (p.Outline == color.RGBA{}) {
		p.Outline = fb.Outline
	}
	if (p.ShadowGround == color.RGBA{}) {
		p.ShadowGround = fb.ShadowGround
	}
	if (p.White == color.RGBA{}) {
		p.White = fb.White
	}
	return p
}

// rampFromHexes turns a slice of hex strings into a 4-tone Ramp. Shorter slices
// are repeated; a single color fills all four tones.
func rampFromHexes(hexes []string) Ramp {
	var r Ramp
	if len(hexes) == 0 {
		return r
	}
	for i := 0; i < 4; i++ {
		r[i] = mustHex(hexes[i%len(hexes)])
	}
	return r
}

// mustHex parses #RRGGBB (alpha forced opaque). Bad input -> opaque magenta so
// the problem is visible rather than silent.
func mustHex(s string) color.RGBA {
	c, ok := parseHex(s)
	if !ok {
		return color.RGBA{R: 255, G: 0, B: 255, A: 255}
	}
	return c
}

func parseHex(s string) (color.RGBA, bool) {
	if len(s) == 7 && s[0] == '#' {
		s = s[1:]
	}
	if len(s) != 6 {
		return color.RGBA{}, false
	}
	var v [3]int
	for i := 0; i < 3; i++ {
		hi, ok1 := hexNibble(s[i*2])
		lo, ok2 := hexNibble(s[i*2+1])
		if !ok1 || !ok2 {
			return color.RGBA{}, false
		}
		v[i] = hi*16 + lo
	}
	return color.RGBA{R: uint8(v[0]), G: uint8(v[1]), B: uint8(v[2]), A: 255}, true
}

func hexNibble(b byte) (int, bool) {
	switch {
	case b >= '0' && b <= '9':
		return int(b - '0'), true
	case b >= 'a' && b <= 'f':
		return int(b-'a') + 10, true
	case b >= 'A' && b <= 'F':
		return int(b-'A') + 10, true
	}
	return 0, false
}

// embeddedPalette is the in-code fallback. It is value-identical to
// assets/palette.json so generation is reproducible without the file.
func embeddedPalette() Palette {
	mk := func(hexes ...string) Ramp { return rampFromHexes(hexes) }
	p := Palette{
		Neutrals: map[string]Ramp{
			"skin":         mk("#7A4A33", "#B07A52", "#D8A878", "#F0D2A8"),
			"metal":        mk("#3A3F4A", "#6B7280", "#9CA3AF", "#E5E7EB"),
			"leather":      mk("#3A2516", "#5C3A1E", "#8A5A2E", "#B98A52"),
			"wood":         mk("#3A2A18", "#5E4326", "#8A6438", "#B98E5C"),
			"cloth":        mk("#4A4A52", "#6E6E78", "#9A9AA4", "#C8C8D0"),
			"outline":      mk("#14131A"),
			"shadowGround": mk("#0A0A12"),
			"white":        mk("#F4F4F8"),
		},
		ClassRamps: map[string]Ramp{
			"warrior":    mk("#5A1410", "#A8281C", "#E2562C", "#FFB070"),
			"sage":       mk("#1B2A6B", "#2748A8", "#3B6FE4", "#9EC2FF"),
			"paladin":    mk("#7A5410", "#C8941C", "#F0C84A", "#FFF2C0"),
			"druid":      mk("#10402E", "#1E8A5A", "#34C088", "#9CF0CC"),
			"bard":       mk("#4A1052", "#9A1EA0", "#D63CC8", "#FFA0EE"),
			"adventurer": mk("#4A2E12", "#9A5E22", "#D68A34", "#F4C880"),
		},
		Rarity: map[string]Ramp{
			"common":    mk("#4A4A52", "#7A7A84", "#A8A8B2", "#D8D8E0"),
			"uncommon":  mk("#16602E", "#2A9A4A", "#48C870", "#A8F0BC"),
			"rare":      mk("#163A78", "#2A66C8", "#4A92F0", "#AECCFF"),
			"epic":      mk("#48166A", "#8A2EC8", "#B45CF0", "#E0AEFF"),
			"legendary": mk("#7A3A08", "#D6741C", "#F4A030", "#FFE0A0"),
		},
		SkinTones: map[string]Ramp{
			"s1": mk("#9C6B53", "#E3B189", "#F1C9A5", "#FBE2C8"),
			"s2": mk("#7A4A33", "#B07A52", "#D8A878", "#F0D2A8"),
			"s3": mk("#5C3424", "#8A5638", "#B07A52", "#D2A380"),
			"s4": mk("#3E2218", "#5F3B2A", "#7A4E36", "#9A6A4A"),
		},
		HairColors: map[string]Ramp{
			"dark":  mk("#14131A", "#2A2630", "#433D4D", "#5E5870"),
			"brown": mk("#3A2516", "#5C3A1E", "#8A5A2E", "#B98A52"),
			"blond": mk("#8A6420", "#C89A3C", "#E8C465", "#F8E6A0"),
			"red":   mk("#5A1410", "#A8281C", "#D6562C", "#F08A50"),
		},
	}
	p.Outline = mustHex("#14131A")
	p.ShadowGround = mustHex("#0A0A12")
	p.White = mustHex("#F4F4F8")
	return p
}
