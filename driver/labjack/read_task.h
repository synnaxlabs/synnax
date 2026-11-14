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
#include <charconv>
#include <cmath>
#include <format>
#include <limits>
#include <map>
#include <optional>
#include <set>
#include <string>
#include <utility>
#include <variant>
#include <vector>

#include "client/cpp/labjack/json.gen.h"
#include "client/cpp/synnax.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/json/json.h"
#include "x/cpp/loop/loop.h"

#include "device/device.h"
#include "driver/common/read_task.h"
#include "driver/common/sample_clock.h"
#include "driver/errors/errors.h"
#include "driver/labjack/labjack.h"
#include "driver/labjack/ljm/LJM_Utilities.h"
#include "driver/labjack/ljm/LabJackM.h"
#include "driver/labjack/ljm/LabJackMModbusMap.h"
#include "driver/transform/transform.h"

namespace driver::labjack {
const std::map<std::string, int> TC_TYPE_LUT = {
    {"B", LJM_ttB},
    {"E", LJM_ttE},
    {"J", LJM_ttJ},
    {"K", LJM_ttK},
    {"N", LJM_ttN},
    {"R", LJM_ttR},
    {"S", LJM_ttS},
    {"T", LJM_ttT},
    {"C", LJM_ttC}
};

const std::string DEVICE_CJC_SOURCE = "TEMPERATURE_DEVICE_K";
const std::string AIR_CJC_SOURCE = "TEMPERATURE_AIR_K";
const std::string AIN_PREFIX = "AIN";
const std::string KELVIN_UNITS = "K";
const std::string CELSIUS_UNITS = "C";
const std::string FAHRENHEIT_UNITS = "F";
using LJM_TemperatureUnits = int;
constexpr LJM_TemperatureUnits LJM_KELVIN = 0;
constexpr LJM_TemperatureUnits LJM_CELSIUS = 1;
constexpr LJM_TemperatureUnits LJM_FARENHEIT = 2;

const std::map<std::string, LJM_TemperatureUnits> TEMPERATURE_UNITS = {
    {KELVIN_UNITS, LJM_KELVIN},
    {CELSIUS_UNITS, LJM_CELSIUS},
    {FAHRENHEIT_UNITS, LJM_FARENHEIT}
};

/// @brief converts the configured temperature units to the LJM constant.
inline LJM_TemperatureUnits
resolve_temperature_units(x::json::Parser &parser, const std::string &units) {
    const auto v = TEMPERATURE_UNITS.find(units);
    if (v == TEMPERATURE_UNITS.end()) {
        parser.field_err("units", "Invalid temperature units: " + units);
        return LJM_KELVIN;
    }
    return v->second;
}

/// @brief converts the configured thermocouple type to the LJM constant.
inline int resolve_tc_type(x::json::Parser &parser, const std::string &tc_type) {
    const auto v = TC_TYPE_LUT.find(tc_type);
    if (v == TC_TYPE_LUT.end()) {
        parser.field_err("thermocouple_type", "Invalid thermocouple type: " + tc_type);
        return LJM_ttK;
    }
    return v->second;
}

/// @brief returns the AIN index a port names, e.g. 3 for "AIN3", or nullopt when the
/// port is not an AIN port.
inline std::optional<int> ain_index(const std::string &port) {
    if (!port.starts_with(AIN_PREFIX)) return std::nullopt;
    int index = 0;
    const char *last = port.data() + port.size();
    const auto res = std::from_chars(port.data() + AIN_PREFIX.size(), last, index);
    if (res.ec != std::errc{} || res.ptr != last) return std::nullopt;
    return index;
}

/// @brief resolves the AIN index a port names, e.g. 3 for "AIN3".
inline int resolve_ain_index(x::json::Parser &parser, const std::string &port) {
    if (const auto index = ain_index(port)) return *index;
    parser.field_err("port", "Invalid port: " + port);
    return 0;
}

/// @brief validates the configured CJC source.
inline std::string
resolve_cjc_source(x::json::Parser &parser, const std::string &cjc_source) {
    if (cjc_source == DEVICE_CJC_SOURCE || cjc_source == AIR_CJC_SOURCE ||
        ain_index(cjc_source))
        return cjc_source;
    parser.field_err("cjc_source", "Invalid CJC source: " + cjc_source);
    return "";
}

/// @brief base class for an input channel configuration.
struct InputChan {
    virtual ~InputChan() = default;

    /// @brief whether data acquisition for the channel is enabled.
    const bool enabled;
    /// @brief the port for the channel ex. AIN1
    std::string port;
    /// @brief the synnax key to write channel data to.
    const synnax::channel::Key synnax_key;

    synnax::channel::Channel ch;

    explicit InputChan(const ::synnax::labjack::BaseReadChannel &cfg):
        enabled(!cfg.disabled), port(cfg.port), synnax_key(cfg.channel) {}

