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
#include <memory>
#include <optional>
#include <set>
#include <string>
#include <tuple>
#include <utility>
#include <vector>

#include "absl/log/log.h"
#include "open62541/types.h"

#include "client/cpp/errors/errors.h"
#include "client/cpp/opcua/json.gen.h"
#include "client/cpp/synnax.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/strings/strings.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/telem.h"

#include "driver/common/read_task.h"
#include "driver/errors/errors.h"
#include "driver/opcua/connection/connection.h"
#include "driver/opcua/device/device.h"
#include "driver/opcua/errors/errors.h"
#include "driver/opcua/telem/telem.h"
#include "driver/opcua/types/types.h"

namespace driver::opcua {
/// @brief the fields that every event records, in the order of the select clause, with
/// their keys in the JSON sample.
const std::vector<std::pair<std::string, std::string>> STANDARD_EVENT_FIELDS = {
    {"EventId", "event_id"},
    {"EventType", "event_type"},
    {"SourceName", "source_name"},
    {"Time", "time"},
    {"Message", "message"},
    {"Severity", "severity"},
};
constexpr std::size_t EVENT_TYPE_FIELD = 1;
constexpr std::size_t TIME_FIELD = 3;

/// @brief how long a read waits for events.
const auto EVENT_WAIT = 100 * x::telem::MILLISECOND;
/// @brief how often a task checks that the server still answers, since a server that
/// stops answering sends no notice to a subscription.
const auto LIVENESS_INTERVAL = 5 * x::telem::SECOND;

/// @brief the configuration of an OPC UA event task.
struct EventTaskConfig {
    /// @brief the device representing the OPC UA server.
    const std::string device_key;
    /// @brief whether the task streams events without storing them.
    const bool data_saving_disabled;
    /// @brief whether the task starts each time the Driver starts.
    const bool auto_start;
    /// @brief the JSON channel that each event writes to, on its index.
    synnax::channel::Channel channel;
    /// @brief the objects whose events the task records. Empty records the events of
    /// the Server object.
    std::vector<types::NodeId> notifiers;
    /// @brief the event types that the task keeps, with their subtypes. Empty keeps
    /// every event.
    std::vector<types::NodeId> event_types;
    /// @brief the fields that the task records beyond STANDARD_EVENT_FIELDS.
    std::vector<::synnax::opcua::EventField> fields;
    /// @brief the browse path of each field.
    std::vector<types::BrowsePath> paths;

    EventTaskConfig(EventTaskConfig &&other) noexcept = default;
    EventTaskConfig(const EventTaskConfig &) = delete;
    EventTaskConfig &operator=(const EventTaskConfig &) = delete;

    EventTaskConfig(
        const std::shared_ptr<synnax::Synnax> &client,
        x::json::Parser &parser,
        const ::synnax::opcua::EventConfig &parsed
    ):
        device_key(parsed.device),
        data_saving_disabled(parsed.data_saving_disabled),
        auto_start(parsed.auto_start),
        fields(parsed.fields) {
        const auto parse_ids = [&](const std::string &path,
                                   const std::vector<std::string> &ids,
                                   std::vector<types::NodeId> &out) {
            for (std::size_t i = 0; i < ids.size(); ++i) {
                auto [id, err] = types::NodeId::parse(ids[i]);
                if (err)
                    parser.field_err(path + "." + std::to_string(i), err.data);
                else
                    out.push_back(std::move(id));
            }
        };
        parse_ids("notifiers", parsed.notifiers, this->notifiers);
        parse_ids("event_types", parsed.event_types, this->event_types);
        std::set<std::string> standard;
        for (const auto &[_, key]: STANDARD_EVENT_FIELDS)
            standard.insert(key);
        std::set<std::string> names;
        for (std::size_t i = 0; i < this->fields.size(); ++i) {
            const auto &f = this->fields[i];
            const auto prefix = "fields." + std::to_string(i) + ".";
            if (f.name.empty())
                parser.field_err(prefix + "name", "Field must have a name");
            else if (standard.contains(f.name))
                parser.field_err(prefix + "name", "A standard field has this name");
            else if (!names.insert(f.name).second)
                parser.field_err(prefix + "name", "Another field has this name");
            auto [path, err] = types::BrowsePath::parse(f.browse_path);
            if (err) parser.field_err(prefix + "browse_path", err.data);
            this->paths.push_back(std::move(path));
        }
        if (parsed.channel == 0) {
            parser.field_err("channel", "Task must have a channel");
            return;
        }
        if (const auto err = device::retrieve_connection(*client, this->device_key)
                                 .second) {
            parser.field_err("device", common::config_err_description(err));
            return;
        }
        auto [ch, err] = client->channels.retrieve(parsed.channel);
        if (err) {
            parser.field_err("channel", "Failed to retrieve channel: " + err.data);
            return;
        }
        if (ch.data_type != x::telem::JSON_T)
            parser.field_err("channel", "Channel must have the json data type");
        else if (ch.index == 0)
            parser.field_err("channel", "Channel must have an index");
        this->channel = std::move(ch);
    }

