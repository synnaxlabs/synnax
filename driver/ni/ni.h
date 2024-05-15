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
#include "driver/pipeline/daqReader.h"
#include "driver/pipeline/acquisition.h"
#include "driver/errors/errors.h" 
#include "driver/breaker/breaker.h"
#include "driver/pipeline/control.h"
#include "nlohmann/json.hpp" // for json parsing
#include "driver/task/task.h"
#include "driver/config/config.h"
#include "driver/ni/error.h"
#include <condition_variable>


// #include "driver/modules/module.h"

namespace ni{

    // TODO: make const
    // std::map<std::string, uint32_t> NI_UNITS_MAP = {
    //     {"Volts", DAQmx_Val_Volts},
    //     {"Amps", DAQmx_Val_Amps},
    //     {"DegF", DAQmx_Val_DegF},
    //     {"DegC", DAQmx_Val_DegC},
    //     {"DegR", DAQmx_Val_DegR},
    //     {"Kelvins", DAQmx_Val_Kelvins},
    //     {"Strain", DAQmx_Val_Strain},
    //     {"Ohms", DAQmx_Val_Ohms},
    //     {"Hz", DAQmx_Val_Hz},
    //     {"Seconds", DAQmx_Val_Seconds},
    //     {"Meters", DAQmx_Val_Meters},
    //     {"Inches", DAQmx_Val_Inches},
    //     {"Degrees", DAQmx_Val_Degrees},
    //     {"Radians", DAQmx_Val_Radians},
    //     {"g", DAQmx_Val_g},
    //     {"MetersPerSecondSquared", DAQmx_Val_MetersPerSecondSquared},
    //     {"Newtons", DAQmx_Val_Newtons},
    //     {"Pounds", DAQmx_Val_Pounds},
    //     {"KilogramForce", DAQmx_Val_KilogramForce},
    //     {"PoundsPerSquareInch", DAQmx_Val_PoundsPerSquareInch},
    //     {"Bar", DAQmx_Val_Bar},
    //     {"Pascals", DAQmx_Val_Pascals},
    //     {"VoltsPerVolt", DAQmx_Val_VoltsPerVolt},
    //     {"mVoltsPerVolt", DAQmx_Val_mVoltsPerVolt},
    //     {"NewtonMeters", DAQmx_Val_NewtonMeters},
    //     {"InchOunces", DAQmx_Val_InchOunces},
    //     {"InchPounds", DAQmx_Val_InchPounds},
    //     {"FootPounds", DAQmx_Val_FootPounds},
    //     {"Strain", DAQmx_Val_Strain},
    //     {"FromTEDS", DAQmx_Val_FromTEDS}
    // };

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
        float* forward_coeffs;
        float* reverse_coeffs; 
        uint32_t num_coeffs;
        float64 min_x;
        float64 max_x;
        int32 num_points;
        int32 poly_order;
        std::string prescaled_units;
        std::string scaled_units;
    } PolynomialScale;

    typedef struct tableScale{
        float* prescaled;
        float* scaled;
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
        /// @brief synnax properties
        std::uint32_t name_space;
        std::string node_id; // TODO: not actually parsed in from task config rn
        uint32_t channel_key;
        // uint32_t* index_key;

        /// @brief NI properties
        std::string name;
        std::string channel_type;
        int terminal_config;
        float min_val;
        float max_val;

        bool custom_scale;
        Scale* scale; // TODO: make pointer
        std::string scale_type;
        std::string scale_name;

        // Default constructor
        ChannelConfig() 
            : name_space(0), channel_key(0), terminal_config(-1),  min_val(0.0f), max_val(0.0f),
                custom_scale(false), scale(nullptr), scale_type(""), scale_name("") {}

        // Destructor
        ~ChannelConfig() {}
    } ChannelConfig;




    typedef struct ReaderConfig{
        std::string device_key;
        std::vector<ChannelConfig> channels;
        std::uint64_t acq_rate = 0;
        std::uint64_t stream_rate = 0;
        std::string device_name;
        std::string task_name; 
        synnax::ChannelKey task_key;
        std::set<uint32_t> index_keys;
    } ReaderConfig;

    /////////////////////////////////////////////////////////////////////////////////////////////
    //                                                                                         //
    //                                     DAQ INTERFACES                                      //
    //                                                                                         //
    /////////////////////////////////////////////////////////////////////////////////////////////

    ///////////////////////////////////////////////////////////////////////////////////
    //                                    daqAnalogReader                            //
    ///////////////////////////////////////////////////////////////////////////////////
    class DaqAnalogReader : public daq::DaqReader{ // public keyword required to store pointer to niDaqreader in a pointer to acqReader
    public:
        // TODO: why do we not pass the task in by reference?
        explicit DaqAnalogReader(TaskHandle taskHandle,
                             const std::shared_ptr<task::Context> &ctx,
                             const synnax::Task task);

        int init();
        freighter::Error start();
        freighter::Error stop();
        std::pair<synnax::Frame, freighter::Error> read();
        bool ok();
        ~DaqAnalogReader();
        void getIndexKeys(); // TODO make a helper not a member function
        std::vector<synnax::ChannelKey> getChannelKeys();
    private:
        // private helper functions
        void parseConfig(config::Parser & parser);
        int checkNIError(int32 error);  
        void parseCustomScale(config::Parser & parser, ChannelConfig & config);
        void deleteScales();
        int createChannel(ni::ChannelConfig &channel);

        // NI related resources
        bool running = false;
        TaskHandle taskHandle = 0;
        double *data;       // pointer to heap allocated dataBuffer to provide to DAQmx read functions
        uint64_t numChannels = 0;
        int numSamplesPerChannel = 0;
        json err_info;


        // Server related resources
        ReaderConfig reader_config;
        std::shared_ptr<task::Context> ctx;
        breaker::Breaker breaker;
        bool ok_state = true;
        int bufferSize = 0; // TODO: make this a member variable
    };


     ///////////////////////////////////////////////////////////////////////////////////
    //                                    DaqDigitalReader                            //
    ///////////////////////////////////////////////////////////////////////////////////
    class DaqDigitalReader : public daq::DaqReader{ // public keyword required to store pointer to niDaqreader in a pointer to acqReader
    public:
        // TODO: why do we not pass the task in by reference?
        explicit DaqDigitalReader(TaskHandle taskHandle,
                             const std::shared_ptr<task::Context> &ctx,
                             const synnax::Task task);


        int init();
        std::pair<synnax::Frame, freighter::Error> read();
        freighter::Error stop();
        freighter::Error start();
        bool ok();
        ~DaqDigitalReader();
        std::vector<synnax::ChannelKey> getChannelKeys();
    private:
        // private helper functions
        void parseConfig(config::Parser & parser);
        int checkNIError(int32 error);
        // NI related resources
        bool running = false;
        TaskHandle taskHandle = 0;
        double *data;       // pointer to heap allocated dataBuffer to provide to DAQmx read functions
        int bufferSize = 0; 
        uint64_t numChannels = 0;
        int numSamplesPerChannel = 0;
        json err_info;

        // Server related resources
        ReaderConfig reader_config;
        std::shared_ptr<task::Context> ctx;
        breaker::Breaker breaker;
        bool ok_state = true;
    };



    ///////////////////////////////////////////////////////////////////////////////////
    //                                    DaqStateWriter                             //
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
    //                                    DaqDigitalWriter                           //
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

    class DaqDigitalWriter : public daq::DaqWriter{
    public:
        explicit DaqDigitalWriter(TaskHandle taskHandle,
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
        ~DaqDigitalWriter();


        std::shared_ptr<ni::daqStateWriter> writer_state_source;
    private:
        // private helper functions
        freighter::Error formatData(synnax::Frame frame);
        void parseConfig(config::Parser &parser);
        int checkNIError(int32 error);

        // NI related resources
        uint8_t *writeBuffer;  
        int bufferSize = 0;   
        int numSamplesPerChannel = 0;
        std::int64_t numChannels = 0;
        TaskHandle taskHandle = 0;
        bool running = false;

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
            void createDevices();
        private:
            json getDeviceProperties(NISysCfgResourceHandle resource);


            json devices;
            json deviceProperties;
            json requestedProperties;
            bool ok_state = true;
            NISysCfgSessionHandle session;
            NISysCfgFilterHandle filter;
            NISysCfgEnumResourceHandle resourcesHandle;
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
        bool running = false;
        pipeline::Acquisition daq_read_pipe; // source is a daqreader 
        TaskHandle taskHandle;
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
        bool running = false;
        pipeline::Control cmd_write_pipe;
        pipeline::Acquisition state_write_pipe;
        TaskHandle taskHandle;
        synnax::Task task;
        std::shared_ptr<task::Context> ctx;
        bool ok_state = true;
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
}



