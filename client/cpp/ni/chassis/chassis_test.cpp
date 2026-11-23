// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"

namespace synnax::ni::chassis {
/// @brief it should create, rename, and delete a chassis.
TEST(Client, CreatesUpdatesAndDeletes) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "ni_chassis_rack"};
    ASSERT_NIL(client.racks.create(r));
    auto chassis = Chassis{
        .rack = r.key,
        .name = "cDAQ-9189",
        .resource_name = x::uuid::create().to_string(),
        .model = "cDAQ-9189",
    };
    ASSERT_NIL(client.ni.chassis.create(chassis));
    ConfigurationUpdate configuration;
    configuration.name = "Bench chassis";
    ASSERT_NIL(client.ni.chassis.update(chassis.key, Update(configuration)));
    const auto got = ASSERT_NIL_P(client.ni.chassis.retrieve(chassis.key));
    ASSERT_EQ(got.name, "Bench chassis");
    ASSERT_EQ(got.resource_name, chassis.resource_name);
    ASSERT_NIL(client.ni.chassis.del({chassis.key}));
    ASSERT_OCCURRED_AS_P(client.ni.chassis.retrieve(chassis.key), x::errors::NOT_FOUND);
}

/// @brief it should retrieve chassis by a filter.
TEST(Client, RetrievesByFilter) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "ni_chassis_rack"};
    ASSERT_NIL(client.racks.create(r));
    const auto serial = x::uuid::create().to_string();
    auto chassis = Chassis{
        .rack = r.key,
        .name = "cDAQ-9189",
        .serial = serial,
        .resource_name = x::uuid::create().to_string(),
        .model = "cDAQ-9189",
    };
    ASSERT_NIL(client.ni.chassis.create(chassis));
    const auto list = ASSERT_NIL_P(client.ni.chassis.retrieve(Where{.serial = serial}));
    ASSERT_EQ(list.size(), 1);
    ASSERT_EQ(list[0].key, chassis.key);
    const auto one = ASSERT_NIL_P(
        client.ni.chassis.retrieve(Where{.serial = serial}, x::filter::one)
    );
    ASSERT_EQ(one.key, chassis.key);
}

/// @brief it should reject a retrieve of one chassis with a limit.
TEST(Client, RejectsLimitOnRetrieveOne) {
    const auto client = new_test_client();
    ASSERT_OCCURRED_AS_P(
        client.ni.chassis.retrieve(
            Where{.name = "cDAQ-9189"},
            x::filter::one,
            RetrieveOptions{.limit = 1}
        ),
        x::errors::VALIDATION
    );
}
}
