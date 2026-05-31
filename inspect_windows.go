//go:build windows

package proctree

import (
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type processEntry32 struct {
	Size            uint32
	CntUsage        uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	CntThreads      uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [syscall.MAX_PATH]uint16
}

type processMemoryCounters struct {
	CB                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
}

type unicodeString struct {
	Length        uint16
	MaximumLength uint16
	_             [4]byte
	Buffer        *uint16
}

const (
	thSnapProcess                 = 0x00000002
	processCommandLineInformation = 60
)

var (
	modKernel32                   = windows.NewLazySystemDLL("kernel32.dll")
	modPsapi                      = windows.NewLazySystemDLL("psapi.dll")
	modNtdll                      = windows.NewLazySystemDLL("ntdll.dll")
	procCreateToolhelp32Snapshot  = modKernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32FirstW           = modKernel32.NewProc("Process32FirstW")
	procProcess32NextW            = modKernel32.NewProc("Process32NextW")
	procGetProcessMemoryInfo      = modPsapi.NewProc("GetProcessMemoryInfo")
	procReadProcessMemory         = modKernel32.NewProc("ReadProcessMemory")
	procNtQueryInformationProcess = modNtdll.NewProc("NtQueryInformationProcess")
)

func readProcessInfo(pid int) (ProcessInfo, error) {
	if !Alive(pid) {
		return ProcessInfo{}, ErrProcessNotFound
	}
	info := ProcessInfo{PID: pid, Status: "running"}
	table, err := snapshotProcesses()
	if err != nil {
		return ProcessInfo{}, err
	}
	for _, ent := range table {
		if int(ent.ProcessID) == pid {
			info.PPID = int(ent.ParentProcessID)
			info.Name = syscall.UTF16ToString(ent.ExeFile[:])
			break
		}
	}
	handle, err := openProcessQuery(uint32(pid))
	if err != nil {
		return info, nil
	}
	defer windows.CloseHandle(handle)

	if parts, err := readCmdlineParts(pid); err == nil {
		info.Cmdline = parts
	}
	if created, err := readCreateTime(pid); err == nil {
		info.CreateTime = created
	}
	if rss, err := windowsRSSHandle(handle); err == nil {
		info.MemoryRSS = rss
	}
	return info, nil
}

func readCmdlineParts(pid int) ([]string, error) {
	handle, err := openProcessQuery(uint32(pid))
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(handle)

	line, err := queryProcessCommandLine(handle)
	if err != nil {
		return nil, err
	}
	if line == "" {
		return nil, fmt.Errorf("empty cmdline")
	}
	return []string{line}, nil
}

func readCreateTime(pid int) (time.Time, error) {
	handle, err := openProcessQuery(uint32(pid))
	if err != nil {
		return time.Time{}, err
	}
	defer windows.CloseHandle(handle)
	return processCreateTime(handle)
}

func listChildPIDs(pid int) ([]int, error) {
	table, err := snapshotProcesses()
	if err != nil {
		return nil, err
	}
	var kids []int
	for _, ent := range table {
		if int(ent.ParentProcessID) == pid {
			kids = append(kids, int(ent.ProcessID))
		}
	}
	return kids, nil
}

func shellPayloadFromParts(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	line := parts[0]
	if len(parts) > 1 {
		line = strings.Join(parts, " ")
	}
	return shellPayloadFromCommandLine(line)
}

func shellPayloadFromCommandLine(line string) string {
	lower := strings.ToLower(line)
	i := strings.Index(lower, "/c")
	if i < 0 {
		return ""
	}
	payload := strings.TrimSpace(line[i+2:])
	payload = strings.Trim(payload, `"`)
	return normalizeWindowsShellPayload(payload)
}

func normalizeWindowsShellPayload(payload string) string {
	payload = strings.TrimSpace(payload)
	payload = strings.ReplaceAll(payload, "1>NUL", ">nul")
	payload = strings.ReplaceAll(payload, "1>nul", ">nul")
	return payload
}

func openProcessQuery(pid uint32) (windows.Handle, error) {
	const access = windows.PROCESS_QUERY_LIMITED_INFORMATION | windows.PROCESS_VM_READ
	handle, err := windows.OpenProcess(access, false, pid)
	if err == nil {
		return handle, nil
	}
	const fullAccess = windows.PROCESS_QUERY_INFORMATION | windows.PROCESS_VM_READ
	return windows.OpenProcess(fullAccess, false, pid)
}

func queryProcessCommandLine(handle windows.Handle) (string, error) {
	buf := make([]byte, 4096)
	var retLen uint32
	r0, _, e1 := procNtQueryInformationProcess.Call(
		uintptr(handle),
		uintptr(processCommandLineInformation),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		uintptr(unsafe.Pointer(&retLen)),
	)
	if r0 != 0 {
		return "", e1
	}
	us := (*unicodeString)(unsafe.Pointer(&buf[0]))
	if us.Buffer == nil || us.Length == 0 {
		return "", fmt.Errorf("cmdline unavailable")
	}
	if us.Length%2 != 0 {
		return "", fmt.Errorf("cmdline unavailable")
	}
	n := int(us.Length / 2)
	raw := make([]uint16, n)
	var nread uintptr
	r0, _, e1 = procReadProcessMemory.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(us.Buffer)),
		uintptr(unsafe.Pointer(&raw[0])),
		uintptr(us.Length),
		uintptr(unsafe.Pointer(&nread)),
	)
	if r0 == 0 {
		return "", e1
	}
	return windows.UTF16ToString(raw), nil
}

func processCreateTime(handle windows.Handle) (time.Time, error) {
	var created, exited, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(handle, &created, &exited, &kernel, &user); err != nil {
		return time.Time{}, err
	}
	return filetimeToTime(created), nil
}

func windowsRSS(pid int) (uint64, error) {
	handle, err := openProcessQuery(uint32(pid))
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(handle)
	return windowsRSSHandle(handle)
}

func windowsRSSHandle(handle windows.Handle) (uint64, error) {
	var counters processMemoryCounters
	counters.CB = uint32(unsafe.Sizeof(counters))
	r1, _, err := procGetProcessMemoryInfo.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&counters)),
		uintptr(counters.CB),
	)
	if r1 == 0 {
		return 0, err
	}
	return uint64(counters.WorkingSetSize), nil
}

func filetimeToTime(ft windows.Filetime) time.Time {
	nsec := ft.Nanoseconds()
	if nsec <= 0 {
		return time.Time{}
	}
	return time.Unix(0, nsec).Local()
}

func snapshotProcesses() ([]processEntry32, error) {
	handle, _, err := procCreateToolhelp32Snapshot.Call(thSnapProcess, 0)
	if handle == uintptr(windows.InvalidHandle) {
		return nil, err
	}
	defer windows.CloseHandle(windows.Handle(handle))
	var entry processEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	r1, _, err := procProcess32FirstW.Call(handle, uintptr(unsafe.Pointer(&entry)))
	if r1 == 0 {
		return nil, err
	}
	var out []processEntry32
	for {
		out = append(out, entry)
		r1, _, err = procProcess32NextW.Call(handle, uintptr(unsafe.Pointer(&entry)))
		if r1 == 0 {
			break
		}
	}
	return out, nil
}
