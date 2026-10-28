// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <cstddef>
#include <memory>
#include <set>
#include <string>
#include <utility>
#include <vector>

#include "open62541/client_highlevel.h"
#include "open62541/common.h"
#include "open62541/types.h"

#include "client/cpp/opcua/json.gen.h"
#include "client/cpp/synnax.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/loop/loop.h"
#include "x/cpp/strings/strings.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/telem.h"

#include "driver/common/read_task.h"
#include "driver/common/sample_clock.h"
#include "driver/errors/errors.h"
#include "driver/opcua/connection/connection.h"
#include "driver/opcua/device/device.h"
#include "driver/opcua/errors/errors.h"
#include "driver/opcua/telem/telem.h"
#include "driver/opcua/types/types.h"

namespace driver::opcua {
struct InputChan {
    /// @brief the OPC UA node id.
    types::NodeId node;
    /// @brief the corresponding channel key to write the variable for the node
    /// from.
    const synnax::channel::Key synnax_key;
    /// @brief the channel fetched from the Synnax server. This does not need to
    /// be provided via the JSON configuration.
    synnax::channel::Channel ch;

    InputChan(const ::synnax::opcua::ReadChannel &parsed, x::json::Parser &parser):
        node(types::NodeId::parse("node_id", parser)), synnax_key(parsed.channel) {
        if (this->synnax_key == 0)
            parser.field_err("channel", "channel must be specified");
    }

    // Move constructor - needed because NodeId is move-only
    InputChan(InputChan &&other) noexcept:
        node(std::move(other.node)),
        synnax_key(other.synnax_key),
        ch(std::move(other.ch)) {}

    // Delete copy constructor - NodeId is move-only
    InputChan(const InputChan &) = delete;
    InputChan &operator=(const InputChan &) = delete;
    InputChan &operator=(InputChan &&) = delete;

    // No manual destructor needed - types::NodeId handles cleanup automatically
};

struct ReadTaskConfig : common::BaseReadTaskConfig {
    /// @brief the device representing the OPC UA server to read from.
    const std::string device_key;
    /// @brief whether each read returns an array of samples per node.
    const bool array_mode;
    /// @brief the number of samples in each array when array_mode is true.
    const size_t array_size;
    /// @brief keys of the index channels for the input channels.
    /// Dynamically populated by querying the core.
    std::set<synnax::channel::Key> index_keys;
    /// @brief the enabled channels to read from the server.
    std::vector<std::unique_ptr<InputChan>> channels;
    /// @brief the number of samples to read on each iteration.
    const size_t samples_per_chan;

    /// @brief Move constructor to allow transfer of ownership
    ReadTaskConfig(ReadTaskConfig &&other) noexcept:
        common::BaseReadTaskConfig(std::move(other)),
        device_key(other.device_key),
        array_mode(other.array_mode),
        array_size(other.array_size),
        index_keys(std::move(other.index_keys)),
        channels(std::move(other.channels)),
        samples_per_chan(other.samples_per_chan) {}

    /// @brief delete copy constructor and copy assignment to prevent accidental
    /// copies.
    ReadTaskConfig(const ReadTaskConfig &) = delete;

    const ReadTaskConfig &operator=(const ReadTaskConfig &) = delete;

    explicit ReadTaskConfig(
        const std::shared_ptr<synnax::Synnax> &client,
        x::json::Parser &parser
    ):
        common::BaseReadTaskConfig(
            parser,
            common::TimingConfig(),
            !parser.field<bool>("array_mode", false)
        ),
        device_key(parser.field<std::string>("device")),
        array_mode(parser.field<bool>("array_mode", false)),
        array_size(parser.field<std::size_t>("array_size", 1)),
        // Array mode never uses samples_per_chan and its stream rate may be zero.
        samples_per_chan(this->array_mode ? 0 : this->sample_rate / this->stream_rate) {
        parser.iter("channels", [&](x::json::Parser &cp) {
            const auto parsed = ::synnax::opcua::ReadChannel::parse(cp);
            if (parsed.disabled) return;
            channels.push_back(std::make_unique<InputChan>(parsed, cp));
        });
        if (this->channels.empty()) {
            parser.field_err("channels", "task must have at least one enabled channel");
            return;
        }
        if (const auto err = device::retrieve_connection(*client, this->device_key)
                                 .second) {
            parser.field_err("device", common::config_err_description(err));
            return;
        }
        std::vector<synnax::channel::Key> keys;
        keys.reserve(this->channels.size());
        for (const auto &ch: this->channels)
            keys.push_back(ch->synnax_key);
        auto [sy_channels, ch_err] = client->channels.retrieve(keys);
        if (ch_err) {
            parser.field_err("channels", "failed to retrieve channels: " + ch_err.data);
            return;
        }
        for (std::size_t i = 0; i < sy_channels.size(); i++) {
            auto ch = sy_channels[i];
            if (ch.index != 0) this->index_keys.insert(ch.index);
            this->channels[i]->ch = ch;
        }
        for (const auto &ch: sy_channels) {
            if (ch.is_index && this->index_keys.contains(ch.key))
                this->index_keys.erase(ch.key);
        }
    }

