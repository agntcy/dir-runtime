// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sql

import (
	"context"
	"path/filepath"
	"testing"
)

// legacyPayload is a workload as stored before agntcy.dir.runtime.v1.Workload
// dropped its deprecated addresses and ports fields.
const legacyPayload = `{"id":"legacy","name":"agent","hostname":"host-1",` +
	`"addresses":["10.0.0.1"],"ports":["8080"],"labels":{"app":"agent"}}`

func TestToRuntimeWorkloadIgnoresRemovedFields(t *testing.T) {
	record := &workloadRecord{ID: "legacy", Payload: []byte(legacyPayload)}

	workload, err := record.ToRuntimeWorkload()
	if err != nil {
		t.Fatalf("ToRuntimeWorkload() error = %v", err)
	}

	if workload.GetId() != "legacy" || workload.GetName() != "agent" || workload.GetHostname() != "host-1" {
		t.Errorf("ToRuntimeWorkload() = %v, want id, name and hostname kept", workload)
	}

	if got := workload.GetLabels()["app"]; got != "agent" {
		t.Errorf("label app = %q, want %q", got, "agent")
	}
}

func TestStoreReadsLegacyPayload(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t, filepath.Join(t.TempDir(), "workloads.db"))

	s, err := New(db)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := db.Create(&workloadRecord{ID: "legacy", Payload: []byte(legacyPayload)}).Error; err != nil {
		t.Fatalf("insert legacy record: %v", err)
	}

	workload, err := s.GetWorkload(ctx, "legacy")
	if err != nil {
		t.Fatalf("GetWorkload() error = %v", err)
	}

	if workload.GetName() != "agent" {
		t.Errorf("GetWorkload() name = %q, want %q", workload.GetName(), "agent")
	}

	workloads, err := s.ListWorkloads(ctx)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}

	if len(workloads) != 1 || workloads[0].GetId() != "legacy" {
		t.Errorf("ListWorkloads() = %v, want the legacy workload", workloads)
	}
}
