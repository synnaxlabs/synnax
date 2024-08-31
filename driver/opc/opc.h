// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "driver/config/config.h"
#include "driver/task/task.h"
#include "include/open62541/types.h"

namespace opc {
    const std::string INTEGRATION_NAME = "opc";

    class Factory final : public task::Factory {
        std::pair<std::unique_ptr<task::Task>, bool> configureTask(
                const std::shared_ptr<task::Context> &ctx,
                const synnax::Task &task
        ) override;

        std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task> > >
        configureInitialTasks(const std::shared_ptr<task::Context> &ctx,
                              const synnax::Rack &rack) override;
    };
}
