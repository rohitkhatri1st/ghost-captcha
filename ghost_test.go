package ghostcaptcha

import (
	"bytes"
	"errors"
	"image"
	"image/gif"
	"strings"
	"testing"
)

// smallOpts keeps rendering cheap in tests: a tiny canvas and an explicit
// low frame count, since letting Frames default (one loop of the
// background scroll, i.e. Height frames at the default cell size) makes
// tests far slower than the assertions need.
func smallOpts() *GhostOptions {
	return &GhostOptions{
		FontSize: 16,
		Width:    80,
		Height:   30,
		Frames:   5,
		Format:   FormatGIF,
	}
}

func TestGenerateGhostFramesEmptyText(t *testing.T) {
	_, _, err := GenerateGhostFrames("", smallOpts())
	if err == nil {
		t.Fatal("GenerateGhostFrames(\"\"): want error, got nil")
	}
}

func TestGenerateGhostFramesDimensionsAndCount(t *testing.T) {
	opts := smallOpts()
	frames, delays, err := GenerateGhostFrames("HI", opts)
	if err != nil {
		t.Fatalf("GenerateGhostFrames: %v", err)
	}

	if len(frames) != opts.Frames {
		t.Fatalf("len(frames) = %d, want %d", len(frames), opts.Frames)
	}
	if len(delays) != opts.Frames {
		t.Fatalf("len(delays) = %d, want %d", len(delays), opts.Frames)
	}
	for i, f := range frames {
		if f.Rect.Dx() != opts.Width || f.Rect.Dy() != opts.Height {
			t.Errorf("frames[%d] size = %dx%d, want %dx%d", i, f.Rect.Dx(), f.Rect.Dy(), opts.Width, opts.Height)
		}
	}
	for i, d := range delays {
		if d != opts.FrameDelay {
			t.Errorf("delays[%d] = %d, want %d (FrameDelay default)", i, d, opts.FrameDelay)
		}
	}
}

func TestGenerateGhostFramesAppliesDefaults(t *testing.T) {
	// A zero-value GhostOptions (aside from a small Frames count, to keep
	// the test fast) should render successfully using every documented
	// default rather than erroring or panicking.
	opts := &GhostOptions{Frames: 3}
	frames, _, err := GenerateGhostFrames("X", opts)
	if err != nil {
		t.Fatalf("GenerateGhostFrames with zero-value options: %v", err)
	}
	if opts.Width <= 0 || opts.Height <= 0 {
		t.Errorf("defaults not applied: Width=%d Height=%d", opts.Width, opts.Height)
	}
	if len(frames) != 3 {
		t.Fatalf("len(frames) = %d, want 3", len(frames))
	}
}

func TestGenerateGhostFramesWidthGrowsWithTextLength(t *testing.T) {
	shortOpts := &GhostOptions{FontSize: 24, Frames: 1}
	if _, _, err := GenerateGhostFrames("A", shortOpts); err != nil {
		t.Fatalf("GenerateGhostFrames(\"A\"): %v", err)
	}

	longOpts := &GhostOptions{FontSize: 24, Frames: 1}
	if _, _, err := GenerateGhostFrames("ABCDEFGHIJKLMNOP", longOpts); err != nil {
		t.Fatalf("GenerateGhostFrames(long text): %v", err)
	}

	if longOpts.Width <= shortOpts.Width {
		t.Errorf("longer text should default to a wider canvas: short.Width=%d, long.Width=%d",
			shortOpts.Width, longOpts.Width)
	}
	// Height must not be affected by line length, only by line count.
	if longOpts.Height != shortOpts.Height {
		t.Errorf("single-line text of any length should default to the same height: short.Height=%d, long.Height=%d",
			shortOpts.Height, longOpts.Height)
	}
}

