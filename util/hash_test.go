package util

import "testing"

func TestReserveHash(t *testing.T) {
	hr, _ := ReserveHash("53dae1de0cf115cbaec8d48893b891231f3907ff5878ede3cde6959000f52076")
	if hr != "00f52076cde695905878ede31f3907ff93b89123aec8d4880cf115cb53dae1de" {
		t.Errorf("expected %s, got %s", "00f52076cde695905878ede31f3907ff93b89123aec8d4880cf115cb53dae1de", hr)
	}
	_, err := ReserveHash("123")
	if err == nil {
		t.Errorf("expected %s, got %s", "hash lenght must be 64", err)
	}
}
