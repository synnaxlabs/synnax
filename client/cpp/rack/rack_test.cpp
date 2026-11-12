// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <set>
#include <sstream>
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"

namespace synnax::rack {
// NOLINTBEGIN(cppcoreguidelines-avoid-non-const-global-variables): one seed per suite,
// printed so that a failing run can be replayed.
std::mt19937 gen_rand = random_generator("Rack Tests");
// NOLINTEND(cppcoreguidelines-avoid-non-const-global-variables)

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

    const auto unique_name = "test_rack_by_name_unique" + std::to_string(gen_rand());
    auto r = Rack{.name = unique_name};
    ASSERT_NIL(client.racks.create(r));
    const auto r2 = ASSERT_NIL_P(client.racks.retrieve(unique_name));
    ASSERT_EQ(r2.name, unique_name);
    ASSERT_EQ(r.key, r2.key);
}

/// @brief it should reject a limit or a cursor on a retrieve of one rack.
TEST(RackTests, testRetrieveRackRejectsPage) {
    const auto client = new_test_client();
    for (const auto &options:
         {RetrieveOptions{.limit = 1}, RetrieveOptions{.after = Cursor{}}})
        for (const auto &err: {
                 client.racks.retrieve("rack", options).second,
                 client.racks.retrieve({.name = "rack"}, x::filter::one, options).second
             }) {
            ASSERT_MATCHES(err, x::errors::VALIDATION);
            ASSERT_EQ(
                err.data,
                "a query for one record takes no limit or cursor, since a page could "
                "hide a second match"
            );
        }
}

/// @brief it should page a retrieve of the racks that a filter matches.
TEST(RackTests, testRetrieveWherePage) {
    const auto client = new_test_client();
    const auto name = "test_rack_page_" + std::to_string(gen_rand());
    for (int i = 0; i < 3; ++i) {
        auto r = Rack{.name = name};
        ASSERT_NIL(client.racks.create(r));
    }
    const auto where = Where{.name = name};
    const auto first = ASSERT_NIL_P(client.racks.retrieve(where, {.limit = 2}));
    ASSERT_EQ(first.size(), 2);
    const auto rest = ASSERT_NIL_P(
        client.racks.retrieve(where, {.after = Cursor{.key = first[1].key}})
    );
    ASSERT_EQ(rest.size(), 1);
}

TEST(RackTests, testCreateTaskOnCreatedRack) {
    const auto client = new_test_client();
    auto r = Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    task::Task t{
        .name = "cat",
        .type = "pagerduty_alert",
        .internal = false,
    };
    ASSERT_NIL(r.tasks.create(t));
    ASSERT_EQ(t.rack, r.key);
}

TEST(RackTests, testCreateTaskOnRetrieveRack) {
    const auto client = new_test_client();
    auto r = Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    auto retrieved = ASSERT_NIL_P(client.racks.retrieve(r.key));
    task::Task t{
        .name = "cat",
        .type = "pagerduty_alert",
        .internal = false,
    };
    ASSERT_NIL(retrieved.tasks.create(t));
    ASSERT_EQ(t.rack, r.key);
    ASSERT_EQ(t.rack, retrieved.key);
}

/// @brief retrieving a rack without include_status should not populate the
/// status field. Status is persisted separately by the Status service, so it
/// is only joined onto the rack when the caller opts in.
TEST(RackTests, testCreateRackWithStatusNotIncludedByDefault) {
    const auto client = new_test_client();
    auto r = Rack{
        .name = "test_rack_status_default",
        .status = Status{
            .key = "rack-status-default-key",
            .variant = synnax::status::VARIANT_SUCCESS,
            .message = "Rack is healthy",
            .time = x::telem::TimeStamp::now(),
        }
    };
    ASSERT_NIL(client.racks.create(r));
    const auto r2 = ASSERT_NIL_P(client.racks.retrieve(r.key));
    ASSERT_EQ(r2.name, "test_rack_status_default");
    ASSERT_FALSE(r2.status.has_value());
}

/// @brief retrieving a rack with include_status=true should join the rack's
/// current operational status from the Status service onto the response.
TEST(RackTests, testCreateRackWithStatusIncluded) {
    const auto client = new_test_client();
    auto r = Rack{
        .name = "test_rack_with_status",
        .status = Status{
            .key = "rack-status-key",
            .variant = synnax::status::VARIANT_SUCCESS,
            .message = "Rack is healthy",
            .time = x::telem::TimeStamp::now(),
        }
    };
    ASSERT_NIL(client.racks.create(r));
    const auto r2 = ASSERT_NIL_P(
        client.racks.retrieve(r.key, {.include_status = true})
    );
    ASSERT_EQ(r2.name, "test_rack_with_status");
    ASSERT_TRUE(r2.status.has_value());
    ASSERT_EQ(r2.status.value().variant, synnax::status::VARIANT_SUCCESS);
    ASSERT_EQ(r2.status.value().message, "Rack is healthy");
}

