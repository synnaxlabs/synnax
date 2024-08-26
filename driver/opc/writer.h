// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once
#include <map>

#include "opc.h"
#include "util.h"
#include "driver/config/config.h"
#include "driver/task/task.h"
#include "driver/pipeline/control.h"
#include "driver/loop/loop.h"

#include "include/open62541/types.h"
#include "include/open62541/types_generated.h"
#include "include/open62541/statuscodes.h"
#include "include/open62541/client_highlevel.h"
#include "include/open62541/common.h"


namespace opc {
    ///////////////////////////////////////////////////////////////////////////////////
    //                             writer channel config                             //
    ///////////////////////////////////////////////////////////////////////////////////
    struct WriterChannelConfig{
        /// @brief the node id.
        std::string node_id;
        UA_NodeId node;
        /// @brief the corresponding channel key to write the variable for the node from.
        ChannelKey cmd_channel;
        ChannelKey state_channel;
        bool enabled;
        /// @brief the channel fetched from the Synnax server. This does not need to
        /// be provided via the JSON configuration.
        Channel ch;

        WriterChannelConfig() = default;

        explicit WriterChannelConfig(
            config::Parser &parser
        ) : node_id(parser.required<std::string>("node_id")),
            node(parseNodeId("node_id", parser)),
            cmd_channel(parser.required<ChannelKey>("cmd_channel")),
            state_channel(parser.required<ChannelKey>("state_channel")
            enabled(parser.optional<bool>("enabled", true)) {
        }
    }; // struct WriterChannelConfig

    ///////////////////////////////////////////////////////////////////////////////////
    //                                    writer config                              //
    ///////////////////////////////////////////////////////////////////////////////////
    struct WriterConfig {
        /// @brief the device representing the OPC UA server to read from.
        std::string device;
        /// @brief the list of channels to read from the server.
        std::vector<WriterChannelConfig> channels;
        /// @brief frequency state of a controlled channel is published
        synnax::Rate state_rate = synnax::Rate(1); // default to 1 Hz

        synnax::ChannelKey state_index_key;

        WriterConfig() = default;

        explicit WriterConfig(config::Parser &parser);

        [[nodiscard]] std::vector<ChannelKey> cmd_keys() const {
            std::vector<ChannelKey> keys;
            for (const auto &channel : channels) keys.push_back(channel.cmd_channel);
            return keys;
        }

        [[nodiscard]] std::vector<ChannelKey> state_keys() const {
            std::vector<ChannelKey> keys;
            for (const auto &channel : channels) keys.push_back(channel.state_channel);
            return keys;
        }
    }; // struct WriterConfig

    ///////////////////////////////////////////////////////////////////////////////////
    //                                   State Source                                //
    ///////////////////////////////////////////////////////////////////////////////////
    /// @brief StateSource is an OPC subscriber which listens for updates to the states of
    /// control channels and writes them to synnax as a source passed into an acquisition
    /// pipeline of a task.
    class StateSource final : public pipeline::Source {
    public:
        explicit StateSource(
            synnax::Rate state_rate,
            const std::shared_ptr<UA_Client> &ua_client,
            const std::shared_ptr<task::Context> &ctx,
            const WriterConfig &cfg
        );

        std::pair<synnax::Frame, freighter::Error> read(breaker::Breaker &breaker) override;

        synnax::Frame get_state();

        void update_state(const synnax::ChannelKey &channel_key, const UA_Variant &value);

        ///@brief registers a subscription to a node on the OPC UA server
        UA_StatusCode add_monitored_item(const UA_NodeId& node_id, const synnax::ChannelKey& channel_key);

        ///@brief static function to pass in to client subscriber when data changes
        static void data_change_handler(
            UA_Client *client,
            UA_UInt32 subId,
            void *subContext,
            UA_UInt32 monId,
            void *monContext,
            UA_DataValue *value
        );

        struct MonitoredItemContext{
            opc::StateSource *source;
            synnax::ChannelKey channelKey;
        };

