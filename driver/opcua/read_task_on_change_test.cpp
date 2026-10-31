// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <limits>
#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/opcua/opcua.h"
#include "driver/opcua/read_task.h"
#include "driver/opcua/testutil/server.h"
#include "driver/opcua/testutil/testutil.h"

namespace driver::opcua {
namespace {
const std::string ENDPOINT = "opc.tcp://localhost:4866";
const std::string DEVICE_KEY = "opc_on_change_test_server_key";

/// @returns a node of the test server named id, in namespace 1, that holds v.
testutil::TestNode double_node(const std::string &id, UA_Double v) {
    UA_Variant value;
    UA_Variant_setScalar(&value, &v, &UA_TYPES[UA_TYPES_DOUBLE]);
    return {1, id, &UA_TYPES[UA_TYPES_DOUBLE], value, id};
}

/// @brief the values and stamps that reads wrote to one channel and its index.
struct Written {
    std::vector<double> values;
    std::vector<x::telem::TimeStamp> stamps;
    std::vector<common::Warning> warnings;

    /// @returns true if a read warned with message.
    [[nodiscard]] bool warned(const std::string &message) const {
        for (const auto &w: this->warnings)
            if (w.message == message) return true;
        return false;
    }
};

/// @brief reads src until count values of ch arrive, or timeout passes.
Written read_values(
    OnChangeReadTaskSource &src,
    const synnax::channel::Channel &ch,
    const std::size_t count,
    const x::telem::TimeSpan timeout = 5 * x::telem::SECOND
) {
    Written w;
    x::breaker::Breaker breaker;
    const auto deadline = std::chrono::steady_clock::now() + timeout.chrono();
    while (w.values.size() < count && std::chrono::steady_clock::now() < deadline) {
        x::telem::Frame fr;
        const auto res = src.read(breaker, fr);
        EXPECT_FALSE(res.error) << res.error;
        if (!res.warning.empty()) w.warnings.push_back(res.warning);
        for (const auto &[key, s]: fr)
            for (std::size_t i = 0; i < s.size(); ++i)
                if (key == ch.key)
                    w.values.push_back(s.at<double>(static_cast<int>(i)));
                else if (key == ch.index)
                    w.stamps.push_back(s.at<x::telem::TimeStamp>(static_cast<int>(i)));
    }
    return w;
}
}

class TestOnChangeReadTask : public ::testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> client;
    std::unique_ptr<testutil::Server> server;
    testutil::ServerConfig server_cfg;
    std::shared_ptr<connection::Pool> pool;
    synnax::rack::Rack rack;
    synnax::channel::Channel value;

    void SetUp() override {
        this->client = std::make_shared<synnax::Synnax>(new_test_client());
        this->value = this->create_channel("on_change");
        this->rack = ASSERT_NIL_P(
            this->client->racks.create("opc_on_change_test_rack")
        );
        connection::Config conn_cfg;
        conn_cfg.endpoint = ENDPOINT;
        conn_cfg.security_mode = "None";
        conn_cfg.security_policy = "None";
        synnax::device::Device dev{
            .key = DEVICE_KEY,
            .rack = this->rack.key,
            .location = ENDPOINT,
            .make = "opc",
            .model = "OPC UA Server",
            .name = "OPC UA On Change Test Server",
            .properties = x::json::json::object(
                {{"connection", testutil::connection_json(conn_cfg)}}
            ),
        };
        ASSERT_NIL(this->client->devices.create(dev));
        this->pool = std::make_shared<connection::Pool>(testutil::create_store());
        this->server_cfg = testutil::ServerConfig::create_default();
        this->server_cfg.port = 4866;
        this->start_server();
    }

    void TearDown() override {
        if (this->server) this->server->stop();
    }

    void start_server() {
        if (this->server) this->server->stop();
        this->server = std::make_unique<testutil::Server>(this->server_cfg);
        this->server->start();
        ASSERT_TRUE(this->server->wait_until_ready());
    }

    /// @returns a new float64 channel on an index of its own.
    synnax::channel::Channel create_channel(const std::string &name) {
        const auto idx = ASSERT_NIL_P(this->client->channels.create(
            make_unique_channel_name(name + "_time"),
            x::telem::TIMESTAMP_T,
            0,
            true
        ));
        return ASSERT_NIL_P(this->client->channels.create(
            make_unique_channel_name(name),
            x::telem::FLOAT64_T,
            idx.key,
            false
        ));
    }

