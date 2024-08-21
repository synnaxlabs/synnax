// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <gtest/gtest.h>
#include "nlohmann/json.hpp"
#include "driver/testutil/testutil.h"
#include "driver/opc/mock_server.h"
#include "driver/opc/writer.h"
//#include "mock_server.h"

using json = nlohmann::json;

TEST(opcWriterTest, testWriterConfigurationFromJSON) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());

    // channels for the opc ua nodes
    auto [idx, idx_err] = client->channels.create(
            "index1",
            TIMESTAMP,
            0,
            true
    );
    ASSERT_FALSE(idx_err) << idx_err.message();
    auto [ch1, ch1_err] = client->channels.create(
            "node1",
            INT32,
            idx.key,
            false
    );
    ASSERT_FALSE(ch1_err) << ch1_err.message();
    auto [ch2, ch2_err] = client->channels.create(
            "node2",
            INT32,
            idx.key,
            false
    );
    ASSERT_FALSE(ch2_err) << ch2_err.message();

    // TODO: add device attribute
    auto j = json{
            {
                    "connection", json{
                    {"endpoint", "opc.tcp://0.0.0.0:4841"},
            }
            },
            {
                    "channels",   std::vector<json>{
                    {{"node_id", "node1"}, {"channel", ch1.key}}, //TODO: change
                    {{"node_id", "node2"}, {"channel", ch2.key}}
            }
            }
    };

    auto sy_task = synnax::Task(
            "my_task",
            "opcWrite",
            to_string(j)
    );

    auto mockCtx = std::make_shared<task::MockContext>(client);
    auto mock = MockServerConfig{
            {
                    {1, "node1"},
                    {1, "node2"}
            }
    };

    // get the writer task
    auto task = opc::WriterTask::configure(
                                        mockCtx,
                                        sy_task
                                    );
}


TEST(opcWriterTest, testWriterConfigurationFromJSON){
    auto client = std::make_shared<synnax::Synnax>(new_test_client());

    // connect to opc ua server

}

