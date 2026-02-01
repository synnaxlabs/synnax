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
/// Log prefix for scan task messages.
const std::string SCAN_LOG_PREFIX = "[ethercat.scan_task] ";

/// Command type for testing an interface.
const std::string TEST_INTERFACE_CMD_TYPE = "test_interface";

/// Network interface information from adapter enumeration.
struct InterfaceInfo {
    /// Interface name (e.g., "eth0", "enp3s0").
    std::string name;
    /// Human-readable description.
    std::string description;
};

/// Configuration for the EtherCAT scan task.
struct ScanTaskConfig : common::ScanTaskConfig {
    /// Backend to use: "auto", "soem", or "igh".
    std::string backend = "auto";

    ScanTaskConfig() = default;

    explicit ScanTaskConfig(xjson::Parser &cfg):
        common::ScanTaskConfig(cfg),
        backend(cfg.field<std::string>("backend", "auto")) {}
};

/// Arguments for the test_interface command.
struct TestInterfaceArgs {
    /// Network interface to test.
    std::string interface;

    explicit TestInterfaceArgs(xjson::Parser &parser):
        interface(parser.field<std::string>("interface")) {}
};

/// Scanner implementation for EtherCAT device discovery.
///
/// The scanner discovers EtherCAT networks and slaves, creating Synnax devices
/// that represent them. It coordinates with the Factory to use cached slave
/// information from active CyclicEngines when available.
class Scanner final : public common::Scanner {
public:
    Scanner(
        std::shared_ptr<task::Context> ctx,
        synnax::Task task,
        ScanTaskConfig cfg,
        std::shared_ptr<engine::Pool> pool
    );

    /// Returns scanner configuration for common::ScanTask.
    [[nodiscard]] common::ScannerConfig config() const override;

    /// Lifecycle method called when the scan task starts.
    xerrors::Error start() override;

    /// Lifecycle method called when the scan task stops.
    xerrors::Error stop() override;

    /// Periodic scan method to discover networks and slaves.
    std::pair<std::vector<synnax::Device>, xerrors::Error>
    scan(const common::ScannerContext &ctx) override;

    /// Handle EtherCAT-specific commands.
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
    /// Tracks slave count per interface to avoid repetitive logging.
    std::unordered_map<std::string, size_t> last_slave_counts;

    /// Enumerates all network interfaces that could have EtherCAT slaves.
    std::vector<InterfaceInfo> enumerate_interfaces();

    /// Probes an interface for EtherCAT slaves.
    std::pair<std::vector<SlaveInfo>, xerrors::Error>
    probe_interface(const std::string &interface) const;

    /// Creates a network device for the given interface and slaves.
    synnax::Device create_network_device(
        const InterfaceInfo &iface,
        const std::vector<SlaveInfo> &slaves
    );

    /// Creates a slave device for the given slave.
    synnax::Device
    create_slave_device(const SlaveInfo &slave, const std::string &network_interface);

    /// Generates a device key for a network.
    std::string generate_network_key(const std::string &interface);

    /// Generates a device key for a slave.
    std::string
    generate_slave_key(const SlaveInfo &slave, const std::string &interface);

    /// Handles the test_interface command.
    void test_interface(const task::Command &cmd) const;
};
}