    /// @returns the config of an On change task that reads each node into each channel,
    /// sampling every 50 ms and publishing every 100 ms.
    [[nodiscard]] x::json::json config(
        const std::vector<std::pair<std::string, synnax::channel::Key>> &channels
    ) const {
        auto chs = x::json::json::array();
        for (const auto &[node, key]: channels)
            chs.push_back({{"node_id", node}, {"channel", key}, {"disabled", false}});
        return {
            {"device", DEVICE_KEY},
            {"mode", "on_change"},
            {"sample_rate", 20},
            {"stream_rate", 10},
            {"channels", chs},
        };
    }

    /// @returns the config of an On change task that reads TestDouble.
    [[nodiscard]] x::json::json config() const {
        return this->config({{"NS=1;S=TestDouble", this->value.key}});
    }

    [[nodiscard]] std::unique_ptr<OnChangeReadTaskSource>
    source(const x::json::json &cfg) const {
        auto p = x::json::Parser(cfg);
        auto parsed = ReadTaskConfig(this->client, p);
        EXPECT_FALSE(p.error()) << p.error();
        return std::make_unique<OnChangeReadTaskSource>(
            this->client,
            this->pool,
            std::move(parsed)
        );
    }

    /// @returns the errors of parsing cfg.
    [[nodiscard]] x::json::json errors_of(const x::json::json &cfg) const {
        auto p = x::json::Parser(cfg);
        const ReadTaskConfig parsed(this->client, p);
        return p.error_json()["errors"];
    }

    /// @brief stores v at time on ch, as an earlier run of the task would.
    void store(
        const synnax::channel::Channel &ch,
        const x::telem::TimeStamp time,
        const double v
    ) const {
        auto writer = ASSERT_NIL_P(this->client->telem.open_writer(
            synnax::framer::WriterConfig{
                .channels = {ch.index, ch.key},
                .start = time,
            }
        ));
        auto fr = x::telem::Frame(2);
        fr.emplace(ch.index, x::telem::Series(time));
        fr.emplace(ch.key, x::telem::Series(v));
        ASSERT_NIL(writer.write(fr));
        ASSERT_NIL_P(writer.commit());
        ASSERT_NIL(writer.close());
    }

    /// @returns the last time that the Core stores on the index of the value channel.
    [[nodiscard]] x::telem::TimeStamp last_stored() const {
        auto [last, err] = common::last_stamps(*this->client, {this->value.index});
        EXPECT_FALSE(err) << err;
        return last[this->value.index];
    }

    /// @brief waits until a task without backfill reads v as the current value of
    /// TestDouble, so the server has applied each change before it.
    void wait_for(const double v) const {
        const auto probe = this->source(this->config());
        ASSERT_NIL(probe->start());
        x::breaker::Breaker breaker;
        ASSERT_EVENTUALLY_EQ(
            [&] {
                x::telem::Frame fr;
                probe->read(breaker, fr);
                return fr.contains(this->value.key)
                         ? fr.at<double>(this->value.key, -1)
                         : std::numeric_limits<double>::quiet_NaN();
            }(),
            v
        );
        ASSERT_NIL(probe->stop());
    }

    /// @brief changes TestDouble on the server to v at time.
    void change(
        const double v,
        const x::telem::TimeStamp time,
        const UA_StatusCode status = UA_STATUSCODE_GOOD
    ) const {
        this->server->change({
            .node = "TestDouble",
            .value = v,
            .time = time,
            .status = status,
        });
    }
};

/// @brief it should write the current value of a node, then each change at its source
/// time, on the index of its channel.
TEST_F(TestOnChangeReadTask, testWritesChangesAtSourceTime) {
    const auto src = this->source(this->config());
    ASSERT_NIL(src->start());
    ASSERT_EQ(read_values(*src, this->value, 1).values.size(), 1);
    const auto time = testutil::now(x::telem::SECOND);
    this->change(7.5, time);
    const auto written = read_values(*src, this->value, 1);
    EXPECT_EQ(written.values, std::vector<double>({7.5}));
    EXPECT_EQ(written.stamps, std::vector({time}));
    EXPECT_TRUE(written.warnings.empty());
    ASSERT_NIL(src->stop());
}

