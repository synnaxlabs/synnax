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
#include <set>
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

/// @brief command type for switching a slave to standalone mode, in which only Synnax
/// sets it up and reads its data. The slave changes after a power restart.
const std::string SET_STANDALONE_CMD_TYPE = "set_standalone";

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

/// @brief arguments for the set_standalone command.
struct SetStandaloneArgs {
    /// @brief key of the slave device to switch.
    std::string device;

    explicit SetStandaloneArgs(x::json::Parser &parser):
        device(parser.field<std::string>("device")) {}
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

    /// @brief periodic scan method to discover networks and slaves. Returns
    /// common::PARTIAL_SCAN_ERROR, naming each interface it could not scan or whose
    /// replies a filter blocks, with the slaves of the others.
    std::pair<std::vector<synnax::device::Device>, x::errors::Error>
    scan(const common::ScannerContext &ctx) override;

    /// @brief handles EtherCAT-specific commands.
    bool exec(
        synnax::task::Command &cmd,
        const synnax::task::Task &task,
        const std::shared_ptr<task::Context> &ctx
    ) override;

    /// @brief sets the status of a changed slave device, or an error status when its
    /// properties are invalid.
    void on_device_set(const synnax::device::Device &dev) override;

private:
    std::shared_ptr<task::Context> ctx;
    synnax::task::Task task;
    ScanTaskConfig cfg;
    std::shared_ptr<engine::Pool> pool;
    /// @brief when to probe again an interface that had no slaves.
    struct Probe {
        /// @brief the scan count at which to probe the interface.
        size_t due = 0;
        /// @brief why the interface had no slaves, which each scan reports until the
        /// next probe, or empty when the user cannot fix it.
        std::string failure;
    };
    /// @brief the next probe of each interface that had no slaves.
    std::unordered_map<std::string, Probe> next_probes;
    /// @brief the scan count at which to discover again each interface with an enabled
    /// slave whose PDO discovery failed.
    std::unordered_map<std::string, size_t> next_rediscoveries;

    /// @brief creates the slave device with key for the given slave, and returns
    /// whether its saved config enables it. The status names any invalid property, and
    /// otherwise reflects the slave's live state when health holds an entry for it.
    [[nodiscard]] std::pair<synnax::device::Device, bool> create_slave_device(
        const slave::DiscoveryResult &slave,
        const std::string &key,
        const std::vector<slave::Health> &health,
        const std::string &master_key,
        const common::ScannerContext &scan_ctx
    ) const;

    /// @brief returns the device key of each slave with an identity on the master
    /// with master_key, in the order of slaves, and adds each key to kept. A slave
    /// takes the stored device with its identity: its vendor, product, and serial
    /// number, or with serial number 0, its network and position. Else a slave with a
    /// serial number takes the stored device of its model at its position saved with
    /// no serial number, as before the Driver read CoE serial numbers. Else it gets a
    /// new key that no stored device holds. A slave whose serial number the master
    /// could not read takes only the stored device of its model at its position, and
    /// else gets an empty key. No slave takes a device in kept.
    [[nodiscard]] static std::vector<std::string> slave_keys(
        const std::vector<slave::DiscoveryResult> &slaves,
        const std::string &master_key,
        const common::ScannerContext &scan_ctx,
        std::set<std::string> &kept
    );

    /// @brief gets base properties from existing device or returns empty JSON.
    static nlohmann::json get_existing_properties(
        const std::string &key,
        const common::ScannerContext &scan_ctx
    );

    /// @brief generates a device key for a slave.
    static std::string
    generate_slave_key(const slave::Properties &slave, const std::string &master_key);

    /// @brief handles the test_interface command.
    void test_interface(const synnax::task::Command &cmd) const;

    /// @brief handles the set_standalone command.
    void set_standalone(const synnax::task::Command &cmd) const;
};

}
