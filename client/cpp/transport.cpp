// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "google/protobuf/empty.pb.h"

#include "client/cpp/transport.h"
#include "freighter/cpp/fgrpc/fgrpc.h"

#include "core/pkg/api/grpc/v1/core/pkg/api/grpc/v1/auth.grpc.pb.h"
#include "core/pkg/api/grpc/v1/core/pkg/api/grpc/v1/auth.pb.h"
#include "core/pkg/api/grpc/v1/core/pkg/api/grpc/v1/channel.grpc.pb.h"
#include "core/pkg/api/grpc/v1/core/pkg/api/grpc/v1/channel.pb.h"
#include "core/pkg/api/grpc/v1/core/pkg/api/grpc/v1/framer.grpc.pb.h"
#include "core/pkg/api/grpc/v1/core/pkg/api/grpc/v1/framer.pb.h"
#include "core/pkg/api/grpc/v1/core/pkg/api/grpc/v1/hardware.grpc.pb.h"
#include "core/pkg/api/grpc/v1/core/pkg/api/grpc/v1/hardware.pb.h"
#include "core/pkg/api/grpc/v1/core/pkg/api/grpc/v1/ranger.grpc.pb.h"
#include "core/pkg/api/grpc/v1/core/pkg/api/grpc/v1/ranger.pb.h"
#include "core/pkg/api/grpc/v1/core/pkg/api/grpc/v1/status.grpc.pb.h"
#include "core/pkg/api/grpc/v1/core/pkg/api/grpc/v1/status.pb.h"

