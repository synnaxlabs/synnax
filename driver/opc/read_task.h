// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// external
#include "open62541/client_highlevel.h"
#include "open62541/types.h"
#include "open62541/common.h"

/// module
#include "x/cpp/xjson/xjson.h"
#include "x/cpp/loop/loop.h"

/// internal
#include "driver/task/task.h"
#include "driver/pipeline/acquisition.h"
#include "driver/task/common/read_task.h"
#include "driver/opc/util.h"

namespace opc {
struct InputChan {
    const bool enabled;
    /// @brief the OPC UA node id.
    const UA_NodeId node;
    /// @brief the corresponding channel key to write the variable for the node from.
    const ChannelKey synnax_key;
    /// @brief the channel fetched from the Synnax server. This does not need to
    /// be provided via the JSON configuration.
    Channel ch;

    explicit InputChan(
        xjson::Parser &parser
    ) : enabled(parser.optional<bool>("enabled", true)),
        node(parse_node_id("node_id", parser)),
        synnax_key(parser.required<ChannelKey>("channel")) {
    }
};

///////////////////////////////////////////////////////////////////////////////////
//                                 ReaderConfig                                  //
///////////////////////////////////////////////////////////////////////////////////
struct ReadTaskConfig {
    /// @brief whether to enable data saving for this task.
    const bool data_saving;
    /// @brief the device representing the OPC UA server to read from.
    const std::string device_key;
    /// @brief sets the acquisition rate.
    const telem::Rate sample_rate;
    /// @brief sets the stream rate.
    const telem::Rate stream_rate;
    /// @brief array_size;
    const size_t array_size;
    /// @brief the config for connecting to the OPC UA server.
    ConnectionConfig conn;
    std::set<synnax::ChannelKey> index_keys;
    /// @brief the list of channels to read from the server.
    std::vector<InputChan> channels;

    explicit ReadTaskConfig(
        const std::shared_ptr<synnax::Synnax> &client,
        xjson::Parser &parser
    ):
        data_saving(parser.optional<bool>("data_saving", true)),
        device_key(parser.required<std::string>("device")),
        sample_rate(parser.required<float>("sample_rate")),
        stream_rate(parser.required<float>("stream_rate")),
        array_size(parser.optional<std::size_t>("array_size", 1)) {
        parser.iter("channels", [&](xjson::Parser &channel_builder) {
            const auto ch = InputChan(channel_builder);
            if (ch.enabled) channels.push_back(ch);
        });
        auto [dev, err] = client->hardware.retrieve_device(this->device_key);
        if (err) {
            parser.field_err("device", "failed to retrieve device: " + err.message());
            return;
        }
        const auto properties = xjson::Parser(dev.properties);
        this->conn = ConnectionConfig(properties.child("connection"));
        if (properties.error())
            parser.
                    field_err("device", properties.error().message());
        std::vector<synnax::ChannelKey> keys;
        keys.reserve(this->channels.size());
        for (const auto &ch: this->channels) keys.push_back(ch.synnax_key);
        auto [sy_channels, ch_err] = client->channels.retrieve(keys);
        if (ch_err) {
            parser.field_err("channels",
                             "failed to retrieve channels: " + ch_err.message());
            return;
        }
        for (std::size_t i = 0; i < sy_channels.size(); i++) {
            auto ch = sy_channels[i];
            if (ch.index != 0) this->index_keys.insert(ch.index);
            this->channels[i].ch = ch;
        }
    }

    [[nodiscard]] synnax::WriterConfig writer_config() const {
        std::vector<synnax::ChannelKey> channel_keys;
        channel_keys.reserve(this->channels.size());
        for (const auto &ch: this->channels)
            channel_keys.push_back(ch.synnax_key);
        return {
            .channels = channel_keys,
            .mode = synnax::data_saving_writer_mode(this->data_saving),
            .enable_auto_commit = true
        };
    }
};

struct ReadRequest {
    UA_ReadRequest req;
    std::vector<UA_ReadValueId> read_value_ids;

    explicit ReadRequest(const ReadTaskConfig &cfg) {
        UA_ReadRequest_init(&this->req);
        read_value_ids.reserve(cfg.channels.size());
        for (const auto &ch: cfg.channels) {
            if (!ch.enabled) continue;
            UA_ReadValueId rvid;
            UA_ReadValueId_init(&rvid);
            rvid.nodeId = ch.node;
            rvid.attributeId = UA_ATTRIBUTEID_VALUE;
            read_value_ids.push_back(rvid);
        }
        req.nodesToRead = read_value_ids.data();
        req.nodesToReadSize = read_value_ids.size();
    }
};

struct ReadResponse {
    UA_ReadResponse resp;

