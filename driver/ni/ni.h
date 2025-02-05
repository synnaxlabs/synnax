// Copyright 2025 Synnax Labs, Inc.
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
#include <queue>
#include <utility>
#include <memory>
#include <atomic>
#include <thread>
#include <set>
#include <condition_variable>

#include "nidaqmx/nidaqmx_api.h"
#include "nidaqmx/nidaqmx.h"
#include "nisyscfg/nisyscfg.h"
#include "nisyscfg/nisyscfg_api.h"

#include "nlohmann/json.hpp"


#include "client/cpp/synnax.h"

#include "driver/ni/ai_channels.h"
#include "driver/ni/error.h"
#include "driver/queue/ts_queue.h"
#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/control.h"
#include "driver/pipeline/middleware.h"

#include "driver/task/task.h"
#include "driver/breaker/breaker.h"
#include "driver/config/config.h"
#include "driver/errors/errors.h"
#include "driver/loop/loop.h"

namespace ni {
inline const std::map<std::string, int32_t> UNITS_MAP = {
    {"Volts", DAQmx_Val_Volts},
    {"Amps", DAQmx_Val_Amps},
    {"DegF", DAQmx_Val_DegF},
    {"F", DAQmx_Val_DegF},
    {"DegC", DAQmx_Val_DegC},
    {"C", DAQmx_Val_DegC},
    {"Celsius", DAQmx_Val_DegC},
    {"Farenheit", DAQmx_Val_DegF},
    {"DegR", DAQmx_Val_DegR},
    {"Rankine", DAQmx_Val_DegR},
    {"Kelvins", DAQmx_Val_Kelvins},
    {"K", DAQmx_Val_Kelvins},
    {"Strain", DAQmx_Val_Strain},
    {"Ohms", DAQmx_Val_Ohms},
    {"Hz", DAQmx_Val_Hz},
    {"Seconds", DAQmx_Val_Seconds},
    {"Meters", DAQmx_Val_Meters},
    {"Inches", DAQmx_Val_Inches},
    {"Degrees", DAQmx_Val_Degrees},
    {"Radians", DAQmx_Val_Radians},
    {"g", DAQmx_Val_g},
    {"MetersPerSecondSquared", DAQmx_Val_MetersPerSecondSquared},
    {"MetersPerSecond", DAQmx_Val_MetersPerSecond},
    // TODO: make sure option is in console
    {"m/s", DAQmx_Val_MetersPerSecond},
    {"InchesPerSecond", DAQmx_Val_InchesPerSecond},
    // TODO: make sure option is in console
    {"mV/m/s", DAQmx_Val_MillivoltsPerMillimeterPerSecond},
    {"MillivoltsPerMillimeterPerSecond", DAQmx_Val_MillivoltsPerMillimeterPerSecond},
    {"MilliVoltsPerInchPerSecond", DAQmx_Val_MilliVoltsPerInchPerSecond},
    {"mVoltsPerNewton", DAQmx_Val_mVoltsPerNewton},
    {"mVoltsPerPound", DAQmx_Val_mVoltsPerPound},
    {"Newtons", DAQmx_Val_Newtons},
    {"Pounds", DAQmx_Val_Pounds},
    {"KilogramForce", DAQmx_Val_KilogramForce},
    {"PoundsPerSquareInch", DAQmx_Val_PoundsPerSquareInch},
    {"Bar", DAQmx_Val_Bar},
    {"Pascals", DAQmx_Val_Pascals},
    {"VoltsPerVolt", DAQmx_Val_VoltsPerVolt},
    {"mVoltsPerVolt", DAQmx_Val_mVoltsPerVolt},
    {"NewtonMeters", DAQmx_Val_NewtonMeters},
    {"InchOunces", DAQmx_Val_InchOunces},
    {"InchPounds", DAQmx_Val_InchPounds},
    {"FootPounds", DAQmx_Val_FootPounds},
    {"Strain", DAQmx_Val_Strain},
    {"FromTEDS", DAQmx_Val_FromTEDS},
    {"VoltsPerG", DAQmx_Val_VoltsPerG}, // TODO: verify this is an option in the console
    {"mVoltsPerG", DAQmx_Val_mVoltsPerG},
    // TODO: verify this is an option in the console
    {"AccelUnit_g", DAQmx_Val_AccelUnit_g}
    // TODO: verify this is an option in the console for sensitivity units
};

static std::string parse_digital_loc(config::Parser &p, const std::string &dev) {
    const auto port = std::to_string(p.required<std::uint64_t>("port"));
    const auto line = std::to_string(p.required<std::uint64_t>("line"));
    return dev + "/port" + port + "/line" + line;
}

struct ChannelConfig {
    uint32_t channel_key;
    uint32_t state_channel_key;
    std::string name;
    std::string channel_type;
    std::unique_ptr<ni::AIChan> ni_channel;
    bool enabled = true;
    synnax::DataType data_type;
};

struct ReaderConfig {
    std::string device_key;
    std::vector<ChannelConfig> channels;
    synnax::Rate sample_rate = synnax::Rate(1);
    synnax::Rate stream_rate = synnax::Rate(1);
    std::string device_name;
    std::string task_name;
    std::string timing_source; // for sample clock
    std::uint64_t period = 0;
    synnax::ChannelKey task_key;
    std::set<uint32_t> index_keys;
};

/////////////////////////////////////////////////////////////////////////////////////////////
//                                                                                         //
//                                     DAQ INTERFACES                                      //
//                                                                                         //
/////////////////////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////////
//                                    NiSource                                   //
///////////////////////////////////////////////////////////////////////////////////

/// @brief an interface for a source that abstracts the common pattern of configuring and acquiring data
/// from a National Instruments device. Serves as base class for special purpose readers.
class Source : public pipeline::Source {
public:
    explicit Source(
        const std::shared_ptr<DAQmx> &dmx,
        TaskHandle task_handle,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    );

