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
#include <utility>
#include <vector>

#include "driver/can/can.h"
#include "driver/common/scan_task.h"

namespace driver::can {
/// @brief reports each channel that a backend finds as a device. A device already
/// tracked for a channel keeps its key and settings. A channel with no device gets a
/// new one, keyed by the rack, backend, and channel, with default bus settings. A
/// backend that cannot scan, such as one whose library is not installed, is skipped.
class Scanner final : public common::Scanner {
    synnax::task::Task task;
    std::shared_ptr<const Backends> backends;

public:
    /// @param task the scan task.
    /// @param backends the backends to scan.
    Scanner(synnax::task::Task task, std::shared_ptr<const Backends> backends);

    [[nodiscard]] common::ScannerConfig config() const override;

    std::pair<std::vector<synnax::device::Device>, x::errors::Error>
    scan(const common::ScannerContext &ctx) override;
};
}
