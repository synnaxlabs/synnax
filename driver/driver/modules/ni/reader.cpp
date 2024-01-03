//
// Created by Emiliano Bonilla on 1/3/24.
//

#include "reader.h"
#include "nlohmann/json.hpp"

using json = nlohmann::json;
using namespace ni;

typedef freighter::Error (*DAQmxCreateChannel) (TaskHandle taskHandle, ChannelConfig config);

freighter::Error create_ai_voltage_channel(TaskHandle taskHandle, ChannelConfig config) {
    json j = json::parse(config.config);
    auto physical_channel = j["physical_channel"].get<std::string>();
    auto max_val = j["max_val"].get<float>();
    auto min_val = j["min_val"].get<float>();
    DAQmxCreateAIVoltageChan(
            taskHandle,
            physical_channel.c_str(),
            NULL,
            DAQmx_Val_Cfg_Default,
            min_val,
            max_val,
            DAQmx_Val_Volts,
            NULL
    );
}

static std::map<std::string, DAQmxCreateChannel> create_channel_map = {
        {"ai_voltage", create_ai_voltage_channel}
};

freighter::Error Reader::start() {
    int32 daqmx_err = DAQmxCreateTask(config.name.c_str(), &task);
    uInt64 samples_per_chan = uInt64(config.sample_rate.value / config.transfer_rate.value);

    for (auto &channel: config.channels) {
        auto create_channel = create_channel_map[channel.type];
        auto err = create_channel(task, channel);
    }

    daqmx_err = DAQmxCfgSampClkTiming(
            task,
            NULL,
            config.sample_rate.value,
            DAQmx_Val_Rising,
            DAQmx_Val_ContSamps,
            samples_per_chan
    );

    daqmx_err = DAQmxStartTask(task);
}

freighter::Error Reader::stop() {
    int32 daqmx_err = DAQmxStopTask(task);
    daqmx_err = DAQmxClearTask(task);
}

std::pair<synnax::Frame, freighter::Error> Reader::read() {
    std::byte data[64];
    int32 samples_read;
    int32 daqmx_err = DAQmxReadAnalogF64(
            task,
            -1,
            0,
            DAQmx_Val_GroupByChannel,
            reinterpret_cast<float64 *>(data),
            64,
            &samples_read,
            NULL
    );

    return std::make_pair(std::move(frame), error);
}