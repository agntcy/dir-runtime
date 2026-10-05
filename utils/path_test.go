// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"path/filepath"
	"testing"
)

func TestExpandHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	tests := []struct {
		path string
		want string
	}{
		{path: "~", want: home},
		{path: "~/workloads.db", want: filepath.Join(home, "workloads.db")},
		{path: "~/a/b/c", want: filepath.Join(home, "a", "b", "c")},
		{path: "/abs/path", want: "/abs/path"},
		{path: "relative/path", want: "relative/path"},
		{path: "~user/path", want: "~user/path"},
		{path: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got, err := ExpandHome(tt.path)
			if err != nil {
				t.Fatalf("ExpandHome(%q) error = %v", tt.path, err)
			}

			if got != tt.want {
				t.Errorf("ExpandHome(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
