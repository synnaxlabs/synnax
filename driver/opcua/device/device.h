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
#include <string>
#include <utility>
#include <vector>

#include "client/cpp/errors/errors.h"
#include "client/cpp/synnax.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"

#include "driver/opcua/connection/connection.h"
#include "driver/opcua/types/types.h"

namespace driver::opcua::device {
struct Properties {
    connection::Config connection;
    std::vector<types::Node> channels;

    explicit Properties(const x::json::Parser &parser):
        connection(parser.child("connection")) {
        if (!parser.has("channels")) return;
        parser.iter("channels", [&](x::json::Parser &cb) {
            channels.emplace_back(cb);
        });
    }
};

/// @brief retrieves the device and parses its connection configuration.
/// @returns the retrieve error, or a validation error for invalid properties.
inline std::pair<connection::Config, x::errors::Error>
retrieve_connection(synnax::Synnax &client, const std::string &key) {
    auto [dev, err] = client.devices.retrieve(key);
    if (err)
        return {
            connection::Config(),
            x::errors::Error(err, "failed to retrieve device: " + err.data)
        };
    const auto parser = x::json::Parser(dev.properties);
    auto cfg = connection::Config(parser.child("connection"));
    if (parser.error()) return {connection::Config(), parser.error()};
    return {std::move(cfg), x::errors::NIL};
}

/// @brief connects a task to its device with the device's current settings, so a
/// corrected endpoint or login takes effect without a restart.
class Connector {
    std::shared_ptr<synnax::Synnax> client;
    std::shared_ptr<connection::Pool> pool;
    /// @brief the key of the device.
    std::string key;
    std::string log_prefix;
    /// @brief the device's connection settings as of the last retrieve.
    connection::Config cfg;

public:
    Connector(
        std::shared_ptr<synnax::Synnax> client,
        std::shared_ptr<connection::Pool> pool,
        std::string key,
        std::string log_prefix
    ):
        client(std::move(client)),
        pool(std::move(pool)),
        key(std::move(key)),
        log_prefix(std::move(log_prefix)) {}

    /// @brief retrieves the device, so a changed trusted server certificate takes
    /// effect when the task starts, then connects.
    /// @returns the error of the retrieve or of the connection.
    std::pair<connection::Pool::Connection, x::errors::Error> start() {
        auto [cfg, err] = retrieve_connection(*this->client, this->key);
        if (err) return {connection::Pool::Connection(nullptr, nullptr, ""), err};
        this->cfg = std::move(cfg);
        return this->pool->acquire(this->cfg, this->log_prefix);
    }

    /// @brief retrieves the device, then connects. Keeps the last settings while the
    /// Core is unavailable.
    /// @returns the error of the retrieve or of the connection.
    std::pair<connection::Pool::Connection, x::errors::Error> connect() {
        auto [cfg, err] = retrieve_connection(*this->client, this->key);
        if (!err)
            this->cfg = std::move(cfg);
        else if (!synnax::errors::is_temporarily_unavailable(err))
            return {connection::Pool::Connection(nullptr, nullptr, ""), err};
        return this->pool->acquire(this->cfg, this->log_prefix);
    }

    /// @brief connects again when the device's settings now resolve to another client
    /// certificate than conn uses. See connection::Pool::refresh.
    std::pair<connection::Pool::Connection, x::errors::Error>
    refresh(const connection::Pool::Connection &conn) {
        return this->pool->refresh(conn, this->cfg, this->log_prefix);
    }
};
}
