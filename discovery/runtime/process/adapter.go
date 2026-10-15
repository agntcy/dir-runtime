// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package process

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/agntcy/dir-runtime/discovery/types"
	"github.com/agntcy/dir-runtime/utils"
	runtimev1 "github.com/agntcy/dir/api/runtime/v1"
	"google.golang.org/protobuf/proto"
)

const (
	descriptorExt     = ".json"
	maxDescriptorSize = 64 << 10
	dirPerm           = 0o700
	fallbackHost      = "localhost"
)

var logger = utils.NewLogger("runtime", "process")

// entry is a discovered workload together with the process it was discovered for.
type entry struct {
	workload  *runtimev1.Workload
	pid       int
	startedAt *time.Time
}

// startSlack allows for the clock granularity between a process starting and writing its
// descriptor's started_at.
const startSlack = time.Second

// isRunning reports whether the process a descriptor describes is still running: its PID
// exists and, when the descriptor says when it was written, the process holding the PID
// started no later than that (otherwise the PID was reused). Where the start time can't
// be read, the PID check alone applies.
func isRunning(pid int, startedAt *time.Time) bool {
	if !isAlive(pid) {
		return false
	}

	if startedAt == nil {
		return true
	}

	started, ok := processStartTime(pid)
	if !ok {
		return true
	}

	return !started.After(startedAt.Add(startSlack))
}

// fileState identifies a descriptor file version, used to log invalid files only once per change.
type fileState struct {
	modTime time.Time
	size    int64
}

// adapter implements the RuntimeAdapter interface for processes announced via descriptor files.
type adapter struct {
	dir          string
	pollInterval time.Duration
	labelKey     string
	labelValue   string
	keepStale    bool
	hostname     string

	mu      sync.Mutex
	known   map[string]entry     // last scan result, used as the watch baseline
	invalid map[string]fileState // invalid descriptor files already logged
}

// NewAdapter creates a new process adapter.
func NewAdapter(cfg Config) (types.RuntimeAdapter, error) {
	if !pidCheckSupported {
		return nil, errors.New("process runtime is only supported on Unix")
	}

	if cfg.Dir == "" {
		return nil, errors.New("process runtime dir is required")
	}

	if cfg.PollInterval <= 0 {
		return nil, fmt.Errorf("process runtime poll interval must be positive, got %s", cfg.PollInterval)
	}

	dir, err := utils.ExpandHome(cfg.Dir)
	if err != nil {
		return nil, fmt.Errorf("invalid process runtime dir: %w", err)
	}

	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return nil, fmt.Errorf("failed to create descriptor directory %s: %w", dir, err)
	}

	if _, err := os.ReadDir(dir); err != nil {
		return nil, fmt.Errorf("failed to read descriptor directory %s: %w", dir, err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = fallbackHost
	}

	logger.Info("watching descriptor directory", "dir", dir, "poll_interval", cfg.PollInterval.String())

	return &adapter{
		dir:          dir,
		pollInterval: cfg.PollInterval,
		labelKey:     cfg.LabelKey,
		labelValue:   cfg.LabelValue,
		keepStale:    cfg.KeepStale,
		hostname:     hostname,
		known:        make(map[string]entry),
		invalid:      make(map[string]fileState),
	}, nil
}

// Type returns the process runtime type.
func (a *adapter) Type() types.RuntimeType {
	return RuntimeType
}

// Close releases adapter resources. The process adapter holds none.
func (a *adapter) Close() error {
	return nil
}

// ListWorkloads returns all discoverable process workloads and stores them as the watch baseline.
func (a *adapter) ListWorkloads(_ context.Context) ([]*runtimev1.Workload, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	current, err := a.scan(a.known)
	if err != nil {
		return nil, err
	}

	a.known = current

	workloads := make([]*runtimev1.Workload, 0, len(current))
	for _, e := range current {
		workloads = append(workloads, e.workload.DeepCopy())
	}

	return workloads, nil
}

// scan reads all descriptors in the directory and returns the discoverable workloads keyed by ID.
// A file that cannot be read or parsed keeps its previous entry while that entry's PID is alive,
// so a descriptor caught mid-write does not make its workload flap.
// The caller must hold a.mu.
func (a *adapter) scan(prev map[string]entry) (map[string]entry, error) {
	files, err := os.ReadDir(a.dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read descriptor directory %s: %w", a.dir, err)
	}

	current := make(map[string]entry, len(files))
	seen := make(map[string]struct{}, len(files))

	for _, file := range files {
		name := file.Name()
		if file.IsDir() || strings.HasPrefix(name, ".") || filepath.Ext(name) != descriptorExt {
			continue
		}

		id := strings.TrimSuffix(name, descriptorExt)
		path := filepath.Join(a.dir, name)
		seen[path] = struct{}{}

		desc, err := readDescriptor(file, path)
		if err != nil {
			a.logInvalid(path, err)

			if previous, ok := prev[id]; ok && isRunning(previous.pid, previous.startedAt) {
				current[id] = previous
			}

			continue
		}

		if desc.Labels[a.labelKey] != a.labelValue {
			delete(a.invalid, path)

			continue
		}

		if !isRunning(desc.PID, desc.StartedAt) {
			// Keep a failed removal in a.invalid, so it is logged once per file version.
			a.removeStale(path, desc.file, desc.PID)

			continue
		}

		delete(a.invalid, path)

		current[id] = entry{workload: desc.toWorkload(id, a.hostname), pid: desc.PID, startedAt: desc.StartedAt}
	}

	// Forget invalid files that no longer exist
	for path := range a.invalid {
		if _, ok := seen[path]; !ok {
			delete(a.invalid, path)
		}
	}

	return current, nil
}

