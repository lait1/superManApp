package main

import (
	"fmt"
	"image/color"
)

// ---------------------------------------------------------------------------
// Auras (z1/z11) and Frames (z12) — rank progress channel #1 (docs/12 §7).
// Both are transparent overlays sized to the full 128x192 canvas.
//
// Aura intensity grows with stage:
//   stage 1 recruit — NO aura file (table: "нет"). We still emit a fully
//                     transparent aura_r1.png so the manifest can map every
//                     stage uniformly and the renderer never 404s.
//   2 seeker  — soft halo (dithered)
//   3 veteran — colored aura (class ramp) + sparks
//   4 master  — stronger pulsing-style aura + particle stream
//   5 legend  — full-canvas glow + bursts
// Aura color is class-neutral here (white/warm) so it composits over any class;
// the renderer can tint per class at runtime if desired.
// ---------------------------------------------------------------------------

func (g *generator) genAuras() {
	g.auras = map[string]string{}
	for _, stage := range rankStages {
		c := newCanvas()
		g.drawAura(c, stage)
		name := fmt.Sprintf("aura_r%d.png", stage)
		g.writePNG(name, c)
		g.auras[fmt.Sprintf("%d", stage)] = name
	}
}

// auraGlow is a warm near-white glow color used for the radial body.
var auraGlow = color.RGBA{R: 255, G: 238, B: 200, A: 255}

func (g *generator) drawAura(c *canvas, stage int) {
	if stage <= 1 {
		return // recruit: transparent placeholder, intentionally empty
	}
	cx := canvasW / 2
	cy := anchorChestCenter.y

	// base radial halo strength per stage
	var maxA int
	var rx, ry int
	switch stage {
	case 2:
		maxA, rx, ry = 70, 40, 64
	case 3:
		maxA, rx, ry = 110, 48, 76
	case 4:
		maxA, rx, ry = 150, 56, 88
	default: // 5
		maxA, rx, ry = 190, 64, 96
	}

	// radial dithered glow centered on the chest, falling off with distance
	for by := 0; by < canvasH; by += block {
		for bx := 0; bx < canvasW; bx += block {
			dx := float64(bx+block/2-cx) / float64(rx)
			dy := float64(by+block/2-cy) / float64(ry)
			d := dx*dx + dy*dy
			if d > 1 {
				continue
			}
			a := float64(maxA) * (1 - d) // linear falloff
			// ordered dithering on the alpha to avoid banding
			a += float64(bayer4(bx, by) - 8)
			if a <= 0 {
				continue
			}
			c.blend(bx, by, withAlpha(auraGlow, uint8(min(a, 255))))
		}
	}

	// sparks / particles for stage >= 3 — deterministic positions around a ring
	if stage >= 3 {
		spark := color.RGBA{R: 255, G: 250, B: 230, A: 235}
		n := 6 + (stage-3)*4 // 6, 10, 14
		ringRx, ringRy := rx+block*2, ry+block*2
		for i := 0; i < n; i++ {
			// deterministic angle via integer table (no RNG, no math import)
			ang := float64(i) / float64(n) * 6.2831853
			sx := cx + int(float64(ringRx)*cosApprox(ang))
			sy := cy + int(float64(ringRy)*sinApprox(ang))
			c.fillBlock(snap(sx), snap(sy), spark)
			if stage >= 4 {
				// trailing particle stream pointing up
				c.set(sx+1, sy-block, withAlpha(spark, 160))
			}
		}
	}

	// legend bursts: bright cross-flashes at the corners of the halo
	if stage >= 5 {
		flash := color.RGBA{R: 255, G: 255, B: 245, A: 255}
		for _, off := range []pt{{0, -ry}, {0, ry}, {-rx, 0}, {rx, 0}} {
			fx, fy := snap(cx+off.x), snap(cy+off.y)
			c.fillBlock(fx, fy, flash)
			c.fillBlock(fx-block, fy, withAlpha(flash, 160))
			c.fillBlock(fx+block, fy, withAlpha(flash, 160))
			c.fillBlock(fx, fy-block, withAlpha(flash, 160))
			c.fillBlock(fx, fy+block, withAlpha(flash, 160))
		}
	}
}

