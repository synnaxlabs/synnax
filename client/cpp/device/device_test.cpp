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

std::mt19937 gen_rand_device = random_generator("Device Tests");

namespace synnax::device {
/// @brief it should correctly create a device.
TEST(DeviceTests, testCreateDevice) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    auto d = Device{
        .key = "asdfjahsdfkasjdfhaks",
        .rack = r.key,
        .location = "test_location",
        .make = "test_make",
        .model = "test_model",
        .name = "test_device",
        .properties = "test_properties",
    };
    ASSERT_NIL(client.devices.create(d));
    ASSERT_EQ(d.name, "test_device");
}

/// @brief it should correctly retrieve a device.
TEST(DeviceTests, testRetrieveDevice) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    auto d = Device{
        .key = "asdfjahsdfkasjdfhaks",
        .rack = r.key,
        .location = "test_location",
        .make = "test_make",
        .model = "test_model",
        .name = "test_device",
        .properties = "test_properties",
    };
    ASSERT_NIL(client.devices.create(d));
    const auto d2 = ASSERT_NIL_P(client.devices.retrieve(d.key));
    ASSERT_EQ(d2.name, "test_device");
    ASSERT_EQ(d2.key, d.key);
}

/// @brief it should correctly retrieve multiple devices.
TEST(DeviceTests, testRetrieveDevices) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));

    auto d1 = Device{
        .key = "device1_key",
        .rack = r.key,
        .location = "location_1",
        .make = "make_1",
        .model = "model_1",
        .name = "test_device_1",
        .properties = "properties_1",
    };
    ASSERT_NIL(client.devices.create(d1));

    auto d2 = Device{
        .key = "device2_key",
        .rack = r.key,
        .location = "location_2",
        .make = "make_2",
        .model = "model_2",
        .name = "test_device_2",
        .properties = "properties_2",
    };
    ASSERT_NIL(client.devices.create(d2));

    const std::vector keys = {d1.key, d2.key};
    const auto devices = ASSERT_NIL_P(client.devices.retrieve(keys));

    ASSERT_EQ(devices.size(), 2);

    const auto it1 = std::find_if(
        devices.begin(),
        devices.end(),
        [&d1](const Device &d) { return d.key == d1.key; }
    );
    ASSERT_NE(it1, devices.end());
    ASSERT_EQ(it1->name, "test_device_1");
    ASSERT_EQ(it1->location, "location_1");

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
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));

    // Create a vector of devices to add
    std::vector devices = {
        Device{
            .key = "device1_key",
            .rack = r.key,
            .location = "location_1",
            .make = "make_1",
            .model = "model_1",
            .name = "test_device_1",
            .properties = {{"properties_1", "value"}},
        },
        Device{
            .key = "device2_key",
            .rack = r.key,
            .location = "location_2",
            .make = "make_2",
            .model = "model_2",
            .name = "test_device_2",
            .properties = {{"properties_2", "value"}},
        },
        Device{
            .key = "device3_key",
            .rack = r.key,
            .location = "location_3",
            .make = "make_3",
            .model = "model_3",
            .name = "test_device_3",
            .properties = {{"properties_3", "value3"}},
        },
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
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));

    auto d1 = Device{
        .key = "device1_key",
        .rack = r.key,
        .location = "location_1",
        .make = "make_1",
        .model = "model_1",
        .name = "test_device_1",
        .configured = false,
        .properties = "properties_1",
    };
    ASSERT_NIL(client.devices.create(d1));

    auto d2 = Device{
        .key = "device2_key",
        .rack = r.key,
        .location = "location_2",
        .make = "make_2",
        .model = "model_2",
        .name = "test_device_2",
        .configured = true,
        .properties = "properties_2",
    };
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
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));

    // Create two devices
    auto d1 = Device{
        .key = "device1_key",
        .rack = r.key,
        .location = "location_1",
        .make = "make_1",
        .model = "model_1",
        .name = "test_device_1",
        .properties = "properties_1",
    };
    ASSERT_NIL(client.devices.create(d1));

    auto d2 = Device{
        .key = "device2_key",
        .rack = r.key,
        .location = "location_2",
        .make = "make_2",
        .model = "model_2",
        .name = "test_device_2",
        .properties = "properties_2",
    };
    ASSERT_NIL(client.devices.create(d2));

    // Delete the first device
    ASSERT_NIL(client.devices.del(d1.key));

    // Verify deleted device returns NOT_FOUND
    ASSERT_OCCURRED_AS_P(client.devices.retrieve(d1.key), x::errors::NOT_FOUND);

    // Verify remaining device can still be retrieved
    const auto retrieved = ASSERT_NIL_P(client.devices.retrieve(d2.key));
    ASSERT_EQ(retrieved.key, d2.key);
    ASSERT_EQ(retrieved.name, "test_device_2");
}

