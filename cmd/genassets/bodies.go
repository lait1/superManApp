package main

import (
	"fmt"
	"image/color"
)

// ---------------------------------------------------------------------------
// Character base layers (docs/12 §4, §8) — decomposed for onboarding
// customization instead of one monolithic body sprite:
//
//   body   (z "body")   skin_<bodyType>_r<stage>_<tone>.png — head, face,
//                       neck, arms, hands, legs in a skin-tone ramp. The
//                       torso is filled too as a safety underlay; the outfit
//                       covers it.
//   outfit (z "outfit") outfit_<class>_r<stage>_<bodyType>.png — tunic,
//                       sleeves, pants, shoes, belt/pauldrons/cape/emblem/
//                       trim per stage, in the class ramp (palette-swap §9).
//   hair   (z "hair")   hair_<style>_<color>.png — drawn over the head,
//                       under headgear. "bald" emits no file.
//
// The head is the SAME size and position for every body type and stage
// (chibi: the body grows, the head doesn't) so hair and headgear items fit
// every variant. Stage evolution (§8) widens/extends the figure below the
// fixed head and adds costume detail.
// ---------------------------------------------------------------------------

// Fixed head geometry shared by skins, hair and headgear items.
const (
	headRX = 24 // head ellipse x-radius
	headRY = 23 // head ellipse y-radius
)

// stageProfile holds per-stage silhouette tuning (docs/12 §8).
type stageProfile struct {
	torsoHalf int  // torso half-width in px (before body-type adjustment)
	torsoBot  int  // torso bottom y (legs start here)
	pauldron  bool // shoulder pads (leather at 2, metal at 3+)
	cape      bool // back cape behind the torso
	emblem    bool // class motif on the chest
	trim      bool // highlight trim on the costume edges
	halo      bool // legend stage: longer cape here; halo itself is in the front aura
}

func profileForStage(stage int) stageProfile {
	switch stage {
	case 1: // recruit — small, simple tunic, timid
		return stageProfile{torsoHalf: 14, torsoBot: 128}
	case 2: // seeker — a bit broader, belt + leather pauldrons
		return stageProfile{torsoHalf: 16, torsoBot: 130, pauldron: true}
	case 3: // veteran — sturdier, cape, metal pauldrons, chest emblem
		return stageProfile{torsoHalf: 18, torsoBot: 132, pauldron: true, cape: true, emblem: true}
	case 4: // master — notably larger, trimmed costume
		return stageProfile{torsoHalf: 20, torsoBot: 136, pauldron: true, cape: true, emblem: true, trim: true}
	default: // 5 legend — final evolution: halo + full regalia
		return stageProfile{torsoHalf: 22, torsoBot: 138, pauldron: true, cape: true, emblem: true, trim: true, halo: true}
	}
}

// torsoHalfFor applies the body-type adjustment: "a" sturdy, "b" slim.
func torsoHalfFor(p stageProfile, bodyType string) int {
	if bodyType == "b" {
		return p.torsoHalf - 2
	}
	return p.torsoHalf + 2
}

// figure is the shared coordinate skeleton derived from stage + body type.
// Both the skin and the outfit are drawn off the same figure so they always
// align without per-file tweaking.
type figure struct {
	cx       int // horizontal center
	half     int // torso half-width
	torsoTop int
	torsoBot int
	armW     int // arm thickness
	armTop   int
	legBot   int // where legs end and shoes begin
	p        stageProfile
}

func figureFor(stage int, bodyType string) figure {
	p := profileForStage(stage)
	return figure{
		cx:       canvasW / 2,
		half:     torsoHalfFor(p, bodyType),
		torsoTop: 68,
		torsoBot: p.torsoBot,
		armW:     block * 2,
		armTop:   72,
		legBot:   160,
		p:        p,
	}
}

// ---------------------------------------------------------------------------
// Skin layer
// ---------------------------------------------------------------------------

