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

#include "absl/log/log.h"

#include "driver/labjack/ljm/LabJackM.h"
#include "driver/labjack/ljm/api.h"
#include "driver/labjack/ljm/errors.h"

namespace driver::labjack::device {
class Device {
public:
    virtual ~Device() = default;

    virtual x::errors::Error
    e_stream_read(double *data, int *dev_scan_backlog, int *ljm_scan_backlog) const = 0;

    [[nodiscard]] virtual x::errors::Error e_stream_stop() const = 0;

    [[nodiscard]] virtual x::errors::Error
    e_write_addr(int addr, int type, double value) const = 0;

    virtual x::errors::Error e_write_addrs(
        size_t num_frames,
        const int *addrs,
        const int *types,
        const double *values,
        int *error_addrs
    ) const = 0;

    [[nodiscard]] virtual x::errors::Error
    start_interval(int interval_handle, int microseconds) const = 0;

    [[nodiscard]] virtual x::errors::Error
    clean_interval(int interval_handle) const = 0;

    [[nodiscard]] virtual x::errors::Error
    e_write_name(const char *Name, double value) const = 0;

    [[nodiscard]] virtual x::errors::Error e_write_names(
        size_t num_frames,
        const char **names,
        const double *values,
        int *err_addr
    ) const = 0;

    [[nodiscard]] virtual x::errors::Error names_to_addrs(
        size_t num_frames,
        const char **names,
        int *addrs,
        int *types
    ) const = 0;

    [[nodiscard]] virtual x::errors::Error
    wait_for_next_interval(int interval_handle, int *skipped_intervals) const = 0;

    [[nodiscard]] virtual x::errors::Error e_read_names(
        size_t num_frames,
        const char **a_names,
        double *a_values,
        int *error_addr
    ) const = 0;

    [[nodiscard]] virtual x::errors::Error
    e_read_name(const char *name, double *value) const = 0;

    [[nodiscard]] virtual x::errors::Error e_stream_start(
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

    x::errors::Error e_stream_read(
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

    [[nodiscard]] x::errors::Error e_stream_stop() const override {
        return parse_error(this->ljm, this->ljm->e_stream_stop(dev_handle));
    }

    [[nodiscard]] x::errors::Error
    e_write_addr(const int addr, const int type, const double value) const override {
        return parse_error(
            this->ljm,
            this->ljm->e_write_addr(this->dev_handle, addr, type, value)
        );
    }

    [[nodiscard]] x::errors::Error e_write_addrs(
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

    [[nodiscard]] x::errors::Error
    start_interval(const int interval_handle, const int microseconds) const override {
        return parse_error(
            this->ljm,
            this->ljm->start_interval(interval_handle, microseconds)
        );
    }

    [[nodiscard]] x::errors::Error
    clean_interval(const int interval_handle) const override {
        return parse_error(this->ljm, this->ljm->clean_interval(interval_handle));
    }

    [[nodiscard]] x::errors::Error
    e_write_name(const char *name, const double vlaue) const override {
        return parse_error(ljm, ljm->e_write_name(dev_handle, name, vlaue));
    }

    [[nodiscard]] x::errors::Error e_write_names(
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

    [[nodiscard]] x::errors::Error names_to_addrs(
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

    [[nodiscard]] x::errors::Error wait_for_next_interval(
        const int interval_handle,
        int *skipped_intervals
    ) const override {
        return parse_error(
            this->ljm,
            this->ljm->wait_for_next_interval(interval_handle, skipped_intervals)
        );
    }

    [[nodiscard]] x::errors::Error e_read_names(
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

    [[nodiscard]] x::errors::Error
    e_read_name(const char *name, double *value) const override {
        return parse_error(
            this->ljm,
            this->ljm->e_read_name(this->dev_handle, name, value)
        );
    }

    [[nodiscard]] x::errors::Error e_stream_start(
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
public:
    virtual ~Manager() = default;

    /// @brief lists the devices visible to the LJM library.
    virtual x::errors::Error list_all(
        int dev_type,
        int conn_type,
        int *num_found,
        int *dev_types,
        int *conn_types,
        int *serial_numbers,
        int *ip_addresses
    ) = 0;

    /// @brief acquires the device with the given serial number, opening it if no
    /// other caller currently holds it.
    virtual std::pair<std::shared_ptr<Device>, x::errors::Error>
    acquire(const std::string &serial_number) = 0;
};

/// @brief a Manager that opens devices through the LJM library, sharing one handle
/// among concurrent holders of the same device.
class LJMManager final : public Manager {
    /// @brief guards the handle cache.
    std::mutex mu;
    /// @brief serializes opens. LJM returns the same handle when an open device is
    /// opened again, so two racing opens of one device would let the loser's close
    /// kill the winner's connection.
    std::mutex open_mu;
    std::map<std::string, std::weak_ptr<Device>> handles;
    std::shared_ptr<ljm::API> ljm;

    /// @brief returns the cached device for the serial number, if a holder still
    /// exists.
    std::shared_ptr<Device> cached(const std::string &serial_number) {
        std::lock_guard lock(this->mu);
        const auto it = this->handles.find(serial_number);
        if (it == this->handles.end()) return nullptr;
        if (auto existing = it->second.lock()) return existing;
        this->handles.erase(it);
        return nullptr;
    }

public:
    explicit LJMManager(const std::shared_ptr<ljm::API> &ljm): ljm(ljm) {}

    x::errors::Error list_all(
        const int dev_type,
        const int conn_type,
        int *num_found,
        int *dev_types,
        int *conn_types,
        int *serial_numbers,
        int *ip_addresses
    ) override {
        // LJM is thread-safe and the handle cache is untouched here, so no lock:
        // a slow open must not block scanning.
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

    std::pair<std::shared_ptr<Device>, x::errors::Error>
    acquire(const std::string &serial_number) override {
        if (auto existing = this->cached(serial_number))
            return {existing, x::errors::NIL};
        // Opening can block for seconds on an unreachable device; only other
        // opens wait on it, never cache hits or the scanner.
        std::lock_guard open_lock(this->open_mu);
        // A racing caller may have opened the device while we waited.
        if (auto existing = this->cached(serial_number))
            return {existing, x::errors::NIL};
        int dev_handle;
        const int
            err = ljm->open(LJM_dtANY, LJM_ctANY, serial_number.c_str(), &dev_handle);
        if (err != 0) return {nullptr, parse_error(ljm, err)};
        auto dev = std::make_shared<LJMDevice>(ljm, dev_handle);
        std::lock_guard lock(this->mu);
        this->handles[serial_number] = dev;
        return {dev, x::errors::NIL};
    }
};
}
