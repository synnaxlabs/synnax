// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <thread>

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/defer/defer.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/rack/status/status.h"

namespace driver::rack::status {
/// @brief a status set transport that fails the first rack heartbeat and the first
/// re-send of task statuses, and echoes every other write.
class ScriptedSetClient final : public synnax::status::SetClient {
    const std::string rack_key;

public:
    std::atomic<int> beats = 0;
    std::atomic<int> task_writes = 0;

    explicit ScriptedSetClient(std::string rack_key): rack_key(std::move(rack_key)) {}

    void use(std::shared_ptr<freighter::Middleware>) override {}

    std::pair<grpc::status::SetResponse, x::errors::Error>
    send(const std::string &, grpc::status::SetRequest &req) override {
        // The first task write is the started status, and the second is a re-send.
        if (req.statuses(0).key() == this->rack_key) {
            if (this->beats++ == 0) return {{}, freighter::UNREACHABLE};
        } else if (this->task_writes++ == 1)
            return {{}, freighter::UNREACHABLE};
        grpc::status::SetResponse res;
        *res.mutable_statuses() = req.statuses();
        return {res, x::errors::NIL};
    }
};

/// @brief a status set transport that holds the second rack heartbeat past LATE_GAP,
/// and echoes every write.
class SlowSetClient final : public synnax::status::SetClient {
    const std::string rack_key;

public:
    std::atomic<int> beats = 0;
    std::atomic<int> task_writes = 0;

    explicit SlowSetClient(std::string rack_key): rack_key(std::move(rack_key)) {}

    void use(std::shared_ptr<freighter::Middleware>) override {}

    std::pair<grpc::status::SetResponse, x::errors::Error>
    send(const std::string &, grpc::status::SetRequest &req) override {
        if (req.statuses(0).key() != this->rack_key)
            this->task_writes++;
        else if (this->beats++ == 1)
            std::this_thread::sleep_for((LATE_GAP + x::telem::SECOND / 2).chrono());
        grpc::status::SetResponse res;
        *res.mutable_statuses() = req.statuses();
        return {res, x::errors::NIL};
    }
};

/// @brief it should report nominal driver status via state streamer.
TEST(stateTests, Nominal) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("test_rack"));
    auto ch = ASSERT_NIL_P(
        client->channels.retrieve(synnax::status::STATUS_SET_CHANNEL_NAME)
    );
    auto ctx = std::make_shared<driver::task::SynnaxContext>(
        client,
        std::make_shared<driver::task::StatusCache>(client->statuses)
    );
    auto task = synnax::task::Task{
        .name = "state",
        .type = driver::rack::status::TASK_TYPE,
        .internal = true
    };
    ASSERT_NIL(rack.tasks.create(task));
    auto hb = driver::rack::status::Task::configure(ctx, task);
    auto cmd = synnax::task::Command{
        .task = task.key,
        .type = "start",
    };
    hb->exec(cmd);
    x::defer::defer stop([&hb]() { hb->stop(false); });
    auto streamer = ASSERT_NIL_P(client->telem.open_streamer(
        synnax::framer::StreamerConfig{
            .channels = {ch.key},
        }
    ));
    x::json::json j;
    for (int i = 0; i < 50; i++) {
        auto frm = ASSERT_NIL_P(streamer.read());
        ASSERT_EQ(frm.size(), 1);
        frm.series->at(0).at(-1, j);
        if (j["details"]["rack"] == rack.key) break;
    }
    EXPECT_EQ(j["details"]["rack"], rack.key);
    EXPECT_EQ(j["variant"], synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(j["message"], "Driver is running");
    ASSERT_NIL(streamer.close());
}

/// @brief it should send the context's current rack name in each heartbeat.
TEST(stateTests, RenamedRack) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("test_rack"));
    auto ch = ASSERT_NIL_P(
        client->channels.retrieve(synnax::status::STATUS_SET_CHANNEL_NAME)
    );
    auto ctx = std::make_shared<driver::task::SynnaxContext>(
        client,
        std::make_shared<driver::task::StatusCache>(client->statuses)
    );
    ctx->set_rack_name(rack.name);
    auto task = synnax::task::Task{
        .name = "state",
        .type = driver::rack::status::TASK_TYPE,
        .internal = true,
    };
    ASSERT_NIL(rack.tasks.create(task));
    auto streamer = ASSERT_NIL_P(client->telem.open_streamer(
        synnax::framer::StreamerConfig{.channels = {ch.key}}
    ));
    auto hb = driver::rack::status::Task::configure(ctx, task);
    ASSERT_NE(hb, nullptr);
    x::defer::defer stop([&hb]() { hb->stop(false); });
    const auto key = synnax::rack::ontology_id(rack.key).string();
    const auto next_beat_name = [&]() -> std::string {
        for (int i = 0; i < 50; i++) {
            auto [frm, err] = streamer.read();
            if (err) return "";
            for (size_t c = 0; c < frm.size(); c++)
                for (const auto &raw: frm.series->at(c).strings())
                    if (const auto j = x::json::json::parse(raw); j["key"] == key)
                        return j["name"];
        }
        return "";
    };
    ASSERT_EQ(next_beat_name(), "test_rack");

    ctx->set_rack_name("renamed_rack");
    // A beat in flight during the rename may still carry the old name.
    bool renamed = false;
    for (int i = 0; i < 3 && !renamed; i++)
        renamed = next_beat_name() == "renamed_rack";
    ASSERT_TRUE(renamed);
    EXPECT_EQ(next_beat_name(), "renamed_rack");
    EXPECT_EQ(next_beat_name(), "renamed_rack");
    ASSERT_NIL(streamer.close());
}

