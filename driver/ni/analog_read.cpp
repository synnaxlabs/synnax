// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cassert>
#include <chrono>
#include <stdio.h>
#include <utility>

#include "client/cpp/telem/telem.h"
#include "driver/ni/ni.h"
#include "glog/logging.h"
#include "nlohmann/json.hpp"

using json = nlohmann::json;

void ni::AnalogReadSource::parse_channels(config::Parser &parser) {
    std::uint64_t c_count = 0;
    parser.iter("channels",
                [&](config::Parser &channel_builder) {
                    ni::ChannelConfig config;
                    // analog channel names are formatted: <device_name>/ai<port>
                    std::string port = std::to_string(
                        channel_builder.required<std::uint64_t>("port"));
                    std::string name = this->reader_config.device_name;
                    config.name = name + "/ai" + port;

                    config.channel_key = channel_builder.required<uint32_t>("channel");
                    config.channel_type = channel_builder.required<std::string>("type");

                    config.ni_channel = this->parse_channel(
                        channel_builder, config.channel_type, config.name);

                    this->channel_map[config.name] =
                            "channels." + std::to_string(c_count);

                    this->port_to_channel[channel_builder.required<std::uint64_t>(
                        "port")] = config.name;

                    config.enabled = channel_builder.optional<bool>("enabled", true);

                    this->reader_config.channels.push_back(config);

                    c_count++;
                });
}

std::shared_ptr<ni::Analog> ni::AnalogReadSource::parse_channel(
    config::Parser &parser, const std::string &channel_type,
    const std::string &channel_name) {
    if (channel_type == "ai_accel")
        return std::make_shared<Acceleration>(
            parser, this->task_handle, channel_name);
    if (channel_type == "ai_accel_4_wire_dc_voltage")
        return std::make_shared<
            Acceleration4WireDCVoltage>(parser, this->task_handle, channel_name);
    if (channel_type == "ai_bridge")
        return std::make_shared<Bridge>(
            parser, this->task_handle, channel_name);
    if (channel_type == "ai_charge")
        return std::make_shared<Charge>(
            parser, this->task_handle, channel_name);
    if (channel_type == "ai_current")
        return std::make_shared<Current>(
            parser, this->task_handle, channel_name);
    if (channel_type == "ai_force_bridge_polynomial")
        return std::make_shared<
            ForceBridgePolynomial>(parser, this->task_handle, channel_name);
    if (channel_type == "ai_force_bridge_table")
        return std::make_shared<
            ForceBridgeTable>(parser, this->task_handle, channel_name);
    if (channel_type == "ai_force_bridge_two_point_lin")
        return std::make_shared<
            ForceBridgeTwoPointLin>(parser, this->task_handle, channel_name);
    if (channel_type == "ai_force_iepe")
        return std::make_shared<ForceIEPE>(
            parser, this->task_handle, channel_name);
    if (channel_type == "ai_microphone")
        return std::make_shared<Microphone>(
            parser, this->task_handle, channel_name);
    if (channel_type == "ai_pressure_bridge_polynomial")
        return std::make_shared<
            PressureBridgePolynomial>(parser, this->task_handle, channel_name);
    if (channel_type == "ai_pressure_bridge_table")
        return std::make_shared<
            PressureBridgeTable>(parser, this->task_handle, channel_name);
    if (channel_type == "ai_pressure_bridge_two_point_lin")
        return std::make_shared<
            PressureBridgeTwoPointLin>(parser, this->task_handle, channel_name);
    if (channel_type == "ai_resistance")
        return std::make_shared<Resistance>(
            parser, this->task_handle, channel_name);
    if (channel_type == "ai_rtd")
        return std::make_shared<RTD>(
            parser, this->task_handle, channel_name);
    if (channel_type == "ai_strain_gauge")
        return std::make_shared<StrainGage>(
            parser, this->task_handle, channel_name);
    if (channel_type == "ai_temp_built_in")
        return std::make_shared<
            TemperatureBuiltInSensor>(parser, this->task_handle, channel_name);
    if (channel_type == "ai_thermocouple")
        return std::make_shared<Thermocouple>(
            parser, this->task_handle, channel_name, this->port_to_channel);
    if (channel_type == "ai_torque_bridge_polynomial")
        return std::make_shared<
            TorqueBridgePolynomial>(parser, this->task_handle, channel_name);
    if (channel_type == "ai_torque_bridge_table")
        return std::make_shared<
            TorqueBridgeTable>(parser, this->task_handle, channel_name);
    if (channel_type == "ai_torque_bridge_two_point_lin")
        return std::make_shared<
            TorqueBridgeTwoPointLin>(parser, this->task_handle, channel_name);
    if (channel_type == "ai_velocity_iepe")
        return std::make_shared<VelocityIEPE>(
            parser, this->task_handle, channel_name);
    if (channel_type == "ai_voltage")
        return std::make_shared<Voltage>(
            parser, this->task_handle, channel_name);

    // If channel type not recognized update task state
    std::string msg = "unknown channel type " + channel_type;
    this->ctx->setState({
        .task = task.key,
        .variant = "error",
        .details = {
            {"running", false},
            {"message", msg}
        }
    });
    this->log_error(msg);
    return nullptr;
}