    [[nodiscard]] synnax::framer::WriterConfig writer_config() const {
        return {
            .channels = {this->channel.key, this->channel.index},
            .mode = common::data_saving_writer_mode(this->data_saving_disabled),
        };
    }

    static std::pair<EventTaskConfig, x::errors::Error> parse(
        const std::shared_ptr<synnax::Synnax> &client,
        const synnax::task::Task &task
    ) {
        auto parser = x::json::Parser(task.config);
        const auto parsed = ::synnax::opcua::EventConfig::parse(parser);
        EventTaskConfig cfg(client, parser, parsed);
        return {std::move(cfg), parser.error()};
    }
};

/// @brief records the events of an OPC UA server to a JSON channel.
///
/// An event writes at its Time, or at the time the Driver received it when it has none.
/// Each event must write after the last sample on the index. At start, the task asks
/// the server to send the state of each condition again. A replayed event at or before
/// the latest event that the task had when the replay began is one that the task has,
/// so the task drops it. Another event at or before the last sample writes 1 ns after
/// it.
class EventTaskSource final : public common::Source {
    /// @brief an event that the subscription delivered.
    struct Event {
        x::json::json sample;
        x::telem::TimeStamp time;
    };

    const EventTaskConfig cfg;
    std::shared_ptr<synnax::Synnax> client;
    device::Connector connector;
    /// @brief the events that the subscription delivered since the last read. It
    /// outlives the connection, since the connection delivers events as it closes.
    std::vector<Event> events;
    /// @brief the namespace URIs of the server, which name the event types of events.
    /// Set before each subscribe, since the server can deliver events during it.
    std::vector<std::string> namespaces;
    /// @brief the key in the JSON sample of each field that the filter selects, in
    /// order. Set before each subscribe, with the filter.
    std::vector<std::string> keys;
    /// @brief the filter of the subscription, which each monitored item shares.
    std::unique_ptr<const types::EventFilter> filter;
    /// @brief the notifiers that the server did not monitor, with the error of each.
    std::vector<std::pair<UA_NodeId, x::errors::Error>> unmonitored;
    /// @brief a problem for each notifier, event type, or field in a namespace that the
    /// server does not have.
    std::vector<std::string> unresolved;
    /// @brief the number of condition replays in progress.
    int replays = 0;
    /// @brief whether the server asked the task to replay the conditions.
    bool replay_required = false;
    /// @brief the last time on the index of the channel.
    x::telem::TimeStamp last{};
    /// @brief the latest event time that the task had when the replay in progress
    /// began.
    x::telem::TimeStamp replay_cutoff{};
    /// @brief when the task last checked that the server answers.
    x::telem::TimeStamp checked{};
    /// @brief when the task last tried again to record what failed.
    x::telem::TimeStamp retried{};
    /// @brief the problems with what the task fails to record.
    common::Warning warning;
    connection::Pool::Connection connection;
    /// @brief the error of the connection attempt at start, which the first read
    /// returns instead of connecting again.
    x::errors::Error start_err;

    /// @returns the value of field i of an event as JSON.
    [[nodiscard]] x::json::json
    to_json(const std::size_t i, const UA_Variant &v) const {
        if (UA_Variant_isEmpty(&v)) return nullptr;
        if (i == EVENT_TYPE_FIELD &&
            UA_Variant_hasScalarType(&v, &UA_TYPES[UA_TYPES_NODEID]))
            return types::NodeId::to_string(
                *static_cast<const UA_NodeId *>(v.data),
                this->namespaces
            );
        if (UA_Variant_hasScalarType(&v, &UA_TYPES[UA_TYPES_LOCALIZEDTEXT])) {
            const auto &text = static_cast<const UA_LocalizedText *>(v.data)->text;
            return std::string(reinterpret_cast<const char *>(text.data), text.length);
        }
        auto [text, err] = telem::to_json(v);
        if (err) return nullptr;
        return x::json::json::parse(text);
    }

