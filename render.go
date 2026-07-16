package ghostcaptcha

import (
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
)

// loadFace parses fontBytes (or the embedded Go Mono font, if nil) at the
// given point size. Go Mono is monospaced, so every glyph carries generous
// side bearing — that spacing keeps letterforms from blurring into each
// other once they're filled with noise instead of solid color.
func loadFace(fontBytes []byte, size float64) (font.Face, error) {
	if fontBytes == nil {
		fontBytes = gomono.TTF
	}
	f, err := opentype.Parse(fontBytes)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(f, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
}
