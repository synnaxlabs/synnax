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
#include "x/cpp/test/test.h"

std::mt19937 gen_rand = random_generator(std::move("Ranger Tests"));

namespace synnax::rack {
/// @brief it should correctly create a rack in the cluster.
TEST(RackTests, testCreateRack) {
    const auto client = new_test_client();
    auto r = Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    ASSERT_EQ(r.name, "test_rack");
}

/// @brief it should correctly retrieve a rack from the cluster.
TEST(RackTests, testRetrieveRack) {
    const auto client = new_test_client();
    auto r = Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    const auto r2 = ASSERT_NIL_P(client.racks.retrieve(r.key));
    ASSERT_EQ(r2.name, "test_rack");
    ASSERT_EQ(r.key, r2.key);
}

/// @brief it should correctly delete a rack from the cluster.
TEST(RackTests, testDeleteRack) {
    const auto client = new_test_client();
    auto r = Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    ASSERT_NIL(client.racks.del(r.key));
    ASSERT_OCCURRED_AS_P(client.racks.retrieve(r.key), x::errors::QUERY);
}
/// @brief it should retrieve a rack by its name.
TEST(RackTests, testRetrieveRackByName) {
    const auto client = new_test_client();

    const auto unique_name = "test_rack_by_name_unique" + std::to_string(rand());
    auto r = Rack{.name = unique_name};
    ASSERT_NIL(client.racks.create(r));
    const auto r2 = ASSERT_NIL_P(client.racks.retrieve(unique_name));
    ASSERT_EQ(r2.name, unique_name);
    ASSERT_EQ(r.key, r2.key);
}

/// @brief it should correctly create and retrieve a rack with a status.
TEST(RackTests, testCreateRackWithStatus) {
    const auto client = new_test_client();
    auto r = Rack{
        .name = "test_rack_with_status",
        .status = rack::Status{
            .key = "rack-status-key",
            .variant = x::status::VARIANT_SUCCESS,
            .message = "Rack is healthy",
            .time = x::telem::TimeStamp::now(),
            .details = StatusDetails{.rack = 123},
        }
    };
    ASSERT_NIL(client.racks.create(r));
    const auto r2 = ASSERT_NIL_P(client.racks.retrieve(r.key));
    ASSERT_EQ(r2.name, "test_rack_with_status");
    ASSERT_EQ(r2.status->key, "rack-status-key");
    ASSERT_EQ(r2.status->variant, x::status::VARIANT_SUCCESS);
    ASSERT_EQ(r2.status->message, "Rack is healthy");
}
}