    /// @returns the numeric ID of an event type in namespace 0, or 0 for another type.
    [[nodiscard]] static UA_UInt32 standard_type(const UA_Variant &v) {
        if (!UA_Variant_hasScalarType(&v, &UA_TYPES[UA_TYPES_NODEID])) return 0;
        const auto &id = *static_cast<const UA_NodeId *>(v.data);
        if (id.namespaceIndex != 0 || id.identifierType != UA_NODEIDTYPE_NUMERIC)
            return 0;
        return id.identifier.numeric;
    }

    void handle(const UA_KeyValueMap &fields) {
        switch (standard_type(fields.map[EVENT_TYPE_FIELD].value)) {
            case UA_NS0ID_REFRESHSTARTEVENTTYPE:
                if (this->replays++ == 0) this->replay_cutoff = this->latest();
                return;
            case UA_NS0ID_REFRESHENDEVENTTYPE:
                this->replays = std::max(this->replays - 1, 0);
                return;
            case UA_NS0ID_REFRESHREQUIREDEVENTTYPE:
                this->replay_required = true;
                return;
            default:
                break;
        }
        Event event{
            .sample = x::json::json::object(),
            .time = x::telem::TimeStamp::now(),
        };
        const auto &time = fields.map[TIME_FIELD].value;
        if (UA_Variant_hasScalarType(&time, &UA_TYPES[UA_TYPES_DATETIME]))
            if (const auto [ts, err] = telem::to_timestamp(
                    *static_cast<const UA_DateTime *>(time.data)
                );
                !err)
                event.time = ts;
        if (this->replays > 0 && event.time <= this->replay_cutoff) return;
        for (std::size_t i = 0; i < fields.mapSize; ++i)
            event.sample[this->keys[i]] = to_json(i, fields.map[i].value);
        this->events.push_back(std::move(event));
    }

    /// @brief asks the server to send the state of each condition again. A server
    /// without conditions refuses, which is normal.
    /// @returns errors::UNREACHABLE when the server did not answer.
    x::errors::Error replay() {
        this->replay_required = false;
        UA_UInt32 id = this->connection.subscription_id();
        UA_Variant arg;
        UA_Variant_setScalar(&arg, &id, &UA_TYPES[UA_TYPES_UINT32]);
        UA_CallMethodRequest req;
        UA_CallMethodRequest_init(&req);
        req.objectId = UA_NODEID_NUMERIC(0, UA_NS0ID_CONDITIONTYPE);
        req.methodId = UA_NODEID_NUMERIC(0, UA_NS0ID_CONDITIONTYPE_CONDITIONREFRESH);
        req.inputArguments = &arg;
        req.inputArgumentsSize = 1;
        auto [res, err] = this->connection.call(req);
        if (err.matches(opcua::errors::UNREACHABLE)) return err;
        if (!err) err = opcua::errors::parse(res.get().results[0].statusCode);
        if (err) VLOG(1) << "[opc.event] the server did not replay conditions: " << err;
        return x::errors::NIL;
    }

    /// @brief sets the warning from the problems with what the task fails to record.
    void warn() {
        auto problems = this->unresolved;
        for (const auto &[id, err]: this->unmonitored)
            problems.push_back(types::NodeId::to_string(id) + ": " + err.data);
        this->warning = {};
        if (!problems.empty())
            this->warning = {
                .message = "Failed to record some events",
                .description = x::strings::join(problems, " "),
            };
    }

