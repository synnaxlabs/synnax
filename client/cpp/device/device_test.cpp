// Copyright 2025 Synnax Labs, Inc.
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
#include "x/cpp/xtest/xtest.h"

namespace synnax {
/// @brief it should correctly create a device.
TEST(DeviceTests, testCreateDevice) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.racks.create(r));
    auto d = Device(
        "asdfjahsdfkasjdfhaks",
        "test_device",
        r.key,
        "test_location",
        "test_make",
        "test_model",
        "test_properties"
    );
    ASSERT_NIL(client.devices.create(d));
    ASSERT_EQ(d.name, "test_device");
}

/// @brief it should correctly retrieve a device.
TEST(DeviceTests, testRetrieveDevice) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.racks.create(r));
    auto d = Device(
        "asdfjahsdfkasjdfhaks",
        "test_device",
        r.key,
        "test_location",
        "test_make",
        "test_model",
        "test_properties"
    );
    ASSERT_NIL(client.devices.create(d));
    const auto d2 = ASSERT_NIL_P(client.devices.retrieve(d.key));
    ASSERT_EQ(d2.name, "test_device");
    ASSERT_EQ(d2.key, d.key);
}

/// @brief it should correctly retrieve multiple devices.
TEST(DeviceTests, testRetrieveDevices) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.racks.create(r));

    // Create first device
    auto d1 = Device(
        "device1_key",
        "test_device_1",
        r.key,
        "location_1",
        "make_1",
        "model_1",
        "properties_1"
    );
    ASSERT_NIL(client.devices.create(d1));

    // Create second device
    auto d2 = Device(
        "device2_key",
        "test_device_2",
        r.key,
        "location_2",
        "make_2",
        "model_2",
        "properties_2"
    );
    ASSERT_NIL(client.devices.create(d2));

    // Retrieve both devices
    std::vector keys = {d1.key, d2.key};
    const auto devices = ASSERT_NIL_P(client.devices.retrieve(keys));

    // Verify we got both devices
    ASSERT_EQ(devices.size(), 2);

    // Find and verify first device
    auto it1 = std::find_if(devices.begin(), devices.end(), [&d1](const Device &d) {
        return d.key == d1.key;
    });
    ASSERT_NE(it1, devices.end());
    ASSERT_EQ(it1->name, "test_device_1");
    ASSERT_EQ(it1->location, "location_1");

    // Find and verify second device
    auto it2 = std::find_if(devices.begin(), devices.end(), [&d2](const Device &d) {
        return d.key == d2.key;
    });
    ASSERT_NE(it2, devices.end());
    ASSERT_EQ(it2->name, "test_device_2");
    ASSERT_EQ(it2->location, "location_2");
}

/// @brief it should correctly create multiple devices at once.
TEST(DeviceTests, testCreateDevices) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.racks.create(r));

    // Create a vector of devices to add
    std::vector devices = {
        Device(
            "device1_key",
            "test_device_1",
            r.key,
            "location_1",
            "make_1",
            "model_1",
            "properties_1"
        ),
        Device(
            "device2_key",
            "test_device_2",
            r.key,
            "location_2",
            "make_2",
            "model_2",
            "properties_2"
        ),
        Device(
            "device3_key",
            "test_device_3",
            r.key,
            "location_3",
            "make_3",
            "model_3",
            "properties_3"
        )
    };

    // Create all devices at once
    ASSERT_NIL(client.devices.create(devices));

    // Verify each device was created correctly by retrieving them
    for (const auto &device: devices) {
        const auto retrieved = ASSERT_NIL_P(client.devices.retrieve(device.key));
        ASSERT_EQ(retrieved.key, device.key);
        ASSERT_EQ(retrieved.name, device.name);
        ASSERT_EQ(retrieved.rack, r.key);
        ASSERT_EQ(retrieved.location, device.location);
        ASSERT_EQ(retrieved.make, device.make);
        ASSERT_EQ(retrieved.model, device.model);
        ASSERT_EQ(retrieved.properties, device.properties);
    }

    // Also test retrieving all devices at once
    std::vector<std::string> keys;
    keys.reserve(devices.size());
    for (const auto &device: devices)
        keys.push_back(device.key);

    const auto retrieved_devices = ASSERT_NIL_P(client.devices.retrieve(keys));
    ASSERT_EQ(retrieved_devices.size(), devices.size());

    // Create a map for easier lookup
    auto device_map = map_device_keys(retrieved_devices);

    // Verify each device is in the retrieved set
    for (const auto &device: devices) {
        ASSERT_TRUE(device_map.find(device.key) != device_map.end());
        const auto &retrieved = device_map[device.key];
        ASSERT_EQ(retrieved.name, device.name);
        ASSERT_EQ(retrieved.rack, r.key);
    }
}

