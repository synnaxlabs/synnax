// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstddef>
#include <limits>
#include <map>
#include <memory>
#include <optional>
#include <set>
#include <string>
#include <tuple>
#include <utility>
#include <vector>

#include "open62541/client_highlevel.h"
#include "open62541/common.h"
#include "open62541/types.h"

#include "client/cpp/errors/errors.h"
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
    /// @brief whether a bad value of a float channel writes NaN. Otherwise, a bad value
    /// skips the read for every channel on the same index.
    const bool nan_fill_enabled;
    /// @brief keys of the index channels for the input channels.
    /// Dynamically populated by querying the core.
    std::set<synnax::channel::Key> index_keys;
    /// @brief the enabled channels to read from the server.
    std::vector<std::unique_ptr<InputChan>> channels;
    /// @brief the number of samples to read on each iteration.
    const size_t samples_per_chan;
    /// @brief the index of each channel, in the order of channels. An index channel is
    /// its own index.
    std::vector<synnax::channel::Key> indexes;

    /// @brief Move constructor to allow transfer of ownership
    ReadTaskConfig(ReadTaskConfig &&other) noexcept:
        common::BaseReadTaskConfig(std::move(other)),
        device_key(other.device_key),
        array_mode(other.array_mode),
        array_size(other.array_size),
        nan_fill_enabled(other.nan_fill_enabled),
        index_keys(std::move(other.index_keys)),
        channels(std::move(other.channels)),
        samples_per_chan(other.samples_per_chan),
        indexes(std::move(other.indexes)) {}

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
        nan_fill_enabled(parser.field<bool>("nan_fill_enabled", false)),
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
            this->indexes.push_back(ch.is_index ? ch.key : ch.index);
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

/// @brief the result of a channel whose namespace the server lacks.
inline const UA_DataValue UNKNOWN_NAMESPACE = [] {
    UA_DataValue value;
    UA_DataValue_init(&value);
    value.hasStatus = true;
    value.status = UA_STATUSCODE_BADNODEIDUNKNOWN;
    return value;
}();

/// @brief the problems with the values of one read, reported as one task warning.
class Quality {
    std::vector<std::string> problems;
    bool bad = false;

public:
    /// @brief records a bad value, or a value that failed to convert.
    void add_bad(const std::string &problem) {
        this->bad = true;
        this->problems.push_back(problem);
    }

    /// @brief records an uncertain value.
    void add_uncertain(const std::string &problem) {
        this->problems.push_back(problem);
    }

    [[nodiscard]] common::Warning warning() const {
        if (this->problems.empty()) return {};
        return {
            .message = this->bad ? "Received bad values" : "Received uncertain values",
            .description = x::strings::join(this->problems, " "),
        };
    }
};

/// @returns NaN as a sample of the data type, or nothing if the data type holds no NaN.
inline std::optional<x::telem::SampleValue> nan(const x::telem::DataType &dt) {
    if (dt != x::telem::FLOAT32_T && dt != x::telem::FLOAT64_T) return std::nullopt;
    return dt.cast(std::numeric_limits<double>::quiet_NaN());
}

class BaseReadTaskSource : public common::Source {
protected:
    const ReadTaskConfig cfg;
    std::shared_ptr<synnax::Synnax> client;
    std::shared_ptr<connection::Pool> pool;
    /// @brief the device's connection settings as of the last retrieve.
    connection::Config conn_cfg;
    connection::Pool::Connection connection;
    /// @brief the error of the connection attempt at start, which the first read
    /// returns instead of connecting again.
    x::errors::Error start_err;
    /// @brief reads the channels on the current connection. It borrows the node IDs
    /// of cfg.
    types::ReadRequestBuilder request_builder;
    /// @brief the place of each channel's result in a response to request_builder, or
    /// nothing when the server lacks the channel's namespace.
    std::vector<std::optional<std::size_t>> positions;

    /// @brief reads the channels on conn from now on, with the namespace indexes of
    /// its server.
    void use(connection::Pool::Connection conn) {
        this->connection = std::move(conn);
        this->build_request();
    }

