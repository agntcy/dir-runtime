// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package resolver

import (
	"context"
	"testing"
	"time"

	"github.com/agntcy/dir-runtime/discovery/resolver/a2a"
	"github.com/agntcy/dir-runtime/discovery/resolver/config"
	"github.com/agntcy/dir-runtime/discovery/resolver/oasf"
	"github.com/agntcy/dir/client"
)

func TestNewResolvers_SkipsAResolverThatFailsToStart(t *testing.T) {
	cfg := config.Config{
		A2A: a2a.Config{Enabled: true},
		OASF: oasf.Config{
			Enabled: true,
			Timeout: time.Second,
			// An unsupported auth mode makes the Directory client, and with it the
			// OASF resolver, fail to start.
			Client: &client.Config{ServerAddress: "127.0.0.1:1", AuthMode: "no-such-mode"},
		},
	}

	resolvers, err := NewResolvers(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewResolvers() error = %v, want the failing resolver skipped", err)
	}

	if len(resolvers) != 1 || resolvers[0].Name() != a2a.ResolverType {
		t.Fatalf("resolvers = %v, want only the A2A resolver", resolvers)
	}
}

func TestNewResolvers_OnlyFailingResolverLeavesNone(t *testing.T) {
	cfg := config.Config{
		OASF: oasf.Config{
			Enabled: true,
			Timeout: time.Second,
			Client:  &client.Config{ServerAddress: "127.0.0.1:1", AuthMode: "no-such-mode"},
		},
	}

	resolvers, err := NewResolvers(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewResolvers() error = %v, want no error", err)
	}

	if len(resolvers) != 0 {
		t.Fatalf("resolvers = %v, want none", resolvers)
	}
}

func TestNewResolvers_NoneEnabledIsAnError(t *testing.T) {
	if _, err := NewResolvers(context.Background(), config.Config{}); err == nil {
		t.Fatal("NewResolvers() with no resolver enabled must fail")
	}
}
