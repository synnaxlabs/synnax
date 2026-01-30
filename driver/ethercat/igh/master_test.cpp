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
    void SetUp() override {
        if (!igh_available()) GTEST_SKIP() << "IgH EtherCAT master not available";
    }
};

TEST_F(MasterTest, InitializesWithKernelModule) {
    Master master(0);
    ASSERT_NIL(master.initialize());
}

TEST_F(MasterTest, DetectsSlaves) {
    Master master(0);
    ASSERT_NIL(master.initialize());

    auto slaves = master.slaves();
    EXPECT_GT(slaves.size(), 0) << "No slaves detected on the EtherCAT network";

    for (const auto &slave: slaves) {
        std::cout << "Slave " << slave.position << ": " << slave.name << " (Vendor: 0x"
                  << std::hex << slave.vendor_id << ", Product: 0x"
                  << slave.product_code << std::dec << ")" << std::endl;
    }
}

TEST_F(MasterTest, CreatesDomain) {
    Master master(0);
    ASSERT_NIL(master.initialize());

    auto domain = master.create_domain();
    ASSERT_NE(master.active_domain(), nullptr);
}

TEST_F(MasterTest, ActivatesAndDeactivates) {
    Master master(0);
    ASSERT_NIL(master.initialize());

    auto domain = master.create_domain();
    ASSERT_NE(master.active_domain(), nullptr);

    ASSERT_NIL(master.activate());
    EXPECT_NE(master.active_domain()->data(), nullptr);

    master.deactivate();
    EXPECT_EQ(master.active_domain(), nullptr);
}

TEST_F(MasterTest, CyclicExchange) {
    Master master(0);
    ASSERT_NIL(master.initialize());

    auto slaves = master.slaves();
    if (slaves.empty()) GTEST_SKIP() << "No slaves for cyclic test";

    auto domain = master.create_domain();
    ASSERT_NE(master.active_domain(), nullptr);

    ASSERT_NIL(master.activate());

    auto *active = master.active_domain();
    ASSERT_NE(active, nullptr);
    ASSERT_NE(active->data(), nullptr);

    for (int i = 0; i < 100; ++i) {
        ASSERT_NIL(master.receive());
        ASSERT_NIL(master.process(*active));
        ASSERT_NIL(master.queue(*active));
        ASSERT_NIL(master.send());
    }

    master.deactivate();
}

TEST_F(MasterTest, InterfaceNameReturnsIgHFormat) {
    Master master(0);
    EXPECT_EQ(master.interface_name(), "igh:0");

    Master master2(1);
    EXPECT_EQ(master2.interface_name(), "igh:1");
}

TEST(IgHAvailabilityTest, ChecksKernelModulePresence) {
    bool available = igh_available();
    std::cout << "IgH EtherCAT master available: " << (available ? "yes" : "no")
              << std::endl;
}

}