// readDescriptor reads and parses a descriptor directory entry.
// Only regular files are accepted; symlinks, FIFOs, sockets and devices are rejected
// so they cannot block the scan or point it at arbitrary files.
func readDescriptor(file fs.DirEntry, path string) (*descriptor, error) {
	if !file.Type().IsRegular() {
		return nil, errors.New("not a regular file")
	}

	return readDescriptorFile(path)
}

// readDescriptorFile reads and parses a descriptor file of at most maxDescriptorSize bytes.
// The file is opened non-blocking and re-checked on the open handle, so an entry replaced
// by a FIFO or device after the directory listing is rejected instead of blocking the scan.
func readDescriptorFile(path string) (*descriptor, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if !info.Mode().IsRegular() {
		return nil, errors.New("not a regular file")
	}

	data, err := io.ReadAll(io.LimitReader(f, maxDescriptorSize+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	if len(data) > maxDescriptorSize {
		return nil, fmt.Errorf("file exceeds %d bytes", maxDescriptorSize)
	}

	desc, err := parseDescriptor(data)
	if err != nil {
		return nil, err
	}

	desc.file = info

	return desc, nil
}

// removeStale deletes the descriptor of a process that is gone, unless the adapter keeps
// stale descriptors. Only descriptors the adapter would list reach here.
//
// It removes only the file it read: writers replace descriptors atomically (a new file
// renamed into place), so a replacement written since the scan is a different file and
// is left alone. The window between this check and the unlink remains.
func (a *adapter) removeStale(path string, readInfo os.FileInfo, pid int) {
	if a.keepStale || readInfo == nil {
		return
	}

	current, err := os.Lstat(path)
	if err != nil || !os.SameFile(readInfo, current) || !current.ModTime().Equal(readInfo.ModTime()) {
		return
	}

	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		a.logInvalid(path, fmt.Errorf("process %d is gone, but its descriptor can't be removed: %w", pid, err))

		return
	}

	logger.Info("removed stale workload descriptor", "file", path, "pid", pid)
}

// logInvalid logs an invalid descriptor once per file version.
// The caller must hold a.mu.
func (a *adapter) logInvalid(path string, err error) {
	var state fileState
	if info, statErr := os.Stat(path); statErr == nil {
		state = fileState{modTime: info.ModTime(), size: info.Size()}
	}

	if last, ok := a.invalid[path]; ok && last.size == state.size && last.modTime.Equal(state.modTime) {
		return
	}

	a.invalid[path] = state

	logger.Warn("skipping invalid workload descriptor", "file", path, "error", err)
}

// WatchEvents polls the descriptor directory and sends workload events to the channel.
// It diffs each scan against the previous one, starting from the last ListWorkloads result.
//
//nolint:wrapcheck
func (a *adapter) WatchEvents(ctx context.Context, eventChan chan<- *types.RuntimeEvent) error {
	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			for _, event := range a.poll() {
				select {
				case eventChan <- event:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}
}

// poll scans the directory once and returns the events since the previous scan.
// On a scan error the previous state is kept, so a transient error does not delete every workload.
func (a *adapter) poll() []*types.RuntimeEvent {
	a.mu.Lock()
	defer a.mu.Unlock()

	current, err := a.scan(a.known)
	if err != nil {
		logger.Error("failed to scan descriptor directory", "error", err)

		return nil
	}

	events := diff(a.known, current)
	a.known = current

	return events
}

// diff returns the events that turn prev into current.
// A workload counts as modified when its content or its PID changed, so a restarted process is re-resolved.
func diff(prev, current map[string]entry) []*types.RuntimeEvent {
	var events []*types.RuntimeEvent

	for id, cur := range current {
		old, existed := prev[id]

		switch {
		case !existed:
			events = append(events, &types.RuntimeEvent{Type: types.RuntimeEventTypeAdded, Workload: cur.workload.DeepCopy()})
		case old.pid != cur.pid || !proto.Equal(old.workload, cur.workload):
			events = append(events, &types.RuntimeEvent{Type: types.RuntimeEventTypeModified, Workload: cur.workload.DeepCopy()})
		}
	}

	for id, old := range prev {
		if _, exists := current[id]; !exists {
			events = append(events, &types.RuntimeEvent{Type: types.RuntimeEventTypeDeleted, Workload: old.workload.DeepCopy()})
		}
	}

	return events
}
