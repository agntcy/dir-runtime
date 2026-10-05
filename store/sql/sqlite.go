// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sql

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/agntcy/dir-runtime/utils"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// StoreType is the identifier for the SQLite store type.
const StoreTypeSqlite = "sqlite"

const (
	// dirPerm is the permission used when creating the database parent directory.
	dirPerm = 0o700

	// busyTimeoutMs is how long a connection waits for a lock held by another process.
	busyTimeoutMs = 5000
)

// NewSqlite creates a new database connection using the pure-Go SQLite driver.
// The database file is shared by every component configured with the same path,
// e.g. discovery writing and server reading. WAL mode and a busy timeout let one
// writer and concurrent readers use the file from separate processes.
func NewSqlite(cfg Config) (*gorm.DB, error) {
	if cfg.Path == "" {
		return nil, errors.New("sqlite path is required")
	}

	path, err := utils.ExpandHome(cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("invalid sqlite path: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		return nil, fmt.Errorf("failed to create directory for SQLite database: %w", err)
	}

	// Pragmas are applied to every pooled connection by the driver.
	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(%d)&_pragma=foreign_keys(1)", path, busyTimeoutMs)

	// Create database
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: gormlogger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			gormlogger.Config{
				SlowThreshold:             200 * time.Millisecond, //nolint:mnd
				LogLevel:                  gormlogger.Warn,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SQLite database: %w", err)
	}

	return db, nil
}
