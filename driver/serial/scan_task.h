// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <functional>
#include <string>
#include <utility>
#include <vector>

#include "driver/common/scan_task.h"
#include "driver/serial/scan.h"
#include "driver/serial/serial.h"

namespace driver::serial {
/// @brief lists the serial ports on the host.
using Lister = std::function<std::pair<std::vector<Info>, x::errors::Error>()>;

/// @brief reports each serial port as a device. A device already tracked for a port
/// keeps its key and settings. A port with no device gets a new one, keyed by the rack
/// and the port path, whose properties hold only the port.
class Scanner final : public common::Scanner {
    synnax::task::Task task;
    Lister list;

public:
    /// @param task the scan task.
    /// @param list lists the host's ports.
    Scanner(synnax::task::Task task, Lister list);

    [[nodiscard]] common::ScannerConfig config() const override;

    std::pair<std::vector<synnax::device::Device>, x::errors::Error>
    scan(const common::ScannerContext &ctx) override;
};
}
