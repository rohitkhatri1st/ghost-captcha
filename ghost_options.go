package ghostcaptcha

import (
	"image/color"
	"math"
)

// defaultFontSize is FontSize's zero-value default.
const defaultFontSize = 70

// canvasMarginXFactor and canvasMarginYFactor size the default canvas's
// margin beyond the rendered text's own footprint, as a multiple of
// FontSize: enough room for TextDrift's wandering (see textFrameAnchors)
// without the letterforms bumping the canvas edge, regardless of how long
// the text is or how many lines it spans. Width gets more margin than
// Height because TextDrift's default (TextDriftRandom) can pick a purely
// horizontal path, which needs full side-to-side room to read as motion
// rather than a twitch.
const (
	canvasMarginXFactor = 2.0
	canvasMarginYFactor = 0.6
)

// GhostOptions controls GenerateGhost's noise animation: the text is drawn
// entirely out of noise cells that scroll one direction, surrounded by
// noise that scrolls the opposite direction. A single frame is
// indistinguishable static — every cell, text or background, is drawn from
// the same distribution and keeps moving — but once the animation plays,
// the two opposing motions make the letterforms stand out.
type GhostOptions struct {
	// FontSize is the point size of the font. Default: 70.
	FontSize float64
	// FontBytes holds raw TTF/OTF font data. Default: the embedded Go Mono font.
	FontBytes []byte

	// LetterSpacing adds this many extra pixels of horizontal gap between
	// each pair of adjacent characters, on top of the font's natural
	// advance width. Negative values pull characters closer together (or
	// overlapping, if negative enough). Default: 0, just the font's
	// natural spacing.
	LetterSpacing int

	// Width and Height are the pixel dimensions of the output file. Left
	// unset, both default to fit the rendered text itself, plus a margin
	// (proportional to FontSize) for TextDrift's wandering room: longer
	// text (or a longer line within multi-line text) defaults to a wider
	// canvas, and text with more "\n"-separated lines defaults to a
	// taller one. Set a field explicitly to take full control of it — an
	// explicit Width or Height never grows or shrinks to fit the text.
	Width  int
	Height int

	// NoiseColorA and NoiseColorB are the two ends of the noise's color
	// range. Every noise cell is an independent random point between them.
	// Defaults: black and white.
	NoiseColorA color.Color
	NoiseColorB color.Color

	// BackgroundCellSize is the pixel size of each square noise grain in
	// the scrolling background. Smaller cell size mean more details and better
	// clarity on the cost of increased size. Default: 1.
	BackgroundCellSize int
	// TextCellSize is the pixel size of each square noise grain used to
	// fill the letterforms. It only affects the noise, not the
	// letterforms' shape, which is always drawn at full pixel resolution.
	// For best output, BackgroundCellSize must be equal to TextCellSize
	// Default: 1.
	TextCellSize int

	// Frames is how many frames the animation contains. Default: just
	// enough for the scrolling to complete exactly one loop, so playback
	// has no visible jump when it restarts.
	Frames int
	// FrameDelay is the delay between animation frames, in centiseconds
	// (1/100th of a second). Default: 4 (40ms).
	FrameDelay int

	// Loop is the GIF loop count. 0 means loop forever. Only applies to
	// FormatGIF. Default: 0.
	Loop int

	// TextDrift selects how the letterforms wander from frame to frame.
	// Default: TextDriftRandom.
	TextDrift TextDrift

	// Format selects the output container/codec. Default: FormatGIF.
	Format Format

	// Encoder overrides how GenerateGhost turns rendered frames into output
	// bytes. Left nil, GenerateGhost picks its own GIF/WebM/MP4 encoder
	// based on Format. Set it to encode the frames yourself instead — with
	// different settings than GenerateGhost's own encoders, or into a
	// format it doesn't support at all. Call GenerateGhostFrames directly
	// if you'd rather skip GenerateGhost and its Format/ffmpeg handling
	// entirely.
	Encoder FrameEncoder
}

// Format selects GenerateGhost's output container/codec.
type Format int

const (
	// FormatGIF encodes a paletted, looping animated GIF using only the
	// Go standard library. This is the default: it has no dependency on
	// ffmpeg being installed (or even runnable at all, e.g. compiled to
	// WebAssembly), at the cost of a much larger file for the same
	// animation than either video format.
	FormatGIF Format = iota
	// FormatWebM encodes VP8 video in a WebM container using ffmpeg's
	// libvpx encoder at its realtime/fastest settings, for callers who'd
	// rather have a smaller/faster file than a dependency-free one.
	// Requires ffmpeg on PATH.
	FormatWebM
	// FormatMP4 encodes H.264 video (libx264, "ultrafast" preset) in an
	// MP4 container, for players/browsers that don't support WebM.
	// Requires ffmpeg on PATH.
	FormatMP4
)

// String returns the format's lowercase name, e.g. for error messages.
func (f Format) String() string {
	switch f {
	case FormatWebM:
		return "webm"
	case FormatMP4:
		return "mp4"
	default:
		return "gif"
	}
}

// TextDrift selects the shape of the path the letterforms wander along
// across frames. Whichever mode is chosen, the path is mathematically
// guaranteed to return to its exact starting point after the animation's
// frame count, so the loop never jumps regardless of drift shape.
type TextDrift int

const (
	// TextDriftRandom draws a fresh drift shape — fixed, ellipse,
	// horizontal, or vertical, plus more tangled figure-eight/rosette
	// shapes the other modes can't produce — with randomized amplitude,
	// phase, and harmonics, every call. This is the default: varying the
	// shape call to call is what keeps the motion from being a fixed,
	// learnable signature.
	TextDriftRandom TextDrift = iota
	// TextDriftFixed locks the letterforms to the canvas center; they
	// don't drift at all, only the noise scrolling through and around
	// them moves.
	TextDriftFixed
	// TextDriftEllipse orbits the letterforms around the center in a
	// randomized ellipse (or circle).
	TextDriftEllipse
	// TextDriftHorizontal wobbles the letterforms left and right through
	// the center, with no vertical movement.
	TextDriftHorizontal
	// TextDriftVertical wobbles the letterforms up and down through the
	// center, with no horizontal movement.
	TextDriftVertical
)

func (o *GhostOptions) setDefaults() {
	if o.FontSize <= 0 {
		o.FontSize = defaultFontSize
	}
	if o.NoiseColorA == nil {
		o.NoiseColorA = color.Black
	}
	if o.NoiseColorB == nil {
		o.NoiseColorB = color.White
	}
	if o.BackgroundCellSize <= 0 {
		o.BackgroundCellSize = 1
	}
	if o.TextCellSize <= 0 {
		o.TextCellSize = 1
	}
	if o.FrameDelay <= 0 {
		o.FrameDelay = 4
	}
}

// setCanvasDefaults defaults Width and Height, if unset, to fit the
// rendered text shape (shapeWidth x shapeHeight, from textShape) plus a
// FontSize-proportional margin for TextDrift's wandering room. It runs
// separately from setDefaults, and later, because — unlike every other
// field — the right default genuinely depends on the text being rendered:
// callers only know shapeWidth/shapeHeight once they've loaded a font and
// measured the (already line-split) text against it.
func (o *GhostOptions) setCanvasDefaults(shapeWidth, shapeHeight int) {
	if o.Width <= 0 {
		o.Width = shapeWidth + int(math.Round(o.FontSize*canvasMarginXFactor))
	}
	if o.Height <= 0 {
		o.Height = shapeHeight + int(math.Round(o.FontSize*canvasMarginYFactor))
	}
}
