package ghostcaptcha

import (
	"bytes"
	"image/gif"
	"strings"
	"testing"
)

// fastCaptchaGhost keeps GenerateCaptcha's rendering step cheap in tests:
// FormatGIF needs no ffmpeg, and a small canvas/explicit low Frames count
// renders almost instantly.
func fastCaptchaGhost() GhostOptions {
	return GhostOptions{
		FontSize: 16,
		Width:    80,
		Height:   30,
		Frames:   5,
		Format:   FormatGIF,
	}
}

func TestGenerateCaptchaDefaultLength(t *testing.T) {
	text, _, err := GenerateCaptcha(CaptchaOptions{Ghost: fastCaptchaGhost()})
	if err != nil {
		t.Fatalf("GenerateCaptcha: %v", err)
	}
	if len(text) != 6 {
		t.Errorf("len(text) = %d, want 6 (default Length)", len(text))
	}
}

func TestGenerateCaptchaCustomLength(t *testing.T) {
	text, _, err := GenerateCaptcha(CaptchaOptions{Length: 10, Ghost: fastCaptchaGhost()})
	if err != nil {
		t.Fatalf("GenerateCaptcha: %v", err)
	}
	if len(text) != 10 {
		t.Errorf("len(text) = %d, want 10", len(text))
	}
}

func TestGenerateCaptchaNonPositiveLengthDefaultsToSix(t *testing.T) {
	for _, length := range []int{0, -1, -100} {
		text, _, err := GenerateCaptcha(CaptchaOptions{Length: length, Ghost: fastCaptchaGhost()})
		if err != nil {
			t.Fatalf("GenerateCaptcha(Length=%d): %v", length, err)
		}
		if len(text) != 6 {
			t.Errorf("Length=%d: len(text) = %d, want 6", length, len(text))
		}
	}
}

func TestGenerateCaptchaDefaultCharset(t *testing.T) {
	for i := 0; i < 20; i++ {
		text, _, err := GenerateCaptcha(CaptchaOptions{Ghost: fastCaptchaGhost()})
		if err != nil {
			t.Fatalf("GenerateCaptcha: %v", err)
		}
		for _, r := range text {
			if !strings.ContainsRune(DefaultCaptchaCharset, r) {
				t.Errorf("text %q contains %q, not in DefaultCaptchaCharset", text, r)
			}
		}
	}
}

func TestGenerateCaptchaCustomCharset(t *testing.T) {
	const charset = "AB"
	text, _, err := GenerateCaptcha(CaptchaOptions{Length: 30, Charset: charset, Ghost: fastCaptchaGhost()})
	if err != nil {
		t.Fatalf("GenerateCaptcha: %v", err)
	}
	for _, r := range text {
		if r != 'A' && r != 'B' {
			t.Errorf("text %q contains %q, want only 'A' or 'B'", text, r)
		}
	}
}

func TestGenerateCaptchaSingleCharCharset(t *testing.T) {
	text, _, err := GenerateCaptcha(CaptchaOptions{Length: 5, Charset: "Z", Ghost: fastCaptchaGhost()})
	if err != nil {
		t.Fatalf("GenerateCaptcha: %v", err)
	}
	if text != "ZZZZZ" {
		t.Errorf("text = %q, want %q", text, "ZZZZZ")
	}
}

func TestGenerateCaptchaTextVariesAcrossCalls(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 20; i++ {
		text, _, err := GenerateCaptcha(CaptchaOptions{Ghost: fastCaptchaGhost()})
		if err != nil {
			t.Fatalf("GenerateCaptcha: %v", err)
		}
		seen[text] = true
	}
	if len(seen) < 2 {
		t.Errorf("20 calls produced only %d distinct text(s); expected randomized output", len(seen))
	}
}

func TestGenerateCaptchaGIFOutputDecodes(t *testing.T) {
	ghost := fastCaptchaGhost()
	_, data, err := GenerateCaptcha(CaptchaOptions{Ghost: ghost})
	if err != nil {
		t.Fatalf("GenerateCaptcha: %v", err)
	}
	decoded, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decoding captcha GIF: %v", err)
	}
	if len(decoded.Image) != ghost.Frames {
		t.Errorf("decoded frame count = %d, want %d", len(decoded.Image), ghost.Frames)
	}
}

func TestGenerateCaptchaPropagatesGhostError(t *testing.T) {
	ghost := fastCaptchaGhost()
	ghost.FontBytes = []byte("not a font")
	text, data, err := GenerateCaptcha(CaptchaOptions{Ghost: ghost})
	if err == nil {
		t.Fatal("GenerateCaptcha with invalid FontBytes: want error, got nil")
	}
	if text != "" || data != nil {
		t.Errorf("on error, want (\"\", nil), got (%q, %v)", text, data)
	}
}

func TestGenerateCaptchaDefaultFormatIsGIF(t *testing.T) {
	// Leaving Ghost entirely at its zero value exercises GenerateCaptcha's
	// true default path, including GenerateGhost's default Format
	// (FormatGIF) and full-size canvas - so this one is slower than the
	// fastCaptchaGhost-based tests above. It needs no ffmpeg, since GIF is
	// the default precisely so this always works with no dependencies.
	_, data, err := GenerateCaptcha(CaptchaOptions{})
	if err != nil {
		t.Fatalf("GenerateCaptcha with zero-value options: %v", err)
	}
	if _, err := gif.DecodeAll(bytes.NewReader(data)); err != nil {
		t.Errorf("default output does not decode as GIF: %v", err)
	}
}

func TestRandomText(t *testing.T) {
	t.Run("length matches n", func(t *testing.T) {
		text, err := randomText("ABC", 8)
		if err != nil {
			t.Fatalf("randomText: %v", err)
		}
		if len([]rune(text)) != 8 {
			t.Errorf("len(text) = %d, want 8", len([]rune(text)))
		}
	})

	t.Run("zero length returns empty string", func(t *testing.T) {
		text, err := randomText("ABC", 0)
		if err != nil {
			t.Fatalf("randomText: %v", err)
		}
		if text != "" {
			t.Errorf("text = %q, want empty", text)
		}
	})

	t.Run("single character charset is deterministic", func(t *testing.T) {
		text, err := randomText("X", 6)
		if err != nil {
			t.Fatalf("randomText: %v", err)
		}
		if text != "XXXXXX" {
			t.Errorf("text = %q, want %q", text, "XXXXXX")
		}
	})

	t.Run("every character comes from the charset", func(t *testing.T) {
		const charset = "AB"
		text, err := randomText(charset, 50)
		if err != nil {
			t.Fatalf("randomText: %v", err)
		}
		for _, r := range text {
			if r != 'A' && r != 'B' {
				t.Errorf("text %q contains %q, want only characters from %q", text, r, charset)
			}
		}
	})
}
