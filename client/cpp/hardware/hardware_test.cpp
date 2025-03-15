// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "gtest/gtest.h"
#include "x/cpp/xtest/xtest.h"

std::mt19937 gen_rand = random_generator("Hardware Tests");

/// @brief it should correctly create a rack in the cluster.
TEST(RackTests, testCreateRack) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.hardware.create_rack(r));
    ASSERT_EQ(r.name, "test_rack");
}

/// @brief it should correctly retrieve a rack from the cluster.
TEST(RackTests, testRetrieveRack) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.hardware.create_rack(r));
    const auto r2 = ASSERT_NIL_P(client.hardware.retrieve_rack(r.key));
    ASSERT_EQ(r2.name, "test_rack");
    ASSERT_EQ(r.key, r2.key);
}

/// @brief it should correctly delete a rack from the cluster.
TEST(RackTests, testDeleteRack) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.hardware.create_rack(r));
    ASSERT_NIL(client.hardware.delete_rack(r.key));
    ASSERT_OCCURRED_AS_P(client.hardware.retrieve_rack(r.key), xerrors::QUERY);
}

/// @brief it should correctly create a module on the rack.
TEST(TaskTests, testCreateTask) {
    auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.hardware.create_rack(r));
    auto m = Task(r.key, "test_module", "mock", "config");
    ASSERT_NIL(r.tasks.create(m));
    ASSERT_EQ(m.name, "test_module");
    ASSERT_EQ(synnax::task_key_rack(m.key), r.key);
    ASSERT_NE(synnax::task_key_local(m.key), 0);
}

/// @brief it should correctly retrieve a module from the rack.
TEST(TaskTests, testRetrieveTask) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.hardware.create_rack(r));
    auto t = Task(r.key, "test_module", "mock", "config");
    ASSERT_NIL(r.tasks.create(t));
    const auto t2 = ASSERT_NIL_P(r.tasks.retrieve(t.key));
    ASSERT_EQ(t2.name, "test_module");
    ASSERT_EQ(synnax::task_key_rack(t.key), r.key);
    ASSERT_EQ(synnax::task_key_local(t2.key), synnax::task_key_local(t.key));
}

/// @brief it should retrieve a task by its name
TEST(TaskTests, testRetrieveTaskByName) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.hardware.create_rack(r));
    const auto rand_name = std::to_string(gen_rand());
    auto t = Task(r.key, rand_name, "mock", "config");
    ASSERT_NIL(r.tasks.create(t));
    const auto t2 = ASSERT_NIL_P(r.tasks.retrieve(rand_name));
    ASSERT_EQ(t2.name, rand_name);
    ASSERT_EQ(synnax::task_key_rack(t.key), r.key);
}

/// @brief it should retrieve a task by its type
TEST(TaskTests, testRetrieveTaskByType) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.hardware.create_rack(r));
    const auto rand_type = std::to_string(gen_rand());
    auto t = Task(r.key, "test_module", rand_type, "config");
    ASSERT_NIL(r.tasks.create(t));
    const auto t2 = ASSERT_NIL_P(r.tasks.retrieve_by_type(rand_type));
    ASSERT_EQ(t2.name, "test_module");
    ASSERT_EQ(synnax::task_key_rack(t.key), r.key);
}

/// @brief it should correctly list the tasks on a rack.
TEST(TaskTests, testListTasks) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.hardware.create_rack(r));
    auto m = Task(r.key, "test_module", "mock", "config");
    ASSERT_NIL(r.tasks.create(m));
    const auto tasks = ASSERT_NIL_P(r.tasks.list());
    ASSERT_EQ(tasks.size(), 1);
    ASSERT_EQ(tasks[0].name, "test_module");
    ASSERT_EQ(synnax::task_key_rack(tasks[0].key), r.key);
    ASSERT_NE(synnax::task_key_local(tasks[0].key), 0);
}

/// @brief it should correctly create a device.
TEST(DeviceTests, testCreateDevice) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.hardware.create_rack(r));
    auto d = Device(
        "asdfjahsdfkasjdfhaks",
        "test_device",
        r.key,
        "test_location",
        "test_identifier",
        "test_make",
        "test_model",
        "test_properties"
    );
    ASSERT_NIL(client.hardware.create_device(d));
    ASSERT_EQ(d.name, "test_device");
}

