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
#include <span>
#include <string>
#include <utility>

#include "x/cpp/errors/errors.h"
#include "x/cpp/lib/lib.h"

#include "driver/errors/errors.h"

namespace driver::bus::vendor {
/// @brief a backend whose vendor library loads but that the Driver cannot drive yet.
const x::errors::Error UNSUPPORTED_ERROR = errors::CONFIGURATION_ERROR.sub(
    "unsupported"
);

/// @brief loads a vendor shared library and checks that it has each function.
/// @returns the missing library error of info when the library or a function is
/// absent.
std::pair<std::unique_ptr<x::lib::Shared>, x::errors::Error> load(
    const std::string &name,
    const LibraryInfo &info,
    std::span<const char *const> functions = {}
);

/// @returns the UNSUPPORTED_ERROR of a backend.
x::errors::Error unsupported(const std::string &backend);
}
