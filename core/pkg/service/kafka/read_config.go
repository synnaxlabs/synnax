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

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/read"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	icdv1 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v1"
	v1 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v1"
	"github.com/synnaxlabs/x/content"
)

// readChannels returns the v1.Channels that reads channels from svc.
func readChannels(svc *channel.Service) v1.Channels {
	return func(
		ctx context.Context,
		snap gorp.Snapshot,
		keys []channel.Key,
	) (map[channel.Key]v1.Channel, error) {
		var found []channel.Channel
		if err := svc.NewRetrieve().
			Where(channel.MatchKeys(keys...)).
			SkipMissingKeys().
			Entries(&found).
			Exec(ctx, snap); err != nil {
			return nil, err
		}
		return lo.SliceToMap(found, func(c channel.Channel) (channel.Key, v1.Channel) {
			return c.Key(), v1.Channel{
				Name:     c.Name,
				DataType: c.DataType,
				Index:    c.IsIndex,
			}
		}), nil
	}
}

// createICD returns the v1.CreateICD that lifts an ICD to the current shape and creates
// it with svc.
func createICD(svc *icd.Service) v1.CreateICD {
	return func(ctx context.Context, tx gorp.Tx, old icdv0.ICD) error {
		ic, err := icdv1.MigrateICD(ctx, old)
		if err != nil {
			return err
		}
		return svc.NewWriter(tx).Create(ctx, &ic)
	}
}

// readResolver resolves read configs against the ICDs and channels they read.
type readResolver struct {
	icds     *icd.Service
	channels v1.Channels
	create   v1.CreateICD
	reads    read.Resolver
	ins      alamos.Instrumentation
}

// resolve moves the inline records of cfg into a new ICD named name, then checks each
// enabled record of cfg against its ICD and its channels, read through snap. It returns
// the write that creates the ICD, or nil when cfg holds no inline records.
func (r readResolver) resolve(
	ctx context.Context,
	snap gorp.Snapshot,
	name string,
	cfg *ReadConfig,
) (func(context.Context, gorp.Tx) error, error) {
	staged, ok, err := v1.ResolveReadConfig(
		ctx, snap, r.channels, name, cfg, v1.NoteLift(r.ins, cfg.Key),
	)
	if err != nil {
		return nil, err
	}
	var (
		icds = make(map[icd.Key]icd.ICD)
		ic   icd.ICD
	)
	if ok {
		if ic, err = icdv1.MigrateICD(ctx, staged); err != nil {
			return nil, err
		}
		icds[ic.Key] = ic
	}
	if err = r.check(ctx, snap, cfg, icds); err != nil || !ok {
		return nil, err
	}
	return func(ctx context.Context, tx gorp.Tx) error {
		return r.icds.NewWriter(tx).Create(ctx, &ic)
	}, nil
}

// hash writes to h the payload of each ICD message that an enabled record of cfg reads,
// so an edit to the format of one changes the config hash of the task.
func (r readResolver) hash(
	ctx context.Context,
	snap gorp.Snapshot,
	cfg *ReadConfig,
	h *content.Hash,
) error {
	return r.reads.Hash(ctx, snap, h, readsOf(cfg))
}

// check checks each enabled record of cfg against its ICD, from icds or else read
// through snap, and its channels.
func (r readResolver) check(
	ctx context.Context,
	snap gorp.Snapshot,
	cfg *ReadConfig,
	icds map[icd.Key]icd.ICD,
) error {
	reads := readsOf(cfg)
	c, err := r.reads.NewChecker(ctx, snap, reads, nil, icds)
	if err != nil {
		return err
	}
	for _, rd := range reads {
		if err = c.Read(rd); err != nil {
			return err
		}
	}
	return nil
}

// readsOf returns the read of each enabled record of cfg.
func readsOf(cfg *ReadConfig) []read.Read {
	var reads []read.Read
	for t, topic := range cfg.Topics {
		for i, rec := range topic.Records {
			if rec.Disabled {
				continue
			}
			path := []string{"topics", strconv.Itoa(t), "records", strconv.Itoa(i)}
			reads = append(reads, read.Read{
				Path:     path,
				ICD:      rec.ICD,
				Messages: rec.Messages,
			})
		}
	}
	return reads
}
