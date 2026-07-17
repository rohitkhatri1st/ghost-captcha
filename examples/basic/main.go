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
	"fmt"
	"os"

	ghostcaptcha "github.com/rohitkhatri1st/ghost-captcha"
)

func main() {
	data, err := ghostcaptcha.GenerateGhost("CAPTCHA", &ghostcaptcha.GhostOptions{
		LetterSpacing: 7,
		// Format:        ghostcaptcha.FormatGIF,
		FontSize: 100,
	})
	if err != nil {
		panic(err)
	}
	// fmt.Println("Captcha Text:", text)
	fmt.Println("Captcha Data Length:", len(data))
	if err := os.WriteFile("output_captcha.gif", data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "error writing file:", err)
		os.Exit(1)
	}
	// fmt.Println("captcha text:", text)
	fmt.Println("wrote output_captcha.gif")
}
