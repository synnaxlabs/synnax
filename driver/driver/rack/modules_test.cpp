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
#include "driver/rack/rack.h"
#include "driver/testutil/testutil.h"
#include "driver/breaker/breaker.h"

class MockModuleFactory : public module::Factory {
public:
    bool configured = false;

    std::unique_ptr<module::Module> configure(
            const std::shared_ptr<synnax::Synnax> &client,
            const synnax::Module &module,
            bool &valid_config,
            json &config_err
    ) override {
        valid_config = false;
        config_err["error"] = "test error";
        return std::make_unique<module::Module>(module);
    }

};

TEST(RackModulesTests, testModuleNominalConfiguration) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto [rack, err] = client->devices.createRack("test_rack");
    ASSERT_FALSE(err) << err.message();
    auto breaker =breaker::Breaker(breaker::Config{
            "test_breaker",
            synnax::TimeSpan(1),
            1,
            1
    });
    std::unique_ptr<MockModuleFactory> factory = std::make_unique<MockModuleFactory>();
    auto modules = device::Modules(rack.key, client, std::move(factory), breaker);
    std::latch latch{1};
    err = modules.start(latch);
    ASSERT_FALSE(err) << err.message();
    auto mod = synnax::Module(
            rack.key,
            "test_module",
            "",
            ""
    );
    auto mod_err = rack.modules.create(mod);
    ASSERT_FALSE(mod_err) << mod_err.message();
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    err = modules.stop();
    ASSERT_FALSE(err) << err.message();
}

