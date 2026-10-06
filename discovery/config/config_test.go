// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"
	"time"

	"github.com/agntcy/dir-runtime/discovery/resolver/a2a"
	resolver "github.com/agntcy/dir-runtime/discovery/resolver/config"
	"github.com/agntcy/dir-runtime/discovery/resolver/oasf"
	runtime "github.com/agntcy/dir-runtime/discovery/runtime/config"
	"github.com/agntcy/dir-runtime/discovery/runtime/docker"
	"github.com/agntcy/dir-runtime/discovery/runtime/k8s"
	"github.com/agntcy/dir-runtime/discovery/runtime/process"
	store "github.com/agntcy/dir-runtime/store/config"
	"github.com/agntcy/dir-runtime/store/crd"
	"github.com/agntcy/dir-runtime/store/etcd"
	"github.com/agntcy/dir-runtime/store/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		Name           string
		EnvVars        map[string]string
		ExpectedConfig *Config
	}{
		{
			Name: "Custom config with Docker runtime and etcd store",
			EnvVars: map[string]string{
				"DISCOVERY_WORKERS":                        "8",
				"DISCOVERY_INSTANCE_ID":                    "custom-instance",
				"DISCOVERY_RUNTIME_TYPE":                   "docker",
				"DISCOVERY_RUNTIME_DOCKER_HOST":            "unix:///custom/docker.sock",
				"DISCOVERY_RUNTIME_DOCKER_LABEL_KEY":       "custom.label/discover",
				"DISCOVERY_RUNTIME_DOCKER_LABEL_VALUE":     "enabled",
				"DISCOVERY_RUNTIME_KUBERNETES_KUBECONFIG":  "path-to-kubeconfig",
				"DISCOVERY_RUNTIME_KUBERNETES_NAMESPACE":   "namespace",
				"DISCOVERY_RUNTIME_KUBERNETES_LABEL_KEY":   "custom.label/key",
				"DISCOVERY_RUNTIME_KUBERNETES_LABEL_VALUE": "custom.label/value",
				"DISCOVERY_RUNTIME_PROCESS_DIR":            "/custom/workloads.d",
				"DISCOVERY_RUNTIME_PROCESS_POLL_INTERVAL":  "5s",
				"DISCOVERY_RUNTIME_PROCESS_LABEL_KEY":      "custom.process/discover",
				"DISCOVERY_RUNTIME_PROCESS_LABEL_VALUE":    "yes",
				"DISCOVERY_STORE_TYPE":                     "etcd",
				"DISCOVERY_STORE_ETCD_HOST":                "etcd.example.com",
				"DISCOVERY_STORE_ETCD_PORT":                "2380",
				"DISCOVERY_STORE_ETCD_USERNAME":            "admin",
				"DISCOVERY_STORE_ETCD_PASSWORD":            "secret",
				"DISCOVERY_STORE_ETCD_DIAL_TIMEOUT":        "10s",
				"DISCOVERY_STORE_ETCD_WORKLOADS_PREFIX":    "/custom/workloads/",
				"DISCOVERY_STORE_CRD_NAMESPACE":            "crd-namespace",
				"DISCOVERY_STORE_CRD_KUBECONFIG":           "crd-kubeconfig",
				"DISCOVERY_STORE_CRD_RESYNC_PERIOD":        "15s",
				"DISCOVERY_STORE_SQLITE_PATH":              "/custom/workloads.db",
				"DISCOVERY_RESOLVER_A2A_ENABLED":           "true",
				"DISCOVERY_RESOLVER_A2A_TIMEOUT":           "10s",
				"DISCOVERY_RESOLVER_A2A_LABEL_KEY":         "custom.a2a/type",
				"DISCOVERY_RESOLVER_A2A_LABEL_VALUE":       "agent",
				"DISCOVERY_RESOLVER_A2A_PATHS":             "/.well-known/new.json",
				"DISCOVERY_RESOLVER_OASF_ENABLED":          "true",
				"DISCOVERY_RESOLVER_OASF_TIMEOUT":          "15s",
				"DISCOVERY_RESOLVER_OASF_LABEL_KEY":        "custom.oasf/record",
			},
			ExpectedConfig: &Config{
				Workers:    8,
				InstanceID: "custom-instance",
				Runtime: runtime.Config{
					Type: docker.RuntimeType,
					Docker: docker.Config{
						Host:       "unix:///custom/docker.sock",
						LabelKey:   "custom.label/discover",
						LabelValue: "enabled",
					},
					Kubernetes: k8s.Config{
						Kubeconfig: "path-to-kubeconfig",
						Namespace:  "namespace",
						LabelKey:   "custom.label/key",
						LabelValue: "custom.label/value",
					},
					Process: process.Config{
						Dir:          "/custom/workloads.d",
						PollInterval: 5 * time.Second,
						LabelKey:     "custom.process/discover",
						LabelValue:   "yes",
					},
				},
				Store: store.Config{
					Type: etcd.StoreType,
					Etcd: etcd.Config{
						Host:            "etcd.example.com",
						Port:            2380,
						Username:        "admin",
						Password:        "secret",
						DialTimeout:     10 * time.Second,
						WorkloadsPrefix: "/custom/workloads/",
					},
					CRD: crd.Config{
						Namespace:    "crd-namespace",
						Kubeconfig:   "crd-kubeconfig",
						ResyncPeriod: 15 * time.Second,
					},
					SQLite: sql.Config{
						Path: "/custom/workloads.db",
					},
				},
				Resolver: resolver.Config{
					A2A: a2a.Config{
						Enabled:    true,
						Timeout:    10 * time.Second,
						LabelKey:   "custom.a2a/type",
						LabelValue: "agent",
						Paths:      []string{"/.well-known/new.json"},
					},
					OASF: oasf.Config{
						Enabled:  true,
						Timeout:  15 * time.Second,
						LabelKey: "custom.oasf/record",
					},
				},
			},
		},
		{
			Name:    "Default config",
			EnvVars: map[string]string{},
			ExpectedConfig: &Config{
				Workers: DefaultWorkers,
				Runtime: runtime.Config{
					Type: DefaultRuntimeType,
					Docker: docker.Config{
						Host:       docker.DefaultHost,
						LabelKey:   docker.DefaultLabelKey,
						LabelValue: docker.DefaultLabelValue,
					},
					Kubernetes: k8s.Config{
						Kubeconfig: k8s.DefaultKubeconfig,
						Namespace:  k8s.DefaultNamespace,
						LabelKey:   k8s.DefaultLabelKey,
						LabelValue: k8s.DefaultLabelValue,
					},
					Process: process.Config{
						Dir:          process.DefaultDir,
						PollInterval: process.DefaultPollInterval,
						LabelKey:     process.DefaultLabelKey,
						LabelValue:   process.DefaultLabelValue,
					},
				},
				Store: store.Config{
					Type: DefaultStoreType,
					Etcd: etcd.Config{
						Host:            etcd.DefaultHost,
						Port:            etcd.DefaultPort,
						Username:        "",
						Password:        "",
						DialTimeout:     etcd.DefaultDialTimeout,
						WorkloadsPrefix: etcd.DefaultWorkloadsPrefix,
					},
					CRD: crd.Config{
						Namespace:    crd.DefaultNamespace,
						Kubeconfig:   "",
						ResyncPeriod: crd.DefaultResyncPeriod,
					},
					SQLite: sql.Config{
						Path: sql.DefaultPath,
					},
				},
				Resolver: resolver.Config{
					A2A: a2a.Config{
						Enabled:    true,
						Timeout:    a2a.DefaultTimeout,
						LabelKey:   a2a.DefaultLabelKey,
						LabelValue: a2a.DefaultLabelValue,
						Paths:      []string{"/.well-known/agent-card.json", "/.well-known/card.json"},
					},
					OASF: oasf.Config{
						Enabled:  true,
						Timeout:  oasf.DefaultTimeout,
						LabelKey: oasf.DefaultLabelKey,
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			for k, v := range test.EnvVars {
				t.Setenv(k, v)
			}

			config, err := LoadConfig()
			require.NoError(t, err)
			assert.Equal(t, *test.ExpectedConfig, *config)
		})
	}
}
