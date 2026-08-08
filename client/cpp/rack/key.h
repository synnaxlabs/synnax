// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>

namespace synnax::rack {
/// @brief Key is a composite identifier for a rack. The high 16 bits contain the core
/// node key, and the low 16 bits contain the local sequential key. Kept in its own
/// header so task headers can reference it without pulling in the full rack types.
using Key = std::uint32_t;
}
