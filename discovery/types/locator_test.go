// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"testing"

	runtimev1 "github.com/agntcy/dir/api/runtime/v1"
	"github.com/stretchr/testify/assert"
)

func TestTCPLocators(t *testing.T) {
	tests := []struct {
		name      string
		addresses []string
		ports     []string
		want      []*runtimev1.WorkloadLocator
	}{
		{
			name:      "one locator per address and port, sorted",
			addresses: []string{"svc.team-a.svc", "10-244-0-9.team-a.pod"},
			ports:     []string{"9999", "8080"},
			want: []*runtimev1.WorkloadLocator{
				{Protocol: LocatorProtocolTCP, Url: "tcp://10-244-0-9.team-a.pod:8080"},
				{Protocol: LocatorProtocolTCP, Url: "tcp://10-244-0-9.team-a.pod:9999"},
				{Protocol: LocatorProtocolTCP, Url: "tcp://svc.team-a.svc:8080"},
				{Protocol: LocatorProtocolTCP, Url: "tcp://svc.team-a.svc:9999"},
			},
		},
		{
			name:      "IPv6 address is bracketed",
			addresses: []string{"::1"},
			ports:     []string{"9999"},
			want:      []*runtimev1.WorkloadLocator{{Protocol: LocatorProtocolTCP, Url: "tcp://[::1]:9999"}},
		},
		{
			name:      "duplicates are removed",
			addresses: []string{"localhost", "localhost"},
			ports:     []string{"80", "80"},
			want:      []*runtimev1.WorkloadLocator{{Protocol: LocatorProtocolTCP, Url: "tcp://localhost:80"}},
		},
		{name: "no ports", addresses: []string{"localhost"}, want: nil},
		{name: "no addresses", ports: []string{"80"}, want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, TCPLocators(tt.addresses, tt.ports))
		})
	}
}

func TestMergeLocators(t *testing.T) {
	tests := []struct {
		name  string
		lists [][]*runtimev1.WorkloadLocator
		want  []*runtimev1.WorkloadLocator
	}{
		{
			name: "sorted by protocol then url",
			lists: [][]*runtimev1.WorkloadLocator{
				{{Protocol: LocatorProtocolTCP, Url: "tcp://host:9999"}},
				{
					{Protocol: LocatorProtocolSLIM, Url: "slim://node:47357/agntcy/shadi/b-a2a"},
					{Protocol: LocatorProtocolSLIM, Url: "slim://node:47357/agntcy/shadi/a-a2a"},
				},
			},
			want: []*runtimev1.WorkloadLocator{
				{Protocol: LocatorProtocolSLIM, Url: "slim://node:47357/agntcy/shadi/a-a2a"},
				{Protocol: LocatorProtocolSLIM, Url: "slim://node:47357/agntcy/shadi/b-a2a"},
				{Protocol: LocatorProtocolTCP, Url: "tcp://host:9999"},
			},
		},
		{
			name: "same protocol and url collapses",
			lists: [][]*runtimev1.WorkloadLocator{
				{{Protocol: LocatorProtocolTCP, Url: "tcp://host:9999"}},
				{{Protocol: LocatorProtocolTCP, Url: "tcp://host:9999"}},
			},
			want: []*runtimev1.WorkloadLocator{{Protocol: LocatorProtocolTCP, Url: "tcp://host:9999"}},
		},
		{
			// One endpoint can serve two bindings, and the locator-protocol filter treats
			// those as different ways in, so the URL alone must not be the key.
			name: "same url under two protocols is kept twice",
			lists: [][]*runtimev1.WorkloadLocator{{
				{Protocol: "jsonrpc", Url: "http://host:8080"},
				{Protocol: "http+json", Url: "http://host:8080"},
			}},
			want: []*runtimev1.WorkloadLocator{
				{Protocol: "http+json", Url: "http://host:8080"},
				{Protocol: "jsonrpc", Url: "http://host:8080"},
			},
		},
		{name: "nothing to merge", lists: nil, want: nil},
		{
			name:  "only empty lists",
			lists: [][]*runtimev1.WorkloadLocator{nil, {}},
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, MergeLocators(tt.lists...))
		})
	}
}
