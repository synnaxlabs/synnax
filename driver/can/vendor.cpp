// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/can/can.h"
#include "driver/can/vendor.h"

namespace driver::can::vendor {
std::pair<std::unique_ptr<x::lib::Shared>, x::errors::Error>
load(const Library &library) {
    if (library.names.empty())
        return {nullptr, {UNSUPPORTED_ERROR, library.unsupported}};
    for (const auto &name: library.names) {
        auto lib = std::make_unique<x::lib::Shared>(name);
        if (lib->load()) return {std::move(lib), x::errors::NIL};
    }
    return {nullptr, driver::errors::missing_lib(library.info)};
}
}
