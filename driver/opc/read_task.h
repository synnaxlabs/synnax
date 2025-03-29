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
#include "driver/pipeline/acquisition.h"
#include "driver/task/common/read_task.h"
#include "driver/opc/util/util.h"
#include "driver/task/common/sample_clock.h"
#include "x/cpp/defer/defer.h"

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
        node(util::parse_node_id("node_id", parser)),
        synnax_key(parser.required<ChannelKey>("channel")) {
    }
};

struct ReadTaskConfig : public common::BaseReadTaskConfig {
    /// @brief the device representing the OPC UA server to read from.
    const std::string device_key;
    /// @brief array_size;
    const size_t array_size;
    /// @brief the config for connecting to the OPC UA server.
    util::ConnectionConfig conn;
    /// @brief keys of the index channels for the input channels.
    std::set<synnax::ChannelKey> index_keys;
    /// @brief the list of channels to read from the server.
    std::vector<InputChan> channels;
    /// @brief the number of samples to read on each iteration.
    const size_t samples_per_chan;

    /// @brief Move constructor to allow transfer of ownership
    ReadTaskConfig(ReadTaskConfig &&other) noexcept:
        common::BaseReadTaskConfig(std::move(other)),
        device_key(other.device_key),
        array_size(other.array_size),
        conn(std::move(other.conn)),
        index_keys(std::move(other.index_keys)),
        channels(std::move(other.channels)),
        samples_per_chan(other.samples_per_chan) {
    }

    /// @brief delete copy constructor and copy assignment to prevent accidental copies.
    ReadTaskConfig(const ReadTaskConfig &) = delete;

    const ReadTaskConfig &operator=(const ReadTaskConfig &) = delete;

    explicit ReadTaskConfig(
        const std::shared_ptr<synnax::Synnax> &client,
        xjson::Parser &parser
    ): common::BaseReadTaskConfig(
           parser,
           common::TimingConfig(),
           parser.optional("array_size", 1) <= 1
       ),
       device_key(parser.required<std::string>("device")),
       array_size(parser.optional<std::size_t>("array_size", 1)),
       samples_per_chan(this->sample_rate / this->stream_rate) {
        parser.iter("channels", [&](xjson::Parser &cp) {
            const auto ch = InputChan(cp);
            if (ch.enabled) channels.push_back(ch);
        });
        if (this->channels.empty()) {
            parser.field_err("channels", "task must have at least one enabled channel");
            return;
        }
        auto [dev, err] = client->hardware.retrieve_device(this->device_key);
        if (err) {
            parser.field_err("device", "failed to retrieve device: " + err.message());
            return;
        }
        const auto properties = xjson::Parser(dev.properties);
        this->conn = util::ConnectionConfig(properties.child("connection"));
        if (properties.error()) {
            parser.field_err("device", properties.error().message());
            return;
        }
        std::vector<synnax::ChannelKey> keys;
        keys.reserve(this->channels.size());
        for (const auto &ch: this->channels) keys.push_back(ch.synnax_key);
        auto [sy_channels, ch_err] = client->channels.retrieve(keys);
        if (ch_err) {
            parser.field_err(
                "channels",
                "failed to retrieve channels: " + ch_err.message()
            );
            return;
        }
        for (std::size_t i = 0; i < sy_channels.size(); i++) {
            auto ch = sy_channels[i];
            if (ch.index != 0) this->index_keys.insert(ch.index);
            this->channels[i].ch = ch;
        }
        for (std::size_t i = 0; i < sy_channels.size(); i++) {
            auto ch = sy_channels[i];
            if (ch.is_index && this->index_keys.find(ch.key) != this->index_keys.end())
                this->index_keys.erase(ch.key);
        }
    }

    std::vector<synnax::Channel> sy_channels() const {
        std::vector<synnax::Channel> chs;
        chs.reserve(this->channels.size());
        for (const auto &ch: this->channels) chs.push_back(ch.ch);
        return chs;
    }

    [[nodiscard]] synnax::WriterConfig writer_config() const {
        std::vector<synnax::ChannelKey> channel_keys;
        channel_keys.reserve(this->channels.size() + this->index_keys.size());
        for (const auto &ch: this->channels)
            channel_keys.push_back(ch.synnax_key);
        for (const auto &idx: this->index_keys) channel_keys.push_back(idx);
        return {
            .channels = channel_keys,
            .mode = synnax::data_saving_writer_mode(this->data_saving),
            .enable_auto_commit = true
        };
    }

    static std::pair<ReadTaskConfig, xerrors::Error> parse(
        const std::shared_ptr<synnax::Synnax> &client,
        const synnax::Task &task
    ) {
        auto parser = xjson::Parser(task.config);
        return {ReadTaskConfig(client, parser), parser.error()};
    }
};

struct ReadRequest {
    UA_ReadRequest base;
    std::vector<UA_ReadValueId> read_value_ids;

