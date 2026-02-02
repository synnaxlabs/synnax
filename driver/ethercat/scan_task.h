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
#include <unordered_map>
#include <utility>
#include <vector>

#include "client/cpp/synnax.h"
#include "x/cpp/xjson/xjson.h"

#include "driver/ethercat/ethercat.h"
#include "driver/ethercat/master/slave_info.h"
#include "driver/task/common/scan_task.h"
#include "driver/task/task.h"

namespace ethercat {

/// @brief log prefix for scan task messages.
const std::string SCAN_LOG_PREFIX = "[ethercat.scan_task] ";

/// @brief command type for testing a master.
const std::string TEST_INTERFACE_CMD_TYPE = "test_interface";

/// @brief configuration for the EtherCAT scan task.
struct ScanTaskConfig : common::ScanTaskConfig {
    ScanTaskConfig() = default;

    explicit ScanTaskConfig(xjson::Parser &cfg): common::ScanTaskConfig(cfg) {}
};

/// @brief arguments for the test_interface command.
struct TestInterfaceArgs {
    /// @brief master key to test (e.g., "igh:0" or "eth0").
    std::string interface;

    explicit TestInterfaceArgs(xjson::Parser &parser):
        interface(parser.field<std::string>("interface")) {}
};

/// @brief scanner implementation for EtherCAT device discovery.
class Scanner final : public common::Scanner {
public:
    Scanner(
        std::shared_ptr<task::Context> ctx,
        synnax::Task task,
        ScanTaskConfig cfg,
        std::shared_ptr<engine::Pool> pool
    );

    /// @brief returns scanner configuration for common::ScanTask.
    [[nodiscard]] common::ScannerConfig config() const override;

    /// @brief lifecycle method called when the scan task starts.
    xerrors::Error start() override;

    /// @brief lifecycle method called when the scan task stops.
    xerrors::Error stop() override;

    /// @brief periodic scan method to discover networks and slaves.
    std::pair<std::vector<synnax::Device>, xerrors::Error>
    scan(const common::ScannerContext &ctx) override;

    /// @brief handles EtherCAT-specific commands.
    bool exec(
        task::Command &cmd,
        const synnax::Task &task,
        const std::shared_ptr<task::Context> &ctx
    ) override;

private:
    std::shared_ptr<task::Context> ctx;
    synnax::Task task;
    ScanTaskConfig cfg;
    std::shared_ptr<engine::Pool> pool;
    std::unordered_map<std::string, size_t> last_slave_counts;

    /// @brief creates a slave device for the given slave.
    synnax::Device create_slave_device(
        const SlaveInfo &slave,
        const std::string &master_key,
        const common::ScannerContext &scan_ctx
    );

    /// @brief gets base properties from existing device or returns empty JSON.
    static nlohmann::json get_existing_properties(
        const std::string &key,
        const common::ScannerContext &scan_ctx
    );

    /// @brief generates a device key for a slave.
    static std::string
    generate_slave_key(const SlaveInfo &slave, const std::string &master_key);

    /// @brief handles the test_interface command.
    void test_interface(const task::Command &cmd) const;
};

}
