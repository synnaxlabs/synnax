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
#include <vector>

#include "nlohmann/json.hpp"
#include "open62541/types.h"

#include "client/cpp/synnax.h"
#include "x/cpp/json/json.h"
#include "x/cpp/telem/telem.h"

#include "driver/opc/connection/connection.h"
#include "driver/opc/types/types.h"
#include "driver/task/common/scan_task.h"
#include "driver/task/task.h"
#include "opc.h"

using json = x::json::json;

namespace driver::opc {
inline const std::string SCAN_LOG_PREFIX = "[" + INTEGRATION_NAME + ".scan_task] ";
/// @brief Configuration for the OPC UA scanner.
struct ScanTaskConfig : driver::task::common::ScanTaskConfig {
    ScanTaskConfig() = default;
    explicit ScanTaskConfig(x::json::Parser &cfg):
        driver::task::common::ScanTaskConfig(cfg) {}
};

///@brief The parameters for connecting to and iterating through nodes in the OPC UA
/// server.
struct ScanCommandArgs {
    connection::Config connection;
    std::string node_id;
    driver::opc::NodeId node;

    explicit ScanCommandArgs(x::json::Parser &parser):
        connection(driver::opc::connection::Config(parser.child("connection"))),
        node_id(parser.field<std::string>("node_id", "")) {
        if (node_id.empty())
            node = NodeId(UA_NODEID_NUMERIC(0, UA_NS0ID_OBJECTSFOLDER));
        else
            node = NodeId::parse("node_id", parser);
    }
};

const std::string BROWSE_CMD_TYPE = "browse";
const std::string TEST_CONNECTION_CMD_TYPE = "test_connection";

/// @brief OPC UA scanner implementing the driver::task::common::Scanner interface.
/// Handles device health monitoring and node browsing for OPC UA servers.
class Scanner final : public driver::task::common::Scanner {
public:
    Scanner(
        std::shared_ptr<driver::task::Context> ctx,
        synnax::task::Task task,
        std::shared_ptr<connection::Pool> conn_pool
    );

    /// @brief Returns scanner configuration for driver::task::common::ScanTask.
    [[nodiscard]] driver::task::common::ScannerConfig config() const override;

    /// @brief Periodic scan method - checks health of all tracked devices.
    std::pair<std::vector<synnax::device::Device>, x::errors::Error>
    scan(const driver::task::common::ScannerContext &scan_ctx) override;

    /// @brief Handle OPC-specific commands (scan nodes, test connection).
    bool exec(
        synnax::task::Command &cmd,
        const synnax::task::Task &task,
        const std::shared_ptr<driver::task::Context> &ctx
    ) override;

private:
    std::shared_ptr<driver::task::Context> ctx;
    synnax::task::Task task;
    std::shared_ptr<connection::Pool> conn_pool;
    ScanTaskConfig cfg;

    /// @brief Browse child nodes of a given OPC UA node.
    void browse_nodes(const synnax::task::Command &cmd) const;

    /// @brief Test connection to an OPC UA server.
    void test_connection(const synnax::task::Command &cmd) const;

    /// @brief Check health of a single device by testing its connection.
    [[nodiscard]] x::errors::Error check_device_health(synnax::device::Device &dev);
};
}
