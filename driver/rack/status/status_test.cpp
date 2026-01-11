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
#include "x/cpp/test/test.h"

#include "driver/rack/status/status.h"

/// @brief it should report nominal driver status via state streamer.
TEST(stateTests, testNominal) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("test_rack"));
    auto ch = ASSERT_NIL_P(client->channels.retrieve(synnax::status::STATUS_SET_CHANNEL_NAME));
    auto ctx = std::make_shared<driver::task::SynnaxContext>(client);
    auto task = synnax::task::Task{.name = "state", .type = "state", .internal = true};
    ASSERT_NIL(rack.tasks.create(task));
    auto hb = driver::rack::status::Task::configure(ctx, task);
    auto cmd = synnax::task::Command{.task = task.key, .type = "start", .args = json{}};
    hb->exec(cmd);
    x::defer::defer stop([&hb]() { hb->stop(false); });
    auto streamer = ASSERT_NIL_P(client->telem.open_streamer(
        synnax::framer::StreamerConfig{
            .channels = {ch.key},
        }
    ));
    json j;
    for (int i = 0; i < 50; i++) {
        auto frm = ASSERT_NIL_P(streamer.read());
        ASSERT_EQ(frm.size(), 1);
        frm.series->at(0).at(-1, j);
        if (j["details"]["rack"] == rack.key) break;
    }
    EXPECT_EQ(j["details"]["rack"], rack.key);
    EXPECT_EQ(j["variant"], x::status::VARIANT_SUCCESS);
    EXPECT_EQ(j["message"], "Driver is running");
    ASSERT_NIL(streamer.close());
}