/// @brief it should correctly delete a device.
TEST(DeviceTests, testDeleteDevice) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));

    auto d = Device{
        .key = "device_key",
        .rack = r.key,
        .location = "test_location",
        .make = "test_make",
        .model = "test_model",
        .name = "test_device",
        .properties = "test_properties",
    };
    ASSERT_NIL(client.devices.create(d));
    ASSERT_NIL(client.devices.del(d.key));

    ASSERT_OCCURRED_AS_P(client.devices.retrieve(d.key), x::errors::NOT_FOUND);
}

/// @brief it should correctly delete multiple devices.
TEST(DeviceTests, testDeleteDevices) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));

    auto d1 = Device{
        .key = "device1_key",
        .rack = r.key,
        .location = "location_1",
        .make = "make_1",
        .model = "model_1",
        .name = "test_device_1",
        .properties = "properties_1",
    };
    ASSERT_NIL(client.devices.create(d1));

    auto d2 = Device{
        .key = "device2_key",
        .rack = r.key,
        .location = "location_2",
        .make = "make_2",
        .model = "model_2",
        .name = "test_device_2",
        .properties = "properties_2",
    };
    ASSERT_NIL(client.devices.create(d2));

    const std::vector keys = {d1.key, d2.key};
    ASSERT_NIL(client.devices.del(keys));

    ASSERT_OCCURRED_AS_P(client.devices.retrieve(keys), x::errors::NOT_FOUND);
}

/// @brief it should retrieve devices using a device::RetrieveRequest with keys and
/// names.
TEST(DeviceTests, testRetrieveWithRequest) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    const auto rand = std::to_string(gen_rand_device());
    auto d1 = Device{
        .key = "req_d1_" + rand,
        .rack = r.key,
        .location = "loc_a",
        .make = "make_a",
        .model = "model_a",
        .name = "req_dev_1_" + rand,
        .properties = "p1",
    };
    auto d2 = Device{
        .key = "req_d2_" + rand,
        .rack = r.key,
        .location = "loc_b",
        .make = "make_b",
        .model = "model_b",
        .name = "req_dev_2_" + rand,
        .properties = "p2",
    };
    auto d3 = Device{
        .key = "req_d3_" + rand,
        .rack = r.key,
        .location = "loc_c",
        .make = "make_c",
        .model = "model_c",
        .name = "req_dev_3_" + rand,
        .properties = "p3",
    };
    ASSERT_NIL(client.devices.create(d1));
    ASSERT_NIL(client.devices.create(d2));
    ASSERT_NIL(client.devices.create(d3));
    device::RetrieveRequest req_keys{};
    req_keys.keys = {d1.key, d3.key};
    const auto devices_keys = ASSERT_NIL_P(client.devices.retrieve(req_keys));
    ASSERT_EQ(devices_keys.size(), 2);
    auto dm = map_device_keys(devices_keys);
    ASSERT_TRUE(dm.find(d1.key) != dm.end());
    ASSERT_TRUE(dm.find(d3.key) != dm.end());
    device::RetrieveRequest req_names{};
    req_names.names = {d1.name, d2.name};
    const auto devices_names = ASSERT_NIL_P(client.devices.retrieve(req_names));
    ASSERT_EQ(devices_names.size(), 2);
}
/// @brief it should retrieve devices with limit and offset pagination.
TEST(DeviceTests, testRetrieveWithLimitOffset) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    const auto rand = std::to_string(gen_rand_device());
    const auto make = "limit_make_" + rand;
    std::vector<Device> devices;
    for (int i = 0; i < 5; ++i) {
        auto d = Device{
            .key = "limit_d_" + rand + "_" + std::to_string(i),
            .rack = r.key,
            .location = "loc",
            .make = make,
            .model = "model",
            .name = "limit_dev_" + rand + "_" + std::to_string(i),
            .properties = "props",
        };
        ASSERT_NIL(client.devices.create(d));
        devices.push_back(d);
    }
    device::RetrieveRequest req_limit;
    req_limit.makes = {make};
    req_limit.limit = 2;
    const auto devices_limited = ASSERT_NIL_P(client.devices.retrieve(req_limit));
    ASSERT_EQ(devices_limited.size(), 2);
    device::RetrieveRequest req_offset;
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
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    const auto rand = std::to_string(gen_rand_device());
    auto d = Device{
        .key = "status_dev_" + rand,
        .rack = r.key,
        .location = "location",
        .make = "make",
        .model = "model",
        .name = "device_with_status",
        .properties = "properties",
    };
    d.status = device::Status{
        .variant = x::status::VARIANT_SUCCESS,
        .message = "Device is connected",
        .time = x::telem::TimeStamp::now(),
        .details = device::StatusDetails{.rack = r.key, .device = d.key}
    };
    ASSERT_NIL(client.devices.create(d));
    const auto d2 = ASSERT_NIL_P(
        client.devices.retrieve(d.key, {.include_status = true})
    );
    ASSERT_EQ(d2.name, "device_with_status");
    ASSERT_TRUE(d.status.has_value());
    ASSERT_EQ(d2.status->variant, x::status::VARIANT_SUCCESS);
    ASSERT_EQ(d2.status->message, "Device is connected");
    ASSERT_EQ(d2.status->details.rack, r.key);
}

