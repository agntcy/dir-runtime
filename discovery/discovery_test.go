// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package discovery

import (
	"context"
	"path/filepath"
	"sort"
	"testing"

	"github.com/agntcy/dir-runtime/discovery/types"
	"github.com/agntcy/dir-runtime/store/sql"
	storetypes "github.com/agntcy/dir-runtime/store/types"
	"github.com/agntcy/dir-runtime/utils"
	runtimev1 "github.com/agntcy/dir/api/runtime/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAdapter is a runtime adapter that lists a fixed set of workloads.
type fakeAdapter struct {
	runtimeType types.RuntimeType
	workloads   []*runtimev1.Workload
}

func (f *fakeAdapter) Type() types.RuntimeType { return f.runtimeType }

func (f *fakeAdapter) Close() error { return nil }

func (f *fakeAdapter) ListWorkloads(context.Context) ([]*runtimev1.Workload, error) {
	return f.workloads, nil
}

func (f *fakeAdapter) WatchEvents(ctx context.Context, _ chan<- *types.RuntimeEvent) error {
	<-ctx.Done()

	return ctx.Err() //nolint:wrapcheck
}

func newTestStore(t *testing.T) storetypes.Store {
	t.Helper()

	db, err := sql.NewSqlite(sql.Config{Path: filepath.Join(t.TempDir(), "workloads.db")})
	require.NoError(t, err)

	s, err := sql.New(db)
	require.NoError(t, err)

	t.Cleanup(func() { _ = s.Close() })

	return s
}

func newTestRunner(t *testing.T, s storetypes.Store, runtimeType types.RuntimeType, instanceID string, live ...*runtimev1.Workload) *runner {
	t.Helper()

	return &runner{
		adapter:    &fakeAdapter{runtimeType: runtimeType, workloads: live},
		store:      s,
		instanceID: instanceID,
		logger:     utils.NewLogger("test", "discovery"),
	}
}

func workload(id string, runtimeType types.RuntimeType, instanceID string) *runtimev1.Workload {
	w := &runtimev1.Workload{Id: id, Name: id, Runtime: string(runtimeType), Annotations: map[string]string{}}
	if instanceID != "" {
		w.Annotations[InstanceAnnotation] = instanceID
	}

	return w
}

func seed(t *testing.T, s storetypes.Store, workloads ...*runtimev1.Workload) {
	t.Helper()

	for _, w := range workloads {
		require.NoError(t, s.RegisterWorkload(context.Background(), w))
	}
}

func reconcile(t *testing.T, r *runner) {
	t.Helper()

	queue := make(chan *runtimev1.Workload, 100)
	require.NoError(t, r.reconcile(context.Background(), queue))
}

func storedIDs(t *testing.T, s storetypes.Store) []string {
	t.Helper()

	ids, err := s.ListWorkloadIDs(context.Background())
	require.NoError(t, err)

	result := make([]string, 0, len(ids))
	for id := range ids {
		result = append(result, id)
	}

	sort.Strings(result)

	return result
}

const (
	docker  types.RuntimeType = "docker"
	process types.RuntimeType = "process"
)

func TestReconcileKeepsOtherRuntimeWorkloads(t *testing.T) {
	s := newTestStore(t)
	seed(t, s,
		workload("container", docker, ""),
		workload("stale-agent", process, ""),
	)

	reconcile(t, newTestRunner(t, s, process, "", workload("live-agent", process, "")))

	assert.Equal(t, []string{"container", "live-agent"}, storedIDs(t, s))
}

func TestReconcileKeepsOtherInstanceWorkloads(t *testing.T) {
	s := newTestStore(t)
	seed(t, s,
		workload("container-b", docker, "host-b"),
		workload("stale-a", docker, "host-a"),
		workload("unscoped", docker, ""),
	)

	reconcile(t, newTestRunner(t, s, docker, "host-a", workload("container-a", docker, "")))

	assert.Equal(t, []string{"container-a", "container-b", "unscoped"}, storedIDs(t, s))
}

func TestReconcileWithoutInstanceIDKeepsScopedWorkloads(t *testing.T) {
	s := newTestStore(t)
	seed(t, s,
		workload("stale", process, ""),
		workload("scoped", process, "host-b"),
	)

	reconcile(t, newTestRunner(t, s, process, ""))

	assert.Equal(t, []string{"scoped"}, storedIDs(t, s))
}

func TestReconcileStampsInstance(t *testing.T) {
	s := newTestStore(t)

	reconcile(t, newTestRunner(t, s, process, "host-a", workload("live-agent", process, "")))

	stored, err := s.GetWorkload(context.Background(), "live-agent")
	require.NoError(t, err)
	assert.Equal(t, "host-a", stored.GetAnnotations()[InstanceAnnotation])
}

func TestReconcileWithoutInstanceIDDoesNotStamp(t *testing.T) {
	s := newTestStore(t)

	reconcile(t, newTestRunner(t, s, process, "", workload("live-agent", process, "")))

	stored, err := s.GetWorkload(context.Background(), "live-agent")
	require.NoError(t, err)
	assert.NotContains(t, stored.GetAnnotations(), InstanceAnnotation)
}

func TestHandleRuntimeEventStampsInstance(t *testing.T) {
	s := newTestStore(t)
	r := newTestRunner(t, s, process, "host-a")
	queue := make(chan *runtimev1.Workload, 1)

	added := &runtimev1.Workload{Id: "agent", Runtime: string(process)}
	r.handleRuntimeEvent(context.Background(), queue, &types.RuntimeEvent{Type: types.RuntimeEventTypeAdded, Workload: added})

	stored, err := s.GetWorkload(context.Background(), "agent")
	require.NoError(t, err)
	assert.Equal(t, "host-a", stored.GetAnnotations()[InstanceAnnotation])
	assert.Equal(t, "host-a", (<-queue).GetAnnotations()[InstanceAnnotation], "resolvers must see the stamped workload")
}

func TestReconcileOverridesRuntimeSuppliedInstanceAnnotation(t *testing.T) {
	for _, instanceID := range []string{"", "host-a"} {
		t.Run("instance="+instanceID, func(t *testing.T) {
			s := newTestStore(t)
			spoofed := workload("agent", process, "someone-else")

			reconcile(t, newTestRunner(t, s, process, instanceID, spoofed))

			stored, err := s.GetWorkload(context.Background(), "agent")
			require.NoError(t, err)

			got, ok := stored.GetAnnotations()[InstanceAnnotation]
			if instanceID == "" {
				assert.False(t, ok, "reserved annotation must be removed without an instance ID, got %q", got)
			} else {
				assert.Equal(t, instanceID, got)
			}
		})
	}
}

func TestReconcileRemovesStaleWorkloadThatSuppliedInstanceAnnotation(t *testing.T) {
	s := newTestStore(t)

	// First run: the runtime reports a workload that carries the reserved annotation itself.
	reconcile(t, newTestRunner(t, s, process, "", workload("agent", process, "someone-else")))

	// Second run: the workload is gone, so the default instance must remove it.
	reconcile(t, newTestRunner(t, s, process, ""))

	assert.Empty(t, storedIDs(t, s))
}

func TestHandleRuntimeEventOverridesRuntimeSuppliedInstanceAnnotation(t *testing.T) {
	s := newTestStore(t)
	r := newTestRunner(t, s, process, "")
	queue := make(chan *runtimev1.Workload, 1)

	added := workload("agent", process, "someone-else")
	r.handleRuntimeEvent(context.Background(), queue, &types.RuntimeEvent{Type: types.RuntimeEventTypeAdded, Workload: added})

	stored, err := s.GetWorkload(context.Background(), "agent")
	require.NoError(t, err)
	assert.NotContains(t, stored.GetAnnotations(), InstanceAnnotation)
}
