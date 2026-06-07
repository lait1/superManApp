package main

import (
	"fmt"
	"image/color"
)

// ---------------------------------------------------------------------------
// Body silhouettes — z3 "body" layer (docs/12 §8). One chibi silhouette per
// class × rank stage. Color comes from the class ramp (palette-swap, §9); the
// silhouette grows / gains a cape as the stage rises (§8 evolution table).
// Drawn on the shared anchor skeleton so any headgear/weapon snaps on later.
//
// We also emit the matching "arms" layer (z7) and "head" layer (z7 face) so the
// renderer can stack them; the body file itself contains torso+legs+head as a
// complete fallback silhouette, while arms/head exist as discrete optional
// overlays. To keep the contract simple, the manifest only lists `body_*`
// files (the head & arms are baked into the body for the placeholder); the
// separate layers are still produced for renderer experimentation.
// ---------------------------------------------------------------------------

// stageProfile holds per-stage silhouette tuning (docs/12 §8).
type stageProfile struct {
	bodyW    int  // torso half-width in canvas px
	bodyTop  int  // torso top y
	bodyBot  int  // torso bottom y (legs start here)
	headR    int  // head radius
	cape     bool // back cape present
	pauldron bool // shoulder pads
	crown    bool // legend "evolution" flourish
}

func profileForStage(stage int) stageProfile {
	switch stage {
	case 1: // recruit — small, simple tunic, timid
		return stageProfile{bodyW: 18, bodyTop: 64, bodyBot: 120, headR: 22}
	case 2: // seeker — taller, belt/pauldron, more confident
		return stageProfile{bodyW: 20, bodyTop: 62, bodyBot: 124, headR: 23, pauldron: true}
	case 3: // veteran — sturdier, cape, open posture
		return stageProfile{bodyW: 22, bodyTop: 60, bodyBot: 128, headR: 24, pauldron: true, cape: true}
	case 4: // master — notably larger, detailed costume, dynamic
		return stageProfile{bodyW: 25, bodyTop: 58, bodyBot: 132, headR: 25, pauldron: true, cape: true}
	default: // 5 legend — final evolution: unique silhouette
		return stageProfile{bodyW: 27, bodyTop: 56, bodyBot: 134, headR: 26, pauldron: true, cape: true, crown: true}
	}
}

func (g *generator) genBodies() {
	g.bodies = map[string]map[string]string{}
	for _, class := range classes {
		g.bodies[class] = map[string]string{}
		ramp := g.pal.classRamp(class)
		for _, stage := range rankStages {
			c := newCanvas()
			g.drawBody(c, ramp, stage)
			name := fmt.Sprintf("body_%s_r%d.png", class, stage)
			g.writePNG(name, c)
			g.bodies[class][fmt.Sprintf("%d", stage)] = name
		}
	}
}

