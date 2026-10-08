// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"testing"
	"time"

	"github.com/agntcy/dir-runtime/discovery/runtime/config"
	"github.com/agntcy/dir-runtime/discovery/runtime/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAdapterProcess(t *testing.T) {
	adapter, err := NewAdapter(config.Config{
		Type: process.RuntimeType,
		Process: process.Config{
			Dir:          t.TempDir(),
			PollInterval: time.Second,
			LabelKey:     process.DefaultLabelKey,
			LabelValue:   process.DefaultLabelValue,
		},
	})
	require.NoError(t, err)

	assert.Equal(t, process.RuntimeType, adapter.Type())
}

func TestNewAdapterUnsupported(t *testing.T) {
	_, err := NewAdapter(config.Config{Type: "unknown"})
	require.ErrorContains(t, err, "unsupported runtime")
}
