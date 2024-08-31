// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.
#pragma once

#include <iostream>
#include <windows.h>
#include "glog/logging.h"
#include "nlohmann/json.hpp"

inline bool does_dll_exist(const char *dll_path) {
    HMODULE hModule = LoadLibrary(dll_path);
    if (hModule == NULL) {
        LOG(ERROR) << "[ni] " << dll_path << " not found";
        return false;
    }
    FreeLibrary(hModule);
    return true;
}

inline void log_dll_error(const std::shared_ptr<task::Context> &ctx,
                          const synnax::Task &task) {
    LOG(ERROR) << "[ni] Required NI DLLs not found, cannot configure task." <<
            std::endl;
    json j = {
        {
            "error",
            "Required NI DLLs not found. To find more information on how to install the required DLLS, please visit https://www.ni.com/en/support/downloads/driver"
        }
    };
    ctx->setState({
        .task = task.key,
        .variant = "error",
        .details = j
    });
}