    [[nodiscard]] std::vector<synnax::channel::Channel> sy_channels() const {
        std::vector<synnax::channel::Channel> chs;
        chs.reserve(this->channels.size());
        for (const auto &ch: this->channels)
            chs.push_back(ch->ch);
        return chs;
    }

    [[nodiscard]] synnax::framer::WriterConfig writer_config() const {
        std::vector<synnax::channel::Key> channel_keys;
        channel_keys.reserve(this->channels.size() + this->index_keys.size());
        for (const auto &ch: this->channels)
            channel_keys.push_back(ch->synnax_key);
        for (const auto &idx: this->index_keys)
            channel_keys.push_back(idx);
        return {
            .channels = channel_keys,
            .mode = common::data_saving_writer_mode(this->data_saving_disabled),
        };
    }

    static std::pair<ReadTaskConfig, x::errors::Error> parse(
        const std::shared_ptr<synnax::Synnax> &client,
        const synnax::task::Task &task
    ) {
        auto parser = x::json::Parser(task.config);
        return {ReadTaskConfig(client, parser), parser.error()};
    }
};

/// @brief Helper to create a ReadRequestBuilder from config.
/// The builder borrows NodeIds from cfg, so cfg must outlive the builder.
static types::ReadRequestBuilder create_read_request(const ReadTaskConfig &cfg) {
    types::ReadRequestBuilder builder;
    for (const auto &ch: cfg.channels)
        builder.add_node(ch->node, UA_ATTRIBUTEID_VALUE);
    return builder;
}

class BaseReadTaskSource : public common::Source {
protected:
    const ReadTaskConfig cfg;
    std::shared_ptr<synnax::Synnax> client;
    std::shared_ptr<connection::Pool> pool;
    /// @brief the device's connection settings as of the last start.
    connection::Config conn_cfg;
    connection::Pool::Connection connection;
    types::ReadRequestBuilder request_builder;

    /// @brief reads the value of every channel. When the read fails after the client
    /// certificate changed, connects again with the new certificate and reads again.
    std::pair<types::ReadResponse, x::errors::Error> read_values() {
        types::ReadResponse res(UA_Client_Service_read(
            this->connection.get(),
            this->request_builder.build()
        ));
        auto err = errors::parse(res.get().responseHeader.serviceResult);
        if (!err) return {std::move(res), x::errors::NIL};
        auto [refreshed, refresh_err] = this->pool->refresh(
            this->connection,
            this->conn_cfg,
            "[opc.read] "
        );
        if (refresh_err) return {std::move(res), refresh_err};
        if (!refreshed) return {std::move(res), err};
        types::ReadResponse retry(UA_Client_Service_read(
            this->connection.get(),
            this->request_builder.build()
        ));
        err = errors::parse(retry.get().responseHeader.serviceResult);
        return {std::move(retry), err};
    }

    BaseReadTaskSource(
        std::shared_ptr<synnax::Synnax> client,
        std::shared_ptr<connection::Pool> pool,
        ReadTaskConfig cfg
    ):
        cfg(std::move(cfg)),
        client(std::move(client)),
        pool(std::move(pool)),
        connection(nullptr, nullptr, ""),
        request_builder(create_read_request(this->cfg)) {}

public:
    [[nodiscard]] synnax::framer::WriterConfig writer_config() const override {
        return this->cfg.writer_config();
    }

    /// @brief retrieves the device again, so a changed endpoint or trusted server
    /// certificate takes effect on restart.
    x::errors::Error start() override {
        auto [conn_cfg, conn_err] = device::retrieve_connection(
            *this->client,
            this->cfg.device_key
        );
        if (conn_err) return conn_err;
        this->conn_cfg = std::move(conn_cfg);
        auto [c, err] = this->pool->acquire(this->conn_cfg, "[opc.read] ");
        if (err) return err;
        connection = std::move(c);
        return x::errors::NIL;
    }

    x::errors::Error stop() override {
        connection = connection::Pool::Connection(nullptr, nullptr, "");
        return x::errors::NIL;
    }
};

class ArrayReadTaskSource final : public BaseReadTaskSource {
    /// @brief regulates the rate blocks of samples are read at.
    x::loop::Timer timer;
    /// @brief where the next block may start at the earliest: the end of the
    /// previous block, or the start of the run.
    x::telem::TimeStamp prev_end = x::telem::TimeStamp(0);

public:
    ArrayReadTaskSource(
        std::shared_ptr<synnax::Synnax> client,
        std::shared_ptr<connection::Pool> pool,
        ReadTaskConfig cfg
    ):
        BaseReadTaskSource(std::move(client), std::move(pool), std::move(cfg)),
        timer(this->cfg.sample_rate / this->cfg.array_size) {}

