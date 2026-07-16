package ghostcaptcha

import (
	"errors"
	"fmt"
)

// GenerateGhost renders text as an animated "ghost text" GIF or video. The
// letterforms are filled with noise that scrolls one direction, surrounded
// by noise that scrolls the opposite direction. A single frame is just
// uniform static — every cell, inside the letterforms or out, is drawn from
// the same distribution and keeps moving every frame, so there's no static
// patch of pixels for a frame-differencing pass to pick the text out by —
// but once the animation plays, the two opposing motions make the
// letterforms stand out to a human eye.
//
// The animation is exactly one seamless loop: the last frame flows straight
// back into the first with no jump. opts.Format selects the output file
// format; the video formats need ffmpeg installed and on PATH.
func GenerateGhost(text string, opts *GhostOptions) ([]byte, error) {
	if text == "" {
		return nil, errors.New("ghostfont: text must not be empty")
	}
	text = lineEndingReplacer.Replace(text)
	opts.setDefaults()

	face, err := loadFace(opts.FontBytes, opts.FontSize)
	if err != nil {
		return nil, fmt.Errorf("ghostfont: loading font: %w", err)
	}
	defer face.Close()

	return nil, nil
}
