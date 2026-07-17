// Command basic is a minimal, runnable demonstration of ghost-captcha:
// generate a random captcha with every option left at its default and write
// the result to disk. Run it with:
//
//	go run ./examples/basic
//
// GhostOptions.Format defaults to FormatGIF, which needs no ffmpeg; set it
// to FormatWebM or FormatMP4 instead for video output (needs ffmpeg on PATH).
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"os"

	ghostcaptcha "github.com/rohitkhatri1st/ghost-captcha"
)

// encodeFirstFrameJPEG is a GhostOptions.Encoder that ignores every frame
// but the first and encodes it as a static JPEG, for callers who want a
// single still image (e.g. an og:image preview) instead of an animation.
func encodeFirstFrameJPEG(frames []*image.Paletted, delays []int, opts *ghostcaptcha.GhostOptions) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, frames[0], nil); err != nil {
		return nil, fmt.Errorf("encoding jpeg: %w", err)
	}
	return buf.Bytes(), nil
}

func main() {
	data, err := ghostcaptcha.GenerateGhost("CAPTCHA", &ghostcaptcha.GhostOptions{
		FontSize:      100,
		LetterSpacing: 7,
		// Format:        ghostcaptcha.FormatGIF,
		// Encoder:       encodeFirstFrameJPEG,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("Captcha Data Length:", len(data))
	if err := os.WriteFile("output_ghost.gif", data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "error writing file:", err)
		os.Exit(1)
	}
	fmt.Println("wrote output_ghost.gif")

	text, data, err := ghostcaptcha.GenerateCaptcha(ghostcaptcha.CaptchaOptions{
		Length:  6,
		Charset: ghostcaptcha.DefaultCaptchaCharset,
		Ghost: ghostcaptcha.GhostOptions{
			FontSize:      100,
			LetterSpacing: 7,
		},
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("Captcha Text:", text)
	fmt.Println("Captcha Data Length:", len(data))
	if err := os.WriteFile("output_captcha.gif", data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "error writing file:", err)
		os.Exit(1)
	}
	fmt.Println("wrote output_captcha.gif")
}