/// @brief it should drop a change at the time of the last value without a warning, and
/// warn about a change before it.
TEST_F(TestOnChangeReadTask, testDropsChangesThatAreNotAfterTheLast) {
    const auto src = this->source(this->config());
    ASSERT_NIL(src->start());
    ASSERT_EQ(read_values(*src, this->value, 1).values.size(), 1);
    const auto time = testutil::now(2 * x::telem::SECOND);
    this->change(1, time);
    ASSERT_EQ(read_values(*src, this->value, 1).values, std::vector<double>({1}));
    this->change(2, time);
    const auto repeated = read_values(*src, this->value, 1, x::telem::SECOND);
    EXPECT_TRUE(repeated.values.empty());
    EXPECT_TRUE(repeated.warnings.empty());
    this->change(3, time - x::telem::SECOND);
    const auto old = read_values(*src, this->value, 1, x::telem::SECOND);
    EXPECT_TRUE(old.values.empty());
    ASSERT_TRUE(old.warned("Rejected values with old timestamps"));
    EXPECT_EQ(
        old.warnings.back().description,
        "Rejected a value of " + this->value.name +
            ": its timestamp is not after the last one written."
    );
    ASSERT_NIL(src->stop());
}

/// @brief it should drop the current value that the server sends at start without a
/// warning when the channel stores a later value.
TEST_F(TestOnChangeReadTask, testDropsRepeatedValueAtStart) {
    const auto stored = testutil::now(10 * x::telem::SECOND);
    this->store(this->value, stored, 1);
    const auto src = this->source(this->config());
    ASSERT_NIL(src->start());
    const auto first = read_values(*src, this->value, 1, x::telem::SECOND);
    EXPECT_TRUE(first.values.empty());
    EXPECT_TRUE(first.warnings.empty());
    this->change(4, stored + x::telem::SECOND);
    const auto written = read_values(*src, this->value, 1);
    EXPECT_EQ(written.values, std::vector<double>({4}));
    EXPECT_EQ(written.stamps, std::vector({stored + x::telem::SECOND}));
    ASSERT_NIL(src->stop());
}

/// @brief it should write an uncertain value with a warning, and skip a bad value with
/// a warning that names its status.
TEST_F(TestOnChangeReadTask, testWarnsAboutQuality) {
    const auto src = this->source(this->config());
    ASSERT_NIL(src->start());
    ASSERT_EQ(read_values(*src, this->value, 1).values.size(), 1);
    this->change(
        5,
        testutil::now(x::telem::SECOND),
        UA_STATUSCODE_UNCERTAININITIALVALUE
    );
    const auto uncertain = read_values(*src, this->value, 1);
    EXPECT_EQ(uncertain.values, std::vector<double>({5}));
    EXPECT_TRUE(uncertain.warned("Received uncertain values"));
    this->change(
        6,
        testutil::now(2 * x::telem::SECOND),
        UA_STATUSCODE_BADSENSORFAILURE
    );
    const auto bad = read_values(*src, this->value, 1, x::telem::SECOND);
    EXPECT_TRUE(bad.values.empty());
    ASSERT_TRUE(bad.warned("Received bad values"));
    EXPECT_NE(
        bad.warnings.back().description.find("BadSensorFailure"),
        std::string::npos
    );
    ASSERT_NIL(src->stop());
}

/// @brief it should read the nodes that exist, and warn about each node that the server
/// does not monitor.
TEST_F(TestOnChangeReadTask, testWarnsAboutUnmonitoredNode) {
    const auto missing = this->create_channel("missing");
    const auto src = this->source(this->config({
        {"NS=1;S=Missing", missing.key},
        {"NS=1;S=TestDouble", this->value.key},
    }));
    ASSERT_NIL(src->start());
    const auto written = read_values(*src, this->value, 1);
    EXPECT_EQ(written.values.size(), 1);
    ASSERT_TRUE(written.warned("Failed to subscribe to some nodes"));
    EXPECT_EQ(written.warnings.back().description.find(missing.name + ": "), 0);
    ASSERT_NIL(src->stop());
}

