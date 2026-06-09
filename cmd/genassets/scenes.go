package main

import (
	"fmt"
	"image/color"
)

// ---------------------------------------------------------------------------
// Scenes — z0 background per class (docs/12 §9). A vertical class-themed
// gradient with ordered (Bayer) dithering so it reads pixel-art, plus a couple
// of simple themed silhouettes (horizon / arches / trees) to hint the locale.
// Opaque (it is the backdrop).
// ---------------------------------------------------------------------------

func (g *generator) genScenes() {
	g.scenes = map[string]string{}
	for _, class := range classes {
		c := newCanvas()
		g.drawScene(c, class)
		name := fmt.Sprintf("scene_%s.png", class)
		g.writePNG(name, c)
		g.scenes[class] = name
	}
}

func (g *generator) drawScene(c *canvas, class string) {
	ramp := g.pal.classRamp(class)
	// Gradient: shadow at top -> base mid -> light glow at the bottom horizon.
	top := ramp[toneShadow]
	mid := ramp[toneBase]
	low := ramp[toneLight]

	// Quantized vertical gradient with ordered dithering at the band seams —
	// the classic pixel-art sky (no per-block noise grid).
	stops := []color.RGBA{
		top,
		lerpColor(top, mid, 0.5),
		mid,
		lerpColor(mid, low, 0.33),
		lerpColor(mid, low, 0.66),
		low,
	}
	for by := 0; by < canvasH; by += block {
		for bx := 0; bx < canvasW; bx += block {
			frac := float64(by) / float64(canvasH) // 0 top .. 1 bottom
			pos := frac * float64(len(stops)-1)
			i := int(pos)
			if i >= len(stops)-1 {
				i = len(stops) - 2
			}
			fr := pos - float64(i)
			col := stops[i]
			if fr > (float64(bayer4(bx, by))+0.5)/16.0 {
				col = stops[i+1]
			}
			c.fillBlock(bx, by, withAlpha(col, 255))
		}
	}

	// themed silhouette near the horizon (docs/12 §9 scene themes)
	horizon := anchorFeetBaseline - block*2
	sil := ramp[toneShadow]
	sil = withAlpha(color.RGBA{R: scale8(sil.R, 0.6), G: scale8(sil.G, 0.6), B: scale8(sil.B, 0.6), A: 255}, 255)

	switch class {
	case "warrior": // arena / mountains — jagged peaks
		for i := 0; i < 5; i++ {
			peakX := block*8 + i*block*6
			peakH := block * (6 + (i%3)*3)
			for y := 0; y < peakH; y += block {
				half := (peakH - y) / 2
				c.rectBlocks(peakX-half, horizon-y, peakX+half+block, horizon-y+block, sil)
			}
		}
	case "sage": // library — bookshelf columns
		for x := block; x < canvasW; x += block * 6 {
			c.rectBlocks(x, horizon-block*22, x+block, horizon, sil)
		}
		for y := horizon - block*22; y < horizon; y += block * 4 {
			c.rectBlocks(0, y, canvasW, y+block, sil)
		}
	case "paladin": // temple — arches/columns
		for x := block * 3; x < canvasW; x += block * 7 {
			c.rectBlocks(x, horizon-block*20, x+block*2, horizon, sil)
		}
		c.rectBlocks(0, horizon-block*22, canvasW, horizon-block*20, sil) // architrave
	case "druid": // grove — tree trunks + canopy
		for i, x := range []int{block * 4, block * 14, block * 24} {
			c.rectBlocks(x, horizon-block*14, x+block*2, horizon, sil)
			r := block * (5 + i%2)
			c.ellipseBlocks(x+block, horizon-block*14, r, r-block, sil)
		}
	case "bard": // stage / plaza — footlights + curtain
		c.rectBlocks(0, horizon-block, canvasW, horizon, ramp[toneHigh]) // stage edge glow
		for x := 0; x < canvasW; x += block * 4 {
			c.rectBlocks(x, 0, x+block, block*10, sil) // hanging curtain folds
		}
	default: // adventurer: road / camp — winding path + tent + campfire
		// path converging to horizon
		for y := horizon; y > horizon-block*20; y -= block {
			w := block*2 + (horizon-y)/2
			c.rectBlocks(canvasW/2-w, y, canvasW/2+w, y+block, lerpColor(low, mid, 0.5))
		}
		// tent: lit left slope, shadowed right slope, dark door opening
		tx := block * 6
		rows := 7
		apexY := horizon - rows*block
		lit := lerpColor(low, ramp[toneHigh], 0.45) // clearly lighter than the backdrop
		for k := 0; k < rows; k++ {
			y := apexY + k*block
			half := 2 + k*2
			c.rectBlocks(tx-half, y, tx, y+block, lit)
			c.rectBlocks(tx, y, tx+half, y+block, sil) // shadow side
		}
		c.rectBlocks(tx-block, horizon-block*3, tx+block, horizon, withAlpha(color.RGBA{R: 10, G: 8, B: 14, A: 255}, 255)) // door
		c.fillBlock(tx-block, apexY-block, ramp[toneHigh])                                                                 // pennant
		// campfire to the right of the tent
		fx := tx + block*9
		c.fillBlock(fx, horizon-block, ramp[toneHigh])
		c.fillBlock(fx-block, horizon-block, ramp[toneLight])
		c.fillBlock(fx, horizon-block*2, ramp[toneLight])
		c.rectBlocks(fx-block*2, horizon, fx+block*2, horizon+block, sil) // logs
	}

	// vignette: darken the top corners a touch so the figure pops (dithered)
	for by := 0; by < canvasH; by += block {
		for bx := 0; bx < canvasW; bx += block {
			edge := false
			if bx < block*3 || bx > canvasW-block*4 || by < block*3 {
				edge = true
			}
			if edge && bayer4(bx, by) < 8 {
				c.blend(bx, by, color.RGBA{R: 0, G: 0, B: 0, A: 60})
			}
		}
	}
}
