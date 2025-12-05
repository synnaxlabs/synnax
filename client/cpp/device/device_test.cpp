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

std::mt19937 gen_rand_device = random_generator("Device Tests");

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
    const std::vector keys = {d1.key, d2.key};
    const auto devices = ASSERT_NIL_P(client.devices.retrieve(keys));

    // Verify we got both devices
    ASSERT_EQ(devices.size(), 2);

    // Find and verify first device
    const auto it1 = std::find_if(devices.begin(), devices.end(), [&d1](const Device &d) {
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

    const std::vector keys = {d1.key, d2.key};
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

    // Verify deleted device returns NOT_FOUND
    ASSERT_OCCURRED_AS_P(client.devices.retrieve(d1.key), xerrors::NOT_FOUND);

    // Verify remaining device can still be retrieved
    const auto retrieved = ASSERT_NIL_P(client.devices.retrieve(d2.key));
    ASSERT_EQ(retrieved.key, d2.key);
    ASSERT_EQ(retrieved.name, "test_device_2");
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

    ASSERT_OCCURRED_AS_P(client.devices.retrieve(d.key), xerrors::NOT_FOUND);
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

/// @brief it should retrieve devices using a DeviceRetrieveRequest with keys and names.
TEST(DeviceTests, testRetrieveWithRequest) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.racks.create(r));
    const auto rand = std::to_string(gen_rand_device());
    auto d1 = Device(
        "req_d1_" + rand,
        "req_dev_1_" + rand,
        r.key,
        "loc_a",
        "make_a",
        "model_a",
        "p1"
    );
    auto d2 = Device(
        "req_d2_" + rand,
        "req_dev_2_" + rand,
        r.key,
        "loc_b",
        "make_b",
        "model_b",
        "p2"
    );
    auto d3 = Device(
        "req_d3_" + rand,
        "req_dev_3_" + rand,
        r.key,
        "loc_c",
        "make_c",
        "model_c",
        "p3"
    );
    ASSERT_NIL(client.devices.create(d1));
    ASSERT_NIL(client.devices.create(d2));
    ASSERT_NIL(client.devices.create(d3));
    DeviceRetrieveRequest req_keys{};
    req_keys.keys = {d1.key, d3.key};
    const auto devices_keys = ASSERT_NIL_P(client.devices.retrieve(req_keys));
    ASSERT_EQ(devices_keys.size(), 2);
    auto dm = map_device_keys(devices_keys);
    ASSERT_TRUE(dm.find(d1.key) != dm.end());
    ASSERT_TRUE(dm.find(d3.key) != dm.end());
    DeviceRetrieveRequest req_names{};
    req_names.names = {d1.name, d2.name};
    const auto devices_names = ASSERT_NIL_P(client.devices.retrieve(req_names));
    ASSERT_EQ(devices_names.size(), 2);
}
/// @brief it should retrieve devices with limit and offset pagination.
TEST(DeviceTests, testRetrieveWithLimitOffset) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.racks.create(r));
    const auto rand = std::to_string(gen_rand_device());
    const auto make = "limit_make_" + rand;
    std::vector<Device> devices;
    for (int i = 0; i < 5; ++i) {
        auto d = Device(
            "limit_d_" + rand + "_" + std::to_string(i),
            "limit_dev_" + rand + "_" + std::to_string(i),
            r.key,
            "loc",
            make,
            "model",
            "props"
        );
        ASSERT_NIL(client.devices.create(d));
        devices.push_back(d);
    }
    DeviceRetrieveRequest req_limit;
    req_limit.makes = {make};
    req_limit.limit = 2;
    const auto devices_limited = ASSERT_NIL_P(client.devices.retrieve(req_limit));
    ASSERT_EQ(devices_limited.size(), 2);
    DeviceRetrieveRequest req_offset;
    req_offset.makes = {make};
    req_offset.limit = 2;
    req_offset.offset = 2;
    const auto devices_offset = ASSERT_NIL_P(client.devices.retrieve(req_offset));
    ASSERT_EQ(devices_offset.size(), 2);
    bool different = true;
    for (const auto &da: devices_limited)
        for (const auto &db: devices_offset)
            if (da.key == db.key) different = false;
    ASSERT_TRUE(different);
}
/// @brief it should correctly create and retrieve a device with a status.
TEST(DeviceTests, testCreateDeviceWithStatus) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.racks.create(r));
    const auto rand = std::to_string(gen_rand_device());
    auto d = Device(
        "status_dev_" + rand,
        "device_with_status",
        r.key,
        "location",
        "make",
        "model",
        "properties"
    );
    d.status.variant = status::variant::SUCCESS;
    d.status.message = "Device is connected";
    d.status.time = telem::TimeStamp::now();
    d.status.details.rack = r.key;
    d.status.details.device = d.key;
    ASSERT_NIL(client.devices.create(d));
    const auto d2 = ASSERT_NIL_P(
        client.devices.retrieve(d.key, {.include_status = true})
    );
    ASSERT_EQ(d2.name, "device_with_status");
    ASSERT_FALSE(d2.status.is_zero());
    ASSERT_EQ(d2.status.variant, status::variant::SUCCESS);
    ASSERT_EQ(d2.status.message, "Device is connected");
    ASSERT_EQ(d2.status.details.rack, r.key);
}

