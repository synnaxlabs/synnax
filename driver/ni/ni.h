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
#include <queue>
#include <utility>
#include <memory>
#include <atomic>
#include <thread>
#include <set>

#include "nidaqmx_api.h"
#include "daqmx.h"
#include "nisyscfg.h"
#include "client/cpp/synnax.h"
#include "driver/task/task.h"
#include "driver/pipeline/acquisition.h"
#include "driver/errors/errors.h" 
#include "driver/breaker/breaker.h"
#include "driver/pipeline/control.h"
#include "nlohmann/json.hpp" 
#include "driver/task/task.h"
#include "driver/config/config.h"
#include "driver/ni/error.h"
#include <condition_variable>
#include "spsc_queue.h"


// #include "driver/modules/module.h"

namespace ni{
    extern const std::map<std::string, int32_t> UNITS_MAP;
    typedef struct LinearScale{
        float64 slope;
        float64 offset;
        std::string prescaled_units;
        std::string scaled_units;
    } LinearScale;

    typedef struct MapScale{
        float64 prescaled_min;
        float64 prescaled_max;
        float64 scaled_min;
        float64 scaled_max;
        std::string prescaled_units;
        std::string scaled_units;
    } MapScale;

    typedef struct PolynomialScale{
        float64* forward_coeffs;
        float64* reverse_coeffs; 
        uint32_t num_coeffs;
        float64 min_x;
        float64 max_x;
        int32 num_points;
        int32 poly_order;
        std::string prescaled_units;
        std::string scaled_units;
    } PolynomialScale;

    typedef struct tableScale{
        float64* prescaled;
        float64* scaled;
        uint32_t num_points;
        std::string prescaled_units;
        std::string scaled_units;
    } TableScale;

    typedef union Scale{
        LinearScale linear;
        MapScale map;
        PolynomialScale polynomial;
        TableScale table;
        // Destructor
        ~Scale() {} 
    } Scale;

    typedef struct ChannelConfig{
        uint32_t channel_key;

        std::string name;
        std::string channel_type;
        int terminal_config;
        float min_val;
        float max_val;

        bool custom_scale;
        Scale* scale; 
        std::string scale_type;
        std::string scale_name;
    } ChannelConfig;

    typedef struct ReaderConfig{
        std::string device_key;
        std::vector<ChannelConfig> channels;
        std::uint64_t sample_rate = 0;
        std::uint64_t stream_rate = 0;
        std::string device_name;
        std::string task_name; 
        std::string timing_source; // for sample clock
        std::uint64_t period = 0;
        synnax::ChannelKey task_key;
        std::set<uint32_t> index_keys;
    } ReaderConfig;

    /////////////////////////////////////////////////////////////////////////////////////////////
    //                                                                                         //
    //                                     DAQ INTERFACES                                      //
    //                                                                                         //
    /////////////////////////////////////////////////////////////////////////////////////////////

    ///////////////////////////////////////////////////////////////////////////////////
    //                                    NiSource                                   //
    ///////////////////////////////////////////////////////////////////////////////////
    class Source : public pipeline::Source{
    public:
        explicit Source(TaskHandle task_handle,
                             const std::shared_ptr<task::Context> &ctx,
                             const synnax::Task task);

        
        int init();
        ~Source();
        void clearTask();
        int checkNIError(int32 error);  
        std::vector<synnax::ChannelKey> getChannelKeys();

        virtual void parseConfig(config::Parser & parser);
        virtual freighter::Error start();
        virtual freighter::Error stop();
        virtual bool ok(); 
        virtual void getIndexKeys();
        
        virtual std::pair<synnax::Frame, freighter::Error> read() = 0;
        virtual void parseChannels(config::Parser & parser) = 0;
        virtual int configureTiming() = 0; 
        virtual void acquireData() = 0;
        virtual int createChannels() = 0;

         typedef struct DataPacket{
            void* data; // actual data
            uint64_t t0;  // initial timestamp
            uint64_t tf;  // final timestamp
            int32 samplesReadPerChannel;
        } DataPacket;
        SPSCQueue<DataPacket> data_queue;


        TaskHandle task_handle = 0;
        ReaderConfig reader_config;
        uint64_t numChannels = 0;
        int numSamplesPerChannel = 0;
        int bufferSize = 0; 