synnax::Transport synnax::Transport::configure(
    const uint16_t port,
    const std::string &ip,
    const std::string &ca_cert_file,
    const std::string &client_cert_file,
    const std::string &client_key_file
) {
    auto base_target = freighter::URL(ip, port, "").to_string();
    auto pool = std::make_shared<fgrpc::Pool>(
        ca_cert_file,
        client_cert_file,
        client_key_file
    );
    return Transport{
        .auth_login = std::make_unique<fgrpc::UnaryClient<
            api::v1::LoginRequest,
            api::v1::LoginResponse,
            api::v1::AuthLoginService>>(pool, base_target),
        .frame_stream = std::make_unique<fgrpc::StreamClient<
            api::v1::FrameStreamerRequest,
            api::v1::FrameStreamerResponse,
            api::v1::FrameStreamerService>>(pool, base_target),
        .frame_write = std::make_unique<fgrpc::StreamClient<
            api::v1::FrameWriterRequest,
            api::v1::FrameWriterResponse,
            api::v1::FrameWriterService>>(pool, base_target),
        .chan_create = std::make_unique<fgrpc::UnaryClient<
            api::v1::ChannelCreateRequest,
            api::v1::ChannelCreateResponse,
            api::v1::ChannelCreateService>>(pool, base_target),
        .chan_retrieve = std::make_unique<fgrpc::UnaryClient<
            api::v1::ChannelRetrieveRequest,
            api::v1::ChannelRetrieveResponse,
            api::v1::ChannelRetrieveService>>(pool, base_target),
        .range_retrieve = std::make_unique<fgrpc::UnaryClient<
            api::v1::RangeRetrieveRequest,
            api::v1::RangeRetrieveResponse,
            api::v1::RangeRetrieveService>>(pool, base_target),
        .range_create = std::make_unique<fgrpc::UnaryClient<
            api::v1::RangeCreateRequest,
            api::v1::RangeCreateResponse,
            api::v1::RangeCreateService>>(pool, base_target),
        .range_kv_delete = std::make_shared<fgrpc::UnaryClient<
            api::v1::RangeKVDeleteRequest,
            google::protobuf::Empty,
            api::v1::RangeKVDeleteService>>(pool, base_target),
        .range_kv_get = std::make_shared<fgrpc::UnaryClient<
            api::v1::RangeKVGetRequest,
            api::v1::RangeKVGetResponse,
            api::v1::RangeKVGetService>>(pool, base_target),
        .range_kv_set = std::make_shared<fgrpc::UnaryClient<
            api::v1::RangeKVSetRequest,
            google::protobuf::Empty,
            api::v1::RangeKVSetService>>(pool, base_target),
        .rack_create_client = std::make_unique<fgrpc::UnaryClient<
            api::v1::HardwareCreateRackRequest,
            api::v1::HardwareCreateRackResponse,
            api::v1::HardwareCreateRackService>>(pool, base_target),
        .rack_retrieve = std::make_unique<fgrpc::UnaryClient<
            api::v1::HardwareRetrieveRackRequest,
            api::v1::HardwareRetrieveRackResponse,
            api::v1::HardwareRetrieveRackService>>(pool, base_target),
        .rack_delete = std::make_unique<fgrpc::UnaryClient<
            api::v1::HardwareDeleteRackRequest,
            google::protobuf::Empty,
            api::v1::HardwareDeleteRackService>>(pool, base_target),
        .module_create = std::make_shared<fgrpc::UnaryClient<
            api::v1::HardwareCreateTaskRequest,
            api::v1::HardwareCreateTaskResponse,
            api::v1::HardwareCreateTaskService>>(pool, base_target),
        .module_retrieve = std::make_shared<fgrpc::UnaryClient<
            api::v1::HardwareRetrieveTaskRequest,
            api::v1::HardwareRetrieveTaskResponse,
            api::v1::HardwareRetrieveTaskService>>(pool, base_target),
        .module_delete = std::make_shared<fgrpc::UnaryClient<
            api::v1::HardwareDeleteTaskRequest,
            google::protobuf::Empty,
            api::v1::HardwareDeleteTaskService>>(pool, base_target),
        .device_create = std::make_unique<fgrpc::UnaryClient<
            api::v1::HardwareCreateDeviceRequest,
            api::v1::HardwareCreateDeviceResponse,
            api::v1::HardwareCreateDeviceService>>(pool, base_target),
        .device_retrieve = std::make_unique<fgrpc::UnaryClient<
            api::v1::HardwareRetrieveDeviceRequest,
            api::v1::HardwareRetrieveDeviceResponse,
            api::v1::HardwareRetrieveDeviceService>>(pool, base_target),
        .device_delete = std::make_unique<fgrpc::UnaryClient<
            api::v1::HardwareDeleteDeviceRequest,
            google::protobuf::Empty,
            api::v1::HardwareDeleteDeviceService>>(pool, base_target),
        .status_retrieve = std::make_shared<fgrpc::UnaryClient<
            api::v1::StatusRetrieveRequest,
            api::v1::StatusRetrieveResponse,
            api::v1::StatusRetrieveService>>(pool, base_target),
        .status_set = std::make_shared<fgrpc::UnaryClient<
            api::v1::StatusSetRequest,
            api::v1::StatusSetResponse,
            api::v1::StatusSetService>>(pool, base_target),
        .status_delete = std::make_shared<fgrpc::UnaryClient<
            api::v1::StatusDeleteRequest,
            google::protobuf::Empty,
            api::v1::StatusDeleteService>>(pool, base_target)
    };
}

void synnax::Transport::use(const std::shared_ptr<freighter::Middleware> &mw) const {
    frame_stream->use(mw);
    frame_write->use(mw);
    chan_create->use(mw);
    chan_retrieve->use(mw);
    range_retrieve->use(mw);
    range_create->use(mw);
    range_kv_delete->use(mw);
    range_kv_get->use(mw);
    range_kv_set->use(mw);
    rack_create_client->use(mw);
    rack_retrieve->use(mw);
    rack_delete->use(mw);
    module_create->use(mw);
    module_retrieve->use(mw);
    module_delete->use(mw);
    device_create->use(mw);
    device_retrieve->use(mw);
    device_delete->use(mw);
    status_retrieve->use(mw);
    status_set->use(mw);
    status_delete->use(mw);
}
