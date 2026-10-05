// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package process

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/agntcy/dir-runtime/discovery/types"
	"github.com/agntcy/dir-runtime/utils"
	runtimev1 "github.com/agntcy/dir/api/runtime/v1"
	"google.golang.org/protobuf/proto"
)

const (
	descriptorExt = ".json"
	dirPerm       = 0o700
	fallbackHost  = "localhost"
)

var logger = utils.NewLogger("runtime", "process")

// entry is a discovered workload together with the PID it was discovered for.
type entry struct {
	workload *runtimev1.Workload
	pid      int
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
	hostname     string

	mu      sync.Mutex
	known   map[string]entry     // last scan result, used as the watch baseline
	invalid map[string]fileState // invalid descriptor files already logged
}

// NewAdapter creates a new process adapter.
func NewAdapter(cfg Config) (types.RuntimeAdapter, error) {
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

		desc, err := readDescriptor(path)
		if err != nil {
			a.logInvalid(path, err)

			if previous, ok := prev[id]; ok && isAlive(previous.pid) {
				current[id] = previous
			}

			continue
		}

		delete(a.invalid, path)

		if desc.Labels[a.labelKey] != a.labelValue || !isAlive(desc.PID) {
			continue
		}

		current[id] = entry{workload: desc.toWorkload(id, a.hostname), pid: desc.PID}
	}

	// Forget invalid files that no longer exist
	for path := range a.invalid {
		if _, ok := seen[path]; !ok {
			delete(a.invalid, path)
		}
	}

	return current, nil
}

// readDescriptor reads and parses a descriptor file.
func readDescriptor(path string) (*descriptor, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return parseDescriptor(data)
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
