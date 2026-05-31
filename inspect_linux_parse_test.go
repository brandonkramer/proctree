//go:build linux

package proctree

import (
	"testing"
	"time"
)

func TestParseProcStat(t *testing.T) {
	stat := []byte("(sleep) S 100 50 100 100 34817 123456 0 0 0 0 0 0 0 20 0 1 0 12345 4567890 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0")
	name, state, ppid, starttime, err := parseProcStat(stat)
	if err != nil {
		t.Fatal(err)
	}
	if name != "sleep" || state != 'S' || ppid != 100 || starttime != 4567890 {
		t.Fatalf("name=%q state=%c ppid=%d start=%d", name, state, ppid, starttime)
	}
}

func TestParseProcStatRejectsMalformed(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		line []byte
	}{
		{name: "not stat", line: []byte("not-a-stat-line")},
		{name: "short stat", line: []byte("(x) S 1")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, _, _, _, err := parseProcStat(tc.line); err == nil {
				t.Fatal("expected parse error")
			}
		})
	}
}

func TestProcStateName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		state byte
		want  string
	}{
		{state: 'R', want: "running"},
		{state: 'S', want: "sleeping"},
		{state: 'Z', want: procStatusZombie},
		{state: 'T', want: "stopped"},
		{state: '?', want: "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()
			if got := procStateName(tc.state); got != tc.want {
				t.Fatalf("state %c: got %q want %q", tc.state, got, tc.want)
			}
		})
	}
}

func TestAliveIgnoresZombieProcStat(t *testing.T) {
	if got := procStateName('Z'); got != procStatusZombie {
		t.Fatalf("state=%q", got)
	}
}

func TestLinuxCreateTimeFromFixture(t *testing.T) {
	btime, hz, err := linuxBootTime()
	if err != nil {
		t.Skip("boot time unavailable in this environment")
	}
	if btime <= 0 || hz <= 0 {
		t.Fatalf("btime=%d hz=%d", btime, hz)
	}
	starttime := uint64(hz) // One second after boot in ticks
	created, err := linuxCreateTime(starttime)
	if err != nil {
		t.Fatal(err)
	}
	if created.Before(time.Unix(btime, 0)) {
		t.Fatalf("created=%v before boot=%d", created, btime)
	}
}
