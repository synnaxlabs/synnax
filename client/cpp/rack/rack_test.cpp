// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <sstream>

#include "gtest/gtest.h"

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/test/test.h"

namespace synnax::rack {
std::mt19937 gen_rand = random_generator(std::move("Ranger Tests"));

/// @brief it should correctly create a rack in the cluster.
TEST(RackTests, testCreateRack) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.racks.create(r));
    ASSERT_EQ(r.name, "test_rack");
}

/// @brief it should correctly retrieve a rack from the cluster.
TEST(RackTests, testRetrieveRack) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.racks.create(r));
    const auto r2 = ASSERT_NIL_P(client.racks.retrieve(r.key));
    ASSERT_EQ(r2.name, "test_rack");
    ASSERT_EQ(r.key, r2.key);
}

/// @brief it should correctly delete a rack from the cluster.
TEST(RackTests, testDeleteRack) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.racks.create(r));
    ASSERT_NIL(client.racks.del(r.key));
    ASSERT_OCCURRED_AS_P(client.racks.retrieve(r.key), x::errors::QUERY);
}
/// @brief it should retrieve a rack by its name.
TEST(RackTests, testRetrieveRackByName) {
    const auto client = new_test_client();

    const auto unique_name = "test_rack_by_name_unique" + std::to_string(rand());
    auto r = Rack(unique_name);
    ASSERT_NIL(client.racks.create(r));
    const auto r2 = ASSERT_NIL_P(client.racks.retrieve(unique_name));
    ASSERT_EQ(r2.name, unique_name);
    ASSERT_EQ(r.key, r2.key);
}

/// @brief it should correctly create and retrieve a rack with a status.
TEST(RackTests, testCreateRackWithStatus) {
    const auto client = new_test_client();
    auto r = Rack("test_rack_with_status");
    r.status.key = "rack-status-key";
    r.status.variant = x::status::variant::SUCCESS;
    r.status.message = "Rack is healthy";
    r.status.time = x::telem::TimeStamp::now();
    r.status.details.rack = 123;
    ASSERT_NIL(client.racks.create(r));
    const auto r2 = ASSERT_NIL_P(client.racks.retrieve(r.key));
    ASSERT_EQ(r2.name, "test_rack_with_status");
    ASSERT_FALSE(r2.status.is_zero());
    ASSERT_EQ(r2.status.key, "rack-status-key");
    ASSERT_EQ(r2.status.variant, x::status::variant::SUCCESS);
    ASSERT_EQ(r2.status.message, "Rack is healthy");
}

/// @brief it should correctly parse StatusDetails from JSON.
TEST(StatusDetailsTests, testParseFromJSON) {
    x::json::json j = {{"rack", 54321}};
    const x::json::Parser parser(j);
    const auto details = StatusDetails::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(details.rack, 54321);
}

/// @brief it should correctly serialize StatusDetails to JSON.
TEST(StatusDetailsTests, testToJSON) {
    constexpr StatusDetails details{.rack = 98765};
    const auto j = details.to_json();
    ASSERT_EQ(j["rack"], 98765);
}

/// @brief it should round-trip StatusDetails through JSON.
TEST(StatusDetailsTests, testRoundTrip) {
    constexpr StatusDetails original{.rack = 11223};
    const auto j = original.to_json();
    const x::json::Parser parser(j);
    const auto recovered = StatusDetails::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(recovered.rack, original.rack);
}

/// @brief it should correctly stream output a rack.
TEST(RackTests, testStreamOutput) {
    auto r = Rack(12345, "my_rack");
    std::ostringstream oss;
    oss << r;
    ASSERT_EQ(oss.str(), "my_rack (12345)");
}
}