func (g *generator) genSkins() {
	g.skins = map[string]map[string]map[string]string{}
	for _, bt := range bodyTypes {
		g.skins[bt] = map[string]map[string]string{}
		for _, stage := range rankStages {
			g.skins[bt][fmt.Sprintf("%d", stage)] = map[string]string{}
			for _, tone := range skinTones {
				c := newCanvas()
				g.drawSkin(c, figureFor(stage, bt), g.pal.skinTone(tone))
				name := fmt.Sprintf("skin_%s_r%d_%s.png", bt, stage, tone)
				g.writePNG(name, c)
				g.skins[bt][fmt.Sprintf("%d", stage)][tone] = name
			}
		}
	}
}

// drawSkin renders head + face + neck + arms + legs (and a torso underlay)
// in the given skin ramp, then outlines and grounds the figure.
func (g *generator) drawSkin(c *canvas, f figure, skin Ramp) {
	// --- torso underlay (covered by the outfit; prevents see-through gaps) ---
	c.shadeRect(f.cx-f.half, f.torsoTop, f.cx+f.half, f.torsoBot, skin)

	// --- legs (pants cover most; skin shows if an outfit is ever cropped) ---
	c.shadeRect(f.cx-f.half+block, f.torsoBot, f.cx-block, f.legBot+block*2, skin)
	c.shadeRect(f.cx+block, f.torsoBot, f.cx+f.half-block, f.legBot+block*2, skin)

	// --- arms ---
	lx := f.cx - f.half - f.armW
	c.shadeRect(lx, f.armTop, f.cx-f.half, f.armTop+32, skin)
	// left hand
	c.rectBlocks(lx, f.armTop+32, f.cx-f.half, f.armTop+40, skin[toneLight])

	rx := f.cx + f.half
	c.shadeRect(rx, f.armTop, rx+f.armW, f.armTop+24, skin)
	// right forearm steps out to the hand anchor so weapons sit in the hand;
	// it starts inside the upper arm (overlap) so the elbow never shows a gap
	c.thickLineBlocks(rx, f.armTop+16, anchorHandRight.x-block, anchorHandRight.y-block, skin[toneBase])
	// right hand snapped to hand_right
	c.rectBlocks(anchorHandRight.x-block, anchorHandRight.y-block, anchorHandRight.x+block, anchorHandRight.y+block, skin[toneLight])

	// --- neck ---
	c.rectBlocks(f.cx-block*2, 56, f.cx+block*2, f.torsoTop+block, skin[toneBase])
	c.rectBlocks(f.cx-block*2, 60, f.cx+block*2, 64, skin[toneShadow]) // chin shadow

	// --- head with soft top-left light ---
	g.drawHead(c, skin)

	// selective dark outline + dithered ground shadow (docs/12 §4)
	c.selectiveOutline(g.pal.Outline)
	c.groundShadow(g.pal.ShadowGround)
}

// drawHead paints the fixed chibi head + readable face at head_center.
func (g *generator) drawHead(c *canvas, skin Ramp) {
	hc := anchorHeadCenter
	c.ellipseBlocksFunc(hc.x, hc.y, headRX, headRY, func(bx, by int) (color.RGBA, bool) {
		dx := float64(bx+block/2-hc.x) / float64(headRX)
		dy := float64(by+block/2-hc.y) / float64(headRY)
		switch {
		case dy > 0.45 && dx*dx+dy*dy > 0.55: // jaw rim
			return skin[toneShadow], true
		case dy < -0.55 && dx < -0.1: // top-left highlight
			return skin[toneHigh], true
		case dy < -0.3: // top light
			return skin[toneLight], true
		default:
			return skin[toneBase], true
		}
	})

	// ears just outside the ellipse
	c.rectBlocks(hc.x-headRX-block, hc.y-block, hc.x-headRX+block, hc.y+block, skin[toneBase])
	c.rectBlocks(hc.x+headRX-block, hc.y-block, hc.x+headRX+block, hc.y+block, skin[toneBase])

	// eyes: 2×2 white with a pupil at the inner-bottom corner (looking ahead)
	eyeY := hc.y - block
	// left eye
	c.fillBlock(hc.x-block*3, eyeY, g.pal.White)
	c.fillBlock(hc.x-block*2, eyeY, g.pal.White)
	c.fillBlock(hc.x-block*3, eyeY+block, g.pal.White)
	c.fillBlock(hc.x-block*2, eyeY+block, g.pal.Outline) // pupil
	// right eye
	c.fillBlock(hc.x+block, eyeY, g.pal.White)
	c.fillBlock(hc.x+block*2, eyeY, g.pal.White)
	c.fillBlock(hc.x+block*2, eyeY+block, g.pal.White)
	c.fillBlock(hc.x+block, eyeY+block, g.pal.Outline) // pupil

	// friendly smile: short line with raised corners
	mouthY := hc.y + block*3
	c.fillBlock(hc.x-block, mouthY, skin[toneShadow])
	c.fillBlock(hc.x, mouthY, skin[toneShadow])
	c.fillBlock(hc.x-block*2, mouthY-block, skin[toneShadow])
	c.fillBlock(hc.x+block, mouthY-block, skin[toneShadow])
}

