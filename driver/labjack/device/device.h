// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <atomic>
#include <chrono>
#include <condition_variable>
#include <functional>
#include <map>
#include <memory>
#include <mutex>
#include <string>
#include <utility>

#include "absl/log/log.h"

#include "driver/labjack/ljm/LabJackM.h"
#include "driver/labjack/ljm/api.h"
#include "driver/labjack/ljm/errors.h"

namespace driver::labjack::device {
/// @brief how often a stream read that waits for scans reads again. LJM calls back on
/// a 4 ms tick, and never when the device stops sending.
constexpr auto STREAM_POLL_INTERVAL = std::chrono::milliseconds(1);

class Device {
public:
    virtual ~Device() = default;

    /// @brief reads the next scans_per_read scans of the stream, and waits for them
    /// when they have not arrived.
    virtual x::errors::Error
    e_stream_read(double *data, int *dev_scan_backlog, int *ljm_scan_backlog) const = 0;

    /// @brief stops the stream. A failed stop still allows the next start, so a lost
    /// connection never blocks it.
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

    [[nodiscard]] virtual x::errors::Error e_read_names(
        size_t num_frames,
        const char **a_names,
        double *a_values,
        int *error_addr
    ) const = 0;

    [[nodiscard]] virtual x::errors::Error
    e_read_name(const char *name, double *value) const = 0;

    /// @brief converts a thermocouple voltage to a temperature in Kelvin.
    /// @param type the LJM thermocouple type, such as LJM_ttK.
    /// @param cjc_k the temperature of the cold junction in Kelvin.
    /// @returns ljm::VOLTAGE_OUT_OF_RANGE or ljm::TEMPERATURE_OUT_OF_RANGE when the
    /// voltage is outside the range of the thermocouple type.
    [[nodiscard]] virtual x::errors::Error
    tc_volts_to_temp(int type, double volts, double cjc_k, double *temp_k) const = 0;

    /// @brief reads the version of the LJM library, such as 1.2304.
    [[nodiscard]] virtual x::errors::Error ljm_version(double *version) const = 0;

    /// @brief starts a stream.
    /// @param configure writes the settings of the stream. It runs only when no other
    /// stream holds the device, so a refused start leaves the device untouched.
    /// @returns ljm::STREAM_IS_ACTIVE when a stream already runs on the device.
    [[nodiscard]] virtual x::errors::Error e_stream_start(
        size_t scans_per_read,
        size_t num_addrs,
        const int *scan_list,
        double *scan_rate,
        const std::function<x::errors::Error()> &configure
    ) const = 0;

    /// @brief returns true once LJM reports that the calibration constants of the
    /// device are invalid. LJM then uses default constants, so readings may be
    /// inaccurate.
    [[nodiscard]] virtual bool calibration_invalid() const = 0;
};

class LJMDevice final : public Device {
    /// @brief the LJM library used to communicate with the device.
    std::shared_ptr<ljm::API> ljm;
    /// @brief the underlying device handle.
    const int dev_handle;
    /// @brief true while a stream runs on the handle. A second LJM start on one handle
    /// succeeds and silently replaces the running stream.
    mutable std::atomic<bool> streaming = false;
    /// @brief true once LJM returns LJME_USING_DEFAULT_CALIBRATION for the handle.
    mutable std::atomic<bool> default_calibration;
    mutable std::mutex scans_mu;
    mutable std::condition_variable scans_cv;
    /// @brief set by the stream callback, and cleared by the read that it wakes.
    mutable bool scans_arrived = false;

    /// @brief the stream callback, which LJM calls from its stream thread when scans
    /// arrive.
    static void on_scans(void *arg) {
        const auto *dev = static_cast<const LJMDevice *>(arg);
        {
            std::lock_guard lock(dev->scans_mu);
            dev->scans_arrived = true;
        }
        dev->scans_cv.notify_one();
    }

    /// @brief converts an LJM code into an error, and records a default calibration.
    x::errors::Error check(const int code) const {
        if (code == LJME_USING_DEFAULT_CALIBRATION) this->default_calibration = true;
        return parse_error(this->ljm, code);
    }

public:
    /// @param open_code the code that LJM returned when it opened the handle.
    LJMDevice(
        const std::shared_ptr<ljm::API> &ljm,
        const int dev_handle,
        const int open_code = LJME_NOERROR
    ):
        ljm(ljm),
        dev_handle(dev_handle),
        default_calibration(open_code == LJME_USING_DEFAULT_CALIBRATION) {}

    ~LJMDevice() override {
        if (const auto err = ljm->close(dev_handle))
            LOG(WARNING) << "[labjack] failed to close device: " << err;
    }

    /// A blocking LJM read can sleep about 100 ms past the arrival of its scans, so the
    /// read waits on the stream callback and STREAM_POLL_INTERVAL instead.
    x::errors::Error e_stream_read(
        double *data,
        int *dev_scan_backlog,
        int *ljm_scan_backlog
    ) const override {
        while (true) {
            const int code = this->ljm->e_stream_read(
                this->dev_handle,
                data,
                dev_scan_backlog,
                ljm_scan_backlog
            );
            if (code != LJME_NO_SCANS_RETURNED) return this->check(code);
            std::unique_lock lock(this->scans_mu);
            this->scans_cv.wait_for(lock, STREAM_POLL_INTERVAL, [this] {
                return this->scans_arrived;
            });
            this->scans_arrived = false;
        }
    }

