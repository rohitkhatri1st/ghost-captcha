package ghostcaptcha

import (
	"image"
	"image/color"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func testPalette() color.Palette {
	return color.Palette{
		color.RGBA{R: 0, G: 0, B: 0, A: 255},       // index 0: black
		color.RGBA{R: 255, G: 255, B: 255, A: 255}, // index 1: white
		color.RGBA{R: 255, G: 0, B: 0, A: 255},     // index 2: red
	}
}

func TestRGB24LUT(t *testing.T) {
	lut := rgb24LUT(testPalette())
	if len(lut) != 3 {
		t.Fatalf("len(lut) = %d, want 3", len(lut))
	}
	want := [][3]byte{
		{0, 0, 0},
		{255, 255, 255},
		{255, 0, 0},
	}
	for i, w := range want {
		if lut[i] != w {
			t.Errorf("lut[%d] = %v, want %v", i, lut[i], w)
		}
	}
}

func TestFramesToRawRGB24Empty(t *testing.T) {
	if got := framesToRawRGB24(nil, 2, 2); got != nil {
		t.Errorf("framesToRawRGB24(nil frames) = %v, want nil", got)
	}
}

func TestFramesToRawRGB24UnevenWorkerSplit(t *testing.T) {
	// With 7 frames split across 5 workers, chunk = ceil(7/5) = 2, so the
	// last worker's range starts at 4*2 = 8, past the last frame (index
	// 6) - framesToRawRGB24 must skip that worker instead of writing out
	// of bounds. Pinning GOMAXPROCS is the only way to force this uneven
	// split deterministically.
	prev := runtime.GOMAXPROCS(5)
	defer runtime.GOMAXPROCS(prev)

	const n, w, h = 7, 2, 2
	palette := testPalette()
	frames := make([]*image.Paletted, n)
	for i := range frames {
		img := image.NewPaletted(image.Rect(0, 0, w, h), palette)
		for p := range img.Pix {
			img.Pix[p] = uint8((i + p) % len(palette))
		}
		frames[i] = img
	}

	raw := framesToRawRGB24(frames, w, h)
	frameSize := w * h * 3
	if len(raw) != n*frameSize {
		t.Fatalf("len(raw) = %d, want %d", len(raw), n*frameSize)
	}

	lut := rgb24LUT(palette)
	for i, f := range frames {
		for p, idx := range f.Pix {
			want := lut[idx]
			got := raw[i*frameSize+p*3 : i*frameSize+p*3+3]
			if got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
				t.Errorf("frame %d pixel %d = %v, want %v", i, p, got, want)
			}
		}
	}
}

func TestFramesToRawRGB24Layout(t *testing.T) {
	const w, h = 2, 2
	palette := testPalette()

	frame0 := image.NewPaletted(image.Rect(0, 0, w, h), palette)
	copy(frame0.Pix, []uint8{0, 1, 2, 0}) // black, white, red, black

	frame1 := image.NewPaletted(image.Rect(0, 0, w, h), palette)
	copy(frame1.Pix, []uint8{2, 2, 1, 1}) // red, red, white, white

	raw := framesToRawRGB24([]*image.Paletted{frame0, frame1}, w, h)

	frameSize := w * h * 3
	if len(raw) != 2*frameSize {
		t.Fatalf("len(raw) = %d, want %d", len(raw), 2*frameSize)
	}

	want := []byte{
		0, 0, 0, // black
		255, 255, 255, // white
		255, 0, 0, // red
		0, 0, 0, // black

		255, 0, 0, // red
		255, 0, 0, // red
		255, 255, 255, // white
		255, 255, 255, // white
	}
	for i := range want {
		if raw[i] != want[i] {
			t.Fatalf("raw[%d] = %d, want %d (full raw = %v)", i, raw[i], want[i], raw)
		}
	}
}

func TestEncodeVideoMissingFFmpeg(t *testing.T) {
	t.Setenv("PATH", "")
	_, err := encodeVideo(nil, 2, 2, "25/1", FormatWebM)
	if err == nil {
		t.Fatal("encodeVideo with empty PATH: want error, got nil")
	}
	if !strings.Contains(err.Error(), "ffmpeg") {
		t.Errorf("error = %v, want it to mention ffmpeg", err)
	}
}

func tinyFrames(t *testing.T) []*image.Paletted {
	t.Helper()
	palette := buildNoisePalette(color.Black, color.White, noisePaletteSteps)
	frames := make([]*image.Paletted, 3)
	for i := range frames {
		img := image.NewPaletted(image.Rect(0, 0, 4, 4), palette)
		for p := range img.Pix {
			img.Pix[p] = uint8((p + i) % noisePaletteSteps)
		}
		frames[i] = img
	}
	return frames
}

func TestEncodeVideoWebM(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not on PATH")
	}
	data, err := encodeVideo(tinyFrames(t), 4, 4, "25/1", FormatWebM)
	if err != nil {
		t.Fatalf("encodeVideo(FormatWebM): %v", err)
	}
	if len(data) == 0 {
		t.Fatal("encodeVideo(FormatWebM) returned empty data")
	}
	// WebM/Matroska containers start with the EBML header magic number.
	ebmlMagic := []byte{0x1A, 0x45, 0xDF, 0xA3}
	if len(data) < 4 || string(data[:4]) != string(ebmlMagic) {
		t.Errorf("output does not start with the EBML magic bytes: got %v", data[:min(8, len(data))])
	}
}

func TestEncodeVideoMP4(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not on PATH")
	}
	data, err := encodeVideo(tinyFrames(t), 4, 4, "25/1", FormatMP4)
	if err != nil {
		t.Fatalf("encodeVideo(FormatMP4): %v", err)
	}
	if len(data) == 0 {
		t.Fatal("encodeVideo(FormatMP4) returned empty data")
	}
	// ISO base media files lead with a box size (4 bytes) followed by a
	// four-character box type; the first box in an MP4 is always "ftyp".
	if len(data) < 8 || string(data[4:8]) != "ftyp" {
		t.Errorf("output does not start with an ftyp box: got %v", data[:min(16, len(data))])
	}
}