    /// @brief applies the configuration to the device.
    virtual x::errors::Error
    apply(const std::shared_ptr<device::Device> &dev, const std::string &device_type) {
        return x::errors::NIL;
    }

    /// @brief the scale applied to raw samples, when the channel carries one.
    [[nodiscard]] virtual const ::synnax::task::Scale *scale() const { return nullptr; }
};

/// @brief the range of a thermocouple input on each model that supports
/// thermocouples, in volts. Each range covers every thermocouple type.
const std::map<std::string, double> THERMOCOUPLE_RANGES = {{T7, 0.1}, {T8, 0.15}};

/// @brief the largest voltage magnitude of any thermocouple type. LJM takes up to
/// 100 µs to reject a larger voltage, so the Driver rejects it first.
constexpr double MAX_THERMOCOUPLE_VOLTS = 0.1;

/// @brief the offset between Kelvin and Celsius.
constexpr double KELVIN_OFFSET = 273.15;

/// @brief configuration for a thermocouple channel. The channel reads the voltage of
/// the thermocouple and the temperature of its cold junction (CJC), and the Driver
/// converts the two to a temperature.
struct ThermocoupleChan final : InputChan {
    /// @brief the LJM thermocouple type, such as LJM_ttK.
    const int type;
    /// @brief the AIN index the thermocouple's positive lead is wired to.
    const int pos_chan;
    /// @brief the AIN port of the negative lead on T7 devices. LJM_GND is single
    /// ended.
    const int neg_chan;
    /// @brief the source of the CJC temperature: TEMPERATURE_DEVICE_K,
    /// TEMPERATURE_AIR_K, or an AIN port wired to a temperature sensor.
    const std::string cjc_source;
    /// @brief the slope that converts the CJC reading to Kelvin.
    const double cjc_slope;
    /// @brief the offset that converts the CJC reading to Kelvin.
    const double cjc_offset;
    /// @brief units for the thermocouple reading
    const LJM_TemperatureUnits units;
    /// @brief the scale applied to the temperature after conversion.
    const ::synnax::task::Scale scale_cfg;
    /// @brief the warning for a voltage outside the range of the thermocouple type.
    const common::Warning out_of_range;
    /// @brief the position of the CJC register in the registers of the task.
    size_t cjc_index = 0;

    ThermocoupleChan(
        const ::synnax::labjack::ThermocoupleReadChannel &cfg,
        x::json::Parser &parser
    ):
        InputChan(cfg),
        type(resolve_tc_type(parser, cfg.thermocouple_type)),
        pos_chan(resolve_ain_index(parser, cfg.port)),
        neg_chan(cfg.neg_chan.value_or(LJM_GND)),
        cjc_source(resolve_cjc_source(parser, cfg.cjc_source)),
        cjc_slope(cfg.cjc_slope),
        cjc_offset(cfg.cjc_offset),
        units(resolve_temperature_units(parser, cfg.units)),
        scale_cfg(cfg.scale),
        out_of_range{
            .message = "Thermocouple out of range",
            .description = cfg.port + " reads a voltage outside the range of its "
                                      "thermocouple type. Check for an open "
                                      "thermocouple or a loose wire. The Driver writes "
                                      "NaN until the voltage is in range.",
        } {
        transform::validate_scale(parser, "scale", this->scale_cfg);
    }

    [[nodiscard]] const ::synnax::task::Scale *scale() const override {
        return &this->scale_cfg;
    }

    /// @brief returns the register that reads the CJC on the model. A T8 has a
    /// temperature sensor at each AIN terminal.
    [[nodiscard]] std::string cjc_register(const std::string &model) const {
        if (model == T8 && this->cjc_source == DEVICE_CJC_SOURCE)
            return "TEMPERATURE" + std::to_string(this->pos_chan);
        return this->cjc_source;
    }

    x::errors::Error apply(
        const std::shared_ptr<device::Device> &dev,
        const std::string &device_type
    ) override {
        const auto range = THERMOCOUPLE_RANGES.find(device_type);
        if (range == THERMOCOUPLE_RANGES.end()) {
            const std::string prefix = "LJM_dt";
            const auto name = device_type.starts_with(prefix)
                                ? device_type.substr(prefix.size())
                                : device_type;
            return x::errors::Error(
                x::errors::VALIDATION,
                this->port + " is a thermocouple channel, but the " + name +
                    " does not support thermocouples."
            );
        }
        if (device_type == T8 && this->cjc_source == AIR_CJC_SOURCE)
            return x::errors::Error(
                x::errors::VALIDATION,
                this->port +
                    " uses the air temperature as its CJC source, which the T8 does "
                    "not have. Select another CJC source."
            );
        // A device keeps a thermocouple feature that other software set until it
        // loses power.
        std::vector<std::pair<std::string, double>> writes = {
            {this->port + "_EF_INDEX", 0},
            {this->port + "_RANGE", range->second},
            {this->port + "_RESOLUTION_INDEX", 0},
        };
        if (device_type == T7)
            writes.emplace_back(this->port + "_NEGATIVE_CH", this->neg_chan);
        for (const auto &[name, value]: writes)
            if (const auto err = dev->e_write_name(name.c_str(), value)) return err;
        return x::errors::NIL;
    }

