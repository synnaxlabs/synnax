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
#include "synnax/testutil/testutil.h"
#include "driver/breaker/breaker.h"

class MockModuleFactory : public module::Factory {
    std::pair<std::unique_ptr<module::Module>, freighter::Error>
    configure(const std::shared_ptr<synnax::Synnax> &client, const synnax::Module &module) override {
        return {std::make_unique<module::Module>(module), freighter::NIL};
    }
};

TEST(RackModulesTests, testModuls) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto [rack, err] = client->devices.createRack("test_rack");
    ASSERT_FALSE(err) << err.message();
    auto modules = device::Modules(rack.key, client, std::make_unique<MockModuleFactory>(), breaker::Breaker(breaker::Config{
            "test_breaker",
            synnax::TimeSpan(1),
            1,
            1
    }));

    std::latch latch{1};
    err = modules.start(latch);
    ASSERT_FALSE(err) << err.message();
}