// drawBody renders the full chibi silhouette for a stage in the given class ramp.
func (g *generator) drawBody(c *canvas, ramp Ramp, stage int) {
	p := profileForStage(stage)
	skin := g.pal.neutral("skin")
	cx := canvasW / 2

	// --- back cape (drawn first so the body covers its top) ---
	if p.cape {
		capeShadow := withAlpha(ramp[toneShadow], 255)
		// trapezoid behind the torso, anchored at back_mount
		top := anchorBackMount.y
		bot := p.bodyBot + 6
		for y := snap(top); y < bot; y += block {
			frac := float64(y-top) / float64(bot-top)
			w := int(float64(p.bodyW+4) + frac*14)
			col := capeShadow
			if checker(0, y) {
				col = ramp[toneBase]
				col.R = scale8(col.R, 0.7)
				col.G = scale8(col.G, 0.7)
				col.B = scale8(col.B, 0.7)
			}
			c.rectBlocks(cx-w, y, cx+w, y+block, col)
		}
	}

	// --- legs ---
	legTop := p.bodyBot
	legBot := anchorFeetBaseline - 4
	legW := p.bodyW / 2
	gap := block
	// left leg
	c.rectBlocksRamp(cx-legW-gap, legTop, cx-gap, legBot, ramp)
	// right leg
	c.rectBlocksRamp(cx+gap, legTop, cx+legW+gap, legBot, ramp)
	// feet (leather, darker base)
	feet := g.pal.neutral("leather")
	c.rectBlocks(cx-legW-gap-block, legBot-block, cx-gap+block, legBot+block, feet[toneBase])
	c.rectBlocks(cx+gap-block, legBot-block, cx+legW+gap+block, legBot+block, feet[toneBase])

	// --- torso (tunic / costume) with vertical shading ---
	c.rectBlocksRamp(cx-p.bodyW, p.bodyTop, cx+p.bodyW, p.bodyBot, ramp)
	// belt band for stage>=2 (seeker gains a belt)
	if stage >= 2 {
		beltY := p.bodyBot - 10
		c.rectBlocks(cx-p.bodyW, beltY, cx+p.bodyW, beltY+block, g.pal.neutral("leather")[toneBase])
		c.rectBlocks(cx-block, beltY-block, cx+block, beltY+block*2, ramp[toneHigh]) // buckle
	}
	// chest emblem at chest_center so amulets/armor read as anchored
	c.rectBlocks(anchorChestCenter.x-block, anchorChestCenter.y-block, anchorChestCenter.x+block, anchorChestCenter.y+block, ramp[toneHigh])

	// --- pauldrons (shoulder pads) ---
	if p.pauldron {
		metal := g.pal.neutral("metal")
		c.rectBlocks(cx-p.bodyW-block, p.bodyTop, cx-p.bodyW+block*2, p.bodyTop+block*2, metal[toneLight])
		c.rectBlocks(cx+p.bodyW-block*2, p.bodyTop, cx+p.bodyW+block, p.bodyTop+block*2, metal[toneLight])
	}

	// --- arms (skin/tunic), right hand lands on hand_right anchor ---
	armW := block * 2
	armTop := p.bodyTop + block
	armBot := p.bodyBot - block*2
	// left arm
	c.rectBlocksRamp(cx-p.bodyW-armW, armTop, cx-p.bodyW, armBot, ramp)
	c.rectBlocks(cx-p.bodyW-armW, armBot-block, cx-p.bodyW, armBot+block, skin[toneLight]) // hand
	// right arm — extend down toward hand_right (96,110)
	rArmX := cx + p.bodyW
	c.rectBlocksRamp(rArmX, armTop, rArmX+armW, armBot, ramp)
	// right hand snapped to anchor
	c.rectBlocks(anchorHandRight.x-block, anchorHandRight.y-block, anchorHandRight.x+block, anchorHandRight.y+block, skin[toneLight])
	// connect forearm down to the hand anchor
	c.rectBlocks(rArmX, armBot-block, anchorHandRight.x, anchorHandRight.y, skin[toneBase])

	// --- head (skin) at head_center, with simple face ---
	g.drawHead(c, skin, p.headR)

	// --- legend flourish: a small crown of class color ---
	if p.crown {
		top := anchorHeadCenter.y - p.headR - block
		for i := -2; i <= 2; i++ {
			h := block
			if i%2 == 0 {
				h = block * 2
			}
			c.rectBlocks(anchorHeadCenter.x+i*block-block/2, top-h+block, anchorHeadCenter.x+i*block+block/2, top+block, ramp[toneHigh])
		}
	}

	// selective dark outline around the whole silhouette (docs/12 §4)
	c.selectiveOutline(g.pal.Outline)

	// soft dithered ground shadow under the figure
	c.groundShadow(g.pal.ShadowGround)
}

// drawHead paints the chibi head + readable face at head_center (docs/12 §4).
func (g *generator) drawHead(c *canvas, skin Ramp, r int) {
	hc := anchorHeadCenter
	// head fill with light-from-top shading
	c.ellipseBlocksFunc(hc.x, hc.y, r, r, func(bx, by int) (color.RGBA, bool) {
		frac := float64(by-(hc.y-r)) / float64(2*r)
		idx := int((1 - frac) * 3.999)
		if idx < 1 {
			idx = 1 // never pure shadow on the face
		}
		if idx > 3 {
			idx = 3
		}
		return skin[idx], true
	})
	// eyes (2 blocks each), whites + dark pupil
	eyeY := hc.y - block
	for _, ex := range []int{hc.x - block*2, hc.x + block} {
		c.fillBlock(ex, eyeY, g.pal.White)
		c.fillBlock(ex, eyeY+block, g.pal.Outline) // pupil
	}
	// friendly mouth (a short bright line)
	mouthY := hc.y + block*2
	c.rectBlocks(hc.x-block, mouthY, hc.x+block*2, mouthY+block, skin[toneShadow])
}
