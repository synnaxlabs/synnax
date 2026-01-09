// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "client/cpp/arc/arc.h"
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

    void use(const std::shared_ptr<freighter::Middleware> &) const;

    std::unique_ptr<AuthLoginClient> auth_login;
    std::unique_ptr<framer::StreamerClient> frame_stream;
    std::unique_ptr<framer::WriterClient> frame_write;
    std::shared_ptr<channel::CreateClient> chan_create;
    std::shared_ptr<channel::RetrieveClient> chan_retrieve;
    std::unique_ptr<RetrieveClient> range_retrieve;
    std::unique_ptr<CreateClient> range_create;
    std::shared_ptr<RangeKVDeleteClient> range_kv_delete;
    std::shared_ptr<RangeKVGetClient> range_kv_get;
    std::shared_ptr<RangeKVSetClient> range_kv_set;
    std::unique_ptr<rack::CreateClient> rack_create_client;
    std::unique_ptr<rack::RetrieveClient> rack_retrieve;
    std::unique_ptr<rack::DeleteClient> rack_delete;
    std::shared_ptr<task::CreateClient> module_create;
    std::shared_ptr<task::RetrieveClient> module_retrieve;
    std::shared_ptr<task::DeleteClient> module_delete;
    std::unique_ptr<device::CreateClient> device_create;
    std::unique_ptr<device::RetrieveClient> device_retrieve;
    std::unique_ptr<device::DeleteClient> device_delete;
    std::shared_ptr<status::RetrieveClient> status_retrieve;
    std::shared_ptr<status::SetClient> status_set;
    std::shared_ptr<status::DeleteClient> status_delete;
    std::shared_ptr<arc::CreateClient> arc_create;
    std::shared_ptr<arc::RetrieveClient> arc_retrieve;
    std::shared_ptr<arc::DeleteClient> arc_delete;
};
}
