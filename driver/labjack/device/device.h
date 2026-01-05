// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <map>
#include <mutex>

#include "glog/logging.h"

#include "driver/labjack/ljm/LabJackM.h"
#include "driver/labjack/ljm/api.h"
#include "driver/labjack/ljm/errors.h"

namespace device {
class Device {
public:
    virtual ~Device() = default;

    virtual xerrors::Error
    e_stream_read(double *data, int *dev_scan_backlog, int *ljm_scan_backlog) const = 0;

    [[nodiscard]] virtual xerrors::Error e_stream_stop() const = 0;

    [[nodiscard]] virtual xerrors::Error
    e_write_addr(int addr, int type, double value) const = 0;

    virtual xerrors::Error e_write_addrs(
        size_t num_frames,
        const int *addrs,
        const int *types,
        const double *values,
        int *error_addrs
    ) const = 0;

    [[nodiscard]] virtual xerrors::Error
    start_interval(int interval_handle, int microseconds) const = 0;

    [[nodiscard]] virtual xerrors::Error clean_interval(int interval_handle) const = 0;

    [[nodiscard]] virtual xerrors::Error
    e_write_name(const char *Name, double value) const = 0;

    [[nodiscard]] virtual xerrors::Error e_write_names(
        size_t num_frames,
        const char **names,
        const double *values,
        int *err_addr
    ) const = 0;

    [[nodiscard]] virtual xerrors::Error names_to_addrs(
        size_t num_frames,
        const char **names,
        int *addrs,
        int *types
    ) const = 0;

    [[nodiscard]] virtual xerrors::Error
    wait_for_next_interval(int interval_handle, int *skipped_intervals) const = 0;

    [[nodiscard]] virtual xerrors::Error e_read_names(
        size_t num_frames,
        const char **a_names,
        double *a_values,
        int *error_addr
    ) const = 0;

    [[nodiscard]] virtual xerrors::Error
    e_read_name(const char *name, double *value) const = 0;

    [[nodiscard]] virtual xerrors::Error e_stream_start(
        size_t scans_per_read,
        size_t num_addrs,
        const int *scan_list,
        double *scan_rate
    ) const = 0;
};

class LJMDevice final : public Device {
    /// @brief the LJM library used to communicate with the device.
    std::shared_ptr<ljm::API> ljm;
    /// @brief the underlying device handle.
    const int dev_handle;

public:
    LJMDevice(const std::shared_ptr<ljm::API> &ljm, const int dev_handle):
        ljm(ljm), dev_handle(dev_handle) {}

    ~LJMDevice() override {
        if (const auto err = ljm->close(dev_handle))
            LOG(WARNING) << "[labjack] failed to close device: " << err;
    }

    xerrors::Error e_stream_read(
        double *data,
        int *dev_scan_backlog,
        int *ljm_scan_backlog
    ) const override {
        return parse_error(
            this->ljm,
            this->ljm->e_stream_read(
                this->dev_handle,
                data,
                dev_scan_backlog,
                ljm_scan_backlog
            )
        );
    }

    [[nodiscard]] xerrors::Error e_stream_stop() const override {
        return parse_error(this->ljm, this->ljm->e_stream_stop(dev_handle));
    }

    [[nodiscard]] xerrors::Error
    e_write_addr(const int addr, const int type, const double value) const override {
        return parse_error(
            this->ljm,
            this->ljm->e_write_addr(this->dev_handle, addr, type, value)
        );
    }

    [[nodiscard]] xerrors::Error e_write_addrs(
        const size_t num_frames,
        const int *addrs,
        const int *types,
        const double *values,
        int *error_addrs
    ) const override {
        return parse_error(
            this->ljm,
            this->ljm->e_write_addrs(
                this->dev_handle,
                static_cast<int>(num_frames),
                addrs,
                types,
                values,
                error_addrs
            )
        );
    }

    [[nodiscard]] xerrors::Error
    start_interval(const int interval_handle, const int microseconds) const override {
        return parse_error(
            this->ljm,
            this->ljm->start_interval(interval_handle, microseconds)
        );
    }