    explicit ReadResponse(const UA_ReadResponse &resp) : resp(resp) {
    }

    ~ReadResponse() {
        UA_ReadResponse_clear(&resp);
    }
};



class BaseReadTaskSource: public common::Source {
protected:
    ReadTaskConfig cfg;
    std::shared_ptr<UA_Client> client;
    ReadRequest request;
    loop::Timer timer;

    BaseReadTaskSource(
        const std::shared_ptr<UA_Client> &client,
        ReadTaskConfig cfg,
        const telem::Rate rate
    ): cfg(std::move(cfg)),
       client(client),
       request(cfg),
       timer(rate) {
    }

    synnax::WriterConfig writer_config() const override {
        return this->cfg.writer_config();
    }
};

class ArrayReadTaskSource final : public BaseReadTaskSource {
    ArrayReadTaskSource(
        const std::shared_ptr<UA_Client> &client,
        const ReadTaskConfig &cfg
    ): BaseReadTaskSource(client, cfg, cfg.sample_rate / cfg.array_size) {
    }

    std::pair<Frame, xerrors::Error> read(breaker::Breaker &breaker) override {
        timer.wait(breaker);
        UA_ReadResponse res = UA_Client_Service_read(client.get(), this->request.req);
        auto fr = Frame(cfg.channels.size() + cfg.index_keys.size());
        for (std::size_t i = 0; i < res.resultsSize; ++i) {
            if (const auto err = parse_error(res.results[i].status))
                return {Frame(), err};
            UA_Variant *value = &res.results[i].value;
            const auto &ch = cfg.channels[i];
            fr.emplace(ch.synnax_key, val_to_series(value, ch.ch.data_type));
        }
        if (!cfg.index_keys.empty()) {
            auto start = telem::TimeStamp::now();
            auto end = start + cfg.array_size * cfg.sample_rate.period();
            auto s = telem::Series::linspace(start, end, cfg.array_size);
            for (const auto &idx: cfg.index_keys) fr.emplace(idx, s.deep_copy());
        }
        return {std::move(fr), xerrors::NIL};
    }
};

class UnaryReadTaskSource final : public BaseReadTaskSource {
public:
    UnaryReadTaskSource(
        const std::shared_ptr<UA_Client> &client,
        const ReadTaskConfig &cfg
    ): BaseReadTaskSource(client, cfg, cfg.sample_rate) {
    }

    std::pair<Frame, xerrors::Error> read(breaker::Breaker &breaker) override {
        auto fr = Frame(cfg.channels.size() + cfg.index_keys.size());
        const auto samples_per_chan = cfg.sample_rate / cfg.stream_rate;
        for (const auto &ch: cfg.channels)
            fr.emplace(ch.synnax_key, telem::Series(ch.ch.data_type, samples_per_chan));
        for (const auto &idx: cfg.index_keys)
            fr.emplace(idx, telem::Series(telem::TIMESTAMP_T, samples_per_chan));
        for (std::size_t i = 0; i < samples_per_chan; i++) {
            UA_ReadResponse res = UA_Client_Service_read(client.get(), this->request.req);
            if (const auto err = parse_error(res.responseHeader.serviceResult))
                return {std::move(fr), err};
            for (std::size_t j = 0; j < res.resultsSize; ++j) {
                auto result = res.results[j];
                if (const auto err = parse_error(result.status))
                    return {std::move(fr), err};
                write_to_series(result.value, fr.series->at(j));
            }
            const auto now = telem::TimeStamp::now();
            for (std::size_t j = cfg.channels.size(); j < fr.size(); j++)
                fr.series->at(j).write(now);
            timer.wait(breaker);
        }
        return std::make_pair(std::move(fr), xerrors::NIL);
    }
};
}


size_t opc::ReaderSource::cap_array_length(
    const size_t i,
    const size_t length
) {
    if (i + length > cfg.array_size) {
        if (curr_state.variant != "warning") {
            curr_state.variant = "warning";
            curr_state.details = json{
                {
                    "message",
                    "Received array of length " + std::to_string(length) +
                    " from OPC UA server, which is larger than the configured size of "
                    + std::to_string(cfg.array_size) + ". Truncating array."
                },
                {"running", true}
            };
            ctx->set_state(curr_state);
        }
        return cfg.array_size - i;
    }
    return length;
}

