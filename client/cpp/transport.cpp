// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "google/protobuf/empty.pb.h"

#include "client/cpp/transport.h"
#include "freighter/cpp/grpc/grpc.h"

#include "core/pkg/api/grpc/arc/arc.grpc.pb.h"
#include "core/pkg/api/grpc/arc/arc.pb.h"
#include "core/pkg/api/grpc/auth/auth.grpc.pb.h"
#include "core/pkg/api/grpc/auth/auth.pb.h"
#include "core/pkg/api/grpc/channel/channel.grpc.pb.h"
#include "core/pkg/api/grpc/channel/channel.pb.h"
#include "core/pkg/api/grpc/device/device.grpc.pb.h"
#include "core/pkg/api/grpc/device/device.pb.h"
#include "core/pkg/api/grpc/framer/framer.grpc.pb.h"
#include "core/pkg/api/grpc/framer/framer.pb.h"
#include "core/pkg/api/grpc/rack/rack.grpc.pb.h"
#include "core/pkg/api/grpc/rack/rack.pb.h"
#include "core/pkg/api/grpc/ranger/kv/kv.grpc.pb.h"
#include "core/pkg/api/grpc/ranger/kv/kv.pb.h"
#include "core/pkg/api/grpc/ranger/ranger.grpc.pb.h"
#include "core/pkg/api/grpc/ranger/ranger.pb.h"
#include "core/pkg/api/grpc/status/status.grpc.pb.h"
#include "core/pkg/api/grpc/status/status.pb.h"
#include "core/pkg/api/grpc/task/task.grpc.pb.h"
#include "core/pkg/api/grpc/task/task.pb.h"

