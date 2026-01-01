// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv

import (
	"context"
	"github.com/synnaxlabs/x/confluence"
	xkv "github.com/synnaxlabs/x/kv"
)

type persist struct {
	db xkv.Atomic
	confluence.LinearTransform[TxRequest, TxRequest]
}

func newPersist(bw xkv.Atomic) segment {
	ps := &persist{db: bw}
	ps.Transform = ps.persist
	return ps
}

func (ps *persist) persist(_ context.Context, br TxRequest) (TxRequest, bool, error) {
	err := br.commitTo(ps.db)
	return br, err == nil, nil
}
