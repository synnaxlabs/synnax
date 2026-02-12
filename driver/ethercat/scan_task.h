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
#include "x/cpp/json/json.h"

#include "driver/common/scan_task.h"
#include "driver/ethercat/ethercat.h"
#include "driver/ethercat/slave/slave.h"
#include "driver/task/task.h"

namespace driver::ethercat {

/// @brief log prefix for scan task messages.
const std::string SCAN_LOG_PREFIX = "[ethercat.scan_task] ";

/// @brief command type for testing a master.
const std::string TEST_INTERFACE_CMD_TYPE = "test_interface";

/// @brief configuration for the EtherCAT scan task.
struct ScanTaskConfig : common::ScanTaskConfig {
    ScanTaskConfig() = default;

    explicit ScanTaskConfig(x::json::Parser &cfg): common::ScanTaskConfig(cfg) {}
};

/// @brief arguments for the test_interface command.
struct TestInterfaceArgs {
    /// @brief master key to test (e.g., "igh:0" or "eth0").
    std::string interface;

    explicit TestInterfaceArgs(x::json::Parser &parser):
        interface(parser.field<std::string>("interface")) {}
};

/// @brief scanner implementation for EtherCAT device discovery.
class Scanner final : public common::Scanner {
public:
    Scanner(
        std::shared_ptr<task::Context> ctx,
        synnax::task::Task task,
        ScanTaskConfig cfg,
        std::shared_ptr<engine::Pool> pool
    );

    /// @brief returns scanner configuration for common::ScanTask.
    [[nodiscard]] common::ScannerConfig config() const override;

    /// @brief lifecycle method called when the scan task starts.
    x::errors::Error start() override;

    /// @brief lifecycle method called when the scan task stops.
    x::errors::Error stop() override;

    /// @brief periodic scan method to discover networks and slaves.
    std::pair<std::vector<synnax::device::Device>, x::errors::Error>
    scan(const common::ScannerContext &ctx) override;

    /// @brief handles EtherCAT-specific commands.
    bool exec(
        task::Command &cmd,
        const synnax::task::Task &task,
        const std::shared_ptr<task::Context> &ctx
    ) override;

    /// @brief handles device updates to sync enabled flag to engine.
    void on_device_set(const synnax::device::Device &dev) override;

private:
    std::shared_ptr<task::Context> ctx;
    synnax::task::Task task;
    ScanTaskConfig cfg;
    std::shared_ptr<engine::Pool> pool;
    std::unordered_map<std::string, size_t> last_slave_counts;

    /// @brief creates a slave device for the given slave.
    synnax::device::Device create_slave_device(
        const slave::DiscoveryResult &slave,
        const std::string &master_key,
        const common::ScannerContext &scan_ctx
    ) const;

    /// @brief gets base properties from existing device or returns empty JSON.
    static nlohmann::json get_existing_properties(
        const std::string &key,
        const common::ScannerContext &scan_ctx
    );

    /// @brief generates a device key for a slave.
    static std::string
    generate_slave_key(const slave::Properties &slave, const std::string &master_key);

    /// @brief handles the test_interface command.
    void test_interface(const task::Command &cmd) const;
};

}
