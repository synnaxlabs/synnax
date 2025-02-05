// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/labjack/reader.h"
#include "driver/labjack/util.h"

///////////////////////////////////////////////////////////////////////////////////
//                                   ReaderSource                                //
///////////////////////////////////////////////////////////////////////////////////
labjack::ReaderSource::ReaderSource(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task task,
    const ReaderConfig &reader_config,
    std::shared_ptr<labjack::DeviceManager> device_manager
) : ctx(ctx),
    task(task),
    reader_config(reader_config),
    device_manager(device_manager) {
    auto breaker_config = breaker::Config{
        .name = task.name,
        .base_interval = 1 * SECOND,
        .max_retries = 20,
        .scale = 1.2,
    };
    this->breaker = breaker::Breaker(breaker_config);
    this->handle = this->device_manager->get_device_handle(this->reader_config.serial_number);

    if (this->reader_config.channels.empty() && this->reader_config.tc_channels.empty())
        this->log_err("No channels enabled/set.");
}

void labjack::ReaderSource::stopped_with_err(const freighter::Error &err) {
    this->log_err(err.message());
}

std::vector<synnax::ChannelKey> labjack::ReaderSource::get_channel_keys() {
    std::vector<synnax::ChannelKey> keys;
    for (auto &channel: this->reader_config.channels) {
        keys.push_back(channel.key);
        // get index key
        auto [channel_info, err] = this->ctx->client->channels.retrieve(channel.key);
        if (err != freighter::NIL) {
            this->log_err("Error retrieving channel for port: " + channel.location);
            continue;
        }
        channel.data_type = channel_info.data_type;
        this->reader_config.index_keys.insert(channel_info.index);
    }

    for (auto &channel: this->reader_config.tc_channels) {
        keys.push_back(channel.key);
        auto [channel_info, err] = this->ctx->client->channels.retrieve(channel.key);
        if (err != freighter::NIL) {
            this->log_err("Error retrieving channel for port: " + channel.location);
            continue;
        }
        channel.data_type = channel_info.data_type;
        this->reader_config.index_keys.insert(channel_info.index);
    }

    for (auto &index_key: this->reader_config.index_keys)
        keys.push_back(index_key);
    return keys;
}


void labjack::ReaderSource::init() {
    if (!this->ok()) return;
    if (this->reader_config.device_type != "") {
        this->init_stream();
        this->init_tcs();
        return;
    }
    auto [dev, err] = this->ctx->client->hardware.retrieveDevice(
        this->reader_config.device_key
    );

    if (err != freighter::NIL)
        return this->log_err("Error retrieving device: " + err.message());

    if (dev.model == "LJM_dtT4")
        this->reader_config.device_type = "T4";
    else if (dev.model == "LJM_dtT7")
        this->reader_config.device_type = "T7";
    else if (dev.model == "LJM_dtT8")
        this->reader_config.device_type = "T8";
    else
        return this->log_err("Unsupported device type: " + dev.model);

    this->init_stream();
    this->init_tcs();
}


void labjack::ReaderSource::init_tcs() {
    if (this->reader_config.tc_channels.empty()) return;
    if (this->reader_config.device_type == "T4")
        return this->log_err("Thermocouple channels not currently supported for T4 devices");

    for (auto &channel: this->reader_config.channels) {
        if (channel.channel_type == "AI") {
            this->check_err(
                LJM_eWriteName(
                    this->handle,
                    (channel.location + "_RESOLUTION_INDEX").c_str(),
                    0
                ), "init_tcs.LJM_eWriteName.RESOLUTION_INDEX"
            );
            if (this->reader_config.device_type == "T7" || this->reader_config.device_type == "T8") {
                this->check_err(
                    LJM_eWriteName(
                        this->handle,
                        (channel.location + "_RANGE").c_str(),
                        0
                    ), "init_tcs.LJM_eWriteName.RANGE"
                );
            }
            if (this->reader_config.device_type == "T7") {
                this->check_err(
                    LJM_eWriteName(
                        this->handle,
                        (channel.location + "_NEGATIVE_CH").c_str(),
                        channel.neg_chan
                    ), "init_tcs.LJM_eWriteName.NEGATIVE_CH"
                );
            }
        }
    }

    // set interval to send read commands to the daq at the specified sample rate
    this->check_err(LJM_StartInterval(
                        this->handle,
                        this->reader_config.sample_rate.period().microseconds()
                    ), "init_tcs.LJM_StartInterval"
    );

    for (auto &channel: this->reader_config.tc_channels)
        this->configure_tc_ain_ef(channel.tc_config);
}

