// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "client/cpp/udp/json.gen.h"

#include "driver/bus/task.h"
#include "driver/common/status.h"
#include "driver/udp/socket.h"
#include "driver/udp/udp.h"

namespace driver::udp {
std::pair<std::unique_ptr<task::Task>, bool> Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    const std::string &cmd_key
) {
    if (task.type == READ_TASK_TYPE)
        return common::handle_config_err(
            ctx,
            task,
            bus::configure_read<
                Socket,
                synnax::udp::Properties,
                synnax::udp::ReadConfig>(this->connections, ctx, task),
            cmd_key
        );
    if (task.type == WRITE_TASK_TYPE)
        return common::handle_config_err(
            ctx,
            task,
            bus::configure_write<
                Socket,
                synnax::udp::Properties,
                synnax::udp::WriteConfig>(this->connections, ctx, task),
            cmd_key
        );
    return {nullptr, false};
}
}
