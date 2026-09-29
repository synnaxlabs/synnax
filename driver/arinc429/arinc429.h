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

#include "client/cpp/arinc429/types.gen.h"
#include "client/cpp/library/types.gen.h"
#include "x/cpp/errors/errors.h"

#include "driver/arinc429/backend.h"
#include "driver/task/task.h"

namespace driver::arinc429 {
/// @brief the name of the ARINC 429 integration.
const std::string INTEGRATION_NAME = "arinc429";
/// @brief the make of ARINC 429 devices.
const std::string MAKE = "ARINC 429";
/// @brief the type of ARINC 429 read tasks.
const std::string READ_TASK_TYPE = INTEGRATION_NAME + "_read";
/// @brief the type of ARINC 429 write tasks.
const std::string WRITE_TASK_TYPE = INTEGRATION_NAME + "_write";
/// @brief the type of the task that lists the channels of each backend.
const std::string SCAN_TASK_TYPE = INTEGRATION_NAME + "_scan";

/// @brief the backends a factory opens channels through, by backend name.
using Backends = std::unordered_map<std::string, std::shared_ptr<Backend>>;

/// @returns the simulated, DDC, and Ballard backends.
Backends create_backends();

/// @brief checks that an ARINC 429 channel can carry a message: it passes
/// codec::arinc429::validate and has no query.
x::errors::Error check(const synnax::library::MessageEntry &message);

/// @brief checks the properties of an ARINC 429 device.
/// @returns the backend the properties name, or CONFIGURATION_ERROR when the backend
/// or speed is unknown.
std::pair<std::shared_ptr<Backend>, x::errors::Error>
resolve(const Backends &backends, const synnax::arinc429::Properties &props);

/// @brief configures ARINC 429 read, write, and scan tasks.
class Factory final : public task::Factory {
    const Backends backends;

public:
    explicit Factory(Backends backends): backends(std::move(backends)) {}

    std::string name() override { return INTEGRATION_NAME; }

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task,
        const std::string &cmd_key
    ) override;

    /// @brief creates the rack's ARINC 429 scan task when it has none.
    std::vector<std::pair<synnax::task::Task, std::unique_ptr<task::Task>>>
    configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::rack::Rack &rack
    ) override;
};
}
