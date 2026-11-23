// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "x/cpp/spatial/types.gen.h"

namespace x::spatial {
/// @brief returns the span of the bounds.
/// @param bounds the bounds to measure.
/// @returns upper - lower, which is negative when lower is above upper.
[[nodiscard]] inline double span(const ClosedBounds &bounds) {
    return bounds.upper - bounds.lower;
}
}
