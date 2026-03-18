// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"

	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/channel/calculation/analyzer"
	"github.com/synnaxlabs/x/gorp"
)

// Writer wraps the distribution-layer channel writer, adding DataType inference
// for calculated channels before persisting.
type Writer struct {
	distchannel.Writer
	analyzer *analyzer.Analyzer
	tx       gorp.Tx
}

// Create creates a single channel, inferring the DataType if it is calculated.
func (w Writer) Create(ctx context.Context, c *Channel, opts ...CreateOption) error {
	channels := []Channel{*c}
	if err := w.CreateMany(ctx, &channels, opts...); err != nil {
		return err
	}
	*c = channels[0]
	return nil
}

// CreateMany creates multiple channels, inferring DataTypes for any calculated
// channels in the batch. Channels within the batch may reference each other by
// name.
func (w Writer) CreateMany(ctx context.Context, channels *[]Channel, opts ...CreateOption) error {
	if len(*channels) == 0 {
		return nil
	}
	for i, ch := range *channels {
		if !ch.IsCalculated() {
			continue
		}
		result, err := w.analyzer.Analyze(ctx, ch)
		if err != nil {
			return err
		}
		(*channels)[i].DataType = result.DataType
	}
	return w.Writer.CreateMany(ctx, channels, opts...)
}
