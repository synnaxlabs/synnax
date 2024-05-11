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

#include "nidaqmx_api.h"
#include "daqmx.h"
#include "nisyscfg.h"
#include "client/cpp/synnax.h"
#include "driver/task/task.h"
#include "driver/pipeline/daqReader.h"
#include "driver/pipeline/acquisition.h"
#include "driver/errors/errors.h" 
#include "driver/breaker/breaker.h"
#include "driver/pipeline/control.h"
#include "nlohmann/json.hpp" // for json parsing
#include "driver/task/task.h"
#include "driver/config/config.h"


// #include "driver/modules/module.h"

namespace ni{

    typedef struct ChannelConfig{
        /// @brief synnax properties
        std::uint32_t name_space;
        std::string node_id; // TODO: not actually parsed in from task config rn
        uint32_t channel_key;

        /// @brief NI properties
        std::string name;
        std::string channel_type;
        float min_val;
        float max_val;
    } ChannelConfig;



    ///////////////////////////////////////////////////////////////////////////////////
    //                                    daqReader                                  //
    ///////////////////////////////////////////////////////////////////////////////////

    typedef struct ReaderConfig{
        std::vector<ChannelConfig> channels;
        std::uint64_t acq_rate = 0;
        std::uint64_t stream_rate = 0;
        std::string device_name;
        std::string task_name; 
        std::string reader_type;
        synnax::ChannelKey task_key;
        bool isDigital = false;
    } ReaderConfig;


    class daqReader : public daq::daqReader{ // public keyword required to store pointer to niDaqreader in a pointer to acqReader
    public:
        // TODO: why do we not pass the task in by reference?
        explicit daqReader(TaskHandle taskHandle,
                             const std::shared_ptr<task::Context> &ctx,
                             const synnax::Task task);

        int init();
        std::pair<synnax::Frame, freighter::Error> read();
        freighter::Error stop();
        freighter::Error start();
        std::vector<synnax::ChannelKey> getChannelKeys();
        bool ok();
        ~daqReader();

    private:
        // private helper functions
        void parseDigitalReaderConfig(config::Parser & parser);
        void parseAnalogReaderConfig(config::Parser & parser);
        int checkNIError(int32 error);

        // NI related resources
        TaskHandle taskHandle = 0;

        double *data;       // pointer to heap allocated dataBuffer to provide to DAQmx read functions
        uInt8 *digitalData; // pointer to heap allocated dataBuffer to provide to DAQmx read functions
        int bufferSize = 0; 
        uint64_t numChannels = 0;
        int numSamplesPerChannel = 0;
        json err_info;


        // Server related resources
        ReaderConfig reader_config;
        std::shared_ptr<task::Context> ctx;
        breaker::Breaker breaker;
        bool ok_state = true;


        std::pair<synnax::Frame, freighter::Error> readAnalog();
        std::pair<synnax::Frame, freighter::Error> readDigital();
    };



    ///////////////////////////////////////////////////////////////////////////////////
    //                                    daqStateWriter                             //
    ///////////////////////////////////////////////////////////////////////////////////

    class daqStateWriter : public pipeline::Source{
        public: 
            explicit daqStateWriter() = default;
            explicit daqStateWriter( std::uint64_t state_rate, synnax::ChannelKey &drive_state_index_key, std::vector<synnax::ChannelKey> &drive_state_channel_keys);
            std::pair<synnax::Frame, freighter::Error> read();
            freighter::Error start();
            freighter::Error stop();
            synnax::Frame getDriveState();
            void updateState(std::queue<synnax::ChannelKey> &modified_state_keys, std::queue<std::uint8_t> &modified_state_values);
        private:
            std::mutex state_mutex;
            std::condition_variable waitingReader;
            std::uint64_t state_rate;
            std::chrono::duration<double> state_period;
            std::map<synnax::ChannelKey, uint8_t> state_map;
            synnax::ChannelKey drive_state_index_key;
    };


    ///////////////////////////////////////////////////////////////////////////////////
    //                                    daqWriter                                  //
    ///////////////////////////////////////////////////////////////////////////////////

