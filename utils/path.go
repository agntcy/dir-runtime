// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExpandHome replaces a leading "~" in path with the current user's home directory.
// Paths without a leading "~" or "~/" are returned unchanged.
func ExpandHome(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve home directory: %w", err)
	}

	return filepath.Join(home, strings.TrimPrefix(path, "~")), nil
}
