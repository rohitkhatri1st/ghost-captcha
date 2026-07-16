package ghostcaptcha

import (
	"image"
	"image/color"
	"math"
	"testing"

	"golang.org/x/image/font"
)

func testFace(t *testing.T) font.Face {
	t.Helper()
	face, err := loadFace(nil, 24)
	if err != nil {
		t.Fatalf("loadFace: %v", err)
	}
	t.Cleanup(func() { face.Close() })
	return face
}

// maskHasInk reports whether any pixel in mask is opaque, i.e. the
// rasterized glyphs actually drew visible pixels rather than just
// reserving blank advance width.
func maskHasInk(mask *image.Alpha) bool {
	for _, a := range mask.Pix {
		if a > 0 {
			return true
		}
	}
	return false
}

func TestTextShapeDimensions(t *testing.T) {
	face := testFace(t)

	single := textShape(face, "A", 0)
	if single.Rect.Dx() <= 0 || single.Rect.Dy() <= 0 {
		t.Fatalf("textShape(\"A\") rect = %v, want positive dimensions", single.Rect)
	}
	if !maskHasInk(single) {
		t.Error("textShape(\"A\") produced no visible ink pixels")
	}

	double := textShape(face, "AA", 0)
	if double.Rect.Dx() <= single.Rect.Dx() {
		t.Errorf("textShape(\"AA\") width %d should exceed textShape(\"A\") width %d",
			double.Rect.Dx(), single.Rect.Dx())
	}
	// Height only depends on the face's metrics, not the text.
	if double.Rect.Dy() != single.Rect.Dy() {
		t.Errorf("textShape(\"AA\") height %d != textShape(\"A\") height %d",
			double.Rect.Dy(), single.Rect.Dy())
	}
}

func TestTextShapeLetterSpacing(t *testing.T) {
	face := testFace(t)

	noSpacing := textShape(face, "AB", 0)
	withSpacing := textShape(face, "AB", 10)

	if withSpacing.Rect.Dx() != noSpacing.Rect.Dx()+10 {
		t.Errorf("letterSpacing=10 width = %d, want %d (base %d + 10)",
			withSpacing.Rect.Dx(), noSpacing.Rect.Dx()+10, noSpacing.Rect.Dx())
	}
}

func TestTextShapeMinimumWidth(t *testing.T) {
	face := testFace(t)
	// Negative letter spacing large enough to overwhelm any advance width
	// should still clamp to a width of at least 1, never zero or negative.
	shape := textShape(face, "AB", -1000)
	if shape.Rect.Dx() < 1 {
		t.Errorf("textShape width = %d, want >= 1", shape.Rect.Dx())
	}
}

func TestTextShapeMultiLineHeight(t *testing.T) {
	face := testFace(t)

	oneLine := textShape(face, "A", 0)
	threeLines := textShape(face, "A\nB\nC", 0)

	if threeLines.Rect.Dy() != oneLine.Rect.Dy()*3 {
		t.Errorf("3-line height = %d, want %d (3x one line's %d)",
			threeLines.Rect.Dy(), oneLine.Rect.Dy()*3, oneLine.Rect.Dy())
	}
}

func TestTextShapeMultiLineWidthIsWidestLine(t *testing.T) {
	face := testFace(t)

	// The second line ("AB") is shorter than the first ("ABCD"); overall
	// width must be the widest line, not a sum or an average of lines.
	multiLine := textShape(face, "ABCD\nAB", 0)
	wantWidth := textShape(face, "ABCD", 0)

	if multiLine.Rect.Dx() != wantWidth.Rect.Dx() {
		t.Errorf("multi-line width = %d, want %d (width of the longest line)",
			multiLine.Rect.Dx(), wantWidth.Rect.Dx())
	}
}

func TestTextShapeMultiLineDrawsEveryLine(t *testing.T) {
	face := testFace(t)
	shape := textShape(face, "A\nB", 0)

	lineHeight := shape.Rect.Dy() / 2
	topHalf := image.NewAlpha(image.Rect(0, 0, shape.Rect.Dx(), lineHeight))
	bottomHalf := image.NewAlpha(image.Rect(0, 0, shape.Rect.Dx(), lineHeight))
	for y := 0; y < lineHeight; y++ {
		for x := 0; x < shape.Rect.Dx(); x++ {
			topHalf.SetAlpha(x, y, shape.AlphaAt(x, y))
			bottomHalf.SetAlpha(x, y, shape.AlphaAt(x, y+lineHeight))
		}
	}

	if !maskHasInk(topHalf) {
		t.Error("first line ('A') drew no ink in the mask's top half")
	}
	if !maskHasInk(bottomHalf) {
		t.Error("second line ('B') drew no ink in the mask's bottom half")
	}
}