func TestGenerateGhostFramesHeightGrowsWithLineCount(t *testing.T) {
	oneLine := &GhostOptions{FontSize: 24, Frames: 1}
	if _, _, err := GenerateGhostFrames("A", oneLine); err != nil {
		t.Fatalf("GenerateGhostFrames(\"A\"): %v", err)
	}

	threeLines := &GhostOptions{FontSize: 24, Frames: 1}
	if _, _, err := GenerateGhostFrames("A\nB\nC", threeLines); err != nil {
		t.Fatalf("GenerateGhostFrames(\"A\\nB\\nC\"): %v", err)
	}

	if threeLines.Height <= oneLine.Height {
		t.Errorf("more lines should default to a taller canvas: oneLine.Height=%d, threeLines.Height=%d",
			oneLine.Height, threeLines.Height)
	}
}

func TestGenerateGhostFramesInvalidFont(t *testing.T) {
	opts := smallOpts()
	opts.FontBytes = []byte("not a font")
	_, _, err := GenerateGhostFrames("HI", opts)
	if err == nil {
		t.Fatal("GenerateGhostFrames with invalid FontBytes: want error, got nil")
	}
}

func TestGenerateGhostFramesNormalizesLineEndings(t *testing.T) {
	// The font has no glyph for raw \r or \t; GenerateGhostFrames must
	// normalize them (via lineEndingReplacer, see TestLineEndingReplacer)
	// before they reach the font renderer instead of erroring.
	opts := smallOpts()
	if _, _, err := GenerateGhostFrames("A\r\nB\tC\rD", opts); err != nil {
		t.Fatalf("GenerateGhostFrames with CRLF/CR/tab text: %v", err)
	}
}

func TestGenerateGhostGIFRoundTrip(t *testing.T) {
	opts := smallOpts()
	data, err := GenerateGhost("HI", opts)
	if err != nil {
		t.Fatalf("GenerateGhost: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("GenerateGhost returned empty data")
	}

	decoded, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decoding GenerateGhost's output as GIF: %v", err)
	}
	if len(decoded.Image) != opts.Frames {
		t.Errorf("decoded frame count = %d, want %d", len(decoded.Image), opts.Frames)
	}
	if decoded.Config.Width != opts.Width || decoded.Config.Height != opts.Height {
		t.Errorf("decoded dimensions = %dx%d, want %dx%d",
			decoded.Config.Width, decoded.Config.Height, opts.Width, opts.Height)
	}
	if decoded.LoopCount != opts.Loop {
		t.Errorf("decoded LoopCount = %d, want %d", decoded.LoopCount, opts.Loop)
	}
}

func TestGenerateGhostEmptyText(t *testing.T) {
	_, err := GenerateGhost("", smallOpts())
	if err == nil {
		t.Fatal("GenerateGhost(\"\"): want error, got nil")
	}
}

func TestGenerateGhostCustomEncoder(t *testing.T) {
	opts := smallOpts()
	var gotFrames []*image.Paletted
	var gotDelays []int
	gotOpts := false
	opts.Encoder = func(frames []*image.Paletted, delays []int, o *GhostOptions) ([]byte, error) {
		gotFrames = frames
		gotDelays = delays
		gotOpts = o == opts
		return []byte("custom-output"), nil
	}

	data, err := GenerateGhost("HI", opts)
	if err != nil {
		t.Fatalf("GenerateGhost with custom Encoder: %v", err)
	}
	if string(data) != "custom-output" {
		t.Errorf("data = %q, want %q", data, "custom-output")
	}
	if len(gotFrames) != opts.Frames {
		t.Errorf("encoder received %d frames, want %d", len(gotFrames), opts.Frames)
	}
	if len(gotDelays) != opts.Frames {
		t.Errorf("encoder received %d delays, want %d", len(gotDelays), opts.Frames)
	}
	if !gotOpts {
		t.Error("encoder did not receive the same *GhostOptions passed to GenerateGhost")
	}
}