// ---------------------------------------------------------------------------
// Blink overlays — one tiny eyelid sprite per skin tone, shown by the
// renderer over the skin layer for ~120ms every few seconds (docs/12 §10
// idle "дыхание + моргание"). Eye geometry is fixed (drawHead), so a single
// overlay fits every body type and stage.
// ---------------------------------------------------------------------------

func (g *generator) genBlinks() {
	g.blinks = map[string]string{}
	for _, tone := range skinTones {
		c := newCanvas()
		g.drawBlink(c, g.pal.skinTone(tone))
		name := fmt.Sprintf("blink_%s.png", tone)
		g.writePNG(name, c)
		g.blinks[tone] = name
	}
}

// drawBlink covers both eyes with skin + a closed-lid lash line.
func (g *generator) drawBlink(c *canvas, skin Ramp) {
	hc := anchorHeadCenter
	eyeY := hc.y - block
	for _, x0 := range []int{hc.x - block*3, hc.x + block} {
		// lids in base skin
		c.rectBlocks(x0, eyeY, x0+block*2, eyeY+block*2, skin[toneBase])
		// closed-lash line on the lower half
		c.rectBlocks(x0, eyeY+block, x0+block*2, eyeY+block*2, skin[toneShadow])
	}
}

// ---------------------------------------------------------------------------
// Outfit layer
// ---------------------------------------------------------------------------

func (g *generator) genOutfits() {
	g.outfits = map[string]map[string]map[string]string{}
	for _, class := range classes {
		g.outfits[class] = map[string]map[string]string{}
		ramp := g.pal.classRamp(class)
		for _, stage := range rankStages {
			g.outfits[class][fmt.Sprintf("%d", stage)] = map[string]string{}
			for _, bt := range bodyTypes {
				c := newCanvas()
				g.drawOutfit(c, figureFor(stage, bt), class, ramp)
				name := fmt.Sprintf("outfit_%s_r%d_%s.png", class, stage, bt)
				g.writePNG(name, c)
				g.outfits[class][fmt.Sprintf("%d", stage)][bt] = name
			}
		}
	}
}

