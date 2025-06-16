// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package reactive

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/slate/spec"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
)

func newStatusChange(_ context.Context, cfg factoryConfig) (bool, error) {
	if cfg.node.Type != spec.StatusChangeType {
		return false, nil
	}
	c := &spec.StatusChangerConfig{}
	if err := c.Parse(cfg.node.Config); err != nil {
		return true, err
	}
	sink := &confluence.UnarySink[spec.Value]{
		Sink: func(ctx context.Context, value spec.Value) error {
			cfg.OnStatusChange(ctx, status.Status[any]{
				Key:     uuid.New().String(),
				Variant: c.Variant,
				Message: c.Message,
				Time:    telem.Now(),
			})
			return nil
		},
	}
	plumber.SetSink[spec.Value](cfg.pipeline, address.Address(cfg.node.Key), sink)
	return true, nil
}
