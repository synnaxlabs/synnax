// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.
#pragma once

#include <iostream>
#include "glog/logging.h"

inline bool does_dll_exist(const char *dll_path) {
    return false;
}

inline void log_dll_error(const std::shared_ptr<task::Context> &ctx,
                          const synnax::Task &task) {
    LOG(ERROR) << "[ni] NI acquisition and control not supported on Linux or macOS" <<
            std::endl;
    json j = {
        {"error", " NI acquisition and control not supported on Linux or macOS"}
    };
    ctx->set_state({
        .task = task.key,
        .variant = "error",
        .details = j
    });
}
