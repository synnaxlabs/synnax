// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstddef>
#include <functional>
#include <memory>

#include "x/cpp/telem/telem.h"

namespace arc::runtime::testutil {
/// @brief returns a node::Context reserve_stamps callback whose reservations start at
/// first and never overlap, as within one scheduler cycle.
inline std::function<x::telem::TimeStamp(size_t)>
reserve_stamps(const x::telem::TimeStamp first = x::telem::TimeStamp(0)) {
    auto next = std::make_shared<x::telem::TimeStamp>(first);
    return [next](const size_t n) {
        const auto reserved = *next;
        *next = *next + static_cast<int64_t>(n);
        return reserved;
    };
}
}