/// @brief it should correctly retrieve multiple devices with statuses.
TEST(DeviceTests, testRetrieveDevicesWithStatus) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.racks.create(r));
    const auto rand = std::to_string(gen_rand_device());
    auto d1 = Device(
        "status_d1_" + rand,
        "device_1_status",
        r.key,
        "loc1",
        "make1",
        "model1",
        "props1"
    );
    d1.status.variant = status::variant::SUCCESS;
    d1.status.message = "Device 1 OK";
    d1.status.time = telem::TimeStamp::now();
    auto d2 = Device(
        "status_d2_" + rand,
        "device_2_status",
        r.key,
        "loc2",
        "make2",
        "model2",
        "props2"
    );
    d2.status.variant = status::variant::WARNING;
    d2.status.message = "Device 2 warning";
    d2.status.time = telem::TimeStamp::now();
    ASSERT_NIL(client.devices.create(d1));
    ASSERT_NIL(client.devices.create(d2));
    const std::vector<std::string> keys = {d1.key, d2.key};
    const auto devices = ASSERT_NIL_P(
        client.devices.retrieve(keys, {.include_status = true})
    );
    ASSERT_EQ(devices.size(), 2);
    auto dm = map_device_keys(devices);
    ASSERT_FALSE(dm[d1.key].status.is_zero());
    ASSERT_EQ(dm[d1.key].status.variant, status::variant::SUCCESS);
    ASSERT_EQ(dm[d1.key].status.message, "Device 1 OK");
    ASSERT_FALSE(dm[d2.key].status.is_zero());
    ASSERT_EQ(dm[d2.key].status.variant, status::variant::WARNING);
    ASSERT_EQ(dm[d2.key].status.message, "Device 2 warning");
}

/// @brief it should correctly parse DeviceStatusDetails from JSON.
TEST(DeviceStatusDetailsTests, testParseFromJSON) {
    json j = {{"rack", 12345}, {"device", "device-abc-123"}};
    const xjson::Parser parser(j);
    const auto details = DeviceStatusDetails::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(details.rack, 12345);
    ASSERT_EQ(details.device, "device-abc-123");
}

/// @brief it should correctly serialize DeviceStatusDetails to JSON.
TEST(DeviceStatusDetailsTests, testToJSON) {
    const DeviceStatusDetails details{
        .rack = 67890,
        .device = "device-xyz-456",
    };
    const auto j = details.to_json();
    ASSERT_EQ(j["rack"], 67890);
    ASSERT_EQ(j["device"], "device-xyz-456");
}

/// @brief it should round-trip DeviceStatusDetails through JSON.
TEST(DeviceStatusDetailsTests, testRoundTrip) {
    DeviceStatusDetails original{
        .rack = 11111,
        .device = "round-trip-device",
    };
    const auto j = original.to_json();
    xjson::Parser parser(j);
    auto recovered = DeviceStatusDetails::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(recovered.rack, original.rack);
    ASSERT_EQ(recovered.device, original.device);
}

/// @brief it should correctly parse a Device from JSON.
TEST(DeviceTests, testParseFromJSON) {
    json j = {
        {"key", "json-device-key"},
        {"name", "json-device-name"},
        {"rack", 99999},
        {"location", "json-location"},
        {"make", "json-make"},
        {"model", "json-model"},
        {"properties", "{\"custom\": true}"},
        {"configured", true}
    };
    xjson::Parser parser(j);
    auto d = Device::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(d.key, "json-device-key");
    ASSERT_EQ(d.name, "json-device-name");
    ASSERT_EQ(d.rack, 99999);
    ASSERT_EQ(d.make, "json-make");
    ASSERT_EQ(d.model, "json-model");
    ASSERT_EQ(d.properties, "{\"custom\": true}");
    ASSERT_EQ(d.configured, true);
}

/// @brief it should handle default values when parsing Device from JSON.
TEST(DeviceTests, testParseFromJSONDefaults) {
    const json j = json::object();
    xjson::Parser parser(j);
    const auto d = Device::parse(parser);
    ASSERT_EQ(d.key, "");
    ASSERT_EQ(d.name, "");
    ASSERT_EQ(d.rack, 0);
    ASSERT_EQ(d.make, "");
    ASSERT_EQ(d.model, "");
    ASSERT_EQ(d.properties, "");
    ASSERT_EQ(d.configured, false);
}
}
