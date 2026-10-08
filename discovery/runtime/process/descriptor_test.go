// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package process

import (
	"testing"

	runtimev1 "github.com/agntcy/dir/api/runtime/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDescriptor(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr string
	}{
		{name: "valid", data: `{"name":"agent","pid":1,"ports":["9999"]}`},
		{name: "numeric ports", data: `{"name":"agent","pid":1,"ports":[9999, 8080]}`},
		{name: "unknown fields ignored", data: `{"name":"agent","pid":1,"ports":["9999"],"extra":true}`},
		{name: "invalid JSON", data: `{"name":`, wantErr: "invalid JSON"},
		{name: "non-numeric port", data: `{"name":"agent","pid":1,"ports":["http"]}`, wantErr: "invalid JSON"},
		{name: "missing name", data: `{"pid":1,"ports":["9999"]}`, wantErr: "name is required"},
		{name: "zero pid", data: `{"name":"agent","pid":0,"ports":["9999"]}`, wantErr: "pid must be positive"},
		{name: "negative pid", data: `{"name":"agent","pid":-5,"ports":["9999"]}`, wantErr: "pid must be positive"},
		{name: "no ports", data: `{"name":"agent","pid":1}`},
		{name: "empty ports", data: `{"name":"agent","pid":1,"ports":[]}`},
		{name: "port zero", data: `{"name":"agent","pid":1,"ports":["0"]}`, wantErr: "invalid port"},
		{name: "port too large", data: `{"name":"agent","pid":1,"ports":[70000]}`, wantErr: "invalid port"},
		{name: "fractional port", data: `{"name":"agent","pid":1,"ports":[99.5]}`, wantErr: "invalid port"},
		{name: "locators", data: `{"name":"agent","pid":1,"locators":[{"protocol":"slim","url":"slim://org/ns/agent"}]}`},
		{name: "locator without protocol", data: `{"name":"agent","pid":1,"locators":[{"url":"slim://org/ns/agent"}]}`, wantErr: "locator 0: protocol is required"},
		{name: "locator without url", data: `{"name":"agent","pid":1,"locators":[{"protocol":"slim"}]}`, wantErr: "locator 0: url is required"},
		{name: "locator url without scheme", data: `{"name":"agent","pid":1,"locators":[{"protocol":"slim","url":"org/ns/agent"}]}`, wantErr: "locator 0: url must include a scheme"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			desc, err := parseDescriptor([]byte(test.data))
			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
				assert.Nil(t, desc)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, "agent", desc.Name)
		})
	}
}

func TestDescriptorToWorkload(t *testing.T) {
	t.Run("applies adapter-set fields and defaults", func(t *testing.T) {
		desc, err := parseDescriptor([]byte(`{"name":"agent","pid":42,"ports":[9999]}`))
		require.NoError(t, err)

		workload := desc.toWorkload("agent-1", "my-host")

		assert.Equal(t, "agent-1", workload.GetId())
		assert.Equal(t, "agent", workload.GetName())
		assert.Equal(t, "my-host", workload.GetHostname())
		assert.Equal(t, "process", workload.GetRuntime())
		assert.Equal(t, "process", workload.GetType())
		assert.Equal(t, []*runtimev1.WorkloadLocator{{Protocol: "tcp", Url: "tcp://127.0.0.1:9999"}}, workload.GetLocators())
		assert.Equal(t, []string{"host"}, workload.GetIsolationGroups())
		assert.NotNil(t, workload.GetLabels())
		assert.NotNil(t, workload.GetAnnotations())
	})

	t.Run("descriptor locators are copied", func(t *testing.T) {
		desc, err := parseDescriptor([]byte(`{
			"name": "agent",
			"pid": 42,
			"locators": [{"protocol": "slim", "url": "slim://org/ns/agent"}]
		}`))
		require.NoError(t, err)

		workload := desc.toWorkload("agent-1", "my-host")

		assert.Equal(t, []*runtimev1.WorkloadLocator{{Protocol: "slim", Url: "slim://org/ns/agent"}}, workload.GetLocators())
	})

	t.Run("no ports and no locators means no locators", func(t *testing.T) {
		desc, err := parseDescriptor([]byte(`{"name":"agent","pid":42,"addresses":["192.168.1.10"]}`))
		require.NoError(t, err)

		assert.Empty(t, desc.toWorkload("agent-1", "my-host").GetLocators())
	})

	t.Run("keeps descriptor values", func(t *testing.T) {
		desc, err := parseDescriptor([]byte(`{
			"name": "agent",
			"pid": 42,
			"labels": {"org.agntcy/discover": "true", "org.agntcy/agent-type": "a2a"},
			"annotations": {"org.agntcy/agent-record": "my-agent:v1.0.0"},
			"addresses": ["192.168.1.10"],
			"ports": ["9999", "8080"],
			"locators": [{"protocol": "slim", "url": "slim://org/ns/agent"}]
		}`))
		require.NoError(t, err)

		workload := desc.toWorkload("agent-1", "my-host")

		assert.Equal(t, map[string]string{"org.agntcy/discover": "true", "org.agntcy/agent-type": "a2a"}, workload.GetLabels())
		assert.Equal(t, map[string]string{"org.agntcy/agent-record": "my-agent:v1.0.0"}, workload.GetAnnotations())
		assert.Equal(t, []*runtimev1.WorkloadLocator{
			{Protocol: "slim", Url: "slim://org/ns/agent"},
			{Protocol: "tcp", Url: "tcp://192.168.1.10:8080"},
			{Protocol: "tcp", Url: "tcp://192.168.1.10:9999"},
		}, workload.GetLocators())
	})
}
