package utils

import (
	"testing"
)

func TestRequired(t *testing.T) {
	Required(1, "number")
}

func TestRequiredPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic, got %v", r)
		}
	}()
	Required("", "")
}
