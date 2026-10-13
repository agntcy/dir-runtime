// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oasf

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/agntcy/dir-runtime/discovery/types"
	"github.com/agntcy/dir-runtime/utils"
	corev1 "github.com/agntcy/dir/api/core/v1"
	runtimev1 "github.com/agntcy/dir/api/runtime/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const ResolverType types.WorkloadResolverType = "oasf"

var logger = utils.NewLogger("resolver", "oasf")

// resolver resolves OASF records for workloads.
type resolver struct {
	timeout  time.Duration
	client   *client.Client
	labelKey string
}

// NewResolver creates a new OASF resolver.
func NewResolver(ctx context.Context, cfg Config) (types.WorkloadResolver, error) {
	// Create context with timeout, inheriting from parent context
	initCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	// Create Dir client
	clientCfg := client.WithEnvConfig()
	if cfg.Client != nil {
		clientCfg = client.WithConfig(cfg.Client)
	}

	dirClient, err := client.New(initCtx, clientCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Dir client: %w", err)
	}

	// Return resolver
	return &resolver{
		timeout:  cfg.Timeout,
		client:   dirClient,
		labelKey: cfg.LabelKey,
	}, nil
}

// Name returns the resolver name.
func (r *resolver) Name() types.WorkloadResolverType {
	return ResolverType
}

// CanResolve returns whether this resolver can resolve the workload.
func (r *resolver) CanResolve(workload *runtimev1.Workload) bool {
	// If workload has a label key, mark it as resolvable
	if _, hasLabel := workload.GetLabels()[r.labelKey]; hasLabel {
		return true
	}

	// If workload has an annotation key, mark it as resolvable.
	// Annotations may need to be used in K8s environments where labels are
	// limited in length and structure, e.g. DNS-styled labels.
	if _, hasAnnotation := workload.GetAnnotations()[r.labelKey]; hasAnnotation {
		return true
	}

	// Otherwise, cannot resolve
	return false
}

// Resolve fetches OASF record for the workload.
func (r *resolver) Resolve(ctx context.Context, workload *runtimev1.Workload) (any, error) {
	// Get the name of the OASF record from the workload labels
	name, version, cid, err := r.extractRecordIDs(workload)
	if err != nil {
		return nil, fmt.Errorf("failed to get OASF record name and version: %w", err)
	}

	logger.Info("resolving OASF record", "workload", workload.GetId(), "name", name, "version", version, "cid", cid)

	// Extract the record CID based on extracted identifiers
	// TODO: this should be done by the Resolve method as part of the resolution process,
	// TODO: but for now we need to do it here to support both name:version and cid-based resolution.
	recordCID := cid
	if recordCID == "" {
		// Fetch the OASF record using the provided context
		resolve, err := r.client.Resolve(ctx, name, version)
		if isNotFound(err) {
			return unresolved(workload, name+":"+version), nil
		}

		if err != nil {
			return nil, fmt.Errorf("failed to fetch OASF record (%s:%s): %w", name, version, err)
		}

		// Get the first info
		if len(resolve.GetRecords()) == 0 {
			return unresolved(workload, name+":"+version), nil
		}

		// Use the first record's CID as the record CID to pull
		recordCID = resolve.GetRecords()[0].GetCid()
	}

	// Pull the full record data using the provided context
	record, err := r.client.Pull(ctx, &corev1.RecordRef{Cid: recordCID})
	if isNotFound(err) {
		return unresolved(workload, recordCID), nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to pull OASF record %s: %w", recordCID, err)
	}

	// Get the record signature verified status
	verified, err := r.client.Verify(ctx, &signv1.VerifyRequest{
		RecordRef: &corev1.RecordRef{Cid: recordCID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to verify OASF record %s: %w", recordCID, err)
	}

	logger.Info("resolved successfully", "workload", workload.GetId(), "record", recordCID)

	// Return the record data along with its validity
	return map[string]any{
		"cid":               recordCID,
		"record":            record.GetData().AsMap(),
		"verified":          verified.GetSuccess(),
		"verification_data": verified,
	}, nil
}

// Apply implements types.WorkloadResolver.
func (r *resolver) Apply(ctx context.Context, workload *runtimev1.Workload, result any) error {
	// Convert result to map
	data, err := utils.InterfaceToStruct(result)
	if err != nil {
		return fmt.Errorf("failed to convert result to struct: %w", err)
	}

	// Update OASF field on workload
	if workload.GetServices() == nil {
		workload.Services = &runtimev1.WorkloadServices{}
	}

	workload.Services.Oasf = data

	return nil
}

// extractRecordIDs extracts the OASF record (name, version, cid) from the workload labels or annotations.
// Returns "name", "version", "", if both name and version are present as "name:version
// Returns "", "", "cid", if only cid is present
// Returns an error if neither is present or if the format is invalid.
//
//nolint:mnd
func (r *resolver) extractRecordIDs(workload *runtimev1.Workload) (string, string, string, error) {
	// Get the record FQDN from the workload labels or annotations
	var recordFQDN string
	if val, hasLabel := workload.GetLabels()[r.labelKey]; hasLabel {
		recordFQDN = val
	} else if val, hasAnnotation := workload.GetAnnotations()[r.labelKey]; hasAnnotation {
		recordFQDN = val
	} else {
		return "", "", "", fmt.Errorf("workload %s does not have label or annotation %s", workload.GetId(), r.labelKey)
	}

	// Split the record FQDN into name and version
	nameParts := strings.SplitN(recordFQDN, ":", 2)

	// If both name and version are present, return as ("name", "version", "")
	if len(nameParts) == 2 {
		return nameParts[0], nameParts[1], "", nil
	}

	// If only one part is present, return as ("", "", "cid")
	if len(nameParts) == 1 {
		return "", "", nameParts[0], nil
	}

	return "", "", "", fmt.Errorf("invalid record FQDN format: %s", recordFQDN)
}

// isNotFound reports whether err says the record doesn't exist on the Directory the
// resolver talks to, as opposed to a failure worth retrying.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}

	if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "not found")
}

// unresolved is the result for a record reference the Directory doesn't know, for example
// a record kept on a private node. It is not an error: the workload is listed without a
// record, and the resolver doesn't retry.
func unresolved(workload *runtimev1.Workload, ref string) map[string]any {
	logger.Info("OASF record not found; listing the workload without it", "workload", workload.GetId(), "record", ref)

	return map[string]any{
		"ref":      ref,
		"resolved": false,
	}
}
