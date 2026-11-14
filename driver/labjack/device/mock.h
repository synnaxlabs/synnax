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
#include <deque>
#include <functional>
#include <map>
#include <memory>
#include <mutex>
#include <string>
#include <utility>
#include <vector>

#include "driver/labjack/device/device.h"

namespace driver::labjack::device {

/// @brief a scripted device for tests. Each error queue serves one error per call, in
/// order, and calls past the end of a queue succeed.
class Mock final : public Device {
    /// @brief returns the next error of the queue, or NIL when it is empty.
    static x::errors::Error next(std::deque<x::errors::Error> &errors) {
        if (errors.empty()) return x::errors::NIL;
        auto err = errors.front();
        errors.pop_front();
        return err;
    }

public:
    /// @brief the errors that e_stream_read returns.
    mutable std::deque<x::errors::Error> stream_read_errors;
    /// @brief the errors that e_read_names returns.
    mutable std::deque<x::errors::Error> read_errors;
    /// @brief the errors that e_write_names returns.
    mutable std::deque<x::errors::Error> write_errors;
    /// @brief the interleaved scans that each e_stream_read copies into its buffer.
    std::vector<double> stream_data;
    /// @brief the value that e_read_name and e_read_names return for each name. Other
    /// names read 0.
    std::map<std::string, double> read_values;
    /// @brief the errors that tc_volts_to_temp returns.
    mutable std::deque<x::errors::Error> tc_errors;
    /// @brief the slope of the thermocouple conversion of the mock.
    double kelvin_per_volt = 25000;
    /// @brief the version that ljm_version returns.
    double library_version = 1.2305;
    /// @brief the values of each successful e_write_names call, by name.
    mutable std::vector<std::map<std::string, double>> name_write_calls;
    /// @brief the value that calibration_invalid returns.
    bool invalid_calibration = false;

    x::errors::Error e_stream_read(
        double *data,
        int *dev_scan_backlog,
        int *ljm_scan_backlog,
        const x::breaker::Breaker &
    ) const override {
        if (this->should_fail) return x::errors::Error("mock failure");
        if (auto err = next(this->stream_read_errors)) return err;
        std::ranges::copy(this->stream_data, data);
        *dev_scan_backlog = 0;
        *ljm_scan_backlog = 0;
        return x::errors::NIL;
    }

    [[nodiscard]] x::errors::Error e_stream_stop() const override {
        const bool stopped = std::exchange(this->streaming, false);
        if (this->should_fail) return x::errors::Error("mock failure");
        if (!stopped) return ljm::STREAM_NOT_RUNNING;
        return x::errors::NIL;
    }

    [[nodiscard]] x::errors::Error
    e_write_addr(int addr, int type, double value) const override {
        if (this->should_fail) return x::errors::Error("mock failure");
        this->addr_writes[addr] = value;
        return x::errors::NIL;
    }

    x::errors::Error e_write_addrs(
        size_t num_frames,
        const int *addrs,
        const int *types,
        const double *values,
        int *error_addrs
    ) const override {
        if (this->should_fail) return x::errors::Error("mock failure");
        for (size_t i = 0; i < num_frames; ++i)
            this->addr_writes[addrs[i]] = values[i];
        return x::errors::NIL;
    }

    [[nodiscard]] x::errors::Error
    e_read_name(const char *name, double *value) const override {
        if (this->should_fail) return x::errors::Error("mock failure");
        const auto it = this->read_values.find(name);
        *value = it == this->read_values.end() ? 0 : it->second;
        return x::errors::NIL;
    }

    /// @brief converts linearly, at kelvin_per_volt.
    [[nodiscard]] x::errors::Error tc_volts_to_temp(
        int type,
        const double volts,
        const double cjc_k,
        double *temp_k
    ) const override {
        if (auto err = next(this->tc_errors)) return err;
        *temp_k = cjc_k + volts * this->kelvin_per_volt;
        return x::errors::NIL;
    }

    [[nodiscard]] x::errors::Error ljm_version(double *version) const override {
        *version = this->library_version;
        return x::errors::NIL;
    }

    [[nodiscard]] x::errors::Error
    e_write_name(const char *name, double value) const override {
        if (this->should_fail) return x::errors::Error("mock failure");
        ++this->name_writes;
        this->name_values[name] = value;
        return x::errors::NIL;
    }

    [[nodiscard]] x::errors::Error e_write_names(
        size_t num_frames,
        const char **names,
        const double *values,
        int *err_addr
    ) const override {
        if (this->should_fail) return x::errors::Error("mock failure");
        if (auto err = next(this->write_errors)) return err;
        auto &call = this->name_write_calls.emplace_back();
        for (size_t i = 0; i < num_frames; ++i)
            call[names[i]] = values[i];
        return x::errors::NIL;
    }

