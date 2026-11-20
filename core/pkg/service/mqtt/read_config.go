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
	"slices"
	"strconv"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/read"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	v2 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v2"
	"github.com/synnaxlabs/x/content"
)

// createICD returns the v2.CreateICD that creates an ICD with svc.
func createICD(svc *icd.Service) v2.CreateICD {
	return func(ctx context.Context, tx gorp.Tx, ic icd.ICD) error {
		return svc.NewWriter(tx).Create(ctx, &ic)
	}
}

// readResolver resolves read configs against the ICDs and channels they read.
type readResolver struct {
	icds     *icd.Service
	channels icdv0.Channels
	create   v2.CreateICD
	ins      alamos.Instrumentation
}

// resolve moves the inline entries of cfg into a new ICD named name, then checks each
// enabled entry of cfg against its ICD and its channels, read through snap. It returns
// the write that creates the ICD, or nil when cfg holds no inline entries.
func (r readResolver) resolve(
	ctx context.Context,
	snap gorp.Snapshot,
	name string,
	cfg *ReadConfig,
) (func(context.Context, gorp.Tx) error, error) {
	staged, ok, err := v2.ResolveReadConfig(
		ctx, snap, r.channels, name, cfg, v2.NoteLift(r.ins, cfg.Key),
	)
	if err != nil {
		return nil, err
	}
	icds := make(map[icd.Key]icd.ICD)
	if ok {
		icds[staged.Key] = staged
	}
	if err = r.check(ctx, snap, cfg, icds); err != nil || !ok {
		return nil, err
	}
	return func(ctx context.Context, tx gorp.Tx) error {
		return r.create(ctx, tx, staged)
	}, nil
}

// hash writes to h the payload of each ICD message that an enabled plain entry of cfg
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
	for _, e := range cfg.Entries {
		entry, ok := e.Variant.(PlainReadEntry)
		if !ok || entry.Disabled {
			continue
		}
		if err := read.Hash(h, icds[entry.ICD], entry.Messages); err != nil {
			return err
		}
	}
	return nil
}

// check checks each enabled entry of cfg against its ICD, from icds or else read
// through snap, and its channels. It skips an ICD or a channel that does not exist,
// which the reference check of the write reports.
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
	for i, e := range cfg.Entries {
		path := []string{"entries", strconv.Itoa(i)}
		if err := checkEntry(c, icds, e, path); err != nil {
			return err
		}
	}
	return nil
}

// referencedKeys returns the ICDs that the enabled entries of cfg name and icds lacks,
// and the channels that the enabled entries name.
func referencedKeys(
	cfg *ReadConfig,
	icds map[icd.Key]icd.ICD,
) (icdKeys []icd.Key, channelKeys []channel.Key) {
	for _, e := range cfg.Entries {
		switch entry := e.Variant.(type) {
		case PlainReadEntry:
			if entry.Disabled {
				continue
			}
			if _, ok := icds[entry.ICD]; !ok {
				icdKeys = append(icdKeys, entry.ICD)
			}
			for _, m := range entry.Messages {
				for _, b := range m.Bindings {
					if b.Channel != 0 {
						channelKeys = append(channelKeys, b.Channel)
					}
				}
			}
		case SparkplugReadEntry:
			if !entry.Disabled && entry.Channel != 0 {
				channelKeys = append(channelKeys, entry.Channel)
			}
		}
	}
	return icdKeys, channelKeys
}

// checkEntry checks e, at path, with c when it is enabled. It skips a plain entry whose
// ICD icds lacks.
func checkEntry(
	c read.Checker,
	icds map[icd.Key]icd.ICD,
	e ReadEntry,
	path []string,
) error {
	switch entry := e.Variant.(type) {
	case PlainReadEntry:
		ic, ok := icds[entry.ICD]
		if entry.Disabled || !ok {
			return nil
		}
		for j, rm := range entry.Messages {
			messagePath := slices.Concat(path, []string{"messages", strconv.Itoa(j)})
			if err := c.Message(rm, ic, messagePath); err != nil {
				return err
			}
		}
	case SparkplugReadEntry:
		if !entry.Disabled && entry.Channel != 0 {
			return c.Claim(entry.Channel, slices.Concat(path, []string{"channel"}))
		}
	}
	return nil
}
