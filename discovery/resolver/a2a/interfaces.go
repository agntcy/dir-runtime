// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package a2a

import (
	"net/url"
	"strings"

	"github.com/agntcy/dir-runtime/discovery/types"
	runtimev1 "github.com/agntcy/dir/api/runtime/v1"
)

// cardInterfacesField is the agent card field listing the protocol bindings an agent answers on.
const cardInterfacesField = "supportedInterfaces"

// cardLocators returns a locator for every entry of the card's supportedInterfaces whose URL
// is dialable on its own. Entries naming only the agent are dropped, see dialableURL.
//
// Unlike the locators a runtime adapter observes, these are claims the workload makes about
// itself. A caller is expected to verify the peer when it dials, not to trust the inventory.
func cardLocators(card map[string]any) []*runtimev1.WorkloadLocator {
	interfaces, _ := card[cardInterfacesField].([]any)

	var locators []*runtimev1.WorkloadLocator

	for _, iface := range interfaces {
		entry, ok := iface.(map[string]any)
		if !ok {
			continue
		}

		rawURL, _ := entry["url"].(string)

		binding, _ := entry["protocolBinding"].(string)

		protocol := locatorProtocol(binding)
		if protocol == "" {
			continue
		}

		dialable, ok := dialableURL(protocol, rawURL)
		if !ok {
			continue
		}

		locators = append(locators, &runtimev1.WorkloadLocator{Protocol: protocol, Url: dialable})
	}

	return locators
}

// locatorProtocol maps an A2A protocolBinding to a locator protocol. SLIMRPC becomes slim,
// the spelling the process adapter and the CLI already use, so the mismatch is absorbed in
// one place; every other binding keeps its own lowercased name.
func locatorProtocol(binding string) string {
	switch normalized := strings.ToLower(strings.TrimSpace(binding)); normalized {
	case "":
		return ""
	case "slimrpc":
		return types.LocatorProtocolSLIM
	default:
		return normalized
	}
}

// dialableURL normalizes a card interface URL and reports whether it carries everything
// needed to open a connection, without relying on configuration the caller may not have.
//
// A unicast endpoint may arrive as a bare host:port, because the card library strips http://
// from GRPC interfaces, so that form is given back the scheme AgentBridge assumes for it.
//
// SLIM is the strict case: only the slim://host:port/name form survives. SLIM resolves a
// name to a route, but the node endpoint to connect to is not part of that resolution and
// has to come from the URL. The slimrpc://org/namespace/name form names the agent and not
// its node. Note that it does parse with a host, "org", so the port is what tells the two
// apart.
func dialableURL(protocol, rawURL string) (string, bool) {
	normalized := strings.TrimSpace(rawURL)
	if normalized == "" {
		return "", false
	}

	parsed, err := url.Parse(normalized)
	if err != nil || parsed.Scheme == "" {
		if protocol == types.LocatorProtocolSLIM {
			return "", false
		}

		normalized = "http://" + strings.TrimPrefix(normalized, "//")

		parsed, err = url.Parse(normalized)
		if err != nil {
			return "", false
		}
	}

	if parsed.Host == "" {
		return "", false
	}

	if protocol == types.LocatorProtocolSLIM && parsed.Port() == "" {
		return "", false
	}

	return normalized, true
}
