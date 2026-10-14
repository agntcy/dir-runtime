// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package resolver

import (
	"context"
	"errors"

	"github.com/agntcy/dir-runtime/discovery/resolver/a2a"
	"github.com/agntcy/dir-runtime/discovery/resolver/config"
	"github.com/agntcy/dir-runtime/discovery/resolver/oasf"
	"github.com/agntcy/dir-runtime/discovery/types"
	"github.com/agntcy/dir-runtime/utils"
)

var logger = utils.NewLogger("resolver", "setup")

// NewResolvers creates the enabled resolvers. Resolvers only enrich workloads, so one
// that fails to start (e.g. the OASF resolver when its Directory client can't be
// created) is logged and left out rather than stopping discovery: workloads are still
// listed, without what that resolver would have added. It returns an error only when
// no resolver is enabled at all.
func NewResolvers(ctx context.Context, cfg config.Config) ([]types.WorkloadResolver, error) {
	if !cfg.A2A.Enabled && !cfg.OASF.Enabled {
		return nil, errors.New("no resolvers enabled")
	}

	var resolvers []types.WorkloadResolver

	if cfg.A2A.Enabled {
		resolvers = append(resolvers, a2a.NewResolver(cfg.A2A))
	}

	if cfg.OASF.Enabled {
		oasfResolver, err := oasf.NewResolver(ctx, cfg.OASF)
		if err != nil {
			logger.Error("OASF resolver failed to start; workloads are listed without their OASF records",
				"error", err)
		} else {
			resolvers = append(resolvers, oasfResolver)
		}
	}

	return resolvers, nil
}
