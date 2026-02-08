// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>

#include "glog/logging.h"

#include "client/cpp/arc/arc.h"
#include "client/cpp/channel/channel.h"
#include "client/cpp/device/device.h"
#include "client/cpp/framer/framer.h"
#include "client/cpp/rack/rack.h"
#include "client/cpp/ranger/ranger.h"
#include "client/cpp/status/status.h"
#include "client/cpp/transport.h"
#include "x/cpp/json/json.h"
#include "x/cpp/log/log.h"
#include "x/cpp/path/path.h"

namespace synnax {
///// @brief Internal namespace. Do not use.
namespace priv {
/// @brief Does a best effort check to ensure the machine is little endian, and
/// warns the user if it is not.
inline void check_little_endian() {
    int num = 1;
    if (*reinterpret_cast<char *>(&num) == 1) return;
    LOG(WARNING) << "Detected big endian system, which Synnax does not support. This "
                    "may silently corrupt telemetry."
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
    /// @brief path to the CA certificate file to use when connecting to a secure
    /// node. This is only required if the node is configured to use TLS.
    std::string ca_cert_file;
    /// @brief path to the client certificate file to use when connecting to a
    /// secure node and using client authentication. This is not required when in
    /// insecure mode or using username/password authentication.
    std::string client_cert_file;
    /// @brief path to the client key file to use when connecting to a secure node
    /// and using client authentication. This is not required when in insecure mode
    /// or using username/password authentication.
    std::string client_key_file;
    /// @brief sets the clock skew threshold at which a warning will be logged.
    x::telem::TimeSpan clock_skew_threshold = x::telem::SECOND * 1;
    /// @brief sets the maximum number of login retries before giving up.
    std::uint32_t max_retries = 5;

    template<typename ParserT>
    void override(ParserT &parser) {
        this->host = parser.field("host", this->host);
        this->port = parser.field("port", this->port);
        this->username = parser.field("username", this->username);
        this->password = parser.field("password", this->password);
        this->client_cert_file = parser.field(
            "client_cert_file",
            this->client_cert_file
        );
        this->client_key_file = parser.field("client_key_file", this->client_key_file);
        this->ca_cert_file = parser.field("ca_cert_file", this->ca_cert_file);
        this->clock_skew_threshold = x::telem::TimeSpan(parser.field(
            "clock_skew_threshold",
            this->clock_skew_threshold.nanoseconds()
        ));
        this->max_retries = parser.field("max_retries", this->max_retries);
    }

    friend std::ostream &operator<<(std::ostream &os, const Config &cfg) {
        os << x::log::SHALE() << "  " << "cluster address" << x::log::RESET() << ": "
           << cfg.address() << "\n"
           << "  " << x::log::SHALE() << "username" << x::log::RESET() << ": "
           << cfg.username << "\n"
           << "  " << x::log::SHALE() << "password" << x::log::RESET() << ": "
           << x::log::sensitive_string(cfg.password) << "\n"
           << "  " << x::log::SHALE() << "secure" << x::log::RESET() << ": "
           << x::log::bool_to_str(cfg.is_secure()) << "\n";
        if (!cfg.is_secure()) return os;
        os << "  " << x::log::SHALE() << "ca_cert_file" << x::log::RESET() << ": "
           << x::path::resolve_relative(cfg.ca_cert_file) << "\n"
           << "  " << x::log::SHALE() << "client_cert_file" << x::log::RESET() << ": "
           << x::path::resolve_relative(cfg.client_cert_file) << "\n"
           << "  " << x::log::SHALE() << "client_key_file" << x::log::RESET() << ": "
           << x::path::resolve_relative(cfg.client_key_file) << "\n";
        return os;
    }

    /// @brief returns true if the configuration uses TLS encryption to secure
    /// communications with the cluster.
    [[nodiscard]] bool is_secure() const { return !this->ca_cert_file.empty(); }

    /// @brief returns the address of the cluster in the form "host:port".
    [[nodiscard]]
    std::string address() const {
        return this->host + ":" + std::to_string(this->port);
    }

    [[nodiscard]] x::json::json to_json() const {
        return {
            {"host", this->host},
            {"port", this->port},
            {"username", this->username},
            {"password", this->password},
            {"ca_cert_file", this->ca_cert_file},
            {"client_cert_file", this->client_cert_file},
            {"client_key_file", this->client_key_file},
            {"clock_skew_threshold", this->clock_skew_threshold.nanoseconds()},
            {"max_retries", this->max_retries}
        };
    }
};

/// @brief Client to perform operations against a Synnax cluster.
class Synnax {
public:
    /// @brief Client for creating and retrieving channels in a cluster.
    channel::Client channels = channel::Client(nullptr, nullptr);
    /// @brief Client for creating, retrieving, and performing operations on ranges
    /// in a cluster.
    ranger::Client ranges = ranger::Client(nullptr, nullptr, kv::Client());
    /// @brief Client for reading and writing telemetry to a cluster.
    framer::Client telem = framer::Client(nullptr, nullptr, channel::Client());
    /// @brief Client for managing racks.
    rack::Client
        racks = rack::Client(nullptr, nullptr, nullptr, nullptr, nullptr, nullptr);
    /// @brief Client for managing devices.
    device::Client devices = device::Client(nullptr, nullptr, nullptr);
    /// @brief Client for managing statuses.
    status::Client statuses = status::Client();
    /// @brief Client for managing Arc automation programs.
    arc::Client arcs = arc::Client(nullptr, nullptr, nullptr);
    std::shared_ptr<auth::Middleware> auth = nullptr;

    /// @brief constructs the Synnax client from the provided configuration.
    explicit Synnax(const Config &cfg) {
        auto t = Transport::configure(
            cfg.port,
            cfg.host,
            cfg.ca_cert_file,
            cfg.client_cert_file,
            cfg.client_key_file
        );
        priv::check_little_endian();
        this->auth = std::make_shared<auth::Middleware>(
            std::move(t.auth_login),
            cfg.username,
            cfg.password,
            cfg.clock_skew_threshold
        );
        t.use(this->auth);
        this->channels = channel::Client(t.chan_retrieve, t.chan_create);
        this->ranges = ranger::Client(
            std::move(t.range_retrieve),
            std::move(t.range_create),
            kv::Client(t.range_kv_get, t.range_kv_set, t.range_kv_delete)
        );
        this->telem = framer::Client(
            std::move(t.frame_stream),
            std::move(t.frame_write),
            channel::Client(t.chan_retrieve, t.chan_create)
        );
        this->racks = rack::Client(
            std::move(t.rack_create_client),
            std::move(t.rack_retrieve),
            std::move(t.rack_delete),
            t.task_create,
            t.task_retrieve,
            t.task_delete
        );
        this->devices = device::Client(
            std::move(t.device_create),
            std::move(t.device_retrieve),
            std::move(t.device_delete)
        );
        this->statuses = status::Client(
            t.status_retrieve,
            t.status_set,
            t.status_delete
        );
        this->arcs = arc::Client(t.arc_retrieve, t.arc_create, t.arc_delete);
    }
};
}