    /// @brief converts a thermocouple voltage to a temperature in the units of the
    /// channel.
    /// @param cjc the reading of the CJC register.
    /// @returns NaN when the voltage is out of range for the thermocouple type, which
    /// an open thermocouple causes.
    [[nodiscard]] std::pair<double, x::errors::Error>
    temperature(const device::Device &dev, const double volts, const double cjc) const {
        constexpr double NaN = std::numeric_limits<double>::quiet_NaN();
        if (std::abs(volts) > MAX_THERMOCOUPLE_VOLTS) return {NaN, x::errors::NIL};
        double kelvin = 0;
        const auto cjc_k = cjc * this->cjc_slope + this->cjc_offset;
        if (const auto err = dev.tc_volts_to_temp(this->type, volts, cjc_k, &kelvin)) {
            if (err.matches(ljm::VOLTAGE_OUT_OF_RANGE) ||
                err.matches(ljm::TEMPERATURE_OUT_OF_RANGE))
                return {NaN, x::errors::NIL};
            return {0, err};
        }
        if (this->units == LJM_CELSIUS) return {kelvin - KELVIN_OFFSET, x::errors::NIL};
        if (this->units == LJM_FARENHEIT)
            return {(kelvin - KELVIN_OFFSET) * 9 / 5 + 32, x::errors::NIL};
        return {kelvin, x::errors::NIL};
    }
};

/// @brief the analog input ranges of each model with selectable ranges, in volts,
/// smallest first. A range of r reads from -r to +r.
const std::map<std::string, std::vector<double>> AI_RANGES = {
    {T7, {0.01, 0.1, 1, 10}},
    {T8, {0.018, 0.036, 0.075, 0.15, 0.3, 0.6, 1.2, 2.4, 4.8, 9.6, 11}},
};

/// @brief configuration for an analog input channel.
struct AIChan final : InputChan {
    /// @brief the largest voltage magnitude the channel reads. The channel uses the
    /// smallest range of the device that covers it.
    const double range;
    /// @brief the negative channel for differential readings. LJM_GND is single ended.
    const int neg_chan;
    /// @brief the scale applied to raw samples after acquisition.
    const ::synnax::task::Scale scale_cfg;

    AIChan(const ::synnax::labjack::AnalogReadChannel &cfg, x::json::Parser &parser):
        InputChan(cfg),
        range(cfg.range),
        neg_chan(cfg.neg_chan.value_or(LJM_GND)),
        scale_cfg(cfg.scale) {
        if (this->range <= 0)
            parser.field_err("range", "Max voltage must be greater than 0");
        transform::validate_scale(parser, "scale", this->scale_cfg);
    }

    [[nodiscard]] const ::synnax::task::Scale *scale() const override {
        return &this->scale_cfg;
    }

