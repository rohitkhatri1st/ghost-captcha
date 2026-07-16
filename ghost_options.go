package ghostcaptcha

// GhostOptions controls GenerateGhost's noise animation: the text is drawn
// entirely out of noise cells that scroll one direction, surrounded by
// noise that scrolls the opposite direction. A single frame is
// indistinguishable static — every cell, text or background, is drawn from
// the same distribution and keeps moving — but once the animation plays,
// the two opposing motions make the letterforms stand out.
type GhostOptions struct {
	// FontSize is the point size of the font. Default: 24.
	FontSize float64
	// FontBytes holds raw TTF/OTF font data. Default: the embedded Go Mono font.
	FontBytes []byte
}

func (o *GhostOptions) setDefaults() {
	if o.FontSize <= 0 {
		o.FontSize = 24
	}
}