    [[nodiscard]] x::errors::Error names_to_addrs(
        size_t num_frames,
        const char **names,
        int *addrs,
        int *types
    ) const override {
        if (this->should_fail) return x::errors::Error("mock failure");
        return x::errors::NIL;
    }

    [[nodiscard]] x::errors::Error e_read_names(
        size_t num_frames,
        const char **a_names,
        double *a_values,
        int *error_addr
    ) const override {
        if (this->should_fail) return x::errors::Error("mock failure");
        if (auto err = next(this->read_errors)) return err;
        for (size_t i = 0; i < num_frames; ++i) {
            const auto it = this->read_values.find(a_names[i]);
            a_values[i] = it == this->read_values.end() ? 0 : it->second;
        }
        return x::errors::NIL;
    }

    [[nodiscard]] x::errors::Error e_stream_start(
        size_t scans_per_read,
        size_t num_addrs,
        const int *scan_list,
        double *scan_rate,
        const std::function<x::errors::Error()> &configure
    ) const override {
        if (this->should_fail) return x::errors::Error("mock failure");
        if (this->streaming) return ljm::STREAM_IS_ACTIVE;
        if (const auto err = configure()) return err;
        this->streaming = true;
        ++this->stream_starts;
        *scan_rate = requested_scan_rate;
        return x::errors::NIL;
    }

    [[nodiscard]] bool calibration_invalid() const override {
        return this->invalid_calibration;
    }

    // Mock control methods
    void set_should_fail(const bool should_fail) { this->should_fail = should_fail; }
    void set_scan_rate(const double rate) { requested_scan_rate = rate; }
    [[nodiscard]] size_t write_count() const { return this->name_writes; }
    /// @brief the number of streams started on the device.
    [[nodiscard]] size_t stream_start_count() const { return this->stream_starts; }
    /// @brief whether a stream runs on the device.
    [[nodiscard]] bool is_streaming() const { return this->streaming; }
    /// @brief the last value written to each register by name.
    [[nodiscard]] const std::map<std::string, double> &written_names() const {
        return this->name_values;
    }
    /// @brief the last value written to each register by address.
    [[nodiscard]] const std::map<int, double> &written_addrs() const {
        return this->addr_writes;
    }

private:
    bool should_fail{false};
    double requested_scan_rate{1000.0};
    mutable bool streaming = false;
    mutable size_t stream_starts = 0;
    mutable size_t name_writes = 0;
    mutable std::map<std::string, double> name_values;
    mutable std::map<int, double> addr_writes;
};

/// @brief a Manager that serves scripted devices for tests instead of opening LJM
/// handles.
class MockManager final : public Manager {
    /// @brief the device handed to callers while a holder is still alive.
    std::weak_ptr<Device> cached;

public:
    /// @brief errors to return from acquire() calls in sequence. Calls past the end
    /// of the sequence succeed.
    std::vector<x::errors::Error> acquire_errors;
    /// @brief number of times acquire() was called.
    size_t acquire_call_count = 0;
    /// @brief opens a device when no holder remains. Defaults to serving dev.
    std::function<std::shared_ptr<Device>()> open;
    /// @brief the device the default open() serves.
    std::shared_ptr<Device> dev;

    explicit MockManager(std::shared_ptr<Device> dev = std::make_shared<Mock>()):
        open([this] { return this->dev; }), dev(std::move(dev)) {}
    MockManager(const MockManager &) = delete;
    MockManager &operator=(const MockManager &) = delete;
    MockManager(MockManager &&) = delete;
    MockManager &operator=(MockManager &&) = delete;

    /// @brief a device that list_all() reports.
    struct Listing {
        int device_type = LJM_dtANY;
        int connection_type = LJM_ctUSB;
        int serial_number = 0;
    };
    /// @brief the devices that list_all() reports on their connection type.
    std::vector<Listing> listed;

    x::errors::Error list_all(
        int,
        const int conn_type,
        int *num_found,
        int *dev_types,
        int *conn_types,
        int *serial_numbers,
        int *ip_addresses
    ) override {
        *num_found = 0;
        for (const auto &listing: this->listed) {
            if (conn_type != LJM_ctANY && listing.connection_type != conn_type)
                continue;
            dev_types[*num_found] = listing.device_type;
            conn_types[*num_found] = listing.connection_type;
            serial_numbers[*num_found] = listing.serial_number;
            ip_addresses[*num_found] = 0;
            ++*num_found;
        }
        return x::errors::NIL;
    }

    std::pair<std::shared_ptr<Device>, x::errors::Error>
    acquire(const std::string &) override {
        const auto i = this->acquire_call_count++;
        if (i < this->acquire_errors.size() && this->acquire_errors[i])
            return {nullptr, this->acquire_errors[i]};
        if (auto existing = this->cached.lock()) return {existing, x::errors::NIL};
        auto opened = this->open();
        this->cached = opened;
        return {opened, x::errors::NIL};
    }
};
}
