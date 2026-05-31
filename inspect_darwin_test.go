//go:build darwin

package proctree

import "testing"

func TestDarwinStatus(t *testing.T) {
	t.Parallel()
	cases := []struct {
		stat int8
		want string
	}{
		{stat: darwinProcStateZombie, want: "zombie"},
		{stat: darwinProcStateRun, want: "running"},
		{stat: darwinProcStateIdle, want: "running"},
		{stat: darwinProcStateSleep, want: "sleeping"},
		{stat: darwinProcStateStop, want: "stopped"},
		{stat: 99, want: "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()
			if got := darwinStatus(tc.stat); got != tc.want {
				t.Fatalf("stat=%d got %q want %q", tc.stat, got, tc.want)
			}
		})
	}
}
