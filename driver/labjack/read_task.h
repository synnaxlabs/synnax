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
#include <vector>
#include <map>
#include <set>

/// module
#include "client/cpp/synnax.h"
#include "x/cpp/xjson/xjson.h"
#include "x/cpp/breaker/breaker.h"

/// internal
#include "device/device.h"
#include "driver/labjack/labjack.h"
#include "driver/task/common/read_task.h"
#include "driver/labjack/ljm/LabJackM.h"
#include "driver/labjack/ljm/LabJackMModbusMap.h"
#include "driver/labjack/ljm/LJM_Utilities.h"
#include "driver/task/common/sample_clock.h"
#include "driver/transform/transform.h"


namespace labjack {
constexpr int SINGLE_ENDED = 199; // default negative channel for single ended signals

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

inline LJM_TemperatureUnits parse_temperature_units(xjson::Parser &parser, const std::string &path) {
    const auto units = parser.required<std::string>(path);
    const auto v = TEMPERATURE_UNITS.find(units);
    if (v == TEMPERATURE_UNITS.end())
        parser.field_err(path, "Invalid temperature units: " + units);
    return v->second;
}


/// @brief parses the thermocouple type from the configuration and converts it to the
/// appropriate LJM type.
inline long parse_tc_type(xjson::Parser &parser, const std::string &path) {
    const auto tc_type = parser.required<std::string>(path);
    const auto v = TC_TYPE_LUT.find(tc_type);
    if (v == TC_TYPE_LUT.end())
        parser.field_err(path, "Invalid thermocouple type: " + tc_type);
    return v->second;
}

/// @brief parses the CJC address for the device.
inline int parse_cjc_addr(xjson::Parser &parser, const std::string &path) {
    const auto cjc_source = parser.required<std::string>(path);
    if (cjc_source == DEVICE_CJC_SOURCE)
        return LJM_TEMPERATURE_DEVICE_K_ADDRESS;
    if (cjc_source == AIR_CJC_SOURCE)
        return LJM_TEMPERATURE_AIR_K_ADDRESS;
    if (cjc_source.find(AIN_PREFIX) != std::string::npos) {
        const int port_num = std::stoi(cjc_source.substr(3));
        return port_num * 2;
    }
    parser.field_err(path, "Invalid CJC source: " + cjc_source);
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
    const synnax::ChannelKey synnax_key;
    const int neg_chan;
    const int pos_chan;

    synnax::Channel ch;

    explicit InputChan(xjson::Parser &parser)
        : enabled(parser.optional<bool>("enabled", true)),
          port(parser.required<std::string>("port")),
          synnax_key(parser.required<uint32_t>("channel")),
          neg_chan(parser.optional<int>("neg_chan", SINGLE_ENDED)),
          pos_chan(parser.optional<int>("pos_chan", 0)) {
    }

    /// @brief applies the configuration to the device.
    virtual xerrors::Error apply(
        const std::shared_ptr<device::Device> &dev,
        const std::string &device_type
    ) { return xerrors::NIL; }
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
    // or order. We use a lookup table provided by labjack to convert our
    // thermocouple constant to the correct index when using the AIN_EF
    // Lookup table: TC_INDEX_LUT[ x - 60001] = AIN_EF_INDEX
    long type;

    ///@brief  Modbus Address to read the CJC sensor
    int cjc_addr;

    ///@brief slope of CJC Voltage to temperature conversion (Kelvin/Volts).
    // if using device temp (cjc_addr is TEMPERATURE_DEVICE_K), set to 1
    // If using a LM34 on some AIN, set to 55.56
    float cjc_slope;

    ///@brief Offset for CJC temp (Kelvin)
    // If cjc_addr = TEMPERATURE_DEVICE_K. set to 0
    // If using InAmp or expansion board, might need to adjust it a few degrees
    // If using LM34 connected to an AIN, set to 255.37
    float cjc_offset;

    ///@brief units for the thermocouple reading
    LJM_TemperatureUnits units;


    explicit ThermocoupleChan(xjson::Parser &parser):
        InputChan(parser),
        type(parse_tc_type(parser, "thermocouple_type")),
        cjc_addr(parse_cjc_addr(parser, "cjc_source")),
        cjc_slope(parser.required<float>("cjc_slope")),
        cjc_offset(parser.required<float>("cjc_offset")),
        units(parse_temperature_units(parser, "units")) {
        this->port = AIN_PREFIX + std::to_string(this->pos_chan) + TC_SUFFIX;
    }

    xerrors::Error apply(
        const std::shared_ptr<device::Device> &ljm,
        const std::string &device_type
    ) override {
        if (const auto err = ljm->e_write_addr(
            41500 + this->pos_chan,
            LJM_UINT16,
            0
        ))
            return err;
        if (device_type == T7) {
            if (const auto err = ljm->e_write_addr(
                41000 + this->pos_chan,
                LJM_UINT16,
                this->neg_chan
            ))
                return err;
            // writing 5 frames of data to modbus registers: tc type, cjc address, slope, offset and units
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

            return ljm->e_write_addrs(
                NUM_FRAMES,
                aAddresses,
                aTypes,
                aValues,
                &err_addr
            );
        }
        return xerrors::NIL;
    }
};

/// @brief configuration for an analog input channel.
struct AIChan final : InputChan {
    /// @brief the voltage range for the channel, starting at 0 and ending at range.
    const double range;

