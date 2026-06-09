package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

// ---------------------------------------------------------------------------
// canvas — a 128x192 RGBA target with helpers that paint on a coarse "big
// pixel" grid (block px squares) so output reads as HD pixel-art (docs/12 §2).
// All coordinates are canvas pixels unless noted; block helpers snap to grid.
// ---------------------------------------------------------------------------

type canvas struct {
	img *image.RGBA
}

func newCanvas() *canvas {
	return &canvas{img: image.NewRGBA(image.Rect(0, 0, canvasW, canvasH))}
}

// set paints a single canvas pixel (clipped, opaque-or-alpha as given).
func (c *canvas) set(x, y int, col color.RGBA) {
	if x < 0 || y < 0 || x >= canvasW || y >= canvasH {
		return
	}
	c.img.SetRGBA(x, y, col)
}

// blend draws col over the existing pixel using col.A as the source alpha
// (straight-alpha "over"). Used for translucent auras / scene dithering.
func (c *canvas) blend(x, y int, col color.RGBA) {
	if x < 0 || y < 0 || x >= canvasW || y >= canvasH {
		return
	}
	dst := c.img.RGBAAt(x, y)
	sa := float64(col.A) / 255
	da := float64(dst.A) / 255
	outA := sa + da*(1-sa)
	if outA <= 0 {
		c.img.SetRGBA(x, y, color.RGBA{})
		return
	}
	mix := func(s, d uint8) uint8 {
		v := (float64(s)*sa + float64(d)*da*(1-sa)) / outA
		if v > 255 {
			v = 255
		}
		return uint8(v + 0.5)
	}
	c.img.SetRGBA(x, y, color.RGBA{
		R: mix(col.R, dst.R),
		G: mix(col.G, dst.G),
		B: mix(col.B, dst.B),
		A: uint8(outA*255 + 0.5),
	})
}

// fillBlock paints a single big-pixel cell whose top-left is (bx,by) in canvas
// coords. Block size is `block`. Opaque set.
func (c *canvas) fillBlock(bx, by int, col color.RGBA) {
	for dy := 0; dy < block; dy++ {
		for dx := 0; dx < block; dx++ {
			c.set(bx+dx, by+dy, col)
		}
	}
}

// rectBlocks fills an axis-aligned rectangle [x0,x1) x [y0,y1) (canvas coords),
// snapping the span to the big-pixel grid. Opaque.
func (c *canvas) rectBlocks(x0, y0, x1, y1 int, col color.RGBA) {
	x0 = snap(x0)
	y0 = snap(y0)
	for y := y0; y < y1; y += block {
		for x := x0; x < x1; x += block {
			c.fillBlock(x, y, col)
		}
	}
}

// rectBlocksRamp fills a rect with a vertical 4-tone shade: highlight at top,
// fading to shadow at the bottom — cheap "form" on a flat block silhouette.
func (c *canvas) rectBlocksRamp(x0, y0, x1, y1 int, ramp Ramp) {
	x0, y0 = snap(x0), snap(y0)
	h := y1 - y0
	if h <= 0 {
		return
	}
	for y := y0; y < y1; y += block {
		frac := float64(y-y0) / float64(h) // 0 top .. 1 bottom
		// top -> highlight, bottom -> shadow
		idx := int((1 - frac) * 3.999)
		if idx < 0 {
			idx = 0
		}
		if idx > 3 {
			idx = 3
		}
		for x := x0; x < x1; x += block {
			c.fillBlock(x, y, ramp[idx])
		}
	}
}

// shadeRect fills a rect with simple form shading: light top row and left
// column, shadow right column and bottom row, base elsewhere. Reads as a lit
// volume without the banded look of a full vertical gradient.
func (c *canvas) shadeRect(x0, y0, x1, y1 int, ramp Ramp) {
	x0, y0 = snap(x0), snap(y0)
	for y := y0; y < y1; y += block {
		for x := x0; x < x1; x += block {
			col := ramp[toneBase]
			switch {
			case y == y0 || x == x0:
				col = ramp[toneLight]
			case x >= x1-block || y >= y1-block:
				col = ramp[toneShadow]
			}
			c.fillBlock(x, y, col)
		}
	}
}

