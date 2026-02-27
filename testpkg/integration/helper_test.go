package integration

import "testing"

func TestConcat(t *testing.T) {
	if got := Concat("hello", " ", "world"); got != "hello world" {
		t.Errorf("Concat = %q, want %q", got, "hello world")
	}
}

func TestRepeat(t *testing.T) {
	if got := Repeat("ab", 3); got != "ababab" {
		t.Errorf("Repeat = %q, want %q", got, "ababab")
	}
}

// Truncate and CountChar intentionally untested to produce partial coverage
