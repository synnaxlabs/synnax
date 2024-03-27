// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// GTest
#include <gtest/gtest.h>
#include <latch>

/// Local headers.
#include "driver/driver/driver.h"
#include "driver/driver/testutil/testutil.h"
#include "driver/driver/breaker/breaker.h"
#include "nlohmann/json.hpp"

using json = nlohmann::json;

class MockModuleFactory : public driver::TaskFactory {
public:
    bool configured = false;

    std::unique_ptr<driver::Task> createTask(
        const std::shared_ptr<driver::TaskContext>& ctx,
        const driver::Task& task
    ) {
        return std::make_unique<driver::Task>();
    }
};

TEST(RackModulesTests, testModuleNominalConfiguration) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto [rack, err] = client->hardware.createRack("test_rack");
    ASSERT_FALSE(err) << err.message();
    auto breaker = breaker::Breaker(breaker::Config{
        "test_breaker",
        synnax::TimeSpan(1),
        1,
        1
    });
    std::unique_ptr<MockModuleFactory> factory = std::make_unique<MockModuleFactory>();
    auto modules = driver::TaskManager(rack.key, client, std::move(factory), breaker);
    std::latch latch{1};
    err = modules.start(latch);
    ASSERT_FALSE(err) << err.message();
    auto mod = synnax::Task(
        rack.key,
        "test_module",
        "",
        ""
    );
    auto mod_err = rack.tasks.create(mod);
    ASSERT_FALSE(mod_err) << mod_err.message();
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    err = modules.stop();
    ASSERT_FALSE(err) << err.message();
}