        bool ok_state = true;
        json err_info;
        std::shared_ptr<task::Context> ctx;
        breaker::Breaker breaker;
        std::atomic<bool> running = false;
        std::thread sample_thread;
        synnax::Task task;
    };

    ///////////////////////////////////////////////////////////////////////////////////
    //                                    AnalogReadSource                           //
    ///////////////////////////////////////////////////////////////////////////////////
    class AnalogReadSource : public Source{ 
    public:
        explicit AnalogReadSource(
            TaskHandle task_handle,
            const std::shared_ptr<task::Context> &ctx,
            const synnax::Task task
        ) : Source(task_handle, ctx, task){}


        std::pair<synnax::Frame, freighter::Error> read() override;
        ~AnalogReadSource();
        // private helper functions
        void acquireData() override;
        int configureTiming() override;
        int createChannels() override;
        void parseChannels(config::Parser &parser) override;

        void parseCustomScale(config::Parser & parser, ChannelConfig & config);
        void deleteScales();
        int createChannel(ChannelConfig &channel);
   

        // NI related resources
        uint64_t numAIChannels = 0;
    };


     ///////////////////////////////////////////////////////////////////////////////////
    //                                    DigitalReadSource                           //
    ///////////////////////////////////////////////////////////////////////////////////
    class DigitalReadSource : public Source{ 
    public:
        explicit DigitalReadSource(
            TaskHandle task_handle,
            const std::shared_ptr<task::Context> &ctx,
            const synnax::Task task
        ) : Source(task_handle, ctx, task){}

        int configureTiming() override;
        void parseChannels(config::Parser & parser) override;    
        void createChannels() override;
        void acquireData() override;
        std::pair<synnax::Frame, freighter::Error> read() override;
    };



    ///////////////////////////////////////////////////////////////////////////////////
    //                                    StateSource                                //
    ///////////////////////////////////////////////////////////////////////////////////
    class StateSource : public pipeline::Source{
        public: 
            explicit StateSource() = default;
            explicit StateSource( std::uint64_t state_rate, synnax::ChannelKey &drive_state_index_key, std::vector<synnax::ChannelKey> &drive_state_channel_keys);
            std::pair<synnax::Frame, freighter::Error> read();
            freighter::Error start();
            freighter::Error stop();
            synnax::Frame getDriveState();
            void updateState(std::queue<synnax::ChannelKey> &modified_state_keys, std::queue<std::uint8_t> &modified_state_values);
        private:
            std::mutex state_mutex;
            std::condition_variable waiting_reader;
            std::uint64_t state_rate;
            std::chrono::duration<double> state_period;
            std::map<synnax::ChannelKey, uint8_t> state_map;
            synnax::ChannelKey drive_state_index_key;
    };


    ///////////////////////////////////////////////////////////////////////////////////
    //                                    DigitalWriteSink                           //
    ///////////////////////////////////////////////////////////////////////////////////
    typedef struct WriterConfig{
        std::vector<ChannelConfig> channels;
        std::uint64_t state_rate = 0;
        std::string device_name;
        std::string device_key;
        std::string task_name; 
        synnax::ChannelKey task_key;


        std::vector<synnax::ChannelKey> drive_state_channel_keys;
        std::vector<synnax::ChannelKey> drive_cmd_channel_keys;

        synnax::ChannelKey drive_state_index_key;
        std::queue<synnax::ChannelKey> modified_state_keys;
        std::queue<std::uint8_t> modified_state_values;
    } WriterConfig;

    class DigitalWriteSink : public pipeline::Sink{
    public:
        explicit DigitalWriteSink(TaskHandle task_handle,
                             const std::shared_ptr<task::Context> &ctx,
                             const synnax::Task task);

        int init();
        freighter::Error write(synnax::Frame frame);
        freighter::Error stop();
        freighter::Error start();
        std::vector<synnax::ChannelKey> getCmdChannelKeys();
        std::vector<synnax::ChannelKey> getStateChannelKeys();
        void getIndexKeys();
        bool ok();
        ~DigitalWriteSink();


        std::shared_ptr<ni::StateSource> writer_state_source;
    private:
        freighter::Error formatData(synnax::Frame frame);
        void parseConfig(config::Parser &parser);
        int checkNIError(int32 error);

        uint8_t *writeBuffer;  
        int bufferSize = 0;   
        int numSamplesPerChannel = 0;
        std::int64_t numChannels = 0;
        TaskHandle task_handle = 0;

        json err_info;

