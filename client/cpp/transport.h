// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "client/cpp/auth/auth.h"
#include "client/cpp/channel/channel.h"
#include "client/cpp/device/device.h"
#include "client/cpp/framer/framer.h"
#include "client/cpp/rack/rack.h"
#include "client/cpp/ranger/ranger.h"
#include "client/cpp/status/status.h"

namespace synnax {
struct Transport {
    static Transport configure(
        uint16_t port,
        const std::string &ip,
        const std::string &ca_cert_file,
        const std::string &client_cert_file,
        const std::string &client_key_file
    );

    void use(const std::shared_ptr<freighter::Middleware> &mw) const;

    std::unique_ptr<AuthLoginClient> auth_login;
    std::unique_ptr<StreamerClient> frame_stream;
    std::unique_ptr<WriterClient> frame_write;
    std::shared_ptr<ChannelCreateClient> chan_create;
    std::shared_ptr<ChannelRetrieveClient> chan_retrieve;
    std::unique_ptr<RangeRetrieveClient> range_retrieve;
    std::unique_ptr<RangeCreateClient> range_create;
    std::shared_ptr<RangeKVDeleteClient> range_kv_delete;
    std::shared_ptr<RangeKVGetClient> range_kv_get;
    std::shared_ptr<RangeKVSetClient> range_kv_set;
    std::unique_ptr<RackCreateClient> rack_create_client;
    std::unique_ptr<RackRetrieveClient> rack_retrieve;
    std::unique_ptr<RackDeleteClient> rack_delete;
    std::shared_ptr<TaskCreateClient> module_create;
    std::shared_ptr<TaskRetrieveClient> module_retrieve;
    std::shared_ptr<TaskDeleteClient> module_delete;
    std::unique_ptr<DeviceCreateClient> device_create;
    std::unique_ptr<DeviceRetrieveClient> device_retrieve;
    std::unique_ptr<DeviceDeleteClient> device_delete;
    std::shared_ptr<StatusRetrieveClient> status_retrieve;
    std::shared_ptr<StatusSetClient> status_set;
    std::shared_ptr<StatusDeleteClient> status_delete;
};
}
