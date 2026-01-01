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

	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
)

func newStorageResponseTranslator(
	host cluster.NodeKey,
) func(ctx context.Context, in ts.IteratorResponse) (Response, bool, error) {
	return func(ctx context.Context, res ts.IteratorResponse) (Response, bool, error) {
		return Response{
			Ack:     res.Ack,
			Variant: ResponseVariant(res.Variant),
			SeqNum:  res.SeqNum,
			NodeKey: host,
			Error:   res.Err,
			Command: Command(res.Command),
			Frame:   core.NewFrameFromStorage(res.Frame),
		}, true, nil
	}
}

func newStorageRequestTranslator(generateSeqNums bool) func(ctx context.Context, in Request) (ts.IteratorRequest, bool, error) {
	seqNum := 0
	return func(ctx context.Context, req Request) (ts.IteratorRequest, bool, error) {
		oReq := ts.IteratorRequest{
			Command: ts.IteratorCommand(req.Command),
			Span:    req.Span,
			Stamp:   req.Stamp,
			Bounds:  req.Bounds,
			SeqNum:  req.SeqNum,
		}
		if generateSeqNums {
			seqNum++
			oReq.SeqNum = seqNum
		}
		return oReq, true, nil
	}
}
