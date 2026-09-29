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
#include <utility>
#include <vector>

#include "driver/common/scan_task.h"
#include "driver/tcp/client.h"
#include "driver/tcp/tcp.h"

namespace driver::tcp {
/// @brief reports whether each tracked TCP device accepts a connection.
class Scanner final : public common::Scanner {
    synnax::task::Task task;
    Config connection;

public:
    /// @param task the scan task.
    /// @param connection how each check connects.
    Scanner(synnax::task::Task task, const Config &connection);

    [[nodiscard]] common::ScannerConfig config() const override;

    std::pair<std::vector<synnax::device::Device>, x::errors::Error>
    scan(const common::ScannerContext &ctx) override;
};
}