/// @brief it should correctly retrieve multiple devices with statuses.
TEST(DeviceTests, testRetrieveDevicesWithStatus) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    const auto rand = std::to_string(gen_rand_device());
    auto d1 = Device{
        .key = "status_d1_" + rand,
        .rack = r.key,
        .location = "loc1",
        .make = "make1",
        .model = "model1",
        .name = "device_1_status",
        .properties = "props1",
        .status = device::Status{
            .variant = x::status::VARIANT_SUCCESS,
            .message = "Device 1 OK",
            .time = x::telem::TimeStamp::now(),
        }
    };
    auto d2 = Device{
        .key = "status_d2_" + rand,
        .rack = r.key,
        .location = "loc2",
        .make = "make2",
        .model = "model2",
        .name = "device_2_status",
        .properties = "props2",
        .status = device::Status{
            .variant = x::status::VARIANT_WARNING,
            .message = "Device 2 Warning",
            .time = x::telem::TimeStamp::now(),
        }
    };
    ASSERT_NIL(client.devices.create(d1));
    ASSERT_NIL(client.devices.create(d2));
    const std::vector<std::string> keys = {d1.key, d2.key};
    const auto devices = ASSERT_NIL_P(
        client.devices.retrieve(keys, {.include_status = true})
    );
    ASSERT_EQ(devices.size(), 2);
    auto dm = map_device_keys(devices);
    ASSERT_EQ(dm[d1.key].status->variant, x::status::VARIANT_SUCCESS);
    ASSERT_EQ(dm[d1.key].status->message, "Device 1 OK");
    ASSERT_EQ(dm[d2.key].status->variant, x::status::VARIANT_WARNING);
    ASSERT_EQ(dm[d2.key].status->message, "Device 2 Warning");
}

/// @brief it should correctly parse device::StatusDetails from JSON.
TEST(DeviceStatusDetailsTests, testParseFromJSON) {
    json j = {{"rack", 12345}, {"device", "device-abc-123"}};
    const x::json::Parser parser(j);
    const auto details = device::StatusDetails::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(details.rack, 12345);
    ASSERT_EQ(details.device, "device-abc-123");
}

/// @brief it should correctly serialize device::StatusDetails to JSON.
TEST(DeviceStatusDetailsTests, testToJSON) {
    const device::StatusDetails details{
        .rack = 67890,
        .device = "device-xyz-456",
    };
    const auto j = details.to_json();
    ASSERT_EQ(j["rack"], 67890);
    ASSERT_EQ(j["device"], "device-xyz-456");
}

/// @brief it should round-trip device::StatusDetails through JSON.
TEST(DeviceStatusDetailsTests, testRoundTrip) {
    device::StatusDetails original{
        .rack = 11111,
        .device = "round-trip-device",
    };
    const auto j = original.to_json();
    x::json::Parser parser(j);
    auto recovered = device::StatusDetails::parse(parser);
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
    x::json::Parser parser(j);
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
    x::json::Parser parser(j);
    const auto d = Device::parse(parser);
    ASSERT_EQ(d.key, "");
    ASSERT_EQ(d.name, "");
    ASSERT_EQ(d.rack, 0);
    ASSERT_EQ(d.make, "");
    ASSERT_EQ(d.model, "");
    ASSERT_EQ(d.properties, nullptr);
    ASSERT_EQ(d.configured, false);
}
}
