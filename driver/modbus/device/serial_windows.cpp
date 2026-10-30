// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cctype>
#include <string>

#include "driver/modbus/device/device.h"

namespace driver::modbus::device {
std::string port_name(const std::string &port) {
    auto path = serial_path(port);
    std::ranges::transform(path, path.begin(), [](const unsigned char c) {
        return static_cast<char>(std::toupper(c));
    });
    return path;
}

std::string canonical_port(const std::string &port) {
    return port_name(port);
}

int PortLock::lock(const std::string &) {
    return 0;
}

void PortLock::unlock() {}
}