namespace synnax::details {
Transport::Transport(
    const uint16_t port,
    const std::string &ip,
    const std::string &ca_cert_file,
    const std::string &client_cert_file,
    const std::string &client_key_file
) {
    auto base_target = x::url::URL(ip, port, "").to_string();
    auto pool = std::make_shared<freighter::grpc::Pool>(
        ca_cert_file,
        client_cert_file,
        client_key_file
    );
    this->auth_login = std::make_unique<freighter::grpc::UnaryClient<
        grpc::auth::LoginRequest,
        grpc::auth::LoginResponse,
        grpc::auth::AuthLoginService>>(pool, base_target);
    this->frame_stream = std::make_unique<freighter::grpc::StreamClient<
        grpc::framer::StreamerRequest,
        grpc::framer::StreamerResponse,
        grpc::framer::FrameStreamerService>>(pool, base_target);
    this->frame_write = std::make_unique<freighter::grpc::StreamClient<
        grpc::framer::WriterRequest,
        grpc::framer::WriterResponse,
        grpc::framer::FrameWriterService>>(pool, base_target);
    this->chan_create = std::make_unique<freighter::grpc::UnaryClient<
        grpc::channel::CreateRequest,
        grpc::channel::CreateResponse,
        grpc::channel::ChannelCreateService>>(pool, base_target);
    this->chan_retrieve = std::make_unique<freighter::grpc::UnaryClient<
        grpc::channel::RetrieveRequest,
        grpc::channel::RetrieveResponse,
        grpc::channel::ChannelRetrieveService>>(pool, base_target);
    this->range_retrieve = std::make_unique<freighter::grpc::UnaryClient<
        grpc::ranger::RetrieveRequest,
        grpc::ranger::RetrieveResponse,
        grpc::ranger::RangeRetrieveService>>(pool, base_target);
    this->range_create = std::make_unique<freighter::grpc::UnaryClient<
        grpc::ranger::CreateRequest,
        grpc::ranger::CreateResponse,
        grpc::ranger::RangeCreateService>>(pool, base_target);
    this->kv_delete = std::make_shared<freighter::grpc::UnaryClient<
        grpc::kv::DeleteRequest,
        google::protobuf::Empty,
        grpc::kv::KVDeleteService>>(pool, base_target);
    this->kv_get = std::make_shared<freighter::grpc::UnaryClient<
        grpc::kv::GetRequest,
        grpc::kv::GetResponse,
        grpc::kv::KVGetService>>(pool, base_target);
    this->kv_set = std::make_shared<freighter::grpc::UnaryClient<
        grpc::kv::SetRequest,
        google::protobuf::Empty,
        grpc::kv::KVSetService>>(pool, base_target);
    this->rack_create_client = std::make_unique<freighter::grpc::UnaryClient<
        grpc::rack::CreateRequest,
        grpc::rack::CreateResponse,
        grpc::rack::RackCreateService>>(pool, base_target);
    this->rack_retrieve = std::make_unique<freighter::grpc::UnaryClient<
        grpc::rack::RetrieveRequest,
        grpc::rack::RetrieveResponse,
        grpc::rack::RackRetrieveService>>(pool, base_target);
    this->rack_delete = std::make_unique<freighter::grpc::UnaryClient<
        grpc::rack::DeleteRequest,
        google::protobuf::Empty,
        grpc::rack::RackDeleteService>>(pool, base_target);
    this->task_create = std::make_shared<freighter::grpc::UnaryClient<
        grpc::task::CreateRequest,
        grpc::task::CreateResponse,
        grpc::task::TaskCreateService>>(pool, base_target);
    this->task_retrieve = std::make_shared<freighter::grpc::UnaryClient<
        grpc::task::RetrieveRequest,
        grpc::task::RetrieveResponse,
        grpc::task::TaskRetrieveService>>(pool, base_target);
    this->task_delete = std::make_shared<freighter::grpc::UnaryClient<
        grpc::task::DeleteRequest,
        google::protobuf::Empty,
        grpc::task::TaskDeleteService>>(pool, base_target);
    this->device_create = std::make_unique<freighter::grpc::UnaryClient<
        grpc::device::CreateRequest,
        grpc::device::CreateResponse,
        grpc::device::DeviceCreateService>>(pool, base_target);
    this->device_retrieve = std::make_unique<freighter::grpc::UnaryClient<
        grpc::device::RetrieveRequest,
        grpc::device::RetrieveResponse,
        grpc::device::DeviceRetrieveService>>(pool, base_target);
    this->device_delete = std::make_unique<freighter::grpc::UnaryClient<
        grpc::device::DeleteRequest,
        google::protobuf::Empty,
        grpc::device::DeviceDeleteService>>(pool, base_target);
    this->status_retrieve = std::make_shared<freighter::grpc::UnaryClient<
        grpc::status::RetrieveRequest,
        grpc::status::RetrieveResponse,
        grpc::status::StatusRetrieveService>>(pool, base_target);
    this->status_set = std::make_shared<freighter::grpc::UnaryClient<
        grpc::status::SetRequest,
        grpc::status::SetResponse,
        grpc::status::StatusSetService>>(pool, base_target);
    this->status_delete = std::make_shared<freighter::grpc::UnaryClient<
        grpc::status::DeleteRequest,
        google::protobuf::Empty,
        grpc::status::StatusDeleteService>>(pool, base_target);
    this->arc_create = std::make_shared<freighter::grpc::UnaryClient<
        grpc::arc::CreateRequest,
        grpc::arc::CreateResponse,
        grpc::arc::ArcCreateService>>(pool, base_target);
    this->arc_retrieve = std::make_shared<freighter::grpc::UnaryClient<
        grpc::arc::RetrieveRequest,
        grpc::arc::RetrieveResponse,
        grpc::arc::ArcRetrieveService>>(pool, base_target);
    this->arc_delete = std::make_shared<freighter::grpc::UnaryClient<
        grpc::arc::DeleteRequest,
        google::protobuf::Empty,
        grpc::arc::ArcDeleteService>>(pool, base_target);
};

void Transport::use(const std::shared_ptr<freighter::Middleware> &mw) const {
    frame_stream->use(mw);
    frame_write->use(mw);
    chan_create->use(mw);
    chan_retrieve->use(mw);
    range_retrieve->use(mw);
    range_create->use(mw);
    kv_delete->use(mw);
    kv_get->use(mw);
    kv_set->use(mw);
    rack_create_client->use(mw);
    rack_retrieve->use(mw);
    rack_delete->use(mw);
    task_create->use(mw);
    task_retrieve->use(mw);
    task_delete->use(mw);
    device_create->use(mw);
    device_retrieve->use(mw);
    device_delete->use(mw);
    status_retrieve->use(mw);
    status_set->use(mw);
    status_delete->use(mw);
    arc_create->use(mw);
    arc_retrieve->use(mw);
    arc_delete->use(mw);
}
}
