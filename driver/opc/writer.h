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
        ChannelKey channel;
        /// @brief the channel fetched from the Synnax server. This does not need to
        /// be provided via the JSON configuration.
        Channel ch;
        bool enabled;
        // TODO: might have to store the data type it is on the OPC server

        WriterChannelConfig() = default;

        explicit WriterChannelConfig(
            config::Parser &parser
        ) : node_id(parser.required<std::string>("node_id")),
            node(parseNodeId("node_id", parser)),
            channel(parser.required<ChannelKey>("channel")),
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


        WriterConfig() = default;

        explicit WriterConfig(config::Parser &parser);

        [[nodiscard]] std::vector<ChannelKey> channelKeys() const {
            std::vector<ChannelKey> keys;
            for (const auto &channel : channels) keys.push_back(channel.channel);
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

    }

    ///////////////////////////////////////////////////////////////////////////////////
    //                                    OPC Sink                                   //
    ///////////////////////////////////////////////////////////////////////////////////
    /// @brief Sink is an OPC writer with an embedded OPC UA client that receives data from
    /// synnax in frames, and writes them to the appropriate nodes in the connected OPC UA
    // Server.
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
                synnax::StreamerConfig streamer_config,
                std::shared_ptr<UA_Client> ua_client,
                opc::DeviceProperties device_props
        ): ctx(ctx),
           task(std::move(task)),
           cfg(std::move(cfg)),
           breaker_cfg(breaker_cfg),
           pipe(pipeline::Control(
                   ctx->client,
                   std::move(streamer_config),
                   std::move(sink),
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
        pipeline::Control pipe;
        // Channel Information
        std::vector<synnax::ChannelKey> state_channel_keys;
        std::vector<synnax::ChannelKey> cmd_channels_keys;
        synnax::ChannelKey state_index_key;
        // OPC UA
        std::shared_ptr<UA_Client> ua_client;
        opc::DeviceProperties device_props;

    }; // class WriterTask
} // namespace opc