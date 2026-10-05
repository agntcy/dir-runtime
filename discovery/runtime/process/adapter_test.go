// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package process

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testPollInterval = 10 * time.Millisecond

func newTestAdapter(t *testing.T) (*adapter, string) {
	t.Helper()

	dir := t.TempDir()

	runtimeAdapter, err := NewAdapter(Config{
		Dir:          dir,
		PollInterval: testPollInterval,
		LabelKey:     DefaultLabelKey,
		LabelValue:   DefaultLabelValue,
	})
	require.NoError(t, err)

	impl, ok := runtimeAdapter.(*adapter)
	require.True(t, ok)

	return impl, dir
}

func discoverable() map[string]string {
	return map[string]string{DefaultLabelKey: DefaultLabelValue}
}

func writeDescriptor(t *testing.T, dir, id string, pid int, labels map[string]string) {
	t.Helper()

	data, err := json.Marshal(map[string]any{
		"name":   id,
		"pid":    pid,
		"labels": labels,
		"ports":  []string{"9999"},
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, id+".json"), data, 0o600))
}

// deadPID returns the PID of a process that has already exited and been reaped.
func deadPID(t *testing.T) int {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), "true")
	require.NoError(t, cmd.Run())

	return cmd.ProcessState.Pid()
}

func listIDs(t *testing.T, a *adapter) []string {
	t.Helper()

	workloads, err := a.ListWorkloads(context.Background())
	require.NoError(t, err)

	ids := make([]string, 0, len(workloads))
	for _, w := range workloads {
		ids = append(ids, w.GetId())
	}

	sort.Strings(ids)

	return ids
}

func TestIsAlive(t *testing.T) {
	assert.True(t, isAlive(os.Getpid()), "current process")
	assert.True(t, isAlive(1), "PID 1 is owned by root; EPERM must count as alive")
	assert.False(t, isAlive(deadPID(t)), "exited process")
}

func TestNewAdapterCreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "workloads.d")

	_, err := NewAdapter(Config{Dir: dir, PollInterval: testPollInterval})
	require.NoError(t, err)

	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
	assert.Equal(t, os.FileMode(0o700), info.Mode().Perm())
}

func TestNewAdapterExpandsHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	runtimeAdapter, err := NewAdapter(Config{Dir: "~/workloads.d", PollInterval: testPollInterval})
	require.NoError(t, err)

	impl, ok := runtimeAdapter.(*adapter)
	require.True(t, ok)
	assert.Equal(t, filepath.Join(home, "workloads.d"), impl.dir)
	assert.DirExists(t, filepath.Join(home, "workloads.d"))
}

func TestNewAdapterInvalidConfig(t *testing.T) {
	_, err := NewAdapter(Config{Dir: "", PollInterval: testPollInterval})
	require.ErrorContains(t, err, "dir is required")

	_, err = NewAdapter(Config{Dir: t.TempDir(), PollInterval: 0})
	require.ErrorContains(t, err, "poll interval must be positive")
}

func TestNewAdapterUnreadableDir(t *testing.T) {
	file := filepath.Join(t.TempDir(), "not-a-dir")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o600))

	_, err := NewAdapter(Config{Dir: file, PollInterval: testPollInterval})
	require.Error(t, err)
}

func TestAdapterType(t *testing.T) {
	a, _ := newTestAdapter(t)

	assert.Equal(t, RuntimeType, a.Type())
	require.NoError(t, a.Close())
}

func TestListWorkloads(t *testing.T) {
	a, dir := newTestAdapter(t)

	writeDescriptor(t, dir, "alive", os.Getpid(), discoverable())
	writeDescriptor(t, dir, "unlabeled", os.Getpid(), map[string]string{"app": "x"})
	writeDescriptor(t, dir, "wrong-value", os.Getpid(), map[string]string{DefaultLabelKey: "false"})
	writeDescriptor(t, dir, "dead", deadPID(t), discoverable())
	writeDescriptor(t, dir, ".hidden", os.Getpid(), discoverable())
	require.NoError(t, os.WriteFile(filepath.Join(dir, "partial.json.tmp"), []byte(`{}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hi"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "broken.json"), []byte(`{"name":`), 0o600))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "subdir.json"), 0o700))

	assert.Equal(t, []string{"alive"}, listIDs(t, a))
}

func TestListWorkloadsReturnsCopies(t *testing.T) {
	a, dir := newTestAdapter(t)
	writeDescriptor(t, dir, "alive", os.Getpid(), discoverable())

	workloads, err := a.ListWorkloads(context.Background())
	require.NoError(t, err)
	require.Len(t, workloads, 1)

	workloads[0].Name = "mutated"

	assert.Equal(t, "alive", a.known["alive"].workload.GetName())
}

func TestListWorkloadsDirRemoved(t *testing.T) {
	a, dir := newTestAdapter(t)
	require.NoError(t, os.RemoveAll(dir))

	_, err := a.ListWorkloads(context.Background())
	require.Error(t, err)
}
