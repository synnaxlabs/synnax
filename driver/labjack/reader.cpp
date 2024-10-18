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
    LOG(INFO) << "initializing labjack device";
    // If already open, will return the same handle as opened device
    // TODO get device type and connection type from the config
    LOG(INFO) << "Serial number is: " << this->reader_config.serial_number; //TODO: remove
    LJM_Open(LJM_dtANY, LJM_ctANY, this->reader_config.serial_number.c_str(), &this->handle);

    // iterate through the channels, for the ones that analog device, need to set the resolution index
    for (auto &channel : this->reader_config.channels) {
        if (channel.channel_types == "AIN") {

            std::string name = channel.location + "_RESOLUTION_INDEX";
            int err = WriteName(this->handle, name.c_str(), 0);

            if(this->reader_config.device_type == "T7" || this->reader_config.device_type == "T8") {
                auto name = channel.location + "_RANGE";
                err = WriteName(this->handle, name.c_str(), 0);
            }
            if(this->reader_config.device_type == "T7") {
                auto name = channel.location + "_NEGATIVE_CH";
                err = WriteName(this->handle, name.c_str(), 10.0);
            }

        }
    }
    LOG(INFO) << "Sample rate: " << this->reader_config.sample_rate.value;
//    int msDelay = 1000;
    auto err = LJM_StartInterval(
            this->handle, // TODO: need to keep unique to device will need to change once i want to define multiple intervals to read data at on a songel device
            this->reader_config.sample_rate.period().microseconds()
    );

    // TODO: check error
}


std::pair<Frame, freighter::Error> labjack::Source::read(breaker::Breaker &breaker) {
//    std::cout << "reading from labjack device";
    int err, error_address;
    int msDelay = 1000; // TODO: change period?

    std::vector<const char*> locations;
    locations.reserve(this->reader_config.channels.size());

    std::vector<double> values;
    values.resize(this->reader_config.channels.size());

    // TODO: move this to init
    int num_names = 0;
    for (const auto& channel : this->reader_config.channels) {
        if (channel.enabled) {
            locations.push_back(channel.location.c_str());
            num_names++;
        }
    }

    int SkippedIntervals;
    err = LJM_WaitForNextInterval(this->handle, &SkippedIntervals);
    ErrorCheck(err, "LJM_WaitForNextInterval");
//    if (SkippedIntervals > 0) {
//        printf("SkippedIntervals: %d\n", SkippedIntervals);
//    }
    err = LJM_eReadNames(
            this->handle,
            num_names,
            locations.data(),
            values.data(),
            &error_address);

    // TOOD: add a breaker for sleep

    // Error checking
    if (err != LJME_NOERROR) {
        std::cerr << "Error in LJM_eReadNames: " << err << std::endl;
    }
    // iterate through locations
    // for each location, find the corresponding channel in the config
    auto f = synnax::Frame(num_names + this->reader_config.index_keys.size());
    int index = 0;
    for(const auto &location : locations) {
        for(const auto &channel : this->reader_config.channels) {
            if(channel.location == location) {
               auto key = this->reader_config.channel_map[channel.location];
               auto s = synnax::Series(channel.data_type, 1);
               write_to_series(s, values[index], channel.data_type);
                f.add(key, std::move(s));
            }
        }
        index++;
    }
    for(auto index_key : this->reader_config.index_keys){
        auto t = synnax::Series(synnax::TIMESTAMP, 1);
        t.write(synnax::TimeStamp::now().value);
        f.add(index_key, std::move(t));
    }

//    LOG(INFO) << "Frame: " << f << std::endl;

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

