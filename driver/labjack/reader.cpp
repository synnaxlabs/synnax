// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/labjack/reader.h"

///////////////////////////////////////////////////////////////////////////////////
//                                   Source                                      //
///////////////////////////////////////////////////////////////////////////////////
void labjack::Source::stopped_with_err(const freighter::Error &err) {
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

std::vector<synnax::ChannelKey> labjack::Source::get_channel_keys() {
    std::vector<synnax::ChannelKey> keys;
    for (auto &channel : this->reader_config.channels)
        keys.push_back(channel.channel_key);
    for (auto &index_key : this->reader_config.index_keys)
        keys.push_back(index_key);
    return keys;
}

void labjack::Source::init(){
    this->init_stream();
}

void labjack::Source::init_stream(){
    LOG(INFO) << "initializing labjack device";
    LOG(INFO) << "reader_config address in init: " << (void*)&this->reader_config;


    LOG(INFO) << "reader config stuff" << this->reader_config.device_type;
    LOG(INFO) << "sample rate: " << this->reader_config.sample_rate.value;
//    double INIT_SCAN_RATE = this->reader_config.sample_rate.value;
//    LOG(INFO) << "INIT_SCAN_RATE: " << INIT_SCAN_RATE;
//    LOG(INFO) << "Stream rate is: " << this->reader_config.stream_rate.value;
//    int SCANS_PER_READ = (int)INIT_SCAN_RATE / this->reader_config.stream_rate.value;
//    double scanRate = SCANS_PER_READ;
//    LOG(INFO) << "checkpoint";
//    this->num_samples_per_chan = SCANS_PER_READ;
//    this->buffer_size = this->reader_config.phys_channels.size() * SCANS_PER_READ; // TODO: i might not need this
//
//    LJM_Open(LJM_dtANY, LJM_ctANY, this->reader_config.serial_number.c_str(), &this->handle);
//    this->port_addresses.resize(this->reader_config.phys_channels.size());
//
//    std::vector<const char*> phys_channel_names;
//    for (const auto& channel : this->reader_config.phys_channels) {
//        phys_channel_names.push_back(channel.c_str());
//    }
//
//    LOG(INFO) << "getting port addresses";
//    auto err = LJM_NamesToAddresses(this->reader_config.phys_channels.size(), phys_channel_names.data(), this->port_addresses.data(), NULL);
//    ErrorCheck(err, "[labjack.reader] LJM_NamesToAddresses error");
//
//    err = LJM_eStreamStart(handle, SCANS_PER_READ, this->reader_config.phys_channels.size(), this->port_addresses.data(), &scanRate);
//    ErrorCheck(err, "[labjack.reader] LJM_eStreamStart error");
    LOG(INFO) << "Finished init";
};

freighter::Error labjack::Source::start(const std::string &cmd_key){
    LOG(INFO) << "reader_config address in start: " << (void*)&this->reader_config;
    LOG(INFO) << "starting labjack device";
//    if(this->breaker.running()) {
//        LOG(INFO) << "breaker already running";
//        return freighter::NIL;
//    }
//    LOG(INFO) << "starting breaker";
//    this->breaker.start();
    LOG(INFO) << "breaker started";
    this->init(); // TODO: do some error handling here before you actually start the sample thread
    this->sample_thread = std::thread(&labjack::Source::acquire_data, this);
    ctx->setState({
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

freighter::Error labjack::Source::stop(const std::string &cmd_key) {
//    if(!this->breaker.running()) return freighter::NIL;
//    this->breaker.stop();

    if(this->sample_thread.joinable()) this->sample_thread.join();
    auto err = LJM_eStreamStop(handle);
    ErrorCheck(err, "[labjack.reader] LJM_eStreamStop error");

    CloseOrDie(this->handle);
    ctx->setState({
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


std::pair<Frame, freighter::Error> labjack::Source::read(breaker::Breaker &breaker) {
    // sleep for a millisecond
    std::this_thread::sleep_for(std::chrono::milliseconds(30));

    int SCANS_PER_READ = num_samples_per_chan;
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
//                    LOG(INFO) << "data: " << d.data[sample * this->reader_config.phys_channels.size() + channel_count];
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

labjack::Source::~Source() {
    auto err = LJM_CleanInterval(this->handle);
    PrintErrorIfError(err, "LJM_CleanInterval");
    CloseOrDie(this->handle);
}
void labjack::Source::write_to_series(
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

void labjack::Source::acquire_data(){
    int numSkippedScans = 0;
    int totalSkippedScans = 0;
    int deviceScanBacklog = 0;
//    while(this->breaker.running()){
    while(true){
        DataPacket data_packet;
        data_packet.data.resize(1000); // TODO: change size to be variable

        data_packet.t0 = synnax::TimeStamp::now().value;

        auto err = LJM_eStreamRead(this->handle, data_packet.data.data(), &numSkippedScans, &deviceScanBacklog);
        ErrorCheck(err, "[labjack.reader] LJM_eStreamRead error");
        data_packet.tf = synnax::TimeStamp::now().value;

        data_queue.enqueue(data_packet);
    }

    auto err = LJM_eStreamStop(handle);
}


////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
/// I might need later

//void labjack::Source::init_basic(){
//    LOG(INFO) << "initializing labjack device";
//    // If already open, will return the same handle as opened device
//    // TODO get device type and connection type from the config
//    LOG(INFO) << "Serial number is: " << this->reader_config.serial_number; //TODO: remove
//    LJM_Open(LJM_dtANY, LJM_ctANY, this->reader_config.serial_number.c_str(), &this->handle);
//
//    // iterate through the channels, for the ones that analog device, need to set the resolution index
//    for (auto &channel : this->reader_config.channels) {
//        if (channel.channel_types == "AIN") {
//
//            std::string name = channel.location + "_RESOLUTION_INDEX";
//            int err = WriteName(this->handle, name.c_str(), 0);
//
//            if(this->reader_config.device_type == "T7" || this->reader_config.device_type == "T8") {
//                auto name = channel.location + "_RANGE";
//                err = WriteName(this->handle, name.c_str(), 0);
//            }
//            if(this->reader_config.device_type == "T7") {
//                auto name = channel.location + "_NEGATIVE_CH";
//                err = WriteName(this->handle, name.c_str(), 10.0);
//            }
//
//        }
//    }
//    LOG(INFO) << "Sample rate: " << this->reader_config.sample_rate.value;
////    int msDelay = 1000;
//    auto err = LJM_StartInterval(
//            this->handle, // TODO: need to keep unique to device will need to change once i want to define multiple intervals to read data at on a songel device
//            this->reader_config.sample_rate.period().microseconds()
//    );
//
//    // TODO: check error
//}