    /// @brief re-arms the block timer, so the first read after a restart waits a
    /// full block.
    x::errors::Error start() override {
        this->timer = x::loop::Timer(this->cfg.sample_rate / this->cfg.array_size);
        this->prev_end = x::telem::TimeStamp::now();
        return BaseReadTaskSource::start();
    }

    [[nodiscard]] std::vector<synnax::channel::Channel> channels() const override {
        return this->cfg.sy_channels();
    }

    common::ReadResult
    read(x::breaker::Breaker &breaker, ::x::telem::Frame &fr) override {
        common::ReadResult res;
        this->timer.wait(breaker);
        auto [ua_res, read_err] = this->read_values();
        const auto end = x::telem::TimeStamp::now();
        if (res.error = read_err; res.error) return res;
        common::initialize_frame(
            fr,
            this->cfg.channels,
            this->cfg.index_keys,
            this->cfg.array_size
        );
        std::vector<std::string> error_messages;
        for (std::size_t i = 0; i < ua_res.get().resultsSize; ++i) {
            auto &result = ua_res.get().results[i];
            const auto &ch = cfg.channels[i];
            if (res.error = errors::parse(result.status); res.error) {
                res.error = driver::errors::wrap_channel_error(
                    res.error,
                    ch->ch.name,
                    types::NodeId::to_string(ch->node.get())
                );
                return res;
            }
            auto &s = fr.series->at(i);
            s.clear();
            auto [written, err] = telem::ua_array_write_to_series(
                s,
                &result.value,
                this->cfg.array_size,
                ch->ch.name
            );
            if (err || written == 0)
                error_messages.push_back(
                    err ? err.data
                        : "Invalid OPC UA array data detected for channel " +
                              ch->ch.name
                );
        }
        if (!error_messages.empty()) {
            fr.clear();
            res.warning = {
                .message = "Skipped invalid data",
                .description = x::strings::join(error_messages, "; "),
            };
            return res;
        }
        // The array holds the samples taken at the sample rate up to the read, so a
        // late read does not compress them. A block never starts before the previous
        // block's end or the start of the run, so stamps stay in order.
        const auto start = std::max(
            this->prev_end,
            end - this->cfg.sample_rate.period() * this->cfg.array_size
        );
        common::generate_index_data(
            fr,
            this->cfg.index_keys,
            start,
            end,
            this->cfg.array_size,
            this->cfg.channels.size()
        );
        this->prev_end = end;
        return res;
    }
};

class UnaryReadTaskSource final : public BaseReadTaskSource {
    /// @brief regulates the rate individual samples are read at.
    x::loop::Timer timer;

public:
    UnaryReadTaskSource(
        std::shared_ptr<synnax::Synnax> client,
        std::shared_ptr<connection::Pool> pool,
        ReadTaskConfig cfg
    ):
        BaseReadTaskSource(std::move(client), std::move(pool), std::move(cfg)),
        timer(this->cfg.sample_rate) {}

    common::ReadResult
    read(x::breaker::Breaker &breaker, ::x::telem::Frame &fr) override {
        common::ReadResult res;
        common::initialize_frame(
            fr,
            this->cfg.channels,
            this->cfg.index_keys,
            this->cfg.samples_per_chan
        );
        for (auto [k, s]: fr)
            s.clear();
        for (std::size_t i = 0; i < this->cfg.samples_per_chan; i++) {
            const auto start = ::x::telem::TimeStamp::now();
            auto [ua_res, read_err] = this->read_values();
            if (res.error = read_err; res.error) return res;
            bool skip_sample = false;
            for (std::size_t j = 0; j < ua_res.get().resultsSize; ++j) {
                UA_DataValue &result = ua_res.get().results[j];
                const auto &ch = this->cfg.channels[j];
                if (res.error = errors::parse(result.status); res.error) {
                    res.error = driver::errors::wrap_channel_error(
                        res.error,
                        ch->ch.name,
                        types::NodeId::to_string(ch->node.get())
                    );
                    return res;
                }
                auto [written, write_err] = telem::write_to_series(
                    fr.series->at(j),
                    result.value
                );
                if (write_err) {
                    skip_sample = true;
                    res.warning = {
                        .message = "Skipped invalid data",
                        .description = "Invalid OPC UA data detected for channel " +
                                       ch->ch.name + ": " + write_err.data,
                    };
                    break;
                }
            }
            if (skip_sample) {
                fr.clear();
                return res;
            }
            const auto end = ::x::telem::TimeStamp::now();
            const auto ts = ::x::telem::TimeStamp::midpoint(start, end);
            for (std::size_t j = this->cfg.channels.size(); j < fr.size(); j++)
                fr.series->at(j).write(ts);
            this->timer.wait(breaker);
        }
        // Do not write empty frames
        if (!fr.series->empty() && fr.series->at(0).empty()) { fr.clear(); }
        return res;
    }

    [[nodiscard]] std::vector<synnax::channel::Channel> channels() const override {
        return this->cfg.sy_channels();
    }
};
}
