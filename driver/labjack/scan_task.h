// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <cctype>
#include <cstdint>
#include <memory>
#include <optional>
#include <stdexcept>
#include <string>
#include <utility>
#include <vector>

#include "client/cpp/labjack/device/device.h"
#include "client/cpp/labjack/json.gen.h"
#include "client/cpp/synnax.h"
#include "x/cpp/lib/lazy.h"

#include "driver/common/scan_task.h"
#include "driver/common/sync.h"
#include "driver/labjack/device/device.h"
#include "driver/labjack/labjack.h"
#include "driver/labjack/ljm/LJM_Utilities.h"
#include "driver/labjack/store.h"

namespace driver::labjack {
const std::string SCAN_LOG_PREFIX = "[" + INTEGRATION_NAME + ".scan_task] ";

const std::vector SCAN_SKIP_ERRORS = {
    ljm::LJME_AUTO_IPS_FILE_NOT_FOUND,
};

/// @brief configuration for the scan task
struct ScanTaskConfig : ::synnax::labjack::ScanConfig {
    explicit ScanTaskConfig(x::json::Parser &cfg):
        ::synnax::labjack::ScanConfig(::synnax::labjack::ScanConfig::parse(cfg)) {
        if (this->tcp_scan_multiplier == 0)
            cfg.field_err("tcp_scan_multiplier", "must be greater than 0");
    }
};

/// @brief returns the typed model and empty ports of an LJM device type, or nothing
/// when Synnax does not support the type.
inline std::optional<std::pair<std::string, synnax::labjack::device::Ports>>
typed_model(const int device_type) {
    namespace lj = synnax::labjack::device;
    switch (device_type) {
        case LJM_dtT4:
            return std::pair{std::string(lj::MODEL_T_4), lj::Ports{lj::T4Ports{}}};
        case LJM_dtT7:
            return std::pair{std::string(lj::MODEL_T_7), lj::Ports{lj::T7Ports{}}};
        case LJM_dtT8:
            return std::pair{std::string(lj::MODEL_T_8), lj::Ports{lj::T8Ports{}}};
        default:
            return std::nullopt;
    }
}

/// @brief returns the typed connection of an LJM connection type, or nothing when the
/// type is not one that a scan reports.
inline std::optional<std::string> typed_connection(const int connection_type) {
    namespace lj = synnax::labjack::device;
    switch (connection_type) {
        case LJM_ctUSB:
            return lj::CONNECTION_USB;
        case LJM_ctETHERNET:
            return lj::CONNECTION_ETHERNET;
        case LJM_ctWIFI:
            return lj::CONNECTION_WIFI;
        default:
            return std::nullopt;
    }
}

/// @brief finds LabJack hardware and keeps the LabJack devices of the rack in step
/// with it. The scanner writes only the identity fields of a stored device.
class Scanner final : public common::Scanner {
    /// @brief the raw synnax task configuration
    const synnax::task::Task task;
    /// @brief configuration for the scan task
    const ScanTaskConfig cfg;
    /// @brief the device manager for handling LabJack connections
    std::shared_ptr<x::lib::Lazy<device::Manager>> device_manager;
    common::Sync<Device> devices;
    /// @brief the devices that the last TCP scan found. A scan without TCP reports them
    /// too, so a network device stays present between TCP scans.
    std::vector<Device> tcp_devices;

    /// @brief returns true if a device in devices has the serial.
    static bool
    contains(const std::vector<Device> &devices, const std::uint32_t serial) {
        return std::ranges::any_of(devices, [serial](const Device &d) {
            return d.serial == serial;
        });
    }

    [[nodiscard]] common::ScannerConfig config() const override {
        return common::ScannerConfig{
            .log_prefix = SCAN_LOG_PREFIX,
            .signals = {
                synnax::labjack::device::SET_CHANNEL,
                synnax::labjack::device::DELETE_CHANNEL,
            },
        };
    }

    /// @brief appends the devices that a scan of the connection type finds.
    x::errors::Error scan_for(
        device::Manager &manager,
        const int connection_type,
        std::vector<Device> &devices
    ) const {
        int device_types[LJM_LIST_ALL_SIZE];
        int connection_types[LJM_LIST_ALL_SIZE];
        int serial_numbers[LJM_LIST_ALL_SIZE];
        int ip_addresses[LJM_LIST_ALL_SIZE];
        int num_found = 0;

        if (const auto err = manager.list_all(
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
            const auto serial = static_cast<std::uint32_t>(serial_numbers[i]);
            // A device on two connections, such as Ethernet and Wi-Fi, lists twice.
            if (contains(devices, serial)) continue;
            auto model = typed_model(device_types[i]);
            if (!model.has_value()) {
                VLOG(1) << SCAN_LOG_PREFIX << "skipping device " << serial
                        << " with unsupported device type " << device_types[i];
                continue;
            }
            auto connection = typed_connection(connection_types[i]);
            if (!connection.has_value()) {
                VLOG(1) << SCAN_LOG_PREFIX << "skipping device " << serial
                        << " with unsupported connection type " << connection_types[i];
                continue;
            }
            const auto serial_str = std::to_string(serial);
            auto name = model->first;
            std::ranges::transform(name, name.begin(), ::toupper);
            name += "-" +
                    serial_str.substr(
                        serial_str.size() - std::min(serial_str.size(), std::size_t{4})
                    );
            devices.push_back({
                .rack = this->task.rack,
                .name = std::move(name),
                .serial = serial,
                .connection = std::move(*connection),
                .model = std::move(model->first),
                .ports = std::move(model->second),
            });
        }
        return x::errors::NIL;
    }

    std::pair<std::vector<synnax::device::Device>, x::errors::Error>
    scan(const common::ScannerContext &ctx) override {
        auto [manager, err] = this->device_manager->get();
        if (err) return {{}, err};
        std::vector<Device> found;
        if (err = this->scan_for(*manager, LJM_ctUSB, found); err) return {{}, err};
        if (ctx.count % this->cfg.tcp_scan_multiplier == 0) {
            std::vector<Device> tcp;
            if (err = this->scan_for(*manager, LJM_ctTCP, tcp).skip(SCAN_SKIP_ERRORS);
                err)
                return {{}, err};
            this->tcp_devices = std::move(tcp);
        }
        for (const auto &dev: this->tcp_devices)
            if (!contains(found, dev.serial)) found.push_back(dev);
        return {{}, this->devices.sync(std::move(found))};
    }

    void on_signal(
        const synnax::channel::Channel &ch,
        const x::telem::Series &series
    ) override {
        if (ch.name == synnax::labjack::device::SET_CHANNEL)
            for (const auto &sample: series.strings()) {
                x::json::Parser parser(sample);
                const auto dev = Device::parse(parser);
                if (!parser.ok()) {
                    LOG(WARNING) << SCAN_LOG_PREFIX
                                 << "failed to parse a set signal: " << parser.error();
                    continue;
                }
                this->devices.on_set(dev);
            }
        else if (ch.name == synnax::labjack::device::DELETE_CHANNEL)
            for (const auto &key: series.uuids())
                this->devices.on_delete(key);
        else
            throw std::invalid_argument("unknown LabJack signal channel " + ch.name);
    }

public:
    Scanner(
        synnax::task::Task task,
        ScanTaskConfig cfg,
        std::shared_ptr<x::lib::Lazy<device::Manager>> device_manager,
        std::unique_ptr<common::Store<Device>> store
    ):
        task(std::move(task)),
        cfg(std::move(cfg)),
        device_manager(std::move(device_manager)),
        devices(std::move(store), this->task.rack, SCAN_LOG_PREFIX) {}
};
}
