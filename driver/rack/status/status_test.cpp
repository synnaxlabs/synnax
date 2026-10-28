// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/defer/defer.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/rack/status/status.h"

namespace driver::rack::status {
/// @brief it should report nominal driver status via state streamer.
TEST(stateTests, Nominal) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("test_rack"));
    auto ch = ASSERT_NIL_P(
        client->channels.retrieve(synnax::status::STATUS_SET_CHANNEL_NAME)
    );
    auto ctx = std::make_shared<driver::task::SynnaxContext>(client);
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
    auto ctx = std::make_shared<driver::task::SynnaxContext>(client);
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
    auto ctx = std::make_shared<driver::task::SynnaxContext>(client);
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
}
