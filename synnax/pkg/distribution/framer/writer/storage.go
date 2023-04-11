// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/storage"
)

func newRequestTranslator() func(ctx context.Context, in Request) (cesium.WriteRequest, bool, error) {
	return func(ctx context.Context, in Request) (storage.TSWriteRequest, bool, error) {
		return cesium.WriteRequest{
			Command: cesium.WriterCommand(in.Command), Frame: in.Frame.ToStorage(),
		}, true, nil
	}
}

func newResponseTranslator(host core.NodeKey) func(ctx context.Context, in cesium.WriteResponse) (Response, bool, error) {
	return func(ctx context.Context, in storage.TSWriteResponse) (Response, bool, error) {
		return Response{
			Command: Command(in.Command),
			Ack:     in.Ack,
			Err:     in.Err,
			SeqNum:  in.SeqNum,
			NodeKey:  host,
		}, true, nil
	}
}
