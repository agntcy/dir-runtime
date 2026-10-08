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
