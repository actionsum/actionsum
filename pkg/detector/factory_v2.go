package detector

import (
	"actionsum/pkg/integrationsV2/hybrid"
	"actionsum/pkg/window"
)

// NewV2 creates a new hybrid detector that works universally
// This is the V2 detector that combines window detection with process monitoring
func NewV2() (window.Detector, error) {
	return hybrid.NewDetector()
}
