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

	"github.com/agntcy/dir-runtime/discovery/types"
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

// startProcess starts a long-running process that is killed when the test ends.
func startProcess(t *testing.T) *exec.Cmd {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), "sleep", "60")
	require.NoError(t, cmd.Start())

	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	return cmd
}

const (
	eventTimeout   = 2 * time.Second
	noEventTimeout = 100 * time.Millisecond
)

// watch starts WatchEvents in the background and returns its event channel.
func watch(t *testing.T, a *adapter) <-chan *types.RuntimeEvent {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	events := make(chan *types.RuntimeEvent, 16)
	done := make(chan error, 1)

	go func() { done <- a.WatchEvents(ctx, events) }()

	t.Cleanup(func() {
		cancel()
		require.ErrorIs(t, <-done, context.Canceled)
	})

	return events
}

func nextEvent(t *testing.T, events <-chan *types.RuntimeEvent) *types.RuntimeEvent {
	t.Helper()

	select {
	case event := <-events:
		return event
	case <-time.After(eventTimeout):
		t.Fatal("timed out waiting for event")

		return nil
	}
}

func assertNoEvent(t *testing.T, events <-chan *types.RuntimeEvent) {
	t.Helper()

	select {
	case event := <-events:
		t.Fatalf("unexpected %s event for %s", event.Type, event.Workload.GetId())
	case <-time.After(noEventTimeout):
	}
}

func TestWatchEventsLifecycle(t *testing.T) {
	a, dir := newTestAdapter(t)
	require.Empty(t, listIDs(t, a))

	events := watch(t, a)

	writeDescriptor(t, dir, "agent", os.Getpid(), discoverable())

	event := nextEvent(t, events)
	assert.Equal(t, types.RuntimeEventTypeAdded, event.Type)
	assert.Equal(t, "agent", event.Workload.GetId())
	assert.Equal(t, "process", event.Workload.GetType())

	assertNoEvent(t, events)

	labels := discoverable()
	labels["org.agntcy/agent-type"] = "a2a"
	writeDescriptor(t, dir, "agent", os.Getpid(), labels)

	event = nextEvent(t, events)
	assert.Equal(t, types.RuntimeEventTypeModified, event.Type)
	assert.Equal(t, "a2a", event.Workload.GetLabels()["org.agntcy/agent-type"])

	require.NoError(t, os.Remove(filepath.Join(dir, "agent.json")))

	event = nextEvent(t, events)
	assert.Equal(t, types.RuntimeEventTypeDeleted, event.Type)
	assert.Equal(t, "agent", event.Workload.GetId())
}

func TestWatchEventsBaselineNotReemitted(t *testing.T) {
	a, dir := newTestAdapter(t)
	writeDescriptor(t, dir, "agent", os.Getpid(), discoverable())
	require.Equal(t, []string{"agent"}, listIDs(t, a))

	events := watch(t, a)

	assertNoEvent(t, events)
}

func TestWatchEventsProcessExit(t *testing.T) {
	a, dir := newTestAdapter(t)
	cmd := startProcess(t)
	writeDescriptor(t, dir, "agent", cmd.Process.Pid, discoverable())
	require.Equal(t, []string{"agent"}, listIDs(t, a))

	events := watch(t, a)

	require.NoError(t, cmd.Process.Kill())
	_ = cmd.Wait()

	event := nextEvent(t, events)
	assert.Equal(t, types.RuntimeEventTypeDeleted, event.Type)
	assert.Equal(t, "agent", event.Workload.GetId())
	assert.FileExists(t, filepath.Join(dir, "agent.json"), "adapter must not delete descriptors")
}

func TestWatchEventsLabelRemoved(t *testing.T) {
	a, dir := newTestAdapter(t)
	writeDescriptor(t, dir, "agent", os.Getpid(), discoverable())
	require.Equal(t, []string{"agent"}, listIDs(t, a))

	events := watch(t, a)

	writeDescriptor(t, dir, "agent", os.Getpid(), map[string]string{"app": "x"})

	event := nextEvent(t, events)
	assert.Equal(t, types.RuntimeEventTypeDeleted, event.Type)
}

func TestWatchEventsPIDChangeIsModified(t *testing.T) {
	a, dir := newTestAdapter(t)
	first := startProcess(t)
	writeDescriptor(t, dir, "agent", first.Process.Pid, discoverable())
	require.Equal(t, []string{"agent"}, listIDs(t, a))

	events := watch(t, a)

	second := startProcess(t)
	writeDescriptor(t, dir, "agent", second.Process.Pid, discoverable())

	event := nextEvent(t, events)
	assert.Equal(t, types.RuntimeEventTypeModified, event.Type)
}

func TestWatchEventsKeepsStateOnTransientParseError(t *testing.T) {
	a, dir := newTestAdapter(t)
	writeDescriptor(t, dir, "agent", os.Getpid(), discoverable())
	require.Equal(t, []string{"agent"}, listIDs(t, a))

	events := watch(t, a)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "agent.json"), []byte(`{"name":`), 0o600))
	assertNoEvent(t, events)

	writeDescriptor(t, dir, "agent", os.Getpid(), discoverable())
	assertNoEvent(t, events)
}

func TestWatchEventsKeepsStateOnScanError(t *testing.T) {
	a, dir := newTestAdapter(t)
	writeDescriptor(t, dir, "agent", os.Getpid(), discoverable())
	require.Equal(t, []string{"agent"}, listIDs(t, a))

	events := watch(t, a)

	require.NoError(t, os.RemoveAll(dir))
	assertNoEvent(t, events)
}

func TestDiff(t *testing.T) {
	a, _ := newTestAdapter(t)
	desc, err := parseDescriptor([]byte(`{"name":"agent","pid":1,"ports":[9999]}`))
	require.NoError(t, err)

	base := entry{workload: desc.toWorkload("same", a.hostname), pid: 1}
	changed := entry{workload: desc.toWorkload("changed", a.hostname), pid: 1}
	changedNew := entry{workload: desc.toWorkload("changed", a.hostname), pid: 1}
	changedNew.workload.Name = "renamed"
	gone := entry{workload: desc.toWorkload("gone", a.hostname), pid: 1}
	added := entry{workload: desc.toWorkload("added", a.hostname), pid: 1}

	events := diff(
		map[string]entry{"same": base, "changed": changed, "gone": gone},
		map[string]entry{"same": base, "changed": changedNew, "added": added},
	)

	got := make(map[string]types.RuntimeEventType, len(events))
	for _, event := range events {
		got[event.Workload.GetId()] = event.Type
	}

	assert.Equal(t, map[string]types.RuntimeEventType{
		"changed": types.RuntimeEventTypeModified,
		"gone":    types.RuntimeEventTypeDeleted,
		"added":   types.RuntimeEventTypeAdded,
	}, got)
}
