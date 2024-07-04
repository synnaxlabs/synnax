// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.


#include <gtest/gtest.h>
#include "driver/testutil/testutil.h";
#include "driver/driver.h";

// TEST(TaskContextTest, testStateUpdate) {
//     auto client = std::make_shared<Synnax>(new_test_client());
//
//     auto [rack, rack_err] = client->hardware.createRack("my_rack");
//     ASSERT_FALSE(rack_err) << rack_err;
//
//     auto task = Task(
//         rack.key,
//         "test",
//         "test",
//         ""
//     );
//     auto err = rack.tasks.create(task);
//     ASSERT_FALSE(err) << err;
//
//     auto [task_state_chan, task_state_err] = client->channels.retrieve("sy_task_state");
//     ASSERT_FALSE(task_state_err) << err;
//     auto [streamer, streamer_err] = client->telem.openStreamer(StreamerConfig{
//         .channels = {task_state_chan.key},
//     });
//
//     std::this_thread::sleep_for(std::chrono::seconds(1));
//
//     driver::TaskContext ctx(client);
//
//     ctx.setState(driver::TaskState{
//         task.key,
//         "test",
//         json::object({
//             {"key", "value"}
//         })
//     });
//
//     auto [res, exc] = streamer.read();
//     ASSERT_FALSE(exc) << exc;
//     ASSERT_EQ(res.size(), 1);
//     std::cout << res.series->at(0).strings()[0] << std::endl;
// }