/// @brief it should describe the intervals that the server chose when it samples or
/// publishes slower than the task asks, without a warning.
TEST_F(TestOnChangeReadTask, testDescribesRevisedIntervals) {
    auto cfg = this->config();
    cfg["sample_rate"] = 1000;
    cfg["stream_rate"] = 500;
    const auto src = this->source(cfg);
    ASSERT_NIL(src->start());
    x::breaker::Breaker breaker;
    x::telem::Frame fr;
    const auto res = src->read(breaker, fr);
    ASSERT_NIL(res.error);
    EXPECT_TRUE(res.warning.empty()) << res.warning.description;
    EXPECT_EQ(
        res.description,
        "The server samples each node every 50ms and sends changes every 100ms, not "
        "every 1ms and 2ms as the task asks."
    );
    ASSERT_NIL(src->stop());
}

/// @brief it should warn about a node that the server does not monitor, then monitor
/// the node once the server adds it.
TEST_F(TestOnChangeReadTask, testMonitorsNodeAddedLater) {
    const auto later = this->create_channel("later");
    const auto src = this->source(this->config({{"NS=1;S=Later", later.key}}));
    ASSERT_NIL(src->start());
    const auto missing = read_values(*src, later, 1, x::telem::SECOND);
    ASSERT_TRUE(missing.warned("Failed to subscribe to some nodes"));
    this->server->add_namespace(
        testutil::SERVER_APPLICATION_URI,
        {double_node("Later", 5)}
    );
    const auto found = read_values(*src, later, 1, 15 * x::telem::SECOND);
    EXPECT_EQ(found.values, std::vector<double>({5}));
    ASSERT_NIL(src->stop());
}

/// @brief it should describe the sampling interval that the server chose for a node
/// that it monitors once the server adds it.
TEST_F(TestOnChangeReadTask, testDescribesIntervalOfNodeAddedLater) {
    const auto later = this->create_channel("later");
    auto cfg = this->config({{"NS=1;S=Later", later.key}});
    cfg["sample_rate"] = 1000;
    const auto src = this->source(cfg);
    ASSERT_NIL(src->start());
    x::breaker::Breaker breaker;
    x::telem::Frame fr;
    auto res = src->read(breaker, fr);
    ASSERT_NIL(res.error);
    ASSERT_EQ(res.warning.message, "Failed to subscribe to some nodes");
    EXPECT_EQ(res.description, "");
    this->server->add_namespace(
        testutil::SERVER_APPLICATION_URI,
        {double_node("Later", 5)}
    );
    const auto deadline = std::chrono::steady_clock::now() + std::chrono::seconds(15);
    while (!res.warning.empty() && std::chrono::steady_clock::now() < deadline) {
        res = src->read(breaker, fr);
        ASSERT_NIL(res.error);
    }
    EXPECT_TRUE(res.warning.empty()) << res.warning.description;
    EXPECT_EQ(
        res.description,
        "The server samples each node every 50ms and sends changes every 100ms, not "
        "every 1ms and 100ms as the task asks."
    );
    ASSERT_NIL(src->stop());
}

/// @brief it should read a structure field of a node that the server adds after the
/// start, once it reads the structure types again.
TEST_F(TestOnChangeReadTask, testReadsFieldOfAddedNode) {
    auto cfg = this->config({{"NS=1;S=TestMotor", this->value.key}});
    cfg["channels"][0]["field"] = "Speed";
    const auto src = this->source(cfg);
    ASSERT_NIL(src->start());
    const auto missing = read_values(*src, this->value, 1, x::telem::SECOND);
    ASSERT_TRUE(missing.values.empty());
    this->server->add_motor({
        .speed = 1500.0,
        .running = true,
        .position = {.x = 1.5F, .y = 2.5F},
    });
    const auto found = read_values(*src, this->value, 1, 25 * x::telem::SECOND);
    EXPECT_EQ(found.values, std::vector<double>({1500}));
    ASSERT_NIL(src->stop());
}

/// @brief it should warn about a structure field on each read while its type fails to
/// read, then read the field once the server defines the type, though the node does not
/// change.
TEST_F(TestOnChangeReadTask, testReadsFieldOnceTypeRecovers) {
    this->server_cfg.motor = testutil::Motor{
        .speed = 1500.0,
        .running = true,
        .position = {.x = 1.5F, .y = 2.5F},
    };
    this->server_cfg.motor_type_missing = true;
    this->start_server();
    auto cfg = this->config({{"NS=1;S=TestMotor", this->value.key}});
    cfg["channels"][0]["field"] = "Speed";
    const auto src = this->source(cfg);
    ASSERT_NIL(src->start());
    const auto missing = read_values(*src, this->value, 1, x::telem::SECOND);
    ASSERT_TRUE(missing.values.empty());
    x::breaker::Breaker breaker;
    x::telem::Frame fr;
    EXPECT_EQ(src->read(breaker, fr).warning.message, "Received bad values");
    this->server->add_motor_type();
    const auto found = read_values(*src, this->value, 1, 25 * x::telem::SECOND);
    EXPECT_EQ(found.values, std::vector<double>({1500}));
    ASSERT_NIL(src->stop());
}