void labjack::ReaderSource::init_stream() {
    if (!this->reader_config.tc_channels.empty()) return;

    double INIT_SCAN_RATE = this->reader_config.sample_rate.value;
    int SCANS_PER_READ = static_cast<int>(INIT_SCAN_RATE / this->reader_config.stream_rate.value);
    double scanRate = INIT_SCAN_RATE;

    this->num_samples_per_chan = SCANS_PER_READ;
    this->buffer_size = this->reader_config.phys_channels.size() * SCANS_PER_READ;

    // iterate through the channels, for the ones that analog device, need to set the resolution index
    for (auto &channel: this->reader_config.channels) {
        if (channel.channel_type == "AI") {
            // Set resolution index to device's default setting (value = 0)
            std::string name = channel.location + "_RESOLUTION_INDEX";
            check_err(WriteName(this->handle, name.c_str(), 0), "init_stream.WriteName.RESOLUTION_INDEX");

            if (this->reader_config.device_type == "T7" || this->reader_config.device_type == "T8") {
                auto name = channel.location + "_RANGE";
                check_err(WriteName(this->handle, name.c_str(), 0), "init_stream.WriteName.RANGE");
            }
            if (this->reader_config.device_type == "T7") {
                auto name = channel.location + "_NEGATIVE_CH";
                check_err(WriteName(this->handle, name.c_str(), channel.neg_chan), "init_stream.WriteName.NEGATIVE_CH");
            }
        }
    }


    this->port_addresses.resize(this->reader_config.phys_channels.size());

    std::vector<const char *> phys_channel_names;
    for (const auto &channel: this->reader_config.phys_channels)
        phys_channel_names.push_back(channel.c_str());

    check_err(
        LJM_NamesToAddresses(
            this->reader_config.phys_channels.size(),
            phys_channel_names.data(),
            this->port_addresses.data(),
            NULL
        ),
        "init_stream.LJM_NamesToAddresses"
    );

    LJM_eStreamStop(handle); // in case it was running

    check_err(
        LJM_eStreamStart(
            handle,
            SCANS_PER_READ,
            this->reader_config.phys_channels.size(),
            this->port_addresses.data(),
            &scanRate
        ),
        "init_stream.LJM_eStreamStart"
    );
};

freighter::Error labjack::ReaderSource::start(const std::string &cmd_key) {
    if (!this->ok())
        return freighter::Error("Device disconnected or is in error. Please reconfigure task and try again");

    if (this->breaker.running()) {
        LOG(INFO) << "[labjack.reader] breaker already running";
        return freighter::NIL;
    }
    this->breaker.start();
    this->init();
    if (!this->ok()) {
        LOG(ERROR) << "Device not initialized properly. Requires reconfigure.";
        return freighter::Error("Device not initialized properly. Requires reconfigure.");
    }
    this->sample_thread = std::thread(&labjack::ReaderSource::acquire_data, this);
    ctx->set_state({
        .task = task.key,
        .key = cmd_key,
        .variant = "success",
        .details = {
            {"running", true},
            {"message", "Task started successfully"}
        }
    });
    return freighter::NIL;
};

freighter::Error labjack::ReaderSource::stop(const std::string &cmd_key) {
    if (!this->breaker.running()) return freighter::NIL;
    this->breaker.stop();

    if (this->sample_thread.joinable()) this->sample_thread.join();
    check_err(LJM_eStreamStop(handle), "stop.LJM_eStreamStop");
    ctx->set_state({
        .task = task.key,
        .key = cmd_key,
        .variant = "success",
        .details = {
            {"running", false},
            {"message", "Task stopped successfully"}
        }
    });
    return freighter::NIL;
}

std::pair<Frame, freighter::Error> labjack::ReaderSource::read_cmd_response(breaker::Breaker &breaker) {
    /// Thermocouples
    // TODO: change when we provide diff index channels for tcs
    int err_addr;
    std::vector<const char *> locations;
    std::vector<double> values;
    for (const auto &channel: this->reader_config.tc_channels)
        if (channel.enabled) locations.push_back(channel.location.c_str());
    for (const auto &channel: this->reader_config.channels)
        if (channel.enabled) locations.push_back(channel.location.c_str());

    int SkippedIntervals;
    check_err(LJM_WaitForNextInterval(this->handle, &SkippedIntervals), "read_cmd_response.LJM_WaitForNextInterval");
    values.resize(locations.size());
    check_err(LJM_eReadNames(
                  this->handle,
                  locations.size(),
                  locations.data(),
                  values.data(),
                  &err_addr),
              "read.LJM_eReadNames"
    );

    auto f = synnax::Frame(locations.size() + this->reader_config.index_keys.size());
    int index = 0;
    for (const auto &loc: locations) {
        for (const auto &channel: this->reader_config.tc_channels) {
            if (channel.location == loc) {
                auto key = this->reader_config.channel_map[channel.location];
                auto s = synnax::Series(channel.data_type, 1);
                write_to_series(s, values[index], channel.data_type);
                f.add(key, std::move(s));
            }
        }
        for (const auto &channel: this->reader_config.channels) {
            if (channel.location == loc) {
                auto key = this->reader_config.channel_map[channel.location];
                auto s = synnax::Series(channel.data_type, 1);
                write_to_series(s, values[index], channel.data_type);
                f.add(key, std::move(s));
            }
        }
        index++;
    }

    for (auto index_key: this->reader_config.index_keys) {
        auto t = synnax::Series(synnax::TIMESTAMP, 1);
        t.write(synnax::TimeStamp::now().value);
        f.add(index_key, std::move(t));
    }

    return std::make_pair(std::move(f), freighter::NIL);
}

