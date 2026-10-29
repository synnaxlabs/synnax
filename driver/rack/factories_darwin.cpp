// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "absl/log/log.h"

#include "driver/common/factory.h"
#include "driver/rack/factories.h"

namespace driver::rack {
std::vector<std::string> platform_integrations() {
    return {};
}

/// @brief deletes the NI scanner that a Driver built with NI left on the rack, as the
/// macOS build has no NI integration to run it.
class LegacyNIFactory final : public task::Factory {
public:
    std::vector<std::pair<synnax::task::Task, std::unique_ptr<task::Task>>>
    configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::rack::Rack &rack
    ) override {
        const auto err = common::delete_legacy_task_by_type(rack, "ni_scanner", "ni");
        if (err) LOG(ERROR) << "[ni] failed to retrieve the legacy NI scanner: " << err;
        return {};
    }

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task,
        const std::string &cmd_key
    ) override {
        return {nullptr, false};
    }

    std::string name() override { return "ni"; }
};

void configure_ni(const Config &, FactoryList &factories) {
    factories.push_back(std::make_unique<LegacyNIFactory>());
}
}