        bool ok_state = true;
        std::shared_ptr<task::Context> ctx;
        WriterConfig writer_config; 
        breaker::Breaker breaker;
    };

    ///////////////////////////////////////////////////////////////////////////////////
    //                                    Scanner                                    //
    ///////////////////////////////////////////////////////////////////////////////////
    class Scanner {
        public: 
            explicit Scanner() = default;
            explicit Scanner(   const std::shared_ptr<task::Context> &ctx,
                                const synnax::Task &task);
            ~Scanner();
            void scan();
            bool ok();
            json getDevices();
            void createDevices();
        private:
            json getDeviceProperties(NISysCfgResourceHandle resource);


            json devices;
            bool ok_state = true;
            NISysCfgSessionHandle session;
            NISysCfgFilterHandle filter;
            NISysCfgEnumResourceHandle resources_handle;
            synnax::Task task;
            std::shared_ptr<task::Context> ctx; 
    };
    
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
        explicit ScannerTask(   const std::shared_ptr<task::Context> &ctx, 
                                synnax::Task task); 

        void exec(task::Command &cmd) override;
        static std::unique_ptr<task::Task> configure(   const std::shared_ptr<task::Context> &ctx,
                                                        const synnax::Task &task);
        void run();
        void start();
        void stop();
        bool ok();
        ~ScannerTask();
    private:
        std::atomic<bool> running = false;
        ni::Scanner scanner;
        synnax::Task task;
        std::shared_ptr<task::Context> ctx;    
        std::thread thread;
        bool ok_state = true;
    };


    ///////////////////////////////////////////////////////////////////////////////////
    //                                    ReaderTask                                 //
    ///////////////////////////////////////////////////////////////////////////////////
    class ReaderTask final : public task::Task {
    public:
        explicit ReaderTask(    const std::shared_ptr<task::Context> &ctx, 
                                synnax::Task task); 
                                
        void exec(task::Command &cmd) override;

        void stop() override;
        void start();
        bool ok();

        static std::unique_ptr<task::Task> configure(   const std::shared_ptr<task::Context> &ctx,
                                                        const synnax::Task &task);
    private:
        std::atomic<bool>  running = false;
        pipeline::Acquisition daq_read_pipe; // source is a daqreader 
        TaskHandle task_handle;
        synnax::Task task;
        std::shared_ptr<task::Context> ctx;
        bool ok_state = true;
    };

    ///////////////////////////////////////////////////////////////////////////////////
    //                                    WriterTask                                 //
    ///////////////////////////////////////////////////////////////////////////////////
    class WriterTask final : public task::Task {
    public:
        explicit WriterTask(    const std::shared_ptr<task::Context> &ctx, 
                                synnax::Task task); 

        void exec(task::Command &cmd) override;
        void stop();
        void start();

        static std::unique_ptr<task::Task> configure(   const std::shared_ptr<task::Context> &ctx,
                                                        const synnax::Task &task);
        bool ok();
    private:
        std::atomic<bool>  running = false;
        pipeline::Control cmd_write_pipe;
        pipeline::Acquisition state_write_pipe;
        TaskHandle task_handle;
        synnax::Task task;
        std::shared_ptr<task::Context> ctx;
        bool ok_state = true;
    };


    ///////////////////////////////////////////////////////////////////////////////////
    //                                    Factory                                    //
    ///////////////////////////////////////////////////////////////////////////////////
    class Factory final : public task::Factory{
    public:
        std::pair<std::unique_ptr<task::Task>, bool> configureTask( const std::shared_ptr<task::Context> &ctx,
                                                                    const synnax::Task &task) override;

        std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task> > >
        configureInitialTasks(      const std::shared_ptr<task::Context> &ctx,
                                    const synnax::Rack &rack) override;

        ~Factory() = default;

    };
}




/*

using json = nlohmann::json;
namespace daq
{
    class DaqReader : public pipeline::Source  //TODD: change to daqReader
    {
    public:
        virtual std::pair<synnax::Frame, freighter::Error> read() = 0;
        virtual freighter::Error start() = 0;
        virtual freighter::Error stop() = 0;
        virtual bool ok() = 0;
    };

    class DaqWriter: public pipeline::Sink{
    public:
        virtual freighter::Error write(synnax::Frame frame) = 0;
        virtual freighter::Error start() = 0;
        virtual freighter::Error stop() = 0;
        // virtual bool ok() = 0;
    };


}


*/