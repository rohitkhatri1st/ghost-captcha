package ghostcaptcha

import (
	"image/color"
	"testing"
)

func TestBuildNoisePalette(t *testing.T) {
	steps := 16
	palette := buildNoisePalette(color.Black, color.White, steps)

	if len(palette) != steps {
		t.Fatalf("len(palette) = %d, want %d", len(palette), steps)
	}

	first := palette[0].(color.RGBA)
	if first.R != 0 || first.G != 0 || first.B != 0 {
		t.Errorf("first palette entry = %+v, want black", first)
	}

	last := palette[steps-1].(color.RGBA)
	if last.R != 255 || last.G != 255 || last.B != 255 {
		t.Errorf("last palette entry = %+v, want white", last)
	}

	// Interpolation should be monotonically non-decreasing from a to b.
	for i := 1; i < steps; i++ {
		prev := palette[i-1].(color.RGBA)
		cur := palette[i].(color.RGBA)
		if cur.R < prev.R {
			t.Errorf("palette not monotonic at step %d: %d < %d", i, cur.R, prev.R)
		}
	}
}

func TestBuildNoisePaletteSingleStep(t *testing.T) {
	// steps=1 makes t = i/(steps-1) a 0/0 float division; that's NaN, not
	// a panic, so this only pins down that the function still returns a
	// palette of the requested length instead of crashing.
	palette := buildNoisePalette(color.Black, color.White, 1)
	if len(palette) != 1 {
		t.Errorf("len(palette) = %d, want 1", len(palette))
	}
}

func TestLerpChannel(t *testing.T) {
	tests := []struct {
		name string
		a, b uint32
		t    float64
		want uint8
	}{
		{"t=0 returns a", 0, 65535, 0, 0},
		{"t=1 returns b", 0, 65535, 1, 255},
		{"t=0.5 midpoint", 0, 65535, 0.5, 127},
		{"a equals b", 30000, 30000, 0.5, uint8(30000 / 257)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lerpChannel(tt.a, tt.b, tt.t); got != tt.want {
				t.Errorf("lerpChannel(%d, %d, %v) = %d, want %d", tt.a, tt.b, tt.t, got, tt.want)
			}
		})
	}
}

func TestNoiseTileDimensionsAndRange(t *testing.T) {
	cols, rows, paletteSize := 5, 7, 16
	tile := newNoiseTile(cols, rows, paletteSize)

	if len(tile.cells) != cols*rows {
		t.Fatalf("len(cells) = %d, want %d", len(tile.cells), cols*rows)
	}
	for _, c := range tile.cells {
		if int(c) >= paletteSize {
			t.Errorf("cell value %d out of range [0, %d)", c, paletteSize)
		}
	}
}

func TestNoiseTileAtWraps(t *testing.T) {
	tile := newNoiseTile(4, 3, 16)

	// In-bounds lookups should match the raw backing slice exactly.
	for row := 0; row < tile.rows; row++ {
		for col := 0; col < tile.cols; col++ {
			want := tile.cells[row*tile.cols+col]
			if got := tile.at(col, row); got != want {
				t.Errorf("at(%d, %d) = %d, want %d", col, row, got, want)
			}
		}
	}

	// Wrapping forward by exactly one tile width/height must reproduce
	// the same value, and negative indices must wrap the same way a
	// mathematical modulo would (not Go's remainder, which can be negative).
	for row := 0; row < tile.rows; row++ {
		for col := 0; col < tile.cols; col++ {
			want := tile.at(col, row)
			if got := tile.at(col+tile.cols, row); got != want {
				t.Errorf("at(%d, %d) [wrapped +cols] = %d, want %d", col+tile.cols, row, got, want)
			}
			if got := tile.at(col-tile.cols, row); got != want {
				t.Errorf("at(%d, %d) [wrapped -cols] = %d, want %d", col-tile.cols, row, got, want)
			}
			if got := tile.at(col, row+tile.rows); got != want {
				t.Errorf("at(%d, %d) [wrapped +rows] = %d, want %d", col, row+tile.rows, got, want)
			}
			if got := tile.at(col, row-tile.rows); got != want {
				t.Errorf("at(%d, %d) [wrapped -rows] = %d, want %d", col, row-tile.rows, got, want)
			}
		}
	}
}

