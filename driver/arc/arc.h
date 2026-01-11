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
#include <utility>
#include <vector>

#include "client/cpp/synnax.h"

#include "driver/task/task.h"

namespace arc {
/// @brief integration name for arc runtime.
const std::string INTEGRATION_NAME = "arc";
/// @brief task type for arc runtime tasks.
const std::string TASK_TYPE = INTEGRATION_NAME;

/// @brief factory for creating arc runtime tasks.
class Factory final : public task::Factory {
public:
    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task
    ) override;

    std::vector<std::pair<synnax::task::Task, std::unique_ptr<task::Task>>>
    configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::rack::Rack &rack
    ) override;

    [[nodiscard]] std::string name() override;
};
}
