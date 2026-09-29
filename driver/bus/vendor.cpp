// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/bus/vendor.h"

namespace driver::bus::vendor {
std::pair<std::unique_ptr<x::lib::Shared>, x::errors::Error> load(
    const std::string &name,
    const LibraryInfo &info,
    const std::span<const char *const> functions
) {
    auto lib = std::make_unique<x::lib::Shared>(name);
    if (!lib->load()) return {nullptr, errors::missing_lib(info)};
    for (const auto *fn: functions)
        if (lib->get_func_ptr(fn) == nullptr)
            return {nullptr, errors::missing_lib(info)};
    return {std::move(lib), x::errors::NIL};
}

x::errors::Error unsupported(const std::string &backend) {
    return x::errors::Error(
        UNSUPPORTED_ERROR,
        "the " + backend +
            " backend is not supported yet. Its vendor library loaded, but the "
            "Driver does not drive it"
    );
}
}