    private:
        synnax::Rate state_rate;
        loop::Timer timer;
        std::shared_ptr<UA_Client> ua_client;
        std::shared_ptr<task::Context> ctx;
        WriterConfig cfg; // TODO: shared pointer?
        std::mutex state_mutex;
        std::condition_variable waiting_reader;
        ///@brief maps channel to the last read value from  corresponding OPC UA Node
        std::map<synnax::ChannelKey, UA_Variant> state_map;
        ///@brief map of channel keys to the corresponding channel on synnax server
        std::map<synnax::ChannelKey, synnax::Channel> state_channels;
        synnax::ChannelKey state_index_key;
        ///@brief thread listening for updates on the OPC UA server
        std::unique_ptr<std::thread> subscriber_thread;
        ///@brief subscription id for the OPC UA server
        UA_UInt32 subscription_id;
    }; // class StateSource

    ///////////////////////////////////////////////////////////////////////////////////
    //                                    OPC Sink                                   //
    ///////////////////////////////////////////////////////////////////////////////////
    /// @brief an OPC writer with embedded OPC UA client that receives data from synnax
    /// in frames, and writes them to the appropriate nodes on the connected OPC UA Server.
    class Sink final : public pipeline::Sink {
        public:
            // Synnax Resources
            std::shared_ptr<task::Context> ctx;
            synnax::Task task;
            std::map<ChannelKey, WriterChannelConfig> cmd_channel_map; // TODO: Change to cmd channel map
            synnax::Frame fr; // TODO: does this need to be a member variable?
            task::State curr_state;

            // OPC UA Resources
            WriterConfig cfg;
            std::shared_ptr<UA_Client> ua_client;
            UA_WriteRequest req; // defined in types_generated.h

            Sink(
                WriterConfig cfg,
                const std::shared_ptr<UA_Client> &ua_client,
                const std::shared_ptr<task::Context> &ctx,
                synnax::Task task
            );

            freighter::Error write(synnax::Frame frame) override;
       
        private:
            void initialize_write_request(const synnax::Frame &frame);

            void initialize_write_value(
                const synnax::Frame &frame,
                uint32_t &index,
                WriterChannelConfig &ch,
                UA_WriteValue *write_value
            );

            void stoppedWithErr(const freighter::Error &err) override;
   
            [[nodiscard]] freighter::Error communicate_response_error(const UA_StatusCode &status);

            void cast_and_set_type(
                const synnax::Frame &frame,
                const uint32_t &series_index,
                const WriterChannelConfig &ch,
                UA_WriteValue *write_value
            );

    }; // class Sink

    ///////////////////////////////////////////////////////////////////////////////////
    //                                    writer task                                //
    ///////////////////////////////////////////////////////////////////////////////////
    ///@brief WriterTask is a is a user specified object which writes data to an OPC UA
    /// server and regularly maintains the state of those outputs.
    class WriterTask final : public task::Task {
    public:
        explicit WriterTask(
                const std::shared_ptr<task::Context> &ctx,
                synnax::Task task,
                WriterConfig cfg,
                const breaker::Config &breaker_cfg,
                std::shared_ptr<pipeline::Sink> sink,
                std::shared_ptr<pipeline::Source> state_source,
                synnax::StreamerConfig streamer_config,
                synnax::WriterConfig writer_config,
                std::shared_ptr<UA_Client> ua_client,
                opc::DeviceProperties device_props
        ): ctx(ctx),
           task(std::move(task)),
           cfg(std::move(cfg)),
           breaker_cfg(breaker_cfg),
           cmd_pipe(pipeline::Control(
                   ctx->client,
                   std::move(streamer_config),
                   std::move(sink),
                   breaker_cfg
           )),
           state_pipe(pipeline::Acquisition(
                   ctx->client,
                   std::move(writer_config),
                   std::move(state_source),
                   breaker_cfg
           )),
           ua_client(std::move(ua_client)),
           device_props(std::move(device_props)) {
        }

        static std::unique_ptr<task::Task> configure(
                const std::shared_ptr<task::Context> &ctx,
                const synnax::Task &task
        );
        void exec(task::Command &cmd) override;
        void stop() override;
        void start();
    private:
        opc::WriterConfig cfg;
        // Task Resources
        std::shared_ptr<task::Context> ctx;
        synnax::Task task;
        breaker::Config breaker_cfg;
        pipeline::Control cmd_pipe;
        pipeline::Acquisition state_pipe;
        // Channel Information
        std::vector<synnax::ChannelKey> state_channel_eys;
        std::vector<synnax::ChannelKey> cmd_channels_keys;
        synnax::ChannelKey state_index_key;
        // OPC UA
        std::shared_ptr<UA_Client> ua_client;
        opc::DeviceProperties device_props;
    }; // class WriterTask
} // namespace opc