/// @brief it should report the task as running while the heartbeat is live, and as
/// not running once it stops.
TEST(stateTests, TaskRunningTransitions) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("test_rack"));
    auto ch = ASSERT_NIL_P(
        client->channels.retrieve(synnax::status::STATUS_SET_CHANNEL_NAME)
    );
    auto ctx = std::make_shared<driver::task::SynnaxContext>(
        client,
        std::make_shared<driver::task::StatusCache>(client->statuses)
    );
    auto task = synnax::task::Task{
        .name = "state",
        .type = driver::rack::status::TASK_TYPE,
        .internal = true,
    };
    ASSERT_NIL(rack.tasks.create(task));
    const auto key = synnax::task::status_key(task);
    // The started status is written once, so the streamer must be open before the
    // task is configured.
    auto streamer = ASSERT_NIL_P(client->telem.open_streamer(
        synnax::framer::StreamerConfig{.channels = {ch.key}}
    ));
    auto hb = driver::rack::status::Task::configure(ctx, task);
    ASSERT_NE(hb, nullptr);

    bool started = false;
    for (int i = 0; i < 50 && !started; i++) {
        auto frm = ASSERT_NIL_P(streamer.read());
        for (size_t c = 0; c < frm.size(); c++)
            for (const auto &raw: frm.series->at(c).strings()) {
                const auto j = x::json::json::parse(raw);
                if (j["key"] != key) continue;
                EXPECT_EQ(j["details"]["running"], true);
                EXPECT_EQ(j["details"]["task"], task.key.to_string());
                started = true;
            }
    }
    EXPECT_TRUE(started);

    hb->stop(false);

    bool stopped = false;
    for (int i = 0; i < 50 && !stopped; i++) {
        auto frm = ASSERT_NIL_P(streamer.read());
        for (size_t c = 0; c < frm.size(); c++)
            for (const auto &raw: frm.series->at(c).strings()) {
                const auto j = x::json::json::parse(raw);
                if (j["key"] != key || j["details"]["running"] != false) continue;
                EXPECT_EQ(j["message"], "Stopped");
                stopped = true;
            }
    }
    EXPECT_TRUE(stopped);
    ASSERT_NIL(streamer.close());
}

/// @brief it should send task statuses again on the first heartbeat that succeeds after
/// a failed one, try again on the next heartbeat when the re-send fails, and stop once
/// a re-send succeeds.
TEST(stateTests, ResendsTaskStatusesUntilTheyReachTheCore) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    const synnax::rack::Rack rack{.key = 1, .name = "test_rack"};
    const auto transport = std::make_shared<ScriptedSetClient>(
        synnax::rack::ontology_id(rack.key).string()
    );
    client->statuses = synnax::status::Client(nullptr, transport, nullptr, nullptr);
    const auto ctx = std::make_shared<driver::task::SynnaxContext>(
        client,
        std::make_shared<driver::task::StatusCache>(client->statuses)
    );
    Task hb(
        rack,
        synnax::task::Task{.name = "state", .type = TASK_TYPE, .internal = true},
        ctx
    );
    x::defer::defer stop([&hb]() { hb.stop(false); });
    // Beat 1 fails, the re-send after beat 2 fails, beat 3 re-sends, and beat 4 sends
    // nothing. Beat 5 starts once beat 4 is done.
    ASSERT_EVENTUALLY_GE_WITH_TIMEOUT(
        transport->beats.load(),
        5,
        std::chrono::seconds(10),
        std::chrono::milliseconds(10)
    );
    EXPECT_EQ(transport->task_writes.load(), 3);
}

/// @brief it should send task statuses again after a heartbeat that succeeds more than
/// LATE_GAP after the one before it, and only once.
TEST(stateTests, ResendsTaskStatusesAfterALateHeartbeat) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    const synnax::rack::Rack rack{.key = 1, .name = "test_rack"};
    const auto transport = std::make_shared<SlowSetClient>(
        synnax::rack::ontology_id(rack.key).string()
    );
    client->statuses = synnax::status::Client(nullptr, transport, nullptr, nullptr);
    const auto ctx = std::make_shared<driver::task::SynnaxContext>(
        client,
        std::make_shared<driver::task::StatusCache>(client->statuses)
    );
    Task hb(
        rack,
        synnax::task::Task{.name = "state", .type = TASK_TYPE, .internal = true},
        ctx
    );
    x::defer::defer stop([&hb]() { hb.stop(false); });
    // Beat 2 is late and re-sends, and beat 3 sends nothing. Beat 4 starts once beat 3
    // is done.
    ASSERT_EVENTUALLY_GE_WITH_TIMEOUT(
        transport->beats.load(),
        4,
        std::chrono::seconds(10),
        std::chrono::milliseconds(10)
    );
    EXPECT_EQ(transport->task_writes.load(), 2);
}
}
