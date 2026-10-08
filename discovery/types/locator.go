// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"net"
	"slices"

	runtimev1 "github.com/agntcy/dir/api/runtime/v1"
)

// LocatorProtocolTCP is the locator protocol for endpoints whose application protocol
// is not known from the runtime, only their network address and port.
const LocatorProtocolTCP = "tcp"

// TCPLocators returns one tcp locator per combination of address and port, sorted by URL.
// It returns nil when either list is empty.
func TCPLocators(addresses, ports []string) []*runtimev1.WorkloadLocator {
	urls := make([]string, 0, len(addresses)*len(ports))

	for _, address := range addresses {
		for _, port := range ports {
			url := LocatorProtocolTCP + "://" + net.JoinHostPort(address, port)
			if !slices.Contains(urls, url) {
				urls = append(urls, url)
			}
		}
	}

	if len(urls) == 0 {
		return nil
	}

	slices.Sort(urls)

	locators := make([]*runtimev1.WorkloadLocator, 0, len(urls))
	for _, url := range urls {
		locators = append(locators, &runtimev1.WorkloadLocator{Protocol: LocatorProtocolTCP, Url: url})
	}

	return locators
}
