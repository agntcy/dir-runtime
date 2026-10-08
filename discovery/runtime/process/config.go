// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package process

import "time"

const (
	// DefaultDir is the default directory watched for workload descriptor files.
	DefaultDir = "~/.agntcy/dir-runtime/workloads.d"

	// DefaultPollInterval is the default interval between descriptor directory scans.
	DefaultPollInterval = 2 * time.Second

	// DefaultLabelKey is the default label key to filter process workloads.
	DefaultLabelKey = "org.agntcy/discover"

	// DefaultLabelValue is the default label value to filter process workloads.
	DefaultLabelValue = "true"
)

// Config holds process runtime configuration.
type Config struct {
	// Dir is the directory containing workload descriptor files. A leading "~" is expanded to the user's home.
	Dir string `json:"dir,omitempty" mapstructure:"dir"`

	// PollInterval is the interval between descriptor directory scans.
	PollInterval time.Duration `json:"poll_interval,omitempty" mapstructure:"poll_interval"`

	// LabelKey is the label key to filter process workloads.
	LabelKey string `json:"label_key,omitempty" mapstructure:"label_key"`

	// LabelValue is the label value to filter process workloads.
	LabelValue string `json:"label_value,omitempty" mapstructure:"label_value"`
}
