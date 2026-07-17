//go:build js && wasm

// Command wasm compiles to WebAssembly and exposes a single JS-callable
// function, generateGhostGIF, so a static page (see docs/index.html) can
// render ghost-captcha images entirely in the browser with no server. It's
// GIF-only: the video Formats shell out to ffmpeg via os/exec, which has no
// meaning inside WASM.
//
// .github/workflows/pages.yml builds this into docs/main.wasm and deploys
// docs/ to GitHub Pages on every push to main - docs/main.wasm is never
// committed, so it never bloats a module download of this package. To
// build and try it locally:
//
//	GOOS=js GOARCH=wasm go build -o docs/main.wasm ./wasm
package main

import (
	"encoding/base64"
	"image/color"
	"strconv"
	"syscall/js"

	ghostcaptcha "github.com/rohitkhatri1st/ghost-captcha"
)

func main() {
	js.Global().Set("generateGhostGIF", js.FuncOf(generateGhostGIF))
	select {}
}

// generateGhostGIF is exposed to JS as generateGhostGIF(text, options).
// options is a plain JS object; every field is optional and mirrors
// GhostOptions, using 0/"" to mean "unset" exactly like GhostOptions itself
// does. It never throws - on success it returns {ok: true, dataURL:
// "data:image/gif;base64,..."}; on failure {ok: false, error: "..."} - so
// callers only need to check .ok, not wrap the call in try/catch.
func generateGhostGIF(this js.Value, args []js.Value) any {
	if len(args) < 1 || args[0].Type() != js.TypeString {
		return result(false, "", "generateGhostGIF requires a text argument")
	}
	text := args[0].String()

	opts := ghostcaptcha.GhostOptions{Format: ghostcaptcha.FormatGIF}
	if len(args) > 1 && args[1].Type() == js.TypeObject {
		o := args[1]
		opts.FontSize = jsFloat(o, "fontSize")
		opts.LetterSpacing = int(jsFloat(o, "letterSpacing"))
		opts.Width = int(jsFloat(o, "width"))
		opts.Height = int(jsFloat(o, "height"))
		if c, ok := jsColor(o, "noiseColorA"); ok {
			opts.NoiseColorA = c
		}
		if c, ok := jsColor(o, "noiseColorB"); ok {
			opts.NoiseColorB = c
		}
		opts.BackgroundCellSize = int(jsFloat(o, "backgroundCellSize"))
		opts.TextCellSize = int(jsFloat(o, "textCellSize"))
		opts.Frames = int(jsFloat(o, "frames"))
		opts.FrameDelay = int(jsFloat(o, "frameDelay"))
		opts.Loop = int(jsFloat(o, "loop"))
		opts.TextDrift = ghostcaptcha.TextDrift(int(jsFloat(o, "textDrift")))
	}

	data, err := ghostcaptcha.GenerateGhost(text, &opts)
	if err != nil {
		return result(false, "", err.Error())
	}
	return result(true, "data:image/gif;base64,"+base64.StdEncoding.EncodeToString(data), "")
}

func result(ok bool, dataURL, errMsg string) js.Value {
	return js.ValueOf(map[string]any{
		"ok":      ok,
		"dataURL": dataURL,
		"error":   errMsg,
	})
}

// jsFloat reads a numeric field from a JS object, returning 0 (GhostOptions'
// own "unset" convention for every numeric field it has) if the field is
// absent, undefined, null, or not a number.
func jsFloat(o js.Value, field string) float64 {
	v := o.Get(field)
	if v.Type() != js.TypeNumber {
		return 0
	}
	return v.Float()
}

// jsColor reads a "#rrggbb" hex color field, returning ok=false if it's
// absent or not a valid hex color so the caller can fall back to
// GhostOptions' own default color instead of an unintended solid black.
func jsColor(o js.Value, field string) (color.Color, bool) {
	v := o.Get(field)
	if v.Type() != js.TypeString {
		return nil, false
	}
	s := v.String()
	if len(s) != 7 || s[0] != '#' {
		return nil, false
	}
	r, err1 := strconv.ParseUint(s[1:3], 16, 8)
	g, err2 := strconv.ParseUint(s[3:5], 16, 8)
	b, err3 := strconv.ParseUint(s[5:7], 16, 8)
	if err1 != nil || err2 != nil || err3 != nil {
		return nil, false
	}
	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}, true
}