int ni::AnalogReadSource::configure_timing() {
    if (this->reader_config.timing_source == "none") {
        if (this->check_ni_error(
            ni::NiDAQmxInterface::CfgSampClkTiming(this->task_handle,
                                                   "",
                                                   this->reader_config.sample_rate.value,
                                                   DAQmx_Val_Rising,
                                                   DAQmx_Val_ContSamps,
                                                   this->reader_config.sample_rate.value))) {
            this->log_error("failed while configuring timing for task "
                            + this->reader_config.task_name);
            return -1;
        }
    } else if (this->check_ni_error(
        ni::NiDAQmxInterface::CfgSampClkTiming(this->task_handle,
                                               this->reader_config.timing_source.c_str(),
                                               this->reader_config.sample_rate.value,
                                               DAQmx_Val_Rising,
                                               DAQmx_Val_ContSamps,
                                               this->reader_config.sample_rate.value))) {
        this->log_error("failed while configuring timing for task "
                        + this->reader_config.task_name);
        return -1;
    }

    // we read data in chunks of num_samples_per_channel such that we can send frames of
    // this->log data of size num_samples_per_channel at the stream rate
    // e.g. if we have 4 channels and want to stream at 100Hz at a 1000hz sample rate
    // make a call to read 10 samples at 100hz
    this->num_samples_per_channel = std::floor(
        this->reader_config.sample_rate.value /
        this->reader_config.stream_rate.value);

    this->buffer_size = this->num_ai_channels * this->num_samples_per_channel;
    this->timer = loop::Timer(this->reader_config.stream_rate);
    return 0;
}

int ni::AnalogReadSource::acquire_data() {
    this->t0 = synnax::TimeStamp::now().value;
    auto err = ni::NiDAQmxInterface::ReadAnalogF64(
        this->task_handle,
        this->num_samples_per_channel,
        -1,
        DAQmx_Val_GroupByChannel,
        this->data.data(),
        this->data.size(),
        &this->samples_read_per_channel,
        NULL
    );
    this->tf = synnax::TimeStamp::now().value;
    return err;
}

std::pair<synnax::Frame, freighter::Error> ni::AnalogReadSource::read(
    breaker::Breaker &breaker) {
    auto f = synnax::Frame(num_channels);

    if (!this->check_ni_error(this->acquire_data()))
        return std::make_pair(std::move(f), freighter::Error(
                                  driver::CRITICAL_HARDWARE_ERROR,
                                  "Failed to read data from from device"));

    // interpolate  timestamps between the initial and final timestamp to ensure
    // non-overlapping timestamps between batched reads
    uint64_t incr = ((this->tf - this->t0) / this->num_samples_per_channel);
    // Construct and populate index channel

    size_t s = this->samples_read_per_channel;
    // Construct and populate synnax frame
    size_t data_index = 0;
    for (int ch = 0; ch < num_channels; ch++) {
        if (this->reader_config.channels[ch].channel_type == "index") {
            auto t = synnax::Series(synnax::TIMESTAMP, this->samples_read_per_channel);
            for (uint64_t i = 0; i < this->samples_read_per_channel; ++i)
                t.write(this->t0 + i * incr);
            f.add(this->reader_config.channels[ch].channel_key, std::move(t));
            continue;
        }
        synnax::Series series = synnax::Series(synnax::FLOAT32, s);
        if (this->reader_config.channels[ch].data_type == synnax::FLOAT32) series = synnax::Series(synnax::FLOAT32, s);
        else if (this->reader_config.channels[ch].data_type == synnax::FLOAT64)
            series = synnax::Series(synnax::FLOAT64, s);
        for (int i = 0; i < this->samples_read_per_channel; i++)
            this->write_to_series(series, this->data[data_index * this->samples_read_per_channel + i],
                                  this->reader_config.channels[ch].data_type);
        f.add(this->reader_config.channels[ch].channel_key, std::move(series));
        data_index++;
    }
    return std::make_pair(std::move(f), freighter::NIL);
}

void ni::AnalogReadSource::write_to_series(synnax::Series &series, double &data,
                                           synnax::DataType data_type) {
    if (data_type == synnax::FLOAT32) series.write(static_cast<float>(data));
    else if (data_type == synnax::FLOAT64) series.write(static_cast<double>(data));
}


int ni::AnalogReadSource::create_channels() {
    auto channels = this->reader_config.channels;
    for (auto &channel: channels) {
        this->num_channels++;
        if (channel.channel_type == "index" || !channel.enabled ||
            !channel.ni_channel)
            continue;
        this->num_ai_channels++;
        this->check_ni_error(channel.ni_channel->create_ni_scale());
        this->check_ni_error(channel.ni_channel->create_ni_channel());
        if (!this->ok()) {
            this->log_error("failed while creating channel " + channel.name);
            return -1;
        }
    }
    return 0;
}

int ni::AnalogReadSource::validate_channels() {
    for (auto &channel: this->reader_config.channels) {
        if (channel.channel_type == "index") {
            if (channel.channel_key == 0) {
                LOG(ERROR) << "[ni.reader] Index channel key is 0";
                return -1;
            }
            continue;
        }
        // if not index, make sure channel type is valid
        auto [channel_info, err] = this->ctx->client->channels.retrieve(
            channel.channel_key);
        if (channel_info.data_type != synnax::FLOAT32 && channel_info.data_type != synnax::FLOAT64) {
            this->log_error("Channel " + channel.name + " is not of type float32 or float64");
            this->ctx->setState({
                .task = task.key,
                .variant = "error",
                .details = {
                    {"running", false},
                    {
                        "message",
                        "Channel " + channel.name + " must be type float32 or float64. Got " + channel_info.data_type.
                        value
                    },
                    {"path", channel.name}
                }
            });
            return -1;
        }
        channel.data_type = channel_info.data_type;
    }
    return 0;
}
