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
struct WriterChannelConfig {
    /// @brief the node id.
    std::string node_id;
    UA_NodeId node;
    /// @brief the corresponding channel key to write the variable for the node from.
    ChannelKey cmd_channel;
    bool enabled;
    /// @brief the channel fetched from the Synnax server. This does not need to
    /// be provided via the JSON configuration.
    Channel ch;

    WriterChannelConfig() = default;

    explicit WriterChannelConfig(
        config::Parser &parser
    ) : node_id(parser.required<std::string>("node_id")),
        node(parse_node_id("node_id", parser)),
        cmd_channel(parser.required<ChannelKey>("channel")),
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
    /// @brief index key for all state channels in this task
    synnax::ChannelKey state_index_key;

    WriterConfig() = default;

    explicit WriterConfig(config::Parser &parser);

    [[nodiscard]] std::vector<ChannelKey> cmd_keys() const {
        std::vector<ChannelKey> keys;
        for (const auto &channel: channels) keys.push_back(channel.cmd_channel);
        return keys;
    }
}; // struct WriterConfig

///////////////////////////////////////////////////////////////////////////////////
//                                    OPC Sink                                   //
///////////////////////////////////////////////////////////////////////////////////
/// @brief an OPC writer with embedded OPC UA client that receives data from synnax
/// in frames, and writes them to the appropriate nodes on the connected OPC UA Server.
class WriterSink final : public pipeline::Sink {
public:
    // Synnax Resources
    std::shared_ptr<task::Context> ctx;
    synnax::Task task;
    std::map<ChannelKey, WriterChannelConfig> cmd_channel_map; // TODO: Change to cmd channel map
    task::State curr_state;

    // OPC UA Resources
    WriterConfig cfg;
    std::shared_ptr<UA_Client> ua_client;
    UA_WriteRequest req; // defined in types_generated.h
    opc::DeviceProperties device_props;

    // keep alive resources (thread and mutex)
    std::thread keep_alive_thread;
    std::mutex client_mutex;
    breaker::Breaker breaker;
    ///@brief the rate at which sink will ping the OPC UA server to maintain the connection
    synnax::Rate ping_rate = synnax::Rate(0.1); // default to every 10s

    WriterSink(
        WriterConfig cfg,
        const std::shared_ptr<UA_Client> &ua_client,
        const std::shared_ptr<task::Context> &ctx,
        synnax::Task task,
        opc::DeviceProperties device_props
    );

    ~WriterSink() override {
        this->breaker.stop();
        this->keep_alive_thread.join();
    }

    freighter::Error write(const synnax::Frame &frame) override;

    void maintain_connection();

private:
    void stopped_with_err(const freighter::Error &err) override;

    static void set_variant(
        UA_Variant *val,
        const synnax::Frame &frame,
        const uint32_t &series_index,
        const synnax::DataType &type
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
    ) : ctx(ctx),
        task(std::move(task)),
        cfg(std::move(cfg)),
        breaker_cfg(breaker_cfg),
        cmd_pipe(pipeline::Control(
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
    pipeline::Control cmd_pipe;

    // Channel Information
    std::vector<synnax::ChannelKey> cmd_channels_keys;

    // OPC UA
    std::shared_ptr<UA_Client> ua_client;
    opc::DeviceProperties device_props;
}; // class WriterTask
} // namespace opc
