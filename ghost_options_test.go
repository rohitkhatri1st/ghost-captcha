package ghostcaptcha

import (
	"image/color"
	"math"
	"testing"
)

func TestSetDefaultsAppliesZeroValueDefaults(t *testing.T) {
	opts := &GhostOptions{}
	opts.setDefaults()

	if opts.FontSize != defaultFontSize {
		t.Errorf("FontSize = %v, want %v", opts.FontSize, defaultFontSize)
	}
	// setDefaults never touches Width/Height - those need the rendered
	// text shape, which only setCanvasDefaults (tested separately below)
	// has access to.
	if opts.Width != 0 {
		t.Errorf("Width = %v, want 0 (untouched by setDefaults)", opts.Width)
	}
	if opts.Height != 0 {
		t.Errorf("Height = %v, want 0 (untouched by setDefaults)", opts.Height)
	}
	if opts.NoiseColorA != color.Black {
		t.Errorf("NoiseColorA = %v, want color.Black", opts.NoiseColorA)
	}
	if opts.NoiseColorB != color.White {
		t.Errorf("NoiseColorB = %v, want color.White", opts.NoiseColorB)
	}
	if opts.BackgroundCellSize != 1 {
		t.Errorf("BackgroundCellSize = %v, want 1", opts.BackgroundCellSize)
	}
	if opts.TextCellSize != 1 {
		t.Errorf("TextCellSize = %v, want 1", opts.TextCellSize)
	}
	if opts.FrameDelay != 4 {
		t.Errorf("FrameDelay = %v, want 4", opts.FrameDelay)
	}
}

func TestSetDefaultsPreservesExplicitValues(t *testing.T) {
	opts := &GhostOptions{
		FontSize:           100,
		Width:              500,
		Height:             200,
		NoiseColorA:        color.RGBA{R: 1, G: 2, B: 3, A: 255},
		NoiseColorB:        color.RGBA{R: 4, G: 5, B: 6, A: 255},
		BackgroundCellSize: 3,
		TextCellSize:       5,
		FrameDelay:         10,
	}
	opts.setDefaults()

	if opts.FontSize != 100 {
		t.Errorf("FontSize = %v, want 100", opts.FontSize)
	}
	if opts.Width != 500 {
		t.Errorf("Width = %v, want 500", opts.Width)
	}
	if opts.Height != 200 {
		t.Errorf("Height = %v, want 200", opts.Height)
	}
	if opts.NoiseColorA != (color.RGBA{R: 1, G: 2, B: 3, A: 255}) {
		t.Errorf("NoiseColorA = %v, want unchanged", opts.NoiseColorA)
	}
	if opts.NoiseColorB != (color.RGBA{R: 4, G: 5, B: 6, A: 255}) {
		t.Errorf("NoiseColorB = %v, want unchanged", opts.NoiseColorB)
	}
	if opts.BackgroundCellSize != 3 {
		t.Errorf("BackgroundCellSize = %v, want 3", opts.BackgroundCellSize)
	}
	if opts.TextCellSize != 5 {
		t.Errorf("TextCellSize = %v, want 5", opts.TextCellSize)
	}
	if opts.FrameDelay != 10 {
		t.Errorf("FrameDelay = %v, want 10", opts.FrameDelay)
	}
}

func TestSetCanvasDefaultsFitsShapePlusMargin(t *testing.T) {
	tests := []struct {
		fontSize                float64
		shapeWidth, shapeHeight int
	}{
		{defaultFontSize, 252, 82},
		{defaultFontSize * 2, 504, 164},
		{100, 10, 5},
	}
	for _, tt := range tests {
		opts := &GhostOptions{FontSize: tt.fontSize}
		opts.setCanvasDefaults(tt.shapeWidth, tt.shapeHeight)

		wantWidth := tt.shapeWidth + int(math.Round(tt.fontSize*canvasMarginXFactor))
		wantHeight := tt.shapeHeight + int(math.Round(tt.fontSize*canvasMarginYFactor))
		if opts.Width != wantWidth {
			t.Errorf("FontSize=%v shape=%dx%d: Width = %d, want %d", tt.fontSize, tt.shapeWidth, tt.shapeHeight, opts.Width, wantWidth)
		}
		if opts.Height != wantHeight {
			t.Errorf("FontSize=%v shape=%dx%d: Height = %d, want %d", tt.fontSize, tt.shapeWidth, tt.shapeHeight, opts.Height, wantHeight)
		}
	}
}

func TestSetCanvasDefaultsGrowsWithShapeSize(t *testing.T) {
	const fontSize = 70
	small := &GhostOptions{FontSize: fontSize}
	small.setCanvasDefaults(50, 20)

	large := &GhostOptions{FontSize: fontSize}
	large.setCanvasDefaults(500, 60)

	if large.Width <= small.Width {
		t.Errorf("wider shape should default to a wider canvas: small.Width=%d, large.Width=%d", small.Width, large.Width)
	}
	if large.Height <= small.Height {
		t.Errorf("taller shape should default to a taller canvas: small.Height=%d, large.Height=%d", small.Height, large.Height)
	}
}

func TestSetCanvasDefaultsPreservesExplicitValues(t *testing.T) {
	opts := &GhostOptions{FontSize: 70, Width: 999, Height: 888}
	opts.setCanvasDefaults(50, 20)

	if opts.Width != 999 {
		t.Errorf("Width = %v, want 999 (explicit value must not be overridden)", opts.Width)
	}
	if opts.Height != 888 {
		t.Errorf("Height = %v, want 888 (explicit value must not be overridden)", opts.Height)
	}
}

func TestFormatString(t *testing.T) {
	tests := []struct {
		format Format
		want   string
	}{
		{FormatWebM, "webm"},
		{FormatGIF, "gif"},
		{FormatMP4, "mp4"},
		{Format(99), "webm"}, // unknown values fall back to the default case
	}
	for _, tt := range tests {
		if got := tt.format.String(); got != tt.want {
			t.Errorf("Format(%d).String() = %q, want %q", tt.format, got, tt.want)
		}
	}
}
