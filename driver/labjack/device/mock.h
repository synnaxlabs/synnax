// Copyright 2025 Synnax Labs, Inc.
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
#include <string>
#include <vector>

#include "driver/labjack/device/device.h"

namespace device {

class Mock final : public Device {
public:
    Mock() = default;
    ~Mock() override = default;

    xerrors::Error e_stream_read(
        double *data,
        int *dev_scan_backlog,
        int *ljm_scan_backlog
    ) const override {
        if (should_fail_) return xerrors::Error("mock failure");
        *dev_scan_backlog = 0;
        *ljm_scan_backlog = 0;
        return xerrors::NIL;
    }

    [[nodiscard]] xerrors::Error e_stream_stop() const override {
        if (should_fail_) return xerrors::Error("mock failure");
        return xerrors::NIL;
    }

    [[nodiscard]] xerrors::Error
    e_write_addr(int addr, int type, double value) const override {
        if (should_fail_) return xerrors::Error("mock failure");
        return xerrors::NIL;
    }

    xerrors::Error e_write_addrs(
        size_t num_frames,
        const int *addrs,
        const int *types,
        const double *values,
        int *error_addrs
    ) const override {
        if (should_fail_) return xerrors::Error("mock failure");
        return xerrors::NIL;
    }

    [[nodiscard]] xerrors::Error
    start_interval(int interval_handle, int microseconds) const override {
        if (should_fail_) return xerrors::Error("mock failure");
        return xerrors::NIL;
    }

    [[nodiscard]] xerrors::Error
    e_write_name(const char *name, double value) const override {
        if (should_fail_) return xerrors::Error("mock failure");
        return xerrors::NIL;
    }

    [[nodiscard]] xerrors::Error e_write_names(
        size_t num_frames,
        const char **names,
        const double *values,
        int *err_addr
    ) const override {
        if (should_fail_) return xerrors::Error("mock failure");
        return xerrors::NIL;
    }

    [[nodiscard]] xerrors::Error names_to_addrs(
        size_t num_frames,
        const char **names,
        int *addrs,
        int *types
    ) const override {
        if (should_fail_) return xerrors::Error("mock failure");
        return xerrors::NIL;
    }

    [[nodiscard]] xerrors::Error
    wait_for_next_interval(int interval_handle, int *skipped_intervals) const override {
        if (should_fail_) return xerrors::Error("mock failure");
        *skipped_intervals = 0;
        return xerrors::NIL;
    }

    [[nodiscard]] xerrors::Error e_read_names(
        size_t num_frames,
        const char **a_names,
        double *a_values,
        int *error_addr
    ) const override {
        if (should_fail_) return xerrors::Error("mock failure");
        return xerrors::NIL;
    }

    [[nodiscard]] xerrors::Error e_stream_start(
        size_t scans_per_read,
        size_t num_addrs,
        const int *scan_list,
        double *scan_rate
    ) const override {
        if (should_fail_) return xerrors::Error("mock failure");
        *scan_rate = requested_scan_rate_;
        return xerrors::NIL;
    }

    // Mock control methods
    void set_should_fail(const bool should_fail) { should_fail_ = should_fail; }
    void set_scan_rate(const double rate) { requested_scan_rate_ = rate; }

private:
    bool should_fail_{false};
    double requested_scan_rate_{1000.0};
};
}