    /// @brief subscribes to the events of the notifiers on conn, and reads on conn from
    /// now on.
    x::errors::Error subscribe(connection::Pool::Connection conn) {
        this->namespaces = conn.server().namespaces;
        const auto &namespaces = this->namespaces;
        this->unresolved.clear();
        const auto missing = [this](const std::string &name) {
            this->unresolved.push_back(
                name + " is in a namespace that the server does not have."
            );
        };
        std::vector<UA_NodeId> notifiers;
        for (const auto &n: this->cfg.notifiers)
            if (const auto id = n.resolve(namespaces); id.has_value())
                notifiers.push_back(*id);
            else
                missing(n.to_string());
        if (this->cfg.notifiers.empty())
            notifiers.push_back(UA_NODEID_NUMERIC(0, UA_NS0ID_SERVER));
        std::vector<UA_NodeId> types;
        for (const auto &t: this->cfg.event_types)
            if (const auto id = t.resolve(namespaces); id.has_value())
                types.push_back(*id);
            else
                missing(t.to_string());
        // The replay markers must pass a filter that keeps only some types.
        if (!this->cfg.event_types.empty())
            for (const auto marker:
                 {UA_NS0ID_REFRESHSTARTEVENTTYPE,
                  UA_NS0ID_REFRESHENDEVENTTYPE,
                  UA_NS0ID_REFRESHREQUIREDEVENTTYPE})
                types.push_back(UA_NODEID_NUMERIC(0, marker));
        std::vector<std::vector<UA_QualifiedName>> paths;
        this->keys.clear();
        for (const auto &[name, key]: STANDARD_EVENT_FIELDS) {
            paths.push_back({UA_QUALIFIEDNAME(0, const_cast<char *>(name.c_str()))});
            this->keys.push_back(key);
        }
        for (std::size_t i = 0; i < this->cfg.fields.size(); ++i) {
            const auto &f = this->cfg.fields[i];
            auto names = this->cfg.paths[i].resolve(namespaces);
            if (!names.has_value()) {
                missing("Field " + f.name);
                continue;
            }
            paths.push_back(std::move(*names));
            this->keys.push_back(f.name);
        }
        this->filter = std::make_unique<const types::EventFilter>(paths, types);
        this->replays = 0;
        auto [results, err] = conn.subscribe(
            notifiers,
            this->filter->get(),
            [this](const UA_KeyValueMap &fields) { this->handle(fields); }
        );
        if (err) return err;
        this->unmonitored.clear();
        for (std::size_t i = 0; i < results.size(); ++i)
            if (results[i]) this->unmonitored.emplace_back(notifiers[i], results[i]);
        this->warn();
        this->connection = std::move(conn);
        this->checked = x::telem::TimeStamp::now();
        this->retried = this->checked;
        if (const auto err = this->replay()) {
            this->connection = connection::Pool::Connection(nullptr, nullptr, "");
            return err;
        }
        return x::errors::NIL;
    }

    x::errors::Error connect() {
        auto [c, err] = this->connector.connect();
        if (err) return err;
        return this->subscribe(std::move(c));
    }

    /// @brief when connection::RETRY_INTERVAL has passed, tries again to record what
    /// failed: a notifier that the server did not monitor, or a notifier, event type,
    /// or field in a namespace that the server does not have. Subscribes again when the
    /// namespaces of the server changed, since the subscription can then name nodes by
    /// stale indexes. Replays the conditions when the server monitors a notifier that
    /// it refused.
    /// @returns an error that loses the connection, or the error of the new
    /// subscription.
    x::errors::Error retry() {
        if (this->unresolved.empty() && this->unmonitored.empty())
            return x::errors::NIL;
        const auto now = x::telem::TimeStamp::now();
        if (now - this->retried < connection::RETRY_INTERVAL) return x::errors::NIL;
        this->retried = now;
        if (!this->unresolved.empty()) {
            types::ReadRequestBuilder builder;
            builder.add_node(UA_NODEID_NUMERIC(0, UA_NS0ID_SERVER_NAMESPACEARRAY));
            auto [res, err] = this->connection.read(builder.build());
            if (err) return err;
            auto [changed, ns_err] = this->connection.update_namespaces(
                res.get().results[0]
            );
            if (ns_err) return ns_err;
            if (changed) return this->subscribe(std::move(this->connection));
        }
        if (this->unmonitored.empty()) return x::errors::NIL;
        std::vector<UA_NodeId> notifiers;
        notifiers.reserve(this->unmonitored.size());
        for (const auto &[id, _]: this->unmonitored)
            notifiers.push_back(id);
        auto [results, err] = this->connection.monitor(notifiers, this->filter->get());
        if (err.matches(errors::UNREACHABLE)) return err;
        if (err) return x::errors::NIL;
        this->unmonitored.clear();
        for (std::size_t i = 0; i < results.size(); ++i)
            if (results[i]) this->unmonitored.emplace_back(notifiers[i], results[i]);
        this->warn();
        if (this->unmonitored.size() < notifiers.size()) return this->replay();
        return x::errors::NIL;
    }

