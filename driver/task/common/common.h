// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "driver/task/task.h"

namespace common {
struct TaskConfig {
    bool data_saving;
    bool auto_start;

    TaskConfig(TaskConfig &&other) noexcept:
        data_saving(other.data_saving), auto_start(other.auto_start) {}

    TaskConfig(const TaskConfig &other) = delete;
    const TaskConfig &operator=(const TaskConfig &other) = delete;

    explicit TaskConfig(xjson::Parser &parser):
        data_saving(parser.optional<bool>("data_saving", false)),
        auto_start(parser.optional<bool>("auto_start", true)) {}
};

struct ConfigureResult {
    std::unique_ptr<task::Task> task;
    bool auto_start = false;
    xerrors::Error error = xerrors::NIL;
};
}