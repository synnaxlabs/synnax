// Copyright 2023 Synnax Labs, Inc.
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
	kvx "github.com/synnaxlabs/x/kv"
)

type persist struct {
	bw kvx.BatchWriter
	confluence.LinearTransform[BatchRequest, BatchRequest]
}

func newPersist(bw kvx.BatchWriter) segment {
	ps := &persist{bw: bw}
	ps.LinearTransform.Transform = ps.persist
	return ps
}

func (ps *persist) persist(_ context.Context, bd BatchRequest) (BatchRequest, bool, error) {
	err := bd.commitTo(ps.bw)
	return bd, err == nil, nil
}
