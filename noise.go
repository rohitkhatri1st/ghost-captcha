package ghostcaptcha

import (
	"image/color"
	"math/rand/v2"
)

// noisePaletteSteps is how many discrete colors interpolated between
// NoiseColorA and NoiseColorB make up a noise image's palette.
const noisePaletteSteps = 16

// buildNoisePalette returns a palette of noisePaletteSteps colors evenly
// interpolated between a and b, so noise cells can be indexed into a
// GIF-compatible image.Paletted image.
func buildNoisePalette(a, b color.Color, steps int) color.Palette {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()

	palette := make(color.Palette, steps)
	for i := range palette {
		t := float64(i) / float64(steps-1)
		palette[i] = color.RGBA{
			R: lerpChannel(ar, br, t),
			G: lerpChannel(ag, bg, t),
			B: lerpChannel(ab, bb, t),
			A: lerpChannel(aa, ba, t),
		}
	}
	return palette
}

// lerpChannel linearly interpolates between two color channel values as
// returned by color.Color.RGBA (16-bit, alpha-premultiplied) and narrows
// the result to the 8 bits color.RGBA expects.
func lerpChannel(a, b uint32, t float64) uint8 {
	v := float64(a) + (float64(b)-float64(a))*t
	// Dividing by 257 converts from 16-bit to 8-bit color channel values, since
	// 65535/257 = 255. The division is exact, so no rounding is needed.
	return uint8(v / 257)
}

// scrollDirection is which way a noise pattern moves across frames.
type scrollDirection int

const (
	scrollLeft scrollDirection = iota
	scrollRight
	scrollUp
	scrollDown
)

// noiseTile is a grid of random palette indices, cols wide and rows tall,
// one entry per noise cell. A frame is produced by sampling the tile at an
// offset that grows with the frame number, so the noise appears to scroll;
// because sampling wraps around the tile's edges, the scroll loops
// seamlessly after exactly cols (horizontal scroll) or rows (vertical
// scroll) frames.
type noiseTile struct {
	cols, rows int
	cells      []uint8
}

func newNoiseTile(cols, rows, paletteSize int) *noiseTile {
	cells := make([]uint8, cols*rows)
	for i := range cells {
		cells[i] = uint8(rand.IntN(paletteSize))
	}
	return &noiseTile{cols: cols, rows: rows, cells: cells}
}

func (t *noiseTile) at(col, row int) uint8 {
	col = ((col % t.cols) + t.cols) % t.cols
	row = ((row % t.rows) + t.rows) % t.rows
	return t.cells[row*t.cols+col]
}

// cellGrid returns how many cellSize x cellSize cells are needed to cover
// a width x height canvas. Adding cellSize - 1 before dividing pushes any
// remainder over the next integer boundary, so it rounds up instead of
// down: width = 203, cellSize = 8 → plain division gives 203/8 = 25
// (truncated, drops 3px), but (203+7)/8 = 26, so the grid has one more
// (partial) cell that covers those last 3 pixels instead of clipping them.
func cellGrid(width, height, cellSize int) (cols, rows int) {
	cols = (width + cellSize - 1) / cellSize
	rows = (height + cellSize - 1) / cellSize
	return cols, rows
}

// sampleTile looks up the palette index tile assigns to canvas pixel
// (x, y) at cellSize granularity, offset by shift cells in dir. Calling
// this with shift = 0, 1, 2, ... for successive frames makes the sampled
// noise scroll in dir.
func sampleTile(tile *noiseTile, x, y, cellSize, shift int, dir scrollDirection) uint8 {
	col, row := x/cellSize, y/cellSize
	switch dir {
	case scrollLeft:
		col += shift
	case scrollRight:
		col -= shift
	case scrollUp:
		row += shift
	case scrollDown:
		row -= shift
	}
	return tile.at(col, row)
}

// opposite returns the scroll direction facing the other way along the
// same axis, so a letterform's noise fill can travel against the
// background noise surrounding it.
func opposite(dir scrollDirection) scrollDirection {
	switch dir {
	case scrollLeft:
		return scrollRight
	case scrollRight:
		return scrollLeft
	case scrollUp:
		return scrollDown
	default:
		return scrollUp
	}
}

// backgroundFrameCount returns how many frames the background noise needs
// to scroll through in dir before it loops seamlessly: opts.Frames if set
// explicitly, otherwise the cell count along the axis of travel (cols for
// a horizontal scroll, rows for a vertical one), since shifting one cell
// per frame returns to the starting tile offset after exactly that many
// frames. Other per-frame effects (e.g. the text drift in text.go) take
// this same count so every looping effect closes at once.
func backgroundFrameCount(opts *GhostOptions, dir scrollDirection) int {
	if opts.Frames > 0 {
		return opts.Frames
	}
	cols, rows := cellGrid(opts.Width, opts.Height, opts.BackgroundCellSize)
	if dir == scrollLeft || dir == scrollRight {
		return cols
	}
	return rows
}
