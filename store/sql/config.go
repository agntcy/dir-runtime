// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sql

const (
	// DefaultPath is the default SQLite database file path.
	DefaultPath = "~/.agntcy/dir-runtime/workloads.db"
)

// Config holds SQLite store configuration.
type Config struct {
	// Path is the SQLite database file path. A leading "~" is expanded to the user's home.
	// Discovery and server must use the same path to share workloads.
	Path string `json:"path,omitempty" mapstructure:"path"`
}
