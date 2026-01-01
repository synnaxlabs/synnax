// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iterator

import (
	"context"

	"github.com/synnaxlabs/x/confluence"
)

type broadcaster struct {
	confluence.DeltaTransformMultiplier[Request, Request]
	seqNum int
}

func newBroadcaster() *broadcaster {
	b := &broadcaster{}
	b.Transform = b.transform
	return b
}

func (b *broadcaster) transform(ctx context.Context, in Request) (out Request, ok bool, err error) {
	out = in
	out.SeqNum = b.seqNum
	b.seqNum++
	return out, true, nil
}
