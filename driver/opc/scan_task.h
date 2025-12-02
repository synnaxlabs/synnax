// Copyright 2025 Synnax Labs, Inc.
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
#include "x/cpp/telem/telem.h"
#include "x/cpp/xjson/xjson.h"

#include "driver/opc/connection/connection.h"
#include "driver/opc/types/types.h"
#include "driver/task/common/scan_task.h"
#include "driver/task/task.h"
#include "opc.h"

using json = nlohmann::json;

namespace opc {
inline const std::string SCAN_LOG_PREFIX = "[" + INTEGRATION_NAME + ".scan_task] ";
inline const ::telem::Rate DEFAULT_SCAN_RATE = ::telem::HERTZ * 0.2;
/// @brief Configuration for the OPC UA scanner.
struct ScannerConfig {
    /// @brief Rate at which to check device status.
    ::telem::Rate scan_rate = DEFAULT_SCAN_RATE;
    /// @brief Whether scanning is enabled.
    bool enabled = true;

    ScannerConfig() = default;

    explicit ScannerConfig(xjson::Parser &cfg):
        scan_rate(
            ::telem::Rate(cfg.field<double>("scan_rate", DEFAULT_SCAN_RATE.hz()))
        ),
        enabled(cfg.field<bool>("enabled", true)) {}
};

///@brief The parameters for connecting to and iterating through nodes in the OPC UA
/// server.
struct ScanCommandArgs {
    connection::Config connection;
    std::string node_id;
    opc::NodeId node;

    explicit ScanCommandArgs(xjson::Parser &parser):
        connection(opc::connection::Config(parser.child("connection"))),
        node_id(parser.field<std::string>("node_id", "")) {
        if (node_id.empty())
            node = NodeId(UA_NODEID_NUMERIC(0, UA_NS0ID_OBJECTSFOLDER));
        else
            node = NodeId::parse("node_id", parser);
    }
};

const std::string BROWSE_CMD_TYPE = "browse";
const std::string TEST_CONNECTION_CMD_TYPE = "test_connection";

/// @brief OPC UA scanner implementing the common::Scanner interface.
/// Handles device health monitoring and node browsing for OPC UA servers.
class Scanner final : public common::Scanner {
public:
    Scanner(
        std::shared_ptr<task::Context> ctx,
        synnax::Task task,
        std::shared_ptr<connection::Pool> conn_pool
    );

    /// @brief Returns scanner configuration for common::ScanTask.
    [[nodiscard]] common::ScannerConfig config() const override;

    /// @brief Periodic scan method - checks health of all tracked devices.
    std::pair<std::vector<synnax::Device>, xerrors::Error>
    scan(const common::ScannerContext &scan_ctx) override;

    /// @brief Handle OPC-specific commands (scan nodes, test connection).
    bool exec(
        task::Command &cmd,
        const synnax::Task &task,
        const std::shared_ptr<task::Context> &ctx
    ) override;

private:
    std::shared_ptr<task::Context> ctx;
    synnax::Task task;
    std::shared_ptr<connection::Pool> conn_pool;
    ScannerConfig cfg;

    /// @brief Browse child nodes of a given OPC UA node.
    void browse_nodes(const task::Command &cmd) const;

    /// @brief Test connection to an OPC UA server.
    void test_connection(const task::Command &cmd) const;

    /// @brief Check health of a single device by testing its connection.
    [[nodiscard]] xerrors::Error check_device_health(synnax::Device &dev) const;
};
}
