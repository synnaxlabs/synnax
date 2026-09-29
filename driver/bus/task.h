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
#include <optional>
#include <string>
#include <utility>

#include "x/cpp/breaker/breaker.h"
#include "x/cpp/json/json.h"

#include "driver/bus/config.h"
#include "driver/bus/connection.h"
#include "driver/bus/read.h"
#include "driver/bus/transport.h"
#include "driver/bus/write.h"
#include "driver/common/common.h"
#include "driver/common/read_task.h"
#include "driver/common/write_task.h"
#include "driver/task/task.h"

namespace driver::bus {
/// @brief retrieves the device with the given key and parses its properties.
/// @returns x::errors::VALIDATION bound to the device field when the device does not
/// exist or its properties are invalid.
template<typename Properties>
std::pair<Properties, x::errors::Error> retrieve_properties(
    const synnax::Synnax &client,
    const x::json::Parser &parser,
    const std::string &key
) {
    if (key.empty()) {
        parser.field_err("device", "this field is required");
        return {Properties{}, parser.error()};
    }
    auto [dev, err] = client.devices.retrieve(key);
    if (err.matches(x::errors::NOT_FOUND)) {
        parser.field_err("device", "device " + key + " does not exist");
        return {Properties{}, parser.error()};
    }
    if (err) return {Properties{}, err};
    x::json::Parser props_parser(dev.properties);
    auto props = Properties::parse(props_parser);
    if (!props_parser.ok()) {
        parser.field_err("device", "device " + dev.name + " has invalid properties");
        return {Properties{}, props_parser.error()};
    }
    return {std::move(props), x::errors::NIL};
}

/// @returns the framing of a read or write config, or nullopt for datagram configs,
/// which have none.
template<typename Config>
std::optional<::synnax::bus::Framing> framing(const Config &cfg) {
    if constexpr (requires { cfg.framing; })
        return cfg.framing;
    else
        return std::nullopt;
}

/// @returns an Acquire of the connection to the device with the given key and
/// properties, opened through Conn.
template<typename Conn, typename Properties>
Acquire acquirer(
    const std::shared_ptr<Connections> &connections,
    const std::string &key,
    Properties props
) {
    auto settings = props.to_json();
    return acquirer(
        connections,
        key,
        std::move(settings),
        opener<Conn>(std::move(props))
    );
}

/// @brief configures a bus read task whose device connects through Conn.
/// @param connections the connections of the integration, shared with its other
/// tasks.
/// @tparam Conn a transport with a static open(Properties).
/// @tparam Properties the device properties of the integration.
/// @tparam Config the read config of the integration.
template<typename Conn, typename Properties, typename Config>
std::pair<common::ConfigureResult, x::errors::Error> configure_read(
    const std::shared_ptr<Connections> &connections,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) {
    x::json::Parser parser(task.config);
    const auto cfg = Config::parse(parser);
    if (!parser.ok()) return {common::ConfigureResult{}, parser.error()};
    auto [props, props_err] = retrieve_properties<Properties>(
        *ctx->client,
        parser,
        cfg.device
    );
    if (props_err) return {common::ConfigureResult{}, props_err};
    auto [resolved, err] = ReadConfig::parse(
        *ctx->client,
        parser,
        cfg,
        cfg,
        framing(cfg)
    );
    if (err) return {common::ConfigureResult{}, err};
    return {
        common::ConfigureResult{
            .task = std::make_unique<common::ReadTask>(
                task,
                ctx,
                x::breaker::default_config(task.name),
                std::make_unique<Source>(
                    std::move(resolved),
                    acquirer<Conn>(connections, cfg.device, std::move(props))
                )
            ),
            .auto_start = cfg.auto_start,
        },
        x::errors::NIL,
    };
}

/// @brief configures a bus write task whose device connects through Conn. See
/// configure_read.
template<typename Conn, typename Properties, typename Config>
std::pair<common::ConfigureResult, x::errors::Error> configure_write(
    const std::shared_ptr<Connections> &connections,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) {
    x::json::Parser parser(task.config);
    const auto cfg = Config::parse(parser);
    if (!parser.ok()) return {common::ConfigureResult{}, parser.error()};
    auto [props, props_err] = retrieve_properties<Properties>(
        *ctx->client,
        parser,
        cfg.device
    );
    if (props_err) return {common::ConfigureResult{}, props_err};
    auto [resolved, err] = WriteConfig::parse(*ctx->client, parser, cfg, framing(cfg));
    if (err) return {common::ConfigureResult{}, err};
    return {
        common::ConfigureResult{
            .task = std::make_unique<common::WriteTask>(
                task,
                ctx,
                x::breaker::default_config(task.name),
                std::make_unique<Sink>(
                    std::move(resolved),
                    std::make_unique<ConnectionTransmitter>(
                        acquirer<Conn>(connections, cfg.device, std::move(props))
                    ),
                    ctx,
                    task
                )
            ),
            .auto_start = cfg.auto_start,
        },
        x::errors::NIL,
    };
}
}
