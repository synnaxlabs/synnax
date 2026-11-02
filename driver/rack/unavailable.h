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

#include "absl/log/log.h"

#include "driver/common/factory.h"
#include "driver/common/status.h"
#include "driver/errors/errors.h"
#include "driver/task/task.h"

namespace driver::rack {
/// @brief stands in for an integration that this Driver does not run. A configure of a
/// task of the integration fails with the reason. A non-empty legacy_scanner_type names
/// the scanner that a Driver built with the integration left on the rack. The factory
/// deletes it at boot.
class UnavailableFactory final : public task::Factory {
    const std::string integration;
    const std::string reason;
    const std::string legacy_scanner_type;

public:
    UnavailableFactory(
        std::string integration,
        std::string reason,
        std::string legacy_scanner_type = ""
    ):
        integration(std::move(integration)),
        reason(std::move(reason)),
        legacy_scanner_type(std::move(legacy_scanner_type)) {}

    std::vector<std::pair<synnax::task::Task, std::unique_ptr<task::Task>>>
    configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::rack::Rack &rack
    ) override {
        if (this->legacy_scanner_type.empty()) return {};
        if (const auto err = common::delete_legacy_task_by_type(
                rack,
                this->legacy_scanner_type,
                this->integration
            ))
            LOG(ERROR) << "[" << this->integration << "] failed to retrieve the legacy "
                       << this->legacy_scanner_type << " task: " << err;
        return {};
    }

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
