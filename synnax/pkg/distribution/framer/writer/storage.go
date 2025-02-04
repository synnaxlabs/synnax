// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
)

func newRequestTranslator() func(ctx context.Context, in Request) (ts.WriterRequest, bool, error) {
	return func(ctx context.Context, in Request) (ts.WriterRequest, bool, error) {
		return ts.WriterRequest{
			Command: ts.WriterCommand(in.Command),
			Frame:   in.Frame.ToStorage(),
			Config:  in.Config.toStorage(),
		}, true, nil
	}
}

func newResponseTranslator(host core.NodeKey) func(ctx context.Context, in ts.WriterResponse) (Response, bool, error) {
	return func(ctx context.Context, in ts.WriterResponse) (Response, bool, error) {
		return Response{
			Command: Command(in.Command),
			Ack:     in.Ack,
			Error:   in.Err,
			SeqNum:  in.SeqNum,
			NodeKey: host,
			End:     in.End,
		}, true, nil
	}
}
