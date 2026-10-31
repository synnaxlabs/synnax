// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <memory>
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/opcua/event_task.h"
#include "driver/opcua/testutil/server.h"
#include "driver/opcua/testutil/testutil.h"
#include "driver/pipeline/testutil/pipeline.h"

namespace driver::opcua {
namespace {
const std::string ENDPOINT = "opc.tcp://localhost:4864";
const std::string DEVICE_KEY = "opc_event_task_test_server_key";

/// @returns the events and stamps that the reads of source wrote, reading until count
/// events arrive or 5 seconds pass.
std::pair<std::vector<x::json::json>, std::vector<x::telem::TimeStamp>>
read_events(EventTaskSource &source, const std::size_t count) {
    std::vector<x::json::json> events;
    std::vector<x::telem::TimeStamp> stamps;
    x::breaker::Breaker breaker;
    const auto deadline = std::chrono::steady_clock::now() + std::chrono::seconds(5);
    while (events.size() < count && std::chrono::steady_clock::now() < deadline) {
        x::telem::Frame fr;
        const auto res = source.read(breaker, fr);
        EXPECT_FALSE(res.error) << res.error;
        for (const auto &[key, s]: fr)
            if (s.data_type() == x::telem::JSON_T)
                for (auto &v: s.json_values())
                    events.push_back(std::move(v));
            else
                for (std::size_t i = 0; i < s.size(); ++i)
                    stamps.push_back(s.at<x::telem::TimeStamp>(static_cast<int>(i)));
    }
    return {events, stamps};
}
}

class TestEventTask : public ::testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> client;
    std::unique_ptr<testutil::Server> server;
    std::shared_ptr<connection::Pool> pool;
    synnax::channel::Channel index;
    synnax::channel::Channel events;

    void SetUp() override {
        this->client = std::make_shared<synnax::Synnax>(new_test_client());
        this->index = ASSERT_NIL_P(this->client->channels.create(
            make_unique_channel_name("event_time"),
            x::telem::TIMESTAMP_T,
            0,
            true
        ));
        this->events = ASSERT_NIL_P(this->client->channels.create(
            make_unique_channel_name("events"),
            x::telem::JSON_T,
            this->index.key,
            false
        ));
        const auto rack = ASSERT_NIL_P(
            this->client->racks.create("opc_event_task_test_rack")
        );
        connection::Config conn_cfg;
        conn_cfg.endpoint = ENDPOINT;
        conn_cfg.security_mode = "None";
        conn_cfg.security_policy = "None";
        synnax::device::Device dev{
            .key = DEVICE_KEY,
            .rack = rack.key,
            .location = ENDPOINT,
            .make = "opc",
            .model = "OPC UA Server",
            .name = "OPC UA Event Task Test Server",
            .properties = x::json::json::object(
                {{"connection", testutil::connection_json(conn_cfg)}}
            ),
        };
        ASSERT_NIL(this->client->devices.create(dev));
        this->pool = std::make_shared<connection::Pool>(testutil::create_store());
        this->start_server();
    }

    void TearDown() override {
        if (this->server) this->server->stop();
    }

    void start_server(const testutil::Pump &pump = {}) {
        auto cfg = testutil::ServerConfig::create_default();
        cfg.port = 4864;
        cfg.pump = pump;
        this->server = std::make_unique<testutil::Server>(cfg);
        this->server->start();
        ASSERT_TRUE(this->server->wait_until_ready());
    }

    /// @returns the config of a task that records the events of notifiers.
    [[nodiscard]] x::json::json
    config(const std::vector<std::string> &notifiers = {"NS=1;S=Pump"}) const {
        return {
            {"device", DEVICE_KEY},
            {"channel", this->events.key},
            {"notifiers", notifiers},
            {"event_types", {"NS=1;S=TestEventType"}},
            {"fields",
             {{{"key", "level"}, {"name", "level"}, {"browse_path", "1:Level"}}}},
        };
    }