    [[nodiscard]] xerrors::Error
    clean_interval(const int interval_handle) const override {
        return parse_error(this->ljm, this->ljm->clean_interval(interval_handle));
    }

    [[nodiscard]] xerrors::Error
    e_write_name(const char *name, const double vlaue) const override {
        return parse_error(ljm, ljm->e_write_name(dev_handle, name, vlaue));
    }

    [[nodiscard]] xerrors::Error e_write_names(
        const size_t num_frames,
        const char **names,
        const double *values,
        int *err_addr
    ) const override {
        return parse_error(
            this->ljm,
            this->ljm->e_write_names(
                this->dev_handle,
                static_cast<int>(num_frames),
                names,
                values,
                err_addr
            )
        );
    }

    [[nodiscard]] xerrors::Error names_to_addrs(
        const size_t num_frames,
        const char **names,
        int *addrs,
        int *types
    ) const override {
        return parse_error(
            this->ljm,
            this->ljm->names_to_addrs(static_cast<int>(num_frames), names, addrs, types)
        );
    }

    [[nodiscard]] xerrors::Error wait_for_next_interval(
        const int interval_handle,
        int *skipped_intervals
    ) const override {
        return parse_error(
            this->ljm,
            this->ljm->wait_for_next_interval(interval_handle, skipped_intervals)
        );
    }

    [[nodiscard]] xerrors::Error e_read_names(
        const size_t num_frames,
        const char **a_names,
        double *a_values,
        int *error_addr
    ) const override {
        return parse_error(
            this->ljm,
            this->ljm->e_read_names(
                this->dev_handle,
                static_cast<int>(num_frames),
                a_names,
                a_values,
                error_addr
            )
        );
    }

    [[nodiscard]] xerrors::Error
    e_read_name(const char *name, double *value) const override {
        return parse_error(
            this->ljm,
            this->ljm->e_read_name(this->dev_handle, name, value)
        );
    }

    [[nodiscard]] xerrors::Error e_stream_start(
        const size_t scans_per_read,
        const size_t num_addrs,
        const int *scan_list,
        double *scan_rate
    ) const override {
        return parse_error(
            this->ljm,
            this->ljm->e_stream_start(
                this->dev_handle,
                static_cast<int>(scans_per_read),
                static_cast<int>(num_addrs),
                scan_list,
                scan_rate
            )
        );
    }
};

/// @brief manager handles the lifecycle of LabJack devices, allowing callers to
/// acquire and release devices for use at will.
class Manager {
    std::mutex mu;
    std::map<std::string, std::weak_ptr<Device>> handles;
    std::shared_ptr<ljm::API> ljm;

public:
    explicit Manager(const std::shared_ptr<ljm::API> &ljm): ljm(ljm) {}

    xerrors::Error list_all(
        const int dev_type,
        const int conn_type,
        int *num_found,
        int *dev_types,
        int *conn_types,
        int *serial_numbers,
        int *ip_addresses
    ) {
        std::lock_guard lock(this->mu);
        return parse_error(
            ljm,
            ljm->list_all(
                dev_type,
                conn_type,
                num_found,
                dev_types,
                conn_types,
                serial_numbers,
                ip_addresses
            )
        );
    }

    std::pair<std::shared_ptr<Device>, xerrors::Error>
    acquire(const std::string &serial_number) {
        std::lock_guard lock(mu);

        const auto it = this->handles.find(serial_number);
        if (it != handles.end()) {
            const auto existing = it->second.lock();
            if (existing != nullptr) return {existing, xerrors::NIL};
            this->handles.erase(it);
        }

        int dev_handle;
        const int
            err = ljm->open(LJM_dtANY, LJM_ctANY, serial_number.c_str(), &dev_handle);
        if (err != 0) return {nullptr, parse_error(ljm, err)};

        auto dev = std::make_shared<LJMDevice>(ljm, dev_handle);
        this->handles[serial_number] = dev; // Stores weak_ptr automatically
        return {dev, xerrors::NIL};
    }
};
}
