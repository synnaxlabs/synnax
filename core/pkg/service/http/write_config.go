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
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/icd/write"
	"github.com/synnaxlabs/x/content"
	"github.com/synnaxlabs/x/validate"
)

// writeResolver resolves write configs against the ICDs and channels they write.
type writeResolver struct {
	icds     *icd.Service
	channels icdv0.Channels
	writes   write.Resolver
	ins      alamos.Instrumentation
}

// resolve moves the staged channel and fields of cfg, which a config from before ICDs
// holds, into a new ICD named name. It then checks each enabled endpoint of cfg against
// its ICD and its channels, read through snap. It returns the write that creates the
// ICD, or nil when cfg holds nothing staged.
func (r writeResolver) resolve(
	ctx context.Context,
	snap gorp.Snapshot,
	name string,
	cfg *WriteConfig,
) (func(context.Context, gorp.Tx) error, error) {
	ic, ok, err := v5.ResolveWriteConfig(
		ctx, snap, r.channels, name, cfg, v5.NoteWriteLift(r.ins, cfg.Key),
	)
	if err != nil {
		return nil, err
	}
	writes, err := writesOf(cfg)
	if err != nil {
		return nil, err
	}
	icds := make(map[icd.Key]icd.ICD)
	if ok {
		icds[ic.Key] = ic
	}
	if err = r.writes.Check(ctx, snap, writes, icds); err != nil || !ok {
		return nil, err
	}
	return func(ctx context.Context, tx gorp.Tx) error {
		return r.icds.NewWriter(tx).Create(ctx, &ic)
	}, nil
}

// hash writes to h the payload of the ICD message that each enabled endpoint of cfg
// sends, so an edit to the format of one changes the config hash of the task.
func (r writeResolver) hash(
	ctx context.Context,
	snap gorp.Snapshot,
	cfg *WriteConfig,
	h *content.Hash,
) error {
	writes, err := writesOf(cfg)
	if err != nil {
		return err
	}
	return r.writes.Hash(ctx, snap, h, writes)
}

// writesOf returns the write of each enabled endpoint of cfg that names an ICD. It
// returns a validation error for an endpoint that names only one of an ICD and a
// message.
func writesOf(cfg *WriteConfig) ([]write.Write, error) {
	var writes []write.Write
	for i, ep := range cfg.Endpoints {
		path := []string{"endpoints", strconv.Itoa(i)}
		switch {
		case ep.ICD != nil && ep.Message == nil:
			return nil, validate.PathedError(
				validate.Errorf("required for an endpoint with an ICD"),
				append(path, "message")...,
			)
		case ep.ICD == nil && ep.Message != nil:
			return nil, validate.PathedError(
				validate.Errorf("required for an endpoint with a message"),
				append(path, "icd")...,
			)
		case ep.ICD != nil && !ep.Disabled:
			writes = append(writes, write.Write{
				Path:    path,
				ICD:     *ep.ICD,
				Message: *ep.Message,
			})
		}
	}
	return writes, nil
}
