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
    for (auto &channel : this->reader_config.channels) {
        keys.push_back(channel.channel_key);
    }
    return keys;
}

void labjack::Source::init(){
    // If already open, will return the same handle as opened device
    // TODO get device type and connection type from the config
    LJM_Open(LJM_dtANY, LJM_ctANY, this->reader_config.serial_number.c_str(), &this->handle);

    // iterate through the channels, for the ones that analog device, need to set the resolution index
    for (auto &channel : this->reader_config.channels) {
        if (channel.channel_types == "AIN") {
            int err = WriteName(this->handle, channel.location.c_str(), 0);
        }
        // TODO: if its T7/T8, will need to set range/gain configs like so:
    }
    int msDelay = 1000;
    auto err = LJM_StartInterval(
            this->handle, // TODO: need to keep unique to device will need to change once i want to define multiple intervals to read data at on a songel device
            msDelay * 1000
    );

    // TODO: check error
}


std::pair<Frame, freighter::Error> labjack::Source::read(breaker::Breaker &breaker) {
//    int err, error_address;
//    int msDelay = 1000; // TODO: change period?
//
//    std::vector<const char*> locations;
//    locations.reserve(config.channels.size());
//
//    std::vector<double> values;
//    locations.resize(config.channels.size());
//
//    // TODO: move this to initt
//    int num_names = 0;
//    for (const auto& channel : config.channels) {
//        if (channel.enabled) {
//            locations.push_back(channel.location.c_str());
//            num_names++;
//        }
//    }
//
////    auto t0 = synnax::TimeStamp::now().value;
//    err = LJM_WaitForNextInterval(INTERVAL_HANDLE, &SkippedIntervals);
//    ErrorCheck(err, "LJM_WaitForNextInterval");
//    if (SkippedIntervals > 0) {
//        printf("SkippedIntervals: %d\n", SkippedIntervals);
//    }
//    err = LJMeReadNames(
//            this->handle,
//            num_names,
//            locations.data(),
//            values.data(),
//            &error_address);
////    auto tf = synnax::TimeStamp::now().value;
//    // TOOD: add a breaker for sleep
//
//    // Error checking
//    if (err != LJME_NOERROR) {
//        std::cerr << "Error in LJM_eReadNames: " << err << std::endl;
//        // Handle error appropriately
//    }
//
//    // Print values
//    std::cout << std::fixed << std::setprecision(6);  // Set precision for floating-point output
//    for (int i = 0; i < num_names; ++i) {
//        std::cout << locations[i] << " : " << values[i] << " V";
//        if (i < num_names - 1) {
//            std::cout << ", ";
//        }
//    }
//    std::cout << std::endl;
//
//    // now i need to construct the frame
//
//    // iterate through locations
//    // for each location, find the corresponding channel in the config
//    auto f = synnax::Frame(num_names + num_index);
//    for(const auto &location : locations) {
//        for(const auto &channel : config.channels) {
//            if(channel.location == location) {
//               auto key = channel_map[channel.location];
//               auto s = synnax::Series(channel.datatype, 1);
//               write_to_series(s, values[location], channel.datatype);
//            }
//        }
//    }
//    // add index channels
//    for(auto channel : this->reader_config.channels){
//        if(channel.channel_types == "INDEX"){
//            auto t = synnax::Series(synnax::TIMESTAMP, {synnax::TimeStamp::now().value});
//            f.add(channel.channel_key, t);
//        }
//    }
    auto f = synnax::Frame(1); // TODO: REMOVE
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