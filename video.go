package ghostcaptcha

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"os/exec"
	"runtime"
	"sync"
)

// rgb24LUT precomputes each palette entry's RGB24 bytes once, since every
// frame renderGhostFrames produces shares the same tiny noise palette —
// there's no need to call color.Color.RGBA (an interface call plus a
// 16-bit-to-8-bit shift) for every pixel of every frame when there are
// only noisePaletteSteps distinct colors total.
func rgb24LUT(palette color.Palette) [][3]byte {
	lut := make([][3]byte, len(palette))
	for i, c := range palette {
		r, g, b, _ := c.RGBA()
		lut[i] = [3]byte{byte(r >> 8), byte(g >> 8), byte(b >> 8)}
	}
	return lut
}

// framesToRawRGB24 flattens frames into one buffer of raw, interleaved
// RGB24 pixels (3 bytes/pixel, frame after frame, no padding or headers)
// — the exact layout ffmpeg's rawvideo demuxer expects, so encodeVideo can
// hand it over with no further conversion. Frames are independent, so the
// per-frame conversion is spread across a bounded worker pool.
func framesToRawRGB24(frames []*image.Paletted, width, height int) []byte {
	if len(frames) == 0 {
		return nil
	}
	lut := rgb24LUT(frames[0].Palette)
	frameSize := width * height * 3
	raw := make([]byte, len(frames)*frameSize)

	workers := min(runtime.GOMAXPROCS(0), len(frames))
	var wg sync.WaitGroup
	chunk := (len(frames) + workers - 1) / workers
	for w := 0; w < workers; w++ {
		start := w * chunk
		end := min(start+chunk, len(frames))
		if start >= end {
			continue
		}
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for i := start; i < end; i++ {
				dst := raw[i*frameSize : (i+1)*frameSize]
				for j, idx := range frames[i].Pix {
					rgb := lut[idx]
					dst[j*3], dst[j*3+1], dst[j*3+2] = rgb[0], rgb[1], rgb[2]
				}
			}
		}(start, end)
	}
	wg.Wait()
	return raw
}

// encodeVideo pipes frames to ffmpeg as raw RGB24 and returns the encoded
// container's bytes. fps is an ffmpeg "num/den" frame rate string (rather
// than a rounded float) so frame delays that don't divide evenly into a
// whole rate still play back at the exact right speed.
//
// Frames are pre-flattened into a single in-memory buffer and handed to
// ffmpeg via cmd.Stdin/cmd.Stdout (not manual StdinPipe/StdoutPipe) so the
// os/exec package's own copy goroutines feed and drain the process
// concurrently — that sidesteps the classic pipe deadlock (writing all of
// stdin before ever reading stdout, which blocks once ffmpeg's stdout
// buffer fills) without hand-rolled goroutine plumbing. Ghost-captcha
// frame counts are small enough (at most a few hundred frames of a few
// hundred pixels) that holding the whole raw stream in memory costs at
// most a few tens of MB.
func encodeVideo(frames []*image.Paletted, width, height int, fps string, format Format) ([]byte, error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, fmt.Errorf("%s output requires ffmpeg on PATH: %w", format, err)
	}

	raw := framesToRawRGB24(frames, width, height)

	args := []string{
		"-hide_banner", "-loglevel", "error",
		"-f", "rawvideo", "-pix_fmt", "rgb24",
		"-s", fmt.Sprintf("%dx%d", width, height),
		"-r", fps,
		"-i", "-",
		"-an",
		// Both codecs below encode 4:2:0 chroma-subsampled output, which
		// requires even width and height; the auto-fit canvas size in
		// GhostOptions.setCanvasDefaults has no such guarantee. Rounding
		// down to the nearest even number here (rather than constraining
		// every caller of Width/Height) keeps both containers safe to
		// re-encode downstream too — e.g. a WebM at an odd size encodes
		// fine under libvpx, but a viewer transcoding that WebM to H.264
		// for preview would hit the same "not divisible by 2" failure
		// MP4 output hits directly.
		"-vf", "scale=trunc(iw/2)*2:trunc(ih/2)*2",
	}
	switch format {
	case FormatMP4:
		// mp4's muxer normally seeks back to patch in the moov atom
		// once it knows the full stream length, but cmd.Stdout here is
		// a pipe, not a seekable file. frag_keyframe+empty_moov writes
		// a fragmented MP4 instead (moov up front, empty; media laid
		// out in self-contained fragments after it) — standard ISO
		// BMFF, playable by browsers/players same as a regular MP4,
		// and it doesn't need to seek to write.
		args = append(args,
			"-c:v", "libx264", "-preset", "ultrafast", "-crf", "23",
			"-pix_fmt", "yuv420p", "-f", "mp4", "-movflags", "frag_keyframe+empty_moov",
		)
	default: // FormatWebM
		args = append(args,
			"-c:v", "libvpx", "-deadline", "realtime", "-cpu-used", "8",
			"-crf", "30", "-b:v", "0", "-f", "webm",
		)
	}
	args = append(args, "-")

	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdin = bytes.NewReader(raw)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("running ffmpeg: %w: %s", err, stderr.String())
	}
	return out.Bytes(), nil
}