// ---------------------------------------------------------------------------
// Frames — z12 portrait frame per rank (docs/12 §7). A border that gains
// thickness, corner ornaments, and color with the stage.
// ---------------------------------------------------------------------------

func (g *generator) genFrames() {
	g.frames = map[string]string{}
	for _, stage := range rankStages {
		c := newCanvas()
		g.drawFrame(c, stage)
		name := fmt.Sprintf("frame_r%d.png", stage)
		g.writePNG(name, c)
		g.frames[fmt.Sprintf("%d", stage)] = name
	}
}

// frame colors per stage: bronze -> silver -> gold -> diamond-white -> fiery.
func frameColor(stage int) (border, accent color.RGBA) {
	switch stage {
	case 1:
		return color.RGBA{0x8A, 0x5A, 0x2E, 255}, color.RGBA{0xB9, 0x8A, 0x52, 255} // bronze
	case 2:
		return color.RGBA{0x9C, 0xA3, 0xAF, 255}, color.RGBA{0xE5, 0xE7, 0xEB, 255} // silver
	case 3:
		return color.RGBA{0xC8, 0x94, 0x1C, 255}, color.RGBA{0xF0, 0xC8, 0x4A, 255} // gold
	case 4:
		return color.RGBA{0x9E, 0xC2, 0xFF, 255}, color.RGBA{0xF4, 0xF4, 0xF8, 255} // diamond
	default:
		return color.RGBA{0xE2, 0x56, 0x2C, 255}, color.RGBA{0xFF, 0xB0, 0x70, 255} // fiery legend
	}
}

func (g *generator) drawFrame(c *canvas, stage int) {
	border, accent := frameColor(stage)
	thickness := block * (1 + stage/2) // 1,1,2,3,3 blocks roughly
	if thickness < block {
		thickness = block
	}

	// outer border ring (block thickness rings inward)
	for t := 0; t < thickness; t += block {
		col := border
		if t == 0 {
			col = g.pal.Outline // dark outer edge
		} else if (t/block)%2 == 1 {
			col = accent
		}
		// top & bottom bars
		c.rectBlocks(t, t, canvasW-t, t+block, col)
		c.rectBlocks(t, canvasH-t-block, canvasW-t, canvasH-t, col)
		// left & right bars
		c.rectBlocks(t, t, t+block, canvasH-t, col)
		c.rectBlocks(canvasW-t-block, t, canvasW-t, canvasH-t, col)
	}

	// corner ornaments grow with rank (docs/12 §7: simple->узорная->орнамент)
	if stage >= 2 {
		orn := accent
		size := block * stage // bigger flourish for higher ranks
		corners := []pt{{0, 0}, {canvasW - size, 0}, {0, canvasH - size}, {canvasW - size, canvasH - size}}
		for _, cr := range corners {
			for i := 0; i < stage; i++ {
				c.fillBlock(cr.x+i*block, cr.y, orn)
				c.fillBlock(cr.x, cr.y+i*block, orn)
			}
		}
	}

	// legend frame: little flame studs along the top bar
	if stage >= 5 {
		flame := accent
		for x := block * 6; x < canvasW-block*5; x += block * 6 {
			c.fillBlock(x, block, flame)
			c.fillBlock(x, 0, withAlpha(flame, 180))
		}
	}
}

// ---------------------------------------------------------------------------
// tiny deterministic trig approximations (avoid importing math for a few sparks).
// Good enough for placing sparks on a ring. Domain: any real radians.
// ---------------------------------------------------------------------------

func cosApprox(x float64) float64 { return sinApprox(x + 1.5707963) }

// sinApprox: Bhaskara I-style approximation, range-reduced to [-pi, pi].
func sinApprox(x float64) float64 {
	const pi = 3.14159265358979
	const twoPi = 2 * pi
	// reduce to [-pi, pi]
	for x > pi {
		x -= twoPi
	}
	for x < -pi {
		x += twoPi
	}
	neg := false
	if x < 0 {
		x = -x
		neg = true
	}
	// Bhaskara on [0, pi]
	r := (16 * x * (pi - x)) / (5*pi*pi - 4*x*(pi-x))
	if neg {
		return -r
	}
	return r
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
