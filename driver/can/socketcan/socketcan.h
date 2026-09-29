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

#include "driver/can/can.h"

/// @brief CAN through the Linux kernel's SocketCAN interfaces. The interface keeps the
/// bitrate and modes that `ip link` gave it, so open ignores the bitrates in the
/// properties.
namespace driver::can::socketcan {
/// @returns the SocketCAN backend on Linux, and an Unavailable backend on every other
/// platform.
[[nodiscard]] std::shared_ptr<can::Backend> load();
}
