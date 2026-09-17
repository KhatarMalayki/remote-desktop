package agent

import (
	"bytes"
	"image"
	"image/jpeg"
	"testing"
)

func TestValidRemoteSessionID(t *testing.T) {
	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{name: "uuid style", value: "sess-4e64c240-20ac-45ad-8058-49933a377534", valid: true},
		{name: "underscore", value: "session_test_1", valid: true},
		{name: "empty", value: "", valid: false},
		{name: "slash", value: "session/agent", valid: false},
		{name: "query injection", value: "session?key=stolen", valid: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := validRemoteSessionID(test.value); got != test.valid {
				t.Fatalf("validRemoteSessionID(%q) = %v, want %v", test.value, got, test.valid)
			}
		})
	}
}

func TestEncodeRemoteFrameProducesDecodableJPEG(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 320, 180))
	encoded, err := encodeRemoteFrame(img)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := jpeg.Decode(bytes.NewReader(encoded))
	if err != nil {
		t.Fatalf("encoded frame is not a JPEG: %v", err)
	}
	if decoded.Bounds().Dx() != 320 || decoded.Bounds().Dy() != 180 {
		t.Fatalf("decoded dimensions = %v, want 320x180", decoded.Bounds())
	}
}

func TestEncodeRemoteFrameProfileDownscalesLargeDesktop(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 3840, 2160))
	encoded, err := encodeRemoteFrameProfile(img, 42, 1280)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := jpeg.Decode(bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Bounds().Dx() != 1280 || decoded.Bounds().Dy() != 720 {
		t.Fatalf("scaled dimensions = %v, want 1280x720", decoded.Bounds())
	}
}
