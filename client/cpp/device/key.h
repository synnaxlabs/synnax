// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>

namespace synnax::device {
/// @brief Key is a unique identifier for a device. Kept in its own header so task
/// headers can reference it without pulling in the full device types.
using Key = std::string;
}