    /// @returns the parsed config, and the errors of the parse.
    [[nodiscard]] std::pair<std::unique_ptr<EventTaskConfig>, x::errors::Error>
    parse(const x::json::json &cfg) const {
        auto parser = x::json::Parser(cfg);
        const auto parsed = ::synnax::opcua::EventConfig::parse(parser);
        auto out = std::make_unique<EventTaskConfig>(this->client, parser, parsed);
        return {std::move(out), parser.error()};
    }

    [[nodiscard]] std::unique_ptr<EventTaskSource>
    source(const x::json::json &cfg) const {
        auto [parsed, err] = this->parse(cfg);
        EXPECT_FALSE(err) << err;
        return std::make_unique<EventTaskSource>(
            this->client,
            this->pool,
            std::move(*parsed)
        );
    }

    /// @brief stores one empty event at time on the event channel.
    void store(const x::telem::TimeStamp time) const {
        auto writer = ASSERT_NIL_P(this->client->telem.open_writer(
            synnax::framer::WriterConfig{
                .channels = {this->index.key, this->events.key},
                .start = time,
            }
        ));
        auto fr = x::telem::Frame(2);
        fr.emplace(this->index.key, x::telem::Series(time));
        fr.emplace(this->events.key, x::telem::Series(x::json::json::object()));
        ASSERT_NIL(writer.write(fr));
        ASSERT_NIL_P(writer.commit());
        ASSERT_NIL(writer.close());
    }

    /// @brief emits an event of the standard type id from the Pump object.
    void emit_marker(const UA_UInt32 id) const {
        this->server->emit(
            {.source = testutil::PUMP, .type = UA_NODEID_NUMERIC(0, id)}
        );
    }

    /// @brief emits a TestEventType event from the Pump object.
    void emit(
        const std::string &message,
        const std::optional<x::telem::TimeStamp> time = std::nullopt
    ) const {
        this->server->emit({
            .source = testutil::PUMP,
            .type = testutil::TEST_EVENT_TYPE,
            .severity = 700,
            .message = message,
            .time = time,
            .level = 2.5,
        });
    }
};

/// @brief it should write each event that passes the filter as a JSON sample at its
/// time, with the standard fields and the configured fields.
TEST_F(TestEventTask, testRecordsEvents) {
    const auto src = this->source(this->config());
    ASSERT_NIL(src->start());
    const auto time = testutil::now(-1 * x::telem::SECOND);
    this->server->emit({.source = testutil::PUMP, .message = "filtered out"});
    this->emit("overpressure", time);
    const auto [events, stamps] = read_events(*src, 1);
    ASSERT_EQ(events.size(), 1);
    ASSERT_EQ(stamps, std::vector{time});
    const auto &e = events[0];
    EXPECT_EQ(e["message"], "overpressure");
    EXPECT_EQ(e["severity"], 700);
    EXPECT_EQ(
        e["event_type"],
        "nsu=" + testutil::SERVER_APPLICATION_URI + ";s=TestEventType"
    );
    EXPECT_EQ(e["source_name"], "Pump");
    EXPECT_EQ(e["level"], 2.5);
    EXPECT_TRUE(e["event_id"].is_string());
    EXPECT_TRUE(e["time"].is_string());
    ASSERT_NIL(src->stop());
}

/// @brief it should record the events of the Server object when the task names no
/// notifiers, which include the events of every object.
TEST_F(TestEventTask, testRecordsServerEventsByDefault) {
    const auto src = this->source(this->config({}));
    ASSERT_NIL(src->start());
    this->emit("started");
    const auto [events, _] = read_events(*src, 1);
    ASSERT_EQ(events.size(), 1);
    EXPECT_EQ(events[0]["message"], "started");
    ASSERT_NIL(src->stop());
}

