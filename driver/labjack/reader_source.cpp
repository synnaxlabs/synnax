// Copyright 2024 Synnax Labs, Inc.
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
void labjack::ReaderSource::stopped_with_err(const freighter::Error &err) {
    LOG(ERROR) << "stopped with error: " << err.message();
    json j = json(err.message());
    this->ctx->setState({
                                .task = this->reader_config.task_key,
                                .variant = "error",
                                .details = {
                                        {"running", false},
                                        {"message", j}
                                }
                        });
}

std::vector<synnax::ChannelKey> labjack::ReaderSource::get_channel_keys() {
    std::vector<synnax::ChannelKey> keys;
    for (auto &channel : this->reader_config.channels) {
        keys.push_back(channel.key);
        // get index key
        auto [channel_info, err] = this->ctx->client->channels.retrieve(channel.key);
        if(err != freighter::NIL) {
            LOG(ERROR) << "[labjack.reader] Error retrieving channel: " << err.message();
            continue;
        }
        LOG(INFO) << "[labjack.reader] index key: " << channel_info.index;
        this->reader_config.index_keys.insert(channel_info.index);
    }
    for (auto &index_key : this->reader_config.index_keys)
        keys.push_back(index_key);
    return keys;
}

void labjack::ReaderSource::init(){
    if(this->reader_config.device_type != "") return this->init_stream();
    auto [dev, err] = this->ctx->client->hardware.retrieveDevice(
            this->reader_config.device_key
    );
    if(err != freighter::NIL){
        LOG(ERROR) << "[labjack.reader] Error retrieving device: " << err.message();
        return;
    }

    if(dev.model == "LJM_dtT4"){
        this->reader_config.device_type = "T4";
    } else if(dev.model == "LJM_dtT7"){
        this->reader_config.device_type = "T7";
    } else if(dev.model == "LJM_dtT8"){
        this->reader_config.device_type = "T8";
    } else {
        LOG(ERROR) << "[labjack.reader] Unsupported device type: " << dev.model;
        return;
    }
    LOG(INFO) << "[labjack.reader] device type: " << this->reader_config.device_type;
    this->init_stream();
}

void labjack::ReaderSource::init_stream(){
    LOG(INFO) << "[labjack.reader] initializing stream";
    double INIT_SCAN_RATE = this->reader_config.sample_rate.value;
    int SCANS_PER_READ = (int)INIT_SCAN_RATE / this->reader_config.stream_rate.value;
    double scanRate = INIT_SCAN_RATE;

    this->num_samples_per_chan = SCANS_PER_READ;
    this->buffer_size = this->reader_config.phys_channels.size() * SCANS_PER_READ;

    {
        std::lock_guard<std::mutex> lock(labjack::device_mutex);
        if(check_err(LJM_Open(LJM_dtANY, LJM_ctANY, this->reader_config.serial_number.c_str(), &this->handle))){
            LOG(ERROR) << "[labjack.reader] LJM_Open error";
            return;
        }
    }
    LOG(INFO) << "[labjack.reader] device opened successfully";
    // iterate through the channels, for the ones that analog device, need to set the resolution index
    for (auto &channel : this->reader_config.channels) {
        if (channel.channel_type == "AI") {
            // Set resolution index to device's default setting (value = 0)
            std::string name = channel.location + "_RESOLUTION_INDEX";
            check_err(WriteName(this->handle, name.c_str(), 0));

            if(this->reader_config.device_type == "T7" || this->reader_config.device_type == "T8") {
                auto name = channel.location + "_RANGE";
                check_err(WriteName(this->handle, name.c_str(), 0));
            }
            if(this->reader_config.device_type == "T7") {
                auto name = channel.location + "_NEGATIVE_CH";
                check_err(WriteName(this->handle, name.c_str(), 10.0));
            }
        } else if (channel.channel_type == "TC"){
            // Set resolution index to device's default setting (value = 0)
            std::string name = channel.location + "_RESOLUTION_INDEX";
            check_err(WriteName(this->handle, name.c_str(), 0));

            if(this->reader_config.device_type == "T4"){
                LOG(ERROR) << "[labjack.driver] thermocouple channels not currently supported for T4 devices";
                continue;
            }

            this->configure_tc_ain_ef(channel.tc_config);

        }
    }
    // TODO: figure out if i need to set this
    //    auto err = LJM_StartInterval(
    //            this->handle,
    //            this->reader_config.sample_rate.period().microseconds()
    //    );
    this->port_addresses.resize(this->reader_config.phys_channels.size());

    std::vector<const char*> phys_channel_names;
    for (const auto& channel : this->reader_config.phys_channels) {
        phys_channel_names.push_back(channel.c_str());
    }
    check_err(LJM_NamesToAddresses(this->reader_config.phys_channels.size(), phys_channel_names.data(), this->port_addresses.data(), NULL));
    LJM_eStreamStop(handle); // in case it was running
    check_err(LJM_eStreamStart(handle, SCANS_PER_READ, this->reader_config.phys_channels.size(), this->port_addresses.data(), &scanRate));
};

