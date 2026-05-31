//go:build darwin

package proctree

import (
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

const (
	darwinProcStateIdle   int8 = 1
	darwinProcStateRun    int8 = 2
	darwinProcStateSleep  int8 = 3
	darwinProcStateStop   int8 = 4
	darwinProcStateZombie int8 = 5
)

func kinfoProc(pid int) (*unix.KinfoProc, error) {
	return unix.SysctlKinfoProc("kern.proc.pid", pid)
}

func readProcessInfo(pid int) (ProcessInfo, error) {
	if !Alive(pid) {
		return ProcessInfo{}, ErrProcessNotFound
	}
	kp, err := kinfoProc(pid)
	if err != nil {
		return ProcessInfo{}, ErrProcessNotFound
	}
	info := ProcessInfo{
		PID:    pid,
		PPID:   int(kp.Eproc.Ppid),
		PGID:   int(kp.Eproc.Pgid),
		Name:   strings.TrimRight(string(kp.Proc.P_comm[:]), "\x00"),
		Status: darwinStatus(kp.Proc.P_stat),
	}
	if parts, err := readCmdlineParts(pid); err == nil {
		info.Cmdline = parts
	}
	info.CreateTime = darwinStartTime(kp)
	if info.MemoryRSS == 0 && kp.Eproc.Xrssize > 0 {
		info.MemoryRSS = uint64(kp.Eproc.Xrssize) * 1024
	}
	return info, nil
}

func readCmdlineParts(pid int) ([]string, error) {
	args, err := darwinProcArgs(pid)
	if err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("empty cmdline")
	}
	return args, nil
}

func readCreateTime(pid int) (time.Time, error) {
	kp, err := kinfoProc(pid)
	if err != nil {
		return time.Time{}, err
	}
	tm := darwinStartTime(kp)
	if tm.IsZero() {
		return time.Time{}, fmt.Errorf("create time unavailable")
	}
	return tm, nil
}

func listChildPIDs(pid int) ([]int, error) {
	procs, err := unix.SysctlKinfoProcSlice("kern.proc.all")
	if err != nil {
		return nil, err
	}
	var kids []int
	for i := range procs {
		kp := &procs[i]
		if int(kp.Eproc.Ppid) == pid {
			kids = append(kids, int(kp.Proc.P_pid))
		}
	}
	return kids, nil
}

func darwinProcArgs(pid int) ([]string, error) {
	buf, err := unix.SysctlRaw("kern.procargs2", pid)
	if err != nil {
		return nil, err
	}
	if len(buf) < 4 {
		return nil, fmt.Errorf("short procargs")
	}
	_ = binary.LittleEndian.Uint32(buf[:4])
	pos := 4
	for pos < len(buf) && buf[pos] != 0 {
		pos++
	}
	if pos >= len(buf) {
		return nil, fmt.Errorf("empty procargs")
	}
	pos++
	var args []string
	for pos < len(buf) {
		if buf[pos] == 0 {
			pos++
			continue
		}
		end := pos
		for end < len(buf) && buf[end] != 0 {
			end++
		}
		args = append(args, string(buf[pos:end]))
		pos = end + 1
	}
	return args, nil
}

func darwinStartTime(kp *unix.KinfoProc) time.Time {
	tv := kp.Proc.P_starttime
	if tv.Sec == 0 && tv.Usec == 0 {
		return time.Time{}
	}
	return time.Unix(tv.Sec, int64(tv.Usec)*1000).Local()
}

func shellPayloadFromParts(parts []string) string {
	if len(parts) >= 3 && parts[1] == "-c" && isShellExecutable(parts[0]) {
		return parts[2]
	}
	if len(parts) == 1 {
		return shellPayloadFromCommandLine(parts[0])
	}
	return shellPayloadFromCommandLine(strings.Join(parts, " "))
}

func shellPayloadFromCommandLine(line string) string {
	const marker = " -c "
	idx := strings.LastIndex(line, marker)
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(line[idx+len(marker):])
}

func darwinStatus(stat int8) string {
	switch stat {
	case darwinProcStateZombie:
		return "zombie"
	case darwinProcStateRun, darwinProcStateIdle:
		return "running"
	case darwinProcStateSleep:
		return "sleeping"
	case darwinProcStateStop:
		return "stopped"
	default:
		return "unknown"
	}
}