    explicit ReadRequest(const ReadTaskConfig &cfg) {
        UA_ReadRequest_init(&this->base);
        read_value_ids.reserve(cfg.channels.size());
        for (const auto &ch: cfg.channels) {
            if (!ch.enabled) continue;
            UA_ReadValueId rvid;
            UA_ReadValueId_init(&rvid);
            rvid.nodeId = ch.node;
            rvid.attributeId = UA_ATTRIBUTEID_VALUE;
            read_value_ids.push_back(rvid);
        }
        base.nodesToRead = read_value_ids.data();
        base.nodesToReadSize = read_value_ids.size();
    }
};

class BaseReadTaskSource : public common::Source {
protected:
    const ReadTaskConfig cfg;
    std::shared_ptr<UA_Client> client;
    ReadRequest request;
    loop::Timer timer;

    BaseReadTaskSource(
        const std::shared_ptr<UA_Client> &client,
        ReadTaskConfig cfg,
        const telem::Rate rate
    ): cfg(std::move(cfg)),
       client(client),
       request(this->cfg),
       timer(rate) {
    }

    synnax::WriterConfig writer_config() const override {
        return this->cfg.writer_config();
    }
};

class ArrayReadTaskSource final : public BaseReadTaskSource {
public:
    ArrayReadTaskSource(
        const std::shared_ptr<UA_Client> &client,
        ReadTaskConfig cfg
    ): BaseReadTaskSource(client, std::move(cfg), cfg.sample_rate / cfg.array_size) {
    }

    std::vector<synnax::Channel> channels() const override {
        return this->cfg.sy_channels();
    }

    std::pair<Frame, xerrors::Error> read(breaker::Breaker &breaker) override {
        this->timer.wait(breaker);
        UA_ReadResponse res = UA_Client_Service_read(
            this->client.get(), this->request.base);
        x::defer clear_res([&res] { UA_ReadResponse_clear(&res); });
        auto fr = Frame(this->cfg.channels.size() + this->cfg.index_keys.size());
        for (std::size_t i = 0; i < res.resultsSize; ++i) {
            auto &result = res.results[i];
            if (const auto err = util::parse_error(result.status))
                return {std::move(fr), err};
            const auto &ch = cfg.channels[i];
            auto [s, err] = util::ua_array_to_series(
                ch.ch.data_type,
                &result.value,
                this->cfg.array_size,
                ch.ch.name
            );
            if (err) return {std::move(fr), err};
            fr.emplace(ch.synnax_key, std::move(s));
        }
        auto start = telem::TimeStamp::now();
        auto end = start + this->cfg.array_size * this->cfg.sample_rate.period();
        common::generate_index_data(
            fr,
            this->cfg.index_keys,
            start,
            end,
            this->cfg.array_size,
            true
        );
        return {std::move(fr), xerrors::NIL};
    }
};

class UnaryReadTaskSource final : public BaseReadTaskSource {
public:
    UnaryReadTaskSource(
        const std::shared_ptr<UA_Client> &client,
        ReadTaskConfig cfg
    ): BaseReadTaskSource(client, std::move(cfg), cfg.sample_rate) {
    }

    std::pair<Frame, xerrors::Error> read(breaker::Breaker &breaker) override {
        auto fr = Frame(cfg.channels.size() + cfg.index_keys.size());
        for (const auto &ch: cfg.channels)
            fr.emplace(
                ch.synnax_key,
                telem::Series(ch.ch.data_type, this->cfg.samples_per_chan)
            );
        for (const auto &idx: cfg.index_keys)
            fr.emplace(
                idx,
                telem::Series(telem::TIMESTAMP_T, this->cfg.samples_per_chan)
            );

        for (std::size_t i = 0; i < this->cfg.samples_per_chan; i++) {
            const auto start = telem::TimeStamp::now();
            UA_ReadResponse res = UA_Client_Service_read(
                this->client.get(),
                this->request.base
            );
            x::defer clear_res([&res] { UA_ReadResponse_clear(&res); });
            if (const auto err = util::parse_error(res.responseHeader.serviceResult))
                return {std::move(fr), err};
            for (std::size_t j = 0; j < res.resultsSize; ++j) {
                UA_DataValue &result = res.results[j];
                if (const auto err = util::parse_error(result.status))
                    return {std::move(fr), err};
                util::write_to_series(fr.series->at(j), result.value);
            }
            const auto end = telem::TimeStamp::now();
            const auto ts = telem::TimeStamp::midpoint(start, end);
            for (std::size_t j = this->cfg.channels.size(); j < fr.size(); j++)
                fr.series->at(j).write(ts);
            this->timer.wait(breaker);
        }
        return std::make_pair(std::move(fr), xerrors::NIL);
    }

    std::vector<synnax::Channel> channels() const override {
        return this->cfg.sy_channels();
    }
};
}
