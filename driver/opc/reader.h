// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "opc.h"
#include "util.h"
#include "x/cpp/config/config.h"
#include "driver/task/task.h"
#include "driver/pipeline/acquisition.h"
#include "x/cpp/loop/loop.h"

namespace opc {
///////////////////////////////////////////////////////////////////////////////////
//                              ReaderChannelConfig                              //
///////////////////////////////////////////////////////////////////////////////////
struct ReaderChannelConfig {
    /// @brief the node id.
    std::string node_id;
    UA_NodeId node;
    /// @brief the corresponding channel key to write the variable for the node from.
    ChannelKey channel;
    /// @brief the channel fetched from the Synnax server. This does not need to
    /// be provided via the JSON configuration.
    Channel ch;
    bool enabled;

    ReaderChannelConfig() = default;

    explicit ReaderChannelConfig(
        config::Parser &parser
    ) : node_id(parser.required<std::string>("node_id")),
        node(parse_node_id("node_id", parser)),
        channel(parser.required<ChannelKey>("channel")),
        enabled(parser.optional<bool>("enabled", true)) {
    }
};

///////////////////////////////////////////////////////////////////////////////////
//                                 ReaderConfig                                  //
///////////////////////////////////////////////////////////////////////////////////
struct ReaderConfig {
    /// @brief the device representing the OPC UA server to read from.
    std::string device;
    /// @brief sets the acquisition rate.
    telem::Rate sample_rate;
    /// @brief sets the stream rate.
    telem::Rate stream_rate;
    /// @brief array_size;
    size_t array_size;
    /// @brief whether to enable data saving for this task.
    bool data_saving;

    /// @brief the list of channels to read from the server.
    std::vector<ReaderChannelConfig> channels;

    ReaderConfig() = default;

    explicit ReaderConfig(config::Parser &parser);

    [[nodiscard]] std::vector<ChannelKey> channel_keys() const {
        auto keys = std::vector<ChannelKey>(channels.size());
        for (std::size_t i = 0; i < channels.size(); i++) keys[i] = channels[i].channel;
        return keys;
    }
};

///////////////////////////////////////////////////////////////////////////////////
//                                    Reader Source                              //
///////////////////////////////////////////////////////////////////////////////////
class ReaderSource final : public pipeline::Source {
public:
    ReaderConfig cfg;
    std::shared_ptr<UA_Client> client;
    std::set<ChannelKey> indexes;
    std::shared_ptr<task::Context> ctx;
    synnax::Task task;

    UA_ReadRequest req;
    std::vector<UA_ReadValueId> readValueIds;
    loop::Timer timer;
    synnax::Frame fr;
    std::unique_ptr<int64_t[]> timestamp_buf;
    int exceed_time_count = 0;
    task::State curr_state;

    ReaderSource(
        ReaderConfig cfg,
        const std::shared_ptr<UA_Client> &client,
        std::set<ChannelKey> indexes,
        std::shared_ptr<task::Context> ctx,
        synnax::Task task
    );

    void initialize_read_request();

    void stopped_with_err(const xerrors::Error &err) override;

    [[nodiscard]] xerrors::Error communicate_value_error(
        const std::string &channel,
        const UA_StatusCode &status
    ) const;

    size_t cap_array_length(
        const size_t i,
        const size_t length
    );

    size_t write_to_series(
        const UA_Variant *val,
        const size_t i,
        telem::Series &s
    );

    std::pair<Frame, xerrors::Error> read(breaker::Breaker &breaker) override;
};

///////////////////////////////////////////////////////////////////////////////////
//                                    Reader Task                                //
///////////////////////////////////////////////////////////////////////////////////
/// @brief a task that reads values from an OPC UA server.
class ReaderTask final : public task::Task {
public:
    explicit ReaderTask(
        const std::shared_ptr<task::Context> &ctx,
        synnax::Task task,
        ReaderConfig cfg,
        const breaker::Config &breaker_config,
        std::shared_ptr<pipeline::Source> source,
        synnax::WriterConfig writer_config,
        std::shared_ptr<UA_Client> ua_client,
        opc::DeviceProperties device_props
    ) : ctx(ctx),
        task(std::move(task)),
        cfg(std::move(cfg)),
        breaker(breaker::Breaker(breaker_config)),
        pipe(pipeline::Acquisition(
            ctx->client,
            std::move(writer_config),
            std::move(source),
            breaker_config
        )),
        ua_client(ua_client),
        device_props(device_props) {
    }

    std::string name() override { return task.name; }

    static std::unique_ptr<task::Task> configure(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    );

    void exec(task::Command &cmd) override;

    void stop() override;

    void stop(const std::string &cmd_key);

    void start();

    void start(const std::string &cmd_key);

private:
    std::shared_ptr<task::Context> ctx;
    synnax::Task task;
    ReaderConfig cfg;
    breaker::Breaker breaker;
    pipeline::Acquisition pipe;
    std::shared_ptr<UA_Client> ua_client;
    opc::DeviceProperties device_props;
};
}