    typedef struct WriterConfig{
        std::vector<ChannelConfig> channels;
        std::uint64_t state_rate = 0;
        std::string device_name;
        std::string task_name; 
        synnax::ChannelKey task_key;


        std::vector<synnax::ChannelKey> drive_state_channel_keys;
        std::vector<synnax::ChannelKey> drive_cmd_channel_keys;

        synnax::ChannelKey drive_state_index_key;
        std::queue<synnax::ChannelKey> modified_state_keys;
        std::queue<std::uint8_t> modified_state_values;
    } WriterConfig;

    class daqWriter : public daq::daqWriter{
    public:
        explicit daqWriter(TaskHandle taskHandle,
                             const std::shared_ptr<task::Context> &ctx,
                             const synnax::Task task);

        int init();
        freighter::Error write(synnax::Frame frame);
        freighter::Error stop();
        freighter::Error start();
        std::vector<synnax::ChannelKey> getCmdChannelKeys();
        std::vector<synnax::ChannelKey> getStateChannelKeys();
        bool ok();
        ~daqWriter();


        std::shared_ptr<ni::daqStateWriter> writer_state_source;
    private:
        // private helper functions
        freighter::Error writeDigital(synnax::Frame frame);
        freighter::Error formatData(synnax::Frame frame);
        void parseDigitalWriterConfig(config::Parser &parser);
        int checkNIError(int32 error);

        // NI related resources
        uint8_t *writeBuffer;  
        int bufferSize = 0;   
        int numSamplesPerChannel = 0;
        std::int64_t numChannels = 0;
        TaskHandle taskHandle = 0;

        json err_info;

        // Server related resources
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
            void testConnection();
            bool ok();
            json getDevices();
        private:
            NISysCfgResourceProperty getPropertyId(std::string property);
            void parseConfig(config::Parser &parser);
            json devices;
            json deviceProperties;
            json requestedProperties;
            bool ok_state = true;
            NISysCfgSessionHandle session;
            NISysCfgFilterHandle filter;
            NISysCfgEnumResourceHandle resourcesHandle;
            synnax::Task task;
            std::shared_ptr<task::Context> ctx; 
            static const std::vector<std::string> required_properties;
            static const std::vector<std::string> optional_properties;
    };

    ///////////////////////////////////////////////////////////////////////////////////
    //                                    ScannerTask                                //
    ///////////////////////////////////////////////////////////////////////////////////
    class ScannerTask final : public task::Task {
    public:
        explicit ScannerTask(   const std::shared_ptr<task::Context> &ctx, 
                                synnax::Task task); 

        void exec(task::Command &cmd) override;
        void stop() override{};
        static std::unique_ptr<task::Task> configure(   const std::shared_ptr<task::Context> &ctx,
                                                        const synnax::Task &task);
    private:
        ni::Scanner scanner;
        synnax::Task task;
        std::shared_ptr<task::Context> ctx;
    
    };


    ///////////////////////////////////////////////////////////////////////////////////
    //                                    ReaderTask                                 //
    ///////////////////////////////////////////////////////////////////////////////////

    class ReaderTask final : public task::Task {
    public:
        explicit ReaderTask(    const std::shared_ptr<task::Context> &ctx, 
                                synnax::Task task); 
                                
        void exec(task::Command &cmd) override;

        void stop() override{};

        static std::unique_ptr<task::Task> configure(   const std::shared_ptr<task::Context> &ctx,
                                                        const synnax::Task &task);
    private:
        pipeline::Acquisition daq_read_pipe; // source is a daqreader 
        TaskHandle taskHandle;
        synnax::Task task;
        std::shared_ptr<task::Context> ctx;
    };

    ///////////////////////////////////////////////////////////////////////////////////
    //                                    WriterTask                                 //
    ///////////////////////////////////////////////////////////////////////////////////

    class WriterTask final : public task::Task {
    public:
        explicit WriterTask(    const std::shared_ptr<task::Context> &ctx, 
                                synnax::Task task); 

