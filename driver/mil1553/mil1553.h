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
#include <unordered_map>
#include <utility>
#include <vector>

#include "client/cpp/library/types.gen.h"
#include "client/cpp/mil1553/types.gen.h"
#include "x/cpp/errors/errors.h"

#include "driver/bus/config.h"
#include "driver/bus/connection.h"
#include "driver/mil1553/backend.h"
#include "driver/task/task.h"

namespace driver::mil1553 {
/// @brief the name of the MIL-STD-1553 integration.
const std::string INTEGRATION_NAME = "mil1553";
/// @brief the make of MIL-STD-1553 devices.
const std::string MAKE = "MIL-STD-1553";
/// @brief the type of MIL-STD-1553 read tasks.
const std::string READ_TASK_TYPE = INTEGRATION_NAME + "_read";
/// @brief the type of MIL-STD-1553 write tasks.
const std::string WRITE_TASK_TYPE = INTEGRATION_NAME + "_write";
/// @brief the type of the task that lists the channels of each backend.
const std::string SCAN_TASK_TYPE = INTEGRATION_NAME + "_scan";

/// @brief the backends a factory opens channels through, by backend name.
using Backends = std::unordered_map<std::string, std::shared_ptr<Backend>>;
/// @brief one open channel, shared by the read and write tasks of its device.
using Connection = bus::BasicConnection<Channel>;
/// @brief acquires the shared channel of a task's device.
using Acquire = bus::BasicAcquire<Channel>;
/// @brief the open channels of the integration, one per device.
using Connections = bus::BasicConnections<Channel>;

/// @returns the simulated, DDC, and Ballard backends.
Backends create_backends();

/// @returns the terminal and subaddress of a command, for errors and warnings.
std::string address(const codec::mil1553::Command &command);

/// @returns a check that a read task on a channel with props can receive a message.
/// A bus controller polls transmit messages on their period, a remote terminal
/// receives messages to its own terminals, and a monitor sees every message.
bus::Check read_check(const synnax::mil1553::Properties &props);

/// @returns a check that a write task on a channel with props can send a message. A
/// bus controller sends receive messages, a remote terminal answers transmit messages
/// for its own terminals, and a monitor sends nothing.
bus::Check write_check(const synnax::mil1553::Properties &props);

/// @brief configures MIL-STD-1553 read, write, and scan tasks. The read and write
/// tasks of one device share its channel.
class Factory final : public task::Factory {
    const Backends backends;
    const std::shared_ptr<Connections> connections = std::make_shared<Connections>();

public:
    explicit Factory(Backends backends): backends(std::move(backends)) {}

    std::string name() override { return INTEGRATION_NAME; }

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task,
        const std::string &cmd_key
    ) override;

    /// @brief creates the rack's MIL-STD-1553 scan task when it has none.
    std::vector<std::pair<synnax::task::Task, std::unique_ptr<task::Task>>>
    configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::rack::Rack &rack
    ) override;
};
}
