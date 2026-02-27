package pure

import "testing"

func TestAdd(t *testing.T) {
	if got := Add(2, 3); got != 5 {
		t.Errorf("Add(2, 3) = %d, want 5", got)
	}
}

func TestMultiply(t *testing.T) {
	if got := Multiply(4, 5); got != 20 {
		t.Errorf("Multiply(4, 5) = %d, want 20", got)
	}
}

func TestAbs(t *testing.T) {
	if got := Abs(-7); got != 7 {
		t.Errorf("Abs(-7) = %d, want 7", got)
	}
	// Intentionally not testing Abs with positive input
	// to leave partial coverage for diff-cover validation
}

// Max and Clamp intentionally untested to produce partial coverage
