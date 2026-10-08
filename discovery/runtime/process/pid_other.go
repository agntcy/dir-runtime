// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//go:build !unix

package process

// pidCheckSupported reports whether process liveness can be checked on this platform.
const pidCheckSupported = false

// isAlive is not supported on this platform; NewAdapter refuses to start before it is called.
func isAlive(_ int) bool {
	return false
}