freighter::Error labjack::ReaderSource::start(const std::string &cmd_key){
    if(this->breaker.running()) {
        LOG(INFO) << "[labjack.reader] breaker already running";
        return freighter::NIL;
    }
    this->breaker.start();
    this->init();
    if(!this->ok()) {
        LOG(ERROR) << "Device not initialized properly. Requires reconfigure.";
        return freighter::Error("Device not initialized properly. Requires reconfigure.");
    }
    this->sample_thread = std::thread(&labjack::ReaderSource::acquire_data, this);
    ctx->setState({
          .task = task.key,
          .key = cmd_key,
          .variant = "success",
          .details = {
                  {"running", true},
                  {"message", "Task started successfully"}
          }
    });
    LOG(INFO) << "[labjack.reader] labjack device started successfully";
    return freighter::NIL;
};

freighter::Error labjack::ReaderSource::stop(const std::string &cmd_key) {
    if(!this->breaker.running()) return freighter::NIL;
    this->breaker.stop();

    if(this->sample_thread.joinable()) this->sample_thread.join();
    check_err(LJM_eStreamStop(handle));
    check_err(LJM_Close(this->handle));

    ctx->setState({
          .task = task.key,
          .key = cmd_key,
          .variant = "success",
          .details = {
                  {"running", false},
                  {"message", "Task stopped successfully"}
          }
    });
    LOG(INFO) << "[labjack.reader] labjack device stopped successfully";
    return freighter::NIL;
}


std::pair<Frame, freighter::Error> labjack::ReaderSource::read(breaker::Breaker &breaker) {
    // sleep for a millisecond
    std::this_thread::sleep_for(std::chrono::milliseconds(30));

    int SCANS_PER_READ = this->num_samples_per_chan;
    auto [d, err] = data_queue.dequeue();

    uint64_t incr = ((d.tf - d.t0) / SCANS_PER_READ);

    auto f = synnax::Frame(this->reader_config.phys_channels.size() + this->reader_config.index_keys.size());
    int channel_count = 0;
    for(const auto &location : this->reader_config.phys_channels) {
        for(const auto &channel : this->reader_config.channels) {
            if(channel.location == location) {
                auto key = this->reader_config.channel_map[channel.location];
                auto s = synnax::Series(channel.data_type, SCANS_PER_READ);
                for (int sample = 0; sample < SCANS_PER_READ; sample++) {
                    write_to_series(s, d.data[sample * this->reader_config.phys_channels.size() + channel_count], channel.data_type);
                }
                f.add(key, std::move(s));
            }
        }
        channel_count++;
    }

    for( auto index_key : this->reader_config.index_keys){
        auto t = synnax::Series(synnax::TIMESTAMP, SCANS_PER_READ);
        for (uint64_t i = 0; i < SCANS_PER_READ; i++){
            t.write(d.t0 + incr * i);
        }
        f.add(index_key, std::move(t));
    }

    return std::make_pair(std::move(f), freighter::NIL);
}

