package provider

import (
	"testing"
)

func TestNormalizeIPAllowList(t *testing.T) {
	t.Parallel()

	got := normalizeIPAllowList([]string{"198.51.100.7", "198.51.100.7/32", "2001:db8::1", "  ", "10.0.0.0/8"})
	want := []string{"10.0.0.0/8", "198.51.100.7/32", "2001:db8::1/128"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