    x::errors::Error apply(
        const std::shared_ptr<device::Device> &dev,
        const std::string &device_type
    ) override {
        if (const auto err = dev->e_write_name(
                (this->port + "_RESOLUTION_INDEX").c_str(),
                0
            ))
            return err;
        if (const auto ranges = AI_RANGES.find(device_type);
            ranges != AI_RANGES.end()) {
            const auto &options = ranges->second;
            const auto range = std::ranges::lower_bound(options, this->range);
            if (range == options.end())
                return x::errors::Error(
                    x::errors::VALIDATION,
                    std::format(
                        "{} has a max voltage of {} V, but the largest range of the "
                        "device is ±{} V.",
                        this->port,
                        this->range,
                        options.back()
                    )
                );
            if (const auto err = dev->e_write_name(
                    (this->port + "_RANGE").c_str(),
                    *range
                ))
                return err;
        }
        if (device_type == T7)
            if (const auto err = dev->e_write_name(
                    (this->port + "_NEGATIVE_CH").c_str(),
                    this->neg_chan
                ))
                return err;
        return x::errors::NIL;
    }
};

/// @brief configuration for a digital input channel.
struct DIChan final : InputChan {
    explicit DIChan(const ::synnax::labjack::DigitalReadChannel &cfg): InputChan(cfg) {}
};

/// @brief parses the input channel from the provided configuration.
/// @returns nullptr if the configuration is invalid, and binds any relevant
/// field errors to the config.
inline std::unique_ptr<InputChan> parse_input_chan(x::json::Parser &cfg) {
    const auto type = cfg.field<std::string>("type");
    if (type == "thermocouple")
        return std::make_unique<ThermocoupleChan>(
            ::synnax::labjack::ThermocoupleReadChannel::parse(cfg),
            cfg
        );
    if (type == "analog")
        return std::make_unique<AIChan>(
            ::synnax::labjack::AnalogReadChannel::parse(cfg),
            cfg
        );
    if (type == "digital")
        return std::make_unique<DIChan>(
            ::synnax::labjack::DigitalReadChannel::parse(cfg)
        );
    cfg.field_err("type", "unknown channel type: " + type);
    return nullptr;
}

/// @brief returns the scan backlog above which the task warns about skew. An absent
/// count falls back to seconds worth of scans at sample_rate.
inline size_t parse_backlog_warn_on_count(
    x::json::Parser &parser,
    const std::string &field,
    const x::telem::Rate sample_rate,
    const float seconds
) {
    if (const auto count = parser.field<std::optional<std::uint32_t>>(field))
        return *count;
    return static_cast<size_t>(sample_rate.hz() * seconds);
}

/// @brief the highest sample rate of a T8 task with thermocouples. Above it, the CJC
/// sensors of a T8 read LJM_DUMMY_VALUE.
const x::telem::Rate T8_MAX_THERMOCOUPLE_RATE = 250 * x::telem::HERTZ;
/// @brief the first T8 firmware that streams its CJC sensors.
constexpr double T8_THERMOCOUPLE_FIRMWARE = 1.0038;
/// @brief the first LJM version that streams the CJC sensors of a T8.
constexpr double T8_THERMOCOUPLE_LJM = 1.2304;

/// @brief returns true if the version is below the minimum. LabJack versions have four
/// decimal places, and the device reports its firmware version as a float32.
inline bool version_below(const double version, const double minimum) {
    return version < minimum - 0.00005;
}

/// @brief configuration for a LabJack read task.
struct ReadTaskConfig : common::BaseReadTaskConfig {
    const std::string device_key;
    /// @brief the indexes of the channels in the task.
    /// Dynamically populated by querying the core.
    std::set<synnax::channel::Key> indexes;
    /// @brief the number of samples per channel to connect on each call to read.
    const std::size_t samples_per_chan;
    /// @brief the configurations for each channel in the task.
    std::vector<std::unique_ptr<InputChan>> channels;
    /// @brief the registers each read requests: the port of each channel, in channel
    /// order, then the CJC registers that no channel reads.
    std::vector<std::string> registers;
    /// @brief the thermocouple channels, by their position in channels.
    std::vector<std::pair<size_t, const ThermocoupleChan *>> thermocouples;
    /// @brief the model of device being read from.
    /// Dynamically populated by querying the core.
    std::string dev_model;
    /// @brief a set of transforms to apply to the frame after reading. Applies
    /// scaling information to channels.
    driver::transform::Chain transform;
    /// @brief the device-side scan backlog to allow before warning the user.
    size_t device_scan_backlog_warn_on_count;
    /// @brief the LJM-side scan backlog to allow before warning the user.
    size_t ljm_scan_backlog_warn_on_count;

    ReadTaskConfig(ReadTaskConfig &&other) noexcept:
        common::BaseReadTaskConfig(std::move(other)),
        device_key(other.device_key),
        indexes(std::move(other.indexes)),
        samples_per_chan(other.samples_per_chan),
        channels(std::move(other.channels)),
        registers(std::move(other.registers)),
        thermocouples(std::move(other.thermocouples)),
        dev_model(std::move(other.dev_model)),
        transform(std::move(other.transform)),
        device_scan_backlog_warn_on_count(other.device_scan_backlog_warn_on_count),
        ljm_scan_backlog_warn_on_count(other.ljm_scan_backlog_warn_on_count) {}

    ReadTaskConfig(const ReadTaskConfig &) = delete;

    const ReadTaskConfig &operator=(const ReadTaskConfig &) = delete;