    void build_request() {
        this->request_builder = types::ReadRequestBuilder();
        this->positions.clear();
        const auto &namespaces = this->connection.server().namespaces;
        for (const auto &ch: this->cfg.channels) {
            const auto id = ch->node.resolve(namespaces);
            if (!id.has_value()) {
                this->positions.emplace_back(std::nullopt);
                continue;
            }
            this->positions.emplace_back(this->request_builder.size());
            this->request_builder.add_node(id.value(), UA_ATTRIBUTEID_VALUE);
        }
    }

    /// @returns the result of channel i in res, a response to request_builder.
    [[nodiscard]] const UA_DataValue &
    result(const UA_ReadResponse &res, const std::size_t i) const {
        const auto pos = this->positions[i];
        if (!pos.has_value()) return UNKNOWN_NAMESPACE;
        return res.results[pos.value()];
    }

    /// @brief connects with the device's current settings, so a corrected endpoint or
    /// login takes effect without a restart. Keeps the last settings while the Core is
    /// unavailable.
    x::errors::Error connect() {
        auto [conn_cfg, retrieve_err] = device::retrieve_connection(
            *this->client,
            this->cfg.device_key
        );
        if (!retrieve_err)
            this->conn_cfg = std::move(conn_cfg);
        else if (!synnax::errors::is_temporarily_unavailable(retrieve_err))
            return retrieve_err;
        auto [c, err] = this->pool->acquire(this->conn_cfg, "[opc.read] ");
        if (err) return err;
        this->use(std::move(c));
        return x::errors::NIL;
    }

    /// @brief reads the value of every channel. Connects first when the task has no
    /// connection, and drops a connection that is lost so the next read connects
    /// again. When the read fails after the client certificate changed, connects again
    /// with the new certificate and reads again.
    std::pair<types::ReadResponse, x::errors::Error> read_values() {
        if (!this->connection) {
            if (this->start_err)
                return {
                    types::ReadResponse(),
                    std::exchange(this->start_err, x::errors::NIL)
                };
            if (auto err = this->connect()) return {types::ReadResponse(), err};
        }
        if (this->request_builder.empty())
            return {types::ReadResponse(), x::errors::NIL};
        auto [res, err] = this->connection.read(this->request_builder.build());
        if (!err) return {std::move(res), x::errors::NIL};
        auto [refreshed, refresh_err] = this->pool->refresh(
            this->connection,
            this->conn_cfg,
            "[opc.read] "
        );
        if (refreshed) {
            this->build_request();
            std::tie(res, err) = this->connection.read(this->request_builder.build());
        }
        if (refresh_err) err = refresh_err;
        if (err.matches(errors::UNREACHABLE))
            this->connection = connection::Pool::Connection(nullptr, nullptr, "");
        return {std::move(res), err};
    }

    BaseReadTaskSource(
        std::shared_ptr<synnax::Synnax> client,
        std::shared_ptr<connection::Pool> pool,
        ReadTaskConfig cfg
    ):
        cfg(std::move(cfg)),
        client(std::move(client)),
        pool(std::move(pool)),
        connection(nullptr, nullptr, "") {}

public:
    [[nodiscard]] synnax::framer::WriterConfig writer_config() const override {
        return this->cfg.writer_config();
    }

    /// @brief retrieves the device again, so a changed trusted server certificate takes
    /// effect on restart. A server that is off or refuses Sessions does not stop the
    /// start, since the reads connect again.
    x::errors::Error start() override {
        auto [conn_cfg, conn_err] = device::retrieve_connection(
            *this->client,
            this->cfg.device_key
        );
        if (conn_err) return conn_err;
        this->conn_cfg = std::move(conn_cfg);
        auto [c, err] = this->pool->acquire(this->conn_cfg, "[opc.read] ");
        if (err.matches(driver::errors::TEMPORARY_HARDWARE_ERROR)) {
            this->start_err = err;
            return x::errors::NIL;
        }
        if (err) return err;
        this->use(std::move(c));
        return x::errors::NIL;
    }

