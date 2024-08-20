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

        WriterChannelConfig() = default;

        explicit WriterChannelConfig(
            config::Parser &parser
        ) : node_id(parser.required<std::string>("node_id")),
            node(parseNodeId("node_id", parser)),
            channel(parser.required<ChannelKey>("channel")),
            enabled(parser.optional<bool>("enabled", true)) {
        }
    };
    struct WriterConfig {
        /// @brief the device representing the OPC UA server to read from.
        std::string device;
        /// @brief sets the acquisition rate.
        Rate sample_rate;
        /// @brief sets the stream rate.
        Rate stream_rate;
        /// @brief array_size;
        size_t array_size;
        /// @brief whether to enable data saving for this task.
        bool data_saving;

        /// @brief the list of channels to read from the server.

        std::vector<WriterChannelConfig> channels;

        WriterConfig() = default;

        explicit WriterConfig(config::Parser &parser);

        [[nodiscard]] std::vector<ChannelKey> channelKeys() const {
            std::vector<ChannelKey> keys;
            for (const auto &channel : channels) keys.push_back(channel.channel);
            return keys;
        }
    }

    ///////////////////////////////////////////////////////////////////////////////////
    //                                    writer task                                //
    ///////////////////////////////////////////////////////////////////////////////////
    class WriterTask : public task::Task {
    public:
        explicit WriterTask(
            const std::shared_ptr<task::Context> &ctx,
            synnax::Task task,
            WriterConfig cfg,
            const breaker::Config &breaker_cfg,
            std::shared_ptr<pipeline::Sink> sink,
            synnax::WriterConfig writer_cfg,
            std::shared_ptr<UA_Client> ua_client,
            opc::DeviceProperties device_props
        ): ctx(ctx),
           task(std::move(task)),
           cfg(std::move(cfg)),
           breaker_cfg(breaker_cfg),
           sink(std::move(sink)),
           writer_cfg(std::move(writer_cfg)),
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
        std::shared_ptr<task::Context> ctx;
        synnax::Task task;
        WriterConfig cfg;
        breaker::Config breaker_cfg;
        pipeline::Control ctrl pipe;
        std::shared_ptr<UA_Client> ua_client;
        opc::DeviceProperties device_props;
    }; // class WriterTask

    ///////////////////////////////////////////////////////////////////////////////////
    //                                    OPC Sink                                   //
    ///////////////////////////////////////////////////////////////////////////////////

    class Sink : public pipeline::Sink {
        public: 
            WriterConfig cfg;
            std::shared_ptr<UA_Client> ua_client;
            std::set<ChannelKey> indexes;
            std::shared_ptr<task::Context> ctx;
            synnax::Task task;
            std::map<ChannelKey, WriterChannelConfig> channel_map;

            UA_WriteRequest req; // defined in types_generated.h
            std::vector<UA_WriteValue> nodes_to_write;
            synnax::Frame fr;
            std::unique_ptr<int64_t[]> timestamp_buf;
            int exceed_time_count = 0;
            task::State curr_state;

            Sink(
                WriterConfig cfg,
                const std::shared_ptr<UA_Client> &ua_client,
                std::set<ChannelKey> indexes,
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
                UA_WriteValue &writeValue
            );

            void stopped_with_err(const freighter::Error &err) override;
   
            [[nodiscard]] freighter::Error communicate_response_error(const UA_StatusCode &status);

            void cast_and_set_type(
                const synnax::Frame &frame,
                const uint32_t &series_index,
                const WriterChannelConfig &ch,
                UA_WriteValue &write_value
            );
    }; // class Sink
} // namespace opc