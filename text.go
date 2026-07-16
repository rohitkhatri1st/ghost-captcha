package ghostcaptcha

import (
	"image"
	"math"
	"math/rand/v2"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// textShape rasterizes text once, at face's size, into a tight alpha mask:
// mask.AlphaAt(x, y).A > 0 marks a pixel as part of a letterform. The mask
// is independent of any frame — renderers place a copy of it at a
// per-frame anchor (see textFrameAnchors) rather than re-rasterizing text
// for every frame.
//
// text may contain "\n" to break it into multiple lines (lineEndingReplacer
// normalizes CRLF/CR to "\n" upstream, before text reaches here): each line
// is measured and drawn independently, stacked top to bottom, so the mask
// grows taller with more lines. The mask's width is its widest line, so it
// grows wider with longer lines rather than with the text's total length.
//
// letterSpacing adds that many extra pixels between each pair of adjacent
// characters on a line, on top of the font's own advance width; characters
// are drawn one at a time so that extra gap can be inserted between them.
func textShape(face font.Face, text string, letterSpacing int) *image.Alpha {
	metrics := face.Metrics()
	lineHeight := (metrics.Ascent + metrics.Descent).Ceil()
	lines := strings.Split(text, "\n")

	width := 0
	for _, line := range lines {
		lineWidth := font.MeasureString(face, line).Ceil()
		if runes := []rune(line); len(runes) > 1 {
			lineWidth += letterSpacing * (len(runes) - 1)
		}
		width = max(width, lineWidth)
	}
	width = max(width, 1)
	height := max(lineHeight*len(lines), 1)

	mask := image.NewAlpha(image.Rect(0, 0, width, height))
	d := &font.Drawer{
		Dst:  mask,
		Src:  image.Opaque,
		Face: face,
	}
	spacing := fixed.I(letterSpacing)
	for lineIdx, line := range lines {
		d.Dot = fixed.Point26_6{X: 0, Y: metrics.Ascent + fixed.I(lineIdx*lineHeight)}
		runes := []rune(line)
		for i, r := range runes {
			d.DrawString(string(r))
			if i < len(runes)-1 {
				d.Dot.X += spacing
			}
		}
	}
	return mask
}

// maxDriftHarmonic bounds how many times textFrameAnchors' path can
// oscillate across one loop, per axis. Higher values trace more tangled
// curves (more crossings); this is a small range so the drift stays a
// gentle wander rather than a frantic zig-zag.
const maxDriftHarmonic = 3

// minDriftAmplitudeFrac keeps a randomized drift axis from rounding away
// to an imperceptible sub-pixel wobble: an active axis's amplitude is
// drawn from at least this fraction of its available range, up to all of it.
const minDriftAmplitudeFrac = 0.35

// randDriftAmplitude draws a random amplitude in
// [minDriftAmplitudeFrac*max, max]. Callers that want an axis to not move
// at all set its amplitude to exactly 0 afterward instead of drawing here.
func randDriftAmplitude(max float64) float64 {
	if max <= 0 {
		return 0
	}
	return max * (minDriftAmplitudeFrac + (1-minDriftAmplitudeFrac)*rand.Float64())
}

// textFrameAnchors returns the text shape's top-left position for each of
// frameCount frames (pass the same frameCount renderNoiseFrames resolved
// for the background, via backgroundFrameCount, so both loops close after
// the same number of frames), following the path drift selects.
//
// Every non-fixed mode traces the shape's path as a Lissajous curve around
// the canvas center: x and y each oscillate at their own integer harmonic,
// amplitude, and phase. Whatever the parameters, every term is a
// sine/cosine at an integer multiple of the base frequency
// 2*pi/frameCount, so the combined curve is exactly periodic over
// frameCount samples: it starts near — and sometimes exactly at — the
// center, wanders, and returns to precisely its starting point with the
// same per-frame step as every other transition, closing the loop with no
// visible jump.
func textFrameAnchors(canvasWidth, canvasHeight, shapeWidth, shapeHeight, frameCount int, drift TextDrift) []image.Point {
	centerX := (canvasWidth - shapeWidth) / 2
	centerY := (canvasHeight - shapeHeight) / 2

	if drift == TextDriftFixed {
		anchors := make([]image.Point, frameCount)
		for i := range anchors {
			anchors[i] = image.Point{X: centerX, Y: centerY}
		}
		return anchors
	}

	maxAmpX := math.Max(0, float64(canvasWidth-shapeWidth)/4)
	maxAmpY := math.Max(0, float64(canvasHeight-shapeHeight)/4)

	// TextDriftEllipse/Horizontal/Vertical each trace a single harmonic;
	// only TextDriftRandom reaches for higher harmonics, since that's
	// what produces the more tangled figure-eight/rosette shapes the
	// other named modes can't.
	kx, ky := 1, 1
	if drift == TextDriftRandom {
		kx = 1 + rand.IntN(maxDriftHarmonic)
		ky = 1 + rand.IntN(maxDriftHarmonic)
	}
	ampX := randDriftAmplitude(maxAmpX)
	ampY := randDriftAmplitude(maxAmpY)
	phaseX := rand.Float64() * 2 * math.Pi
	phaseY := rand.Float64() * 2 * math.Pi

	switch drift {
	case TextDriftHorizontal:
		ampY = 0
	case TextDriftVertical:
		ampX = 0
	case TextDriftRandom:
		switch rand.IntN(3) { // sometimes collapse to a single flat axis
		case 0:
			ampY = 0
		case 1:
			ampX = 0
		}
	}

	anchors := make([]image.Point, frameCount)
	for i := range anchors {
		t := 2 * math.Pi * float64(i) / float64(frameCount)
		anchors[i] = image.Point{
			X: centerX + int(math.Round(ampX*math.Cos(float64(kx)*t+phaseX))),
			Y: centerY + int(math.Round(ampY*math.Sin(float64(ky)*t+phaseY))),
		}
	}
	return anchors
}

// isTextPixel reports whether canvas pixel (x, y) lands on a letterform
// when shape is anchored at anchor — i.e. whether (x, y) is "text" in the
// frame anchor belongs to.
func isTextPixel(shape *image.Alpha, anchor image.Point, x, y int) bool {
	sx, sy := x-anchor.X, y-anchor.Y
	if sx < 0 || sy < 0 || sx >= shape.Rect.Dx() || sy >= shape.Rect.Dy() {
		return false
	}
	return shape.AlphaAt(sx, sy).A > 0
}
