// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/xtest/xtest.h"

#include "driver/ethercat/igh/master.h"

namespace ethercat::igh {

class MasterTest : public ::testing::Test {
protected:
    std::unique_ptr<Manager> manager;
    std::shared_ptr<master::Master> master_instance;

    void SetUp() override {
        auto [mgr, err] = Manager::open();
        if (err) GTEST_SKIP() << "IgH EtherCAT master not available: " << err.message();
        manager = std::move(mgr);
        auto [m, create_err] = manager->create("igh:0");
        if (create_err)
            GTEST_SKIP() << "Failed to create master: " << create_err.message();
        master_instance = std::move(m);
    }
};

TEST_F(MasterTest, InitializesWithKernelModule) {
    ASSERT_NIL(master_instance->initialize());
}

TEST_F(MasterTest, DetectsSlaves) {
    ASSERT_NIL(master_instance->initialize());

    auto slaves = master_instance->slaves();
    EXPECT_GT(slaves.size(), 0) << "No slaves detected on the EtherCAT network";

    for (const auto &slave: slaves) {
        std::cout << "Slave " << slave.position << ": " << slave.name << " (Vendor: 0x"
                  << std::hex << slave.vendor_id << ", Product: 0x"
                  << slave.product_code << std::dec << ")" << std::endl;
    }
}

TEST_F(MasterTest, ActivatesAndDeactivates) {
    ASSERT_NIL(master_instance->initialize());
    ASSERT_NIL(master_instance->activate());
    master_instance->deactivate();
}

TEST_F(MasterTest, CyclicExchange) {
    ASSERT_NIL(master_instance->initialize());

    auto slaves = master_instance->slaves();
    if (slaves.empty()) GTEST_SKIP() << "No slaves for cyclic test";

    ASSERT_NIL(master_instance->activate());

    for (int i = 0; i < 100; ++i) {
        ASSERT_NIL(master_instance->receive());
        ASSERT_NIL(master_instance->send());
    }

    master_instance->deactivate();
}

TEST_F(MasterTest, InterfaceNameReturnsIgHFormat) {
    EXPECT_EQ(master_instance->interface_name(), "igh:0");

    auto [m2, err] = manager->create("igh:1");
    ASSERT_NIL(err);
    EXPECT_EQ(m2->interface_name(), "igh:1");
}

TEST(ManagerOpenTest, ChecksDeviceAndLibraryAvailability) {
    auto [mgr, err] = Manager::open();
    std::cout << "IgH EtherCAT master available: " << (!err ? "yes" : "no")
              << std::endl;
    if (err) std::cout << "Error: " << err.message() << std::endl;
}

}