    explicit AIChan(xjson::Parser &parser):
        InputChan(parser),
        range(parser.optional<double>("range", 10.0)) {
    }

    xerrors::Error apply(
        const std::shared_ptr<device::Device> &dev,
        const std::string &device_type
    ) override {
        if (const auto err = dev->e_write_name(
            (this->port + "_RESOLUTION_INDEX").c_str(),
            0
        ))
            return err;
        if (device_type == T7 || device_type == T8)
            if (const auto err = dev->e_write_name(
                (this->port + "_RANGE").c_str(),
                0
            ))
                return err;
        if (device_type == T7)
            if (const auto err = dev->e_write_name(
                (this->port + "_NEGATIVE_CH").c_str(),
                this->neg_chan
            ))
                return err;
        return xerrors::NIL;
    }
};

/// @brief configuration for a digital input channel.
struct DIChan final : InputChan {
    explicit DIChan(xjson::Parser &parser):
        InputChan(parser) {
    }
};

template<typename T>
using InputChanFactory = std::function<std::unique_ptr<T>(xjson::Parser &cfg)>;

#define INPUT_CHAN_FACTORY(type, class) \
    {type, [](xjson::Parser& cfg) { return std::make_unique<class>(cfg); }}

inline std::map<std::string, InputChanFactory<InputChan> > INPUTS = {
    INPUT_CHAN_FACTORY("TC", ThermocoupleChan),
    INPUT_CHAN_FACTORY("AI", AIChan),
    INPUT_CHAN_FACTORY("DI", DIChan)
};

/// @brief parses the input channel from the provided configuration.
/// @returns nullptr if the configuration is in valid, and binds any relevant
/// field errors to the config.
inline std::unique_ptr<InputChan> parse_input_chan(xjson::Parser &cfg) {
    const auto type = cfg.required<std::string>("type");
    const auto input = INPUTS.find(type);
    if (input != INPUTS.end()) return input->second(cfg);
    cfg.field_err("type", "unknown channel type: " + type);
    return nullptr;
}

/// @brief configuration for a LabJack read task.
struct ReadTaskConfig {
    /// @brief whether data saving is enabled for the task.
    const bool data_saving;
    /// @brief the key of the device to read from.
    const std::string device_key;
    /// @brief how fast to sample data from the device.
    const telem::Rate sample_rate;
    /// @brief how fast to push sampled data to synnax.
    const telem::Rate stream_rate;
    /// @brief the connection method used to communicate with the device.
    std::string conn_method;
    std::set<synnax::ChannelKey> index_keys;
    /// @brief the number of samples per channel to connect on each call to read.
    const std::size_t samples_per_chan;
    /// @brief the configurations for each channel in the task.
    std::vector<std::unique_ptr<InputChan> > channels;
    /// @brief the model of device being read from.
    std::string dev_model;
    transform::Chain transform;

    ReadTaskConfig(ReadTaskConfig &&other) noexcept:
        data_saving(other.data_saving),
        device_key(other.device_key),
        sample_rate(other.sample_rate),
        stream_rate(other.stream_rate),
        conn_method(other.conn_method),
        index_keys(std::move(other.index_keys)),
        samples_per_chan(other.samples_per_chan),
        channels(std::move(other.channels)),
        dev_model(std::move(other.dev_model)),
        transform(std::move(other.transform)) {
    }

