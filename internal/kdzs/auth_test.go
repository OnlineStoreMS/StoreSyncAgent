package kdzs

import "testing"

func TestHashPassword(t *testing.T) {
	got := hashPassword("Yyz@201314.")
	want := "4CB24789FD657CCB63CF3C52A94C3F88"
	if got != want {
		t.Fatalf("hashPassword() = %q, want %q", got, want)
	}
}
