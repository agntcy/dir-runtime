// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sql

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/agntcy/dir-runtime/store/types"
	runtimev1 "github.com/agntcy/dir/api/runtime/v1"
	"gorm.io/gorm"
)

func openTestDB(t *testing.T, path string) *gorm.DB {
	t.Helper()

	db, err := NewSqlite(Config{Path: path})
	if err != nil {
		t.Fatalf("NewSqlite() error = %v", err)
	}

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err != nil {
			t.Errorf("db.DB() error = %v", err)

			return
		}

		if err := sqlDB.Close(); err != nil {
			t.Errorf("close error = %v", err)
		}
	})

	return db
}

func newTestStore(t *testing.T, path string) types.Store {
	t.Helper()

	s, err := New(openTestDB(t, path))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return s
}

func TestNewSqliteCreatesParentDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "dir", "workloads.db")

	newTestStore(t, path)

	info, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("parent dir not created: %v", err)
	}

	if perm := info.Mode().Perm(); perm != dirPerm {
		t.Errorf("parent dir perm = %o, want %o", perm, dirPerm)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("database file not created: %v", err)
	}
}

func TestNewSqliteExpandsHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	newTestStore(t, "~/.agntcy/workloads.db")

	if _, err := os.Stat(filepath.Join(home, ".agntcy", "workloads.db")); err != nil {
		t.Errorf("database not created under home: %v", err)
	}
}

func TestNewSqliteEmptyPath(t *testing.T) {
	_, err := NewSqlite(Config{Path: ""})
	if err == nil || !strings.Contains(err.Error(), "path is required") {
		t.Errorf("NewSqlite() error = %v, want 'path is required'", err)
	}
}

func TestNewSqlitePragmas(t *testing.T) {
	db := openTestDB(t, filepath.Join(t.TempDir(), "workloads.db"))

	var mode string
	if err := db.Raw("PRAGMA journal_mode").Scan(&mode).Error; err != nil {
		t.Fatalf("journal_mode query error = %v", err)
	}

	if mode != "wal" {
		t.Errorf("journal_mode = %q, want %q", mode, "wal")
	}

	var timeout int
	if err := db.Raw("PRAGMA busy_timeout").Scan(&timeout).Error; err != nil {
		t.Fatalf("busy_timeout query error = %v", err)
	}

	if timeout != busyTimeoutMs {
		t.Errorf("busy_timeout = %d, want %d", timeout, busyTimeoutMs)
	}

	var foreignKeys int
	if err := db.Raw("PRAGMA foreign_keys").Scan(&foreignKeys).Error; err != nil {
		t.Fatalf("foreign_keys query error = %v", err)
	}

	if foreignKeys != 1 {
		t.Errorf("foreign_keys = %d, want 1", foreignKeys)
	}
}

func TestSqliteSharedBetweenStores(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "workloads.db")

	writer := newTestStore(t, path)
	reader := newTestStore(t, path)

	if err := writer.RegisterWorkload(ctx, &runtimev1.Workload{Id: "w1", Name: "agent"}); err != nil {
		t.Fatalf("RegisterWorkload() error = %v", err)
	}

	workload, err := reader.GetWorkload(ctx, "w1")
	if err != nil {
		t.Fatalf("reader GetWorkload() error = %v", err)
	}

	if workload.GetName() != "agent" {
		t.Errorf("reader got name %q, want %q", workload.GetName(), "agent")
	}

	if err := writer.DeregisterWorkload(ctx, "w1"); err != nil {
		t.Fatalf("DeregisterWorkload() error = %v", err)
	}

	ids, err := reader.ListWorkloadIDs(ctx)
	if err != nil {
		t.Fatalf("reader ListWorkloadIDs() error = %v", err)
	}

	if len(ids) != 0 {
		t.Errorf("reader still sees workloads %v after deregister", ids)
	}
}

func TestSqliteDataSurvivesReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "workloads.db")

	db, err := NewSqlite(Config{Path: path})
	if err != nil {
		t.Fatalf("NewSqlite() error = %v", err)
	}

	first, err := New(db)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := first.RegisterWorkload(ctx, &runtimev1.Workload{Id: "w1"}); err != nil {
		t.Fatalf("RegisterWorkload() error = %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}

	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close error = %v", err)
	}

	ids, err := newTestStore(t, path).ListWorkloadIDs(ctx)
	if err != nil {
		t.Fatalf("ListWorkloadIDs() error = %v", err)
	}

	if _, ok := ids["w1"]; !ok {
		t.Errorf("workload w1 lost after reopen, got %v", ids)
	}
}

func TestSqliteConcurrentWriterAndReader(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	path := filepath.Join(t.TempDir(), "workloads.db")
	writer := newTestStore(t, path)
	reader := newTestStore(t, path)

	const iterations = 200

	var (
		mu   sync.Mutex
		errs []error
		wg   sync.WaitGroup
	)

	record := func(err error) {
		mu.Lock()
		defer mu.Unlock()

		errs = append(errs, err)
	}

	wg.Go(func() {
		for i := range iterations {
			id := fmt.Sprintf("w%d", i%10)

			if err := writer.RegisterWorkload(ctx, &runtimev1.Workload{Id: id, Name: fmt.Sprint(i)}); err != nil {
				record(fmt.Errorf("register %s: %w", id, err))
			}

			if i%3 == 0 {
				if err := writer.DeregisterWorkload(ctx, id); err != nil {
					record(fmt.Errorf("deregister %s: %w", id, err))
				}
			}
		}
	})

	wg.Go(func() {
		for range iterations {
			if _, err := reader.ListWorkloads(ctx); err != nil {
				record(fmt.Errorf("list: %w", err))
			}
		}
	})

	wg.Wait()

	for _, err := range errs {
		t.Error(err)
	}
}

func TestNewSqliteSpecialCharactersInPath(t *testing.T) {
	for _, name := range []string{"work?loads.db", "work loads.db", "work#loads.db", "work%3Floads.db"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, name)

			newTestStore(t, path)

			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatalf("ReadDir() error = %v", err)
			}

			if _, err := os.Stat(path); err != nil {
				names := make([]string, 0, len(entries))
				for _, e := range entries {
					names = append(names, e.Name())
				}

				t.Errorf("database not created at %q, directory has %v", path, names)
			}
		})
	}
}

func TestNewSqliteRelativePath(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	newTestStore(t, filepath.Join("data", "workloads.db"))

	if _, err := os.Stat(filepath.Join(dir, "data", "workloads.db")); err != nil {
		t.Errorf("database not created relative to working directory: %v", err)
	}
}

func TestStoreCloseClosesPool(t *testing.T) {
	s := newTestStore(t, filepath.Join(t.TempDir(), "workloads.db"))

	if err := s.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if _, err := s.ListWorkloads(context.Background()); err == nil || !strings.Contains(err.Error(), "database is closed") {
		t.Errorf("ListWorkloads() after Close error = %v, want 'database is closed'", err)
	}
}