    explicit ReadTaskConfig(
        const std::shared_ptr<synnax::Synnax> &client,
        x::json::Parser &parser,
        const common::TimingConfig timing_cfg = common::TimingConfig()
    ):
        common::BaseReadTaskConfig(parser, timing_cfg),
        device_key(parser.field<std::string>("device", "")),
        samples_per_chan(sample_rate / stream_rate),
        channels(parser.map<std::unique_ptr<InputChan>>(
            "channels",
            [&](x::json::Parser &ch_cfg)
                -> std::pair<std::unique_ptr<InputChan>, bool> {
                auto ch = parse_input_chan(ch_cfg);
                if (ch == nullptr) return {nullptr, false};
                return {std::move(ch), ch->enabled};
            }
        )),
        device_scan_backlog_warn_on_count(parse_backlog_warn_on_count(
            parser,
            "device_scan_backlog_warn_on_count",
            this->sample_rate,
            2
        )),
        ljm_scan_backlog_warn_on_count(parse_backlog_warn_on_count(
            parser,
            "ljm_scan_backlog_warn_on_count",
            this->sample_rate,
            1
        )) {
        if (this->channels.empty()) {
            parser.field_err("channels", "task must have at least one enabled channel");
            return;
        }
        auto [dev, err] = client->devices.retrieve(this->device_key);
        if (err) {
            parser.field_err("device", "failed to retrieve device: " + err.data);
            return;
        }
        this->dev_model = dev.model;
        std::vector<synnax::channel::Key> keys;
        keys.reserve(this->channels.size());
        for (const auto &ch: this->channels)
            keys.push_back(ch->synnax_key);
        const auto [sy_channels, ch_err] = client->channels.retrieve(keys);
        if (ch_err) {
            parser.field_err("channels", "failed to retrieve channels: " + ch_err.data);
            return;
        }
        size_t i = 0;
        for (const auto &ch: sy_channels) {
            if (ch.index != 0) this->indexes.insert(ch.index);
            this->channels[i++]->ch = ch;
        }
        for (const auto &ch: this->channels)
            this->registers.push_back(ch->port);
        for (size_t j = 0; j < this->channels.size(); ++j) {
            auto *tc = dynamic_cast<ThermocoupleChan *>(this->channels[j].get());
            if (tc == nullptr) continue;
            const auto cjc = tc->cjc_register(this->dev_model);
            const auto it = std::ranges::find(this->registers, cjc);
            tc->cjc_index = static_cast<size_t>(it - this->registers.begin());
            if (it == this->registers.end())
                this->registers.push_back(cjc);
            else if (
                tc->cjc_index < this->channels.size() &&
                dynamic_cast<const ThermocoupleChan *>(
                    this->channels[tc->cjc_index].get()
                ) != nullptr
            )
                parser.field_err(
                    "channels",
                    cjc +
                        " is a thermocouple channel, so it cannot be the CJC "
                        "source of " +
                        tc->port
                );
            this->thermocouples.emplace_back(j, tc);
        }
        if (this->dev_model == T8 && !this->thermocouples.empty() &&
            this->sample_rate > T8_MAX_THERMOCOUPLE_RATE)
            parser.field_err(
                "sample_rate",
                "A T8 task with thermocouples samples at 250 Hz or less"
            );
        auto scale_transform = std::make_unique<transform::Scale>();
        for (const auto &ch: this->channels)
            if (const auto *scale = ch->scale())
                scale_transform->add(ch->ch.key, ch->ch.data_type, *scale);
        this->transform.add(std::move(scale_transform));
    }

    [[nodiscard]] std::vector<synnax::channel::Channel> sy_channels() const {
        std::vector<synnax::channel::Channel> chs;
        chs.reserve(this->channels.size());
        for (const auto &ch: this->channels)
            chs.push_back(ch->ch);
        return chs;
    }

    /// @brief returns configuration for opening a writer to write data to Synnax.
    [[nodiscard]] synnax::framer::WriterConfig writer() const {
        std::vector<synnax::channel::Key> keys;
        keys.reserve(this->channels.size() + this->indexes.size());
        for (const auto &ch: this->channels)
            keys.push_back(ch->ch.key);
        for (const auto &idx: this->indexes)
            keys.push_back(idx);
        return synnax::framer::WriterConfig{
            .channels = keys,
            .mode = common::data_saving_writer_mode(this->data_saving_disabled),
        };
    }

    /// @brief parses the configuration from the provided Synnax task.
    /// @param client - used to retrieve remote information about the task.
    /// @param task - the raw synnax task config.
    /// @param timing_cfg - the timing configuration for the task.
    /// @returns the configuration an error. If the error is not NIL, the
    /// configuration is invalid and should not be used.
    static std::pair<ReadTaskConfig, x::errors::Error> parse(
        const std::shared_ptr<synnax::Synnax> &client,
        const synnax::task::Task &task,
        const common::TimingConfig timing_cfg
    ) {
        auto parser = x::json::Parser(task.config);
        return {ReadTaskConfig(client, parser, timing_cfg), parser.error()};
    }

    /// @brief returns true if the task polls the device instead of streaming. A T7
    /// cannot stream its CJC sensors, so a T7 task with thermocouples polls.
    [[nodiscard]] bool polled() const {
        return !this->thermocouples.empty() && this->dev_model != T8;
    }

    /// @brief checks the firmware and LJM versions that the task needs, then applies
    /// the settings of each channel to the device.
    [[nodiscard]] x::errors::Error
    apply(const std::shared_ptr<device::Device> &dev) const {
        if (this->dev_model == T8 && !this->thermocouples.empty())
            if (const auto err = check_t8_thermocouple_versions(*dev)) return err;
        for (const auto &ch: this->channels)
            if (const auto err = ch->apply(dev, this->dev_model)) return err;
        return x::errors::NIL;
    }

