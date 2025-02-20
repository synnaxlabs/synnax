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
#include "client/cpp/framer/framer.h"
#include "client/cpp/hardware/hardware.h"
#include "client/cpp/ranger/ranger.h"
#include "x/cpp/config/config.h"

using namespace synnax;


namespace synnax {
///// @brief Internal namespace. Do not use.
namespace priv {
/// @brief Does a best effort check to ensure the machine is little endian, and warns the user if it is not.
inline void check_little_endian() {
    const std::uint32_t value = 1;
    const auto *const bytes = reinterpret_cast<const unsigned char*>(&value);
    if (bytes[0] == 1) return;
    std::cout
            << "WARNING: Detected big endian system, which Synnax does not support. This may silently corrupt telemetry.\n";
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

    void override(config::Parser &parser) {
        this->host = parser.optional("host", this->host);
        this->port = parser.optional("port", this->port);
        this->username = parser.optional("username", this->username);
        this->password = parser.optional("password", this->password);
        this->client_cert_file = parser.optional("client_cert_file", this->client_cert_file);
        this->client_key_file = parser.optional("client_key_file", this->client_key_file);
        this->ca_cert_file = parser.optional("ca_cert_file", this->ca_cert_file);
    }

    /// @brief Converts the configuration to a JSON object.
    [[nodiscard]] json to_json() const {
        return {
            {"host", this->host},
            {"port", this->port},
            {"username", this->username},
            {"password", this->password},
            {"ca_cert_file", this->ca_cert_file},
            {"client_cert_file", this->client_cert_file},
            {"client_key_file", this->client_key_file},
        };
    }
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
