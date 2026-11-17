// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <atomic>
#include <cstdint>

namespace x::bench {
/// @brief the calls to and bytes from every form of the global operator new and new[].
/// Linking this library replaces them so that they count.
// NOLINTBEGIN(cppcoreguidelines-avoid-non-const-global-variables): the global operator
// new counts into these.
extern std::atomic<int64_t> alloc_count;
extern std::atomic<int64_t> alloc_bytes;
// NOLINTEND(cppcoreguidelines-avoid-non-const-global-variables)
}
