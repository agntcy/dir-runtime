// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package process

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/agntcy/dir-runtime/discovery/types"
	runtimev1 "github.com/agntcy/dir/api/runtime/v1"
)

const (
	// RuntimeType is the process runtime type.
	RuntimeType types.RuntimeType = "process"

	// defaultAddress is used when a descriptor does not list any addresses.
	defaultAddress = "127.0.0.1"

	// isolationGroup is the isolation group of every process workload on this host.
	isolationGroup = "host"

	maxPort = 65535
)

// descriptor is the JSON file a process writes to announce itself.
type descriptor struct {
	Name        string            `json:"name"`
	PID         int               `json:"pid"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Addresses   []string          `json:"addresses"`
	Ports       []json.Number     `json:"ports"`
}

// parseDescriptor decodes and validates a workload descriptor.
// Ports may be given as JSON strings or numbers.
func parseDescriptor(data []byte) (*descriptor, error) {
	var desc descriptor
	if err := json.Unmarshal(data, &desc); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	if desc.Name == "" {
		return nil, errors.New("name is required")
	}

	if desc.PID <= 0 {
		return nil, fmt.Errorf("pid must be positive, got %d", desc.PID)
	}

	if len(desc.Ports) == 0 {
		return nil, errors.New("at least one port is required")
	}

	for _, port := range desc.Ports {
		value, err := strconv.Atoi(port.String())
		if err != nil || value < 1 || value > maxPort {
			return nil, fmt.Errorf("invalid port %q", port.String())
		}
	}

	return &desc, nil
}

// toWorkload converts the descriptor into a workload, filling in the fields the adapter owns.
func (d *descriptor) toWorkload(id, hostname string) *runtimev1.Workload {
	addresses := d.Addresses
	if len(addresses) == 0 {
		addresses = []string{defaultAddress}
	}

	ports := make([]string, 0, len(d.Ports))
	for _, port := range d.Ports {
		ports = append(ports, port.String())
	}

	labels := d.Labels
	if labels == nil {
		labels = make(map[string]string)
	}

	annotations := d.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}

	return &runtimev1.Workload{
		Id:              id,
		Name:            d.Name,
		Hostname:        hostname,
		Runtime:         string(RuntimeType),
		Type:            runtimev1.WorkloadType_WORKLOAD_TYPE_PROCESS.GetName(),
		Labels:          labels,
		Annotations:     annotations,
		Addresses:       addresses,
		Ports:           ports,
		IsolationGroups: []string{isolationGroup},
	}
}