    /// @brief converts the thermocouple voltages of one read to temperatures, in place.
    /// @param values one value per register, in the order of registers.
    /// @param warning set when a thermocouple is out of range.
    [[nodiscard]] x::errors::Error convert_thermocouples(
        const device::Device &dev,
        double *values,
        common::Warning &warning
    ) const {
        for (const auto &[i, tc]: this->thermocouples) {
            const auto [temp, err] = tc->temperature(
                dev,
                values[i],
                values[tc->cjc_index]
            );
            if (err) return err;
            if (std::isnan(temp)) warning = tc->out_of_range;
            values[i] = temp;
        }
        return x::errors::NIL;
    }

private:
    /// @brief returns an error if the firmware of the T8 or the LJM library is too old
    /// to stream the CJC sensors of the T8.
    static x::errors::Error check_t8_thermocouple_versions(const device::Device &dev) {
        double firmware = 0;
        if (const auto err = dev.e_read_name("FIRMWARE_VERSION", &firmware)) return err;
        if (version_below(firmware, T8_THERMOCOUPLE_FIRMWARE))
            return x::errors::Error(
                x::errors::VALIDATION,
                std::format(
                    "Thermocouples on a T8 need firmware {:.4f} or later, but the "
                    "device has firmware {:.4f}. Update the firmware with Kipling.",
                    T8_THERMOCOUPLE_FIRMWARE,
                    firmware
                )
            );
        double ljm_version = 0;
        if (const auto err = dev.ljm_version(&ljm_version)) return err;
        if (version_below(ljm_version, T8_THERMOCOUPLE_LJM))
            return x::errors::Error(
                x::errors::VALIDATION,
                std::format(
                    "Thermocouples on a T8 need LJM {:.4f} or later, but the Driver "
                    "loaded LJM {:.4f}. Install the latest LJM.",
                    T8_THERMOCOUPLE_LJM,
                    ljm_version
                )
            );
        return x::errors::NIL;
    }
};

/// @brief the warning a stream reports when the device skipped scans.
const common::Warning SKIPPED_SCANS_WARNING{
    .message = "Skipped samples",
    .description = "The buffer of the device filled, so the device skipped samples. "
                   "Lower the sample rate, or check the connection to the device.",
};

/// @brief the warning a source reports while LJM uses the default calibration of the
/// device.
const common::Warning DEFAULT_CALIBRATION_WARNING{
    .message = "Using default calibration",
    .description = "The calibration constants of the device are not valid, so readings "
                   "may be inaccurate. Contact LabJack support.",
};

/// @brief a source that polls the device with one request per sample, on a software
/// clock. T7 tasks with thermocouples use it, because a T7 cannot stream its CJC
/// sensors.
class UnarySource final : public common::Source {
    /// @brief the configuration for the read task.
    ReadTaskConfig cfg;
    /// @brief acquires the device.
    const std::shared_ptr<device::Manager> devs;
    /// @brief the device, or null while the source holds none.
    std::shared_ptr<device::Device> dev;
    /// @brief regulates the rate of the requests.
    x::loop::Timer timer;
    /// @brief the registers of each request.
    std::vector<const char *> registers;
    /// @brief the values of one request, in the order of registers.
    std::vector<double> values;

    /// @brief acquires the device and applies the channel settings. A device that lost
    /// power also lost its settings, so every connect applies them.
    x::errors::Error connect() {
        auto [d, err] = this->devs->acquire(this->cfg.device_key);
        if (err) return err;
        if (const auto apply_err = this->cfg.apply(d)) return apply_err;
        this->dev = std::move(d);
        this->timer = x::loop::Timer(this->cfg.sample_rate);
        return x::errors::NIL;
    }

public:
    UnarySource(const std::shared_ptr<device::Manager> &devs, ReadTaskConfig cfg):
        cfg(std::move(cfg)), devs(devs), values(this->cfg.registers.size()) {
        this->registers.reserve(this->cfg.registers.size());
        for (const auto &reg: this->cfg.registers)
            this->registers.push_back(reg.c_str());
    }

    /// @brief claims the device and applies the channel settings.
    x::errors::Error start() override { return this->connect(); }

    [[nodiscard]] std::vector<synnax::channel::Channel> channels() const override {
        return this->cfg.sy_channels();
    }

    x::errors::Error stop() override {
        this->dev.reset();
        return x::errors::NIL;
    }

