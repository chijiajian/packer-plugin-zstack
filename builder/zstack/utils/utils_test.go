// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package utils

import "testing"

func TestMBToBytes(t *testing.T) {
	cases := []struct {
		in   int64
		want int64
	}{
		{0, 0},
		{1, 1024 * 1024},
		{2048, 2048 * 1024 * 1024},
	}
	for _, c := range cases {
		if got := MBToBytes(c.in); got != c.want {
			t.Errorf("MBToBytes(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestBytesToMB(t *testing.T) {
	if got := BytesToMB(1024 * 1024); got != 1 {
		t.Errorf("BytesToMB(1MiB) = %d, want 1", got)
	}
	if got := BytesToMB(3*1024*1024 + 1); got != 3 {
		t.Errorf("BytesToMB truncation expected")
	}
	if got := BytesToMB(0); got != 0 {
		t.Errorf("BytesToMB(0) = %d, want 0", got)
	}
}

func TestGBToBytes(t *testing.T) {
	if got := GBToBytes(2); got != 2*1024*1024*1024 {
		t.Errorf("GBToBytes(2) wrong: %d", got)
	}
}

func TestBytesToGB(t *testing.T) {
	if got := BytesToGB(5 * 1024 * 1024 * 1024); got != 5 {
		t.Errorf("BytesToGB(5GiB) = %d, want 5", got)
	}
}

func TestRoundTripMB(t *testing.T) {
	for _, mb := range []int64{1, 512, 4096, 16384} {
		if got := BytesToMB(MBToBytes(mb)); got != mb {
			t.Errorf("round-trip MB failed: %d → %d", mb, got)
		}
	}
}