func TestCellGrid(t *testing.T) {
	tests := []struct {
		name                string
		width, height, size int
		wantCols, wantRows  int
	}{
		{"exact division", 200, 80, 8, 25, 10},
		{"rounds up remainder", 203, 83, 8, 26, 11},
		{"cell size 1 matches pixels", 400, 100, 1, 400, 100},
		{"cell larger than canvas", 5, 5, 8, 1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cols, rows := cellGrid(tt.width, tt.height, tt.size)
			if cols != tt.wantCols || rows != tt.wantRows {
				t.Errorf("cellGrid(%d, %d, %d) = (%d, %d), want (%d, %d)",
					tt.width, tt.height, tt.size, cols, rows, tt.wantCols, tt.wantRows)
			}
		})
	}
}

func TestSampleTileScrolls(t *testing.T) {
	tile := newNoiseTile(10, 10, 16)
	cellSize := 2
	x, y := 4, 6 // maps to col=2, row=3

	base := sampleTile(tile, x, y, cellSize, 0, scrollLeft)
	if base != tile.at(2, 3) {
		t.Fatalf("shift=0 sample = %d, want tile.at(2,3) = %d", base, tile.at(2, 3))
	}

	tests := []struct {
		dir  scrollDirection
		want uint8
	}{
		{scrollLeft, tile.at(2+3, 3)},
		{scrollRight, tile.at(2-3, 3)},
		{scrollUp, tile.at(2, 3+3)},
		{scrollDown, tile.at(2, 3-3)},
	}
	for _, tt := range tests {
		if got := sampleTile(tile, x, y, cellSize, 3, tt.dir); got != tt.want {
			t.Errorf("sampleTile dir=%v shift=3 = %d, want %d", tt.dir, got, tt.want)
		}
	}
}

func TestOpposite(t *testing.T) {
	tests := []struct {
		dir  scrollDirection
		want scrollDirection
	}{
		{scrollLeft, scrollRight},
		{scrollRight, scrollLeft},
		{scrollUp, scrollDown},
		{scrollDown, scrollUp},
	}
	for _, tt := range tests {
		if got := opposite(tt.dir); got != tt.want {
			t.Errorf("opposite(%v) = %v, want %v", tt.dir, got, tt.want)
		}
		// opposite must be its own inverse.
		if got := opposite(opposite(tt.dir)); got != tt.dir {
			t.Errorf("opposite(opposite(%v)) = %v, want %v", tt.dir, got, tt.dir)
		}
	}
}

func TestBackgroundFrameCount(t *testing.T) {
	t.Run("explicit Frames wins", func(t *testing.T) {
		opts := &GhostOptions{Width: 400, Height: 100, BackgroundCellSize: 1, Frames: 42}
		if got := backgroundFrameCount(opts, scrollDown); got != 42 {
			t.Errorf("backgroundFrameCount = %d, want 42", got)
		}
		if got := backgroundFrameCount(opts, scrollLeft); got != 42 {
			t.Errorf("backgroundFrameCount = %d, want 42", got)
		}
	})

	t.Run("horizontal scroll uses column count", func(t *testing.T) {
		opts := &GhostOptions{Width: 400, Height: 100, BackgroundCellSize: 8}
		cols, _ := cellGrid(opts.Width, opts.Height, opts.BackgroundCellSize)
		for _, dir := range []scrollDirection{scrollLeft, scrollRight} {
			if got := backgroundFrameCount(opts, dir); got != cols {
				t.Errorf("backgroundFrameCount(dir=%v) = %d, want %d (cols)", dir, got, cols)
			}
		}
	})

	t.Run("vertical scroll uses row count", func(t *testing.T) {
		opts := &GhostOptions{Width: 400, Height: 100, BackgroundCellSize: 8}
		_, rows := cellGrid(opts.Width, opts.Height, opts.BackgroundCellSize)
		for _, dir := range []scrollDirection{scrollUp, scrollDown} {
			if got := backgroundFrameCount(opts, dir); got != rows {
				t.Errorf("backgroundFrameCount(dir=%v) = %d, want %d (rows)", dir, got, rows)
			}
		}
	})
}
