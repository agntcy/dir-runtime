// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:errcheck
package a2a

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	runtimev1 "github.com/agntcy/dir/api/runtime/v1"
)

func TestNewResolver(t *testing.T) {
	cfg := Config{
		Enabled:    true,
		Timeout:    5 * time.Second,
		LabelKey:   "test-key",
		LabelValue: "test-value",
		Paths:      []string{"/.well-known/agent-card.json"},
	}

	res := NewResolver(cfg)
	if res == nil {
		t.Fatal("NewResolver returned nil")
	}

	r, ok := res.(*resolver)
	if !ok {
		t.Fatal("NewResolver did not return *resolver")
	}

	if r.timeout != cfg.Timeout {
		t.Errorf("timeout = %v, want %v", r.timeout, cfg.Timeout)
	}

	if r.labelKey != cfg.LabelKey {
		t.Errorf("labelKey = %v, want %v", r.labelKey, cfg.LabelKey)
	}

	if r.labelValue != cfg.LabelValue {
		t.Errorf("labelValue = %v, want %v", r.labelValue, cfg.LabelValue)
	}
}

func TestResolver_Name(t *testing.T) {
	r := NewResolver(Config{})
	if r.Name() != ResolverType {
		t.Errorf("Name() = %v, want %v", r.Name(), ResolverType)
	}
}

