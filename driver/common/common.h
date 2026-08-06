// Copyright 2026 Synnax Labs, Inc.
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

/// @brief converts a data_saving boolean to the appropriate WriterMode.
inline synnax::framer::WriterMode data_saving_writer_mode(const bool data_saving) {
    if (data_saving) return synnax::framer::WriterMode::PersistStream;
    return synnax::framer::WriterMode::StreamOnly;
}

}