/// @brief it should write an event at or before the last stored sample 1 ns after it,
/// so that no event is lost.
TEST_F(TestEventTask, testWritesAfterLastStoredSample) {
    const auto stored = testutil::now();
    this->store(stored);
    const auto src = this->source(this->config());
    ASSERT_NIL(src->start());
    this->emit("late", stored - x::telem::SECOND);
    this->emit("early", stored - 2 * x::telem::SECOND);
    this->emit("new", stored + x::telem::SECOND);
    const auto [events, stamps] = read_events(*src, 3);
    ASSERT_EQ(events.size(), 3);
    EXPECT_EQ(events[0]["message"], "early");
    EXPECT_EQ(events[1]["message"], "late");
    EXPECT_EQ(events[2]["message"], "new");
    EXPECT_EQ(
        stamps,
        std::vector({
            stored + x::telem::NANOSECOND,
            stored + 2 * x::telem::NANOSECOND,
            stored + x::telem::SECOND,
        })
    );
    ASSERT_NIL(src->stop());
}

/// @brief it should skip a replayed event at or before the last stored sample, since
/// the task stored it before, and keep a new event at the same time.
TEST_F(TestEventTask, testSkipsReplayedStoredEvents) {
    const auto stored = testutil::now();
    this->store(stored);
    const auto src = this->source(this->config());
    ASSERT_NIL(src->start());
    this->emit_marker(UA_NS0ID_REFRESHSTARTEVENTTYPE);
    this->emit("replayed", stored - x::telem::SECOND);
    this->emit_marker(UA_NS0ID_REFRESHENDEVENTTYPE);
    this->emit("new", stored - x::telem::SECOND);
    const auto [events, stamps] = read_events(*src, 1);
    ASSERT_EQ(events.size(), 1);
    EXPECT_EQ(events[0]["message"], "new");
    EXPECT_EQ(stamps, std::vector{stored + x::telem::NANOSECOND});
    ASSERT_NIL(src->stop());
}

/// @brief it should keep each new replayed event when several share a time, writing
/// each later one 1 ns after the one before it.
TEST_F(TestEventTask, testKeepsReplayedEventsAtOneTime) {
    const auto src = this->source(this->config());
    ASSERT_NIL(src->start());
    const auto time = testutil::now(-1 * x::telem::SECOND);
    this->emit_marker(UA_NS0ID_REFRESHSTARTEVENTTYPE);
    this->emit("first", time);
    this->emit("second", time);
    this->emit_marker(UA_NS0ID_REFRESHENDEVENTTYPE);
    const auto [events, stamps] = read_events(*src, 2);
    ASSERT_EQ(events.size(), 2);
    EXPECT_EQ(events[0]["message"], "first");
    EXPECT_EQ(events[1]["message"], "second");
    EXPECT_EQ(stamps, std::vector({time, time + x::telem::NANOSECOND}));
    ASSERT_NIL(src->stop());
}

/// @brief it should keep a new replayed event that a second replay sends again before
/// the task writes it, and write it once.
TEST_F(TestEventTask, testKeepsReplayedEventThatReplaysAgain) {
    const auto src = this->source(this->config());
    ASSERT_NIL(src->start());
    const auto time = testutil::now(-1 * x::telem::SECOND);
    for (int i = 0; i < 2; ++i) {
        this->emit_marker(UA_NS0ID_REFRESHSTARTEVENTTYPE);
        this->emit("active", time);
        this->emit_marker(UA_NS0ID_REFRESHENDEVENTTYPE);
    }
    const auto [events, stamps] = read_events(*src, 1);
    ASSERT_EQ(events.size(), 1);
    EXPECT_EQ(events[0]["message"], "active");
    EXPECT_EQ(stamps, std::vector{time});
    ASSERT_NIL(src->stop());
}

