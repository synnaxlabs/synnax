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
	"github.com/synnaxlabs/x/validate"
)

// readResolver resolves read configs against the ICDs and channels they read.
type readResolver struct {
	icds     *icd.Service
	channels icdv0.Channels
	reads    read.Resolver
	ins      alamos.Instrumentation
}

// resolve moves the inline fields of cfg into a new ICD named name, refuses an endpoint
// that sets fields and reads no messages, then checks each endpoint of cfg against its
// ICD and its channels, read through snap. It returns the write that creates the ICD,
// or nil when cfg holds no inline fields.
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
	if err = refuseFields(cfg); err != nil {
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

// check checks each enabled endpoint of cfg that names an ICD, in order, against that
// ICD, from icds or else read through snap, and its channels.
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

// refuseFields returns a validation error for the first endpoint of cfg that sets
// fields and reads no messages, as a client from before ICDs sends.
func refuseFields(cfg *ReadConfig) error {
	for i, ep := range cfg.Endpoints {
		if len(ep.Fields) > 0 && len(ep.Messages) == 0 {
			return validate.PathedError(validate.Errorf(
				"an endpoint no longer reads fields. Set icd and messages to read the "+
					"response as ICD messages",
			), "endpoints", strconv.Itoa(i), "fields")
		}
	}
	return nil
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
