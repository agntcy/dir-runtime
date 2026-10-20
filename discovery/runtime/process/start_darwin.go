// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//go:build darwin

package process

import (
	"time"

	"golang.org/x/sys/unix"
)

// processStartTime returns when the process with the given PID started (sysctl kern.proc.pid).
func processStartTime(pid int) (time.Time, bool) {
	info, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil || info.Proc.P_pid != int32(pid) { //nolint:gosec // PIDs fit in int32
		return time.Time{}, false
	}

	start := info.Proc.P_starttime

	return time.Unix(start.Sec, int64(start.Usec)*int64(time.Microsecond)), true
}
