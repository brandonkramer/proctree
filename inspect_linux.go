//go:build linux

package proctree

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func readProcessInfo(pid int) (ProcessInfo, error) {
	if !Alive(pid) {
		return ProcessInfo{}, ErrProcessNotFound
	}
	info := ProcessInfo{PID: pid}
	stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return ProcessInfo{}, ErrProcessNotFound
	}
	name, state, ppid, starttime, err := parseProcStat(stat)
	if err != nil {
		return ProcessInfo{}, err
	}
	info.Name = name
	info.Status = procStateName(state)
	info.PPID = ppid
	if pgid, err := syscall.Getpgid(pid); err == nil {
		info.PGID = pgid
	}
	parts, err := readCmdlineParts(pid)
	if err == nil {
		info.Cmdline = parts
	}
	if created, err := linuxCreateTime(starttime); err == nil {
		info.CreateTime = created
	}
	if rss, err := readLinuxRSS(pid); err == nil {
		info.MemoryRSS = rss
	}
	if n, err := readLinuxOpenFiles(pid); err == nil {
		info.OpenFiles = n
	}
	return info, nil
}

func readCmdlineParts(pid int) ([]string, error) {
	cmdline, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return nil, err
	}
	null := byte(0)
	text := string(bytes.TrimRight(cmdline, string(null)))
	if text == "" {
		return nil, fmt.Errorf("empty cmdline")
	}
	return strings.Split(text, string(null)), nil
}

func readCreateTime(pid int) (time.Time, error) {
	stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return time.Time{}, err
	}
	_, _, _, starttime, err := parseProcStat(stat)
	if err != nil {
		return time.Time{}, err
	}
	return linuxCreateTime(starttime)
}

func listChildPIDs(pid int) ([]int, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	var out []int
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		child, err := strconv.Atoi(ent.Name())
		if err != nil {
			continue
		}
		stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", child))
		if err != nil {
			continue
		}
		_, _, ppid, _, err := parseProcStat(stat)
		if err != nil || ppid != pid {
			continue
		}
		out = append(out, child)
	}
	return out, nil
}

func shellPayloadFromParts(parts []string) string {
	if len(parts) >= 3 && parts[1] == "-c" && isShellExecutable(parts[0]) {
		return parts[2]
	}
	return ""
}

func shellPayloadFromCommandLine(line string) string {
	const marker = " -c "
	idx := strings.LastIndex(line, marker)
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(line[idx+len(marker):])
}

func parseProcStat(stat []byte) (name string, state byte, ppid int, starttime uint64, err error) {
	open := bytes.IndexByte(stat, '(')
	close := bytes.LastIndexByte(stat, ')')
	if open < 0 || close <= open {
		return "", 0, 0, 0, fmt.Errorf("invalid proc stat")
	}
	name = string(stat[open+1 : close])
	rest := strings.Fields(string(stat[close+2:]))
	if len(rest) < 20 {
		return "", 0, 0, 0, fmt.Errorf("short proc stat")
	}
	state = rest[0][0]
	ppid, _ = strconv.Atoi(rest[1])
	start, _ := strconv.ParseUint(rest[19], 10, 64)
	return name, state, ppid, start, nil
}

func procStateName(state byte) string {
	switch state {
	case 'R':
		return "running"
	case 'S', 'I':
		return "sleeping"
	case 'D':
		return "blocked"
	case 'Z':
		return "zombie"
	case 'T', 't':
		return "stopped"
	default:
		return "unknown"
	}
}

func linuxCreateTime(starttime uint64) (time.Time, error) {
	btime, hz, err := linuxBootTime()
	if err != nil {
		return time.Time{}, err
	}
	secs := float64(btime) + float64(starttime)/float64(hz)
	return time.Unix(int64(secs), int64((secs-float64(int64(secs)))*1e9)), nil
}

func linuxBootTime() (btime int64, hz int64, err error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "btime ") {
			fields := strings.Fields(line)
			if len(fields) != 2 {
				break
			}
			btime, err = strconv.ParseInt(fields[1], 10, 64)
			if err != nil {
				return 0, 0, err
			}
			break
		}
	}
	if btime == 0 {
		return 0, 0, fmt.Errorf("btime unavailable")
	}
	hz = int64(syscall.Sysconf(syscall.SC_CLK_TCK))
	if hz <= 0 {
		hz = 100
	}
	return btime, hz, nil
}

func readLinuxRSS(pid int) (uint64, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/statm", pid))
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return 0, fmt.Errorf("short statm")
	}
	pages, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0, err
	}
	pageSize := uint64(os.Getpagesize())
	return pages * pageSize, nil
}

func readLinuxOpenFiles(pid int) (int, error) {
	entries, err := os.ReadDir(fmt.Sprintf("/proc/%d/fd", pid))
	if err != nil {
		return 0, err
	}
	return len(entries), nil
}