// drawOutfit renders the class costume for a stage on the shared figure.
func (g *generator) drawOutfit(c *canvas, f figure, class string, ramp Ramp) {
	leather := g.pal.neutral("leather")
	metal := g.pal.neutral("metal")

	// --- cape behind the torso (stage 3+) ---
	if f.p.cape {
		top := f.torsoTop + block
		bot := f.torsoBot + block*2
		if f.p.halo {
			bot += block * 2 // legend cape flows longer
		}
		for y := snap(top); y < bot; y += block {
			frac := float64(y-top) / float64(bot-top)
			w := f.half + 2 + int(frac*10)
			col := ramp[toneShadow]
			if f.p.trim && y >= bot-block { // lit hem on master/legend capes
				col = ramp[toneLight]
			}
			c.rectBlocks(f.cx-w, y, f.cx+w, y+block, col)
		}
	}

	// --- tunic ---
	c.shadeRect(f.cx-f.half, f.torsoTop, f.cx+f.half, f.torsoBot, ramp)
	// collar
	c.rectBlocks(f.cx-f.half+block, f.torsoTop, f.cx+f.half-block, f.torsoTop+block, ramp[toneHigh])

	// --- sleeves over the upper arms ---
	lx := f.cx - f.half - f.armW
	rx := f.cx + f.half
	c.shadeRect(lx, f.armTop, f.cx-f.half, f.armTop+20, ramp)
	c.shadeRect(rx, f.armTop, rx+f.armW, f.armTop+20, ramp)

	// --- bracers on the forearms (stage 4+) ---
	if f.p.trim {
		c.rectBlocks(lx, f.armTop+20, f.cx-f.half, f.armTop+28, metal[toneLight])
		c.rectBlocks(rx, f.armTop+20, rx+f.armW, f.armTop+28, metal[toneLight])
	}

	// --- belt with buckle (stage 2+) ---
	if f.p.pauldron {
		beltY := f.torsoBot - 12
		c.rectBlocks(f.cx-f.half, beltY, f.cx+f.half, beltY+block, leather[toneBase])
		c.rectBlocks(f.cx-block, beltY-block, f.cx+block, beltY+block*2, ramp[toneHigh]) // buckle
	}

	// --- pants (darker class hue) ---
	c.rectBlocks(f.cx-f.half+block, f.torsoBot, f.cx-block, f.legBot, ramp[toneShadow])
	c.rectBlocks(f.cx+block, f.torsoBot, f.cx+f.half-block, f.legBot, ramp[toneShadow])

	// --- simple shoes (boot items replace these visually when equipped) ---
	c.shadeRect(f.cx-f.half, f.legBot, f.cx-block, anchorFeetBaseline, leather)
	c.shadeRect(f.cx+block, f.legBot, f.cx+f.half, anchorFeetBaseline, leather)

	// --- pauldrons: leather at stage 2, metal from stage 3; they sit on the
	// shoulder line, capping the sleeve tops ---
	if f.p.pauldron {
		pal := leather
		if f.p.cape {
			pal = metal
		}
		grow := 0
		if f.p.trim {
			grow = block // master/legend pauldrons are larger
		}
		pTop := f.armTop - block
		c.shadeRect(f.cx-f.half-f.armW-grow, pTop, f.cx-f.half+block, pTop+block*3, pal)
		c.shadeRect(f.cx+f.half-block, pTop, f.cx+f.half+f.armW+grow, pTop+block*3, pal)
		if f.p.trim { // studs
			c.fillBlock(f.cx-f.half-block*2, pTop+block, ramp[toneHigh])
			c.fillBlock(f.cx+f.half+block, pTop+block, ramp[toneHigh])
		}
	}

	// --- chest emblem: small class motif (stage 3+) ---
	if f.p.emblem {
		g.drawClassMotif(c, class, ramp)
	}

	// --- trim: lit costume edges (stage 4+) ---
	if f.p.trim {
		c.rectBlocks(f.cx-f.half, f.torsoTop+block, f.cx-f.half+block, f.torsoBot-block, ramp[toneLight])
		c.rectBlocks(f.cx+f.half-block, f.torsoTop+block, f.cx+f.half, f.torsoBot-block, ramp[toneLight])
		c.rectBlocks(f.cx-f.half, f.torsoBot-block, f.cx+f.half, f.torsoBot, ramp[toneHigh]) // hem
	}

	// (the legend halo lives in the front aura — see drawAuraFront — so it
	// renders above hair and headgear)

	c.selectiveOutline(g.pal.Outline)
}

// drawClassMotif paints a tiny per-class glyph centered under the collar.
func (g *generator) drawClassMotif(c *canvas, class string, ramp Ramp) {
	cc := anchorChestCenter
	hi := ramp[toneHigh]
	put := func(dx, dy int) { c.fillBlock(cc.x+dx*block-block, cc.y+dy*block-block, hi) }
	switch class {
	case "warrior": // crossed blades "V"
		put(-2, -1)
		put(2, -1)
		put(-1, 0)
		put(1, 0)
		put(0, 1)
	case "sage": // open book
		c.rectBlocks(cc.x-block*3, cc.y-block, cc.x-block/2, cc.y+block, hi)
		c.rectBlocks(cc.x+block/2, cc.y-block, cc.x+block*3, cc.y+block, hi)
		c.fillBlock(cc.x-block/2-block, cc.y-block, ramp[toneShadow]) // spine
	case "paladin": // plus / holy cross
		put(0, -1)
		put(0, 0)
		put(0, 1)
		put(-1, 0)
		put(1, 0)
	case "druid": // leaf: small triangle with a stem
		put(0, -1)
		put(-1, 0)
		put(0, 0)
		put(1, 0)
		c.fillBlock(cc.x-block, cc.y+block-block, ramp[toneLight]) // stem
	case "bard": // eighth note
		put(1, -1)
		put(1, 0)
		put(0, 1)
		put(-1, 1)
	default: // adventurer — compass points
		put(0, -1)
		put(-2, 0)
		put(2, 0)
		put(0, 1)
	}
}