/// @brief it should backfill a structure field once the server defines its type, from
/// the last stored value through the changes that failed to decode.
TEST_F(TestOnChangeReadTask, testBackfillsFieldOnceTypeRecovers) {
    this->server_cfg.motor = testutil::Motor{
        .speed = 0,
        .running = true,
        .position = {},
    };
    this->server_cfg.motor_type_missing = true;
    this->server_cfg.historizing = true;
    this->start_server();
    const auto stored = testutil::now(-1 * x::telem::MINUTE);
    this->store(this->value, stored, 0);
    const auto change_speed = [&](const int speed) {
        this->server->change({
            .node = "TestMotor",
            .value =
                testutil::Motor{
                    .speed = static_cast<double>(speed),
                    .running = true,
                    .position = {},
                },
            .time = stored + x::telem::SECOND * speed,
        });
    };
    change_speed(1);
    change_speed(2);
    // The server applies changes in order, so a later change shows that it applied
    // these.
    this->change(-1, testutil::now());
    this->wait_for(-1);
    auto cfg = this->config({{"NS=1;S=TestMotor", this->value.key}});
    cfg["channels"][0]["field"] = "Speed";
    cfg["history_backfill_enabled"] = true;
    const auto src = this->source(cfg);
    ASSERT_NIL(src->start());
    ASSERT_TRUE(read_values(*src, this->value, 1, x::telem::SECOND).values.empty());
    change_speed(3);
    change_speed(4);
    this->change(-2, testutil::now());
    this->wait_for(-2);
    ASSERT_TRUE(read_values(*src, this->value, 1, x::telem::SECOND).values.empty());
    this->server->add_motor_type();
    const auto found = read_values(*src, this->value, 4, 25 * x::telem::SECOND);
    EXPECT_EQ(found.values, std::vector<double>({1, 2, 3, 4}));
    EXPECT_EQ(
        found.stamps,
        std::vector({
            stored + x::telem::SECOND,
            stored + 2 * x::telem::SECOND,
            stored + 3 * x::telem::SECOND,
            stored + 4 * x::telem::SECOND,
        })
    );
    ASSERT_NIL(src->stop());
}

/// @brief it should warn about a node in a namespace that the server lacks, then
/// subscribe to the node once the server adds the namespace.
TEST_F(TestOnChangeReadTask, testSubscribesToNamespaceAddedLater) {
    const auto later = this->create_channel("later");
    const auto src = this->source(this->config({{"nsu=urn:later;s=Later", later.key}}));
    ASSERT_NIL(src->start());
    const auto missing = read_values(*src, later, 1, x::telem::SECOND);
    ASSERT_TRUE(missing.warned("Failed to subscribe to some nodes"));
    EXPECT_EQ(
        missing.warnings.back().description,
        later.name + ": the server has no namespace urn:later."
    );
    this->server->add_namespace("urn:later", {double_node("Later", 5)});
    const auto found = read_values(*src, later, 1, 15 * x::telem::SECOND);
    EXPECT_EQ(found.values, std::vector<double>({5}));
    ASSERT_NIL(src->stop());
}

/// @brief it should subscribe again after the server restarts.
TEST_F(TestOnChangeReadTask, testSubscribesAgainAfterServerRestart) {
    const auto src = this->source(this->config());
    ASSERT_NIL(src->start());
    ASSERT_EQ(read_values(*src, this->value, 1).values.size(), 1);
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
    this->change(8, testutil::now(x::telem::SECOND));
    const auto written = read_values(*src, this->value, 2, 2 * x::telem::SECOND);
    ASSERT_FALSE(written.values.empty());
    EXPECT_EQ(written.values.back(), 8);
    ASSERT_NIL(src->stop());
}

