// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>

#include "gtest/gtest.h"

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"

namespace synnax::ni::device {
namespace {
/// @brief creates a rack and a device on it with the given resource name.
std::pair<Device, rack::Rack>
create_device(const Synnax &client, const std::string &resource_name) {
    auto r = rack::Rack{.name = "ni_device_rack"};
    EXPECT_NIL(client.racks.create(r));
    auto dev = Device{
        .rack = r.key,
        .name = "NI 9205",
        .resource_name = resource_name,
        .model = "9205",
    };
    EXPECT_NIL(client.ni.devices.create(dev));
    return {dev, r};
}
}

/// @brief it should create a device and retrieve it by key.
TEST(Client, CreatesAndRetrievesByKey) {
    const auto client = new_test_client();
    const auto [dev, r] = create_device(client, x::uuid::create().to_string());
    ASSERT_FALSE(dev.key.is_nil());
    const auto got = ASSERT_NIL_P(client.ni.devices.retrieve(dev.key));
    ASSERT_EQ(got.name, "NI 9205");
    ASSERT_EQ(got.rack, r.key);
}

/// @brief it should retrieve a device by its resource name.
TEST(Client, RetrievesByResourceName) {
    const auto client = new_test_client();
    const auto resource_name = x::uuid::create().to_string();
    const auto [dev, r] = create_device(client, resource_name);
    const auto got = ASSERT_NIL_P(client.ni.devices.retrieve(
        Where{.resource_name = resource_name},
        x::filter::one
    ));
    ASSERT_EQ(got.key, dev.key);
}

/// @brief it should write the identity group and keep the name as stored.
TEST(Client, UpdatesIdentityAndKeepsName) {
    const auto client = new_test_client();
    const auto [dev, r] = create_device(client, x::uuid::create().to_string());
    IdentityUpdate identity;
    identity.rack = r.key;
    identity.serial = "01A";
    identity.resource_name = dev.resource_name;
    identity.model = "9205";
    ASSERT_NIL(client.ni.devices.update(dev.key, Update(identity)));
    const auto got = ASSERT_NIL_P(client.ni.devices.retrieve(dev.key));
    ASSERT_EQ(got.serial, "01A");
    ASSERT_EQ(got.name, "NI 9205");
}

/// @brief it should delete a device.
TEST(Client, Deletes) {
    const auto client = new_test_client();
    const auto [dev, r] = create_device(client, x::uuid::create().to_string());
    ASSERT_NIL(client.ni.devices.del({dev.key}));
    ASSERT_OCCURRED_AS_P(client.ni.devices.retrieve(dev.key), x::errors::NOT_FOUND);
}
}
