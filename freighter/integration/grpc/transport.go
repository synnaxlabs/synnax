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
	"github.com/synnaxlabs/freighter/fgrpc"
	echov1 "github.com/synnaxlabs/freighter/integration/grpc/gen/proto/go/v1"
	"github.com/synnaxlabs/freighter/integration/http"
)

type Server = fgrpc.UnaryServer[
	http.Message,
	*echov1.Message,
	http.Message,
	*echov1.Message,
]

var _ echov1.EchoServiceServer = (*Server)(nil)

func New() *Server {
	return &Server{
		RequestTranslator:  echoMessageTranslator{},
		ResponseTranslator: echoMessageTranslator{},
		ServiceDesc:        &echov1.EchoService_ServiceDesc,
	}
}
