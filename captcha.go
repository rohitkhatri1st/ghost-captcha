package ghostcaptcha

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// DefaultCaptchaCharset is the character pool CaptchaOptions.Charset draws
// from when left unset: uppercase letters and digits, minus 0/O/1/I/L,
// which are easy to mistake for each other once rendered as noise.
const DefaultCaptchaCharset = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// CaptchaOptions controls GenerateCaptcha.
type CaptchaOptions struct {
	// Length is how many random characters the captcha text contains.
	// Default: 6.
	Length int

	// Charset is the pool of characters the captcha text is drawn from.
	// Default: DefaultCaptchaCharset.
	Charset string

	// Ghost carries every GenerateGhost visual/animation option — font,
	// size, colors, speed, and so on. Any field left at its zero value
	// gets a captcha-appropriate default instead of GenerateGhost's
	// generic one (e.g. a larger FontSize, and Width/Height sized to fit
	// the generated text itself rather than arbitrary text). Set a field
	// explicitly to take full control of it.
	Ghost GhostOptions
}

func GenerateCaptcha(opts CaptchaOptions) (text string, data []byte, err error) {
	if opts.Length <= 0 {
		opts.Length = 6
	}
	charset := opts.Charset
	if charset == "" {
		charset = DefaultCaptchaCharset
	}

	text, err = randomText(charset, opts.Length)
	if err != nil {
		return "", nil, fmt.Errorf("ghostfont: generating captcha text: %w", err)
	}
	ghostOpts := getCaptchaGhostOptsDefaults(&opts)

	data, err = GenerateGhost(text, ghostOpts)
	if err != nil {
		return "", nil, err
	}
	return text, data, nil
}

// TODO: Fill this function correctly
func getCaptchaGhostOptsDefaults(opts *CaptchaOptions) *GhostOptions {
	return &opts.Ghost
}

// randomText draws n characters uniformly at random from charset using a
// cryptographically secure source, since a captcha's only job is to be
// hard to predict.
func randomText(charset string, n int) (string, error) {
	runes := []rune(charset)
	out := make([]rune, n)
	for i := range out {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(runes))))
		if err != nil {
			return "", err
		}
		out[i] = runes[idx.Int64()]
	}
	return string(out), nil
}
