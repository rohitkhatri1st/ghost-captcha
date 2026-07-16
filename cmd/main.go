package main

import (
	"fmt"
	"os"

	ghostcaptcha "github.com/rohitkhatri1st/ghost-captcha"
)

func main() {
	text, data, err := ghostcaptcha.GenerateCaptcha(ghostcaptcha.CaptchaOptions{
		Ghost: ghostcaptcha.GhostOptions{},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("Captcha Text:", text)
	fmt.Println("Captcha Data Length:", len(data))
	if err := os.WriteFile("output_captcha.webm", data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "error writing file:", err)
		os.Exit(1)
	}
	fmt.Println("captcha text:", text)
	fmt.Println("wrote output_captcha.webm")
}
