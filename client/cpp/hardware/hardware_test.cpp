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
TEST(HardwareTests, testCreateRack) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.hardware.create_rack(r));
    ASSERT_EQ(r.name, "test_rack");
}

/// @brief it should correctly retrieve a rack from the cluster.
TEST(HardwareTests, testRetrieveRack) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.hardware.create_rack(r));
    const auto r2 = ASSERT_NIL_P(client.hardware.retrieve_rack(r.key));
    ASSERT_EQ(r2.name, "test_rack");
    ASSERT_EQ(r.key, r2.key);
}

/// @brief it should correctly delete a rack from the cluster.
TEST(HardwareTest, testDeleteRack) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.hardware.create_rack(r));
    ASSERT_NIL(client.hardware.delete_rack(r.key));
    ASSERT_OCCURRED_AS_P(client.hardware.retrieve_rack(r.key), xerrors::QUERY);
}

/// @brief it should correctly create a module on the rack.
TEST(HardwareTests, testCreateTask) {
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
TEST(HardwareTests, testRetrieveTask) {
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
TEST(HardwareTests, testRetrieveTaskByName) {
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
TEST(HardwareTests, testRetrieveTaskByType) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.hardware.create_rack(r));
    const auto rand_type = std::to_string(gen_rand());
    auto t = Task(r.key, "test_module", rand_type, "config");
    ASSERT_NIL(r.tasks.create(t));
    const auto t2 = ASSERT_NIL_P(r.tasks.retrieveByType(rand_type));
    ASSERT_EQ(t2.name, "test_module");
    ASSERT_EQ(synnax::task_key_rack(t.key), r.key);
}

/// @brief it should correctly list the tasks on a rack.
TEST(HardwareTests, testListTasks) {
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
TEST(HardwareTests, testCreateDevice) {
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
TEST(HardwareTests, testRetrieveDevice) {
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