std::pair<Frame, freighter::Error> labjack::ReaderSource::read_stream(breaker::Breaker &breaker) {
    int SCANS_PER_READ = this->num_samples_per_chan;
    auto [d, ok] = data_queue.dequeue();
    if (!ok) {
        this->stop("");
        ctx->set_state({
            .task = this->task.key,
            .variant = "error",
            .details = {
                {"running", false},
                {"message", "Failed to read data off device. Either disconnected or acquisition was disrupted."}
            }
        });
        return std::make_pair(
            Frame(0), freighter::Error(
                "Failed to read data off device. Either disconnected or acquisition was disrupted."));
    }

    uint64_t incr = ((d.tf - d.t0) / SCANS_PER_READ);

    auto f = synnax::Frame(this->reader_config.phys_channels.size() + this->reader_config.index_keys.size());
    int channel_count = 0;
    for (const auto &location: this->reader_config.phys_channels) {
        for (const auto &channel: this->reader_config.channels) {
            if (channel.location == location) {
                auto key = this->reader_config.channel_map[channel.location];
                auto s = synnax::Series(channel.data_type, SCANS_PER_READ);
                for (int sample = 0; sample < SCANS_PER_READ; sample++) {
                    write_to_series(s, d.data[sample * this->reader_config.phys_channels.size() + channel_count],
                                    channel.data_type);
                }
                f.add(key, std::move(s));
            }
        }
        channel_count++;
    }

    for (auto index_key: this->reader_config.index_keys) {
        auto t = synnax::Series(synnax::TIMESTAMP, SCANS_PER_READ);
        for (uint64_t i = 0; i < SCANS_PER_READ; i++) {
            t.write(d.t0 + incr * i);
        }
        f.add(index_key, std::move(t));
    }

    return std::make_pair(std::move(f), freighter::NIL);
}

std::pair<Frame, freighter::Error> labjack::ReaderSource::read(breaker::Breaker &breaker) {
    if (this->ok() == false) {
        return std::make_pair(
            Frame(0), freighter::Error("Device disconnected or is in error. Please reconfigure task and try again"));
    }
    if (this->reader_config.tc_channels.empty())
        return this->read_stream(breaker);
    return this->read_cmd_response(breaker);
}


labjack::ReaderSource::~ReaderSource() {
    this->stop("");
}

void labjack::ReaderSource::write_to_series(
    synnax::Series &series,
    double &data,
    synnax::DataType data_type) {
    if (data_type == synnax::FLOAT32) series.write(static_cast<float>(data));
    else if (data_type == synnax::FLOAT64) series.write(static_cast<double>(data));
    else if (data_type == synnax::SY_UINT8) series.write(static_cast<uint8_t>(data));
    else if (data_type == synnax::SY_UINT16) series.write(static_cast<uint16_t>(data));
    else if (data_type == synnax::INT16) series.write(static_cast<int16_t>(data));
    else if (data_type == synnax::UINT32) series.write(static_cast<uint32_t>(data));
    else if (data_type == synnax::INT32) series.write(static_cast<int32_t>(data));
    else if (data_type == synnax::UINT64) series.write(static_cast<uint64_t>(data));
    else if (data_type == synnax::INT64) series.write(static_cast<int64_t>(data));
    else {
        LOG(ERROR) << "Unsupported data type: " << data_type.value;
    }
}

