// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <utility>
#include <vector>

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
}
