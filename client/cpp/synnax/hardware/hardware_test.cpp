// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// GTest
#include <include/gtest/gtest.h>

// Internal
#include "client/cpp/synnax/synnax.h"
#include "client/cpp/synnax/testutil/testutil.h"

/// @brief it should correctly create a rack in the cluster.
TEST(HardwareTests, testCreateRack) {
    auto client = new_test_client();
    auto r = synnax::Rack("test_rack");
    auto err = client.devices.createRack(r);
    ASSERT_FALSE(err) << err.message();
    ASSERT_EQ(r.name, "test_rack");
}

/// @brief it should correctly retrieve a rack from the cluster.
TEST(HardwareTests, testRetrieveRack) {
    auto client = new_test_client();
    auto r = synnax::Rack("test_rack");
    auto err = client.devices.createRack(r);
    ASSERT_FALSE(err) << err.message();
    auto [r2, r2err] = client.devices.retrieveRack(r.key.value);
    ASSERT_FALSE(r2err) << err.message();
    ASSERT_EQ(r2.name, "test_rack");
    ASSERT_EQ(r.key.value, r2.key.value);
}

/// @brief it should correctly delete a rack from the cluster.
TEST(HardwareTest, testDeleteRack) {
    auto client = new_test_client();
    auto r = synnax::Rack("test_rack");
    auto err = client.devices.createRack(r);
    ASSERT_FALSE(err) << err.message();
    auto err2 = client.devices.deleteRack(r.key.value);
    ASSERT_FALSE(err2) << err2.message();
    auto [r2, r2err] = client.devices.retrieveRack(r.key.value);
    ASSERT_EQ(r2err.type, synnax::QUERY_ERROR);
}

/// @brief it should correctly create a module on the rack.
TEST(HardwareTests, testCreateTask) {
    auto client = new_test_client();
    auto r = synnax::Rack("test_rack");
    auto err = client.devices.createRack(r);
    ASSERT_FALSE(err) << err.message();
    auto m = synnax::Task(r.key, "test_module", "mock", "config");
    auto err2 = r.tasks.create(m);
    ASSERT_FALSE(err2) << err2.message();
    ASSERT_EQ(m.name, "test_module");
    ASSERT_EQ(m.key.rack_key().value, r.key.value);
    ASSERT_NE(m.key.local_key(), 0);
}

/// @brief it should correctly retrieve a module from the rack.
TEST(HardwareTests, testRetrieveTask) {
    auto client = new_test_client();
    auto r = synnax::Rack("test_rack");
    auto err = client.devices.createRack(r);
    ASSERT_FALSE(err) << err.message();
    auto m = synnax::Task(r.key, "test_module", "mock", "config");
    auto err2 = r.tasks.create(m);
    ASSERT_FALSE(err2) << err2.message();
    auto [m2, m2err] = r.tasks.retrieve(m.key.local_key());
    ASSERT_FALSE(m2err) << m2err.message();
    ASSERT_EQ(m2.name, "test_module");
    ASSERT_EQ(m2.key.rack_key().value, r.key.value);
    ASSERT_EQ(m2.key.local_key(), m.key.local_key());
}

/// @brief it should correctly list the tasks on a rack.
TEST(HardwareTests, testListTasks) {
    auto client = new_test_client();
    auto r = synnax::Rack("test_rack");
    auto err = client.devices.createRack(r);
    ASSERT_FALSE(err) << err.message();
    auto m = synnax::Task(r.key, "test_module", "mock", "config");
    auto err2 = r.tasks.create(m);
    ASSERT_FALSE(err2) << err2.message();
    auto [tasks, err3] = r.tasks.list();
    ASSERT_FALSE(err3) << err3.message();
    ASSERT_EQ(tasks.size(), 1);
    ASSERT_EQ(tasks[0].name, "test_module");
    ASSERT_EQ(tasks[0].key.rack_key().value, r.key.value);
    ASSERT_NE(tasks[0].key.local_key(), 0);
}