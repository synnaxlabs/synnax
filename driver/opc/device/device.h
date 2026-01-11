// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <memory>
#include <string>
#include <utility>

/// external
#include "open62541/client.h"

/// module
#include "x/cpp/telem/series.h"
#include "x/cpp/errors/errors.h"

/// internal
#include "driver/opc/connection/connection.h"
#include "driver/opc/errors/errors.h"
#include "driver/opc/telem/telem.h"
#include "driver/opc/types/types.h"

namespace driver::opc::device {
struct Properties {
    driver::opc::connection::Config connection;
    std::vector<Node> channels;

    Properties(
        const driver::opc::connection::Config &connection,
        const std::vector<Node> &channels
    ):
        connection(connection), channels(channels) {}

    explicit Properties(const x::json::Parser &parser):
        connection(parser.child("connection")) {
        if (!parser.has("channels")) return;
        parser.iter("channels", [&](x::json::Parser &cb) { channels.emplace_back(cb); });
    }

    x::json::json to_json() const {
        x::json::json j;
        j["connection"] = connection.to_json();
        j["channels"] = x::json::json::array();
        for (const auto &ch: channels)
            j["channels"].push_back(ch.to_json());
        return j;
    }
};
}
