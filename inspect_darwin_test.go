//go:build darwin

package proctree

import "testing"

func TestDarwinStatus(t *testing.T) {
	cases := map[int8]string{
		darwinProcStateZombie: "zombie",
		darwinProcStateRun:    "running",
		darwinProcStateIdle:   "running",
		darwinProcStateSleep:  "sleeping",
		darwinProcStateStop:   "stopped",
		99:                    "unknown",
	}
	for stat, want := range cases {
		if got := darwinStatus(stat); got != want {
			t.Fatalf("stat=%d got %q want %q", stat, got, want)
		}
	}
}