labjack::ReaderSource::~ReaderSource() {
    this->stop("");
//    check_err(LJM_CleanInterval(handle));
    check_err(LJM_Close(this->handle));
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

void labjack::ReaderSource::acquire_data(){
    int numSkippedScans = 0;
    int totalSkippedScans = 0;
    int deviceScanBacklog = 0;
    while(this->breaker.running() && this->ok()){
        DataPacket data_packet;
        data_packet.data.resize(this->buffer_size);

        data_packet.t0 = synnax::TimeStamp::now().value;
        if(check_err(
                LJM_eStreamRead(
                    this->handle,
                    data_packet.data.data(),
                    &numSkippedScans,
                    &deviceScanBacklog
                ))){
            LOG(ERROR) << "[labjack.reader] LJM_eStreamRead error";
            break;
        }
        data_packet.tf = synnax::TimeStamp::now().value;
        data_queue.enqueue(data_packet);
    }
    check_err(LJM_eStreamStop(handle));
    LOG(INFO) << "[labjack.reader] acquire_data loop stopped successfully";
}

void labjack::ReaderSource::configure_tc_ain_ef(TCConfig tc_config){
    // writing 5 frames of data to modbus registers: tc type, cjc address, slope, offset and units
    enum{ NUM_FRAMES = 5};
    int aAddresses[NUM_FRAMES];
    int aTypes[NUM_FRAMES];
    double aValues[NUM_FRAMES];
    int err_addr = INITIAL_ERR_ADDRESS;

    // For setting up the AIN#_EF_INDEX (thermocouple type)
    aAddresses[0] = 9000+2*tc_config.pos_chan;
    aTypes[0] = LJM_UINT32;
    aValues[0] = TC_INDEX_LUT[tc_config.type - 6001];

    // For setting up the AIN#_EF_CONFIG_A (temperature units)
    aAddresses[1] = 9300+2*tc_config.pos_chan;
    aTypes[1] = LJM_UINT32;

    if(tc_config.units == "K") aValues[1] = 0;
    else if(tc_config.units == "C") aValues[1] = 1;
    else if(tc_config.units == "F") aValues[1] = 2;


    // For setting up the AIN#_EF_CONFIG_B (CJC address)
    aAddresses[2] = 9600+2*tc_config.pos_chan;
    aTypes[2] = LJM_UINT32;
    aValues[2] = tc_config.cjc_addr;

    // For setting up the AIN#_EF_CONFIG_D (CJC slope)
    aAddresses[3] = 10200+2*tc_config.pos_chan;
    aTypes[3] = LJM_FLOAT32;
    aValues[3] = tc_config.cjc_slope;

    // For setting up the AIN#_EF_CONFIG_E (CJC offset)
    aAddresses[4] = 10500+2*tc_config.pos_chan;
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
            )
        );

    if(this->reader_config.device_type == "T7") {
        // For setting up the AIN#_NEGATIVE_CH (negative channel)
        this->check_err(
                LJM_eWriteAddress(
                    handle,
                    41000+tc_config.pos_chan,
                    LJM_UINT32,
                    tc_config.neg_chan
                )
            );
    }
}

int labjack::ReaderSource::check_err(int err){
    if(err == 0) return 0;

    char err_msg[LJM_MAX_NAME_SIZE];
    LJM_ErrorToString(err, err_msg);

    this->ctx->setState({
        .task = this->task.key,
        .variant = "error",
        .details = {
            {"running", false},
            {"message", err_msg}
        }
    });

    LOG(ERROR) << "[labjack.reader] " << err_msg;

    this->ok_state = false;
    return -1;
}


bool labjack::ReaderSource::ok(){
    return this->ok_state;
}