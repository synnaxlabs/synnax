// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <random>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"

namespace synnax::labjack::device {
namespace {
/// @brief returns a serial number that no other test device has.
std::uint32_t new_serial() {
    static std::mt19937 gen{std::random_device{}()};
    return std::uniform_int_distribution<std::uint32_t>{1, 4'000'000'000}(gen);
}

/// @brief creates a rack and a T7 on it.
std::pair<Device, rack::Rack> create_device(const Synnax &client) {
    auto r = rack::Rack{.name = "labjack_device_rack"};
    EXPECT_NIL(client.racks.create(r));
    auto dev = Device{
        .rack = r.key,
        .name = "T7",
        .serial = new_serial(),
        .connection = CONNECTION_USB,
        .model = MODEL_T_7,
        .ports = T7Ports{},
    };
    EXPECT_NIL(client.labjack.devices.create(dev));
    return {dev, r};
}

/// @brief returns an action that sets the value at the path.
Action set_at(std::vector<std::string> path, x::json::json value) {
    return set(SetPayload{.path = std::move(path), .value = std::move(value)});
}
}

/// @brief it should create a device and retrieve it by key.
TEST(Client, CreatesAndRetrievesByKey) {
    const auto client = new_test_client();
    const auto [dev, r] = create_device(client);
    ASSERT_FALSE(dev.key.is_nil());
    const auto got = ASSERT_NIL_P(client.labjack.devices.retrieve(dev.key));
    ASSERT_EQ(got.name, "T7");
    ASSERT_EQ(got.rack, r.key);
    ASSERT_EQ(got.serial, dev.serial);
}

/// @brief it should retrieve a device by its serial number.
TEST(Client, RetrievesBySerial) {
    const auto client = new_test_client();
    const auto [dev, r] = create_device(client);
    const auto got = ASSERT_NIL_P(
        client.labjack.devices.retrieve(Where{.serial = dev.serial}, x::filter::one)
    );
    ASSERT_EQ(got.key, dev.key);
}

/// @brief it should apply the actions and keep the other fields.
TEST(Client, DispatchesActions) {
    const auto client = new_test_client();
    const auto [dev, r] = create_device(client);
    ASSERT_NIL(client.labjack.devices.dispatch(
        dev.key,
        {set_at({"name"}, "Renamed"), set_at({"connection"}, CONNECTION_ETHERNET)}
    ));
    const auto got = ASSERT_NIL_P(client.labjack.devices.retrieve(dev.key));
    ASSERT_EQ(got.name, "Renamed");
    ASSERT_EQ(got.connection, CONNECTION_ETHERNET);
    ASSERT_EQ(got.serial, dev.serial);
}

/// @brief it should refuse an action that changes the model.
TEST(Client, RefusesModelChange) {
    const auto client = new_test_client();
    const auto [dev, r] = create_device(client);
    ASSERT_OCCURRED_AS(
        client.labjack.devices.dispatch(dev.key, {set_at({"model"}, MODEL_T_4)}),
        x::errors::VALIDATION
    );
}

/// @brief it should delete a device.
TEST(Client, Deletes) {
    const auto client = new_test_client();
    const auto [dev, r] = create_device(client);
    ASSERT_NIL(client.labjack.devices.del({dev.key}));
    ASSERT_OCCURRED_AS_P(
        client.labjack.devices.retrieve(dev.key),
        x::errors::NOT_FOUND
    );
}

/// @brief it should set the key of each device that a list create stores, in order.
TEST(Client, CreatesListInOrder) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "labjack_device_rack"};
    ASSERT_NIL(client.racks.create(r));
    std::vector<Device> devices{
        {.rack = r.key,
         .name = "First",
         .serial = new_serial(),
         .connection = CONNECTION_USB,
         .model = MODEL_T_4,
         .ports = T4Ports{}},
        {.rack = r.key,
         .name = "Second",
         .serial = new_serial(),
         .connection = CONNECTION_USB,
         .model = MODEL_T_8,
         .ports = T8Ports{}},
    };
    ASSERT_NIL(client.labjack.devices.create(devices));
    for (const auto &dev: devices) {
        const auto got = ASSERT_NIL_P(client.labjack.devices.retrieve(dev.key));
        ASSERT_EQ(got.name, dev.name);
    }
}

/// @brief it should return NOT_FOUND for a dispatch to a missing device.
TEST(Client, DispatchToMissingDevice) {
    const auto client = new_test_client();
    ASSERT_OCCURRED_AS(
        client.labjack.devices.dispatch(x::uuid::create(), {set_at({"name"}, "R")}),
        x::errors::NOT_FOUND
    );
}

/// @brief it should return MULTIPLE_RESULTS when more than one device matches.
TEST(Client, RetrieveOneOfMany) {
    const auto client = new_test_client();
    const auto [first, r] = create_device(client);
    auto second = first;
    second.key = x::uuid::UUID{};
    second.serial = new_serial();
    ASSERT_NIL(client.labjack.devices.create(second));
    ASSERT_OCCURRED_AS_P(
        client.labjack.devices.retrieve(Where{.rack = r.key}, x::filter::one),
        x::errors::MULTIPLE_RESULTS
    );
}

/// @brief it should reject a retrieve of one device with a limit.
TEST(Client, RejectsLimitOnRetrieveOne) {
    const auto client = new_test_client();
    ASSERT_OCCURRED_AS_P(
        client.labjack.devices
            .retrieve(Where{.name = "T7"}, x::filter::one, RetrieveOptions{.limit = 1}),
        x::errors::VALIDATION
    );
}
}
