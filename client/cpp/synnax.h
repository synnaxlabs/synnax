// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>

#include "client/cpp/transport.h"
#include "client/cpp/channel/channel.h"
#include "x/cpp/xerrors/errors.h"
#include "client/cpp/framer/framer.h"
#include "client/cpp/hardware/hardware.h"
#include "client/cpp/ranger/ranger.h"

using namespace synnax;


namespace synnax {
///// @brief Internal namespace. Do not use.
namespace priv {
/// @brief Does a best effort check to ensure the machine is little endian, and warns the user if it is not.
inline void check_little_endian() {
    int num = 1;
    if (*reinterpret_cast<char *>(&num) == 1) return;
    std::cout
            << "WARNING: Detected big endian system, which Synnax does not support. This may silently corrupt telemetry."
            << std::endl;
}
}

/// @brief Configuration for opening a Synnax client.
/// @see Synnax
struct Config {
    /// @brief the host of a node in the cluster.
    std::string host = "localhost";
    /// @brief the port for the specified host.
    std::uint16_t port = 9090;
    /// @brief the username to use when authenticating with the node.
    std::string username = "synnax";
    /// @brief the password to use when authenticating with the node.
    std::string password = "seldon";
    /// @brief path to the CA certificate file to use when connecting to a secure node.
    /// This is only required if the node is configured to use TLS.
    std::string ca_cert_file;
    /// @brief path to the client certificate file to use when connecting to a secure
    /// node and using client authentication. This is not required when in insecure mode
    /// or using username/password authentication.
    std::string client_cert_file;
    /// @brief path to the client key file to use when connecting to a secure node and
    /// using client authentication. This is not required when in insecure mode or using
    /// username/password authentication.
    std::string client_key_file;
};

/// @brief Client to perform operations against a Synnax cluster.
class Synnax {
public:
    /// @brief Client for creating and retrieving channels in a cluster.
    ChannelClient channels = ChannelClient(nullptr, nullptr);
    /// @brief Client for creating, retrieving, and performing operations on ranges in a cluster.
    RangeClient ranges = RangeClient(nullptr, nullptr, nullptr, nullptr, nullptr);
    /// @brief Client for reading and writing telemetry to a cluster.
    FrameClient telem = FrameClient(nullptr, nullptr);
    /// @brief Client for managing devices and their configuration.
    HardwareClient hardware = HardwareClient(
        nullptr,
        nullptr,
        nullptr,
        nullptr,
        nullptr,
        nullptr,
        nullptr,
        nullptr,
        nullptr
    );
    std::shared_ptr<AuthMiddleware> auth = nullptr;

    /// @brief constructs the Synnax client from the provided configuration.
    explicit Synnax(const Config &cfg) {
        auto t = Transport(
            cfg.port,
            cfg.host,
            cfg.ca_cert_file,
            cfg.client_cert_file,
            cfg.client_key_file
        );
        priv::check_little_endian();
        auth = std::make_shared<AuthMiddleware>(
            std::move(t.auth_login),
            cfg.username,
            cfg.password,
            5
        );
        t.use(auth);
        channels = ChannelClient(std::move(t.chan_retrieve),
                                 std::move(t.chan_create));
        ranges = RangeClient(
            std::move(t.range_retrieve),
            std::move(t.range_create),
            t.range_kv_get,
            t.range_kv_set,
            t.range_kv_delete
        );
        telem = FrameClient(std::move(t.frame_stream), std::move(t.frame_write));
        hardware = HardwareClient(
            std::move(t.rack_create_client),
            std::move(t.rack_retrieve),
            std::move(t.rack_delete),
            t.module_create,
            t.module_retrieve,
            t.module_delete,
            std::move(t.device_create),
            std::move(t.device_retrieve),
            std::move(t.device_delete)
        );
    }
};
}
