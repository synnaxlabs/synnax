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
#include <windows.h>
#include "glog/logging.h"
#include "nlohmann/json.hpp"

namespace labjack {
inline bool does_dll_exist(const char *dll_path) {
    HMODULE hModule = LoadLibrary(dll_path);
    if (hModule == NULL) {
        LOG(ERROR) << "[labjack] " << dll_path << " not found";
        return false;
    }
    FreeLibrary(hModule);
    return true;
}
}