/// @brief it should write the history of a node from the last stored value to the start
/// of the subscription, a page at a time, before its live changes.
TEST_F(TestOnChangeReadTask, testBackfillsHistory) {
    this->server_cfg.historizing = true;
    this->server_cfg.max_history_values = 2;
    this->start_server();
    const auto stored = testutil::now(-1 * x::telem::MINUTE);
    this->store(this->value, stored, 0);
    for (int i = 1; i <= 3; ++i)
        this->change(i, stored + x::telem::SECOND * i);
    this->wait_for(3);
    auto cfg = this->config();
    cfg["history_backfill_enabled"] = true;
    const auto src = this->source(cfg);
    ASSERT_NIL(src->start());
    const auto history = read_values(*src, this->value, 3);
    EXPECT_EQ(history.values, std::vector<double>({1, 2, 3}));
    EXPECT_EQ(
        history.stamps,
        std::vector({
            stored + x::telem::SECOND,
            stored + 2 * x::telem::SECOND,
            stored + 3 * x::telem::SECOND,
        })
    );
    EXPECT_TRUE(history.warnings.empty());
    const auto time = testutil::now(x::telem::SECOND);
    this->change(4, time);
    const auto live = read_values(*src, this->value, 1);
    EXPECT_EQ(live.values, std::vector<double>({4}));
    EXPECT_EQ(live.stamps, std::vector({time}));
    ASSERT_NIL(src->stop());
}

/// @brief it should hold a change that arrives during a backfill until the history
/// before it is written.
TEST_F(TestOnChangeReadTask, testHoldsChangesDuringBackfill) {
    this->server_cfg.historizing = true;
    this->server_cfg.max_history_values = 1;
    this->start_server();
    const auto stored = testutil::now(-1 * x::telem::MINUTE);
    this->store(this->value, stored, 0);
    for (int i = 1; i <= 3; ++i)
        this->change(i, stored + x::telem::SECOND * i);
    // A current value after the start of the subscription arrives with the first
    // notification, while the backfill still has pages to read.
    const auto time = testutil::now(10 * x::telem::SECOND);
    this->change(4, time);
    this->wait_for(4);
    auto cfg = this->config();
    cfg["history_backfill_enabled"] = true;
    const auto src = this->source(cfg);
    ASSERT_NIL(src->start());
    const auto written = read_values(*src, this->value, 4);
    EXPECT_EQ(written.values, std::vector<double>({1, 2, 3, 4}));
    EXPECT_EQ(written.stamps.back(), time);
    ASSERT_NIL(src->stop());
}

/// @brief it should read history up to the first change of the subscription, so it
/// keeps the changes after the Driver's time when the server's clock is ahead.
TEST_F(TestOnChangeReadTask, testBackfillsToFirstChange) {
    this->server_cfg.historizing = true;
    this->start_server();
    const auto stored = testutil::now(-1 * x::telem::MINUTE);
    this->store(this->value, stored, 0);
    this->change(1, stored + x::telem::SECOND);
    const auto ahead = testutil::now(10 * x::telem::SECOND);
    this->change(2, ahead);
    this->change(3, ahead + x::telem::SECOND);
    this->wait_for(3);
    auto cfg = this->config();
    cfg["history_backfill_enabled"] = true;
    const auto src = this->source(cfg);
    ASSERT_NIL(src->start());
    const auto written = read_values(*src, this->value, 3);
    EXPECT_EQ(written.values, std::vector<double>({1, 2, 3}));
    EXPECT_EQ(
        written.stamps,
        std::vector({stored + x::telem::SECOND, ahead, ahead + x::telem::SECOND})
    );
    ASSERT_NIL(src->stop());
}

