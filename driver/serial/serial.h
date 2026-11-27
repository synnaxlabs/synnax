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
#include <string>
#include <utility>

#include "x/cpp/breaker/breaker.h"

#include "driver/serial/ports.h"
#include "driver/task/task.h"

namespace driver::serial {
/// @brief INTEGRATION_NAME is the name of the serial integration.
const std::string INTEGRATION_NAME = "serial";

/// @brief Factory configures the tasks that read and send ICD messages on serial ports.
/// The tasks on one device share its port.
class Factory final : public task::Factory {
    std::shared_ptr<Ports> ports = std::make_shared<Ports>();

public:
    std::string name() override { return INTEGRATION_NAME; }

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task,
        const std::string &cmd_key,
        x::breaker::Breaker &cancel
    ) override;
};
}
