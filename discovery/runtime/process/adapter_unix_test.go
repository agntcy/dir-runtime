// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//go:build unix

package process

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	runtimev1 "github.com/agntcy/dir/api/runtime/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const blockTimeout = 2 * time.Second

// listIDsWithin lists workload IDs, failing the test if the scan blocks.
func listIDsWithin(t *testing.T, a *adapter) []string {
	t.Helper()

	type result struct {
		workloads []*runtimev1.Workload
		err       error
	}

	done := make(chan result, 1)

	go func() {
		workloads, err := a.ListWorkloads(context.Background())
		done <- result{workloads: workloads, err: err}
	}()

	select {
	case res := <-done:
		require.NoError(t, res.err)

		ids := make([]string, 0, len(res.workloads))
		for _, w := range res.workloads {
			ids = append(ids, w.GetId())
		}

		sort.Strings(ids)

		return ids
	case <-time.After(blockTimeout):
		t.Fatal("ListWorkloads blocked")

		return nil
	}
}

func TestListWorkloadsSkipsFIFO(t *testing.T) {
	a, dir := newTestAdapter(t)
	writeDescriptor(t, dir, "alive", os.Getpid(), discoverable())
	require.NoError(t, syscall.Mkfifo(filepath.Join(dir, "fifo.json"), 0o600))

	assert.Equal(t, []string{"alive"}, listIDsWithin(t, a))
}

func TestListWorkloadsSkipsSymlinkToDevice(t *testing.T) {
	a, dir := newTestAdapter(t)
	writeDescriptor(t, dir, "alive", os.Getpid(), discoverable())
	require.NoError(t, os.Symlink("/dev/zero", filepath.Join(dir, "zero.json")))

	assert.Equal(t, []string{"alive"}, listIDsWithin(t, a))
}

func TestListWorkloadsSkipsSymlinkToDescriptor(t *testing.T) {
	a, dir := newTestAdapter(t)
	target := filepath.Join(t.TempDir(), "elsewhere.json")
	writeDescriptor(t, filepath.Dir(target), "elsewhere", os.Getpid(), discoverable())
	require.NoError(t, os.Symlink(target, filepath.Join(dir, "linked.json")))

	assert.Empty(t, listIDsWithin(t, a))
}

func TestListWorkloadsSkipsOversizedDescriptor(t *testing.T) {
	a, dir := newTestAdapter(t)
	writeDescriptor(t, dir, "alive", os.Getpid(), discoverable())

	padding := strings.Repeat(" ", maxDescriptorSize)
	data := `{"name":"big","pid":` + strconv.Itoa(os.Getpid()) + `,"labels":{"org.agntcy/discover":"true"},"ports":[9999]}` + padding
	require.NoError(t, os.WriteFile(filepath.Join(dir, "big.json"), []byte(data), 0o600))

	assert.Equal(t, []string{"alive"}, listIDs(t, a))
}

func TestReadDescriptorRejectsReplacedFIFO(t *testing.T) {
	// A FIFO that replaces a regular file between the directory listing and the open must not block.
	path := filepath.Join(t.TempDir(), "swapped.json")
	require.NoError(t, syscall.Mkfifo(path, 0o600))

	done := make(chan error, 1)

	go func() {
		_, err := readDescriptorFile(path)
		done <- err
	}()

	select {
	case err := <-done:
		require.ErrorContains(t, err, "not a regular file")
	case <-time.After(blockTimeout):
		t.Fatal("readDescriptorFile blocked on a FIFO")
	}
}

func TestWatchEventsContinuesPastFIFO(t *testing.T) {
	a, dir := newTestAdapter(t)
	require.NoError(t, syscall.Mkfifo(filepath.Join(dir, "fifo.json"), 0o600))
	require.Empty(t, listIDsWithin(t, a))

	events := watch(t, a)

	writeDescriptor(t, dir, "agent", os.Getpid(), discoverable())

	event := nextEvent(t, events)
	assert.Equal(t, "agent", event.Workload.GetId())
}
