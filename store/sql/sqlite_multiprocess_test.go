// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sql

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	runtimev1 "github.com/agntcy/dir/api/runtime/v1"
)

const (
	childDBEnv        = "SQLITE_MULTIPROCESS_CHILD_DB"
	multiprocessCount = 100
)

// TestSqliteMultiprocessChild is the child side of TestSqliteMultiprocess.
// It only runs when re-executed by that test, in a separate OS process.
func TestSqliteMultiprocessChild(t *testing.T) {
	path := os.Getenv(childDBEnv)
	if path == "" {
		t.Skip("only runs as a child of TestSqliteMultiprocess")
	}

	ctx := context.Background()
	s := newTestStore(t, path)

	for i := range multiprocessCount {
		if err := s.RegisterWorkload(ctx, &runtimev1.Workload{Id: fmt.Sprintf("child-%d", i)}); err != nil {
			t.Fatalf("child register %d: %v", i, err)
		}

		if _, err := s.ListWorkloads(ctx); err != nil {
			t.Fatalf("child list %d: %v", i, err)
		}
	}
}

// TestSqliteMultiprocess checks that two OS processes can read and write the same
// database file at the same time, as discovery and server do.
func TestSqliteMultiprocess(t *testing.T) {
	if os.Getenv(childDBEnv) != "" {
		t.Skip("parent side")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	path := filepath.Join(t.TempDir(), "workloads.db")
	s := newTestStore(t, path)

	// Re-execute this test binary as the child process.
	//nolint:gosec // G204: os.Args[0] is the running test binary, arguments are constants.
	child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestSqliteMultiprocessChild$", "-test.count=1")

	child.Env = append(os.Environ(), childDBEnv+"="+path)

	output := make(chan []byte, 1)
	childErr := make(chan error, 1)

	go func() {
		out, err := child.CombinedOutput()

		output <- out

		childErr <- err
	}()

	for i := range multiprocessCount {
		if err := s.RegisterWorkload(ctx, &runtimev1.Workload{Id: fmt.Sprintf("parent-%d", i)}); err != nil {
			t.Errorf("parent register %d: %v", i, err)
		}

		if _, err := s.ListWorkloads(ctx); err != nil {
			t.Errorf("parent list %d: %v", i, err)
		}
	}

	out := <-output
	if err := <-childErr; err != nil {
		t.Fatalf("child process failed: %v\n%s", err, out)
	}

	ids, err := s.ListWorkloadIDs(ctx)
	if err != nil {
		t.Fatalf("ListWorkloadIDs() error = %v", err)
	}

	for i := range multiprocessCount {
		for _, id := range []string{fmt.Sprintf("parent-%d", i), fmt.Sprintf("child-%d", i)} {
			if _, ok := ids[id]; !ok {
				t.Errorf("workload %s missing after both processes finished", id)
			}
		}
	}
}