    int init();

    ~Source() override;


    /// @brief performs type checking for synnax channels
    virtual int validate_channels() = 0;

    /// @brief quickly starts and stops task to check for immediate errors
    freighter::Error cycle();


    /// @brief wrapper which goes around any NI function calls and checks for errors
    /// @param error the error code returned by the NI function
    int check_ni_error(int32 error);

    /// @brief formats NI error into parseable format for console display
    void jsonify_error(std::string);


    void log_error(const std::string &err_msg);

    std::vector<synnax::ChannelKey> get_channel_keys() const;

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
    read(breaker::Breaker &breaker);

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
        const TaskHandle task_handle,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) : Source(dmx, task_handle, ctx, task) {
    }

    std::pair<synnax::Frame, freighter::Error>
    read(breaker::Breaker &breaker) override;

    void acquire_data() override;

    int configure_timing() override;

    int create_channels() override;

    std::unique_ptr<ni::AIChan> parse_channel(
        config::Parser &parser,
        const std::string &type,
        const std::string &name
    );

    void parse_channels(config::Parser &parser) override;

    int validate_channels() override;

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
//                                    StateSource                                //
///////////////////////////////////////////////////////////////////////////////////
class StateSource final : public pipeline::Source {
public:
    explicit StateSource() = default;

    explicit StateSource(
        float state_rate, // TODO: should this be a float?
        const synnax::ChannelKey &state_index_key,
        std::vector<synnax::ChannelKey> &state_channel_keys
    );

    std::pair<synnax::Frame, freighter::Error> read(breaker::Breaker &breaker) override;

    synnax::Frame get_state();

    void update_state(
        std::queue<synnax::ChannelKey> &modified_state_keys,
        std::queue<std::uint8_t> &modified_state_values
    );

private:
    std::mutex state_mutex;
    std::condition_variable waiting_reader;
    synnax::Rate state_rate = synnax::Rate(1);
    std::map<synnax::ChannelKey, uint8_t> state_map;
    synnax::ChannelKey state_index_key;
    loop::Timer timer;
}; // class StateSource

///////////////////////////////////////////////////////////////////////////////////
//                                    DigitalWriteSink                           //
///////////////////////////////////////////////////////////////////////////////////
struct WriterConfig {
    std::vector<ChannelConfig> channels;
    float state_rate = 0;
    std::string device_name;
    std::string device_key;
    std::string task_name;
    synnax::ChannelKey task_key;

    std::vector<synnax::ChannelKey> state_channel_keys;
    std::vector<synnax::ChannelKey> drive_cmd_channel_keys;

    synnax::ChannelKey state_index_key;
    std::queue<synnax::ChannelKey> modified_state_keys;
    std::queue<std::uint8_t> modified_state_values;
}; // struct WriterConfig

class DigitalWriteSink final : public pipeline::Sink {
public:
    explicit DigitalWriteSink(
        const std::shared_ptr<DAQmx> &dmx,
        TaskHandle task_handle,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    );

    ~DigitalWriteSink();

    int init();

    freighter::Error write(const synnax::Frame &frame) override;

    freighter::Error stop(const std::string &cmd_key);

    freighter::Error start(const std::string &cmd_key);

    freighter::Error start_ni();

    freighter::Error stop_ni();

    freighter::Error cycle();

    std::vector<synnax::ChannelKey> get_cmd_channel_keys();

    std::vector<synnax::ChannelKey> get_state_channel_keys();

    void get_index_keys();

    bool ok() const;

    void jsonify_error(std::string);

    void stopped_with_err(const freighter::Error &err) override;

    void log_error(const std::string &err_msg);

    void clear_task();

    std::shared_ptr<ni::StateSource> writer_state_source;

private:
    freighter::Error format_data(const synnax::Frame &frame);

    void parse_config(config::Parser &parser);

    int check_ni_error(int32 error);

    const std::shared_ptr<DAQmx> dmx;

    uint8_t *write_buffer = nullptr;
    int buffer_size = 0;
    int num_samples_per_channel = 0;
    TaskHandle task_handle = 0;

