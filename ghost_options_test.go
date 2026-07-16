package ghostcaptcha

import (
	"image/color"
	"testing"
)

func TestSetDefaultsAppliesZeroValueDefaults(t *testing.T) {
	opts := &GhostOptions{}
	opts.setDefaults()

	if opts.FontSize != defaultFontSize {
		t.Errorf("FontSize = %v, want %v", opts.FontSize, defaultFontSize)
	}
	if opts.Width != defaultWidth {
		t.Errorf("Width = %v, want %v", opts.Width, defaultWidth)
	}
	if opts.Height != defaultHeight {
		t.Errorf("Height = %v, want %v", opts.Height, defaultHeight)
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

func TestSetDefaultsWidthHeightScaleWithFontSize(t *testing.T) {
	tests := []struct {
		fontSize              float64
		wantWidth, wantHeight int
	}{
		{defaultFontSize, defaultWidth, defaultHeight},
		{defaultFontSize * 2, defaultWidth * 2, defaultHeight * 2},
		{defaultFontSize / 2, defaultWidth / 2, defaultHeight / 2},
	}
	for _, tt := range tests {
		opts := &GhostOptions{FontSize: tt.fontSize}
		opts.setDefaults()
		if opts.Width != tt.wantWidth {
			t.Errorf("FontSize=%v: Width = %d, want %d", tt.fontSize, opts.Width, tt.wantWidth)
		}
		if opts.Height != tt.wantHeight {
			t.Errorf("FontSize=%v: Height = %d, want %d", tt.fontSize, opts.Height, tt.wantHeight)
		}
	}
}

func TestSetDefaultsZeroFontSizeUsesDefaultCanvas(t *testing.T) {
	// FontSize left at its zero value must resolve to defaultFontSize
	// before Width/Height are derived, not divide-by-zero or scale off
	// the raw zero value.
	opts := &GhostOptions{}
	opts.setDefaults()
	if opts.Width != defaultWidth || opts.Height != defaultHeight {
		t.Errorf("zero FontSize: Width=%d Height=%d, want %d,%d", opts.Width, opts.Height, defaultWidth, defaultHeight)
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