func TestGenerateGhostCustomEncoderError(t *testing.T) {
	opts := smallOpts()
	wantErr := errors.New("boom")
	opts.Encoder = func(frames []*image.Paletted, delays []int, o *GhostOptions) ([]byte, error) {
		return nil, wantErr
	}

	_, err := GenerateGhost("HI", opts)
	if err == nil {
		t.Fatal("GenerateGhost: want error from failing encoder, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("error = %v, want it to wrap %v", err, wantErr)
	}
}

func TestGenerateGhostCustomEncoderSkipsFFmpegCheck(t *testing.T) {
	// A custom Encoder is responsible for its own dependencies, so
	// GenerateGhost must not fail the ffmpeg-on-PATH preflight check for
	// video formats when Encoder is set - even with PATH emptied out.
	opts := smallOpts()
	opts.Format = FormatWebM
	called := false
	opts.Encoder = func(frames []*image.Paletted, delays []int, o *GhostOptions) ([]byte, error) {
		called = true
		return []byte("ok"), nil
	}

	t.Setenv("PATH", "")
	data, err := GenerateGhost("HI", opts)
	if err != nil {
		t.Fatalf("GenerateGhost with custom encoder and empty PATH: %v", err)
	}
	if !called {
		t.Error("custom encoder was not called")
	}
	if string(data) != "ok" {
		t.Errorf("data = %q, want %q", data, "ok")
	}
}

func TestGenerateGhostMissingFFmpeg(t *testing.T) {
	// Setting PATH to empty makes exec.LookPath fail to find ffmpeg
	// regardless of whether the host actually has it installed, so this
	// test doesn't need to skip based on the host's environment.
	opts := smallOpts()
	opts.Format = FormatWebM
	t.Setenv("PATH", "")

	_, err := GenerateGhost("HI", opts)
	if err == nil {
		t.Fatal("GenerateGhost with FormatWebM and empty PATH: want error, got nil")
	}
	if !strings.Contains(err.Error(), "ffmpeg") {
		t.Errorf("error = %v, want it to mention ffmpeg", err)
	}
}

func TestDefaultFrameEncoder(t *testing.T) {
	if got := defaultFrameEncoder(FormatGIF); got == nil {
		t.Error("defaultFrameEncoder(FormatGIF) = nil")
	}
	if got := defaultFrameEncoder(FormatWebM); got == nil {
		t.Error("defaultFrameEncoder(FormatWebM) = nil")
	}
	if got := defaultFrameEncoder(FormatMP4); got == nil {
		t.Error("defaultFrameEncoder(FormatMP4) = nil")
	}
}

func TestRenderGhostFrameSingleFrameIsUniformDistribution(t *testing.T) {
	// The core anti-OCR property: every pixel, text or background, draws
	// from the same palette. This doesn't prove indistinguishability
	// statistically, but it does pin down that renderGhostFrame never
	// special-cases text pixels into a different value range.
	opts := smallOpts()
	frames, _, err := GenerateGhostFrames("W", opts)
	if err != nil {
		t.Fatalf("GenerateGhostFrames: %v", err)
	}
	paletteLen := len(frames[0].Palette)
	for _, f := range frames {
		for _, idx := range f.Pix {
			if int(idx) >= paletteLen {
				t.Fatalf("pixel index %d out of palette range [0, %d)", idx, paletteLen)
			}
		}
	}
}

func TestGenerateGhostFramesAnimates(t *testing.T) {
	// At least some pixels must differ between frames - otherwise the
	// output isn't actually an animation.
	opts := smallOpts()
	opts.Frames = 3
	frames, _, err := GenerateGhostFrames("HI", opts)
	if err != nil {
		t.Fatalf("GenerateGhostFrames: %v", err)
	}
	if len(frames) < 2 {
		t.Fatal("need at least 2 frames to check animation")
	}
	if bytes.Equal(frames[0].Pix, frames[1].Pix) {
		t.Error("frame 0 and frame 1 are pixel-identical; expected the noise to scroll between frames")
	}
}
