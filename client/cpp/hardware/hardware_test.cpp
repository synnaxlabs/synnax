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

std::mt19937 gen_rand = random_generator("Hardware Tests");

/// @brief it should correctly create a rack in the cluster.
TEST(HardwareTests, testCreateRack) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    const auto err = client.hardware.create_rack(r);
    ASSERT_FALSE(err) << err.message();
    ASSERT_EQ(r.name, "test_rack");
}

/// @brief it should correctly retrieve a rack from the cluster.
TEST(HardwareTests, testRetrieveRack) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    const auto err = client.hardware.create_rack(r);
    ASSERT_FALSE(err) << err.message();
    const auto [r2, r2err] = client.hardware.retrieve_rack(r.key);
    ASSERT_FALSE(r2err) << err.message();
    ASSERT_EQ(r2.name, "test_rack");
    ASSERT_EQ(r.key, r2.key);
}

/// @brief it should correctly delete a rack from the cluster.
TEST(HardwareTest, testDeleteRack) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    const auto err = client.hardware.create_rack(r);
    ASSERT_FALSE(err) << err.message();
    const auto err2 = client.hardware.delete_rack(r.key);
    ASSERT_FALSE(err2) << err2.message();
    const auto [r2, r2err] = client.hardware.retrieve_rack(r.key);
    ASSERT_TRUE(r2err.matches(xerrors::QUERY_ERROR)) << r2err;
}

/// @brief it should correctly create a module on the rack.
TEST(HardwareTests, testCreateTask) {
    auto client = new_test_client();
    auto r = Rack("test_rack");
    auto err = client.hardware.create_rack(r);
    ASSERT_FALSE(err) << err.message();
    auto m = Task(r.key, "test_module", "mock", "config");
    auto err2 = r.tasks.create(m);
    ASSERT_FALSE(err2) << err2.message();
    ASSERT_EQ(m.name, "test_module");
    ASSERT_EQ(synnax::task_key_rack(m.key), r.key);
    ASSERT_NE(synnax::task_key_local(m.key), 0);
}

/// @brief it should correctly retrieve a module from the rack.
TEST(HardwareTests, testRetrieveTask) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    const auto err = client.hardware.create_rack(r);
    ASSERT_FALSE(err) << err.message();
    auto t = Task(r.key, "test_module", "mock", "config");
    const auto err2 = r.tasks.create(t);
    ASSERT_FALSE(err2) << err2.message();
    const auto [t2, m2err] = r.tasks.retrieve(t.key);
    ASSERT_FALSE(m2err) << m2err.message();
    ASSERT_EQ(t2.name, "test_module");
    ASSERT_EQ(synnax::task_key_rack(t.key), r.key);
    ASSERT_EQ(synnax::task_key_local(t2.key), synnax::task_key_local(t.key));
}

/// @brief it should retrieve a task by its name
TEST(HardwareTests, testRetrieveTaskByName) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    const auto err = client.hardware.create_rack(r);
    ASSERT_FALSE(err) << err.message();
    const auto rand_name = std::to_string(gen_rand());
    auto t = Task(r.key, rand_name, "mock", "config");
    const auto err2 = r.tasks.create(t);
    ASSERT_FALSE(err2) << err2.message();
    const auto [t2, m2err] = r.tasks.retrieve(rand_name);
    ASSERT_FALSE(m2err) << m2err.message();
    ASSERT_EQ(t2.name, rand_name);
    ASSERT_EQ(synnax::task_key_rack(t.key), r.key);
}

/// @brief it should retrieve a task by its type
TEST(HardwareTests, testRetrieveTaskByType) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    const auto err = client.hardware.create_rack(r);
    ASSERT_FALSE(err) << err.message();
    const auto rand_type = std::to_string(gen_rand());
    auto t = Task(r.key, "test_module", rand_type, "config");
    const auto err2 = r.tasks.create(t);
    ASSERT_FALSE(err2) << err2.message();
    const auto [t2, m2err] = r.tasks.retrieveByType(rand_type);
    ASSERT_FALSE(m2err) << m2err.message();
    ASSERT_EQ(t2.name, "test_module");
    ASSERT_EQ(synnax::task_key_rack(t.key), r.key);
}

/// @brief it should correctly list the tasks on a rack.
TEST(HardwareTests, testListTasks) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    const auto err = client.hardware.create_rack(r);
    ASSERT_FALSE(err) << err.message();
    auto m = Task(r.key, "test_module", "mock", "config");
    const auto err2 = r.tasks.create(m);
    ASSERT_FALSE(err2) << err2.message();
    const auto [tasks, err3] = r.tasks.list();
    ASSERT_FALSE(err3) << err3.message();
    ASSERT_EQ(tasks.size(), 1);
    ASSERT_EQ(tasks[0].name, "test_module");
    ASSERT_EQ(synnax::task_key_rack(tasks[0].key), r.key);
    ASSERT_NE(synnax::task_key_local(tasks[0].key), 0);
}

/// @brief it should correctly create a device.
TEST(HardwareTests, testCreateDevice) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    const auto err = client.hardware.create_rack(r);
    ASSERT_FALSE(err) << err.message();
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
    const auto err2 = client.hardware.create_device(d);
    ASSERT_FALSE(err2) << err2.message();
    ASSERT_EQ(d.name, "test_device");
}

/// @brief it should correctly retrieve a device.
TEST(HardwareTests, testRetrieveDevice) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    const auto err = client.hardware.create_rack(r);
    ASSERT_FALSE(err) << err.message();
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
    const auto err2 = client.hardware.create_device(d);
    ASSERT_FALSE(err2) << err2.message();
    const auto [d2, d2err] = client.hardware.retrieve_device(d.key);
    ASSERT_FALSE(d2err) << d2err.message();
    ASSERT_EQ(d2.name, "test_device");
    ASSERT_EQ(d2.key, d.key);
}
