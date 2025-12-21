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

	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/confluence"
)

func newRequestTranslator() confluence.TransformFunc[Request, ts.WriterRequest] {
	return func(_ context.Context, in Request) (ts.WriterRequest, bool, error) {
		return ts.WriterRequest{
			Command: ts.WriterCommand(in.Command),
			Frame:   in.Frame.ToStorage(),
			Config:  in.Config.toStorage(),
			SeqNum:  in.SeqNum,
		}, true, nil
	}
}

func newResponseTranslator(host cluster.NodeKey) confluence.TransformFunc[ts.WriterResponse, Response] {
	return func(_ context.Context, in ts.WriterResponse) (Response, bool, error) {
		return Response{
			Command:    Command(in.Command),
			SeqNum:     in.SeqNum,
			NodeKey:    host,
			End:        in.End,
			Authorized: in.Authorized,
			Err:        in.Err,
		}, true, nil
	}
}
