// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// external
#include "gtest/gtest.h"

/// module
#include "x/cpp/xtest/xtest.h"

/// internal
#include "client/cpp/testutil/testutil.h"
#include "driver/rack/state/state.h"

/// @brief tests the nominal heartbeat case.
TEST(HeartbeatTests, testNominal) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack  = ASSERT_NIL_P(client->hardware.create_rack("test_rack"));
    auto ch = ASSERT_NIL_P(client->channels.retrieve("sy_rack_state"));
    auto ctx = std::make_shared<task::SynnaxContext>(client);
    auto hb = rack::state::Task::configure(
        ctx,
        synnax::Task(rack.key, "heartbeat", "heartbeat", "", true)
    );
    auto cmd = task::Command(0, "start", {});
    hb->exec(cmd);
    auto [streamer, strm_err] = client->telem.open_streamer(synnax::StreamerConfig{
        .channels = {ch.key},
    });
    ASSERT_FALSE(strm_err) << strm_err.message();
    json j;
    for (int i = 0; i < 50; i++) {
        auto [frm, msg_err] = streamer.read();
        ASSERT_FALSE(msg_err) << msg_err.message();
        ASSERT_EQ(frm.size(), 1);
        frm.series->at(0).at(-1, j);
        if (j["key"] == rack.key) break;
    }
    EXPECT_EQ(j["key"], rack.key);
    EXPECT_EQ(j["variant"], "success");
    EXPECT_EQ(j["message"], "Driver is running");
    hb->stop(false);
    const auto err = streamer.close();
    ASSERT_FALSE(err) << err.message();
}
