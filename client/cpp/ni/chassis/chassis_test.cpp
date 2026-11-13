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
}
