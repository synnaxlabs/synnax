// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "open62541/types.h"

#include "x/cpp/xjson/xjson.h"

#include "driver/opc/util/conn_pool.h"
#include "driver/task/task.h"

namespace opc {
const std::string INTEGRATION_NAME = "opc";
const std::string SCAN_TASK_TYPE = "opc_scan";
const std::string WRITE_TASK_TYPE = "opc_write";
const std::string READ_TASK_TYPE = "opc_read";

struct Factory final : public task::Factory {
    Factory(): conn_pool_(std::make_shared<util::ConnectionPool>()) {}

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) override;

    std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task>>>
    configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Rack &rack
    ) override;

    std::string name() override { return INTEGRATION_NAME; }

    std::shared_ptr<util::ConnectionPool> conn_pool() const { return conn_pool_; }

private:
    std::shared_ptr<util::ConnectionPool> conn_pool_;
};
}