/// @brief it should correctly handle the configured field.
TEST(DeviceTests, testDeviceConfigured) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.racks.create(r));

    auto d1 = Device(
        "device1_key",
        "test_device_1",
        r.key,
        "location_1",
        "make_1",
        "model_1",
        "properties_1"
    );
    d1.configured = false;
    ASSERT_NIL(client.devices.create(d1));

    auto d2 = Device(
        "device2_key",
        "test_device_2",
        r.key,
        "location_2",
        "make_2",
        "model_2",
        "properties_2"
    );
    d2.configured = true;
    ASSERT_NIL(client.devices.create(d2));

    const auto retrieved1 = ASSERT_NIL_P(client.devices.retrieve(d1.key));
    ASSERT_FALSE(retrieved1.configured);

    const auto retrieved2 = ASSERT_NIL_P(client.devices.retrieve(d2.key));
    ASSERT_TRUE(retrieved2.configured);

    std::vector keys = {d1.key, d2.key};
    const auto devices = ASSERT_NIL_P(client.devices.retrieve(keys));
    auto device_map = map_device_keys(devices);

    ASSERT_FALSE(device_map[d1.key].configured);
    ASSERT_TRUE(device_map[d2.key].configured);
}

/// @brief it should correctly handle retrieving devices after deletion.
TEST(DeviceTests, testRetrieveDevicesAfterDeletion) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.racks.create(r));

    // Create two devices
    auto d1 = Device(
        "device1_key",
        "test_device_1",
        r.key,
        "location_1",
        "make_1",
        "model_1",
        "properties_1"
    );
    ASSERT_NIL(client.devices.create(d1));

    auto d2 = Device(
        "device2_key",
        "test_device_2",
        r.key,
        "location_2",
        "make_2",
        "model_2",
        "properties_2"
    );
    ASSERT_NIL(client.devices.create(d2));

    // Delete the first device
    ASSERT_NIL(client.devices.del(d1.key));

    // Try to retrieve both devices
    std::vector<std::string> keys;
    keys.push_back(d1.key);
    keys.push_back(d2.key);
    auto devices = ASSERT_NIL_P(client.devices.retrieve(keys, true));

    // Assert that we got at least one device back (the non-deleted one)
    ASSERT_GE(devices.size(), 1);

    // Verify that the remaining device is the second one
    bool found = false;
    for (const Device &device: devices) {
        if (device.key == d2.key) {
            ASSERT_EQ(device.name, "test_device_2");
            found = true;
            break;
        }
    }
    ASSERT_TRUE(found);
}

/// @brief it should correctly delete a device.
TEST(DeviceTests, testDeleteDevice) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.racks.create(r));

    auto d = Device(
        "device_key",
        "test_device",
        r.key,
        "test_location",
        "test_make",
        "test_model",
        "test_properties"
    );
    ASSERT_NIL(client.devices.create(d));

    ASSERT_NIL(client.devices.del(d.key));

    auto [_, err] = client.devices.retrieve(d.key);
    ASSERT_TRUE(err);
    ASSERT_MATCHES(err, xerrors::NOT_FOUND);
}

/// @brief it should correctly delete multiple devices.
TEST(DeviceTests, testDeleteDevices) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.racks.create(r));

    auto d1 = Device(
        "device1_key",
        "test_device_1",
        r.key,
        "location_1",
        "make_1",
        "model_1",
        "properties_1"
    );
    ASSERT_NIL(client.devices.create(d1));

    auto d2 = Device(
        "device2_key",
        "test_device_2",
        r.key,
        "location_2",
        "make_2",
        "model_2",
        "properties_2"
    );
    ASSERT_NIL(client.devices.create(d2));

    const std::vector keys = {d1.key, d2.key};
    ASSERT_NIL(client.devices.del(keys));

    ASSERT_OCCURRED_AS_P(client.devices.retrieve(keys), xerrors::NOT_FOUND);
}

/// @brief it should correctly handle ignore_not_found flag.
TEST(DeviceTests, testRetrieveDeviceIgnoreNotFound) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.racks.create(r));

    // Test retrieving non-existent device with ignore_not_found=true
    const auto [device1, err1] = client.devices.retrieve(
        "nonexistent_key",
        true // ignore_not_found
    );
    ASSERT_FALSE(err1);
    ASSERT_TRUE(device1.key.empty());

    // Test retrieving multiple devices with some not found
    auto d1 = Device(
        "device1_key",
        "test_device_1",
        r.key,
        "location_1",
        "make_1",
        "model_1",
        "properties_1"
    );
    ASSERT_NIL(client.devices.create(d1));

    std::vector<std::string> keys = {d1.key, "nonexistent_key"};
    const auto [devices, err2] = client.devices.retrieve(
        keys,
        true // ignore_not_found
    );
    ASSERT_FALSE(err2);
    ASSERT_EQ(devices.size(), 1);
    ASSERT_EQ(devices[0].key, d1.key);
}
}
