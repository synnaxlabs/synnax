// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstddef>
#include <filesystem>
#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "client/cpp/synnax.h"
#include "x/cpp/thread/rt/rt.h"

#include "driver/arc/status/keys.h"
#include "driver/common/common.h"
#include "driver/task/task.h"

namespace driver::arc {
/// @brief integration name for arc runtime.
const std::string INTEGRATION_NAME = "arc";
/// @brief task type for arc runtime tasks.
const std::string TASK_TYPE = "arc";

/// @brief factory for creating arc runtime tasks.
class Factory final : public task::Factory {
    std::shared_ptr<x::thread::rt::Manager> rt_manager;
    /// @brief the keys of resolved status sets, or the error that kept them from
    /// loading, whose data tells the user what to do.
    std::pair<std::shared_ptr<status::Keys>, x::errors::Error> keys;

    /// @brief configures an arc runtime task.
    std::pair<common::ConfigureResult, x::errors::Error> configure(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task
    );

public:
    /// @brief loads the status keys at keys_path that the Core cluster with the key
    /// cluster gave, which hold at most max_keys names. on_create gets each file a
    /// save of the keys creates.
    Factory(
        std::shared_ptr<x::thread::rt::Manager> rt_manager,
        const std::filesystem::path &keys_path,
        const std::string &cluster,
        std::size_t max_keys,
        journal::OnCreate on_create = journal::no_create_hook
    );

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task,
        const std::string &cmd_key,
        x::breaker::Breaker &cancel
    ) override;

    [[nodiscard]] std::string name() override;
};
}