/// @brief it should record the events of the notifiers that exist, and warn about each
/// notifier that the server does not monitor.
TEST_F(TestEventTask, testWarnsAboutMissingNotifier) {
    const auto src = this->source(this->config({"NS=1;S=Missing", "NS=1;S=Pump"}));
    ASSERT_NIL(src->start());
    x::breaker::Breaker breaker;
    x::telem::Frame fr;
    const auto res = src->read(breaker, fr);
    ASSERT_NIL(res.error);
    EXPECT_EQ(res.warning.message, "Failed to record some events");
    EXPECT_EQ(
        res.warning.description,
        "NS=1;S=Missing: The node id refers to a node that does not exist in the "
        "server address space."
    );
    this->emit("still recorded");
    const auto [events, _] = read_events(*src, 1);
    ASSERT_EQ(events.size(), 1);
    ASSERT_NIL(src->stop());
}

/// @brief it should ask the server to send the active conditions again once the server
/// monitors a notifier that it refused at start.
TEST_F(TestEventTask, testReplaysConditionsOfRecoveredNotifier) {
    this->server->stop();
    this->start_server({.event_notifier = 0});
    const auto src = this->source(this->config());
    ASSERT_NIL(src->start());
    x::breaker::Breaker breaker;
    x::telem::Frame fr;
    auto res = src->read(breaker, fr);
    ASSERT_NIL(res.error);
    EXPECT_EQ(res.warning.message, "Failed to record some events");
    EXPECT_EQ(this->server->refreshes, 1);
    this->server->set_pump_event_notifier(UA_EVENTNOTIFIER_SUBSCRIBE_TO_EVENT);
    const auto deadline = std::chrono::steady_clock::now() + std::chrono::seconds(15);
    while (!res.warning.message.empty() &&
           std::chrono::steady_clock::now() < deadline) {
        res = src->read(breaker, fr);
        ASSERT_NIL(res.error);
    }
    EXPECT_EQ(res.warning.message, "");
    EXPECT_EQ(this->server->refreshes, 2);
    ASSERT_NIL(src->stop());
}

/// @brief it should select a field whose browse path names its namespace by URI.
TEST_F(TestEventTask, testSelectsFieldByNamespaceURI) {
    auto cfg = this->config();
    cfg["fields"] = {{
        {"key", "level"},
        {"name", "level"},
        {"browse_path", "nsu=" + testutil::SERVER_APPLICATION_URI + ";Level"},
    }};
    const auto src = this->source(cfg);
    ASSERT_NIL(src->start());
    this->emit("pumped");
    const auto [events, _] = read_events(*src, 1);
    ASSERT_EQ(events.size(), 1);
    EXPECT_EQ(events[0]["level"], 2.5);
    ASSERT_NIL(src->stop());
}

/// @brief it should warn about a field in a namespace that the server does not have,
/// then select the field once the server adds the namespace.
TEST_F(TestEventTask, testSelectsFieldOfNamespaceAddedLater) {
    auto cfg = this->config();
    cfg["fields"] = {
        {{"key", "speed"}, {"name", "speed"}, {"browse_path", "nsu=urn:later;Speed"}}
    };
    const auto src = this->source(cfg);
    ASSERT_NIL(src->start());
    x::breaker::Breaker breaker;
    x::telem::Frame fr;
    auto res = src->read(breaker, fr);
    ASSERT_NIL(res.error);
    EXPECT_EQ(res.warning.message, "Failed to record some events");
    EXPECT_EQ(
        res.warning.description,
        "Field speed is in a namespace that the server does not have."
    );
    this->server->add_namespace("urn:later", {});
    const auto deadline = std::chrono::steady_clock::now() + std::chrono::seconds(15);
    while (!res.warning.message.empty() &&
           std::chrono::steady_clock::now() < deadline) {
        res = src->read(breaker, fr);
        ASSERT_NIL(res.error);
    }
    EXPECT_EQ(res.warning.message, "");
    this->emit("pumped");
    const auto [events, _] = read_events(*src, 1);
    ASSERT_EQ(events.size(), 1);
    ASSERT_TRUE(events[0].contains("speed"));
    EXPECT_TRUE(events[0]["speed"].is_null());
    ASSERT_NIL(src->stop());
}