/// @brief it should create a rack with integrations and retrieve them.
TEST(RackTests, testCreateRackWithIntegrations) {
    const auto client = new_test_client();
    auto r = Rack{
        .name = "test_rack_with_integrations",
        .integrations = {"ni", "opc", "modbus"},
    };
    ASSERT_NIL(client.racks.create(r));
    const auto r2 = ASSERT_NIL_P(client.racks.retrieve(r.key));
    ASSERT_EQ(r2.integrations.size(), 3);
    ASSERT_EQ(r2.integrations[0], "ni");
    ASSERT_EQ(r2.integrations[1], "opc");
    ASSERT_EQ(r2.integrations[2], "modbus");
}

/// @brief it should retrieve the racks that match any branch of an or.
TEST(RackTests, testRetrieveWhere) {
    const auto client = new_test_client();
    auto a = Rack{.name = "test_rack_where_a", .integrations = {"labjack"}};
    auto b = Rack{.name = "test_rack_where_b" + std::to_string(gen_rand())};
    auto c = Rack{.name = "test_rack_where_c"};
    ASSERT_NIL(client.racks.create(a));
    ASSERT_NIL(client.racks.create(b));
    ASSERT_NIL(client.racks.create(c));
    const auto racks = ASSERT_NIL_P(client.racks.retrieve(
        (Where{.integrations = "labjack"} || Where{.name = b.name}) &&
        Where{.key = {a.key, b.key, c.key, 4123456789}}
    ));
    std::set<Key> keys;
    for (const auto &r: racks)
        keys.insert(r.key);
    ASSERT_EQ(keys, (std::set<Key>{a.key, b.key}));
}

/// @brief it should retrieve the one rack that a filter matches.
TEST(RackTests, testRetrieveOneWhere) {
    const auto client = new_test_client();
    auto r = Rack{.name = "test_rack_one_" + std::to_string(gen_rand())};
    ASSERT_NIL(client.racks.create(r));
    const auto r2 = ASSERT_NIL_P(
        client.racks.retrieve({.name = r.name}, x::filter::one)
    );
    ASSERT_EQ(r2.key, r.key);
}

/// @brief it should name the filter in the error when no rack matches it.
TEST(RackTests, testRetrieveOneWhereNotFound) {
    const auto client = new_test_client();
    const auto name = "test_rack_none_" + std::to_string(gen_rand());
    const auto [_, err] = client.racks.retrieve({.name = name}, x::filter::one);
    ASSERT_MATCHES(err, x::errors::NOT_FOUND);
    ASSERT_EQ(err.data, "Found no rack where name = \"" + name + "\"");
}

/// @brief it should return an error when more than one rack matches a filter.
TEST(RackTests, testRetrieveOneWhereMultipleFound) {
    const auto client = new_test_client();
    const auto name = "test_rack_twice_" + std::to_string(gen_rand());
    for (int i = 0; i < 2; ++i) {
        auto r = Rack{.name = name};
        ASSERT_NIL(client.racks.create(r));
    }
    ASSERT_OCCURRED_AS_P(
        client.racks.retrieve({.name = name}, x::filter::one),
        x::errors::MULTIPLE_RESULTS
    );
}

/// @brief it should set the integrations and version of a rack with a dispatch, and
/// keep its name.
TEST(RackTests, testDispatchSetsIntegrationsAndVersion) {
    const auto client = new_test_client();
    auto r = Rack{.name = "test_rack_dispatch", .integrations = {"ni"}};
    ASSERT_NIL(client.racks.create(r));
    const std::vector<std::string> integrations = {"ni", "opc", "arc"};
    ASSERT_NIL(client.racks.dispatch(
        r.key,
        {set({.path = {"integrations"}, .value = integrations}),
         set({.path = {"version"}, .value = "1.2.3"})}
    ));
    const auto got = ASSERT_NIL_P(client.racks.retrieve(r.key));
    ASSERT_EQ(got.integrations, integrations);
    ASSERT_EQ(got.version, "1.2.3");
    ASSERT_EQ(got.name, "test_rack_dispatch");
}

/// @brief it should return a not found error when it dispatches to a rack that does not
/// exist.
TEST(RackTests, testDispatchNotFound) {
    const auto client = new_test_client();
    const auto r = ASSERT_NIL_P(client.racks.create("test_rack_dispatch_deleted"));
    ASSERT_NIL(client.racks.del(r.key));
    ASSERT_OCCURRED_AS(
        client.racks.dispatch(r.key, {set({.path = {"name"}, .value = "renamed"})}),
        x::errors::NOT_FOUND
    );
}

/// @brief it should correctly stream output a rack.
TEST(RackTests, testStreamOutput) {
    auto r = Rack{.key = 12345, .name = "my_rack"};
    std::ostringstream oss;
    oss << r;
    ASSERT_EQ(oss.str(), "my_rack (12345)");
}

TEST(RackTests, testRackKeyNode) {
    ASSERT_EQ(rack_key_node(65538), 1);
}

TEST(RackTests, testRackKeyNodeZero) {
    ASSERT_EQ(rack_key_node(0), 0);
}

TEST(RackTests, testRackKeyNodeHighBits) {
    const Key key = static_cast<Key>(0xFFFF) << 16;
    ASSERT_EQ(rack_key_node(key), 0xFFFF);
}
}
