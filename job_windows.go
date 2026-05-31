//go:build windows

package proctree

import (
	"os/exec"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	jobObjectExtendedLimitInformation = 9
	jobObjectLimitKillOnJobClose      = 0x00002000
)

type jobBasicLimitInfo struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
}

type jobIoCounters struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

type jobExtendedLimitInfo struct {
	BasicLimitInformation jobBasicLimitInfo
	IoInfo                jobIoCounters
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

var (
	procCreateJobObjectW         = modKernel32.NewProc("CreateJobObjectW")
	procAssignProcessToJobObject = modKernel32.NewProc("AssignProcessToJobObject")
	procSetInformationJobObject  = modKernel32.NewProc("SetInformationJobObject")
	procTerminateJobObject       = modKernel32.NewProc("TerminateJobObject")

	pidJobs sync.Map // int -> windows.Handle
)

func attachRunJob(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	job, err := createKillOnCloseJob()
	if err != nil {
		return err
	}
	if err := assignPIDToJob(cmd.Process.Pid, job); err != nil {
		_ = windows.CloseHandle(job)
		return err
	}
	if prev, loaded := pidJobs.Swap(cmd.Process.Pid, job); loaded {
		_ = windows.CloseHandle(prev.(windows.Handle))
	}
	return nil
}

func releaseRunJob(pid int) {
	if v, ok := pidJobs.LoadAndDelete(pid); ok {
		_ = windows.CloseHandle(v.(windows.Handle))
	}
}

func killWindowsJob(pid int) bool {
	v, ok := pidJobs.LoadAndDelete(pid)
	if !ok {
		return false
	}
	job := v.(windows.Handle)
	_, _, _ = procTerminateJobObject.Call(uintptr(job), 1)
	_ = windows.CloseHandle(job)
	return true
}

func createKillOnCloseJob() (windows.Handle, error) {
	r0, _, err := procCreateJobObjectW.Call(0, 0)
	if r0 == 0 {
		return 0, err
	}
	job := windows.Handle(r0)
	var info jobExtendedLimitInfo
	info.BasicLimitInformation.LimitFlags = jobObjectLimitKillOnJobClose
	r0, _, err = procSetInformationJobObject.Call(
		uintptr(job),
		uintptr(jobObjectExtendedLimitInformation),
		uintptr(unsafe.Pointer(&info)),
		uintptr(unsafe.Sizeof(info)),
	)
	if r0 == 0 {
		_ = windows.CloseHandle(job)
		return 0, err
	}
	return job, nil
}

func assignPIDToJob(pid int, job windows.Handle) error {
	const access = windows.PROCESS_TERMINATE | 0x0100 // PROCESS_SET_QUOTAS
	handle, err := windows.OpenProcess(access, false, uint32(pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)
	r0, _, err := procAssignProcessToJobObject.Call(uintptr(job), uintptr(handle), 0)
	if r0 == 0 {
		return err
	}
	return nil
}
