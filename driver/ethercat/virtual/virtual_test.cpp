// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <thread>

#include "gtest/gtest.h"

#include "x/cpp/xtest/xtest.h"

#include "driver/ethercat/virtual/config.h"
#include "driver/ethercat/virtual/datagram.h"
#include "driver/ethercat/virtual/esc.h"
#include "driver/ethercat/virtual/frame.h"
#include "driver/ethercat/virtual/object_dictionary.h"
#include "driver/ethercat/virtual/state_machine.h"
#include "driver/ethercat/virtual/veth.h"

namespace ethercat::virtual_esc {

TEST(FrameTest, ParseValidEtherCATFrame) {
    std::vector<uint8_t> raw_frame = {
        0x01,
        0x02,
        0x03,
        0x04,
        0x05,
        0x06,
        0x07,
        0x08,
        0x09,
        0x0A,
        0x0B,
        0x0C,
        0x88,
        0xA4,
        0x10,
        0x10,
    };
    Frame frame;
    EXPECT_TRUE(frame.parse(raw_frame));
    EXPECT_TRUE(frame.eth_header.is_ethercat());
    EXPECT_TRUE(frame.ec_header.is_command_frame());
}

TEST(FrameTest, RejectNonEtherCATFrame) {
    std::vector<uint8_t> raw_frame = {
        0x01,
        0x02,
        0x03,
        0x04,
        0x05,
        0x06,
        0x07,
        0x08,
        0x09,
        0x0A,
        0x0B,
        0x0C,
        0x08,
        0x00,
        0x00,
        0x00,
    };
    Frame frame;
    EXPECT_FALSE(frame.parse(raw_frame));
}

TEST(StateMachineTest, InitialState) {
    StateMachine sm;
    EXPECT_EQ(sm.current_state(), slave::State::INIT);
    EXPECT_FALSE(sm.is_operational());
}

TEST(StateMachineTest, ValidStateTransitions) {
    StateMachine sm;
    EXPECT_TRUE(sm.request_state(static_cast<uint16_t>(slave::State::PRE_OP)));
    EXPECT_EQ(sm.current_state(), slave::State::PRE_OP);
    EXPECT_TRUE(sm.request_state(static_cast<uint16_t>(slave::State::SAFE_OP)));
    EXPECT_EQ(sm.current_state(), slave::State::SAFE_OP);
    EXPECT_TRUE(sm.request_state(static_cast<uint16_t>(slave::State::OP)));
    EXPECT_EQ(sm.current_state(), slave::State::OP);
    EXPECT_TRUE(sm.is_operational());
}

TEST(StateMachineTest, InvalidStateTransition) {
    StateMachine sm;
    EXPECT_FALSE(sm.request_state(static_cast<uint16_t>(slave::State::OP)));
    EXPECT_EQ(sm.current_state(), slave::State::INIT);
    EXPECT_EQ(sm.al_status_code(), ALStatusCode::INVALID_STATE_CHANGE);
}

TEST(ObjectDictionaryTest, ReadIdentity) {
    ObjectDictionary od;
    od.set_identity(0xDEAD, 0xBEEF, 0x0001, 0x12345678);
    auto vendor = od.read(OD_IDENTITY, 1);
    ASSERT_TRUE(vendor.has_value());
    ASSERT_EQ(vendor->size(), 4);
    uint32_t vendor_id = 0;
    std::memcpy(&vendor_id, vendor->data(), 4);
    EXPECT_EQ(vendor_id, 0xDEAD);
    auto product = od.read(OD_IDENTITY, 2);
    ASSERT_TRUE(product.has_value());
    uint32_t product_code = 0;
    std::memcpy(&product_code, product->data(), 4);
    EXPECT_EQ(product_code, 0xBEEF);
}

TEST(ObjectDictionaryTest, ReadPDOMapping) {
    ObjectDictionary od;
    PDOConfig tx_pdo;
    tx_pdo.index = 0x1A00;
    tx_pdo.entries = {
        {0x6000, 0x01, 16},
        {0x6000, 0x02, 32},
    };
    od.add_tx_pdo(tx_pdo);
    auto count = od.read(OD_TXPDO_ASSIGN, 0);
    ASSERT_TRUE(count.has_value());
    EXPECT_EQ((*count)[0], 1);
    auto pdo_idx = od.read(OD_TXPDO_ASSIGN, 1);
    ASSERT_TRUE(pdo_idx.has_value());
    uint16_t idx = 0;
    std::memcpy(&idx, pdo_idx->data(), 2);
    EXPECT_EQ(idx, 0x1A00);
    auto mapping_count = od.read(OD_TXPDO_MAPPING, 0);
    ASSERT_TRUE(mapping_count.has_value());
    EXPECT_EQ((*mapping_count)[0], 2);
    auto mapping = od.read(OD_TXPDO_MAPPING, 1);
    ASSERT_TRUE(mapping.has_value());
    uint32_t packed = 0;
    std::memcpy(&packed, mapping->data(), 4);
    EXPECT_EQ(packed, 0x60000110);
}

TEST(ConfigTest, DefaultTestConfig) {
    auto cfg = default_test_config();
    EXPECT_EQ(cfg.vendor_id, TEST_VENDOR_ID);
    EXPECT_EQ(cfg.product_code, TEST_PRODUCT_CODE);
    EXPECT_EQ(cfg.tx_pdos.size(), 1);
    EXPECT_EQ(cfg.rx_pdos.size(), 1);
    EXPECT_EQ(cfg.tx_pdo_bytes(), 10);
    EXPECT_EQ(cfg.rx_pdo_bytes(), 10);
}

TEST(VethTest, CanCreateVethReturnsCorrectly) {
    bool can_create = can_create_veth();
    EXPECT_EQ(can_create, geteuid() == 0);
}

class VirtualESCTest : public ::testing::Test {
protected:
    void SetUp() override {
        if (!can_create_veth()) GTEST_SKIP() << "requires root privileges";
    }

    void TearDown() override {}
};

TEST_F(VirtualESCTest, CreateAndStartESC) {
    VethPair veth("vesc_test");
    auto err = veth.create();
    ASSERT_NIL(err);
    VirtualESC esc(minimal_test_config());
    err = esc.start(veth.slave_interface());
    ASSERT_NIL(err);
    EXPECT_TRUE(esc.is_running());
    EXPECT_EQ(esc.current_state(), slave::State::INIT);
    esc.stop();
    EXPECT_FALSE(esc.is_running());
}

TEST_F(VirtualESCTest, InputOutputDataAccess) {
    auto cfg = minimal_test_config();
    VirtualESC esc(cfg);
    esc.set_input<uint8_t>(0, 0x42);
    EXPECT_EQ(esc.get_input<uint8_t>(0), 0x42);
    auto input_data = esc.get_input_data();
    EXPECT_EQ(input_data[0], 0x42);
    std::vector<uint8_t> new_input = {0xFF};
    esc.set_input_data(new_input);
    EXPECT_EQ(esc.get_input<uint8_t>(0), 0xFF);
}

}