/// @brief it should report a lost server, then subscribe again when the server returns.
TEST_F(TestEventTask, testSubscribesAgainAfterServerRestart) {
    const auto src = this->source(this->config());
    ASSERT_NIL(src->start());
    this->server->stop();
    x::breaker::Breaker breaker;
    x::errors::Error err;
    const auto deadline = std::chrono::steady_clock::now() + std::chrono::seconds(10);
    while (!err && std::chrono::steady_clock::now() < deadline) {
        x::telem::Frame fr;
        err = src->read(breaker, fr).error;
    }
    ASSERT_OCCURRED_AS(err, errors::UNREACHABLE);
    this->start_server();
    x::telem::Frame fr;
    ASSERT_EVENTUALLY_NIL(src->read(breaker, fr).error);
    this->emit("back");
    const auto [events, _] = read_events(*src, 1);
    ASSERT_EQ(events.size(), 1);
    EXPECT_EQ(events[0]["message"], "back");
    ASSERT_NIL(src->stop());
}

/// @brief it should write events through a read task.
TEST_F(TestEventTask, testTaskWritesEvents) {
    const synnax::task::Task task{
        .name = "OPC UA Event Task Test",
        .type = "opc_event"
    };
    const auto ctx = std::make_shared<driver::task::MockContext>(this->client);
    const auto factory = std::make_shared<pipeline::testutil::WriterFactory>();
    auto [cfg, err] = this->parse(this->config());
    ASSERT_NIL(err);
    common::ReadTask rt(
        task,
        ctx,
        x::breaker::default_config(task.name),
        std::make_unique<EventTaskSource>(this->client, this->pool, std::move(*cfg)),
        factory
    );
    rt.start("start");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    EXPECT_EQ(ctx->statuses[0].message, "Started task");
    this->emit("through the task");
    ASSERT_EVENTUALLY_GE(factory->writes->size(), 1);
    rt.stop("stop", true);
    const auto &written = factory->writes->at(0);
    EXPECT_TRUE(written.contains(this->events.key));
    EXPECT_TRUE(written.contains(this->index.key));
}

/// @brief it should reject a channel that does not hold JSON.
TEST_F(TestEventTask, testRejectsChannelWithoutJSON) {
    auto cfg = this->config();
    cfg["channel"] = this->index.key;
    const auto [_, err] = this->parse(cfg);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_NE(err.data.find("Channel must have the json data type"), std::string::npos);
}

/// @brief it should reject a field whose name a standard field or another field has, or
/// whose browse path has an empty name or a namespace URI with no ';' after it.
TEST_F(TestEventTask, testRejectsInvalidFields) {
    auto cfg = this->config();
    cfg["fields"] = {
        {{"key", "a"}, {"name", "message"}, {"browse_path", "Message"}},
        {{"key", "b"}, {"name", "b"}, {"browse_path", "Pump//Speed"}},
        {{"key", "c"}, {"name", "b"}, {"browse_path", "nsu=urn:pump"}},
    };
    const auto [_, err] = this->parse(cfg);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_NE(err.data.find("A standard field has this name"), std::string::npos);
    EXPECT_NE(err.data.find("Another field has this name"), std::string::npos);
    EXPECT_NE(err.data.find("Browse path must name each element"), std::string::npos);
    EXPECT_NE(
        err.data.find("Browse path must end each namespace URI with ';'"),
        std::string::npos
    );
}

/// @brief it should reject a notifier that is not a node ID.
TEST_F(TestEventTask, testRejectsInvalidNotifier) {
    const auto [_, err] = this->parse(this->config({"not a node"}));
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_NE(err.data.find("notifiers.0"), std::string::npos);
}
}