func TestIsTextPixel(t *testing.T) {
	mask := image.NewAlpha(image.Rect(0, 0, 4, 4))
	mask.SetAlpha(1, 2, color.Alpha{A: 255})
	anchor := image.Point{X: 10, Y: 20}

	tests := []struct {
		name string
		x, y int
		want bool
	}{
		{"inside, opaque pixel", 11, 22, true},
		{"inside, transparent pixel", 10, 20, false},
		{"before anchor x", 9, 22, false},
		{"before anchor y", 11, 19, false},
		{"past shape width", 14, 22, false},
		{"past shape height", 11, 24, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isTextPixel(mask, anchor, tt.x, tt.y); got != tt.want {
				t.Errorf("isTextPixel(%d, %d) = %v, want %v", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func TestTextFrameAnchorsFixed(t *testing.T) {
	const frameCount = 50
	anchors := textFrameAnchors(400, 100, 60, 20, frameCount, TextDriftFixed)

	if len(anchors) != frameCount {
		t.Fatalf("len(anchors) = %d, want %d", len(anchors), frameCount)
	}
	want := image.Point{X: (400 - 60) / 2, Y: (100 - 20) / 2}
	for i, a := range anchors {
		if a != want {
			t.Errorf("anchors[%d] = %v, want %v (fixed drift never moves)", i, a, want)
		}
	}
}

func TestTextFrameAnchorsHorizontalHasNoVerticalMovement(t *testing.T) {
	const frameCount = 40
	centerY := (100 - 20) / 2
	anchors := textFrameAnchors(400, 100, 60, 20, frameCount, TextDriftHorizontal)

	if len(anchors) != frameCount {
		t.Fatalf("len(anchors) = %d, want %d", len(anchors), frameCount)
	}
	for i, a := range anchors {
		if a.Y != centerY {
			t.Errorf("anchors[%d].Y = %d, want %d (horizontal drift must not move vertically)", i, a.Y, centerY)
		}
	}
}

func TestTextFrameAnchorsVerticalHasNoHorizontalMovement(t *testing.T) {
	const frameCount = 40
	centerX := (400 - 60) / 2
	anchors := textFrameAnchors(400, 100, 60, 20, frameCount, TextDriftVertical)

	if len(anchors) != frameCount {
		t.Fatalf("len(anchors) = %d, want %d", len(anchors), frameCount)
	}
	for i, a := range anchors {
		if a.X != centerX {
			t.Errorf("anchors[%d].X = %d, want %d (vertical drift must not move horizontally)", i, a.X, centerX)
		}
	}
}

func TestTextFrameAnchorsEllipseStaysInBounds(t *testing.T) {
	const (
		canvasWidth, canvasHeight = 400, 100
		shapeWidth, shapeHeight   = 60, 20
		frameCount                = 60
	)
	centerX := (canvasWidth - shapeWidth) / 2
	centerY := (canvasHeight - shapeHeight) / 2
	maxAmpX := math.Max(0, float64(canvasWidth-shapeWidth)/4)
	maxAmpY := math.Max(0, float64(canvasHeight-shapeHeight)/4)

	// Run several times since the amplitude/phase are randomized per call.
	for trial := 0; trial < 20; trial++ {
		anchors := textFrameAnchors(canvasWidth, canvasHeight, shapeWidth, shapeHeight, frameCount, TextDriftEllipse)
		if len(anchors) != frameCount {
			t.Fatalf("len(anchors) = %d, want %d", len(anchors), frameCount)
		}
		for i, a := range anchors {
			if math.Abs(float64(a.X-centerX)) > maxAmpX+1 {
				t.Errorf("trial %d anchors[%d].X = %d strays beyond centerX(%d) +/- maxAmpX(%v)",
					trial, i, a.X, centerX, maxAmpX)
			}
			if math.Abs(float64(a.Y-centerY)) > maxAmpY+1 {
				t.Errorf("trial %d anchors[%d].Y = %d strays beyond centerY(%d) +/- maxAmpY(%v)",
					trial, i, a.Y, centerY, maxAmpY)
			}
		}
	}
}

func TestTextFrameAnchorsZeroFrames(t *testing.T) {
	for _, drift := range []TextDrift{TextDriftFixed, TextDriftRandom, TextDriftEllipse, TextDriftHorizontal, TextDriftVertical} {
		anchors := textFrameAnchors(400, 100, 60, 20, 0, drift)
		if len(anchors) != 0 {
			t.Errorf("drift=%v: len(anchors) = %d, want 0", drift, len(anchors))
		}
	}
}

func TestRandDriftAmplitude(t *testing.T) {
	if got := randDriftAmplitude(0); got != 0 {
		t.Errorf("randDriftAmplitude(0) = %v, want 0", got)
	}
	if got := randDriftAmplitude(-5); got != 0 {
		t.Errorf("randDriftAmplitude(-5) = %v, want 0", got)
	}

	const max = 100.0
	for i := 0; i < 200; i++ {
		got := randDriftAmplitude(max)
		if got < minDriftAmplitudeFrac*max || got > max {
			t.Fatalf("randDriftAmplitude(%v) = %v, want in [%v, %v]",
				max, got, minDriftAmplitudeFrac*max, max)
		}
	}
}
