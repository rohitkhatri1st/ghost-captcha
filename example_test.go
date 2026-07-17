package ghostcaptcha_test

import (
	"fmt"
	"image"

	ghostcaptcha "github.com/rohitkhatri1st/ghost-captcha"
)

// Basic usage: draw random captcha text and render it. text is what the
// caller checks a user's answer against; data is the encoded image to show
// them. Format defaults to FormatGIF, so this example needs no ffmpeg.
func ExampleGenerateCaptcha() {
	text, data, err := ghostcaptcha.GenerateCaptcha(ghostcaptcha.CaptchaOptions{
		// Frames is capped only to keep this example quick to run; it has
		// no effect on the defaulted Width/Height shown elsewhere below.
		Ghost: ghostcaptcha.GhostOptions{Frames: 10},
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(len(text) == 6, len(data) > 0)
	// Output: true true
}

// CaptchaOptions.Length and Charset control the generated text itself.
func ExampleGenerateCaptcha_customCharset() {
	text, _, err := ghostcaptcha.GenerateCaptcha(ghostcaptcha.CaptchaOptions{
		Length:  4,
		Charset: "AB",
		Ghost:   ghostcaptcha.GhostOptions{Frames: 10},
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	onlyAB := true
	for _, r := range text {
		if r != 'A' && r != 'B' {
			onlyAB = false
		}
	}
	fmt.Println(len(text) == 4, onlyAB)
	// Output: true true
}

// GenerateGhost renders specific text rather than random captcha text, with
// custom drift and multi-line layout ("\n" breaks the text into lines
// stacked top to bottom; Width/Height are left unset here, so they default
// to fit however many lines and however long they are).
func ExampleGenerateGhost() {
	data, err := ghostcaptcha.GenerateGhost("HELLO\nWORLD", &ghostcaptcha.GhostOptions{
		TextDrift: ghostcaptcha.TextDriftEllipse,
		Frames:    10, // capped only to keep this example quick to run
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(len(data) > 0)
	// Output: true
}

// GhostOptions.Encoder swaps out GenerateGhost's default GIF/WebM/MP4
// encoding for a custom one, while still reusing its rendering (and, for
// video Formats, its ffmpeg-on-PATH preflight check).
func ExampleGenerateGhost_customEncoder() {
	var frameCount int
	opts := &ghostcaptcha.GhostOptions{
		Frames: 10, // capped only to keep this example quick to run
		Encoder: func(frames []*image.Paletted, delays []int, opts *ghostcaptcha.GhostOptions) ([]byte, error) {
			frameCount = len(frames)
			return []byte("my custom encoding"), nil
		},
	}
	data, err := ghostcaptcha.GenerateGhost("HELLO", opts)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(string(data), frameCount > 0)
	// Output: my custom encoding true
}

// GenerateGhostFrames renders the raw animation frames without encoding
// them into any container format, for callers who'd rather encode the
// animation themselves than use GenerateGhost's built-in GIF/WebM/MP4
// support.
func ExampleGenerateGhostFrames() {
	// Frames is capped only to keep this example quick to run.
	frames, delays, err := ghostcaptcha.GenerateGhostFrames("HELLO", &ghostcaptcha.GhostOptions{Frames: 10})
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(len(frames) == len(delays), len(frames) > 0)
	// Output: true true
}