/// @brief it should read the history of each node from its own last stored value, when
/// the nodes store their last values at different times.
TEST_F(TestOnChangeReadTask, testBackfillsEachNodeFromItsOwnStart) {
    this->server_cfg.test_nodes.push_back(double_node("OtherDouble", 0));
    this->server_cfg.historizing = true;
    this->server_cfg.max_history_values = 2;
    this->start_server();
    const auto other = this->create_channel("other");
    const auto old = testutil::now(-1 * x::telem::MINUTE);
    const auto recent = testutil::now(-10 * x::telem::SECOND);
    this->store(this->value, old, 0);
    this->store(other, recent, 0);
    // Values that the other channel already stores fill a page from the start of the
    // value channel.
    for (int i = 1; i <= 4; ++i)
        this->server->change({
            .node = "OtherDouble",
            .value = static_cast<double>(-i),
            .time = recent - x::telem::SECOND * (5 - i),
        });
    for (int i = 1; i <= 2; ++i)
        this->server->change({
            .node = "OtherDouble",
            .value = static_cast<double>(i),
            .time = recent + x::telem::SECOND * i,
        });
    this->change(1, old + x::telem::SECOND);
    this->wait_for(1);
    auto cfg = this->config({
        {"NS=1;S=TestDouble", this->value.key},
        {"NS=1;S=OtherDouble", other.key},
    });
    cfg["history_backfill_enabled"] = true;
    const auto src = this->source(cfg);
    ASSERT_NIL(src->start());
    const auto written = read_values(*src, other, 2);
    EXPECT_EQ(written.values, std::vector<double>({1, 2}));
    ASSERT_NIL(src->stop());
}

/// @brief it should backfill a node from the earliest last value that its channels
/// store, when another channel of the node stores none.
TEST_F(TestOnChangeReadTask, testBackfillsNodeWithNewChannel) {
    this->server_cfg.historizing = true;
    this->start_server();
    const auto added = this->create_channel("added");
    const auto stored = testutil::now(-1 * x::telem::MINUTE);
    this->store(this->value, stored, 0);
    for (int i = 1; i <= 2; ++i)
        this->change(i, stored + x::telem::SECOND * i);
    this->wait_for(2);
    auto cfg = this->config({
        {"NS=1;S=TestDouble", this->value.key},
        {"NS=1;S=TestDouble", added.key},
    });
    cfg["history_backfill_enabled"] = true;
    const auto src = this->source(cfg);
    ASSERT_NIL(src->start());
    const auto written = read_values(*src, this->value, 2);
    EXPECT_EQ(written.values, std::vector<double>({1, 2}));
    ASSERT_NIL(src->stop());
}

/// @brief it should backfill the history of a node that the server monitors only after
/// the start.
TEST_F(TestOnChangeReadTask, testBackfillsNodeMonitoredLater) {
    this->server_cfg.historizing = true;
    this->start_server();
    const auto later = this->create_channel("later");
    const auto stored = testutil::now(-1 * x::telem::MINUTE);
    this->store(later, stored, 0);
    auto cfg = this->config({{"NS=1;S=Later", later.key}});
    cfg["history_backfill_enabled"] = true;
    const auto src = this->source(cfg);
    ASSERT_NIL(src->start());
    const auto missing = read_values(*src, later, 1, x::telem::SECOND);
    ASSERT_TRUE(missing.warned("Failed to subscribe to some nodes"));
    this->server->add_namespace(
        testutil::SERVER_APPLICATION_URI,
        {double_node("Later", 0)}
    );
    for (int i = 1; i <= 2; ++i)
        this->server->change({
            .node = "Later",
            .value = static_cast<double>(i),
            .time = stored + x::telem::SECOND * i,
        });
    const auto found = read_values(*src, later, 2, 15 * x::telem::SECOND);
    EXPECT_EQ(found.values, std::vector<double>({1, 2}));
    ASSERT_NIL(src->stop());
}

/// @brief it should fill the gap of an outage from the history when it subscribes
/// again, on the same Session.
TEST_F(TestOnChangeReadTask, testBackfillsOutageAfterReconnect) {
    connection::Config conn_cfg;
    conn_cfg.endpoint = ENDPOINT;
    conn_cfg.security_mode = "None";
    conn_cfg.security_policy = "None";
    conn_cfg.client_timeout_ms = 1000;
    auto dev = ASSERT_NIL_P(this->client->devices.retrieve(DEVICE_KEY));
    dev.properties = x::json::json::object(
        {{"connection", testutil::connection_json(conn_cfg)}}
    );
    ASSERT_NIL(this->client->devices.create(dev));
    this->server_cfg.historizing = true;
    this->start_server();
    this->change(1, testutil::now(-10 * x::telem::SECOND));
    this->wait_for(1);
    auto cfg = this->config();
    cfg["history_backfill_enabled"] = true;
    const auto src = this->source(cfg);
    ASSERT_NIL(src->start());
    ASSERT_EQ(read_values(*src, this->value, 1).values, std::vector<double>({1}));
    const auto sessions = this->server->sessions_created.load();
    this->server->frozen = true;
    this->change(2, testutil::now(-5 * x::telem::SECOND));
    this->change(3, testutil::now(-4 * x::telem::SECOND));
    x::breaker::Breaker breaker;
    x::errors::Error err;
    const auto deadline = std::chrono::steady_clock::now() + std::chrono::seconds(20);
    while (!err && std::chrono::steady_clock::now() < deadline) {
        x::telem::Frame fr;
        err = src->read(breaker, fr).error;
        EXPECT_FALSE(fr.contains(this->value.key));
    }
    this->server->frozen = false;
    ASSERT_OCCURRED_AS(err, errors::UNREACHABLE);
    const auto written = read_values(*src, this->value, 2, 10 * x::telem::SECOND);
    EXPECT_EQ(written.values, std::vector<double>({2, 3}));
    EXPECT_EQ(this->server->sessions_created.load(), sessions);
    ASSERT_NIL(src->stop());
}

