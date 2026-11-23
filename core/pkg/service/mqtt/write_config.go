// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mqtt

import (
	"context"
	"strconv"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/icd/write"
	v3 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v3"
	"github.com/synnaxlabs/x/content"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// createWriteICD returns the v3.CreateICD that creates an ICD with svc.
func createWriteICD(svc *icd.Service) v3.CreateICD {
	return func(ctx context.Context, tx gorp.Tx, ic icd.ICD) error {
		return svc.NewWriter(tx).Create(ctx, &ic)
	}
}

// writeResolver resolves write configs against the ICDs and channels they write.
type writeResolver struct {
	icds     *icd.Service
	channels icdv0.Channels
	ins      alamos.Instrumentation
}

// resolve moves the inline targets of cfg into a new ICD named name, then checks each
// enabled plain target of cfg against its ICD and its channels, read through snap. It
// returns the write that creates the ICD, or nil when cfg holds no inline targets.
func (r writeResolver) resolve(
	ctx context.Context,
	snap gorp.Snapshot,
	name string,
	cfg *WriteConfig,
) (func(context.Context, gorp.Tx) error, error) {
	ic, ok, err := v3.ResolveWriteConfig(
		ctx, snap, r.channels, name, cfg, v3.NoteLift(r.ins, cfg.Key),
	)
	if err != nil {
		return nil, err
	}
	icds := make(map[icd.Key]icd.ICD)
	if ok {
		icds[ic.Key] = ic
	}
	if err = r.check(ctx, snap, cfg, icds); err != nil || !ok {
		return nil, err
	}
	return func(ctx context.Context, tx gorp.Tx) error {
		return r.icds.NewWriter(tx).Create(ctx, &ic)
	}, nil
}

// hash writes to h the payload of the ICD message that each enabled plain target of cfg
// sends, so an edit to the format of one changes the config hash of the task.
func (r writeResolver) hash(
	ctx context.Context,
	snap gorp.Snapshot,
	cfg *WriteConfig,
	h *content.Hash,
) error {
	targets := enabledPlainTargets(cfg)
	if len(targets) == 0 {
		return nil
	}
	icds, err := r.retrieve(ctx, snap, targets, nil)
	if err != nil {
		return err
	}
	for _, t := range targets {
		if err = write.Hash(h, icds[t.ICD], t.Message); err != nil {
			return err
		}
	}
	return nil
}

// check builds the encoder of each enabled plain target of cfg from its ICD, from icds
// or else read through snap, and its channels. It skips a target whose ICD does not
// exist, which the reference check of the write reports.
func (r writeResolver) check(
	ctx context.Context,
	snap gorp.Snapshot,
	cfg *WriteConfig,
	icds map[icd.Key]icd.ICD,
) error {
	targets := enabledPlainTargets(cfg)
	if len(targets) == 0 {
		return nil
	}
	icds, err := r.retrieve(ctx, snap, targets, icds)
	if err != nil {
		return err
	}
	var keys []channel.Key
	for _, t := range targets {
		for _, b := range write.ChannelBindings(t.Message) {
			if b.Channel != 0 {
				keys = append(keys, b.Channel)
			}
		}
	}
	var channels map[channel.Key]icdv0.Channel
	if len(keys) > 0 {
		if channels, err = r.channels(ctx, snap, lo.Uniq(keys)); err != nil {
			return err
		}
	}
	dataTypes := lo.MapValues(
		channels,
		func(ch icdv0.Channel, _ channel.Key) telem.DataType { return ch.DataType },
	)
	for i, tg := range cfg.Targets {
		t, ok := tg.Variant.(PlainWriteTarget)
		if !ok || t.Disabled {
			continue
		}
		ic, ok := icds[t.ICD]
		if !ok {
			continue
		}
		if _, err = write.Encoder(t.Message, ic, dataTypes); err != nil {
			return validate.PathedError(err, "targets", strconv.Itoa(i), "message")
		}
	}
	return nil
}

// retrieve returns icds with the ICD of each of targets that icds lacks, read through
// snap. An ICD that does not exist is absent from the result.
func (r writeResolver) retrieve(
	ctx context.Context,
	snap gorp.Snapshot,
	targets []PlainWriteTarget,
	icds map[icd.Key]icd.ICD,
) (map[icd.Key]icd.ICD, error) {
	missing := lo.Uniq(
		lo.FilterMap(targets, func(t PlainWriteTarget, _ int) (icd.Key, bool) {
			_, ok := icds[t.ICD]
			return t.ICD, !ok
		}),
	)
	if len(missing) == 0 {
		return icds, nil
	}
	var found []icd.ICD
	if err := r.icds.NewRetrieve().
		Where(icd.MatchKeys(missing...)).
		SkipMissingKeys().
		Entries(&found).
		Exec(ctx, snap); err != nil {
		return nil, err
	}
	return lo.Assign(
		icds,
		lo.KeyBy(found, func(ic icd.ICD) icd.Key { return ic.Key }),
	), nil
}

// enabledPlainTargets returns the enabled plain targets of cfg.
func enabledPlainTargets(cfg *WriteConfig) []PlainWriteTarget {
	return lo.FilterMap(
		cfg.Targets,
		func(t WriteTarget, _ int) (PlainWriteTarget, bool) {
			plain, ok := t.Variant.(PlainWriteTarget)
			return plain, ok && !plain.Disabled
		},
	)
}