// thickLineBlocks draws a 2-block-wide stepped line between two points on the
// block grid (used for bent limbs / hair tails). Deterministic, no AA.
func (c *canvas) thickLineBlocks(x0, y0, x1, y1 int, col color.RGBA) {
	steps := abs(y1-y0) / block
	if s := abs(x1-x0) / block; s > steps {
		steps = s
	}
	if steps == 0 {
		steps = 1
	}
	for i := 0; i <= steps; i++ {
		x := x0 + (x1-x0)*i/steps
		y := y0 + (y1-y0)*i/steps
		c.fillBlock(snap(x), snap(y), col)
		c.fillBlock(snap(x)+block, snap(y), col)
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// ellipseBlocks fills a block-grid ellipse centered at (cx,cy) with radii rx,ry.
func (c *canvas) ellipseBlocks(cx, cy, rx, ry int, col color.RGBA) {
	c.ellipseBlocksFunc(cx, cy, rx, ry, func(bx, by int) (color.RGBA, bool) {
		return col, true
	})
}

// ellipseBlocksFunc fills a block-grid ellipse, calling paint(bx,by) per block;
// paint returns the color and whether to draw the block (lets callers dither).
func (c *canvas) ellipseBlocksFunc(cx, cy, rx, ry int, paint func(bx, by int) (color.RGBA, bool)) {
	if rx <= 0 || ry <= 0 {
		return
	}
	for by := snap(cy - ry); by <= cy+ry; by += block {
		for bx := snap(cx - rx); bx <= cx+rx; bx += block {
			// test the block center against the ellipse equation
			px := bx + block/2 - cx
			py := by + block/2 - cy
			fx := float64(px) / float64(rx)
			fy := float64(py) / float64(ry)
			if fx*fx+fy*fy <= 1.0 {
				if col, ok := paint(bx, by); ok {
					c.fillBlock(bx, by, col)
				}
			}
		}
	}
}

// circleRingBlocks paints the block ring (annulus) between inner and outer radii.
func (c *canvas) circleRingBlocks(cx, cy, rOuter, rInner int, col color.RGBA) {
	for by := snap(cy - rOuter); by <= cy+rOuter; by += block {
		for bx := snap(cx - rOuter); bx <= cx+rOuter; bx += block {
			px := bx + block/2 - cx
			py := by + block/2 - cy
			d2 := px*px + py*py
			if d2 <= rOuter*rOuter && d2 >= rInner*rInner {
				c.fillBlock(bx, by, col)
			}
		}
	}
}

// selectiveOutline adds a dark contour around every non-transparent block edge
// that borders transparency (docs/12 §4: selective outlining, darker outside).
// Operates on the block grid: a block is "filled" if its center pixel has alpha.
func (c *canvas) selectiveOutline(outline color.RGBA) {
	type cell struct{ bx, by int }
	var toDraw []cell
	for by := 0; by < canvasH; by += block {
		for bx := 0; bx < canvasW; bx += block {
			if c.blockAlpha(bx, by) > 0 {
				continue // only outline empty cells adjacent to filled ones
			}
			if c.blockAlpha(bx-block, by) > 0 || c.blockAlpha(bx+block, by) > 0 ||
				c.blockAlpha(bx, by-block) > 0 || c.blockAlpha(bx, by+block) > 0 {
				toDraw = append(toDraw, cell{bx, by})
			}
		}
	}
	for _, cc := range toDraw {
		c.fillBlock(cc.bx, cc.by, outline)
	}
}

// blockAlpha reports the alpha of a block's top-left pixel (grid sampling).
func (c *canvas) blockAlpha(bx, by int) uint8 {
	if bx < 0 || by < 0 || bx >= canvasW || by >= canvasH {
		return 0
	}
	return c.img.RGBAAt(bx, by).A
}

// groundShadow paints the dithered elliptical shadow-pad under the figure
// (docs/12 §4) sitting on the feet baseline.
func (c *canvas) groundShadow(ground color.RGBA) {
	cy := anchorFeetBaseline
	c.ellipseBlocksFunc(canvasW/2, cy, 28, 8, func(bx, by int) (color.RGBA, bool) {
		col := ground
		// dither: skip every other block on the outer edge for a soft pad
		if checker(bx, by) {
			col.A = 110
		} else {
			col.A = 170
		}
		// blend rather than overwrite so it reads as a soft pad
		c.blend(bx, by, withDither(col, bx, by))
		return col, false // we already painted via blend
	})
}

// writePNG encodes the canvas to <dir>/<name> and bumps the file counter.
func (g *generator) writePNG(name string, c *canvas) {
	path := filepath.Join(g.dir, name)
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create:", err)
		os.Exit(1)
	}
	defer f.Close()
	if err := png.Encode(f, c.img); err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		os.Exit(1)
	}
	g.fileCount++
}

// ---------------------------------------------------------------------------
// small deterministic helpers (NO random — everything is a function of coords)
// ---------------------------------------------------------------------------

// snap rounds v down to the big-pixel grid.
func snap(v int) int {
	if v < 0 {
		// floor division for negatives
		return -((-v + block - 1) / block) * block
	}
	return (v / block) * block
}

// checker is a deterministic 1-block checkerboard (Bayer-ish) used for dithering.
func checker(x, y int) bool {
	return ((x/block)+(y/block))%2 == 0
}

// bayer4 returns a value 0..15 from a 4x4 ordered-dither matrix at block coords —
// deterministic gradient dithering without any RNG.
func bayer4(x, y int) int {
	m := [4][4]int{
		{0, 8, 2, 10},
		{12, 4, 14, 6},
		{3, 11, 1, 9},
		{15, 7, 13, 5},
	}
	bx := (x / block) % 4
	by := (y / block) % 4
	if bx < 0 {
		bx += 4
	}
	if by < 0 {
		by += 4
	}
	return m[by][bx]
}

// withDither nudges alpha slightly by Bayer position so soft pads look grainy
// rather than banded. Pure function of position.
func withDither(col color.RGBA, x, y int) color.RGBA {
	adj := bayer4(x, y) - 8 // -8..+7
	a := int(col.A) + adj
	if a < 0 {
		a = 0
	}
	if a > 255 {
		a = 255
	}
	col.A = uint8(a)
	return col
}

// scale8 multiplies an 8-bit channel by f (0..1), clamped.
func scale8(v uint8, f float64) uint8 {
	r := float64(v) * f
	if r < 0 {
		r = 0
	}
	if r > 255 {
		r = 255
	}
	return uint8(r + 0.5)
}

// withAlpha returns col with a given alpha.
func withAlpha(col color.RGBA, a uint8) color.RGBA {
	col.A = a
	return col
}

// lerpColor linearly interpolates two opaque colors by t in [0,1].
func lerpColor(a, b color.RGBA, t float64) color.RGBA {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	mix := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t + 0.5) }
	return color.RGBA{R: mix(a.R, b.R), G: mix(a.G, b.G), B: mix(a.B, b.B), A: 255}
}
