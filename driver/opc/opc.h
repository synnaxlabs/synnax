// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "open62541/types.h"

#include "x/cpp/json/json.h"

#include "driver/opc/connection/connection.h"
#include "driver/task/task.h"

namespace driver::opc {
const std::string INTEGRATION_NAME = "opc";
const std::string SCAN_TASK_TYPE = "opc_scan";
const std::string WRITE_TASK_TYPE = "opc_write";
const std::string READ_TASK_TYPE = "opc_read";

struct Factory final : task::Factory {
    Factory(): conn_pool_(std::make_shared<connection::Pool>()) {}

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task
    ) override;

    std::vector<std::pair<synnax::task::Task, std::unique_ptr<task::Task>>>
    configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::rack::Rack &rack
    ) override;

    std::string name() override { return INTEGRATION_NAME; }

    std::shared_ptr<connection::Pool> conn_pool() const { return conn_pool_; }

private:
    std::shared_ptr<connection::Pool> conn_pool_;
};
}
