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

#include "absl/log/log.h"

#include "client/cpp/arc/arc.h"
#include "client/cpp/channel/channel.h"
#include "client/cpp/connection/checker.h"
#include "client/cpp/control/control.h"
#include "client/cpp/device/device.h"
#include "client/cpp/framer/framer.h"
#include "client/cpp/rack/rack.h"
#include "client/cpp/ranger/ranger.h"
#include "client/cpp/status/status.h"
#include "client/cpp/transport.h"
#include "client/cpp/version/version.h"
#include "x/cpp/json/json.h"
#include "x/cpp/log/log.h"

namespace synnax {
///// @brief Internal namespace. Do not use.
namespace details {
/// @brief Does a best effort check to ensure the machine is little endian, and warns
/// the user if it is not.
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
    /// @brief the host of a Core to connect to.
    std::string host = "localhost";
    /// @brief the port for the specified host.
    std::uint16_t port = 9090;
    /// @brief the username to use when authenticating with the Core.
    std::string username = "synnax";
    /// @brief the password to use when authenticating with the Core.
    std::string password = "seldon";
    /// @brief use TLS encryption. The system trust store verifies the Core's
    /// certificate.
    bool secure = false;
    /// @brief sets the clock skew threshold at which a warning will be logged.
    x::telem::TimeSpan clock_skew_threshold = x::telem::SECOND * 1;
    /// @brief sets the maximum number of login retries before giving up.
    std::uint32_t max_retries = 5;
    /// @brief an API key to log in with. When set, the username and password are
    /// ignored.
    std::string api_key;

    template<typename ParserT>
    void override(ParserT &parser) {
        this->host = parser.field("host", this->host);
        this->port = parser.field("port", this->port);
        this->username = parser.field("username", this->username);
        this->password = parser.field("password", this->password);
        this->secure = parser.field("secure", this->secure);
        this->clock_skew_threshold = x::telem::TimeSpan(parser.field(
            "clock_skew_threshold",
            this->clock_skew_threshold.nanoseconds()
        ));
        this->max_retries = parser.field("max_retries", this->max_retries);
        this->api_key = parser.field("api_key", this->api_key);
    }

    friend std::ostream &operator<<(std::ostream &os, const Config &cfg) {
        os << x::log::SHALE() << "  " << "cluster address" << x::log::RESET() << ": "
           << cfg.address() << "\n"
           << "  " << x::log::SHALE() << "username" << x::log::RESET() << ": "
           << cfg.username << "\n"
           << "  " << x::log::SHALE() << "password" << x::log::RESET() << ": "
           << x::log::sensitive_string(cfg.password) << "\n"
           << "  " << x::log::SHALE() << "api key" << x::log::RESET() << ": "
           << x::log::sensitive_string(cfg.api_key) << "\n"
           << "  " << x::log::SHALE() << "secure" << x::log::RESET() << ": "
           << x::log::bool_to_str(cfg.secure);
        return os;
    }

    /// @brief returns the address of the Core in the form "host:port".
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
            {"secure", this->secure},
            {"clock_skew_threshold", this->clock_skew_threshold.nanoseconds()},
            {"max_retries", this->max_retries},
            {"api_key", this->api_key}
        };
    }
};

/// @brief Client to perform operations against a Synnax Core.
class Synnax {
    details::Transport t;

public:
    /// @brief Client for creating and retrieving channels from the Core.
    channel::Client channels;
    std::shared_ptr<auth::Middleware> auth;
    /// @brief Client for creating, listing, and deleting API keys.
    auth::KeyClient api_keys;
    /// @brief Connectivity checker that polls the Core for health and clock skew.
    std::shared_ptr<connection::Checker> connectivity;
    /// @brief Client for creating, retrieving, and performing operations on ranges from
    /// the Core.
    ranger::Client ranges;
    task::Client tasks;
    /// @brief Client for reading and writing telemetry to the Core.
    framer::Client telem;
    /// @brief Client for managing racks.
    rack::Client racks;
    /// @brief Client for managing devices.
    device::Client devices;
    /// @brief Client for managing statuses.
    status::Client statuses;
    /// @brief Client for managing Arc automation programs.
    arc::Client arcs;
    /// @brief Client for reading the control state of channels.
    control::Client control;

    /// @brief constructs the Synnax client from the provided configuration.
    explicit Synnax(const Config &cfg):
        t(cfg.port, cfg.host, cfg.secure),
        channels(this->t.chan_retrieve, this->t.chan_create),
        auth([&]() -> std::shared_ptr<auth::Middleware> {
            auto mw = std::make_shared<auth::Middleware>(
                std::move(this->t.auth_session_create),
                std::move(this->t.auth_session_renew),
                std::move(this->t.auth_session_retrieve),
                std::move(this->t.auth_session_delete),
                cfg.api_key.empty() ? auth::password_proof(cfg.username, cfg.password)
                                    : auth::api_key_proof(cfg.api_key)
            );
            this->t.use(mw);
            return mw;
        }()),
        api_keys(
            std::move(this->t.auth_key_create),
            std::move(this->t.auth_key_retrieve),
            std::move(this->t.auth_key_delete)
        ),
        connectivity(
            std::make_shared<connection::Checker>(
                std::move(this->t.connectivity_check),
                30 * x::telem::SECOND,
                SYNNAX_CLIENT_VERSION,
                cfg.host,
                cfg.clock_skew_threshold
            )
        ),
        ranges(
            std::move(this->t.range_retrieve),
            std::move(this->t.range_create),
            std::move(this->t.range_set_end),
            ranger::kv::Client(
                this->t.range_kv_get,
                this->t.range_kv_set,
                this->t.range_kv_delete
            )
        ),
        tasks(this->t.task_create, this->t.task_retrieve, this->t.task_delete),
        telem(
            std::move(this->t.frame_stream),
            std::move(this->t.frame_write),
            channel::Client(this->t.chan_retrieve, this->t.chan_create)
        ),
        racks(
            std::move(this->t.rack_create_client),
            std::move(this->t.rack_retrieve),
            std::move(this->t.rack_delete),
            this->tasks
        ),
        devices(
            std::move(this->t.device_create),
            std::move(this->t.device_retrieve),
            std::move(this->t.device_delete)
        ),
        statuses(
            this->t.status_retrieve,
            this->t.status_set,
            this->t.status_delete,
            this->t.status_set_by_key_or_name
        ),
        arcs(this->t.arc_retrieve, this->t.arc_create, this->t.arc_delete),
        control(this->t.control_retrieve) {
        details::check_little_endian();
    }

    Synnax(Synnax &&) = default;
    Synnax &operator=(Synnax &&) = default;

    ~Synnax() {
        if (this->connectivity) this->connectivity->stop();
        // The destructor is the client's close, so the session ends here. It cannot
        // report a failure, and a session it cannot reach idles out on its own.
        if (this->auth) this->auth->logout();
    }
};
}
