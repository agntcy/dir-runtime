// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"net"
	"slices"
	"strings"

	runtimev1 "github.com/agntcy/dir/api/runtime/v1"
)

// LocatorProtocolTCP is the locator protocol for endpoints whose application protocol
// is not known from the runtime, only their network address and port.
const LocatorProtocolTCP = "tcp"

// LocatorProtocolSLIM is the locator protocol for endpoints reached through a SLIM node.
// An A2A agent card spells the same binding SLIMRPC.
const LocatorProtocolSLIM = "slim"

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

// MergeLocators returns the locators of every given list, without duplicates and in a
// stable order. It returns nil when the result is empty.
//
// Locators are keyed on protocol and URL together rather than URL alone, because one
// endpoint can serve two protocols and the locator-protocol filter treats those as
// genuinely different ways to reach the workload. Sorting keeps a caller that takes the
// first locator on the same answer from one resolve to the next.
func MergeLocators(lists ...[]*runtimev1.WorkloadLocator) []*runtimev1.WorkloadLocator {
	var merged []*runtimev1.WorkloadLocator

	seen := map[[2]string]bool{}

	for _, list := range lists {
		for _, locator := range list {
			key := [2]string{locator.GetProtocol(), locator.GetUrl()}
			if seen[key] {
				continue
			}

			seen[key] = true

			merged = append(merged, locator)
		}
	}

	if len(merged) == 0 {
		return nil
	}

	slices.SortFunc(merged, func(a, b *runtimev1.WorkloadLocator) int {
		if byProtocol := strings.Compare(a.GetProtocol(), b.GetProtocol()); byProtocol != 0 {
			return byProtocol
		}

		return strings.Compare(a.GetUrl(), b.GetUrl())
	})

	return merged
}
