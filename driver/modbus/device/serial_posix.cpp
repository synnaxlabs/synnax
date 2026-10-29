// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <filesystem>
#include <string>
#include <system_error>

#include "driver/modbus/device/device.h"

namespace driver::modbus::device {
std::string canonical_port(const std::string &port) {
    std::error_code ec;
    const auto path = std::filesystem::weakly_canonical(port, ec);
    // A path that cannot resolve also fails to open, and the open reports why.
    if (ec) return port;
    return path.string();
}
}
