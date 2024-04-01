// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// GTest

#include "gtest/gtest.h"
#include "client/cpp/synnax/synnax.h"
#include "client/cpp/synnax/testutil/testutil.h"

/// @brief it should correctly create a rack in the cluster.
TEST(HardwareTests, testCreateRack) {
    auto client = new_test_client();
    auto r = Rack("test_rack");
    auto err = client.hardware.createRack(r);
    ASSERT_FALSE(err) << err.message();
    ASSERT_EQ(r.name, "test_rack");
}

/// @brief it should correctly retrieve a rack from the cluster.
TEST(HardwareTests, testRetrieveRack) {
    auto client = new_test_client();
    auto r = Rack("test_rack");
    auto err = client.hardware.createRack(r);
    ASSERT_FALSE(err) << err.message();
    auto [r2, r2err] = client.hardware.retrieveRack(r.key);
    ASSERT_FALSE(r2err) << err.message();
    ASSERT_EQ(r2.name, "test_rack");
    ASSERT_EQ(r.key, r2.key);
}

/// @brief it should correctly delete a rack from the cluster.
TEST(HardwareTest, testDeleteRack) {
    auto client = new_test_client();
    auto r = Rack("test_rack");
    auto err = client.hardware.createRack(r);
    ASSERT_FALSE(err) << err.message();
    auto err2 = client.hardware.deleteRack(r.key);
    ASSERT_FALSE(err2) << err2.message();
    auto [r2, r2err] = client.hardware.retrieveRack(r.key);
    ASSERT_TRUE(r2err.matches(synnax::QUERY_ERROR)) << r2err;
}

/// @brief it should correctly create a module on the rack.
TEST(HardwareTests, testCreateTask) {
    auto client = new_test_client();
    auto r = Rack("test_rack");
    auto err = client.hardware.createRack(r);
    ASSERT_FALSE(err) << err.message();
    auto m = Task(r.key, "test_module", "mock", "config");
    auto err2 = r.tasks.create(m);
    ASSERT_FALSE(err2) << err2.message();
    ASSERT_EQ(m.name, "test_module");
    ASSERT_EQ(synnax::taskKeyRack(m.key), r.key);
    ASSERT_NE(synnax::taskKeyLocal(m.key), 0);
}

/// @brief it should correctly retrieve a module from the rack.
TEST(HardwareTests, testRetrieveTask) {
    auto client = new_test_client();
    auto r = Rack("test_rack");
    auto err = client.hardware.createRack(r);
    ASSERT_FALSE(err) << err.message();
    auto t = Task(r.key, "test_module", "mock", "config");
    auto err2 = r.tasks.create(t);
    ASSERT_FALSE(err2) << err2.message();
    auto [t2, m2err] = r.tasks.retrieve(t.key);
    ASSERT_FALSE(m2err) << m2err.message();
    ASSERT_EQ(t2.name, "test_module");
    ASSERT_EQ(synnax::taskKeyRack(t.key), r.key);
    ASSERT_EQ(synnax::taskKeyLocal(t2.key), synnax::taskKeyLocal(t.key));
}

/// @brief it should correctly list the tasks on a rack.
TEST(HardwareTests, testListTasks) {
    auto client = new_test_client();
    auto r = Rack("test_rack");
    auto err = client.hardware.createRack(r);
    ASSERT_FALSE(err) << err.message();
    auto m = Task(r.key, "test_module", "mock", "config");
    auto err2 = r.tasks.create(m);
    ASSERT_FALSE(err2) << err2.message();
    auto [tasks, err3] = r.tasks.list();
    ASSERT_FALSE(err3) << err3.message();
    ASSERT_EQ(tasks.size(), 1);
    ASSERT_EQ(tasks[0].name, "test_module");
    ASSERT_EQ(synnax::taskKeyRack(tasks[0].key), r.key);
    ASSERT_NE(synnax::taskKeyLocal(tasks[0].key), 0);
}