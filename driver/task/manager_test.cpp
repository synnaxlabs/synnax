// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"
#include "driver/task/task.h"
#include "driver/driver.h"
#include "driver/testutil/testutil.h"
#include "driver/breaker/breaker.h"
#include "nlohmann/json.hpp"

using json = nlohmann::json;

class MockTaskFactory : public task::Factory {
public:
    bool configured = false;

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) override { return {nullptr, false}; }
};

TEST(TaskManagerTests, testModuleNominalConfiguration) {
    auto client = new_test_client();
    auto [rack, err] = client->hardware.createRack("test_rack");
    ASSERT_FALSE(err) << err.message();

    auto breaker = breaker::Breaker(breaker::Config{
        "test_breaker",
        synnax::TimeSpan(1),
        1,
        1
    });
    std::unique_ptr<MockTaskFactory> factory = std::make_unique<MockTaskFactory>();
    auto task_manager = task::Manager(
        rack.key,
        client,
        std::move(factory),
        breaker
    );
    std::atomic done = false;
    err = task_manager.start(done);
    ASSERT_FALSE(err) << err.message();

    auto task_err = synnax::Task(
        rack.key,
        "test_module",
        "",
        ""
    );
    auto t_err = rack.tasks.create(task_err);
    ASSERT_FALSE(t_err) << t_err.message();

    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    err = task_manager.stop();
    ASSERT_FALSE(err) << err.message();
}
