// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <string>

/// external
#include "ljm/LJM_Utilities.h"
#include "nlohmann/json.hpp"

/// module
#include "client/cpp/synnax.h"

/// internal
#include "device/device.h"
#include "driver/labjack/labjack.h"
#include "driver/task/common/scan_task.h"

namespace labjack {
/// @brief an extension of the default synnax device that includes LabJack
/// properties
struct Device : synnax::Device {
    /// @brief the connection type (USB, TCP, etc)
    std::string connection_type;

    Device() = default;

    explicit Device(const synnax::Device &device, std::string connection_type):
        synnax::Device(device), connection_type(std::move(connection_type)) {}

    /// @brief returns the synnax device representation with json properties
    [[nodiscard]] synnax::Device to_synnax() const {
        return synnax::Device(
            this->key,
            this->name,
            this->rack,
            this->location,
            this->make,
            this->model,
            nlohmann::to_string(json{{"connection_type", this->connection_type}})
        );
    }
};

/// @brief the default rate for scanning devices
const auto DEFAULT_SCAN_RATE = telem::Rate(0.5);

/// @brief configuration for the scan task
struct ScanTaskConfig {
    /// @brief the rate at which to scan for devices
    const telem::Rate rate;
    /// @brief whether the scan task is enabled
    const bool enabled;
    /// @brief how often to scan TCP devices relative to USB devices
    const int tcp_scan_multiplier;

    explicit ScanTaskConfig(xjson::Parser &cfg):
        rate(telem::Rate(cfg.optional<double>("rate", DEFAULT_SCAN_RATE.hz()))),
        enabled(cfg.optional<bool>("enabled", true)),
        tcp_scan_multiplier(cfg.optional<int>("tcp_scan_multiplier", 10)) {}
};


class Scanner final : public common::Scanner {
    /// @brief the raw synnax task configuration
    const synnax::Task task;
    /// @brief configuration for the scan task
    const ScanTaskConfig cfg;
    /// @brief the device manager for handling LabJack connections
    std::shared_ptr<device::Manager> device_manager;

    /// @brief scans for devices with the given type and connection
    xerrors::Error
    scan_for(int connection_type, std::vector<synnax::Device> &devices) const {
        int device_types[LJM_LIST_ALL_SIZE];
        int connection_types[LJM_LIST_ALL_SIZE];
        int serial_numbers[LJM_LIST_ALL_SIZE];
        int ip_addresses[LJM_LIST_ALL_SIZE];
        int num_found = 0;

        if (const auto err = device_manager->list_all(
                LJM_dtANY,
                connection_type,
                &num_found,
                device_types,
                connection_types,
                serial_numbers,
                ip_addresses
            ))
            return err;

        for (int i = 0; i < num_found; i++) {
            const auto serial_str = std::to_string(serial_numbers[i]);
            const auto device_type_str = std::string(NumberToDeviceType(device_types[i])
            );
            const auto conn_type_str = std::string(
                NumberToConnectionType(connection_types[i])
            );

            auto last_four = serial_str.length() >= 4
                               ? serial_str.substr(serial_str.length() - 4)
                               : serial_str;
            auto name = device_type_str + "-" + last_four;

            auto sy_dev = synnax::Device(
                serial_str,
                name,
                synnax::task_key_rack(this->task.key),
                conn_type_str,
                MAKE,
                device_type_str,
                "" // Properties will be set in Device constructor
            );

            auto dev = Device(sy_dev, conn_type_str);
            devices.push_back(dev.to_synnax());
        }
        return xerrors::NIL;
    }

    std::pair<std::vector<synnax::Device>, xerrors::Error>
    scan(const common::ScannerContext &ctx) override {
        std::vector<synnax::Device> devs;
        xerrors::Error err;
        if (err = this->scan_for(LJM_ctUSB, devs); err) return {devs, err};
        if (ctx.count % this->cfg.tcp_scan_multiplier == 0)
            err = this->scan_for(LJM_ctTCP, devs);
        return {devs, err};
    }

public:
    explicit Scanner(
        synnax::Task task,
        ScanTaskConfig cfg,
        std::shared_ptr<device::Manager> device_manager
    ):
        task(std::move(task)),
        cfg(std::move(cfg)),
        device_manager(std::move(device_manager)) {}
};
}