// ---------------------------------------------------------------------------
// Hair layer
// ---------------------------------------------------------------------------

func (g *generator) genHair() {
	g.hair = map[string]map[string]string{}
	for _, style := range hairstyles {
		if style == "bald" {
			continue // bald = no layer file
		}
		g.hair[style] = map[string]string{}
		for _, hc := range hairColors {
			c := newCanvas()
			g.drawHair(c, style, g.pal.hairColor(hc))
			name := fmt.Sprintf("hair_%s_%s.png", style, hc)
			g.writePNG(name, c)
			g.hair[style][hc] = name
		}
	}
}

// drawHair paints a hairstyle over the fixed head. Styles must keep the face
// (eyes at y≈36..44) clear and stay below y=8 so the legend halo reads.
func (g *generator) drawHair(c *canvas, style string, ramp Ramp) {
	hc := anchorHeadCenter

	// shared cap: upper part of a slightly larger ellipse, with a jagged fringe
	cap := func(fringeY int) {
		c.ellipseBlocksFunc(hc.x, hc.y-block/2, headRX+2, headRY+2, func(bx, by int) (color.RGBA, bool) {
			limit := fringeY
			// fringe notches: every other column dips one block lower
			if checker(bx, 0) {
				limit += block
			}
			// temples: the cap hugs the head sides further down
			if bx < hc.x-headRX+10 || bx >= hc.x+headRX-10 {
				limit = hc.y - block
			}
			if by >= limit {
				return color.RGBA{}, false
			}
			switch {
			case by < 20:
				return ramp[toneLight], true
			case by >= limit-block:
				return ramp[toneShadow], true // under-fringe shadow
			default:
				return ramp[toneBase], true
			}
		})
		// top-left shine
		c.fillBlock(hc.x-block*3, 16, ramp[toneHigh])
		c.fillBlock(hc.x-block*2, 12, ramp[toneHigh])
	}

	switch style {
	case "short":
		cap(hc.y - block*2)
	case "spiky":
		cap(hc.y - block*2)
		// spikes above the hairline (kept below y=8 — the legend halo zone)
		for i, sx := range []int{hc.x - 20, hc.x - 12, hc.x - 4, hc.x + 4, hc.x + 12} {
			h := block * 2
			if i%2 == 1 {
				h = block * 3
			}
			c.rectBlocks(sx, 20-h, sx+block, 24, ramp[toneBase])
			c.fillBlock(sx, 20-h, ramp[toneLight])
		}
	case "long":
		cap(hc.y - block*2)
		// side curtains flowing to the shoulders
		for _, sx := range []int{hc.x - headRX - block, hc.x + headRX - block} {
			c.rectBlocks(sx, hc.y-block*2, sx+block*2, hc.y+44, ramp[toneBase])
			c.rectBlocks(sx, hc.y+36, sx+block*2, hc.y+44, ramp[toneShadow]) // tips
			c.fillBlock(sx, hc.y-block, ramp[toneLight])
		}
	case "ponytail":
		cap(hc.y - block*2)
		// tie + tail swinging right, behind the shoulder
		tie := pt{hc.x + headRX - block, hc.y - 12}
		c.fillBlock(tie.x, tie.y, ramp[toneHigh])
		c.thickLineBlocks(tie.x+block, tie.y, tie.x+block*3, tie.y+28, ramp[toneBase])
		c.rectBlocks(tie.x+block*2, tie.y+28, tie.x+block*4, tie.y+40, ramp[toneShadow]) // tip
	}

	c.selectiveOutline(g.pal.Outline)
}
