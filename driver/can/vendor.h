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

#include "x/cpp/errors/errors.h"
#include "x/cpp/lib/lib.h"

#include "driver/errors/errors.h"

/// @brief the vendor libraries that CAN backends load at runtime. Each backend defines
/// its library in one source file per platform, which its BUILD file selects.
namespace driver::can::vendor {
/// @brief the shared library a backend loads on the platform the build targets.
struct Library {
    /// @brief the file names to try, in order. Empty when the platform lacks the
    /// library.
    std::vector<std::string> names;
    /// @brief the library's name and download page.
    LibraryInfo info;
    /// @brief why the platform cannot run the backend, when names is empty.
    std::string unsupported;
};

/// @returns the first of the library's names that loads, UNSUPPORTED_ERROR when it has
/// no names, and x::lib::LOAD_ERROR with its download page when none loads.
[[nodiscard]] std::pair<std::unique_ptr<x::lib::Shared>, x::errors::Error>
load(const Library &library);
}
