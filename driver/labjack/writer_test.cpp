// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <stdio.h>

#include "client/cpp/synnax.h"
#include "driver/labjack/task.h"
#include "driver/labjack/writer.h"
#include "driver/testutil/testutil.h"

#include <include/gtest/gtest.h>
#include "glog/logging.h"
#include "nlohmann/json.hpp"

using json = nlohmann::json;

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//                                                   Basic Tests                                                //
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////

TEST(write_tests, labjack_t4){

    LOG(INFO)  << "Test labjack writes t4";

    auto client std::make_shared<synnax::Synnax>(new_test_client());

    auto [state_idx, tErr1] = client->channels.create("do_state_idx", synnax::TIMESTAMP, 0, true);
    ASSERT_FALSE(tErr1) << tErr1.message();

    auto [cmd_idx, tErr2] = channel->channels.create("do_cmd_idx", synnax::INT32, true);
    ASSERT_FALSE(tErr2) << tErr2.message();

    // TODO: test schematic using a float channel
    auto [state, aErr] = client->channels.create("do_state", synnax::SY_UINT8, state_idx.key, false);
    ASSERT_FALSE(aErr) << aErr.message();

    auto [cmd, cErr] = client->channels.create("do_cmd", synnax::SY_UINT8, cmd_idx.key, false);
    ASSERT_FALSE(cErr) << cErr.message();

    // create a writer to write commands out the cmd pipe
    auto cmd_writer_config = synnax::WriterConfig{
        .channels = std::vector<synnax::ChannelKey>{cmd_idx.key, cmd.key},
        .start = TimeStamp::now(),
        .mode = synnax::StreamOnly
    };


}