void labjack::ReaderSource::acquire_data() {
    if (!this->reader_config.tc_channels.empty()) {
        return;
    }
    int numSkippedScans = 0;
    int totalSkippedScans = 0;
    int deviceScanBacklog = 0;
    while (this->breaker.running() && this->ok()) {
        DataPacket data_packet;
        data_packet.data.resize(this->buffer_size);
        data_packet.t0 = synnax::TimeStamp::now().value;
        if (check_err(
            LJM_eStreamRead(
                this->handle,
                data_packet.data.data(),
                &numSkippedScans,
                &deviceScanBacklog
            ), "acquire_data.LJM_eStreamRead")) break;
        data_packet.tf = synnax::TimeStamp::now().value;
        data_queue.enqueue(data_packet);
    }
    check_err(LJM_eStreamStop(handle), "acquire_data.LJM_eStreamStop");
}

void labjack::ReaderSource::configure_tc_ain_ef(TCConfig tc_config) {
    // set resolution index
    this->check_err(
        LJM_eWriteAddress(
            handle,
            41500 + tc_config.pos_chan,
            LJM_UINT16,
            0
        ), "configure_tc_ain_ef.LJM_eWriteAddress.resolutionIndex"
    );
    if (this->reader_config.device_type == "T7") {
        // For setting up the AIN#_NEGATIVE_CH (negative channel)
        this->check_err(
            LJM_eWriteAddress(
                handle,
                41000 + tc_config.pos_chan,
                LJM_UINT16,
                tc_config.neg_chan
            ), "configure_tc_ain_ef.LJM_eWriteAddress.negChan"
        );
    }
    // writing 5 frames of data to modbus registers: tc type, cjc address, slope, offset and units
    enum { NUM_FRAMES = 5 };
    int aAddresses[NUM_FRAMES];
    int aTypes[NUM_FRAMES];
    double aValues[NUM_FRAMES];
    int err_addr = INITIAL_ERR_ADDRESS;

    // For setting up the AIN#_EF_INDEX (thermocouple type)
    aAddresses[0] = 9000 + 2 * tc_config.pos_chan;
    aTypes[0] = LJM_UINT32;
    aValues[0] = TC_INDEX_LUT[tc_config.type - 6001];

    // For setting up the AIN#_EF_CONFIG_A (temperature units)
    aAddresses[1] = 9300 + 2 * tc_config.pos_chan;
    aTypes[1] = LJM_UINT32;

    if (tc_config.units == "K") aValues[1] = 0;
    else if (tc_config.units == "C") aValues[1] = 1;
    else if (tc_config.units == "F") aValues[1] = 2;

    // For setting up the AIN#_EF_CONFIG_B (CJC address)
    aAddresses[2] = 9600 + 2 * tc_config.pos_chan;
    aTypes[2] = LJM_UINT32;
    aValues[2] = tc_config.cjc_addr;

    // For setting up the AIN#_EF_CONFIG_D (CJC slope)
    aAddresses[3] = 10200 + 2 * tc_config.pos_chan;
    aTypes[3] = LJM_FLOAT32;
    aValues[3] = tc_config.cjc_slope;

    // For setting up the AIN#_EF_CONFIG_E (CJC offset)
    aAddresses[4] = 10500 + 2 * tc_config.pos_chan;
    aTypes[4] = LJM_FLOAT32;
    aValues[4] = tc_config.cjc_offset;

    this->check_err(
        LJM_eWriteAddresses(
            handle,
            NUM_FRAMES,
            aAddresses,
            aTypes,
            aValues,
            &err_addr
        ), "configure_tc_ain_ef.LJM_eWriteAddresses"
    );
}

int labjack::ReaderSource::check_err(int err, std::string caller) {
    labjack::check_err_internal(
        err,
        caller,
        "reader",
        this->ctx,
        this->ok_state,
        this->task.key
    );

    if (err == LJME_RECONNECT_FAILED ||
        err == LJME_NO_RESPONSE_BYTES_RECEIVED ||
        err == LJME_INCORRECT_NUM_COMMAND_BYTES_SENT ||
        err == LJME_NO_COMMAND_BYTES_SENT ||
        err == LJME_INCORRECT_NUM_RESPONSE_BYTES_RECEIVED
    ) {
        this->device_manager->close_device(this->reader_config.serial_number);
    }
    return err;
}

bool labjack::ReaderSource::ok() {
    return this->ok_state;
}

std::vector<synnax::ChannelKey> labjack::ReaderSource::get_ai_channel_keys() {
    std::vector<synnax::ChannelKey> keys;
    for (auto &channel: this->reader_config.channels) {
        if (channel.channel_type == "AI") {
            keys.push_back(channel.key);
        }
    }
    return keys;
}

void labjack::ReaderSource::log_err(std::string msg) {
    LOG(ERROR) << "[labjack.reader] " << msg;
    this->ok_state = false;
    ctx->set_state({
        .task = this->task.key,
        .variant = "error",
        .details = {
            {"running", false},
            {"message", msg}
        }
    });
}
