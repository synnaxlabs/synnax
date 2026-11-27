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

#include "client/cpp/udp/json.gen.h"
#include "client/cpp/udp/types.gen.h"

#include "driver/bus/tasks.h"
#include "driver/common/status.h"
#include "driver/errors/errors.h"
#include "driver/transport/transport.h"
#include "driver/udp/socket.h"
#include "driver/udp/udp.h"

namespace driver::udp {
std::pair<std::shared_ptr<transport::Connection>, x::errors::Error>
Devices::acquire(const Device &dev) {
    return this->connections.acquire(
        dev.key,
        dev.props.to_json(),
        {.open = [props = dev.props]()
             -> std::pair<std::unique_ptr<transport::Transport>, x::errors::Error> {
            Address addr{
                .local_address = props.local_address,
                .port = props.local_port
            };
            if (props.remote) {
                addr.remote_host = props.remote->host;
                addr.remote_port = props.remote->port;
            }
            auto [socket, err] = Socket::open(addr, RESOLVE_TIMEOUT);
            return {std::move(socket), err};
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
            bus::configure_read<synnax::udp::ReadConfig, synnax::udp::Properties>(
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
            bus::configure_write<synnax::udp::WriteConfig, synnax::udp::Properties>(
                ctx,
                task,
                acquire,
                [](const Device &dev) {
                    if (dev.props.remote.has_value()) return x::errors::NIL;
                    return x::errors::Error(
                        driver::errors::CONFIGURATION_ERROR,
                        "The device '" + dev.name +
                            "' has no remote to send to. Set a remote on the device."
                    );
                }
            ),
            cmd_key,
            cancel
        );
    return {nullptr, false};
}
}
