package hybrid

import (
	"testing"
)

func TestIsScreenLocked(t *testing.T) {
	detector, err := NewDetector()
	if err != nil {
		t.Fatalf("Failed to create detector: %v", err)
	}

	locked := detector.isScreenLocked()
	t.Logf("Screen is locked: %v", locked)
}