    uint64_t num_channels = 0;

    json err_info;

    bool ok_state = true;
    std::shared_ptr<task::Context> ctx;
    WriterConfig writer_config;
    breaker::Breaker breaker;
    synnax::Task task;
    std::map<std::string, std::string> channel_map;
}; // class DigitalWriteSink

///////////////////////////////////////////////////////////////////////////////////
//                                    Scanner                                    //
///////////////////////////////////////////////////////////////////////////////////
class Scanner final {
public:
    explicit Scanner() = default;

    explicit Scanner(
        const std::shared_ptr<SysCfg> &syscfg,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    );

    ~Scanner();

    void scan();

    bool ok() const;

    json get_devices();

    void create_devices();

    void set_scan_thread(const std::shared_ptr<std::thread> &scan_thread);

    void join_scan_thread() const;

    void log_err(std::string err_msg);

private:
    const std::vector<std::string> IGNORED_MODEL_PREFIXES = {"O", "cRIO", "nown"};  // Add more prefixes as needed

    std::shared_ptr<SysCfg> syscfg;

    json get_device_properties(NISysCfgResourceHandle resource);

    json devices;
    std::set<std::string> device_keys;
    bool ok_state = true;
    NISysCfgSessionHandle session;
    NISysCfgFilterHandle filter;
    NISysCfgEnumResourceHandle resources_handle;
    synnax::Task task;
    std::shared_ptr<task::Context> ctx;
    std::shared_ptr<std::thread> scan_thread = nullptr;
    //optional scan thread a task could be running
}; // class Scanner

/////////////////////////////////////////////////////////////////////////////////////////////
//                                                                                         //
//                                    TASK INTERFACES                                      //
//                                                                                         //
/////////////////////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////////
//                                    ScannerTask                                //
///////////////////////////////////////////////////////////////////////////////////
class ScannerTask final : public task::Task {
public:
    explicit ScannerTask(
        const std::shared_ptr<SysCfg> &syscfg,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    );

    void exec(task::Command &cmd) override;

    static std::unique_ptr<task::Task> configure(
        const std::shared_ptr<SysCfg> &syscfg,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    );

    void run();

    void stop() override;

    std::string name() override { return task.name; }

    bool ok() const;

private:
    std::shared_ptr<SysCfg> syscfg;
    breaker::Breaker breaker;
    ni::Scanner scanner;
    std::shared_ptr<task::Context> ctx;
    synnax::Task task;
    std::shared_ptr<std::thread> thread;
    bool ok_state = true;
    synnax::Rate scan_rate = synnax::Rate(1);
}; // class ScannerTask
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
        const breaker::Config &breaker_config
    );

    void exec(task::Command &cmd) override;

    void stop() override;

    void stop(const std::string &cmd_key);

    void start(const std::string &cmd_key);

    bool ok() const;

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

///////////////////////////////////////////////////////////////////////////////////
//                                    WriterTask                                 //
///////////////////////////////////////////////////////////////////////////////////
class WriterTask final : public task::Task {
public:
    explicit WriterTask(const std::shared_ptr<task::Context> &ctx,
                        synnax::Task task,
                        std::shared_ptr<pipeline::Sink> sink,
                        std::shared_ptr<ni::DigitalWriteSink> ni_sink,
                        std::shared_ptr<pipeline::Source> writer_state_source,
                        synnax::WriterConfig writer_config,
                        synnax::StreamerConfig streamer_config,
                        const breaker::Config &breaker_config);

    explicit WriterTask() = default;

    void exec(task::Command &cmd) override;

    void stop() override;

    void stop(const std::string &cmd_key);

    void start(const std::string &cmd_key);

    static std::unique_ptr<task::Task> configure(
        const std::shared_ptr<DAQmx> &dmx,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    );

    bool ok() const;

    std::string name() override { return task.name; }

private:
    std::atomic<bool> running = false;
    std::shared_ptr<task::Context> ctx;
    synnax::Task task;
    pipeline::Control cmd_write_pipe;
    pipeline::Acquisition state_write_pipe;
    bool ok_state = true;
    std::shared_ptr<ni::DigitalWriteSink> sink;
}; // class WriterTask

///////////////////////////////////////////////////////////////////////////////////
//                                    Factory                                    //
///////////////////////////////////////////////////////////////////////////////////
class Factory final : public task::Factory {
public:
    Factory(
        const std::shared_ptr<DAQmx> &dmx,
        const std::shared_ptr<SysCfg> &syscfg
    );

    bool check_health(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) const;

    static std::shared_ptr<ni::Factory> create();

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task) override;

    std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task> > >
    configure_initial_tasks(const std::shared_ptr<task::Context> &ctx,
                            const synnax::Rack &rack) override;

private:
    bool dlls_present = false;
    std::shared_ptr<DAQmx> dmx;
    std::shared_ptr<SysCfg> syscfg;
};

const std::string INTEGRATION_NAME = "ni";
} // namespace ni
