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
#include <thread>

#include "nlohmann/json.hpp"

#include "client/cpp/synnax.h"
#include "x/cpp/breaker/breaker.h"

/// internal
#include "device/device.h"
#include "driver/task/task.h"

// Currently supports: T7, T4, T5, Digit products.

namespace labjack {
const std::string SCAN_CMD_TYPE = "scan";
const std::string STOP_CMD_TYPE = "stop";

/// @brief an extension of the default synnax device that includes LabJack properties
struct Device : synnax::Device {
    /// @brief the serial number of the device
    int serial_number = 0;
    /// @brief the device type (T7, T4, etc)
    std::string device_type;
    /// @brief the connection type (USB, TCP, etc) 
    std::string connection_type;

    Device() = default;

    explicit Device(
        const synnax::Device &device,
        const int serial_number,
        std::string device_type,
        std::string connection_type
    ): synnax::Device(device),
       serial_number(serial_number),
       device_type(std::move(device_type)),
       connection_type(std::move(connection_type)) {
    }

    /// @brief returns the synnax device representation with json properties
    [[nodiscard]] synnax::Device to_synnax() const {
        return synnax::Device(
            this->key,
            this->name,
            this->rack,
            this->location,
            this->identifier,
            this->make,
            this->model,
            nlohmann::to_string(json{
                {"serial_number", this->serial_number},
                {"device_type", this->device_type},
                {"connection_type", this->connection_type}
            })
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
        tcp_scan_multiplier(cfg.optional<int>("tcp_scan_multiplier", 10)) {
    }
};


class ScanTask final : public task::Task {
    /// @brief the raw synnax task configuration
    const synnax::Task task;
    /// @brief configuration for the scan task
    const ScanTaskConfig cfg;
    /// @brief the breaker for managing thread lifecycle
    breaker::Breaker breaker;
    /// @brief the task context to communicate state updates
    std::shared_ptr<task::Context> ctx;
    /// @brief the scan thread
    std::thread thread;
    /// @brief the current list of scanned devices
    std::unordered_map<std::string, Device> devices;
    /// @brief the device manager for handling LabJack connections
    std::shared_ptr<device::Manager> device_manager;
    /// @brief the current task state
    task::State state;

    /// @brief scans for devices with the given type and connection
    xerrors::Error scan_for(int device_type, int connection_type);
    /// @brief updates devices in the remote Synnax cluster
    xerrors::Error update_remote();
    /// @brief the main scan task run loop
    void run();

    xerrors::Error scan_internal(const size_t tcp_counter) {
        if (const auto err = this->scan_for(LJM_dtANY, LJM_ctUSB))
            return err;
        if (tcp_counter % this->cfg.tcp_scan_multiplier == 0)
            if (const auto err = this->scan_for(LJM_dtANY, LJM_ctTCP))
                return err;
        return this->update_remote();
    }

public:
    explicit ScanTask(
        const std::shared_ptr<task::Context> &ctx,
        synnax::Task task,
        ScanTaskConfig cfg,
        std::shared_ptr<device::Manager> device_manager
    );

    /// @brief implements task::Task to execute commands
    void exec(task::Command &cmd) override;
    /// @brief stops the scan task
    void stop(bool will_reconfigure) override;
    /// @brief starts the scan task
    void start();
    /// @brief performs a single scan of hardware
    xerrors::Error scan();
    /// @brief returns the task name
    std::string name() override { return task.name; }

    /// @brief creates a new scan task from configuration
    static std::pair<std::unique_ptr<task::Task>, xerrors::Error> configure(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task,
        const std::shared_ptr<device::Manager>& dev_manager
    );
};
};
