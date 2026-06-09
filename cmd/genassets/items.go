package main

import (
	"image/color"
)

// ---------------------------------------------------------------------------
// Items — equippable paper-doll layers (docs/04 §3 slots, §4 rarity; docs/12
// §5 z-order + anchors). Every catalog item now has a DISTINCT sprite (not a
// per-slot template): materials use the neutral ramps (metal/leather/wood),
// while the rarity ramp colors accents — trim, gems, glow. Each sprite is
// drawn on the full 128x192 canvas aligned to its slot anchor.
//
// The id/slot/rarity rows mirror shop_items (memory/seed.go is the canon;
// migrations/0004 adds the purchasable counterparts). Slots:
//   head, armor, weapon, amulet, back, boots — regular paper-doll layers;
//   aura       — cosmetic cape rendered behind the figure (auraBack position);
//   background — a full scene that replaces the class backdrop.
// ---------------------------------------------------------------------------

// itemSpec describes one equipment piece and how to draw it.
type itemSpec struct {
	ID     string
	Slot   string
	Rarity string
	draw   func(g *generator, c *canvas, rarity Ramp)
}

// itemCatalog covers every visual item. Order here is the manifest order.
var itemCatalog = []itemSpec{
	// head
	{"helm_iron", "head", "common", (*generator).drawHelmIron},
	{"hood_seeker", "head", "uncommon", (*generator).drawHoodSeeker},
	{"crown_sage", "head", "epic", (*generator).drawCrownSage},
	// armor
	{"vest_padded", "armor", "common", (*generator).drawVestPadded},
	{"armor_vit", "armor", "rare", (*generator).drawArmorVit},
	{"armor_titan", "armor", "legendary", (*generator).drawArmorTitan},
	// weapon
	{"sword_short", "weapon", "common", (*generator).drawSwordShort},
	{"staff_arcane", "weapon", "rare", (*generator).drawStaffArcane},
	{"blade_focus", "weapon", "epic", (*generator).drawBladeFocus},
	// amulet
	{"amulet_owl", "amulet", "rare", (*generator).drawAmuletOwl},
	{"amulet_sun", "amulet", "legendary", (*generator).drawAmuletSun},
	// back
	{"cloak_traveler", "back", "uncommon", (*generator).drawCloakTraveler},
	{"wings_phoenix", "back", "legendary", (*generator).drawWingsPhoenix},
	// aura-slot cosmetic (renders behind the whole figure)
	{"legendary_cape_conquistador", "aura", "legendary", (*generator).drawCapeConquistador},
	// boots
	{"boots_leather", "boots", "common", (*generator).drawBootsLeather},
	{"boots_swift", "boots", "rare", (*generator).drawBootsSwift},
	// background (opaque scene replacement)
	{"bg_neon_city", "background", "uncommon", (*generator).drawNeonCity},
}

