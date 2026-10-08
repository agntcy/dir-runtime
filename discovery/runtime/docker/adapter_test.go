// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"testing"

	runtimev1 "github.com/agntcy/dir/api/runtime/v1"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/stretchr/testify/assert"
)

func tcp(url string) *runtimev1.WorkloadLocator {
	return &runtimev1.WorkloadLocator{Protocol: "tcp", Url: url}
}

func TestContainerToWorkloadLocators(t *testing.T) {
	summary := container.Summary{
		ID:    "abc123",
		Names: []string{"/agent"},
		Ports: []container.PortSummary{{PrivatePort: 9999, PublicPort: 19999}},
		NetworkSettings: &container.NetworkSettingsSummary{
			Networks: map[string]*network.EndpointSettings{"team-a": {}},
		},
	}

	t.Run("network mode uses container name per network and private port", func(t *testing.T) {
		workload := (&adapter{}).containerToWorkload(summary)

		assert.Equal(t, []*runtimev1.WorkloadLocator{tcp("tcp://agent.team-a:9999")}, workload.GetLocators())
		assert.Equal(t, []string{"team-a"}, workload.GetIsolationGroups())
	})

	t.Run("host mode uses the host address and public port", func(t *testing.T) {
		workload := (&adapter{hostMode: true}).containerToWorkload(summary)

		assert.Equal(t, []*runtimev1.WorkloadLocator{tcp("tcp://0.0.0.0:19999")}, workload.GetLocators())
	})
}

func TestInspectToWorkloadLocators(t *testing.T) {
	inspect := container.InspectResponse{
		ID:   "abc123",
		Name: "/agent",
		NetworkSettings: &container.NetworkSettings{
			Networks: map[string]*network.EndpointSettings{"team-a": {}, "team-b": {}},
			Ports: network.PortMap{
				network.MustParsePort("9999/tcp"): {{HostPort: "19999"}},
			},
		},
	}

	workload := (&adapter{}).inspectToWorkload(inspect)

	assert.Equal(t, []*runtimev1.WorkloadLocator{
		tcp("tcp://agent.team-a:9999"),
		tcp("tcp://agent.team-b:9999"),
	}, workload.GetLocators())
}
