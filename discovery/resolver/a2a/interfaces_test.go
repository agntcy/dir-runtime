// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package a2a

import (
	"encoding/json"
	"testing"

	runtimev1 "github.com/agntcy/dir/api/runtime/v1"
)

// cardWithInterfaces builds a card the way the resolver receives one, by decoding JSON,
// so the tests exercise the same any-typed shapes that come off the wire.
func cardWithInterfaces(t *testing.T, interfacesJSON string) map[string]any {
	t.Helper()

	var card map[string]any
	if err := json.Unmarshal([]byte(`{"name":"copilot","supportedInterfaces":`+interfacesJSON+`}`), &card); err != nil {
		t.Fatalf("bad test fixture: %v", err)
	}

	return card
}

func TestCardLocators(t *testing.T) {
	tests := []struct {
		name       string
		interfaces string
		want       []*runtimev1.WorkloadLocator
	}{
		{
			name:       "SLIMRPC node form becomes a slim locator",
			interfaces: `[{"url":"slim://node:47357/agntcy/shadi/copilot-a2a","protocolBinding":"SLIMRPC"}]`,
			want: []*runtimev1.WorkloadLocator{
				{Protocol: "slim", Url: "slim://node:47357/agntcy/shadi/copilot-a2a"},
			},
		},
		{
			name:       "DID-derived name is kept verbatim",
			interfaces: `[{"url":"slim://node:47357/agntcy/shadi/zKey-a2a","protocolBinding":"SLIMRPC"}]`,
			want: []*runtimev1.WorkloadLocator{
				{Protocol: "slim", Url: "slim://node:47357/agntcy/shadi/zKey-a2a"},
			},
		},
		{
			name:       "hostless slimrpc name is dropped",
			interfaces: `[{"url":"slimrpc://agntcy/shadi/secops-a","protocolBinding":"SLIMRPC"}]`,
			want:       nil,
		},
		{
			name:       "slim URL without a port is dropped",
			interfaces: `[{"url":"slim://agntcy/shadi/secops-a","protocolBinding":"SLIMRPC"}]`,
			want:       nil,
		},
		{
			name:       "bare host:port gains the scheme AgentBridge assumes",
			interfaces: `[{"url":"127.0.0.1:50051","protocolBinding":"GRPC"}]`,
			want:       []*runtimev1.WorkloadLocator{{Protocol: "grpc", Url: "http://127.0.0.1:50051"}},
		},
		{
			name:       "JSONRPC keeps its URL",
			interfaces: `[{"url":"http://127.0.0.1:8080","protocolBinding":"JSONRPC"}]`,
			want:       []*runtimev1.WorkloadLocator{{Protocol: "jsonrpc", Url: "http://127.0.0.1:8080"}},
		},
		{
			name:       "HTTP+JSON keeps its binding name and https scheme",
			interfaces: `[{"url":"https://0.0.0.0:8443","protocolBinding":"HTTP+JSON"}]`,
			want:       []*runtimev1.WorkloadLocator{{Protocol: "http+json", Url: "https://0.0.0.0:8443"}},
		},
		{
			name:       "an interface without a binding is dropped",
			interfaces: `[{"url":"http://127.0.0.1:8080"}]`,
			want:       nil,
		},
		{
			name:       "an interface without a URL is dropped",
			interfaces: `[{"protocolBinding":"GRPC"}]`,
			want:       nil,
		},
		{
			name:       "a card with no interfaces yields nothing",
			interfaces: `[]`,
			want:       nil,
		},
		{
			name:       "entries of the wrong shape are skipped, not fatal",
			interfaces: `["nonsense",{"url":"http://127.0.0.1:8080","protocolBinding":"JSONRPC"}]`,
			want:       []*runtimev1.WorkloadLocator{{Protocol: "jsonrpc", Url: "http://127.0.0.1:8080"}},
		},
		{
			name:       "a SLIM and a unicast interface both promote",
			interfaces: `[{"url":"slim://node:47357/agntcy/shadi/copilot-a2a","protocolBinding":"SLIMRPC"},{"url":"https://127.0.0.1:28443","protocolBinding":"HTTP+JSON"}]`,
			want: []*runtimev1.WorkloadLocator{
				{Protocol: "slim", Url: "slim://node:47357/agntcy/shadi/copilot-a2a"},
				{Protocol: "http+json", Url: "https://127.0.0.1:28443"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cardLocators(cardWithInterfaces(t, tt.interfaces))

			if len(got) != len(tt.want) {
				t.Fatalf("cardLocators() = %v, want %v", got, tt.want)
			}

			for i := range tt.want {
				if got[i].GetProtocol() != tt.want[i].GetProtocol() || got[i].GetUrl() != tt.want[i].GetUrl() {
					t.Fatalf("cardLocators()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestCardLocators_RealAgentBridgeCard uses a card captured verbatim from a running
// AgentBridge, so the parsing stays honest about the shape actually served rather than the
// shape these tests assume.
func TestCardLocators_RealAgentBridgeCard(t *testing.T) {
	const served = `{
	  "name": "debugger",
	  "description": "agentbridge adapter for 'debugger'.",
	  "version": "1.0.0",
	  "supportedInterfaces": [
	    {
	      "url": "slim://192.168.1.125:46357/agntcy/shadi/z6MkvYRE3byJ3YLPhpBirUAEpvbgNUp7Cc96KAYwCHkKn1gy-a2a",
	      "protocolBinding": "SLIMRPC",
	      "protocolVersion": "1.0"
	    }
	  ],
	  "capabilities": {"streaming": true, "pushNotifications": false},
	  "defaultInputModes": ["text/plain"],
	  "defaultOutputModes": ["text/plain"],
	  "skills": []
	}`

	var card map[string]any
	if err := json.Unmarshal([]byte(served), &card); err != nil {
		t.Fatalf("bad fixture: %v", err)
	}

	got := cardLocators(card)
	if len(got) != 1 {
		t.Fatalf("cardLocators() = %v, want exactly one locator", got)
	}

	if got[0].GetProtocol() != "slim" {
		t.Errorf("protocol = %q, want %q", got[0].GetProtocol(), "slim")
	}

	want := "slim://192.168.1.125:46357/agntcy/shadi/z6MkvYRE3byJ3YLPhpBirUAEpvbgNUp7Cc96KAYwCHkKn1gy-a2a"
	if got[0].GetUrl() != want {
		t.Errorf("url = %q, want %q", got[0].GetUrl(), want)
	}
}

func TestCardLocators_MissingField(t *testing.T) {
	if got := cardLocators(map[string]any{"name": "copilot"}); got != nil {
		t.Errorf("cardLocators() = %v, want nil for a card with no supportedInterfaces", got)
	}
}

func TestApply_PromotesCardInterfacesIntoLocators(t *testing.T) {
	r := NewResolver(Config{})

	workload := &runtimev1.Workload{
		Id:       "test-workload",
		Locators: []*runtimev1.WorkloadLocator{{Protocol: "tcp", Url: "tcp://10.244.0.9:9999"}},
	}

	card := cardWithInterfaces(t, `[{"url":"slim://node:47357/agntcy/shadi/copilot-a2a","protocolBinding":"SLIMRPC"}]`)

	if err := r.Apply(t.Context(), workload, card); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	want := []*runtimev1.WorkloadLocator{
		{Protocol: "slim", Url: "slim://node:47357/agntcy/shadi/copilot-a2a"},
		{Protocol: "tcp", Url: "tcp://10.244.0.9:9999"},
	}

	got := workload.GetLocators()
	if len(got) != len(want) {
		t.Fatalf("locators = %v, want %v", got, want)
	}

	for i := range want {
		if got[i].GetProtocol() != want[i].GetProtocol() || got[i].GetUrl() != want[i].GetUrl() {
			t.Fatalf("locators[%d] = %v, want %v", i, got[i], want[i])
		}
	}

	// The observed TCP locator has to survive: CanResolve gates on an HTTP-capable origin,
	// so losing it would make the workload unresolvable from the next cycle on.
	labelled := NewResolver(Config{LabelKey: DefaultLabelKey, LabelValue: DefaultLabelValue})
	if !labelled.CanResolve(&runtimev1.Workload{
		Labels:   map[string]string{DefaultLabelKey: DefaultLabelValue},
		Locators: got,
	}) {
		t.Error("workload became unresolvable after promotion")
	}
}

func TestApply_IsIdempotent(t *testing.T) {
	r := NewResolver(Config{})

	workload := &runtimev1.Workload{
		Id:       "test-workload",
		Locators: []*runtimev1.WorkloadLocator{{Protocol: "tcp", Url: "tcp://10.244.0.9:9999"}},
	}

	card := cardWithInterfaces(t, `[{"url":"slim://node:47357/agntcy/shadi/copilot-a2a","protocolBinding":"SLIMRPC"}]`)

	if err := r.Apply(t.Context(), workload, card); err != nil {
		t.Fatalf("first Apply() error = %v", err)
	}

	first := workload.GetLocators()

	if err := r.Apply(t.Context(), workload, card); err != nil {
		t.Fatalf("second Apply() error = %v", err)
	}

	second := workload.GetLocators()
	if len(first) != len(second) {
		t.Fatalf("re-resolving changed the locator list: %v then %v", first, second)
	}

	for i := range first {
		if first[i].GetUrl() != second[i].GetUrl() {
			t.Fatalf("re-resolving reordered the locator list: %v then %v", first, second)
		}
	}
}
