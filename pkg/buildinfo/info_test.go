package buildinfo

import "testing"

func TestShort_DefaultValues(t *testing.T) {
	want := "dev (unknown)"
	got := Short()
	if got != want {
		t.Errorf("Short() = %q, want %q", got, want)
	}
}

func TestShort_CustomValues(t *testing.T) {
	t.Cleanup(func() {
		Version = "dev"
		BuildDate = "unknown"
	})

	Version = "1.2.3"
	BuildDate = "2026-01-15"

	want := "1.2.3 (2026-01-15)"
	got := Short()
	if got != want {
		t.Errorf("Short() = %q, want %q", got, want)
	}
}