func (g *generator) genItems() {
	for _, it := range itemCatalog {
		c := newCanvas()
		it.draw(g, c, g.pal.rarity(it.Rarity))
		if it.Slot != "background" { // scenes are opaque, no outline pass
			c.selectiveOutline(g.pal.Outline)
		}

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

// ---------------------------------------------------------------------------
// head
// ---------------------------------------------------------------------------

// helmDome paints the upper-half-of-head dome in the given ramp.
func (g *generator) helmDome(c *canvas, ramp Ramp, r int) {
	hc := anchorHeadCenter
	c.ellipseBlocksFunc(hc.x, hc.y-block, r, r, func(bx, by int) (color.RGBA, bool) {
		if by > hc.y-block {
			return color.RGBA{}, false // top half only — the face stays open
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
}

// drawHelmIron: plain metal dome + brow band + nose guard.
func (g *generator) drawHelmIron(c *canvas, rarity Ramp) {
	hc := anchorHeadCenter
	metal := g.pal.neutral("metal")
	g.helmDome(c, metal, headRX+2)
	// brow band
	c.rectBlocks(hc.x-headRX, hc.y-block, hc.x+headRX, hc.y, metal[toneShadow])
	// nose guard between the eyes
	c.rectBlocks(hc.x-block, hc.y-block, hc.x+block, hc.y+block*2, metal[toneBase])
	// crest stud carries the rarity accent
	c.fillBlock(hc.x-block, hc.y-block-headRX, rarity[toneHigh])
}

// drawHoodSeeker: cloth hood with a deep face opening, draped to the shoulders.
func (g *generator) drawHoodSeeker(c *canvas, rarity Ramp) {
	hc := anchorHeadCenter
	g.helmDome(c, rarity, headRX+4)
	// sides draped down toward the shoulders
	for _, sx := range []int{hc.x - headRX - block*2, hc.x + headRX} {
		c.rectBlocks(sx, hc.y-block, sx+block*2, hc.y+28, rarity[toneBase])
		c.rectBlocks(sx, hc.y+20, sx+block*2, hc.y+28, rarity[toneShadow])
	}
	// pointy tip
	c.fillBlock(hc.x-block, hc.y-block-headRX-block*2, rarity[toneBase])
	c.fillBlock(hc.x-block, hc.y-block-headRX-block, rarity[toneLight])
	// shadowed brow line — the "mysterious seeker" look
	c.rectBlocks(hc.x-headRX+block*2, hc.y-block*2, hc.x+headRX-block*2, hc.y-block, rarity[toneShadow])
}

// drawCrownSage: golden band with three points and rarity gems.
func (g *generator) drawCrownSage(c *canvas, rarity Ramp) {
	hc := anchorHeadCenter
	gold := g.pal.classRamp("paladin")
	bandY := hc.y - headRY - block // sits on top of the head/hair
	// band
	c.rectBlocks(hc.x-20, bandY, hc.x+20, bandY+block*2, gold[toneBase])
	c.rectBlocks(hc.x-20, bandY, hc.x+20, bandY+block, gold[toneLight])
	// three points
	for _, dx := range []int{-16, -4, 8} {
		c.rectBlocks(hc.x+dx, bandY-block*2, hc.x+dx+block*2, bandY, gold[toneBase])
		c.fillBlock(hc.x+dx, bandY-block*2, gold[toneHigh])
	}
	// gems on the band (epic accent)
	for _, dx := range []int{-12, 0, 12} {
		c.fillBlock(hc.x+dx-block, bandY+block, rarity[toneLight])
	}
}

// ---------------------------------------------------------------------------
// armor — drawn at a middle-ground torso width (item sprites are stage-agnostic)
// ---------------------------------------------------------------------------

// drawVestPadded: stitched padded leather vest.
func (g *generator) drawVestPadded(c *canvas, rarity Ramp) {
	cc := anchorChestCenter
	leather := g.pal.neutral("leather")
	w, top, bot := 18, cc.y-26, cc.y+28
	c.shadeRect(cc.x-w, top, cc.x+w, bot, leather)
	// horizontal stitch lines — the "padded" read
	for y := top + block*2; y < bot-block; y += block * 3 {
		c.rectBlocks(cc.x-w+block, y, cc.x+w-block, y+block, leather[toneShadow])
	}
	// collar + rarity stud
	c.rectBlocks(cc.x-w, top, cc.x+w, top+block, leather[toneLight])
	c.fillBlock(cc.x-block, top+block, rarity[toneLight])
}

// drawArmorVit: guard plate with a shield emblem (rare blue accents).
func (g *generator) drawArmorVit(c *canvas, rarity Ramp) {
	cc := anchorChestCenter
	metal := g.pal.neutral("metal")
	w, top, bot := 20, cc.y-28, cc.y+26
	c.shadeRect(cc.x-w, top, cc.x+w, bot, metal)
	// small pauldron caps
	c.rectBlocks(cc.x-w-block, top, cc.x-w+block, top+block*3, metal[toneLight])
	c.rectBlocks(cc.x+w-block, top, cc.x+w+block, top+block*3, metal[toneLight])
	// shield emblem at the chest in rarity color
	c.rectBlocks(cc.x-block*2, cc.y-block*2, cc.x+block*2, cc.y+block, rarity[toneBase])
	c.rectBlocks(cc.x-block, cc.y+block, cc.x+block, cc.y+block*2, rarity[toneBase])
	c.fillBlock(cc.x-block, cc.y-block, rarity[toneHigh])
	// belt line
	c.rectBlocks(cc.x-w, bot-block, cc.x+w, bot, metal[toneShadow])
}

// drawArmorTitan: heavy plate with oversized pauldrons and legendary glow.
func (g *generator) drawArmorTitan(c *canvas, rarity Ramp) {
	cc := anchorChestCenter
	metal := g.pal.neutral("metal")
	w, top, bot := 24, cc.y-30, cc.y+30
	c.shadeRect(cc.x-w, top, cc.x+w, bot, metal)
	// massive pauldrons
	c.shadeRect(cc.x-w-block*3, top-block, cc.x-w+block*2, top+block*4, metal)
	c.shadeRect(cc.x+w-block*2, top-block, cc.x+w+block*3, top+block*4, metal)
	c.fillBlock(cc.x-w-block*2, top, rarity[toneHigh])
	c.fillBlock(cc.x+w+block, top, rarity[toneHigh])
	// glowing core + rivet studs in legendary orange
	c.rectBlocks(cc.x-block*2, cc.y-block*2, cc.x+block*2, cc.y+block*2, rarity[toneBase])
	c.fillBlock(cc.x-block, cc.y-block, rarity[toneHigh])
	c.fillBlock(cc.x, cc.y, rarity[toneLight])
	for _, dy := range []int{-24, 24} {
		c.fillBlock(cc.x-w+block, cc.y+dy, rarity[toneLight])
		c.fillBlock(cc.x+w-block*2, cc.y+dy, rarity[toneLight])
	}
}

// ---------------------------------------------------------------------------
// weapons — held in the right hand at hand_right
// ---------------------------------------------------------------------------

// drawSwordShort: modest one-hand sword.
func (g *generator) drawSwordShort(c *canvas, rarity Ramp) {
	h := anchorHandRight
	metal := g.pal.neutral("metal")
	wood := g.pal.neutral("wood")
	// grip through the fist
	c.rectBlocks(h.x-block, h.y-block, h.x+block, h.y+block*3, wood[toneBase])
	c.fillBlock(h.x-block, h.y+block*3, wood[toneShadow]) // pommel
	// crossguard
	c.rectBlocks(h.x-block*2, h.y-block*2, h.x+block*2, h.y-block, metal[toneShadow])
	// blade with a lit edge
	top := h.y - 48
	c.rectBlocks(h.x-block, top, h.x, h.y-block*2, metal[toneBase])
	c.rectBlocks(h.x, top, h.x+block, h.y-block*2, metal[toneLight])
	c.fillBlock(h.x-block, top-block, metal[toneHigh]) // tip
	_ = rarity
}

// drawStaffArcane: wooden staff with a glowing orb.
func (g *generator) drawStaffArcane(c *canvas, rarity Ramp) {
	h := anchorHandRight
	wood := g.pal.neutral("wood")
	// shaft runs past the hand down to mid-leg
	c.rectBlocks(h.x-block, h.y-56, h.x+block, h.y+28, wood[toneBase])
	c.rectBlocks(h.x-block, h.y-56, h.x, h.y+28, wood[toneLight]) // lit side
	// orb at the top with a ring and sparkle
	oy := h.y - 64
	c.ellipseBlocks(h.x, oy, block*3, block*3, rarity[toneBase])
	c.fillBlock(h.x-block, oy-block, rarity[toneHigh]) // inner glow
	c.circleRingBlocks(h.x, oy, block*4, block*3, rarity[toneShadow])
	c.fillBlock(h.x+block*3, oy-block*3, rarity[toneLight]) // sparkle
}

// drawBladeFocus: long epic blade with a violet glow edge.
func (g *generator) drawBladeFocus(c *canvas, rarity Ramp) {
	h := anchorHandRight
	metal := g.pal.neutral("metal")
	wood := g.pal.neutral("wood")
	c.rectBlocks(h.x-block, h.y-block, h.x+block, h.y+block*3, wood[toneBase])
	// winged crossguard with a gem
	c.rectBlocks(h.x-block*3, h.y-block*2, h.x+block*3, h.y-block, metal[toneShadow])
	c.fillBlock(h.x-block, h.y-block*2, rarity[toneHigh]) // gem
	// long blade
	top := h.y - 68
	c.rectBlocks(h.x-block, top, h.x, h.y-block*2, metal[toneLight])
	c.rectBlocks(h.x, top, h.x+block, h.y-block*2, metal[toneHigh])
	// sparse violet glow motes along the edge
	for i, y := 0, top+block; y < h.y-block*3; y, i = y+block*3, i+1 {
		side := block
		if i%2 == 1 {
			side = -block * 2
		}
		c.fillBlock(h.x+side, y, withAlpha(rarity[toneLight], 170))
	}
	c.fillBlock(h.x-block, top-block, rarity[toneHigh]) // charged tip
}

// ---------------------------------------------------------------------------
// amulets — pendant at chest_center
// ---------------------------------------------------------------------------

// amuletChain paints a short cord "V" from under the collar to the pendant —
// kept tight so the outline pass doesn't turn it into stray marks on the chest.
func (g *generator) amuletChain(c *canvas) {
	cc := anchorChestCenter
	chain := g.pal.neutral("metal")[toneLight]
	c.fillBlock(cc.x-block*3, cc.y-20, chain)
	c.fillBlock(cc.x-block*2, cc.y-16, chain)
	c.fillBlock(cc.x+block*2, cc.y-20, chain)
	c.fillBlock(cc.x+block, cc.y-16, chain)
}

// drawAmuletOwl: round pendant with two owl eyes.
func (g *generator) drawAmuletOwl(c *canvas, rarity Ramp) {
	cc := anchorChestCenter
	g.amuletChain(c)
	metal := g.pal.neutral("metal")
	c.ellipseBlocks(cc.x, cc.y, block*3, block*3, metal[toneBase])
	c.circleRingBlocks(cc.x, cc.y, block*3+2, block*3-1, metal[toneShadow])
	// owl eyes in rare blue + tiny beak
	c.fillBlock(cc.x-block*2, cc.y-block, rarity[toneLight])
	c.fillBlock(cc.x+block, cc.y-block, rarity[toneLight])
	c.fillBlock(cc.x-block, cc.y+block, metal[toneHigh])
}

// drawAmuletSun: radiant sun disc.
func (g *generator) drawAmuletSun(c *canvas, rarity Ramp) {
	cc := anchorChestCenter
	g.amuletChain(c)
	// rays
	for _, d := range []pt{{0, -16}, {0, 16}, {-16, 0}, {16, 0}, {-12, -12}, {12, -12}, {-12, 12}, {12, 12}} {
		c.fillBlock(cc.x+d.x-block/2, cc.y+d.y-block/2, rarity[toneLight])
	}
	// disc
	c.ellipseBlocks(cc.x, cc.y, block*3, block*3, rarity[toneBase])
	c.fillBlock(cc.x-block, cc.y-block, rarity[toneHigh])
	c.fillBlock(cc.x, cc.y, rarity[toneHigh])
}

// ---------------------------------------------------------------------------
// back & aura capes — hang from back_mount, behind the body
// ---------------------------------------------------------------------------

// capeShape fills a trapezoid cape from the shoulders down.
func (g *generator) capeShape(c *canvas, bot, wTop, wBot int, body, hem color.RGBA) {
	bm := anchorBackMount
	top := bm.y - block*4
	for y := snap(top); y < bot; y += block {
		frac := float64(y-top) / float64(bot-top)
		w := wTop + int(frac*float64(wBot-wTop))
		col := body
		if y >= bot-block*2 {
			col = hem
		}
		c.rectBlocks(bm.x-w, y, bm.x+w, y+block, col)
	}
}

// drawCloakTraveler: plain road cloak with a clasp.
func (g *generator) drawCloakTraveler(c *canvas, rarity Ramp) {
	leather := g.pal.neutral("leather")
	g.capeShape(c, 140, 16, 30, rarity[toneShadow], rarity[toneBase])
	// collar + clasp
	bm := anchorBackMount
	c.rectBlocks(bm.x-block*3, bm.y-block*5, bm.x+block*3, bm.y-block*4, leather[toneLight])
	c.fillBlock(bm.x-block, bm.y-block*5, leather[toneHigh])
}

// drawWingsPhoenix: flame wings spreading up and out.
func (g *generator) drawWingsPhoenix(c *canvas, rarity Ramp) {
	bm := anchorBackMount
	// each wing: stepped diagonal rows getting longer toward the top
	for i := 0; i < 9; i++ {
		y := bm.y - 8 - i*block
		span := 10 + i*5
		inner := 12 + i*2
		// tone gets hotter toward the tips
		tone := toneShadow
		if i > 2 {
			tone = toneBase
		}
		if i > 5 {
			tone = toneLight
		}
		c.rectBlocks(bm.x-inner-span, y, bm.x-inner, y+block, rarity[tone])
		c.rectBlocks(bm.x+inner, y, bm.x+inner+span, y+block, rarity[tone])
	}
	// tip flares
	c.fillBlock(bm.x-58, bm.y-44, rarity[toneHigh])
	c.fillBlock(bm.x+54, bm.y-44, rarity[toneHigh])
	// feather notches: knock out alternating bottom blocks for a wing read
	c.rectBlocks(bm.x-34, bm.y-8+block, bm.x-30, bm.y-8+block*2, color.RGBA{})
	c.rectBlocks(bm.x+30, bm.y-8+block, bm.x+34, bm.y-8+block*2, color.RGBA{})
}

// drawCapeConquistador: grand gold-trimmed cape (aura slot — behind everything).
func (g *generator) drawCapeConquistador(c *canvas, rarity Ramp) {
	gold := g.pal.classRamp("paladin")
	g.capeShape(c, 152, 22, 40, rarity[toneShadow], gold[toneBase])
	bm := anchorBackMount
	// gold trim columns along the outer edge
	top := bm.y - block*4
	for y := snap(top); y < 152; y += block {
		frac := float64(y-top) / float64(152-top)
		w := 22 + int(frac*float64(40-22))
		c.fillBlock(bm.x-w, y, gold[toneLight])
		c.fillBlock(bm.x+w-block, y, gold[toneLight])
	}
	// shoulder mantle
	c.rectBlocks(bm.x-26, top, bm.x+26, top+block*2, rarity[toneBase])
	c.fillBlock(bm.x-block, top, gold[toneHigh]) // clasp
}

// ---------------------------------------------------------------------------
// boots — replace the outfit's simple shoes visually
// ---------------------------------------------------------------------------

// bootPair paints two boots over the shoe zone.
func (g *generator) bootPair(c *canvas, ramp Ramp) {
	base := anchorFeetBaseline
	cx := canvasW / 2
	for _, side := range []int{-1, 1} {
		x0 := cx + side*block
		x1 := cx + side*20
		if side < 0 {
			x0, x1 = x1, x0
		}
		c.shadeRect(x0, base-24, x1, base, ramp)
		c.rectBlocks(x0, base-24, x1, base-20, ramp[toneLight]) // cuff
	}
}

// drawBootsLeather: plain leather boots.
func (g *generator) drawBootsLeather(c *canvas, rarity Ramp) {
	g.bootPair(c, g.pal.neutral("leather"))
	_ = rarity
}

// drawBootsSwift: boots with little ankle wings (rare accents).
func (g *generator) drawBootsSwift(c *canvas, rarity Ramp) {
	leather := g.pal.neutral("leather")
	g.bootPair(c, leather)
	base := anchorFeetBaseline
	cx := canvasW / 2
	// trim + ankle wings
	for _, side := range []int{-1, 1} {
		edge := cx + side*20
		c.fillBlock(edge-block+side*block, base-16, rarity[toneLight])
		c.fillBlock(edge-block+side*block*2, base-12, g.pal.White)
		c.fillBlock(edge-block+side*block*3, base-16, g.pal.White)
	}
}

// ---------------------------------------------------------------------------
// background — full opaque scene that replaces the class backdrop
// ---------------------------------------------------------------------------

// drawNeonCity: night city with lit windows and neon strips.
func (g *generator) drawNeonCity(c *canvas, rarity Ramp) {
	_ = rarity
	// night sky gradient (deep indigo -> purple horizon)
	top := color.RGBA{0x12, 0x10, 0x2A, 255}
	mid := color.RGBA{0x2A, 0x18, 0x4A, 255}
	low := color.RGBA{0x4A, 0x20, 0x5A, 255}
	stops := []color.RGBA{top, lerpColor(top, mid, 0.5), mid, lerpColor(mid, low, 0.5), low}
	for by := 0; by < canvasH; by += block {
		for bx := 0; bx < canvasW; bx += block {
			frac := float64(by) / float64(canvasH)
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
			c.fillBlock(bx, by, col)
		}
	}

	// stars (deterministic sparse pattern in the upper sky)
	starCol := color.RGBA{0xC8, 0xC8, 0xE8, 255}
	for by := 0; by < 56; by += block {
		for bx := 0; bx < canvasW; bx += block {
			if (bx*7+by*13)%97 == 0 {
				c.fillBlock(bx, by, starCol)
			}
		}
	}

	// building silhouettes with lit windows
	bld := color.RGBA{0x0A, 0x08, 0x18, 255}
	winA := color.RGBA{0xF0, 0xD2, 0x6A, 255} // warm window
	winB := color.RGBA{0x4A, 0xC8, 0xE8, 255} // cool window
	heights := []int{52, 84, 64, 96, 56, 76}
	bw := canvasW / len(heights)
	for i, h := range heights {
		x0 := i * bw
		yTop := anchorFeetBaseline - h
		c.rectBlocks(x0, yTop, x0+bw, canvasH, bld)
		// neon strip on the roofline, alternating colors
		neon := color.RGBA{0xE8, 0x3C, 0xC8, 255}
		if i%2 == 1 {
			neon = color.RGBA{0x3C, 0xD8, 0xE8, 255}
		}
		c.rectBlocks(x0, yTop, x0+bw, yTop+block, neon)
		// windows: sparse deterministic grid
		for wy := yTop + block*2; wy < canvasH-block*2; wy += block * 3 {
			for wx := x0 + block; wx < x0+bw-block; wx += block * 2 {
				switch (wx*5 + wy*11) % 7 {
				case 0:
					c.fillBlock(wx, wy, winA)
				case 3:
					c.fillBlock(wx, wy, winB)
				}
			}
		}
	}

	// ground line
	c.rectBlocks(0, anchorFeetBaseline, canvasW, canvasH, color.RGBA{0x10, 0x0C, 0x20, 255})
}
