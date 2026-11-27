// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <string>
#include <utility>

#include "client/cpp/serial/json.gen.h"
#include "x/cpp/json/json.h"

#include "driver/common/status.h"
#include "driver/serial/port.h"
#include "driver/serial/read_task.h"
#include "driver/serial/serial.h"
#include "driver/transport/errors.h"

namespace driver::serial {
std::pair<synnax::serial::Properties, x::errors::Error>
retrieve_properties(const synnax::device::Client &devices, const std::string &key) {
    auto [dev, err] = devices.retrieve(key);
    if (err) return {synnax::serial::Properties{}, err};
    x::json::Parser parser(x::json::json(dev.properties));
    auto props = synnax::serial::Properties::parse(parser);
    return {std::move(props), parser.error()};
}

std::pair<std::shared_ptr<transport::Connection>, x::errors::Error> acquire(
    transport::Connections &connections,
    const std::string &key,
    const synnax::serial::Properties &props
) {
    auto [conn, err] = connections.acquire(
        key,
        props.to_json(),
        {.open = [props] {
            auto [port, open_err] = Port::open(props);
            return std::pair<std::unique_ptr<transport::Transport>, x::errors::Error>{
                std::move(port),
                open_err
            };
        }}
    );
    if (err) return {nullptr, err};
    auto guard = conn->lock();
    if (auto open_err = guard.transport().second.skip(transport::UNREACHABLE_ERROR))
        return {nullptr, open_err};
    return {std::move(conn), x::errors::NIL};
}

std::pair<std::unique_ptr<task::Task>, bool> Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    const std::string &cmd_key,
    x::breaker::Breaker &cancel
) {
    if (task.type == READ_TASK_TYPE)
        return common::handle_config_err(
            ctx,
            task,
            configure_read(ctx, task, this->connections),
            cmd_key,
            cancel
        );
    return {nullptr, false};
}
}
