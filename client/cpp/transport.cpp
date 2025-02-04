// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "client/cpp/transport.h"

#include "freighter/cpp/fgrpc/fgrpc.h"
#include "google/protobuf/empty.pb.h"
#include "synnax/pkg/api/grpc/v1/synnax/pkg/api/grpc/v1/auth.grpc.pb.h"
#include "synnax/pkg/api/grpc/v1/synnax/pkg/api/grpc/v1/auth.pb.h"
#include "synnax/pkg/api/grpc/v1/synnax/pkg/api/grpc/v1/channel.grpc.pb.h"
#include "synnax/pkg/api/grpc/v1/synnax/pkg/api/grpc/v1/channel.pb.h"
#include "synnax/pkg/api/grpc/v1/synnax/pkg/api/grpc/v1/framer.grpc.pb.h"
#include "synnax/pkg/api/grpc/v1/synnax/pkg/api/grpc/v1/framer.pb.h"
#include "synnax/pkg/api/grpc/v1/synnax/pkg/api/grpc/v1/hardware.grpc.pb.h"
#include "synnax/pkg/api/grpc/v1/synnax/pkg/api/grpc/v1/hardware.pb.h"
#include "synnax/pkg/api/grpc/v1/synnax/pkg/api/grpc/v1/ranger.grpc.pb.h"
#include "synnax/pkg/api/grpc/v1/synnax/pkg/api/grpc/v1/ranger.pb.h"

using namespace api;

Transport::Transport(
    uint16_t port,
    const std::string &ip,
    const std::string &ca_cert_file,
    const std::string &client_cert_file,
    const std::string &client_key_file
) {
    auto base_target = freighter::URL(ip, port, "").toString();
    std::shared_ptr<fgrpc::Pool> pool = nullptr;
    if (ca_cert_file.empty() && client_cert_file.empty() && client_key_file.empty()) {
        pool = std::make_shared<fgrpc::Pool>();
    } else if (client_cert_file.empty() && client_key_file.empty()) {
        pool = std::make_shared<fgrpc::Pool>(ca_cert_file);
    } else {
        pool = std::make_shared<fgrpc::Pool>(ca_cert_file, client_cert_file,
                                             client_key_file);
    }


    auth_login = std::make_unique<fgrpc::UnaryClient<
        v1::LoginRequest,
        v1::LoginResponse,
        v1::AuthLoginService
    > >(pool, base_target);


    frame_stream = std::make_unique<fgrpc::StreamClient<
        v1::FrameStreamerRequest,
        v1::FrameStreamerResponse,
        v1::FrameStreamerService
    > >(pool, base_target);

    frame_write = std::make_unique<fgrpc::StreamClient<
        v1::FrameWriterRequest,
        v1::FrameWriterResponse,
        v1::FrameWriterService
    > >(pool, base_target);

    chan_create = std::make_unique<fgrpc::UnaryClient<
        v1::ChannelCreateRequest,
        v1::ChannelCreateResponse,
        v1::ChannelCreateService
    > >(pool, base_target);

    chan_retrieve = std::make_unique<fgrpc::UnaryClient<
        v1::ChannelRetrieveRequest,
        v1::ChannelRetrieveResponse,
        v1::ChannelRetrieveService
    > >(pool, base_target);

    range_retrieve = std::make_unique<fgrpc::UnaryClient<
        v1::RangeRetrieveRequest,
        v1::RangeRetrieveResponse,
        v1::RangeRetrieveService
    > >(pool, base_target);

    range_create = std::make_unique<fgrpc::UnaryClient<
        v1::RangeCreateRequest,
        v1::RangeCreateResponse,
        v1::RangeCreateService
    > >(pool, base_target);

    range_kv_delete = std::make_shared<fgrpc::UnaryClient<
        v1::RangeKVDeleteRequest,
        google::protobuf::Empty,
        v1::RangeKVDeleteService
    > >(pool, base_target);

    range_kv_get = std::make_shared<fgrpc::UnaryClient<
        v1::RangeKVGetRequest,
        v1::RangeKVGetResponse,
        v1::RangeKVGetService
    > >(pool, base_target);

    range_kv_set = std::make_shared<fgrpc::UnaryClient<
        v1::RangeKVSetRequest,
        google::protobuf::Empty,
        v1::RangeKVSetService
    > >(pool, base_target);

    rack_create_client = std::make_unique<fgrpc::UnaryClient<
        v1::HardwareCreateRackRequest,
        v1::HardwareCreateRackResponse,
        v1::HardwareCreateRackService
    > >(pool, base_target);

    rack_retrieve = std::make_unique<fgrpc::UnaryClient<
        v1::HardwareRetrieveRackRequest,
        v1::HardwareRetrieveRackResponse,
        v1::HardwareRetrieveRackService
    > >(pool, base_target);

    rack_delete = std::make_unique<fgrpc::UnaryClient<
        v1::HardwareDeleteRackRequest,
        google::protobuf::Empty,
        v1::HardwareDeleteRackService
    > >(pool, base_target);

    module_create = std::make_shared<fgrpc::UnaryClient<
        v1::HardwareCreateTaskRequest,
        v1::HardwareCreateTaskResponse,
        v1::HardwareCreateTaskService
    > >(pool, base_target);

    module_retrieve = std::make_shared<fgrpc::UnaryClient<
        v1::HardwareRetrieveTaskRequest,
        v1::HardwareRetrieveTaskResponse,
        v1::HardwareRetrieveTaskService
    > >(pool, base_target);

    module_delete = std::make_shared<fgrpc::UnaryClient<
        v1::HardwareDeleteTaskRequest,
        google::protobuf::Empty,
        v1::HardwareDeleteTaskService
    > >(pool, base_target);

    device_create = std::make_unique<fgrpc::UnaryClient<
        v1::HardwareCreateDeviceRequest,
        v1::HardwareCreateDeviceResponse,
        v1::HardwareCreateDeviceService
    > >(pool, base_target);

    device_retrieve = std::make_unique<fgrpc::UnaryClient<
        v1::HardwareRetrieveDeviceRequest,
        v1::HardwareRetrieveDeviceResponse,
        v1::HardwareRetrieveDeviceService
    > >(pool, base_target);

    device_delete = std::make_unique<fgrpc::UnaryClient<
        v1::HardwareDeleteDeviceRequest,
        google::protobuf::Empty,
        v1::HardwareDeleteDeviceService
    > >(pool, base_target);
}

void Transport::use(const std::shared_ptr<freighter::Middleware> &mw) const {
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
}
