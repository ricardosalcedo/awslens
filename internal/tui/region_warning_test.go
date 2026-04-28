package tui

import "testing"

func TestFormatRegionWarning(t *testing.T) {
	tests := []struct {
		name   string
		failed []string
		want   string
	}{
		{"nil", nil, ""},
		{"empty", []string{}, ""},
		{"one", []string{"us-east-1"}, "⚠ us-east-1 failed"},
		{"two", []string{"eu-west-1", "us-east-1"}, "⚠ eu-west-1, us-east-1 failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatRegionWarning(tt.failed)
			if got != tt.want {
				t.Errorf("formatRegionWarning(%v) = %q, want %q", tt.failed, got, tt.want)
			}
		})
	}
}
