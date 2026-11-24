// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kafka

import (
	"context"
	"strconv"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/write"
	v1 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v1"
	"github.com/synnaxlabs/x/content"
	"github.com/synnaxlabs/x/validate"
)

// writeResolver resolves write configs against the ICDs and channels that they write.
type writeResolver struct {
	icds     *icd.Service
	channels v1.Channels
	writes   write.Resolver
	ins      alamos.Instrumentation
}

// resolve moves the inline channels of cfg into targets that send a new ICD named name,
// then checks cfg, read through snap. A config holds channels or targets, not both, and
// the message of each enabled target must build from its ICD and its channels. It
// returns the write that creates the ICD, or nil when cfg holds no inline channels.
func (r writeResolver) resolve(
	ctx context.Context,
	snap gorp.Snapshot,
	name string,
	cfg *WriteConfig,
) (func(context.Context, gorp.Tx) error, error) {
	ic, ok, err := v1.ResolveWriteConfig(
		ctx, snap, r.channels, name, cfg, v1.NoteLift(r.ins, cfg.Key),
	)
	if err != nil {
		return nil, err
	}
	if len(cfg.Channels) > 0 && len(cfg.Targets) > 0 {
		return nil, validate.PathedError(
			validate.Errorf("a task cannot have both channels and targets"), "targets",
		)
	}
	icds := make(map[icd.Key]icd.ICD)
	if ok {
		icds[ic.Key] = ic
	}
	if err = r.writes.Check(ctx, snap, writesOf(cfg), icds); err != nil || !ok {
		return nil, err
	}
	return func(ctx context.Context, tx gorp.Tx) error {
		return r.icds.NewWriter(tx).Create(ctx, &ic)
	}, nil
}

// hash writes to h the payload of the ICD message that each enabled target of cfg
// sends, so an edit to the format of one changes the config hash of the task.
func (r writeResolver) hash(
	ctx context.Context,
	snap gorp.Snapshot,
	cfg *WriteConfig,
	h *content.Hash,
) error {
	return r.writes.Hash(ctx, snap, h, writesOf(cfg))
}

// writesOf returns the write of each enabled target of cfg.
func writesOf(cfg *WriteConfig) []write.Write {
	var writes []write.Write
	for i, t := range cfg.Targets {
		if !t.Disabled {
			writes = append(writes, write.Write{
				Path:    []string{"targets", strconv.Itoa(i)},
				ICD:     t.ICD,
				Message: t.Message,
			})
		}
	}
	return writes
}
