// Copyright 2025 Synnax Labs, Inc.
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
	dcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
)

func newStorageResponseTranslator(
	host dcore.NodeKey,
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

func newStorageRequestTranslator() func(ctx context.Context, in Request) (ts.IteratorRequest, bool, error) {
	return func(ctx context.Context, req Request) (ts.IteratorRequest, bool, error) {
		return ts.IteratorRequest{
			Command: ts.IteratorCommand(req.Command),
			Span:    req.Span,
			Stamp:   req.Stamp,
			Bounds:  req.Bounds,
		}, true, nil
	}
}