    x::errors::Error stop() override {
        this->connection = connection::Pool::Connection(nullptr, nullptr, "");
        this->start_err = x::errors::NIL;
        return x::errors::NIL;
    }
};

class ArrayReadTaskSource final : public BaseReadTaskSource {
    /// @brief regulates the read rate and times each block of samples.
    common::SoftwareTimedSampleClock sample_clock;

public:
    ArrayReadTaskSource(
        std::shared_ptr<synnax::Synnax> client,
        std::shared_ptr<connection::Pool> pool,
        ReadTaskConfig cfg
    ):
        BaseReadTaskSource(std::move(client), std::move(pool), std::move(cfg)),
        sample_clock(this->cfg.sample_rate / this->cfg.array_size) {}

    /// @brief re-arms the block timer. Without it, the first block after a restart
    /// spans only the time the read took instead of a full acquisition window.
    x::errors::Error start() override {
        this->sample_clock = common::SoftwareTimedSampleClock(
            this->cfg.sample_rate / this->cfg.array_size
        );
        return BaseReadTaskSource::start();
    }

    [[nodiscard]] std::vector<synnax::channel::Channel> channels() const override {
        return this->cfg.sy_channels();
    }

    common::ReadResult
    read(x::breaker::Breaker &breaker, ::x::telem::Frame &fr) override {
        common::ReadResult res;
        const auto start = this->sample_clock.wait(breaker);
        auto [ua_res, read_err] = this->read_values();
        if (res.error = read_err; res.error) return res;
        fr.clear();
        Quality quality;
        std::vector<std::optional<x::telem::Series>> values;
        values.reserve(this->cfg.channels.size());
        std::set<synnax::channel::Key> skipped;
        const auto &results = ua_res.get();
        for (std::size_t i = 0; i < this->cfg.channels.size(); ++i) {
            const auto &ch = *this->cfg.channels[i];
            const auto &result = this->result(results, i);
            x::telem::Series s(ch.ch.data_type, this->cfg.array_size);
            x::errors::Error err;
            if (UA_StatusCode_isBad(result.status))
                quality.add_bad(errors::channel_status(ch.ch.name, result.status));
            else {
                if (UA_StatusCode_isUncertain(result.status))
                    quality.add_uncertain(
                        errors::channel_status(ch.ch.name, result.status)
                    );
                err = telem::ua_array_write_to_series(
                          s,
                          &result.value,
                          this->cfg.array_size,
                          ch.ch.name
                )
                          .second;
                if (err) quality.add_bad(ch.ch.name + ": " + err.data + ".");
            }
            if (!UA_StatusCode_isBad(result.status) && !err) {
                values.emplace_back(std::move(s));
                continue;
            }
            const auto fill = this->cfg.nan_fill_enabled ? nan(ch.ch.data_type)
                                                         : std::nullopt;
            if (!fill.has_value()) {
                skipped.insert(this->cfg.indexes[i]);
                values.emplace_back(std::nullopt);
                continue;
            }
            x::telem::Series filled(ch.ch.data_type, this->cfg.array_size);
            for (std::size_t j = 0; j < this->cfg.array_size; ++j)
                filled.write(*fill);
            values.emplace_back(std::move(filled));
        }
        for (std::size_t i = 0; i < values.size(); ++i)
            if (values[i].has_value() && !skipped.contains(this->cfg.indexes[i]))
                fr.emplace(
                    this->cfg.channels[i]->synnax_key,
                    std::move(values[i].value())
                );
        // Stamping ahead of the clock would make the next run of this task open its
        // writer inside data this run already wrote.
        const auto end = this->sample_clock.end();
        for (const auto &idx: this->cfg.index_keys)
            if (!skipped.contains(idx))
                fr.emplace(
                    idx,
                    x::telem::Series::linspace(start, end, this->cfg.array_size)
                );
        res.warning = quality.warning();
        return res;
    }
};

