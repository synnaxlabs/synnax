// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <memory>
#include <thread>

#include "gtest/gtest.h"

#include "driver/ethercat/soem/master.h"
#include "driver/ethercat/virtual/config.h"
#include "driver/ethercat/virtual/esc.h"
#include "driver/ethercat/virtual/veth.h"
#include "x/cpp/xtest/xtest.h"

namespace ethercat {

class SOEMVirtualTest : public ::testing::Test {
protected:
    std::unique_ptr<virtual_esc::VethPair> veth;
    std::unique_ptr<virtual_esc::VirtualESC> slave;
    std::unique_ptr<soem::Master> master;

    void SetUp() override {
        if (!virtual_esc::can_create_veth("ecsoem")) GTEST_SKIP() << "requires root privileges or pre-existing veth pair";
        this->veth = std::make_unique<virtual_esc::VethPair>("ecsoem");
        auto err = this->veth->create();
        if (err) GTEST_SKIP() << "failed to create veth pair: " << err.message();
        this->slave = std::make_unique<virtual_esc::VirtualESC>(
            virtual_esc::default_test_config()
        );
        err = this->slave->start(this->veth->slave_interface());
        if (err) GTEST_SKIP() << "failed to start virtual ESC: " << err.message();
        std::this_thread::sleep_for(std::chrono::milliseconds(100));
        this->master = std::make_unique<soem::Master>(this->veth->master_interface());
    }

    void TearDown() override {
        if (this->master) this->master->deactivate();
        if (this->slave) this->slave->stop();
    }
};

TEST_F(SOEMVirtualTest, SlaveDiscovery) {
    auto err = this->master->initialize();
    ASSERT_NIL(err);
    auto slaves = this->master->slaves();
    ASSERT_EQ(slaves.size(), 1);
    EXPECT_EQ(slaves[0].vendor_id, virtual_esc::TEST_VENDOR_ID);
    EXPECT_EQ(slaves[0].product_code, virtual_esc::TEST_PRODUCT_CODE);
    EXPECT_EQ(slaves[0].position, 1);
}

TEST_F(SOEMVirtualTest, StateTransitionToPreOp) {
    auto err = this->master->initialize();
    ASSERT_NIL(err);
    auto state = this->master->slave_state(1);
    EXPECT_EQ(state, slave::State::PRE_OP);
}

TEST_F(SOEMVirtualTest, StateTransitionToOp) {
    auto err = this->master->initialize();
    ASSERT_NIL(err);
    err = this->master->activate();
    ASSERT_NIL(err);
    ASSERT_EVENTUALLY_TRUE(this->master->all_slaves_operational());
    EXPECT_TRUE(this->slave->is_operational());
}

TEST_F(SOEMVirtualTest, PDODiscovery) {
    auto err = this->master->initialize();
    ASSERT_NIL(err);
    auto slaves = this->master->slaves();
    ASSERT_EQ(slaves.size(), 1);
    EXPECT_GT(slaves[0].input_pdos.size(), 0);
    EXPECT_GT(slaves[0].output_pdos.size(), 0);
}

TEST_F(SOEMVirtualTest, CyclicExchange) {
    auto err = this->master->initialize();
    ASSERT_NIL(err);
    err = this->master->activate();
    ASSERT_NIL(err);
    ASSERT_EVENTUALLY_TRUE(this->master->all_slaves_operational());
    this->slave->set_input<uint16_t>(0, 0x1234);
    err = this->master->send();
    ASSERT_NIL(err);
    err = this->master->receive();
    ASSERT_NIL(err);
    auto input = this->master->input_data();
    ASSERT_GE(input.size(), 2);
    uint16_t received = 0;
    std::memcpy(&received, input.data(), 2);
    EXPECT_EQ(received, 0x1234);
}

TEST_F(SOEMVirtualTest, WriteOutputData) {
    auto err = this->master->initialize();
    ASSERT_NIL(err);
    err = this->master->activate();
    ASSERT_NIL(err);
    ASSERT_EVENTUALLY_TRUE(this->master->all_slaves_operational());
    auto output = this->master->output_data();
    ASSERT_GE(output.size(), 2);
    const uint16_t value = 0xABCD;
    std::memcpy(output.data(), &value, 2);
    err = this->master->send();
    ASSERT_NIL(err);
    err = this->master->receive();
    ASSERT_NIL(err);
    std::this_thread::sleep_for(std::chrono::milliseconds(10));
    EXPECT_EQ(this->slave->get_output<uint16_t>(0), 0xABCD);
}

TEST_F(SOEMVirtualTest, WorkingCounter) {
    auto err = this->master->initialize();
    ASSERT_NIL(err);
    err = this->master->activate();
    ASSERT_NIL(err);
    ASSERT_EVENTUALLY_TRUE(this->master->all_slaves_operational());
    for (int i = 0; i < 10; i++) {
        err = this->master->send();
        ASSERT_NIL(err);
        err = this->master->receive();
        ASSERT_NIL(err);
    }
}

TEST_F(SOEMVirtualTest, MultipleDataTypes) {
    auto err = this->master->initialize();
    ASSERT_NIL(err);
    err = this->master->activate();
    ASSERT_NIL(err);
    ASSERT_EVENTUALLY_TRUE(this->master->all_slaves_operational());
    this->slave->set_input<uint16_t>(0, 0x1234);
    this->slave->set_input<int32_t>(2, -12345);
    this->slave->set_input<int16_t>(6, -100);
    this->slave->set_input<uint8_t>(8, 0xFF);
    this->slave->set_input<uint8_t>(9, 0x42);
    err = this->master->send();
    ASSERT_NIL(err);
    err = this->master->receive();
    ASSERT_NIL(err);
    auto input = this->master->input_data();
    ASSERT_GE(input.size(), 10);
    uint16_t status = 0;
    std::memcpy(&status, input.data(), 2);
    EXPECT_EQ(status, 0x1234);
    int32_t position = 0;
    std::memcpy(&position, input.data() + 2, 4);
    EXPECT_EQ(position, -12345);
    int16_t velocity = 0;
    std::memcpy(&velocity, input.data() + 6, 2);
    EXPECT_EQ(velocity, -100);
    EXPECT_EQ(input[8], 0xFF);
    EXPECT_EQ(input[9], 0x42);
}

}
