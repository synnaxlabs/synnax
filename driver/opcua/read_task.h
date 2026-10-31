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
#include <limits>
#include <map>
#include <memory>
#include <numeric>
#include <optional>
#include <set>
#include <string>
#include <tuple>
#include <utility>
#include <vector>

#include "absl/log/log.h"
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
#include "driver/opcua/structure/structure.h"
#include "driver/opcua/telem/telem.h"
#include "driver/opcua/types/types.h"

namespace driver::opcua {
struct InputChan {
    /// @brief the OPC UA node id.
    types::NodeId node;
    /// @brief the member of the node's structure value to read, or empty to read the
    /// whole value.
    structure::Path field;
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
        if (parsed.field.empty()) return;
        auto [field, err] = structure::parse(parsed.field);
        if (err) parser.field_err("field", err.data);
        this->field = std::move(field);
    }

    // Move constructor - needed because NodeId is move-only
    InputChan(InputChan &&other) noexcept:
        node(std::move(other.node)),
        field(std::move(other.field)),
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
    /// @brief how the task reads its nodes: sampled or on_change.
    const std::string mode;
    /// @brief whether the task subscribes to the changes of its nodes instead of
    /// reading them at the sample rate.
    const bool on_change;
    /// @brief whether an On change task reads the history of its nodes from the last
    /// stored value to the start of each subscription.
    const bool history_backfill_enabled;
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
    /// @brief the indexes whose stamps come from the server: each index in On change
    /// mode, and each index channel that a DateTime node gives in Sampled mode.
    std::vector<synnax::channel::Key> stamped;