class UnaryReadTaskSource final : public BaseReadTaskSource {
    /// @brief regulates the rate individual samples are read at.
    x::loop::Timer timer;
    /// @brief the samples of each channel in the current read.
    std::vector<std::vector<x::telem::SampleValue>> samples;
    /// @brief the stamps of each generated index in the current read.
    std::map<synnax::channel::Key, std::vector<x::telem::TimeStamp>> stamps;

    /// @brief adds the value of each channel at ts, skipping every channel on the
    /// index of a channel with a bad value.
    void collect(const UA_ReadResponse &res, const x::telem::TimeStamp ts, Quality &q) {
        std::vector<std::optional<x::telem::SampleValue>> values;
        values.reserve(this->cfg.channels.size());
        std::set<synnax::channel::Key> skipped;
        for (std::size_t i = 0; i < this->cfg.channels.size(); ++i) {
            const auto &ch = *this->cfg.channels[i];
            const auto &result = this->result(res, i);
            if (UA_StatusCode_isBad(result.status)) {
                q.add_bad(errors::channel_status(ch.ch.name, result.status));
                values.emplace_back(std::nullopt);
            } else {
                if (UA_StatusCode_isUncertain(result.status))
                    q.add_uncertain(errors::channel_status(ch.ch.name, result.status));
                auto [sample, err] = telem::to_sample(result.value, ch.ch.data_type);
                if (err) {
                    q.add_bad(ch.ch.name + ": " + err.data + ".");
                    values.emplace_back(std::nullopt);
                } else
                    values.emplace_back(std::move(sample));
            }
            if (values.back().has_value()) continue;
            if (this->cfg.nan_fill_enabled) values.back() = nan(ch.ch.data_type);
            if (!values.back().has_value()) skipped.insert(this->cfg.indexes[i]);
        }
        for (std::size_t i = 0; i < values.size(); ++i)
            if (!skipped.contains(this->cfg.indexes[i]))
                this->samples[i].push_back(std::move(values[i].value()));
        for (auto &[idx, idx_stamps]: this->stamps)
            if (!skipped.contains(idx)) idx_stamps.push_back(ts);
    }

public:
    UnaryReadTaskSource(
        std::shared_ptr<synnax::Synnax> client,
        std::shared_ptr<connection::Pool> pool,
        ReadTaskConfig cfg
    ):
        BaseReadTaskSource(std::move(client), std::move(pool), std::move(cfg)),
        timer(this->cfg.sample_rate),
        samples(this->cfg.channels.size()) {
        for (const auto &idx: this->cfg.index_keys)
            this->stamps[idx] = {};
    }

    common::ReadResult
    read(x::breaker::Breaker &breaker, ::x::telem::Frame &fr) override {
        common::ReadResult res;
        for (auto &s: this->samples)
            s.clear();
        for (auto &[_, s]: this->stamps)
            s.clear();
        Quality quality;
        for (std::size_t i = 0; i < this->cfg.samples_per_chan; i++) {
            const auto start = ::x::telem::TimeStamp::now();
            auto [ua_res, read_err] = this->read_values();
            if (res.error = read_err; res.error) return res;
            const auto end = ::x::telem::TimeStamp::now();
            this->collect(ua_res, ::x::telem::TimeStamp::midpoint(start, end), quality);
            this->timer.wait(breaker);
        }
        fr.clear();
        for (std::size_t i = 0; i < this->samples.size(); ++i)
            if (!this->samples[i].empty())
                fr.emplace(
                    this->cfg.channels[i]->synnax_key,
                    telem::to_series(
                        this->cfg.channels[i]->ch.data_type,
                        this->samples[i]
                    )
                );
        for (const auto &[idx, idx_stamps]: this->stamps)
            if (!idx_stamps.empty()) fr.emplace(idx, x::telem::Series(idx_stamps));
        res.warning = quality.warning();
        return res;
    }

    [[nodiscard]] std::vector<synnax::channel::Channel> channels() const override {
        return this->cfg.sy_channels();
    }
};
}