    /// @brief reads samples_per_chan samples. After a temporary error, the source
    /// releases the device, and the next read claims it again.
    common::ReadResult
    read(x::breaker::Breaker &breaker, x::telem::Frame &fr) override {
        common::ReadResult res;
        if (this->dev == nullptr)
            if (res.error = this->connect(); res.error) return res;
        const auto n_channels = this->cfg.channels.size();
        common::initialize_frame(
            fr,
            this->cfg.channels,
            this->cfg.indexes,
            this->cfg.samples_per_chan
        );
        for (auto &s: *fr.series)
            s.clear();
        for (size_t i = 0; i < this->cfg.samples_per_chan && breaker.running(); ++i) {
            if (!this->timer.wait(breaker).second) res.warning = common::SKEW_WARNING;
            const auto start = x::telem::TimeStamp::now();
            int err_addr = 0;
            if (res.error = this->dev->e_read_names(
                    this->registers.size(),
                    this->registers.data(),
                    this->values.data(),
                    &err_addr
                );
                res.error) {
                if (res.error.matches(errors::TEMPORARY_HARDWARE_ERROR))
                    this->dev.reset();
                return res;
            }
            const auto end = x::telem::TimeStamp::now();
            if (res.error = this->cfg.convert_thermocouples(
                    *this->dev,
                    this->values.data(),
                    res.warning
                );
                res.error)
                return res;
            for (size_t j = 0; j < n_channels; ++j)
                fr.series->at(j).write_casted(&this->values[j], 1);
            const auto time = x::telem::TimeStamp(end - (end - start) / 2);
            for (size_t j = n_channels; j < fr.size(); ++j)
                fr.series->at(j).write(time);
        }
        if (res.warning.empty() && this->dev->calibration_invalid())
            res.warning = DEFAULT_CALIBRATION_WARNING;
        res.error = this->cfg.transform.transform(fr);
        return res;
    }

    [[nodiscard]] synnax::framer::WriterConfig writer_config() const override {
        return this->cfg.writer();
    }
};

/// @brief a source that reads from the device through an LJM stream. It is much
/// faster than polling, and every task but a T7 task with thermocouples uses it.
class StreamSource final : public common::Source {
    /// @brief the configuration for the read task.
    ReadTaskConfig cfg;
    /// @brief acquires the device.
    const std::shared_ptr<device::Manager> devs;
    /// @brief the device, or null while the source holds none.
    std::shared_ptr<device::Device> dev;
    /// @brief sample clock used to get timestamp information for the task.
    common::HardwareTimedSampleClock sample_clock;
    /// @brief buffer containing interleaved data directly from device, with one value
    /// per register in each scan.
    std::vector<double> interleaved_buf;
    /// @brief buffer containing channel-grouped data after deinterleaving
    std::vector<double> channel_grouped_buf;
    /// @brief the indices of the scans that the last read kept.
    std::vector<size_t> kept_scans;

    /// @brief moves the scans that LJM filled with LJM_DUMMY_VALUE out of
    /// interleaved_buf, keeping the order of the rest. The device skips scans when its
    /// buffer fills.
    /// @returns the number of scans kept.
    size_t drop_skipped_scans() {
        const size_t n_registers = this->cfg.registers.size();
        this->kept_scans.clear();
        for (size_t scan = 0; scan < this->cfg.samples_per_chan; ++scan) {
            const auto begin = this->interleaved_buf.begin() +
                               static_cast<std::ptrdiff_t>(scan * n_registers);
            const auto end = begin + static_cast<std::ptrdiff_t>(n_registers);
            if (std::find(begin, end, LJM_DUMMY_VALUE) != end) continue;
            std::copy(
                begin,
                end,
                this->interleaved_buf.begin() +
                    static_cast<std::ptrdiff_t>(this->kept_scans.size() * n_registers)
            );
            this->kept_scans.push_back(scan);
        }
        return this->kept_scans.size();
    }

    /// @brief converts the thermocouple voltages of the first n_scans scans of
    /// interleaved_buf to temperatures.
    x::errors::Error
    convert_thermocouples(const size_t n_scans, common::Warning &warning) {
        if (this->cfg.thermocouples.empty()) return x::errors::NIL;
        const size_t n_registers = this->cfg.registers.size();
        for (size_t scan = 0; scan < n_scans; ++scan)
            if (const auto err = this->cfg.convert_thermocouples(
                    *this->dev,
                    this->interleaved_buf.data() + scan * n_registers,
                    warning
                ))
                return err;
        return x::errors::NIL;
    }

    /// @brief deinterleaves the channel values of the first n_scans scans of
    /// interleaved_buf into channel_grouped_buf.
    std::vector<double> &deinterleave(const size_t n_scans) {
        const size_t n_registers = this->cfg.registers.size();
        if (n_registers == 1) return this->interleaved_buf;
        for (size_t ch = 0; ch < this->cfg.channels.size(); ch++)
            for (size_t scan = 0; scan < n_scans; scan++)
                this->channel_grouped_buf
                    [ch * n_scans +
                     scan] = this->interleaved_buf[ch + scan * n_registers];
        return this->channel_grouped_buf;
    }

    /// @brief writes the timestamps of the kept scans of a read from start to end.
    void write_index(
        x::telem::Frame &fr,
        const x::telem::TimeStamp &start,
        const x::telem::TimeStamp &end
    ) const {
        const auto n_scans = this->cfg.samples_per_chan;
        const auto n_channels = this->cfg.channels.size();
        if (this->kept_scans.size() == n_scans) {
            common::generate_index_data(
                fr,
                this->cfg.indexes,
                start,
                end,
                n_scans,
                n_channels
            );
            return;
        }
        const auto step = (end - start) / static_cast<int64_t>(n_scans);
        for (size_t i = n_channels; i < fr.size(); ++i) {
            auto &s = fr.series->at(i);
            s.clear();
            for (const auto scan: this->kept_scans)
                s.write(start + step * static_cast<int64_t>(scan));
        }
    }