    /// @brief Move constructor to allow transfer of ownership
    ReadTaskConfig(ReadTaskConfig &&other) noexcept:
        common::BaseReadTaskConfig(std::move(other)),
        device_key(other.device_key),
        array_mode(other.array_mode),
        array_size(other.array_size),
        nan_fill_enabled(other.nan_fill_enabled),
        mode(other.mode),
        on_change(other.on_change),
        history_backfill_enabled(other.history_backfill_enabled),
        index_keys(std::move(other.index_keys)),
        channels(std::move(other.channels)),
        samples_per_chan(other.samples_per_chan),
        indexes(std::move(other.indexes)),
        stamped(std::move(other.stamped)) {}

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
        mode(parser.field<std::string>("mode", ::synnax::opcua::READ_MODE_SAMPLED)),
        on_change(this->mode == ::synnax::opcua::READ_MODE_ON_CHANGE),
        history_backfill_enabled(parser.field<bool>("history_backfill_enabled", false)),
        // Array mode never uses samples_per_chan and its stream rate may be zero.
        samples_per_chan(this->array_mode ? 0 : this->sample_rate / this->stream_rate) {
        if (this->mode != ::synnax::opcua::READ_MODE_SAMPLED && !this->on_change)
            parser.field_err("mode", "Mode must be sampled or on_change");
        if (this->on_change && this->array_mode)
            parser.field_err("array_mode", "On change mode cannot read arrays");
        // The place of each enabled channel in the channels of the config.
        std::vector<std::size_t> places;
        std::size_t place = 0;
        parser.iter("channels", [&](x::json::Parser &cp) {
            const auto parsed = ::synnax::opcua::ReadChannel::parse(cp);
            if (parsed.disabled) {
                place++;
                return;
            }
            if (this->array_mode && !parsed.field.empty())
                cp.field_err("field", "array mode cannot read structure fields");
            channels.push_back(std::make_unique<InputChan>(parsed, cp));
            places.push_back(place++);
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
            if (ch.is_index || this->on_change)
                this->stamped.push_back(ch.is_index ? ch.key : ch.index);
        }
        if (!this->on_change) return;
        std::set<synnax::channel::Key> owned;
        for (std::size_t i = 0; i < sy_channels.size(); ++i) {
            const auto path = "channels." + std::to_string(places[i]) + ".channel";
            if (sy_channels[i].is_index)
                parser.field_err(path, "On change mode cannot read an index channel");
            else if (
                sy_channels[i].index == 0 || !owned.insert(sy_channels[i].index).second
            )
                parser.field_err(
                    path,
                    "Channel must have its own index in On change mode"
                );
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

/// @brief the problems of one read, reported as one task warning named after the worst
/// kind. Each problem appears once.
class Problems {
    /// @brief the kinds of problem, from the least to the most severe.
    enum Kind : std::uint8_t {
        NONE,
        UNCERTAIN,
        REJECTED,
        HISTORY,
        BAD,
        UNMONITORED,
    };
    std::vector<std::string> problems;
    Kind worst = NONE;

    void add(const Kind kind, std::string problem) {
        this->worst = std::max(this->worst, kind);
        if (std::ranges::find(this->problems, problem) == this->problems.end())
            this->problems.push_back(std::move(problem));
    }

public:
    /// @brief records a bad value, or a value that failed to convert.
    void add_bad(std::string problem) { this->add(BAD, std::move(problem)); }

    /// @brief records an uncertain value.
    void add_uncertain(std::string problem) {
        this->add(UNCERTAIN, std::move(problem));
    }

    /// @brief records a value whose time is not after the last one on its index.
    void add_rejected(std::string problem) { this->add(REJECTED, std::move(problem)); }

    /// @brief records a node that the server does not monitor.
    void add_unmonitored(std::string problem) {
        this->add(UNMONITORED, std::move(problem));
    }

    /// @brief records history that the server did not return.
    void add_history(std::string problem) { this->add(HISTORY, std::move(problem)); }

    /// @brief adds the problems of other.
    void merge(const Problems &other) {
        this->worst = std::max(this->worst, other.worst);
        for (const auto &p: other.problems)
            if (std::ranges::find(this->problems, p) == this->problems.end())
                this->problems.push_back(p);
    }

    [[nodiscard]] common::Warning warning() const {
        std::string message;
        switch (this->worst) {
            case UNMONITORED:
                message = "Failed to subscribe to some nodes";
                break;
            case BAD:
                message = "Received bad values";
                break;
            case HISTORY:
                message = "Failed to read the history of some nodes";
                break;
            case REJECTED:
                message = "Rejected values with old timestamps";
                break;
            case UNCERTAIN:
                message = "Received uncertain values";
                break;
            case NONE:
                return {};
        }
        return {
            .message = std::move(message),
            .description = x::strings::join(this->problems, " "),
        };
    }
};

/// @returns NaN as a sample of the data type, or nothing if the data type holds no NaN.
inline std::optional<x::telem::SampleValue> nan(const x::telem::DataType &dt) {
    if (dt != x::telem::FLOAT32_T && dt != x::telem::FLOAT64_T) return std::nullopt;
    return dt.cast(std::numeric_limits<double>::quiet_NaN());
}

/// @brief connects a read task to its server, and turns the values of its nodes into
/// samples of its channels.
class BaseReadTaskSource : public common::Source {
protected:
    const ReadTaskConfig cfg;
    std::shared_ptr<synnax::Synnax> client;
    device::Connector connector;
    connection::Pool::Connection connection;
    /// @brief the error of the connection attempt at start, which the first read
    /// returns instead of connecting again.
    x::errors::Error start_err;
    /// @brief the structure types of the nodes of field channels, on the current
    /// connection.
    structure::Types types;
    /// @brief the member that each field channel reads, or the error that keeps the
    /// channel from reading it, on the current connection.
    std::vector<std::pair<structure::Field, x::errors::Error>> fields;
    /// @brief the last time written to each index of cfg.stamped.
    std::map<synnax::channel::Key, x::telem::TimeStamp> last;
    /// @brief when read_types last ran.
    x::telem::TimeStamp types_read{};

    /// @brief reads the channels on conn from now on, with the namespace indexes of its
    /// server.
    /// @returns the error that keeps the source from reading on conn, after which the
    /// source has no connection.
    virtual x::errors::Error use(connection::Pool::Connection conn) = 0;

    /// @brief reads the structure types of the nodes of field channels, and finds the
    /// member that each field channel reads.
    /// @returns only an error that loses the connection. Each field channel reports any
    /// other error when it reads.
    x::errors::Error read_types() {
        this->types_read = x::telem::TimeStamp::now();
        this->types = structure::Types();
        this->fields.assign(this->cfg.channels.size(), {});
        const auto &namespaces = this->connection.server().namespaces;
        types::ReadRequestBuilder req;
        // The place in req of the node of each field channel. Field channels on one
        // node share it.
        std::vector<std::optional<std::size_t>> places(this->cfg.channels.size());
        std::map<std::string, std::size_t> nodes;
        for (std::size_t i = 0; i < this->cfg.channels.size(); ++i) {
            const auto &ch = *this->cfg.channels[i];
            if (ch.field.empty()) continue;
            const auto id = ch.node.resolve(namespaces);
            if (!id.has_value()) continue;
            const auto [it, added] = nodes.try_emplace(
                types::NodeId::to_string(id.value()),
                req.size()
            );
            places[i] = it->second;
            if (added) req.add_node(id.value(), UA_ATTRIBUTEID_DATATYPE);
        }
        if (req.empty()) return x::errors::NIL;
        auto [res, err] = this->connection.read(req.build());
        std::vector<std::optional<UA_NodeId>> data_types(req.size());
        std::vector<UA_NodeId> known;
        for (std::size_t k = 0; !err && k < data_types.size(); ++k)
            if (const auto &v = res.get().results[k].value;
                UA_Variant_hasScalarType(&v, &UA_TYPES[UA_TYPES_NODEID])) {
                data_types[k] = *static_cast<const UA_NodeId *>(v.data);
                known.push_back(data_types[k].value());
            }
        if (!err)
            std::tie(this->types, err) = structure::Types::read(
                this->connection,
                known
            );
        if (err.matches(errors::UNREACHABLE)) return err;
        for (std::size_t i = 0; i < this->cfg.channels.size(); ++i) {
            const auto place = places[i];
            if (!place.has_value()) continue;
            auto &field = this->fields[i];
            const auto &data_type = data_types[place.value()];
            if (err)
                field.second = x::errors::Error(
                    err,
                    "failed to read the structure types: " + err.data
                );
            else if (!data_type.has_value())
                field.second = x::errors::Error(
                    x::errors::VALIDATION,
                    "the server did not report the data type of the node"
                );
            else
                field = this->types.field(
                    data_type.value(),
                    this->cfg.channels[i]->field
                );
        }
        return x::errors::NIL;
    }

    /// @brief reads the structure types again when a field channel failed and
    /// connection::RETRY_INTERVAL has passed.
    /// @returns only an error that loses the connection.
    x::errors::Error retry_types() {
        const bool failing = std::ranges::any_of(this->fields, [](const auto &field) {
            return static_cast<bool>(field.second);
        });
        if (!failing ||
            x::telem::TimeStamp::now() - this->types_read < connection::RETRY_INTERVAL)
            return x::errors::NIL;
        return this->read_types();
    }

    /// @returns the problem of channel i when the server lacks its namespace.
    [[nodiscard]] std::string missing_namespace(const std::size_t i) const {
        const auto &ch = *this->cfg.channels[i];
        return ch.ch.name + ": the server has no namespace " + ch.node.namespace_uri() +
               ".";
    }

    x::errors::Error connect() {
        auto [conn, err] = this->connector.connect();
        if (err) return err;
        return this->use(std::move(conn));
    }

    /// @returns the value of channel i that convert gives from result, or nothing when
    /// the value is bad or fails to convert. Adds the problems of the value to q when
    /// it is set.
    template<typename T, typename Convert>
    std::optional<T> value(
        const std::size_t i,
        const UA_DataValue &result,
        Problems *q,
        const Convert &convert
    ) const {
        const auto &ch = *this->cfg.channels[i];
        const auto &name = ch.ch.name;
        if (UA_StatusCode_isBad(result.status)) {
            if (q == nullptr) return std::nullopt;
            if (!ch.node.resolve(this->connection.server().namespaces).has_value())
                q->add_bad(this->missing_namespace(i));
            else
                q->add_bad(errors::channel_status(name, result.status));
            return std::nullopt;
        }
        if (q != nullptr && UA_StatusCode_isUncertain(result.status))
            q->add_uncertain(errors::channel_status(name, result.status));
        auto [v, err] = convert(result.value);
        if (!err) return std::move(v);
        if (q != nullptr) q->add_bad(name + ": " + err.data + ".");
        return std::nullopt;
    }

    /// @returns the sample of channel i in v. A field channel decodes the structure of
    /// its node into decoded, which every field channel on the node shares.
    std::pair<x::telem::SampleValue, x::errors::Error> to_sample(
        const std::size_t i,
        const UA_Variant &v,
        std::optional<std::pair<types::Variant, x::errors::Error>> &decoded
    ) const {
        const auto &ch = *this->cfg.channels[i];
        if (ch.field.empty()) return telem::to_sample(v, ch.ch.data_type);
        const auto &[field, field_err] = this->fields[i];
        if (field_err) return {{}, field_err};
        if (!decoded.has_value()) decoded = this->types.decode(v);
        if (decoded->second) return {{}, decoded->second};
        return structure::Types::sample(decoded->first.get(), field, ch.ch.data_type);
    }

    BaseReadTaskSource(
        std::shared_ptr<synnax::Synnax> client,
        std::shared_ptr<connection::Pool> pool,
        ReadTaskConfig cfg
    ):
        cfg(std::move(cfg)),
        client(client),
        connector(
            std::move(client),
            std::move(pool),
            this->cfg.device_key,
            "[opc.read] "
        ),
        connection(nullptr, nullptr, "") {}

public:
    BaseReadTaskSource(const BaseReadTaskSource &) = delete;
    BaseReadTaskSource &operator=(const BaseReadTaskSource &) = delete;
    BaseReadTaskSource(BaseReadTaskSource &&) = delete;
    BaseReadTaskSource &operator=(BaseReadTaskSource &&) = delete;
    ~BaseReadTaskSource() override = default;

    [[nodiscard]] synnax::framer::WriterConfig writer_config() const override {
        return this->cfg.writer_config();
    }

    [[nodiscard]] std::vector<synnax::channel::Channel> channels() const override {
        return this->cfg.sy_channels();
    }

    /// @brief finds the last stored time of each index whose stamps come from the
    /// server, and retrieves the device again, so a changed trusted server certificate
    /// takes effect on restart. A server that is off or refuses Sessions does not stop
    /// the start, since the reads connect again.
    x::errors::Error start() override {
        auto [last, last_err] = common::last_stamps(*this->client, this->cfg.stamped);
        if (last_err) return last_err;
        this->last = std::move(last);
        auto [c, err] = this->connector.start();
        if (!err) err = this->use(std::move(c));
        if (err.matches(driver::errors::TEMPORARY_HARDWARE_ERROR)) {
            this->start_err = err;
            return x::errors::NIL;
        }
        return err;
    }

    x::errors::Error stop() override {
        this->connection = connection::Pool::Connection(nullptr, nullptr, "");
        this->start_err = x::errors::NIL;
        return x::errors::NIL;
    }
};

/// @brief reads the value of each channel at the sample rate.
class SampledReadTaskSource : public BaseReadTaskSource {
    /// @brief the indexes of cfg.stamped that a read gave stamps since the start.
    std::set<synnax::channel::Key> read_indexes;
    /// @brief the place of the server's namespace array in a response to
    /// request_builder, which reads it while the server lacks the namespace of a
    /// channel.
    std::optional<std::size_t> namespace_position;

    x::errors::Error use(connection::Pool::Connection conn) override {
        this->connection = std::move(conn);
        this->build_request();
        const auto err = this->read_types();
        if (err) this->connection = connection::Pool::Connection(nullptr, nullptr, "");
        return err;
    }

    void build_request() {
        this->request_builder = types::ReadRequestBuilder();
        this->positions.clear();
        const auto &namespaces = this->connection.server().namespaces;
        std::map<std::string, std::size_t> places;
        for (const auto &ch: this->cfg.channels) {
            const auto id = ch->node.resolve(namespaces);
            if (!id.has_value()) {
                this->positions.emplace_back(std::nullopt);
                continue;
            }
            const auto [it, added] = places.try_emplace(
                types::NodeId::to_string(id.value()),
                this->request_builder.size()
            );
            this->positions.emplace_back(it->second);
            if (added) this->request_builder.add_node(id.value(), UA_ATTRIBUTEID_VALUE);
        }
        this->namespace_position.reset();
        if (std::ranges::all_of(this->positions, [](const auto &p) {
                return p.has_value();
            }))
            return;
        this->namespace_position = this->request_builder.size();
        this->request_builder.add_node(
            UA_NODEID_NUMERIC(0, UA_NS0ID_SERVER_NAMESPACEARRAY),
            UA_ATTRIBUTEID_VALUE
        );
    }

    /// @brief reads request_builder. When the read finds a namespace that the
    /// connection lacked, builds the request again, reads the structure types again,
    /// and reads again.
    std::pair<types::ReadResponse, x::errors::Error> read_request() {
        if (auto err = this->retry_types()) return {types::ReadResponse(), err};
        auto [res, err] = this->connection.read(this->request_builder.build());
        if (err || !this->namespace_position.has_value()) return {std::move(res), err};
        auto [changed, ns_err] = this->connection.update_namespaces(
            res.get().results[this->namespace_position.value()]
        );
        if (ns_err || !changed) return {std::move(res), ns_err};
        this->build_request();
        if (auto types_err = this->read_types())
            return {types::ReadResponse(), types_err};
        return this->connection.read(this->request_builder.build());
    }

protected:
    /// @brief reads the channels on the current connection. It borrows the node IDs of
    /// cfg.
    types::ReadRequestBuilder request_builder;
    /// @brief the place of each channel's result in a response to request_builder, or
    /// nothing when the server lacks the channel's namespace. Channels on the same node
    /// share a place.
    std::vector<std::optional<std::size_t>> positions;

    /// @brief checks the stamps that the DateTime node of the index channel ch gives.
    /// Each must be after the one before it and after the last one written.
    /// @returns true if the stamps write. Otherwise, the read skips the index, and q
    /// gets a warning unless the stamps repeat what the task wrote: the first read
    /// since the start, or a read of the last sample again.
    bool accept(
        const synnax::channel::Channel &ch,
        const std::vector<x::telem::TimeStamp> &stamps,
        Problems &q
    ) {
        const bool first = this->read_indexes.insert(ch.key).second;
        auto &prev = this->last[ch.key];
        auto t = prev;
        for (std::size_t j = 0; j < stamps.size(); ++j) {
            if (stamps[j] > t) {
                t = stamps[j];
                continue;
            }
            if (!first && (j != 0 || stamps[j] != prev))
                q.add_rejected(
                    "Rejected a value of " + ch.name +
                    ": its timestamp is not after the last one written."
                );
            return false;
        }
        prev = t;
        return true;
    }

    /// @returns the result of channel i in res, a response to request_builder.
    [[nodiscard]] const UA_DataValue &
    result(const UA_ReadResponse &res, const std::size_t i) const {
        const auto pos = this->positions[i];
        if (!pos.has_value()) return UNKNOWN_NAMESPACE;
        return res.results[pos.value()];
    }

    /// @returns the NaN that replaces a bad value of channel i, or nothing when NaN
    /// fill is off or the channel's data type holds no NaN, which skips the channel's
    /// index.
    [[nodiscard]] std::optional<x::telem::SampleValue> fill(const std::size_t i) const {
        if (!this->cfg.nan_fill_enabled) return std::nullopt;
        return nan(this->cfg.channels[i]->ch.data_type);
    }

    /// @brief reads the value of every channel. Connects first when the task has no
    /// connection, and drops a connection that is lost so the next read connects again.
    /// When the read fails after the client certificate changed, connects again with
    /// the new certificate and reads again.
    std::pair<types::ReadResponse, x::errors::Error> read_values() {
        if (!this->connection) {
            if (this->start_err)
                return {
                    types::ReadResponse(),
                    std::exchange(this->start_err, x::errors::NIL)
                };
            if (auto err = this->connect()) return {types::ReadResponse(), err};
        }
        auto [res, err] = this->read_request();
        if (!err) return {std::move(res), x::errors::NIL};
        auto [next, refresh_err] = this->connector.refresh(this->connection);
        if (next) {
            err = this->use(std::move(next));
            if (!err) std::tie(res, err) = this->read_request();
        }
        if (refresh_err) err = refresh_err;
        if (err.matches(errors::UNREACHABLE))
            this->connection = connection::Pool::Connection(nullptr, nullptr, "");
        return {std::move(res), err};
    }

    SampledReadTaskSource(
        std::shared_ptr<synnax::Synnax> client,
        std::shared_ptr<connection::Pool> pool,
        ReadTaskConfig cfg
    ):
        BaseReadTaskSource(std::move(client), std::move(pool), std::move(cfg)) {}

public:
    x::errors::Error start() override {
        this->read_indexes.clear();
        return BaseReadTaskSource::start();
    }
};

class ArrayReadTaskSource final : public SampledReadTaskSource {
    /// @brief regulates the read rate and times each block of samples.
    common::SoftwareTimedSampleClock sample_clock;

public:
    ArrayReadTaskSource(
        std::shared_ptr<synnax::Synnax> client,
        std::shared_ptr<connection::Pool> pool,
        ReadTaskConfig cfg
    ):
        SampledReadTaskSource(std::move(client), std::move(pool), std::move(cfg)),
        sample_clock(this->cfg.sample_rate / this->cfg.array_size) {}

    /// @brief re-arms the block timer. Without it, the first block after a restart
    /// spans only the time the read took instead of a full acquisition window.
    x::errors::Error start() override {
        this->sample_clock = common::SoftwareTimedSampleClock(
            this->cfg.sample_rate / this->cfg.array_size
        );
        return SampledReadTaskSource::start();
    }

    common::ReadResult
    read(x::breaker::Breaker &breaker, ::x::telem::Frame &fr) override {
        common::ReadResult res;
        const auto start = this->sample_clock.wait(breaker);
        auto [ua_res, read_err] = this->read_values();
        if (res.error = read_err; res.error) return res;
        fr.clear();
        Problems problems;
        std::vector<std::optional<x::telem::Series>> values;
        values.reserve(this->cfg.channels.size());
        std::set<synnax::channel::Key> skipped;
        const auto &results = ua_res.get();
        for (std::size_t i = 0; i < this->cfg.channels.size(); ++i) {
            const auto &ch = this->cfg.channels[i]->ch;
            auto series = this->value<x::telem::Series>(
                i,
                this->result(results, i),
                &problems,
                [&](const UA_Variant &v) {
                    x::telem::Series s(ch.data_type, this->cfg.array_size);
                    auto err = telem::ua_array_write_to_series(
                                   s,
                                   &v,
                                   this->cfg.array_size,
                                   ch.name
                    )
                                   .second;
                    return std::pair{std::move(s), err};
                }
            );
            if (!series.has_value())
                if (const auto fill = this->fill(i); fill.has_value()) {
                    series.emplace(ch.data_type, this->cfg.array_size);
                    for (std::size_t j = 0; j < this->cfg.array_size; ++j)
                        series->write(*fill);
                }
            if (!series.has_value()) skipped.insert(this->cfg.indexes[i]);
            values.push_back(std::move(series));
        }
        for (std::size_t i = 0; i < values.size(); ++i) {
            const auto &ch = this->cfg.channels[i]->ch;
            const auto &series = values[i];
            if (!ch.is_index || !series.has_value() || skipped.contains(ch.key))
                continue;
            std::vector<x::telem::TimeStamp> stamps;
            stamps.reserve(series->size());
            for (std::size_t j = 0; j < series->size(); ++j)
                stamps.push_back(series->at<x::telem::TimeStamp>(static_cast<int>(j)));
            if (!this->accept(ch, stamps, problems)) skipped.insert(ch.key);
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
        res.warning = problems.warning();
        return res;
    }
};

class UnaryReadTaskSource final : public SampledReadTaskSource {
    /// @brief regulates the rate individual samples are read at.
    x::loop::Timer timer;
    /// @brief the samples of each channel in the current read.
    std::vector<std::vector<x::telem::SampleValue>> samples;
    /// @brief the stamps of each generated index in the current read.
    std::map<synnax::channel::Key, std::vector<x::telem::TimeStamp>> stamps;

    /// @brief adds the value of each channel at ts, skipping every channel on the index
    /// of a channel with a bad value.
    void
    collect(const UA_ReadResponse &res, const x::telem::TimeStamp ts, Problems &q) {
        std::vector<std::optional<x::telem::SampleValue>> values;
        values.reserve(this->cfg.channels.size());
        std::set<synnax::channel::Key> skipped;
        // The structure at each place in res, decoded once for every field channel on
        // its node.
        std::vector<std::optional<std::pair<types::Variant, x::errors::Error>>>
            structures;
        for (std::size_t i = 0; i < this->cfg.channels.size(); ++i) {
            auto sample = this->value<x::telem::SampleValue>(
                i,
                this->result(res, i),
                &q,
                [&](const UA_Variant &v) {
                    if (structures.empty()) structures.resize(res.resultsSize);
                    // A channel without a place reads a bad value, which never decodes.
                    return this
                        ->to_sample(i, v, structures[this->positions[i].value()]);
                }
            );
            if (!sample.has_value()) sample = this->fill(i);
            if (!sample.has_value()) skipped.insert(this->cfg.indexes[i]);
            values.push_back(std::move(sample));
        }
        for (std::size_t i = 0; i < values.size(); ++i) {
            const auto &ch = this->cfg.channels[i]->ch;
            const auto &value = values[i];
            if (ch.is_index && value.has_value() && !skipped.contains(ch.key) &&
                !this->accept(ch, {std::get<x::telem::TimeStamp>(*value)}, q))
                skipped.insert(ch.key);
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
        SampledReadTaskSource(std::move(client), std::move(pool), std::move(cfg)),
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
        Problems problems;
        // A failed read drops the block, so the stamps that it accepted never write.
        const auto written = this->last;
        for (std::size_t i = 0; i < this->cfg.samples_per_chan; i++) {
            const auto start = ::x::telem::TimeStamp::now();
            auto [ua_res, read_err] = this->read_values();
            if (read_err) {
                this->last = written;
                res.error = read_err;
                return res;
            }
            const auto end = ::x::telem::TimeStamp::now();
            this->collect(
                ua_res,
                ::x::telem::TimeStamp::midpoint(start, end),
                problems
            );
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
        res.warning = problems.warning();
        return res;
    }
};

/// @brief how long an On change read waits for changes.
const auto CHANGE_WAIT = 100 * x::telem::MILLISECOND;
/// @brief the most history values that one request reads for each node.
constexpr UA_UInt32 HISTORY_VALUES_PER_NODE = 1000;

/// @returns the time of value: its source time, or its server time when it has none, or
/// the time now when it has neither.
inline x::telem::TimeStamp change_time(const UA_DataValue &value) {
    if (value.hasSourceTimestamp)
        if (const auto [ts, err] = telem::to_timestamp(value.sourceTimestamp); !err)
            return ts;
    if (value.hasServerTimestamp)
        if (const auto [ts, err] = telem::to_timestamp(value.serverTimestamp); !err)
            return ts;
    return x::telem::TimeStamp::now();
}

/// @returns the time of value that a history read compares with its span.
inline UA_DateTime history_time(const UA_DataValue &value) {
    return value.hasSourceTimestamp ? value.sourceTimestamp : value.serverTimestamp;
}

/// @brief reads the changes of each node from a subscription, at the time of each
/// change, on an index that only its channel writes.
///
/// A change writes only when its time is after the last one on its index. The first
/// change of each node after a subscribe repeats the value that the server has, as does
/// history, so the source drops such a change without a warning when it is not after
/// the last one. With history backfill, the source reads the history of each node that
/// allows it from the last stored value to the first change of the node, one page per
/// read, and holds the changes of the node until its history is written.
class OnChangeReadTaskSource final : public BaseReadTaskSource {
    struct Change {
        x::telem::TimeStamp time;
        x::telem::SampleValue value;
        /// @brief whether the change may repeat a value that the task wrote before.
        bool replayed;
    };

    /// @brief the changes of each channel since the last read, from the subscription
    /// and from history.
    std::vector<std::vector<Change>> live;
    std::vector<std::vector<Change>> history;
    /// @brief the problems of the changes since the last read.
    Problems problems;
    /// @brief the channels whose nodes the subscription does not monitor, which each
    /// read reports again.
    Problems unmonitored;
    /// @brief the history that the subscription failed to read, which each read reports
    /// again.
    Problems history_problems;
    /// @brief the intervals that the server chose. The sampling interval is the longest
    /// of any monitored node.
    connection::Intervals granted{};
    /// @brief the node of each monitored item. Borrows the node IDs of cfg.
    std::vector<UA_NodeId> nodes;
    /// @brief the channels of each monitored item, by their place in cfg.channels.
    std::vector<std::vector<std::size_t>> readers;
    /// @brief the error of each monitored item that the server does not monitor.
    std::vector<x::errors::Error> failed;
    /// @brief whether each monitored item has yet to deliver a change.
    std::vector<bool> fresh;
    /// @brief the time from which each monitored item reads history next, or nothing
    /// when it reads no more.
    std::vector<std::optional<UA_DateTime>> backfill;
    /// @brief the monitored item of each channel, by place in nodes, or nothing when
    /// the server lacks the channel's namespace.
    std::vector<std::optional<std::size_t>> items;
    /// @brief where the history of each monitored item ends: the time of its first
    /// change, which holds the value that the server had when it began to monitor the
    /// node, or the later time of the value that retry_fields reads. Nothing until one
    /// of them arrives.
    std::vector<std::optional<UA_DateTime>> history_end;
    /// @brief when the source last tried to subscribe to what the subscription lacks.
    x::telem::TimeStamp retried{};

    /// @returns the value of channel i in value, or nothing when the value is bad or
    /// fails to convert. Shares decoded with the other channels of the node. Adds the
    /// problems of the value to problems when it is set.
    std::optional<x::telem::SampleValue> sample(
        const std::size_t i,
        const UA_DataValue &value,
        std::optional<std::pair<types::Variant, x::errors::Error>> &decoded,
        Problems *problems
    ) const {
        return this->value<x::telem::SampleValue>(
            i,
            value,
            problems,
            [&](const UA_Variant &v) { return this->to_sample(i, v, decoded); }
        );
    }

    /// @returns the intervals that the task asks the server for.
    [[nodiscard]] connection::Intervals asked() const {
        return {
            .sampling = this->cfg.sample_rate.period(),
            .publishing = this->cfg.stream_rate.period(),
        };
    }

    /// @returns a description of the intervals that the server chose when they are
    /// longer than the task asks, or an empty string.
    [[nodiscard]] std::string revised() const {
        const auto asked = this->asked();
        const auto &got = this->granted;
        if (got.sampling <= asked.sampling && got.publishing <= asked.publishing)
            return "";
        return "The server samples each node every " + got.sampling.to_string() +
               " and sends changes every " + got.publishing.to_string() +
               ", not every " + asked.sampling.to_string() + " and " +
               asked.publishing.to_string() + " as the task asks.";
    }

    void handle(const std::size_t item, const UA_DataValue &value) {
        const bool replayed = this->fresh[item];
        this->fresh[item] = false;
        if (replayed) this->history_end[item] = history_time(value);
        const auto time = change_time(value);
        std::optional<std::pair<types::Variant, x::errors::Error>> decoded;
        for (const auto i: this->readers[item])
            if (auto s = this->sample(i, value, decoded, &this->problems);
                s.has_value())
                this->live[i].push_back({time, std::move(*s), replayed});
    }

    /// @brief subscribes to the nodes of the channels on the current connection.
    x::errors::Error subscribe() {
        const auto &namespaces = this->connection.server().namespaces;
        this->nodes.clear();
        this->readers.clear();
        this->items.assign(this->cfg.channels.size(), std::nullopt);
        this->history_problems = Problems();
        std::map<std::string, std::size_t> places;
        for (std::size_t i = 0; i < this->cfg.channels.size(); ++i) {
            const auto id = this->cfg.channels[i]->node.resolve(namespaces);
            if (!id.has_value()) continue;
            const auto [it, added] = places.try_emplace(
                types::NodeId::to_string(id.value()),
                this->nodes.size()
            );
            if (added) {
                this->nodes.push_back(id.value());
                this->readers.emplace_back();
            }
            this->readers[it->second].push_back(i);
            this->items[i] = it->second;
        }
        this->failed.assign(this->nodes.size(), x::errors::NIL);
        this->fresh.assign(this->nodes.size(), true);
        this->backfill.assign(this->nodes.size(), std::nullopt);
        this->history_end.assign(this->nodes.size(), std::nullopt);
        this->retried = x::telem::TimeStamp::now();
        auto [monitored, err] = this->connection.subscribe(
            this->nodes,
            this->asked(),
            [this](const std::size_t item, const UA_DataValue &value) {
                this->handle(item, value);
            }
        );
        if (err) return err;
        this->failed = std::move(monitored.errors);
        this->list_unmonitored();
        this->granted = monitored.intervals;
        if (!this->cfg.history_backfill_enabled) return x::errors::NIL;
        std::vector<std::size_t> all(this->nodes.size());
        // NOLINTNEXTLINE(modernize-use-ranges): macOS libc++ has no std::ranges::iota.
        std::iota(all.begin(), all.end(), 0);
        return this->start_backfill(all);
    }

    /// @brief reads the structure types again when a field channel failed, then reads
    /// the value of each node whose field channels now decode, since a node that does
    /// not change sends no value to decode again. With history backfill, it also reads
    /// the history of each such node up to that value, which holds the values that
    /// failed to decode.
    /// @returns only an error that loses the connection.
    x::errors::Error retry_fields() {
        std::vector<bool> failing(this->fields.size());
        for (std::size_t i = 0; i < this->fields.size(); ++i)
            failing[i] = static_cast<bool>(this->fields[i].second);
        if (auto err = this->retry_types()) return err;
        std::map<std::size_t, std::vector<std::size_t>> fixed;
        for (std::size_t i = 0; i < failing.size(); ++i)
            if (const auto &item = this->items[i];
                failing[i] && !this->fields[i].second && item.has_value() &&
                !this->failed[item.value()])
                fixed[item.value()].push_back(i);
        if (fixed.empty()) return x::errors::NIL;
        types::ReadRequestBuilder builder;
        for (const auto &[item, _]: fixed)
            builder.add_node(this->nodes[item]);
        auto req = builder.build();
        req.timestampsToReturn = UA_TIMESTAMPSTORETURN_BOTH;
        auto [res, err] = this->connection.read(req);
        if (err.matches(errors::UNREACHABLE)) return err;
        if (err) {
            this->problems.add_bad(
                "Failed to read the nodes of the fields that now decode: " + err.data +
                "."
            );
            return x::errors::NIL;
        }
        std::vector<std::size_t> recovered;
        std::size_t k = 0;
        for (const auto &[item, channels]: fixed) {
            const auto &value = res.get().results[k++];
            const auto time = change_time(value);
            std::optional<std::pair<types::Variant, x::errors::Error>> decoded;
            for (const auto i: channels)
                if (auto s = this->sample(i, value, decoded, &this->problems);
                    s.has_value())
                    this->live[i].push_back({time, std::move(*s), true});
            const auto read_at = history_time(value);
            auto &end = this->history_end[item];
            end = std::max(end.value_or(read_at), read_at);
            recovered.push_back(item);
        }
        if (!this->cfg.history_backfill_enabled) return x::errors::NIL;
        return this->start_backfill(recovered);
    }

    /// @returns the problems of the field channels of monitored nodes whose structure
    /// types failed to read, which each read reports again.
    [[nodiscard]] Problems undecodable() const {
        Problems found;
        for (std::size_t i = 0; i < this->cfg.channels.size(); ++i) {
            const auto &item = this->items[i];
            const auto &err = this->fields[i].second;
            if (err && item.has_value() && !this->failed[item.value()])
                found.add_bad(this->cfg.channels[i]->ch.name + ": " + err.data + ".");
        }
        return found;
    }

    /// @brief lists the channels whose nodes the subscription does not monitor.
    void list_unmonitored() {
        this->unmonitored = Problems();
        for (std::size_t i = 0; i < this->cfg.channels.size(); ++i) {
            const auto &item = this->items[i];
            if (!item.has_value())
                this->unmonitored.add_unmonitored(this->missing_namespace(i));
            else if (const auto &err = this->failed[item.value()])
                this->unmonitored.add_unmonitored(
                    this->cfg.channels[i]->ch.name + ": " + err.data + "."
                );
        }
    }

    /// @brief when connection::RETRY_INTERVAL has passed, subscribes again when the
    /// server added a namespace that a channel lacks, and tries again to monitor each
    /// node that the server did not monitor.
    x::errors::Error retry() {
        const bool unresolved = std::ranges::any_of(this->items, [](const auto &item) {
            return !item.has_value();
        });
        const bool failing = std::ranges::any_of(this->failed, [](const auto &err) {
            return static_cast<bool>(err);
        });
        if (!unresolved && !failing) return x::errors::NIL;
        const auto now = x::telem::TimeStamp::now();
        if (now - this->retried < connection::RETRY_INTERVAL) return x::errors::NIL;
        this->retried = now;
        if (unresolved) {
            types::ReadRequestBuilder builder;
            builder.add_node(UA_NODEID_NUMERIC(0, UA_NS0ID_SERVER_NAMESPACEARRAY));
            auto [res, err] = this->connection.read(builder.build());
            if (err) return err;
            auto [changed, ns_err] = this->connection.update_namespaces(
                res.get().results[0]
            );
            if (ns_err) return ns_err;
            if (changed) return this->use(std::move(this->connection));
        }
        if (!failing) return x::errors::NIL;
        std::vector<UA_NodeId> retried_nodes;
        std::vector<std::size_t> retried_items;
        for (std::size_t item = 0; item < this->failed.size(); ++item)
            if (this->failed[item]) {
                retried_nodes.push_back(this->nodes[item]);
                retried_items.push_back(item);
            }
        auto [monitored, err] = this->connection.monitor(retried_nodes, retried_items);
        if (err.matches(errors::UNREACHABLE)) return err;
        std::vector<std::size_t> recovered;
        for (std::size_t k = 0; k < retried_items.size(); ++k) {
            const auto item = retried_items[k];
            // A failed request leaves its nodes and the ones after it unmonitored.
            this->failed[item] = k < monitored.errors.size() ? monitored.errors[k]
                                                             : err;
            if (this->failed[item]) continue;
            recovered.push_back(item);
        }
        if (!recovered.empty())
            this->granted.sampling = std::max(
                this->granted.sampling,
                monitored.intervals.sampling
            );
        this->list_unmonitored();
        if (!this->cfg.history_backfill_enabled) return x::errors::NIL;
        return this->start_backfill(recovered);
    }

    /// @brief starts the backfill of each item that the server monitors, whose access
    /// level allows history reads, and whose channels store values.
    x::errors::Error start_backfill(const std::vector<std::size_t> &items) {
        if (items.empty()) return x::errors::NIL;
        types::ReadRequestBuilder builder;
        for (const auto item: items)
            builder.add_node(this->nodes[item], UA_ATTRIBUTEID_ACCESSLEVEL);
        auto [res, err] = this->connection.read(builder.build());
        if (err.matches(errors::UNREACHABLE)) return err;
        if (err) {
            this->history_problems.add_history(
                "Failed to read the access levels of the nodes: " + err.data + "."
            );
            return x::errors::NIL;
        }
        for (std::size_t k = 0; k < items.size(); ++k) {
            const auto item = items[k];
            const auto &level = res.get().results[k].value;
            if (this->failed[item] ||
                !UA_Variant_hasScalarType(&level, &UA_TYPES[UA_TYPES_BYTE]) ||
                (*static_cast<const UA_Byte *>(level.data) &
                 UA_ACCESSLEVELMASK_HISTORYREAD) == 0)
                continue;
            auto from = x::telem::TimeStamp::max();
            // A channel that stores nothing yet has no gap to fill.
            for (const auto i: this->readers[item])
                if (const auto stored = this->last[this->cfg.indexes[i]];
                    stored != x::telem::TimeStamp::min())
                    from = std::min(from, stored);
            if (from == x::telem::TimeStamp::max()) continue;
            this->backfill[item] = telem::to_date_time(from) + 1;
        }
        return x::errors::NIL;
    }

    /// @brief reads the next page of history of each node that backfills and has
    /// delivered its first change. Nodes that read the same span of time share a
    /// request.
    x::errors::Error read_history() {
        std::map<std::pair<UA_DateTime, UA_DateTime>, std::vector<std::size_t>> spans;
        for (std::size_t item = 0; item < this->backfill.size(); ++item) {
            auto &from = this->backfill[item];
            const auto &end = this->history_end[item];
            if (!from.has_value() || !end.has_value()) continue;
            // The first change is not after the last stored value, so no history is
            // missing.
            if (end.value() <= from.value())
                from = std::nullopt;
            else
                spans[{from.value(), end.value()}].push_back(item);
        }
        for (const auto &[span, pending]: spans)
            if (auto err = this->read_page(span.first, span.second, pending))
                return err;
        return x::errors::NIL;
    }

    /// @brief reads the next page of history of each item in pending, from start to
    /// end.
    x::errors::Error read_page(
        const UA_DateTime start,
        const UA_DateTime end,
        const std::vector<std::size_t> &pending
    ) {
        UA_ReadRawModifiedDetails details;
        UA_ReadRawModifiedDetails_init(&details);
        details.startTime = start;
        details.endTime = end;
        details.numValuesPerNode = HISTORY_VALUES_PER_NODE;
        std::vector<UA_HistoryReadValueId> ids(pending.size());
        for (std::size_t k = 0; k < pending.size(); ++k) {
            UA_HistoryReadValueId_init(&ids[k]);
            ids[k].nodeId = this->nodes[pending[k]];
        }
        UA_HistoryReadRequest req;
        UA_HistoryReadRequest_init(&req);
        UA_ExtensionObject_setValueNoDelete(
            &req.historyReadDetails,
            &details,
            &UA_TYPES[UA_TYPES_READRAWMODIFIEDDETAILS]
        );
        req.timestampsToReturn = UA_TIMESTAMPSTORETURN_BOTH;
        req.nodesToRead = ids.data();
        req.nodesToReadSize = ids.size();
        auto [res, err] = this->connection.read_history(req);
        if (err.matches(errors::UNREACHABLE)) return err;
        if (err) {
            for (const auto item: pending)
                this->backfill[item] = std::nullopt;
            this->history_problems.add_history(
                "Failed to read the history of the nodes: " + err.data + "."
            );
            return x::errors::NIL;
        }
        for (std::size_t k = 0; k < pending.size(); ++k)
            this->take_history(pending[k], res.get().results[k]);
        return x::errors::NIL;
    }

    /// @brief adds the values of result to the history of the channels of item, and
    /// moves the backfill of item past them.
    void take_history(const std::size_t item, const UA_HistoryReadResult &result) {
        auto &from = this->backfill[item];
        if (UA_StatusCode_isBad(result.statusCode)) {
            from = std::nullopt;
            const auto err = errors::parse(result.statusCode);
            for (const auto i: this->readers[item])
                this->history_problems.add_history(
                    this->cfg.channels[i]->ch.name + ": " + err.data + "."
                );
            return;
        }
        const auto &content = result.historyData.content.decoded;
        const auto *data = result.historyData.encoding >= UA_EXTENSIONOBJECT_DECODED &&
                                   content.type == &UA_TYPES[UA_TYPES_HISTORYDATA]
                             ? static_cast<const UA_HistoryData *>(content.data)
                             : nullptr;
        UA_DateTime next = from.value();
        for (std::size_t j = 0; data != nullptr && j < data->dataValuesSize; ++j) {
            const auto &value = data->dataValues[j];
            next = std::max(next, history_time(value) + 1);
            const auto time = change_time(value);
            std::optional<std::pair<types::Variant, x::errors::Error>> decoded;
            for (const auto i: this->readers[item])
                if (auto s = this->sample(i, value, decoded, nullptr); s.has_value())
                    this->history[i].push_back({time, std::move(*s), true});
        }
        // A server that returns no values with a continuation point would never end.
        if (result.continuationPoint.length == 0 || next == from.value())
            from = std::nullopt;
        else
            from = next;
    }

    /// @brief moves the changes of each channel into fr: its history, then its live
    /// changes once its history is written.
    void take(x::telem::Frame &fr) {
        for (std::size_t i = 0; i < this->cfg.channels.size(); ++i) {
            const auto &ch = *this->cfg.channels[i];
            const auto idx = this->cfg.indexes[i];
            auto &last = this->last[idx];
            std::vector<x::telem::SampleValue> samples;
            std::vector<x::telem::TimeStamp> stamps;
            const auto write = [&](std::vector<Change> &changes) {
                for (auto &c: changes) {
                    if (c.time <= last) {
                        if (!c.replayed && c.time < last)
                            this->problems.add_rejected(
                                "Rejected a value of " + ch.ch.name +
                                ": its timestamp is not after the last one written."
                            );
                        continue;
                    }
                    last = c.time;
                    samples.push_back(std::move(c.value));
                    stamps.push_back(c.time);
                }
                changes.clear();
            };
            write(this->history[i]);
            const auto item = this->items[i];
            if (!item.has_value() || !this->backfill[item.value()].has_value())
                write(this->live[i]);
            if (samples.empty()) continue;
            fr.emplace(ch.synnax_key, telem::to_series(ch.ch.data_type, samples));
            fr.emplace(idx, x::telem::Series(stamps));
        }
    }

    x::errors::Error use(connection::Pool::Connection conn) override {
        this->connection = std::move(conn);
        auto err = this->read_types();
        if (!err) err = this->subscribe();
        if (err) this->connection = connection::Pool::Connection(nullptr, nullptr, "");
        return err;
    }

public:
    OnChangeReadTaskSource(
        std::shared_ptr<synnax::Synnax> client,
        std::shared_ptr<connection::Pool> pool,
        ReadTaskConfig cfg
    ):
        BaseReadTaskSource(std::move(client), std::move(pool), std::move(cfg)),
        live(this->cfg.channels.size()),
        history(this->cfg.channels.size()) {}

    OnChangeReadTaskSource(const OnChangeReadTaskSource &) = delete;
    OnChangeReadTaskSource &operator=(const OnChangeReadTaskSource &) = delete;
    OnChangeReadTaskSource(OnChangeReadTaskSource &&) = delete;
    OnChangeReadTaskSource &operator=(OnChangeReadTaskSource &&) = delete;

    /// @brief closes the connection first, since a closing connection delivers changes
    /// to members that the base class outlives.
    ~OnChangeReadTaskSource() override {
        this->connection = connection::Pool::Connection(nullptr, nullptr, "");
    }

    x::errors::Error start() override {
        for (auto &c: this->live)
            c.clear();
        for (auto &c: this->history)
            c.clear();
        this->problems = Problems();
        return BaseReadTaskSource::start();
    }

    /// @brief reads a page of history while a backfill runs, then waits up to
    /// CHANGE_WAIT for changes and writes them. Connects and subscribes first when the
    /// task has no connection, and drops the connection when it fails, so the next read
    /// subscribes again.
    common::ReadResult read(x::breaker::Breaker &, x::telem::Frame &fr) override {
        common::ReadResult res;
        fr.clear();
        if (!this->connection) {
            if (this->start_err) {
                res.error = std::exchange(this->start_err, x::errors::NIL);
                return res;
            }
            if (res.error = this->connect(); res.error) return res;
        }
        res.error = this->read_history();
        if (!res.error) res.error = this->connection.iterate(CHANGE_WAIT);
        // The first value of a node that retry monitors decodes with the types.
        if (!res.error) res.error = this->retry_fields();
        if (!res.error) res.error = this->retry();
        if (res.error) {
            this->connection = connection::Pool::Connection(nullptr, nullptr, "");
            return res;
        }
        this->take(fr);
        auto reported = this->unmonitored;
        reported.merge(this->undecodable());
        reported.merge(this->history_problems);
        reported.merge(this->problems);
        this->problems = Problems();
        res.warning = reported.warning();
        res.description = this->revised();
        return res;
    }
};
}