    /// @brief reads the server's clock when LIVENESS_INTERVAL has passed since the last
    /// check.
    x::errors::Error check() {
        const auto now = x::telem::TimeStamp::now();
        if (now - this->checked < LIVENESS_INTERVAL) return x::errors::NIL;
        this->checked = now;
        types::ReadRequestBuilder builder;
        builder.add_node(
            UA_NODEID_NUMERIC(0, UA_NS0ID_SERVER_SERVERSTATUS_CURRENTTIME)
        );
        return this->connection.read(builder.build()).second;
    }

    /// @returns the latest time of the events that the task wrote or holds to write.
    [[nodiscard]] x::telem::TimeStamp latest() const {
        auto time = this->last;
        for (const auto &e: this->events)
            time = std::max(time, e.time);
        return time;
    }

    /// @brief moves the delivered events into fr, in order of time.
    void take(x::telem::Frame &fr) {
        std::ranges::stable_sort(this->events, {}, &Event::time);
        std::vector<x::json::json> samples;
        std::vector<x::telem::TimeStamp> stamps;
        for (auto &e: this->events) {
            if (e.time <= this->last) e.time = this->last + x::telem::NANOSECOND;
            this->last = e.time;
            samples.push_back(std::move(e.sample));
            stamps.push_back(e.time);
        }
        this->events.clear();
        if (samples.empty()) return;
        fr.emplace(this->cfg.channel.key, x::telem::Series(samples));
        fr.emplace(this->cfg.channel.index, x::telem::Series(stamps));
    }

    /// @returns the last time stored on the index of the channel, or the minimum
    /// timestamp when it stores none after 1970.
    [[nodiscard]] std::pair<x::telem::TimeStamp, x::errors::Error> last_stored() const {
        // The Core finds nothing in bounds whose span overflows a TimeSpan.
        auto [iter, err] = this->client->telem.open_iterator({
            .channels = {this->cfg.channel.index},
            .bounds = {x::telem::TimeStamp(0), x::telem::TimeStamp::max()},
            .chunk_size = 1,
        });
        if (err) return {x::telem::TimeStamp(), err};
        auto latest = x::telem::TimeStamp::min();
        auto [found, seek_err] = iter.seek_last();
        if (!seek_err && found)
            std::tie(found, seek_err) = iter.prev(synnax::framer::AUTO_SPAN);
        if (!seek_err && found)
            for (const auto &s: *iter.value.series)
                if (!s.empty())
                    latest = std::max(latest, s.at<x::telem::TimeStamp>(-1));
        const auto close_err = iter.close();
        if (seek_err) return {x::telem::TimeStamp(), seek_err};
        return {latest, close_err};
    }

public:
    EventTaskSource(
        std::shared_ptr<synnax::Synnax> client,
        std::shared_ptr<connection::Pool> pool,
        EventTaskConfig cfg
    ):
        cfg(std::move(cfg)),
        client(client),
        connector(
            std::move(client),
            std::move(pool),
            this->cfg.device_key,
            "[opc.event] "
        ),
        connection(nullptr, nullptr, "") {}

    [[nodiscard]] synnax::framer::WriterConfig writer_config() const override {
        return this->cfg.writer_config();
    }

    [[nodiscard]] std::vector<synnax::channel::Channel> channels() const override {
        return {this->cfg.channel};
    }

    /// @brief finds the last stored event, then subscribes. A server that is off or
    /// refuses Sessions does not stop the start, since the reads connect again.
    x::errors::Error start() override {
        auto [last, last_err] = this->last_stored();
        if (last_err) return last_err;
        this->last = last;
        this->events.clear();
        auto [c, err] = this->connector.start();
        if (!err) err = this->subscribe(std::move(c));
        if (err.matches(driver::errors::TEMPORARY_HARDWARE_ERROR)) {
            this->start_err = err;
            return x::errors::NIL;
        }
        return err;
    }

    x::errors::Error stop() override {
        this->connection = connection::Pool::Connection(nullptr, nullptr, "");
        this->start_err = x::errors::NIL;
        this->events.clear();
        return x::errors::NIL;
    }

    /// @brief waits up to EVENT_WAIT for events, and writes the ones that arrived.
    /// Connects and subscribes first when the task has no connection, and drops the
    /// connection when it fails, so the next read subscribes again.
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
        res.error = this->connection.iterate(EVENT_WAIT);
        if (!res.error && this->replay_required) res.error = this->replay();
        if (!res.error) res.error = this->check();
        if (!res.error) res.error = this->retry();
        if (res.error) {
            this->connection = connection::Pool::Connection(nullptr, nullptr, "");
            return res;
        }
        this->take(fr);
        res.warning = this->warning;
        return res;
    }
};
}
