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

#include "client/cpp/tcp/json.gen.h"
#include "client/cpp/tcp/types.gen.h"

#include "driver/bus/tasks.h"
#include "driver/common/status.h"
#include "driver/tcp/client.h"
#include "driver/tcp/tcp.h"
#include "driver/transport/transport.h"

namespace driver::tcp {
std::pair<std::shared_ptr<transport::Connection>, x::errors::Error>
Devices::acquire(const Device &dev) {
    return this->connections.acquire(
        dev.key,
        dev.props.to_json(),
        {.open = [props = dev.props]()
             -> std::pair<std::unique_ptr<transport::Transport>, x::errors::Error> {
            auto [client, err] = Client::connect(
                props.host,
                props.port,
                CONNECT_TIMEOUT
            );
            return {std::move(client), err};
        }}
    );
}

std::pair<std::unique_ptr<task::Task>, bool> Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    const std::string &cmd_key,
    x::breaker::Breaker &cancel
) {
    const auto acquire = [devices = this->devices](const Device &dev) -> bus::Acquire {
        return [devices, dev] { return devices->acquire(dev); };
    };
    if (task.type == READ_TASK_TYPE)
        return common::handle_config_err(
            ctx,
            task,
            bus::configure_read<synnax::tcp::ReadConfig, synnax::tcp::Properties>(
                ctx,
                task,
                this->readers,
                acquire
            ),
            cmd_key,
            cancel
        );
    if (task.type == WRITE_TASK_TYPE)
        return common::handle_config_err(
            ctx,
            task,
            bus::configure_write<synnax::tcp::WriteConfig, synnax::tcp::Properties>(
                ctx,
                task,
                acquire
            ),
            cmd_key,
            cancel
        );
    return {nullptr, false};
}
}
