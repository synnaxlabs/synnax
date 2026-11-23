// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http

import (
	"context"
	"strconv"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	v5 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v5"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/read"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	"github.com/synnaxlabs/x/content"
)

// readResolver resolves read configs against the ICDs and channels they read.
type readResolver struct {
	icds     *icd.Service
	channels icdv0.Channels
	ins      alamos.Instrumentation
}

// resolve moves the inline fields of cfg into a new ICD named name, then checks each
// endpoint of cfg against its ICD and its channels, read through snap. It returns the
// write that creates the ICD, or nil when cfg holds no inline fields.
func (r readResolver) resolve(
	ctx context.Context,
	snap gorp.Snapshot,
	name string,
	cfg *ReadConfig,
) (func(context.Context, gorp.Tx) error, error) {
	ic, ok, err := v5.ResolveReadConfig(
		ctx, snap, r.channels, name, cfg, v5.NoteLift(r.ins, cfg.Key),
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

// readsICD reports whether ep is enabled and names an ICD.
func readsICD(ep ReadEndpoint) bool { return !ep.Disabled && ep.ICD != nil }

// hash writes to h the payload of each ICD message that an enabled endpoint of cfg
// reads, so an edit to the format of one changes the config hash of the task.
func (r readResolver) hash(
	ctx context.Context,
	snap gorp.Snapshot,
	cfg *ReadConfig,
	h *content.Hash,
) error {
	icdKeys, _ := referencedKeys(cfg, nil)
	if len(icdKeys) == 0 {
		return nil
	}
	var found []icd.ICD
	if err := r.icds.NewRetrieve().
		Where(icd.MatchKeys(lo.Uniq(icdKeys)...)).
		Entries(&found).
		Exec(ctx, snap); err != nil {
		return err
	}
	icds := lo.KeyBy(found, func(ic icd.ICD) icd.Key { return ic.Key })
	for _, ep := range cfg.Endpoints {
		if !readsICD(ep) {
			continue
		}
		if err := read.Hash(h, icds[*ep.ICD], ep.Messages); err != nil {
			return err
		}
	}
	return nil
}

// check checks each enabled endpoint of cfg that names an ICD against that ICD, from
// icds or else read through snap, and its channels. It skips an ICD or a channel that
// does not exist, which the reference check of the write reports.
func (r readResolver) check(
	ctx context.Context,
	snap gorp.Snapshot,
	cfg *ReadConfig,
	icds map[icd.Key]icd.ICD,
) error {
	icdKeys, channelKeys := referencedKeys(cfg, icds)
	if len(icdKeys) > 0 {
		var found []icd.ICD
		if err := r.icds.NewRetrieve().
			Where(icd.MatchKeys(lo.Uniq(icdKeys)...)).
			SkipMissingKeys().
			Entries(&found).
			Exec(ctx, snap); err != nil {
			return err
		}
		for _, ic := range found {
			icds[ic.Key] = ic
		}
	}
	var channels map[channel.Key]icdv0.Channel
	if len(channelKeys) > 0 {
		var err error
		if channels, err = r.channels(ctx, snap, lo.Uniq(channelKeys)); err != nil {
			return err
		}
	}
	c := read.NewChecker(lo.MapValues(
		channels,
		func(ch icdv0.Channel, _ channel.Key) read.Channel { return read.Channel(ch) },
	))
	for i, ep := range cfg.Endpoints {
		if !readsICD(ep) {
			continue
		}
		ic, ok := icds[*ep.ICD]
		if !ok {
			continue
		}
		for j, rm := range ep.Messages {
			path := []string{"endpoints", strconv.Itoa(i), "messages", strconv.Itoa(j)}
			if err := c.Message(rm, ic, path); err != nil {
				return err
			}
		}
	}
	return nil
}

// referencedKeys returns the ICDs that the enabled endpoints of cfg name and icds
// lacks, and the channels that those endpoints name.
func referencedKeys(
	cfg *ReadConfig,
	icds map[icd.Key]icd.ICD,
) (icdKeys []icd.Key, channelKeys []channel.Key) {
	for _, ep := range cfg.Endpoints {
		if !readsICD(ep) {
			continue
		}
		if _, ok := icds[*ep.ICD]; !ok {
			icdKeys = append(icdKeys, *ep.ICD)
		}
		for _, m := range ep.Messages {
			for _, b := range m.Bindings {
				if b.Channel != 0 {
					channelKeys = append(channelKeys, b.Channel)
				}
			}
		}
	}
	return icdKeys, channelKeys
}