/// @brief it should correctly retrieve a device.
TEST(DeviceTests, testRetrieveDevice) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.hardware.create_rack(r));
    auto d = Device(
        "asdfjahsdfkasjdfhaks",
        "test_device",
        r.key,
        "test_location",
        "test_identifier",
        "test_make",
        "test_model",
        "test_properties"
    );
    ASSERT_NIL(client.hardware.create_device(d));
    const auto d2 = ASSERT_NIL_P(client.hardware.retrieve_device(d.key));
    ASSERT_EQ(d2.name, "test_device");
    ASSERT_EQ(d2.key, d.key);
}

/// @brief it should correctly retrieve multiple devices.
TEST(DeviceTests, testRetrieveDevices) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.hardware.create_rack(r));

    // Create first device
    auto d1 = Device(
        "device1_key",
        "test_device_1",
        r.key,
        "location_1",
        "identifier_1",
        "make_1",
        "model_1",
        "properties_1"
    );
    ASSERT_NIL(client.hardware.create_device(d1));

    // Create second device
    auto d2 = Device(
        "device2_key",
        "test_device_2",
        r.key,
        "location_2",
        "identifier_2",
        "make_2",
        "model_2",
        "properties_2"
    );
    ASSERT_NIL(client.hardware.create_device(d2));

    // Retrieve both devices
    std::vector keys = {d1.key, d2.key};
    const auto devices = ASSERT_NIL_P(client.hardware.retrieve_devices(keys));

    // Verify we got both devices
    ASSERT_EQ(devices.size(), 2);

    // Find and verify first device
    auto it1 = std::find_if(devices.begin(), devices.end(),
                            [&d1](const Device &d) { return d.key == d1.key; });
    ASSERT_NE(it1, devices.end());
    ASSERT_EQ(it1->name, "test_device_1");
    ASSERT_EQ(it1->location, "location_1");

    // Find and verify second device
    auto it2 = std::find_if(devices.begin(), devices.end(),
                            [&d2](const Device &d) { return d.key == d2.key; });
    ASSERT_NE(it2, devices.end());
    ASSERT_EQ(it2->name, "test_device_2");
    ASSERT_EQ(it2->location, "location_2");
}

/// @brief it should correctly create multiple devices at once.
TEST(DeviceTests, testCreateDevices) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.hardware.create_rack(r));

    // Create a vector of devices to add
    std::vector devices = {
        Device(
            "device1_key",
            "test_device_1",
            r.key,
            "location_1",
            "identifier_1",
            "make_1",
            "model_1",
            "properties_1"
        ),
        Device(
            "device2_key",
            "test_device_2",
            r.key,
            "location_2",
            "identifier_2",
            "make_2",
            "model_2",
            "properties_2"
        ),
        Device(
            "device3_key",
            "test_device_3",
            r.key,
            "location_3",
            "identifier_3",
            "make_3",
            "model_3",
            "properties_3"
        )
    };

    // Create all devices at once
    ASSERT_NIL(client.hardware.create_devices(devices));

    // Verify each device was created correctly by retrieving them
    for (const auto &device: devices) {
        const auto retrieved =
                ASSERT_NIL_P(client.hardware.retrieve_device(device.key));
        ASSERT_EQ(retrieved.key, device.key);
        ASSERT_EQ(retrieved.name, device.name);
        ASSERT_EQ(retrieved.rack, r.key);
        ASSERT_EQ(retrieved.location, device.location);
        ASSERT_EQ(retrieved.identifier, device.identifier);
        ASSERT_EQ(retrieved.make, device.make);
        ASSERT_EQ(retrieved.model, device.model);
        ASSERT_EQ(retrieved.properties, device.properties);
    }

    // Also test retrieving all devices at once
    std::vector<std::string> keys;
    keys.reserve(devices.size());
    for (const auto &device: devices)
        keys.push_back(device.key);

    const auto retrieved_devices = ASSERT_NIL_P(client.hardware.retrieve_devices(keys));
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