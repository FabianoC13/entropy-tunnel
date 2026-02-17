package camouflage

import (
	"testing"
)

func TestSelectFingerprint(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"chrome", "chrome", "HelloChrome_Auto", false},
		{"firefox", "firefox", "HelloFirefox_Auto", false},
		{"chrome-120", "chrome-120", "HelloChrome_120", false},
		{"safari", "safari", "HelloSafari_Auto", false},
		{"edge", "edge", "HelloEdge_Auto", false},
		{"ios", "ios", "HelloIOS_Auto", false},
		{"android", "android", "HelloAndroid_11_OkHttp", false},
		{"random", "random", "HelloRandomized", false},
		{"randomized", "randomized", "HelloRandomizedALPN", false},
		{"invalid", "netscape", "", true},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SelectFingerprint(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("SelectFingerprint(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("SelectFingerprint(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRandomFingerprint(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		fp := RandomFingerprint()
		if _, ok := SupportedFingerprints[fp]; !ok {
			t.Errorf("RandomFingerprint() returned unsupported fingerprint %q", fp)
		}
		seen[fp] = true
	}
	// Should see at least 2 different fingerprints over 100 trials
	if len(seen) < 2 {
		t.Errorf("RandomFingerprint() returned only %d unique fingerprints over 100 trials", len(seen))
	}
}

func TestPadPayload(t *testing.T) {
	tests := []struct {
		name       string
		dataLen    int
		targetSize int
		wantLen    int
	}{
		{"pad small", 10, 100, 100},
		{"already large", 200, 100, 200},
		{"exact", 100, 100, 100},
		{"empty", 0, 50, 50},
		{"zero target", 10, 0, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, tt.dataLen)
			result := PadPayload(data, tt.targetSize)
			if len(result) != tt.wantLen {
				t.Errorf("PadPayload(len=%d, target=%d) = len %d, want %d",
					tt.dataLen, tt.targetSize, len(result), tt.wantLen)
			}
		})
	}
}

func TestPadPayload_RandomContent(t *testing.T) {
	data := []byte{1, 2, 3}
	padded := PadPayload(data, 100)

	// Original data should be preserved
	for i := 0; i < 3; i++ {
		if padded[i] != data[i] {
			t.Errorf("original data corrupted at index %d", i)
		}
	}
}

func TestListFingerprints(t *testing.T) {
	fps := ListFingerprints()
	if len(fps) == 0 {
		t.Error("ListFingerprints() returned empty list")
	}
	if len(fps) != len(SupportedFingerprints) {
		t.Errorf("ListFingerprints() returned %d items, expected %d",
			len(fps), len(SupportedFingerprints))
	}
}

func TestNoiseInjector_Create(t *testing.T) {
	ni := NewNoiseInjector(100, 10, 50, nil)
	if ni == nil {
		t.Error("NewNoiseInjector returned nil")
	}
	if ni.interval != 100 {
		t.Errorf("expected interval 100, got %d", ni.interval)
	}
	if ni.minBytes != 10 {
		t.Errorf("expected minBytes 10, got %d", ni.minBytes)
	}
}
