package feishu

import "testing"

func TestSign(t *testing.T) {
	sign, err := Sign("test-secret", 1599360473)
	if err != nil {
		t.Fatal(err)
	}
	if sign == "" {
		t.Fatal("expected non-empty sign")
	}
	if _, err := Sign("", 1599360473); err != nil {
		t.Fatal(err)
	}
}
