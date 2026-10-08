// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//go:build unix

package process

import (
	"errors"
	"syscall"
)

// pidCheckSupported reports whether process liveness can be checked on this platform.
const pidCheckSupported = true

// isAlive reports whether a process with the given PID exists.
// EPERM means the process exists but belongs to another user, so it counts as alive.
func isAlive(pid int) bool {
	err := syscall.Kill(pid, 0)

	return err == nil || errors.Is(err, syscall.EPERM)
}
