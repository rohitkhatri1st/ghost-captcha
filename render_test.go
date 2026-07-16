package ghostcaptcha

import (
	"testing"

	"golang.org/x/image/font/gofont/gomono"
)

func TestLoadFaceEmbeddedFont(t *testing.T) {
	face, err := loadFace(nil, 24)
	if err != nil {
		t.Fatalf("loadFace(nil, 24) error: %v", err)
	}
	defer face.Close()

	metrics := face.Metrics()
	if metrics.Ascent <= 0 {
		t.Errorf("Ascent = %v, want > 0", metrics.Ascent)
	}
	if metrics.Descent <= 0 {
		t.Errorf("Descent = %v, want > 0", metrics.Descent)
	}
}

func TestLoadFaceSizeAffectsMetrics(t *testing.T) {
	small, err := loadFace(nil, 12)
	if err != nil {
		t.Fatalf("loadFace(nil, 12) error: %v", err)
	}
	defer small.Close()

	large, err := loadFace(nil, 48)
	if err != nil {
		t.Fatalf("loadFace(nil, 48) error: %v", err)
	}
	defer large.Close()

	smallHeight := small.Metrics().Ascent + small.Metrics().Descent
	largeHeight := large.Metrics().Ascent + large.Metrics().Descent
	if largeHeight <= smallHeight {
		t.Errorf("larger point size should measure taller: size12 height=%v, size48 height=%v", smallHeight, largeHeight)
	}
}

func TestLoadFaceExplicitBytes(t *testing.T) {
	// Passing the embedded font's own bytes back in exercises the
	// non-nil fontBytes path (as opposed to the nil-defaults-to-embedded
	// path every other test uses).
	embeddedFace, err := loadFace(nil, 24)
	if err != nil {
		t.Fatalf("loadFace(nil, 24) error: %v", err)
	}
	embeddedFace.Close()

	face, err := loadFace(gomono.TTF, 24)
	if err != nil {
		t.Fatalf("loadFace(explicit bytes, 24) error: %v", err)
	}
	defer face.Close()

	if face.Metrics().Ascent <= 0 {
		t.Errorf("Ascent = %v, want > 0", face.Metrics().Ascent)
	}
}

func TestLoadFaceInvalidBytes(t *testing.T) {
	_, err := loadFace([]byte("not a font"), 24)
	if err == nil {
		t.Fatal("loadFace with garbage bytes: want error, got nil")
	}
}
