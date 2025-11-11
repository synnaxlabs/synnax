// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "client/cpp/framer/framer.h"

#include "driver/task/task.h"

namespace common {
/// @brief a common base configuration for tasks.
struct BaseTaskConfig {
    /// @brief whether data saving is enabled for the task.
    bool data_saving;
    /// @brief whether the task should be auto-started after configuration. This
    /// includes automatic start on driver start.
    bool auto_start;

    BaseTaskConfig(BaseTaskConfig &&other) noexcept:
        data_saving(other.data_saving), auto_start(other.auto_start) {}

    BaseTaskConfig(const BaseTaskConfig &other) = delete;
    const BaseTaskConfig &operator=(const BaseTaskConfig &other) = delete;

    explicit BaseTaskConfig(xjson::Parser &parser):
        data_saving(parser.optional<bool>("data_saving", true)),
        auto_start(parser.optional<bool>("auto_start", false)) {}
};

/// @brief a common base configuration result for tasks that is used across various
/// helper functions.
struct ConfigureResult {
    /// @brief the task instantiated by a specific task driver. Should be null if error
    /// is not xerrors::NIL.
    std::unique_ptr<task::Task> task;
    /// @brief whether to auto-start the task if no error occurred.
    bool auto_start = false;
    /// @brief the error that occurred during configuration. If no error occurred, this
    /// field should be set to xerrors::NIL.
    xerrors::Error error = xerrors::NIL;
};

/// @brief Helper to handle config parsing results. Sets auto_start from the config
/// and populates error if parsing failed.
/// @tparam ConfigType The configuration type (must have an auto_start field)
/// @param result The ConfigureResult to populate
/// @param cfg The parsed configuration
/// @param err The parse error (if any)
/// @return true if parsing succeeded (safe to use cfg), false if error occurred
template<typename ConfigType>
bool handle_parse_result(
    ConfigureResult &result,
    const ConfigType &cfg,
    const xerrors::Error &err
) {
    if (err) {
        result.error = err;
        return false;
    }
    result.auto_start = cfg.auto_start;
    return true;
}

/// @brief converts a data_saving boolean to the appropriate WriterMode.
inline synnax::WriterMode data_saving_writer_mode(const bool data_saving) {
    if (data_saving) return synnax::WriterMode::PersistStream;
    return synnax::WriterMode::StreamOnly;
}
}