func TestResolver_CanResolve(t *testing.T) {
	r := NewResolver(Config{
		LabelKey:   "org.agntcy/agent-type",
		LabelValue: "a2a",
	})

	tests := []struct {
		name     string
		workload *runtimev1.Workload
		want     bool
	}{
		{
			name: "can resolve workload with matching label and tcp locator",
			workload: &runtimev1.Workload{
				Labels:   map[string]string{"org.agntcy/agent-type": "a2a"},
				Locators: []*runtimev1.WorkloadLocator{{Protocol: "tcp", Url: "tcp://localhost:8080"}},
			},
			want: true,
		},
		{
			name: "can resolve workload with http locator",
			workload: &runtimev1.Workload{
				Labels:   map[string]string{"org.agntcy/agent-type": "a2a"},
				Locators: []*runtimev1.WorkloadLocator{{Protocol: "http", Url: "http://localhost:8080"}},
			},
			want: true,
		},
		{
			name: "cannot resolve workload without label",
			workload: &runtimev1.Workload{
				Labels:   map[string]string{},
				Locators: []*runtimev1.WorkloadLocator{{Protocol: "tcp", Url: "tcp://localhost:8080"}},
			},
			want: false,
		},
		{
			name: "cannot resolve workload with wrong label value",
			workload: &runtimev1.Workload{
				Labels:   map[string]string{"org.agntcy/agent-type": "other"},
				Locators: []*runtimev1.WorkloadLocator{{Protocol: "tcp", Url: "tcp://localhost:8080"}},
			},
			want: false,
		},
		{
			name: "cannot resolve workload without locators",
			workload: &runtimev1.Workload{
				Labels: map[string]string{"org.agntcy/agent-type": "a2a"},
			},
			want: false,
		},
		{
			name: "cannot resolve workload with only non-HTTP locators",
			workload: &runtimev1.Workload{
				Labels:   map[string]string{"org.agntcy/agent-type": "a2a"},
				Locators: []*runtimev1.WorkloadLocator{{Protocol: "slim", Url: "slim://org/namespace/agent"}},
			},
			want: false,
		},
		{
			name: "case insensitive label value match",
			workload: &runtimev1.Workload{
				Labels:   map[string]string{"org.agntcy/agent-type": "A2A"},
				Locators: []*runtimev1.WorkloadLocator{{Protocol: "tcp", Url: "tcp://localhost:8080"}},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.CanResolve(tt.workload); got != tt.want {
				t.Errorf("CanResolve() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolver_Resolve(t *testing.T) {
	// Create a test server that returns a valid agent card
	agentCard := map[string]any{
		"name":        "test-agent",
		"version":     "1.0.0",
		"description": "A test agent",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/agent-card.json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(agentCard)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Extract host and port from test server
	// server.URL is like "http://127.0.0.1:12345"
	r := NewResolver(Config{
		Timeout:    5 * time.Second,
		LabelKey:   "test",
		LabelValue: "true",
		Paths:      []string{"/.well-known/agent-card.json"},
	})

	t.Run("resolves successfully from valid endpoint", func(t *testing.T) {
		// We need to use the test server's actual address
		// Parse the URL to get host:port
		workload := &runtimev1.Workload{
			Id:       "test-workload",
			Labels:   map[string]string{"test": "true"},
			Locators: []*runtimev1.WorkloadLocator{{Protocol: "tcp", Url: "tcp://" + server.Listener.Addr().String()}},
		}

		ctx := context.Background()

		result, err := r.Resolve(ctx, workload)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}

		resultMap, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("Result is not map[string]any: %T", result)
		}

		if resultMap["name"] != "test-agent" {
			t.Errorf("Result name = %v, want 'test-agent'", resultMap["name"])
		}
	})

	t.Run("returns error when no endpoints reachable", func(t *testing.T) {
		workload := &runtimev1.Workload{
			Id:       "test-workload",
			Labels:   map[string]string{"test": "true"},
			Locators: []*runtimev1.WorkloadLocator{{Protocol: "tcp", Url: "tcp://127.0.0.1:99999"}}, // Invalid port
		}

		ctx := context.Background()

		_, err := r.Resolve(ctx, workload)
		if err == nil {
			t.Error("Resolve() should return error for unreachable endpoint")
		}
	})

	t.Run("resolves from http locator and skips non-HTTP locators", func(t *testing.T) {
		workload := &runtimev1.Workload{
			Id:     "test-workload",
			Labels: map[string]string{"test": "true"},
			Locators: []*runtimev1.WorkloadLocator{
				{Protocol: "slim", Url: "slim://org/namespace/agent"},
				{Protocol: "http", Url: server.URL},
			},
		}

		result, err := r.Resolve(context.Background(), workload)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}

		if resultMap, _ := result.(map[string]any); resultMap["name"] != "test-agent" {
			t.Errorf("Result = %v, want agent card", result)
		}
	})

	t.Run("https locator is not probed over plain http", func(t *testing.T) {
		workload := &runtimev1.Workload{
			Id:       "test-workload",
			Labels:   map[string]string{"test": "true"},
			Locators: []*runtimev1.WorkloadLocator{{Protocol: "https", Url: "https://" + server.Listener.Addr().String()}},
		}

		if _, err := r.Resolve(context.Background(), workload); err == nil {
			t.Error("Resolve() should fail: the test server only speaks plain http")
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		workload := &runtimev1.Workload{
			Id:       "test-workload",
			Labels:   map[string]string{"test": "true"},
			Locators: []*runtimev1.WorkloadLocator{{Protocol: "tcp", Url: "tcp://127.0.0.1:8080"}},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := r.Resolve(ctx, workload)
		if err == nil {
			t.Error("Resolve() should return error for cancelled context")
		}
	})
}

func TestResolver_Apply(t *testing.T) {
	r := NewResolver(Config{})

	t.Run("applies result to workload", func(t *testing.T) {
		workload := &runtimev1.Workload{
			Id: "test-workload",
		}

		result := map[string]any{
			"name":    "test-agent",
			"version": "1.0.0",
		}

		ctx := context.Background()

		err := r.Apply(ctx, workload, result)
		if err != nil {
			t.Fatalf("Apply() error = %v", err)
		}

		if workload.GetServices() == nil {
			t.Fatal("Services is nil after Apply")
		}

		if workload.GetServices().GetA2A() == nil {
			t.Fatal("Services.A2A is nil after Apply")
		}

		fields := workload.GetServices().GetA2A().GetFields()
		if fields["name"].GetStringValue() != "test-agent" {
			t.Errorf("A2A name = %v, want 'test-agent'", fields["name"].GetStringValue())
		}
	})

	t.Run("preserves existing services", func(t *testing.T) {
		workload := &runtimev1.Workload{
			Id:       "test-workload",
			Services: &runtimev1.WorkloadServices{},
		}

		result := map[string]any{
			"name": "test-agent",
		}

		ctx := context.Background()

		err := r.Apply(ctx, workload, result)
		if err != nil {
			t.Fatalf("Apply() error = %v", err)
		}

		if workload.GetServices().GetA2A() == nil {
			t.Fatal("Services.A2A is nil after Apply")
		}
	})
}
