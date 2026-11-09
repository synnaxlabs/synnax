// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>

#include "client/cpp/framer/framer.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/telem/telem.h"

#include "driver/task/task.h"

namespace driver::common {
/// @brief a common base configuration result for tasks that is used across various
/// helper functions.
struct ConfigureResult {
    /// @brief the task instantiated by a specific task driver. Should be null if error
    /// is not x::errors::NIL.
    std::unique_ptr<task::Task> task;
    /// @brief whether to auto-start the task if no error occurred.
    bool auto_start = false;
};

/// @brief returns a breaker configuration that retries without a limit, at most 5 s
/// apart, so a task waits for a lost device to return.
inline x::breaker::Config unlimited_retry_config(const std::string &name) {
    return x::breaker::Config{
        .name = name,
        .max_retries = x::breaker::RETRY_INFINITELY,
        .max_interval = 5 * x::telem::SECOND,
    };
}

/// @brief converts a data_saving_disabled boolean to the appropriate WriterMode.
inline synnax::framer::WriterMode
data_saving_writer_mode(const bool data_saving_disabled) {
    if (data_saving_disabled) return synnax::framer::WriterMode::StreamOnly;
    return synnax::framer::WriterMode::PersistStream;
}
}
