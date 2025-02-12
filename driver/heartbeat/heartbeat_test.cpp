// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"
#include "driver/heartbeat/heartbeat.h"
#include "driver/testutil/testutil.h"

/// @brief tests the nominal heartbeat case.
TEST(HeartbeatTests, testNominal) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto [rack, rack_err] = client->hardware.create_rack("test_rack");
    ASSERT_FALSE(rack_err) << rack_err.message();
    auto [ch, ch_err] = client->channels.retrieve("sy_rack_heartbeat");
    auto ctx = std::make_shared<task::SynnaxContext>(client);
    auto hb = heartbeat::Heartbeat::configure(
        ctx,
        synnax::Task()
    );
    auto cmd = task::Command(0, "start", {});
    hb->exec(cmd);
    auto [streamer, strm_err] = client->telem.open_streamer(synnax::StreamerConfig{
        .channels = {ch.key},
    });
    ASSERT_FALSE(strm_err) << strm_err.message();
    auto [frm, msg_err] = streamer.read();
    ASSERT_FALSE(msg_err) << msg_err.message();
    ASSERT_EQ(frm.size(), 1);
    hb->stop();
}
