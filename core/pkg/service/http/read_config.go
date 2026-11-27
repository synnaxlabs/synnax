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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
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
	reads    read.Resolver
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
	if err = r.reads.Check(ctx, snap, readsOf(cfg), icds); err != nil || !ok {
		return nil, err
	}
	return func(ctx context.Context, tx gorp.Tx) error {
		return r.icds.NewWriter(tx).Create(ctx, &ic)
	}, nil
}

// hash writes to h the payload of each ICD message that an enabled endpoint of cfg
// reads, so an edit to the format of one changes the config hash of the task.
func (r readResolver) hash(
	ctx context.Context,
	snap gorp.Snapshot,
	cfg *ReadConfig,
	h *content.Hash,
) error {
	return r.reads.Hash(ctx, snap, h, readsOf(cfg))
}

// readsOf returns the read of each enabled endpoint of cfg that names an ICD.
func readsOf(cfg *ReadConfig) []read.Read {
	var reads []read.Read
	for i, ep := range cfg.Endpoints {
		if !ep.Disabled && ep.ICD != nil {
			reads = append(reads, read.Read{
				Path:     []string{"endpoints", strconv.Itoa(i)},
				ICD:      *ep.ICD,
				Messages: ep.Messages,
			})
		}
	}
	return reads
}
