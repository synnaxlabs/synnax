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
#include <set>
#include <string>
#include <variant>
#include <vector>

#include "client/cpp/labjack/json.gen.h"
#include "client/cpp/synnax.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/json/json.h"

#include "device/device.h"
#include "driver/common/read_task.h"
#include "driver/common/sample_clock.h"
#include "driver/labjack/labjack.h"
#include "driver/labjack/ljm/LJM_Utilities.h"
#include "driver/labjack/ljm/LabJackM.h"
#include "driver/labjack/ljm/LabJackMModbusMap.h"
#include "driver/transform/transform.h"

namespace driver::labjack {
///@brief look up table mapping LJM TC Type to TC AIN_EF index
// Thermocouple type:		 B  E  J  K  N  R  S  T  C
const int TC_INDEX_LUT[9] = {28, 20, 21, 22, 27, 23, 25, 24, 30};

const std::map<std::string, long> TC_TYPE_LUT = {
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
inline long resolve_tc_type(x::json::Parser &parser, const std::string &tc_type) {
    const auto v = TC_TYPE_LUT.find(tc_type);
    if (v == TC_TYPE_LUT.end()) {
        parser.field_err("thermocouple_type", "Invalid thermocouple type: " + tc_type);
        return LJM_ttK;
    }
    return v->second;
}

/// @brief resolves the CJC modbus address for the configured CJC source.
inline int resolve_cjc_addr(x::json::Parser &parser, const std::string &cjc_source) {
    if (cjc_source == DEVICE_CJC_SOURCE) return LJM_TEMPERATURE_DEVICE_K_ADDRESS;
    if (cjc_source == AIR_CJC_SOURCE) return LJM_TEMPERATURE_AIR_K_ADDRESS;
    if (cjc_source.find(AIN_PREFIX) != std::string::npos) {
        const int port_num = std::stoi(cjc_source.substr(3));
        return port_num * 2;
    }
    parser.field_err("cjc_source", "Invalid CJC source: " + cjc_source);
    return 0;
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

    explicit InputChan(const ::synnax::labjack::BaseInputChannel &cfg):
        enabled(!cfg.disabled), port(cfg.port), synnax_key(cfg.channel) {}

    /// @brief applies the configuration to the device.
    virtual x::errors::Error
    apply(const std::shared_ptr<device::Device> &dev, const std::string &device_type) {
        return x::errors::NIL;
    }

    /// @brief the scale applied to raw samples, when the channel carries one.
    [[nodiscard]] virtual const ::synnax::labjack::Scale *scale() const {
        return nullptr;
    }
};

const std::string TC_SUFFIX = "_EF_READ_A";

/// @brief class for a thermocouple channel configuration.
struct ThermocoupleChan final : InputChan {
    ///@brief The thermocouple type
    // Supported TC types are:
    //     LJM_ttB (val=6001)
    //     LJM_ttE (val=6002)
    //     LJM_ttJ (val=6003)
    //     LJM_ttK (val=6004)
    //     LJM_ttN (val=6005)
    //     LJM_ttR (val=6006)
    //     LJM_ttS (val=6007)
    //     LJM_ttT (val=6008)
    //     LJM_ttC (val=6009)
    // Note that the values above do not align with the AIN_EF index values
    // or order. We use a lookup table provided by LabJack to convert our
    // thermocouple constant to the correct index when using the AIN_EF
    // Lookup table: TC_INDEX_LUT[ x - 60001] = AIN_EF_INDEX
    const long type;

    ///@brief the AIN port the thermocouple's positive lead is wired to.
    const int pos_chan;

    ///@brief the AIN port of the negative lead on T7 devices. 199 is single ended.
    const int neg_chan;

    ///@brief  Modbus Address to read the CJC sensor
    const int cjc_addr;

    ///@brief slope of CJC Voltage to temperature conversion (Kelvin/Volts).
    // if using device temp (cjc_addr is TEMPERATURE_DEVICE_K), set to 1
    // If using a LM34 on some AIN, set to 55.56
    const double cjc_slope;

    ///@brief Offset for CJC temp (Kelvin)
    // If cjc_addr = TEMPERATURE_DEVICE_K. set to 0
    // If using InAmp or expansion board, might need to adjust it a few degrees
    // If using LM34 connected to an AIN, set to 255.37
    const double cjc_offset;

    ///@brief units for the thermocouple reading
    const LJM_TemperatureUnits units;

    ///@brief the scale applied to raw samples after acquisition.
    const ::synnax::labjack::Scale scale_cfg;

    ThermocoupleChan(
        const ::synnax::labjack::InputChannelTc &cfg,
        x::json::Parser &parser
    ):
        InputChan(cfg),
        type(resolve_tc_type(parser, cfg.thermocouple_type)),
        pos_chan(cfg.pos_chan),
        neg_chan(cfg.neg_chan),
        cjc_addr(resolve_cjc_addr(parser, cfg.cjc_source)),
        cjc_slope(cfg.cjc_slope),
        cjc_offset(cfg.cjc_offset),
        units(resolve_temperature_units(parser, cfg.units)),
        scale_cfg(cfg.scale) {
        this->port = AIN_PREFIX + std::to_string(this->pos_chan) + TC_SUFFIX;
    }

    [[nodiscard]] const ::synnax::labjack::Scale *scale() const override {
        return &this->scale_cfg;
    }

    x::errors::Error apply(
        const std::shared_ptr<device::Device> &ljm,
        const std::string &device_type
    ) override {
        if (const auto err = ljm->e_write_addr(41500 + this->pos_chan, LJM_UINT16, 0))
            return err;
        if (device_type == T7) {
            if (const auto err = ljm->e_write_addr(
                    41000 + this->pos_chan,
                    LJM_UINT16,
                    this->neg_chan
                ))
                return err;
            // writing 5 frames of data to modbus registers: tc type, cjc address,
            // slope, offset and units
            enum { NUM_FRAMES = 5 };
            int aAddresses[NUM_FRAMES];
            int aTypes[NUM_FRAMES];
            double aValues[NUM_FRAMES];
            int err_addr = INITIAL_ERR_ADDRESS;

            // For setting up the AIN#_EF_INDEX (thermocouple type)
            aAddresses[0] = 9000 + 2 * pos_chan;
            aTypes[0] = LJM_UINT32;
            aValues[0] = TC_INDEX_LUT[this->type - 6001];

            // For setting up the AIN#_EF_CONFIG_A (temperature units)
            aAddresses[1] = 9300 + 2 * this->pos_chan;
            aTypes[1] = LJM_UINT32;
            aValues[1] = this->units;

            // For setting up the AIN#_EF_CONFIG_B (CJC address)
            aAddresses[2] = 9600 + 2 * this->pos_chan;
            aTypes[2] = LJM_UINT32;
            aValues[2] = this->cjc_addr;

            // For setting up the AIN#_EF_CONFIG_D (CJC slope)
            aAddresses[3] = 10200 + 2 * this->pos_chan;
            aTypes[3] = LJM_FLOAT32;
            aValues[3] = this->cjc_slope;

            // For setting up the AIN#_EF_CONFIG_E (CJC offset)
            aAddresses[4] = 10500 + 2 * this->pos_chan;
            aTypes[4] = LJM_FLOAT32;
            aValues[4] = this->cjc_offset;

            return ljm
                ->e_write_addrs(NUM_FRAMES, aAddresses, aTypes, aValues, &err_addr);
        }
        return x::errors::NIL;
    }
};

/// @brief configuration for an analog input channel.
struct AIChan final : InputChan {
    /// @brief the voltage range for the channel, starting at 0 and ending at range.
    const double range;
    /// @brief the negative channel for differential readings. 199 is single ended.
    const int neg_chan;
    /// @brief the scale applied to raw samples after acquisition.
    const ::synnax::labjack::Scale scale_cfg;

    explicit AIChan(const ::synnax::labjack::InputChannelAI &cfg):
        InputChan(cfg),
        range(cfg.range),
        neg_chan(cfg.neg_chan),
        scale_cfg(cfg.scale) {}

    [[nodiscard]] const ::synnax::labjack::Scale *scale() const override {
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
        if (device_type == T7 || device_type == T8)
            if (const auto err = dev->e_write_name((this->port + "_RANGE").c_str(), 0))
                return err;
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
    explicit DIChan(const ::synnax::labjack::InputChannelDI &cfg): InputChan(cfg) {}
};

/// @brief parses the input channel from the provided configuration.
/// @returns nullptr if the configuration is invalid, and binds any relevant
/// field errors to the config.
inline std::unique_ptr<InputChan> parse_input_chan(x::json::Parser &cfg) {
    const auto type = cfg.field<std::string>("type");
    if (type == "TC")
        return std::make_unique<ThermocoupleChan>(
            ::synnax::labjack::InputChannelTc::parse(cfg),
            cfg
        );
    if (type == "AI")
        return std::make_unique<AIChan>(::synnax::labjack::InputChannelAI::parse(cfg));
    if (type == "DI")
        return std::make_unique<DIChan>(::synnax::labjack::InputChannelDI::parse(cfg));
    cfg.field_err("type", "unknown channel type: " + type);
    return nullptr;
}

/// @brief registers the channel's configured scale on the scale transform.
inline void add_scale(
    transform::Scale &scales,
    const synnax::channel::Key key,
    const x::telem::DataType &dt,
    const ::synnax::labjack::Scale &scale
) {
    if (const auto *l = std::get_if<::synnax::labjack::ScaleLinear>(&scale))
        scales.add(key, transform::UnaryLinearScale(l->slope, l->offset, dt));
    else if (const auto *m = std::get_if<::synnax::labjack::ScaleMap>(&scale))
        scales.add(
            key,
            transform::UnaryMapScale(
                m->pre_scaled_min,
                m->pre_scaled_max,
                m->scaled_min,
                m->scaled_max,
                dt
            )
        );
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
    /// @brief the model of device being read from.
    /// Dynamically populated by querying the core.
    std::string dev_model;
    /// @brief a set of transforms to apply to the frame after reading. Applies
    /// scaling information to channels.
    driver::transform::Chain transform;
    /// @brief the number of skipped scans to allow before warning the user.
    size_t device_scan_backlog_warn_on_count;
    /// @brief the size of the buffer to use for reading data from the device.
    size_t ljm_scan_backlog_warn_on_count;

    ReadTaskConfig(ReadTaskConfig &&other) noexcept:
        common::BaseReadTaskConfig(std::move(other)),
        device_key(other.device_key),
        indexes(std::move(other.indexes)),
        samples_per_chan(other.samples_per_chan),
        channels(std::move(other.channels)),
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
        device_scan_backlog_warn_on_count(parser.field<size_t>(
            "device_scan_backlog_warn_on_count",
            this->sample_rate.hz() * 2 // Default to 2 seconds of scans.
        )),
        ljm_scan_backlog_warn_on_count(parser.field<size_t>(
            "ljm_scan_backlog_warn_on_count",
            this->sample_rate.hz() // Default to 1 second of scans.
        )) {
        if (this->channels.empty()) {
            parser.field_err("channels", "task must have at least one enabled channel");
            return;
        }
        auto [dev, err] = client->devices.retrieve(this->device_key);
        if (err) {
            parser.field_err("device", "failed to retrieve device: " + err.message());
            return;
        }
        this->dev_model = dev.model;
        std::vector<synnax::channel::Key> keys;
        keys.reserve(this->channels.size());
        for (const auto &ch: this->channels)
            keys.push_back(ch->synnax_key);
        const auto [sy_channels, ch_err] = client->channels.retrieve(keys);
        if (ch_err) {
            parser.field_err(
                "channels",
                "failed to retrieve channels: " + ch_err.message()
            );
            return;
        }
        size_t i = 0;
        for (const auto &ch: sy_channels) {
            if (ch.index != 0) this->indexes.insert(ch.index);
            this->channels[i++]->ch = ch;
        }
        auto scale_transform = std::make_unique<transform::Scale>();
        for (const auto &ch: this->channels)
            if (const auto *scale = ch->scale())
                add_scale(*scale_transform, ch->ch.key, ch->ch.data_type, *scale);
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

    /// @brief returns true if the task has any thermocouples.
    [[nodiscard]] bool has_thermocouples() const {
        for (const auto &ch: this->channels)
            if (dynamic_cast<ThermocoupleChan *>(ch.get())) return true;
        return false;
    }

    [[nodiscard]] x::errors::Error
    apply(const std::shared_ptr<device::Device> &dev) const {
        for (const auto &ch: this->channels)
            if (const auto err = ch->apply(dev, this->dev_model)) return err;
        return x::errors::NIL;
    }
};

/// @brief a source implementation that reads from LabJack devices via a unary
/// request-response cycle on each acquisition. This source is only used when the
/// task has thermocouples, as LJM does not support streaming of thermocouple data.
class UnarySource final : public common::Source {
    /// @brief the configuration for the read task.
    ReadTaskConfig cfg;
    /// @brief the API of the device we're reading from.
    const std::shared_ptr<device::Device> dev;
    /// @brief a handle to the interval that is regulating the sample clock.
    const int interval_handle;

public:
    UnarySource(const std::shared_ptr<device::Device> &dev, ReadTaskConfig cfg):
        cfg(std::move(cfg)), dev(dev), interval_handle(0) {}

    x::errors::Error start() override {
        if (const auto err = this->cfg.apply(this->dev)) return err;
        return this->dev->start_interval(
            this->interval_handle,
            static_cast<int>(this->cfg.sample_rate.period().microseconds())
        );
    }

    [[nodiscard]] std::vector<synnax::channel::Channel> channels() const override {
        return this->cfg.sy_channels();
    }

    x::errors::Error stop() override {
        return this->dev->clean_interval(this->interval_handle);
    }

    common::ReadResult
    read(x::breaker::Breaker &breaker, x::telem::Frame &data) override {
        common::ReadResult res;
        common::initialize_frame(data, this->cfg.channels, this->cfg.indexes, 1);
        int err_addr;
        std::vector<const char *> locations;
        std::vector<double> values;
        for (const auto &channel: this->cfg.channels)
            if (channel->enabled) locations.push_back(channel->port.c_str());
        int skipped_intervals;
        if (res.error = this->dev->wait_for_next_interval(
                this->interval_handle,
                &skipped_intervals
            );
            res.error)
            return res;

        values.resize(locations.size());
        if (res.error = this->dev->e_read_names(
                locations.size(),
                locations.data(),
                values.data(),
                &err_addr
            );
            res.error)
            return res;
        for (size_t i = 0; i < this->cfg.channels.size(); ++i) {
            auto &s = data.series->at(i);
            s.clear();
            s.write_casted(&values[i], 1);
        }
        const auto start = x::telem::TimeStamp::now();
        const auto end = start;
        common::generate_index_data(
            data,
            this->cfg.indexes,
            start,
            end,
            1,
            this->cfg.channels.size()
        );
        res.error = this->cfg.transform.transform(data);
        return res;
    }

    [[nodiscard]] synnax::framer::WriterConfig writer_config() const override {
        return this->cfg.writer();
    }
};

/// @brief a source implementation that reads from LabJack devices via the LJM
/// streaming protocol. This is much higher performance than unary request/response
/// cycles, and is preferred in cases where we don't acquire data from
/// thermocouples.
class StreamSource final : public common::Source {
    /// @brief the configuration for the read task.
    ReadTaskConfig cfg;
    /// @brief the API to the device we're reading from.
    const std::shared_ptr<device::Device> dev;
    /// @brief sample clock used to get timestamp information for the task.
    common::HardwareTimedSampleClock sample_clock;
    /// @brief buffer containing interleaved data directly from device
    std::vector<double> interleaved_buf;
    /// @brief buffer containing channel-grouped data after deinterleaving
    std::vector<double> channel_grouped_buf;

    /// @brief Deinterleaves data from interleaved_buf into channel_grouped_buf
    std::vector<double> &deinterleave() {
        const size_t n_channels = this->cfg.channels.size();
        if (n_channels <= 1) return this->interleaved_buf;
        const size_t n_samples = this->cfg.samples_per_chan;
        for (size_t ch = 0; ch < n_channels; ch++)
            for (size_t sample = 0; sample < n_samples; sample++)
                this->channel_grouped_buf
                    [ch * n_samples +
                     sample] = this->interleaved_buf[ch + sample * n_channels];
        return this->channel_grouped_buf;
    }

public:
    StreamSource(const std::shared_ptr<device::Device> &dev, ReadTaskConfig cfg):
        cfg(std::move(cfg)),
        dev(dev),
        sample_clock(
            common::HardwareTimedSampleClockConfig::create_simple(
                this->cfg.sample_rate,
                this->cfg.stream_rate,
                this->cfg.timing.correct_skew
            )
        ),
        interleaved_buf(this->cfg.samples_per_chan * this->cfg.channels.size()),
        channel_grouped_buf(this->cfg.samples_per_chan * this->cfg.channels.size()) {}

    /// @brief returns the configuration for opening the synnax writer.
    [[nodiscard]] synnax::framer::WriterConfig writer_config() const override {
        return this->cfg.writer();
    }

    x::errors::Error start() override { return this->restart(false); }

    [[nodiscard]] std::vector<synnax::channel::Channel> channels() const override {
        return this->cfg.sy_channels();
    }

    /// @brief restarts the source.
    x::errors::Error restart(const bool force) {
        this->stop();
        if (const auto err = this->cfg.apply(this->dev); err && !force) return err;
        std::vector<int> temp_ports(this->cfg.channels.size());
        std::vector<const char *> physical_channels;
        physical_channels.reserve(this->cfg.channels.size());
        for (const auto &channel: this->cfg.channels)
            physical_channels.push_back(channel->port.c_str());
        if (const auto err = this->dev->names_to_addrs(
                this->cfg.channels.size(),
                physical_channels.data(),
                temp_ports.data(),
                nullptr
            ))
            return err;
        auto scan_rate = static_cast<double>(this->cfg.sample_rate.hz());
        if (const auto err = this->dev->e_stream_start(
                this->cfg.samples_per_chan,
                this->cfg.channels.size(),
                temp_ports.data(),
                &scan_rate
            ))
            return err;
        this->sample_clock.reset();
        return x::errors::NIL;
    }

    x::errors::Error stop() override { return this->dev->e_stream_stop(); }

    common::ReadResult
    read(x::breaker::Breaker &breaker, x::telem::Frame &fr) override {
        common::ReadResult res;
        const auto n_channels = this->cfg.channels.size();
        const auto n_samples = this->cfg.samples_per_chan;
        common::initialize_frame(fr, this->cfg.channels, this->cfg.indexes, n_samples);

        const auto start = this->sample_clock.wait(breaker);
        int device_scan_backlog;
        int ljm_scan_backlog;
        if (res.error = translate_error(this->dev->e_stream_read(
                this->interleaved_buf.data(),
                &device_scan_backlog,
                &ljm_scan_backlog
            ));
            res.error) {
            if (res.error.matches(ljm::TEMPORARILY_UNREACHABLE)) this->restart(true);
            return res;
        }
        if (static_cast<size_t>(device_scan_backlog) >
            this->cfg.device_scan_backlog_warn_on_count)
            res.warning = common::skew_warning(device_scan_backlog);
        if (static_cast<size_t>(ljm_scan_backlog) >
            this->cfg.ljm_scan_backlog_warn_on_count)
            res.warning = common::skew_warning(ljm_scan_backlog);
        const auto end = this->sample_clock.end();
        common::transfer_buf(this->deinterleave(), fr, n_channels, n_samples);
        common::generate_index_data(
            fr,
            this->cfg.indexes,
            start,
            end,
            n_samples,
            n_channels
        );
        res.error = this->cfg.transform.transform(fr);
        return res;
    }
};
}
