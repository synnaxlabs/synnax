// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <vector>
#include <map>
#include <memory>
#include <set>
#include <thread>
#include <atomic>


#include "nidaqmx/nidaqmx_api.h"
#include "driver/ni/ni.h"
#include "driver/ni/channels.h"
#include "driver/queue/ts_queue.h"
#include "driver/pipeline/acquisition.h"
#include "driver/task/task.h"
#include "driver/breaker/breaker.h"
#include "driver/config/config.h"
#include "driver/loop/loop.h"

namespace ni {

struct ReaderChannelConfig {
    uint32_t channel_key;
    uint32_t state_channel_key;
    std::string name;
    std::string channel_type;
    std::shared_ptr<ni::Analog> ni_channel;
    bool enabled = true;
    synnax::DataType data_type;
};

struct ReaderConfig {
    std::string device_key;
    std::vector<ReaderChannelConfig> channels;
    synnax::Rate sample_rate = synnax::Rate(1);
    synnax::Rate stream_rate = synnax::Rate(1);
    std::string device_name;
    std::string task_name;
    std::string timing_source; // for sample clock
    std::uint64_t period = 0;
    synnax::ChannelKey task_key;
    std::set<uint32_t> index_keys;
};

///////////////////////////////////////////////////////////////////////////////////
//                                    Source                                     //
///////////////////////////////////////////////////////////////////////////////////
/// @brief an interface for a source that abstracts the common pattern of configuring and acquiring data
/// from a National Instruments device. Serves as base class for special purpose readers.
class Source : public pipeline::Source {
public:
    explicit Source(
        const std::shared_ptr<DAQmx> &dmx,
        TaskHandle task_handle,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task task
    );

    int init();

    ~Source();

    /// @brief performs type checking for synnax channels
    virtual int validate_channels() = 0;

    /// @brief quickly starts and stops task to check for immediate errors
    freighter::Error cycle();

    /// @brief wrapper which goes around any NI function calls and checks for errors
    /// @param error the error code returned by the NI function
    int check_ni_error(int32 error);

    /// @brief formats NI error into parseable format for console display
    void jsonify_error(std::string);

    void log_error(std::string err_msg);

    std::vector<synnax::ChannelKey> get_channel_keys();

    virtual void parse_config(config::Parser &parser);

    virtual freighter::Error start(const std::string &cmd_key);

    virtual freighter::Error stop(const std::string &cmd_key);

    virtual freighter::Error start_ni();

    virtual freighter::Error stop_ni();

    void clear_task();

    virtual void stopped_with_err(const freighter::Error &err) override;

    virtual bool ok();

    virtual void get_index_keys();

    virtual std::pair<synnax::Frame, freighter::Error>
    read(breaker::Breaker &breaker) = 0;

    virtual void parse_channels(config::Parser &parser) = 0;

    virtual int configure_timing() = 0;

    virtual void acquire_data() = 0;

    virtual int create_channels() = 0;

    /// @brief shared resources between daq sampling thread and acquisition thread
    struct DataPacket {
        std::vector<double> analog_data;
        std::vector<std::uint8_t> digital_data;
        uint64_t t0; // initial timestamp
        uint64_t tf; // final timestamp
        int32 samples_read_per_channel;
    };

    TSQueue<DataPacket> data_queue;
    std::thread sample_thread;

    /// @brief NI related resources
    TaskHandle task_handle = 0;
    ReaderConfig reader_config;
    int num_samples_per_channel = 0;
    int buffer_size = 0;
    uint64_t num_channels = 0;
    bool ok_state = true;

    /// @brief Synnax related resources
    json err_info;
    std::shared_ptr<task::Context> ctx;
    breaker::Breaker breaker;
    synnax::Task task;
    loop::Timer timer;
    loop::Timer sample_timer;

    const std::shared_ptr<DAQmx> dmx;

    /// @brief maps ni channel name to path in task configuration json
    std::map<std::string, std::string> channel_map;
}; // class Source

///////////////////////////////////////////////////////////////////////////////////
//                                    AnalogReadSource                           //
///////////////////////////////////////////////////////////////////////////////////
class AnalogReadSource final : public Source {
public:
    explicit AnalogReadSource(
        const std::shared_ptr<DAQmx> &dmx,
        TaskHandle task_handle,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) : Source(dmx, task_handle, ctx, task) {
    }

    std::pair<synnax::Frame, freighter::Error>
    read(breaker::Breaker &breaker) override;

    void acquire_data() override;

    int configure_timing() override;

    int create_channels() override;

    std::shared_ptr<ni::Analog> parse_channel(
        config::Parser &parser,
        const std::string &channel_type,
        const std::string &channel_name
    );

    void parse_channels(config::Parser &parser) override;

    int validate_channels() override;

    void write_to_series(synnax::Series &series, double &data,
                         synnax::DataType data_type);

    // NI related resources
    std::map<std::int32_t, std::string> port_to_channel;
    uint64_t num_ai_channels = 0;
}; // class AnalogReadSource

///////////////////////////////////////////////////////////////////////////////////
//                                    DigitalReadSource                          //
///////////////////////////////////////////////////////////////////////////////////
class DigitalReadSource final : public Source {
public:
    explicit DigitalReadSource(
        const std::shared_ptr<DAQmx> &dmx,
        TaskHandle task_handle,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) : Source(dmx, task_handle, ctx, task) {
    }

    std::pair<synnax::Frame, freighter::Error>
    read(breaker::Breaker &breaker) override;

    void acquire_data() override;

    int configure_timing() override;

    int validate_channels() override;

    int create_channels() override;

    void parse_channels(config::Parser &parser) override;
}; // class DigitalReadSource


///////////////////////////////////////////////////////////////////////////////////
//                                    ReaderTask                                 //
///////////////////////////////////////////////////////////////////////////////////
class ReaderTask final : public task::Task {
public:
    explicit ReaderTask(
        const std::shared_ptr<DAQmx> &dmx,
        const std::shared_ptr<task::Context> &ctx,
        synnax::Task task,
        std::shared_ptr<pipeline::Source> source,
        std::shared_ptr<ni::Source> ni_source,
        synnax::WriterConfig writer_config,
        const breaker::Config breaker_config
    );

    void exec(task::Command &cmd) override;

    void stop() override;

    void stop(const std::string &cmd_key);

    void start(const std::string &cmd_key);

    bool ok();

    std::string name() override { return task.name; }

    static std::unique_ptr<task::Task> configure(
        const std::shared_ptr<DAQmx> &dmx,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    );

private:
    const std::shared_ptr<DAQmx> &dmx;
    std::atomic<bool> running = false;
    std::shared_ptr<task::Context> ctx;
    synnax::Task task;
    pipeline::Acquisition daq_read_pipe; // source is a daqreader
    bool ok_state = true;
    std::shared_ptr<ni::Source> source;
    std::shared_ptr<pipeline::TareMiddleware> tare_mw;
}; // class ReaderTask
} // namespace ni