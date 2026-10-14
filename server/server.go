// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/agntcy/dir-runtime/server/database"
	grpcserver "github.com/agntcy/dir-runtime/server/grpc"
	storetypes "github.com/agntcy/dir-runtime/store/types"
	runtimev1 "github.com/agntcy/dir/api/runtime/v1"
	"google.golang.org/grpc"
)

type options struct {
	acceptedSchemaURLs []string
}

// Option configures the runtime services registered by Register.
type Option func(*options) error

// WithAcceptedSchemaURLs sets the OASF schema servers whose A2A card extensions are read
// when filtering workloads by agent (see ListWorkloadsRequest). When unset, only
// runtimev1.DefaultOASFSchemaURL is accepted. Each URL must use https, or plain http on a
// loopback host, e.g. a schema server running locally for testing.
func WithAcceptedSchemaURLs(urls []string) Option {
	return func(o *options) error {
		for _, u := range urls {
			if err := ValidateSchemaURL(u); err != nil {
				return err
			}
		}

		o.acceptedSchemaURLs = urls

		return nil
	}
}

// ValidateSchemaURL checks that u is an absolute https URL, or an http URL on a loopback host.
func ValidateSchemaURL(u string) error {
	parsed, err := url.Parse(u)
	if err != nil {
		return fmt.Errorf("invalid OASF schema URL %q: %w", u, err)
	}

	if parsed.Hostname() == "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
		return fmt.Errorf("invalid OASF schema URL %q: want scheme://host[:port][/path]", u)
	}

	switch parsed.Scheme {
	case "https":
		return nil
	case "http":
		if isLoopbackHost(parsed.Hostname()) {
			return nil
		}

		return fmt.Errorf("OASF schema URL %q: plain http is only accepted on a loopback host", u)
	default:
		return fmt.Errorf("OASF schema URL %q: scheme must be https (or http on a loopback host)", u)
	}
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}

	ip := net.ParseIP(host)

	return ip != nil && ip.IsLoopback()
}

// Register registers runtime services on the provided gRPC registrar.
func Register(server *grpc.Server, store storetypes.StoreReader, opts ...Option) error {
	o := &options{}

	for _, opt := range opts {
		if opt == nil {
			return errors.New("option cannot be nil")
		}

		if err := opt(o); err != nil {
			return err
		}
	}

	// Create database
	db, err := database.NewDatabase(store, database.WithAcceptedSchemaURLs(o.acceptedSchemaURLs))
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	// Register discovery service
	discoveryServer := grpcserver.NewDiscovery(db)
	runtimev1.RegisterDiscoveryServiceServer(server, discoveryServer)

	return nil
}
