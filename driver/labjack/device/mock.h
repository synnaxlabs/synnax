// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <functional>
#include <map>
#include <memory>
#include <mutex>
#include <string>
#include <vector>

#include "driver/labjack/device/device.h"

namespace driver::labjack::device {

class Mock final : public Device {
public:
    Mock() = default;
    ~Mock() override = default;

    x::errors::Error e_stream_read(
        double *data,
        int *dev_scan_backlog,
        int *ljm_scan_backlog
    ) const override {
        if (should_fail_) return x::errors::Error("mock failure");
        *dev_scan_backlog = 0;
        *ljm_scan_backlog = 0;
        return x::errors::NIL;
    }

    [[nodiscard]] x::errors::Error e_stream_stop() const override {
        if (should_fail_) return x::errors::Error("mock failure");
        return x::errors::NIL;
    }

    [[nodiscard]] x::errors::Error
    e_write_addr(int addr, int type, double value) const override {
        if (should_fail_) return x::errors::Error("mock failure");
        return x::errors::NIL;
    }

    x::errors::Error e_write_addrs(
        size_t num_frames,
        const int *addrs,
        const int *types,
        const double *values,
        int *error_addrs
    ) const override {
        if (should_fail_) return x::errors::Error("mock failure");
        return x::errors::NIL;
    }

    [[nodiscard]] x::errors::Error
    start_interval(int interval_handle, int microseconds) const override {
        if (should_fail_) return x::errors::Error("mock failure");
        return x::errors::NIL;
    }

    [[nodiscard]] x::errors::Error clean_interval(int interval_handle) const override {
        if (should_fail_) return x::errors::Error("mock failure");
        return x::errors::NIL;
    }

    [[nodiscard]] x::errors::Error
    e_read_name(const char *name, double *value) const override {
        if (should_fail_) return x::errors::Error("mock failure");
        *value = 0;
        return x::errors::NIL;
    }

    [[nodiscard]] x::errors::Error
    e_write_name(const char *name, double value) const override {
        if (should_fail_) return x::errors::Error("mock failure");
        return x::errors::NIL;
    }

    [[nodiscard]] x::errors::Error e_write_names(
        size_t num_frames,
        const char **names,
        const double *values,
        int *err_addr
    ) const override {
        if (should_fail_) return x::errors::Error("mock failure");
        return x::errors::NIL;
    }

    [[nodiscard]] x::errors::Error names_to_addrs(
        size_t num_frames,
        const char **names,
        int *addrs,
        int *types
    ) const override {
        if (should_fail_) return x::errors::Error("mock failure");
        return x::errors::NIL;
    }

    [[nodiscard]] x::errors::Error
    wait_for_next_interval(int interval_handle, int *skipped_intervals) const override {
        if (should_fail_) return x::errors::Error("mock failure");
        *skipped_intervals = 0;
        return x::errors::NIL;
    }

    [[nodiscard]] x::errors::Error e_read_names(
        size_t num_frames,
        const char **a_names,
        double *a_values,
        int *error_addr
    ) const override {
        if (should_fail_) return x::errors::Error("mock failure");
        return x::errors::NIL;
    }

    [[nodiscard]] x::errors::Error e_stream_start(
        size_t scans_per_read,
        size_t num_addrs,
        const int *scan_list,
        double *scan_rate
    ) const override {
        if (should_fail_) return x::errors::Error("mock failure");
        *scan_rate = requested_scan_rate_;
        return x::errors::NIL;
    }

    // Mock control methods
    void set_should_fail(const bool should_fail) { should_fail_ = should_fail; }
    void set_scan_rate(const double rate) { requested_scan_rate_ = rate; }

private:
    bool should_fail_{false};
    double requested_scan_rate_{1000.0};
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

    x::errors::Error list_all(int, int, int *, int *, int *, int *, int *) override {
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
