// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package grpc

import (
	"context"
	"github.com/synnaxlabs/freighter/fgrpc"
	echov1 "github.com/synnaxlabs/freighter/integration/grpc/gen/proto/go/v1"
	"github.com/synnaxlabs/freighter/integration/http"
)

type echoMessageTranslator struct{}

var _ fgrpc.Translator[http.Message, *echov1.Message] = (*echoMessageTranslator)(nil)

func (e echoMessageTranslator) Forward(ctx context.Context, msg http.Message) (*echov1.Message, error) {
	return &echov1.Message{
		Id:      uint32(msg.ID),
		Message: msg.Message,
	}, nil
}

func (e echoMessageTranslator) Backward(ctx context.Context, msg *echov1.Message) (http.Message, error) {
	return http.Message{
		ID:      int(msg.Id),
		Message: msg.Message,
	}, nil
}