    ReadTaskConfig(const ReadTaskConfig &) = delete;

    const ReadTaskConfig &operator=(const ReadTaskConfig &) = delete;

    explicit ReadTaskConfig(
        const std::shared_ptr<synnax::Synnax> &client,
        xjson::Parser &parser
    ): data_saving(parser.optional<bool>("data_saving", false)),
       device_key(parser.required<std::string>("device")),
       sample_rate(telem::Rate(parser.optional<int>("sample_rate", 1))),
       stream_rate(telem::Rate(parser.optional<int>("stream_rate", 1))),
       conn_method(parser.optional<std::string>("conn_method", "")),
       samples_per_chan(sample_rate / stream_rate) {
        parser.iter("channels", [this](xjson::Parser &p) {
            auto ch = parse_input_chan(p);
            if (ch != nullptr && ch->enabled) this->channels.push_back(std::move(ch));
        });
        if (this->channels.empty()) {
            parser.field_err("channels", "task must have at least one enabled channel");
            return;
        }
        auto [dev, err] = client->hardware.retrieve_device(this->device_key);
        if (err) {
            parser.field_err("device", "failed to retrieve device: " + err.message());
            return;
        }
        this->dev_model = dev.model;
        std::vector<synnax::ChannelKey> keys;
        keys.reserve(this->channels.size());
        for (const auto &ch: this->channels) keys.push_back(ch->synnax_key);
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
            if (ch.index != 0) this->index_keys.insert(ch.index);
            this->channels[i++]->ch = ch;
        }

        const auto channel_map = synnax::map_channel_Keys(sy_channels);
        auto scale_transform = std::make_unique<transform::Scale>(parser, channel_map);
        this->transform.add(std::move(scale_transform));
    }

    std::vector<synnax::Channel> sy_channels() const {
        std::vector<synnax::Channel> chs;
        chs.reserve(this->channels.size());
        for (const auto &ch: this->channels) chs.push_back(ch->ch);
        return chs;
    }

    /// @brief returns configuration for opening a writer to write data to Synnax.
    [[nodiscard]] synnax::WriterConfig writer() const {
        std::vector<synnax::ChannelKey> keys;
        keys.reserve(this->channels.size() + this->index_keys.size());
        for (const auto &ch: this->channels) keys.push_back(ch->ch.key);
        for (const auto &idx: this->index_keys) keys.push_back(idx);
        return synnax::WriterConfig{
            .channels = keys,
            .mode = synnax::data_saving_writer_mode(this->data_saving),
            .enable_auto_commit = true
        };
    }

    /// @brief parses the configuration from the provided Synnax task.
    /// @param client - used to retrieve remote information about the task.
    /// @param task - the raw synnax task config.
    /// @returns the configuration an error. If the error is not NIL, the configuration
    /// is invalid and should not be used.
    static std::pair<ReadTaskConfig, xerrors::Error> parse(
        const std::shared_ptr<synnax::Synnax> &client,
        const synnax::Task &task
    ) {
        auto parser = xjson::Parser(task.config);
        return {ReadTaskConfig(client, parser), parser.error()};
    }

    /// @brief returns true if the task has any thermocouples.
    [[nodiscard]] bool has_thermocouples() const {
        for (const auto &ch: this->channels)
            if (dynamic_cast<ThermocoupleChan *>(ch.get())) return true;
        return false;
    }
};

/// @brief a source implementation that reads from labjack devices via a unary
/// request-response cycle on each acquisition. This source is only used when the task
/// has thermocouples, as LJM does not support streaming of thermocouple data.
class UnarySource final : public common::Source {
    /// @brief the configuration for the read task.
    ReadTaskConfig cfg;
    /// @brief the API of the device we're reading from.
    const std::shared_ptr<device::Device> dev;
    /// @brief a handle to the interval that is regulating the sample clock.
    const int interval_handle;

public:
    UnarySource(
        const std::shared_ptr<device::Device> &dev,
        ReadTaskConfig cfg
    ): cfg(std::move(cfg)), dev(dev), interval_handle(0) {
    }