        void exec(task::Command &cmd) override;
        void stop() override{};

        static std::unique_ptr<task::Task> configure(   const std::shared_ptr<task::Context> &ctx,
                                                        const synnax::Task &task);
    private:
        pipeline::Control cmd_write_pipe;
        pipeline::Acquisition state_write_pipe;
        TaskHandle taskHandle;
        synnax::Task task;
        std::shared_ptr<task::Context> ctx;
    };


    ///////////////////////////////////////////////////////////////////////////////////
    //                                    Factory                                    //
    ///////////////////////////////////////////////////////////////////////////////////

    class Factory final : public task::Factory{
    public:
        // member functions
        std::pair<std::unique_ptr<task::Task>, bool> configureTask( const std::shared_ptr<task::Context> &ctx,
                                                                    const synnax::Task &task) override;

        std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task> > >
        configureInitialTasks(      const std::shared_ptr<task::Context> &ctx,
                                    const synnax::Rack &rack) override;

        ~Factory() = default;

        // member variables
        std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task> > > tasks;
    };

    // TODO: move relevant interfaces here:


}




// namespace ni{
    
//     ///////////////////////////////////////////////////////////////////////////////////
//     //                                    NiReaderTask                               //
//     ///////////////////////////////////////////////////////////////////////////////////
//     class NiReaderTask final : public task::Task {
//     public:
//         explicit NiReaderTask(  const std::shared_ptr<task::Context> &ctx, 
//                                 synnax::Task task); 

//         static std::unique_ptr<task::Task> configure(
//             const std::shared_ptr<task::Context> &ctx,
//             const synnax::Task &task
//         );
                                
//         void exec(task::Command &cmd) override;

//         void stop() override{};
//     private:
//         pipeline::Acquisition daq_read_pipe; // source is a daqreader 
//         Taskhandle taskhandle;
//     }

//     ///////////////////////////////////////////////////////////////////////////////////
//     //                                    NiWriterTask                               //
//     ///////////////////////////////////////////////////////////////////////////////////
//     class NiWriterTask final : public task::Task {
//     public:
//         explicit NiWriterTask(  const std::shared_ptr<task::Context> &ctx, 
//                                 synnax::Task task); 

        
//         static std::unique_ptr<task::Task> configure(
//             const std::shared_ptr<task::Context> &ctx,
//             const synnax::Task &task
//         );

//         void exec(task::Command &cmd) override;
//         void stop() override{};
//     private:
//         pipeline::Acquisition cmd_read_pipe; // source reads from synnax (TODO: make this source)
//         pipeline::Control state_write_pipe;
//         Taskhandle taskhandle;
//     }


//     //////////////////////////////////////////////// OLD CODE



//     class niTaskFactory : public module::Factory {
//     public:
//         niTaskFactory() {};
//         std::unique_ptr<module::Module> createModule(TaskHandle taskhandle,
//                                                     const std::shared_ptr<synnax::Synnax> &client,
//                                                     const json &config,
//                                                     bool &valid_config,
//                                                     json &config_err);

//         std::unique_ptr <NiAnalogReaderTask> createAnalogReaderTask(TaskHandle taskhandle,
//                                                                     std::shared_ptr<synnax::Synnax> client,
//                                                                     bool &valid_config,
//                                                                     const json &config,
//                                                                     json &config_err);

//         std::unique_ptr <NiDigitalReaderTask> createDigitalReaderTask(TaskHandle taskhandle,
//                                                                     std::shared_ptr<synnax::Synnax> client,
//                                                                     bool &valid_config,
//                                                                     const json &config,
//                                                                     json &config_err);

//         std::unique_ptr <NiDigitalWriterTask> createDigitalWriterTask(TaskHandle taskhandle,
//                                                                     std::shared_ptr<synnax::Synnax> client,
//                                                                     bool &valid_config,
//                                                                     const json &config,
//                                                                     json &config_err);

//         bool validChannelConfig(const json &config, json &config_err);

//         // TODO: createDigitalReaderTask
//         // TODO: createDigitalWriterTask
        
//     };
// }