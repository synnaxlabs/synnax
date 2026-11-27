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
#include "client/cpp/serial/types.gen.h"

#include "driver/bus/tasks.h"
#include "driver/common/status.h"
#include "driver/serial/serial.h"

namespace driver::serial {
std::pair<std::unique_ptr<task::Task>, bool> Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    const std::string &cmd_key,
    x::breaker::Breaker &cancel
) {
    const auto acquire = [ports = this->ports](const Device &dev) -> bus::Acquire {
        return [ports, dev] { return ports->acquire(dev); };
    };
    if (task.type == READ_TASK_TYPE)
        return common::handle_config_err(
            ctx,
            task,
            bus::configure_read<synnax::serial::ReadConfig, synnax::serial::Properties>(
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
            bus::configure_write<
                synnax::serial::WriteConfig,
                synnax::serial::Properties>(ctx, task, acquire),
            cmd_key,
            cancel
        );
    return {nullptr, false};
}
}
