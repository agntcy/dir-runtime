// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//go:build !linux && !darwin

package process

import "time"

// processStartTime is not available on this platform; liveness falls back to the PID check.
func processStartTime(_ int) (time.Time, bool) {
	return time.Time{}, false
}