    [[nodiscard]] x::errors::Error e_stream_stop() const override {
        const auto err = this->check(this->ljm->e_stream_stop(dev_handle));
        this->streaming = false;
        return err;
    }

    [[nodiscard]] x::errors::Error
    e_write_addr(const int addr, const int type, const double value) const override {
        return this->check(
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
        return this->check(this->ljm->e_write_addrs(
            this->dev_handle,
            static_cast<int>(num_frames),
            addrs,
            types,
            values,
            error_addrs
        ));
    }

    [[nodiscard]] x::errors::Error
    e_write_name(const char *name, const double vlaue) const override {
        return this->check(this->ljm->e_write_name(dev_handle, name, vlaue));
    }

    [[nodiscard]] x::errors::Error e_write_names(
        const size_t num_frames,
        const char **names,
        const double *values,
        int *err_addr
    ) const override {
        return this->check(this->ljm->e_write_names(
            this->dev_handle,
            static_cast<int>(num_frames),
            names,
            values,
            err_addr
        ));
    }

    [[nodiscard]] x::errors::Error names_to_addrs(
        const size_t num_frames,
        const char **names,
        int *addrs,
        int *types
    ) const override {
        return this->check(
            this->ljm->names_to_addrs(static_cast<int>(num_frames), names, addrs, types)
        );
    }

    [[nodiscard]] x::errors::Error e_read_names(
        const size_t num_frames,
        const char **a_names,
        double *a_values,
        int *error_addr
    ) const override {
        return this->check(this->ljm->e_read_names(
            this->dev_handle,
            static_cast<int>(num_frames),
            a_names,
            a_values,
            error_addr
        ));
    }

    [[nodiscard]] x::errors::Error
    e_read_name(const char *name, double *value) const override {
        return this->check(this->ljm->e_read_name(this->dev_handle, name, value));
    }

    [[nodiscard]] x::errors::Error tc_volts_to_temp(
        const int type,
        const double volts,
        const double cjc_k,
        double *temp_k
    ) const override {
        return parse_error(
            this->ljm,
            this->ljm->tc_volts_to_temp(type, volts, cjc_k, temp_k)
        );
    }

    [[nodiscard]] x::errors::Error ljm_version(double *version) const override {
        return parse_error(
            this->ljm,
            this->ljm->read_library_config(LJM_LIBRARY_VERSION, version)
        );
    }

    [[nodiscard]] x::errors::Error e_stream_start(
        const size_t scans_per_read,
        const size_t num_addrs,
        const int *scan_list,
        double *scan_rate,
        const std::function<x::errors::Error()> &configure
    ) const override {
        if (this->streaming.exchange(true))
            return x::errors::Error(
                ljm::STREAM_IS_ACTIVE,
                "A LabJack runs one stream at a time, and another task is streaming "
                "from this device. Add these channels to the running task."
            );
        auto err = configure();
        // Makes an LJM stream read return no scans instead of sleeping.
        if (!err)
            err = this->check(this->ljm->write_library_config(
                LJM_STREAM_SCANS_RETURN,
                LJM_STREAM_SCANS_RETURN_ALL_OR_NONE
            ));
        if (!err)
            err = this->check(this->ljm->e_stream_start(
                this->dev_handle,
                static_cast<int>(scans_per_read),
                static_cast<int>(num_addrs),
                scan_list,
                scan_rate
            ));
        if (err) {
            this->streaming = false;
            return err;
        }
        err = this->check(this->ljm->set_stream_callback(
            this->dev_handle,
            &LJMDevice::on_scans,
            const_cast<LJMDevice *>(this)
        ));
        if (err) {
            if (const auto stop_err = this->e_stream_stop())
                LOG(WARNING) << "[labjack] failed to stop stream: " << stop_err;
        }
        return err;
    }

    [[nodiscard]] bool calibration_invalid() const override {
        return this->default_calibration;
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

/// @brief opens the device with the given serial number.
using Opener = std::function<
    std::pair<std::shared_ptr<Device>, x::errors::Error>(const std::string &)>;

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
    /// @brief opens a device when no holder remains.
    Opener open;

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
    /// @brief lists and opens devices through the LJM library.
    explicit LJMManager(const std::shared_ptr<ljm::API> &ljm):
        LJMManager(
            ljm,
            [ljm](const std::string &serial_number)
                -> std::pair<std::shared_ptr<Device>, x::errors::Error> {
                int dev_handle = 0;
                const int code = ljm->open(
                    LJM_dtANY,
                    LJM_ctANY,
                    serial_number.c_str(),
                    &dev_handle
                );
                if (const auto err = parse_error(ljm, code)) return {nullptr, err};
                return {
                    std::make_shared<LJMDevice>(ljm, dev_handle, code),
                    x::errors::NIL
                };
            }
        ) {}

    /// @brief lists devices through the LJM library and opens them through open.
    LJMManager(std::shared_ptr<ljm::API> ljm, Opener open):
        ljm(std::move(ljm)), open(std::move(open)) {}

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
        auto [dev, open_err] = this->open(serial_number);
        if (open_err) return {nullptr, open_err};
        // A stream outlives the process that started it, and no caller holds this new
        // handle, so any stream on the device belongs to another process.
        if (const auto stop_err = dev->e_stream_stop();
            stop_err && !stop_err.matches(ljm::STREAM_NOT_RUNNING))
            return {nullptr, stop_err};
        std::lock_guard lock(this->mu);
        this->handles[serial_number] = dev;
        return {dev, x::errors::NIL};
    }
};
}
