// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std.
#include <memory>

/// internal
#include "synnax/framer/framer.h"
#include "synnax/ranger/ranger.h"
#include "synnax/channel/channel.h"
#include "synnax/transport.h"

using namespace synnax;


namespace synnax {

// @brief Configuration for opening a Synnax login_client.
struct Config {
    // @brief the host of a node in the cluster.
    std::string host;
    // @brief the port for the specified host.
    std::uint16_t port;
    // @brief whether to use TLS when connecting to the host. Only works when the node is running in secure mode.
    bool secure;
    // @brief the username to use when authenticating with the node.
    std::string username;
    // @brief the password to use when authenticating with the node.
    std::string password;
};

const std::string ERROR_PREFIX = "sy.api.";
const std::string VALIDATION_ERROR = ERROR_PREFIX + "validation";
const std::string QUERY_ERROR = ERROR_PREFIX + "query";

class Client {
public:
    ChannelClient channels = ChannelClient(nullptr, nullptr);
    RangeClient ranges = RangeClient(nullptr, nullptr, nullptr, nullptr, nullptr);
    FrameClient telem = FrameClient(nullptr, nullptr);

    explicit Client(const Config &cfg) {
        auto t = Transport(cfg.port, cfg.host);
        // TODO: fix this memory leak.
        freighter::Middleware *auth_mw = new AuthMiddleware(t.auth_login, cfg.username, cfg.password);
        t.use(auth_mw);
        channels = ChannelClient(t.chan_retrieve, t.chan_create);
        ranges = RangeClient(
                t.range_retrieve,
                t.range_create,
                t.range_kv_get,
                t.range_kv_set,
                t.range_kv_delete
        );
        telem = FrameClient(t.frame_stream, t.frame_write);
    }
};
}