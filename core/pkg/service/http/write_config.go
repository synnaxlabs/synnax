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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	v5 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v5"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
)

// writeResolver resolves write configs against the ICDs and channels they write.
type writeResolver struct {
	icds     *icd.Service
	channels icdv0.Channels
	ins      alamos.Instrumentation
}

// resolve moves the staged channel and fields of cfg, which a config from before ICDs
// holds, into a new ICD named name, and returns the write that creates that ICD.
func (r writeResolver) resolve(
	ctx context.Context,
	snap gorp.Snapshot,
	name string,
	cfg *WriteConfig,
) (func(context.Context, gorp.Tx) error, error) {
	ic, ok, err := v5.ResolveWriteConfig(
		ctx, snap, r.channels, name, cfg, v5.NoteWriteLift(r.ins, cfg.Key),
	)
	if err != nil || !ok {
		return nil, err
	}
	return func(ctx context.Context, tx gorp.Tx) error {
		return r.icds.NewWriter(tx).Create(ctx, &ic)
	}, nil
}