    xerrors::Error start() override {
        return this->dev->start_interval(
            this->interval_handle,
            static_cast<int>(this->cfg.sample_rate.period().microseconds())
        );
    }

    std::vector<synnax::Channel> channels() const override {
        return this->cfg.sy_channels();
    }

    xerrors::Error stop() override {
        return this->dev->clean_interval(this->interval_handle);
    }

    std::pair<Frame, xerrors::Error> read(breaker::Breaker &breaker) override {
        int err_addr;
        std::vector<const char *> locations;
        std::vector<double> values;
        for (const auto &channel: this->cfg.channels)
            if (channel->enabled) locations.push_back(channel->port.c_str());
        int skipped_intervals;
        if (const auto err = this->dev->wait_for_next_interval(
            this->interval_handle, &skipped_intervals
        ))
            return {Frame(), err};
        values.resize(locations.size());
        if (const auto err = this->dev->e_read_names(
            locations.size(),
            locations.data(),
            values.data(),
            &err_addr
        ))
            return {Frame(), err};

        auto f = synnax::Frame(locations.size() + this->cfg.index_keys.size());
        int i = 0;
        for (const auto &chan: this->cfg.channels)
            f.emplace(
                chan->synnax_key,
                telem::Series(chan->ch.data_type.cast(values[i++]))
            );
        const auto start = telem::TimeStamp::now();
        const auto end = start;
        common::generate_index_data(f, this->cfg.index_keys, start, end, 1);
        auto err = this->cfg.transform.transform(f);
        return std::make_pair(std::move(f), err);
    }

    [[nodiscard]] synnax::WriterConfig writer_config() const override {
        return this->cfg.writer();
    }
};

/// @brief a source implementation that reads from labjack deices via the LJM streaming
/// protocol. This is much higher performance than unary request/response cycles, and
/// is preferred in cases where we don't acquire data from thermocouples.
class StreamSource final : public common::Source {
    /// @brief the configuration for the read task.
    ReadTaskConfig cfg;
    /// @brief the API to the device we're reading from.
    const std::shared_ptr<device::Device> dev;
    /// @brief sample clock used to get timestamp information for the task.
    common::HardwareTimedSampleClock sample_clock;
    /// @brief re-usable buffer of values we load data into before converting it to a
    /// frame.
    std::vector<double> buf;

public:
    StreamSource(
        const std::shared_ptr<device::Device> &dev,
        ReadTaskConfig cfg
    ): cfg(std::move(cfg)),
       dev(dev),
       sample_clock(this->cfg.sample_rate),
       buf(this->cfg.samples_per_chan * this->cfg.channels.size()) {
    }

    /// @brief returns the configuration for opening the synnax writer.
    [[nodiscard]] synnax::WriterConfig writer_config() const override {
        return this->cfg.writer();
    }

    xerrors::Error start() override { return this->restart(); }

    [[nodiscard]] std::vector<synnax::Channel> channels() const override {
        return this->cfg.sy_channels();
    }

    /// @brief restarts the source.
    xerrors::Error restart() {
        this->stop();
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
        return xerrors::NIL;
    }

    xerrors::Error stop() override {
        return this->dev->e_stream_stop();
    }

    std::pair<Frame, xerrors::Error> read(breaker::Breaker &breaker) override {
        const auto n = this->cfg.samples_per_chan;
        const auto start = this->sample_clock.wait(breaker);
        int num_skipped_scans;
        int scan_backlog;
        if (auto err = translate_error(this->dev->e_stream_read(
            this->buf.data(),
            &num_skipped_scans,
            &scan_backlog
        ))) {
            // If the device is currently unreachable, try closing and reopening the
            // stream to recover.
            if (err.matches(ljm::TEMPORARILY_UNREACHABLE))
                this->restart();
            return {Frame(), err};
        }
        const auto end = this->sample_clock.end(n);
        auto f = synnax::Frame(this->cfg.channels.size() + this->cfg.index_keys.size());
        int i = 0;
        for (const auto &ch: this->cfg.channels)
            f.emplace(
                ch->synnax_key,
                telem::Series::cast(ch->ch.data_type, buf.data() + i++ * n, n)
            );
        common::generate_index_data(f, this->cfg.index_keys, start, end, n);
        auto err = this->cfg.transform.transform(f);
        return {std::move(f), xerrors::NIL};
    }
};
}
