// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package database

import (
	"context"
	"fmt"

	storetypes "github.com/agntcy/dir-runtime/store/types"
	v1 "github.com/agntcy/dir/api/runtime/v1"
)

type Database struct {
	store              storetypes.StoreReader
	acceptedSchemaURLs []string
}

// Option configures a Database.
type Option func(*Database)

// WithAcceptedSchemaURLs sets the OASF schema servers whose card extensions are read
// when filtering by agent. When unset, only runtimev1.DefaultOASFSchemaURL is accepted.
func WithAcceptedSchemaURLs(urls []string) Option {
	return func(d *Database) {
		d.acceptedSchemaURLs = urls
	}
}

func NewDatabase(store storetypes.StoreReader, opts ...Option) (*Database, error) {
	db := &Database{
		store: store,
	}

	for _, opt := range opts {
		opt(db)
	}

	return db, nil
}

func (d *Database) Count(ctx context.Context) int {
	ids, err := d.store.ListWorkloadIDs(ctx)
	if err != nil {
		return 0
	}

	return len(ids)
}

func (d *Database) Get(ctx context.Context, id string) (*v1.Workload, error) {
	// Fetch workload from store
	workload, err := d.store.GetWorkload(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get workload by ID: %w", err)
	}

	return workload, nil
}

// List returns the workloads matching every filter of req, as documented on
// ListWorkloadsRequest. A nil request matches every workload.
func (d *Database) List(ctx context.Context, req *v1.ListWorkloadsRequest) ([]*v1.Workload, error) {
	// Fetch all workloads
	result, err := d.store.ListWorkloads(ctx)
	if err != nil {
		return []*v1.Workload{}, fmt.Errorf("failed to list workloads: %w", err)
	}

	if req == nil {
		req = &v1.ListWorkloadsRequest{}
	}

	opts := v1.FilterOptions{AcceptedSchemaURLs: d.acceptedSchemaURLs}

	// The reference workload of a reachability filter is one of the listed workloads;
	// if none has its ID, nothing matches.
	if req.ReachableFromWorkloadId != nil {
		for _, workload := range result {
			if workload.GetId() == req.GetReachableFromWorkloadId() {
				opts.ReachableFrom = workload

				break
			}
		}
	}

	var filtered []*v1.Workload

	for _, workload := range result {
		if workload.MatchesFilters(req, opts) {
			filtered = append(filtered, workload)
		}
	}

	return filtered, nil
}
