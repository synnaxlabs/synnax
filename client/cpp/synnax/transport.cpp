// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// freighter
#include "freighter/freighter.h"
#include "freighter/gRPC/client.h"

/// protos and grpc
#include "v1/framer.pb.h"
#include "v1/framer.grpc.pb.h"
#include "v1/ranger.pb.h"
#include "v1/ranger.grpc.pb.h"
#include "v1/channel.pb.h"
#include "v1/channel.grpc.pb.h"
#include "v1/auth.pb.h"
#include "v1/auth.grpc.pb.h"

/// internal
#include "synnax/transport.h"

using namespace api;

Transport::Transport(uint16_t port, const std::string &ip) {
    auto base_target = Freighter::URL(ip, port).toString();
    auto pool = new GRPCPool();

    auth_login = new GRPCUnaryClient<
            v1::LoginResponse,
            v1::LoginRequest,
            grpc::Status,
            v1::AuthLoginService
    >(pool, base_target);


    frame_iter = new GRPCStreamClient<
            v1::FrameIteratorResponse,
            v1::FrameIteratorRequest,
            grpc::Status,
            v1::FrameService
    >(pool, base_target);

    frame_stream = new GRPCStreamClient<
            v1::FrameStreamerResponse,
            v1::FrameStreamerRequest,
            grpc::Status,
            v1::FrameService
    >(pool, base_target);

    frame_write = new GRPCStreamClient<
            v1::FrameWriterResponse,
            v1::FrameWriterRequest,
            grpc::Status,
            v1::FrameService
    >(pool, base_target);

    chan_create = new GRPCUnaryClient<
            v1::ChannelCreateResponse,
            v1::ChannelCreateRequest,
            grpc::Status,
            v1::ChannelCreateService
    >(pool, base_target);

    chan_retrieve = new GRPCUnaryClient<
            v1::ChannelRetrieveResponse,
            v1::ChannelRetrieveRequest,
            grpc::Status,
            v1::ChannelRetrieveService
    >(pool, base_target);

    range_retrieve = new GRPCUnaryClient<
            v1::RangeRetrieveResponse,
            v1::RangeRetrieveRequest,
            grpc::Status,
            v1::RangeRetrieveService
    >(pool, base_target);

    range_create = new GRPCUnaryClient<
            v1::RangeCreateResponse,
            v1::RangeCreateRequest,
            grpc::Status,
            v1::RangeCreateService
    >(pool, base_target);

    range_kv_delete = new GRPCUnaryClient<
            Ranger::Empty,
            v1::RangeKVDeleteRequest,
            grpc::Status,
            v1::RangeKVDeleteService
    >(pool, base_target);

    range_kv_get = new GRPCUnaryClient<
            v1::RangeKVGetResponse,
            v1::RangeKVGetRequest,
            grpc::Status,
            v1::RangeKVGetService
    >(pool, base_target);

    range_kv_set = new GRPCUnaryClient<
            Ranger::Empty,
            v1::RangeKVSetRequest,
            grpc::Status,
            v1::RangeKVSetService
    >(pool, base_target);
}

void Transport::use(Freighter::Middleware *mw) const {
    auth_login->use(mw);
    frame_iter->use(mw);
    frame_stream->use(mw);
    frame_write->use(mw);
    chan_create->use(mw);
    chan_retrieve->use(mw);
    range_retrieve->use(mw);
    range_create->use(mw);
    range_kv_delete->use(mw);
    range_kv_get->use(mw);
    range_kv_set->use(mw);
}