    /// @brief claims the device and starts the stream. A device that lost power also
    /// lost its settings, so every connect applies them.
    x::errors::Error connect() {
        auto [d, acquire_err] = this->devs->acquire(this->cfg.device_key);
        if (acquire_err) return acquire_err;
        std::vector<int> addrs(this->cfg.registers.size());
        std::vector<const char *> registers;
        registers.reserve(this->cfg.registers.size());
        for (const auto &reg: this->cfg.registers)
            registers.push_back(reg.c_str());
        if (const auto err = d->names_to_addrs(
                registers.size(),
                registers.data(),
                addrs.data(),
                nullptr
            ))
            return err;
        auto scan_rate = this->cfg.sample_rate.hz();
        if (const auto err = d->e_stream_start(
                this->cfg.samples_per_chan,
                addrs.size(),
                addrs.data(),
                &scan_rate,
                [this, &d] { return this->cfg.apply(d); }
            ))
            return err;
        this->dev = std::move(d);
        this->sample_clock.reset();
        return x::errors::NIL;
    }

public:
    StreamSource(const std::shared_ptr<device::Manager> &devs, ReadTaskConfig cfg):
        cfg(std::move(cfg)),
        devs(devs),
        sample_clock(
            common::HardwareTimedSampleClockConfig::create_simple(
                this->cfg.sample_rate,
                this->cfg.stream_rate,
                this->cfg.timing.correct_skew
            )
        ),
        interleaved_buf(this->cfg.samples_per_chan * this->cfg.registers.size()),
        channel_grouped_buf(this->cfg.samples_per_chan * this->cfg.channels.size()) {
        this->kept_scans.reserve(this->cfg.samples_per_chan);
    }

    /// @brief returns the configuration for opening the synnax writer.
    [[nodiscard]] synnax::framer::WriterConfig writer_config() const override {
        return this->cfg.writer();
    }

    /// @brief claims the device and starts the stream.
    x::errors::Error start() override { return this->restart(); }

    [[nodiscard]] std::vector<synnax::channel::Channel> channels() const override {
        return this->cfg.sy_channels();
    }

    /// @brief stops the stream, releases the device, and starts the stream again. The
    /// claim is released before the acquire, so the device gets a fresh LJM handle
    /// when no other task holds it.
    x::errors::Error restart() {
        this->stop();
        return this->connect();
    }

    x::errors::Error stop() override {
        if (this->dev == nullptr) return x::errors::NIL;
        const auto err = this->dev->e_stream_stop();
        this->dev.reset();
        return err;
    }

    /// @brief reads samples_per_chan scans. After a temporary error, the source stops
    /// the stream, and the next read starts it again.
    common::ReadResult
    read(x::breaker::Breaker &breaker, x::telem::Frame &fr) override {
        common::ReadResult res;
        if (this->dev == nullptr)
            if (res.error = this->restart(); res.error) return res;
        const auto n_channels = this->cfg.channels.size();
        common::initialize_frame(
            fr,
            this->cfg.channels,
            this->cfg.indexes,
            this->cfg.samples_per_chan
        );
        const auto start = this->sample_clock.wait(breaker);
        int device_scan_backlog = 0;
        int ljm_scan_backlog = 0;
        if (res.error = this->dev->e_stream_read(
                this->interleaved_buf.data(),
                &device_scan_backlog,
                &ljm_scan_backlog,
                breaker
            );
            res.error) {
            if (res.error.matches(errors::TEMPORARY_HARDWARE_ERROR)) this->stop();
            return res;
        }
        if (this->dev->calibration_invalid()) res.warning = DEFAULT_CALIBRATION_WARNING;
        if (std::cmp_greater(
                device_scan_backlog,
                this->cfg.device_scan_backlog_warn_on_count
            ) ||
            std::cmp_greater(
                ljm_scan_backlog,
                this->cfg.ljm_scan_backlog_warn_on_count
            ))
            res.warning = common::SKEW_WARNING;
        const auto end = this->sample_clock.end();
        const auto n_scans = this->drop_skipped_scans();
        if (n_scans < this->cfg.samples_per_chan) res.warning = SKIPPED_SCANS_WARNING;
        if (n_scans == 0) {
            fr.clear();
            return res;
        }
        if (res.error = this->convert_thermocouples(n_scans, res.warning); res.error)
            return res;
        common::transfer_buf(this->deinterleave(n_scans), fr, n_channels, n_scans);
        this->write_index(fr, start, end);
        res.error = this->cfg.transform.transform(fr);
        return res;
    }
};
}
