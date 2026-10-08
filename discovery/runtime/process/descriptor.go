// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package process

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
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
	Locators    []locator         `json:"locators"`
}

// locator is an endpoint the process can be reached at, e.g. a SLIM name.
type locator struct {
	Protocol string `json:"protocol"`
	URL      string `json:"url"`
}

// parseDescriptor decodes and validates a workload descriptor.
// Locators, addresses and ports are all optional. Ports may be given as JSON strings or numbers.
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

	for _, port := range desc.Ports {
		value, err := strconv.Atoi(port.String())
		if err != nil || value < 1 || value > maxPort {
			return nil, fmt.Errorf("invalid port %q", port.String())
		}
	}

	for i, loc := range desc.Locators {
		if err := loc.validate(); err != nil {
			return nil, fmt.Errorf("locator %d: %w", i, err)
		}
	}

	return &desc, nil
}

func (l locator) validate() error {
	if l.Protocol == "" {
		return errors.New("protocol is required")
	}

	if l.URL == "" {
		return errors.New("url is required")
	}

	if u, err := url.Parse(l.URL); err != nil || u.Scheme == "" {
		return fmt.Errorf("url must include a scheme, got %q", l.URL)
	}

	return nil
}

// toWorkload converts the descriptor into a workload, filling in the fields the adapter owns.
func (d *descriptor) toWorkload(id, hostname string) *runtimev1.Workload {
	locators := make([]*runtimev1.WorkloadLocator, 0, len(d.Locators))
	for _, loc := range d.Locators {
		locators = append(locators, &runtimev1.WorkloadLocator{Protocol: loc.Protocol, Url: loc.URL})
	}

	// Addresses and ports become tcp locators; ports without addresses are local.
	if len(d.Ports) > 0 {
		addresses := d.Addresses
		if len(addresses) == 0 {
			addresses = []string{defaultAddress}
		}

		ports := make([]string, 0, len(d.Ports))
		for _, port := range d.Ports {
			ports = append(ports, port.String())
		}

		locators = append(locators, types.TCPLocators(addresses, ports)...)
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
		Locators:        locators,
		IsolationGroups: []string{isolationGroup},
	}
}