/// @brief it should read no history when the channel stores nothing yet.
TEST_F(TestOnChangeReadTask, testSkipsBackfillWithoutStoredValues) {
    this->server_cfg.historizing = true;
    this->start_server();
    this->change(1, testutil::now(-2 * x::telem::MINUTE));
    this->change(2, testutil::now(-1 * x::telem::MINUTE));
    this->wait_for(2);
    auto cfg = this->config();
    cfg["history_backfill_enabled"] = true;
    const auto src = this->source(cfg);
    ASSERT_NIL(src->start());
    const auto written = read_values(*src, this->value, 2, x::telem::SECOND);
    EXPECT_EQ(written.values, std::vector<double>({2}));
    ASSERT_NIL(src->stop());
}

/// @brief it should reject channels that share an index, an index channel, and array
/// mode.
TEST_F(TestOnChangeReadTask, testRejectsInvalidConfig) {
    const auto shared = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("shared"),
        x::telem::FLOAT64_T,
        this->value.index,
        false
    ));
    auto errs = this->errors_of(this->config({
        {"NS=1;S=TestDouble", this->value.key},
        {"NS=1;S=TestFloat", shared.key},
    }));
    ASSERT_EQ(errs.size(), 1);
    EXPECT_EQ(errs[0]["path"], "channels.1.channel");
    EXPECT_EQ(errs[0]["message"], "Channel must have its own index in On change mode");
    errs = this->errors_of(this->config({{"NS=1;S=TestDouble", this->value.index}}));
    ASSERT_EQ(errs.size(), 1);
    EXPECT_EQ(errs[0]["path"], "channels.0.channel");
    EXPECT_EQ(errs[0]["message"], "On change mode cannot read an index channel");
    auto cfg = this->config();
    cfg["array_mode"] = true;
    errs = this->errors_of(cfg);
    ASSERT_EQ(errs.size(), 1);
    EXPECT_EQ(errs[0]["path"], "array_mode");
    EXPECT_EQ(errs[0]["message"], "On change mode cannot read arrays");
    cfg = this->config();
    cfg["mode"] = "sometimes";
    errs = this->errors_of(cfg);
    ASSERT_EQ(errs.size(), 1);
    EXPECT_EQ(errs[0]["path"], "mode");
    EXPECT_EQ(errs[0]["message"], "Mode must be sampled or on_change");
}

/// @brief it should write changes to the Core through a task that the factory
/// configures.
TEST_F(TestOnChangeReadTask, testFactoryTaskWritesToCore) {
    Factory factory(testutil::create_store());
    const auto ctx = std::make_shared<driver::task::MockContext>(this->client);
    synnax::task::Task task{
        .rack = this->rack.key,
        .name = "OPC UA On Change Test",
        .type = READ_TASK_TYPE,
    };
    task.config = this->config();
    auto [t, ok] = factory.configure_task(ctx, task, "configure");
    ASSERT_TRUE(ok);
    ASSERT_NE(t, nullptr);
    synnax::task::Command start{.task = task.key, .type = "start", .key = "start"};
    t->exec(start);
    const auto time = testutil::now(x::telem::SECOND);
    this->change(9, time);
    ASSERT_EVENTUALLY_EQ(this->last_stored(), time);
    synnax::task::Command stop{.task = task.key, .type = "stop", .key = "stop"};
    t->exec(stop);
}
}
