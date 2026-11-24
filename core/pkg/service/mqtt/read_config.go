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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/read"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	icdv1 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v2"
	"github.com/synnaxlabs/x/content"
)

// createICD returns the v2.CreateICD that lifts an ICD to the current shape and creates
// it with svc.
func createICD(svc *icd.Service) v2.CreateICD {
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
	channels icdv0.Channels
	reads    read.Resolver
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

// hash writes to h the payload of each ICD message that an enabled plain entry of cfg
// reads, so an edit to the format of one changes the config hash of the task.
func (r readResolver) hash(
	ctx context.Context,
	snap gorp.Snapshot,
	cfg *ReadConfig,
	h *content.Hash,
) error {
	return r.reads.Hash(ctx, snap, h, readsOf(cfg))
}

// check checks each enabled entry of cfg, in order, against its ICD, from icds or else
// read through snap, and its channels.
func (r readResolver) check(
	ctx context.Context,
	snap gorp.Snapshot,
	cfg *ReadConfig,
	icds map[icd.Key]icd.ICD,
) error {
	c, err := r.reads.NewChecker(ctx, snap, readsOf(cfg), icds)
	if err != nil {
		return err
	}
	for i, e := range cfg.Entries {
		switch entry := e.Variant.(type) {
		case PlainReadEntry:
			if !entry.Disabled {
				err = c.Read(readOf(i, entry))
			}
		case SparkplugReadEntry:
			if !entry.Disabled && entry.Channel != 0 {
				path := []string{"entries", strconv.Itoa(i), "channel"}
				err = c.Claim(entry.Channel, path)
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// readsOf returns the read of each enabled plain entry of cfg.
func readsOf(cfg *ReadConfig) []read.Read {
	var reads []read.Read
	for i, e := range cfg.Entries {
		if entry, ok := e.Variant.(PlainReadEntry); ok && !entry.Disabled {
			reads = append(reads, readOf(i, entry))
		}
	}
	return reads
}

// readOf returns the read of e, the plain entry at i of a config.
func readOf(i int, e PlainReadEntry) read.Read {
	return read.Read{
		Path:     []string{"entries", strconv.Itoa(i)},
		ICD:      e.ICD,
		Messages: e.Messages,
	}
}
