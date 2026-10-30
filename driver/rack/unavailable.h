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

#include "driver/common/status.h"
#include "driver/errors/errors.h"
#include "driver/task/task.h"

namespace driver::rack {
/// @brief stands in for an integration that this Driver does not run. A configure of a
/// task of the integration fails with the reason.
class UnavailableFactory final : public task::Factory {
    const std::string integration;
    const std::string reason;

public:
    UnavailableFactory(std::string integration, std::string reason):
        integration(std::move(integration)), reason(std::move(reason)) {}

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task,
        const std::string &cmd_key
    ) override {
        if (task.type != this->integration &&
            !task.type.starts_with(this->integration + "_"))
            return {nullptr, false};
        return common::handle_config_err(
            ctx,
            task,
            {common::ConfigureResult{},
             x::errors::Error(errors::CONFIGURATION_ERROR, this->reason)},
            cmd_key
        );
    }

    std::string name() override { return this->integration; }
};
}
