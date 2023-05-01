// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package relay

import (
	"context"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/relay"
	rv1 "github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/gen/proto/go/relay/v1"
)

type relayRequestTranslator struct{}

func (r relayRequestTranslator) Forward(
	_ context.Context,
	req *rv1.RelayRequest,
) (relay.ReadRequest, error) {
	return relay.ReadRequest{
		Keys: lo.Map(
			req.Keys,
			func(k uint32, _ int) channel.Key { return channel.Key(k) }),
	}, nil
}

func (r relayRequestTranslator) Backward(
	_ context.Context,
	req relay.ReadRequest,
) (*rv1.RelayRequest, error) {
	return &rv1.RelayRequest{
		Keys: lo.Map(
			req.Keys,
			func(k channel.Key, _ int) uint32 { return uint32(k) }),
	}, nil
}

type relayResponseTranslator struct{}

func (r relayResponseTranslator) Forward(
	_ context.Context,
	resp relay.ReadResponse,
) (*rv1.RelayResponse, error) {
	return &rv1.RelayResponse{
		Frame: resp.Frame,
	}, nil
}
