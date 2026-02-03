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

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/mock/master.h"

namespace ethercat {

class MasterTest : public ::testing::Test {
protected:
    std::shared_ptr<mock::Master> master;

    void SetUp() override { master = std::make_shared<mock::Master>("mock0"); }
};

TEST_F(MasterTest, InitializeSuccess) {
    master->add_slave(mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));
    ASSERT_NIL(master->initialize());
    EXPECT_TRUE(master->is_initialized());
    EXPECT_TRUE(master->was_called("initialize"));
}

TEST_F(MasterTest, InitializeFailure) {
    master->inject_init_error(xerrors::Error(MASTER_INIT_ERROR, "interface not found"));
    ASSERT_OCCURRED_AS(master->initialize(), MASTER_INIT_ERROR);
    EXPECT_FALSE(master->is_initialized());
}

TEST_F(MasterTest, ActivateSuccess) {
    master->add_slave(mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));
    ASSERT_NIL(master->initialize());
    ASSERT_NIL(master->activate());
    EXPECT_TRUE(master->is_activated());
    EXPECT_TRUE(master->was_called("activate"));
    EXPECT_TRUE(master->all_slaves_operational());
}

TEST_F(MasterTest, ActivateFailure) {
    master->add_slave(mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));
    ASSERT_NIL(master->initialize());
    master->inject_activate_error(xerrors::Error(ACTIVATION_ERROR, "failed to map IO"));
    ASSERT_OCCURRED_AS(master->activate(), ACTIVATION_ERROR);
    EXPECT_FALSE(master->is_activated());
}

TEST_F(MasterTest, ActivateWithoutInitializeFails) {
    ASSERT_OCCURRED_AS(master->activate(), ACTIVATION_ERROR);
}

TEST_F(MasterTest, SlaveDiscovery) {
    master->add_slave(mock::MockSlaveConfig(0, 0x100, 0x200, "Slave1"));
    master->add_slave(mock::MockSlaveConfig(1, 0x100, 0x201, "Slave2"));
    master->add_slave(mock::MockSlaveConfig(2, 0x100, 0x202, "Slave3"));

    ASSERT_NIL(master->initialize());

    auto slaves = master->slaves();
    ASSERT_EQ(slaves.size(), 3);
    EXPECT_EQ(slaves[0].position, 0);
    EXPECT_EQ(slaves[0].vendor_id, 0x100);
    EXPECT_EQ(slaves[0].product_code, 0x200);
    EXPECT_EQ(slaves[0].name, "Slave1");
    EXPECT_EQ(slaves[1].position, 1);
    EXPECT_EQ(slaves[2].position, 2);
}

TEST_F(MasterTest, SlaveStateQueriesBeforeActivation) {
    master->add_slave(mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));
    ASSERT_NIL(master->initialize());

    EXPECT_EQ(master->slave_state(0), slave::State::INIT);
    EXPECT_FALSE(master->all_slaves_operational());
}

TEST_F(MasterTest, SlaveStateQueriesAfterActivation) {
    master->add_slave(mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));
    master->add_slave(mock::MockSlaveConfig(1, 0x1, 0x3, "Slave2"));
    ASSERT_NIL(master->initialize());
    ASSERT_NIL(master->activate());

    EXPECT_EQ(master->slave_state(0), slave::State::OP);
    EXPECT_EQ(master->slave_state(1), slave::State::OP);
    EXPECT_TRUE(master->all_slaves_operational());
}

TEST_F(MasterTest, SlaveStateQueryUnknownPosition) {
    master->add_slave(mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));
    ASSERT_NIL(master->initialize());

    EXPECT_EQ(master->slave_state(99), slave::State::UNKNOWN);
}

TEST_F(MasterTest, PDOOffsetLookup) {
    auto cfg = mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1")
                   .with_input_pdos({
                       pdo::Properties(
                           0x1600,
                           0x6000,
                           1,
                           16,
                           true,
                           "Input1",
                           telem::INT16_T
                       ),
                       pdo::Properties(
                           0x1600,
                           0x6000,
                           2,
                           32,
                           true,
                           "Input2",
                           telem::INT32_T
                       ),
                   })
                   .with_output_pdos({
                       pdo::Properties(
                           0x1A00,
                           0x7000,
                           1,
                           16,
                           false,
                           "Output1",
                           telem::INT16_T
                       ),
                   });
    master->add_slave(cfg);
    ASSERT_NIL(master->initialize());
    ASSERT_NIL(master->activate());

    pdo::Entry input1(0, 0x6000, 1, 16, true);
    pdo::Entry input2(0, 0x6000, 2, 32, true);
    pdo::Entry output1(0, 0x7000, 1, 16, false);

    auto offset1 = this->master->pdo_offset(input1);
    auto offset2 = this->master->pdo_offset(input2);
    auto offset3 = this->master->pdo_offset(output1);

    EXPECT_EQ(offset1.byte, 0);
    EXPECT_EQ(offset2.byte, 2);
    EXPECT_EQ(offset3.byte, 0);
}

TEST_F(MasterTest, BufferAccessAfterActivation) {
    this->master->add_slave(mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));
    ASSERT_NIL(this->master->initialize());
    ASSERT_NIL(this->master->activate());

    EXPECT_FALSE(this->master->input_data().empty());
    EXPECT_FALSE(this->master->output_data().empty());
    EXPECT_GT(this->master->input_data().size(), 0);
    EXPECT_GT(this->master->output_data().size(), 0);
}

TEST_F(MasterTest, BufferAccessBeforeActivation) {
    this->master->add_slave(mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));
    ASSERT_NIL(this->master->initialize());

    EXPECT_TRUE(this->master->input_data().empty());
    EXPECT_TRUE(this->master->output_data().empty());
    EXPECT_EQ(this->master->input_data().size(), 0);
    EXPECT_EQ(this->master->output_data().size(), 0);
}

TEST_F(MasterTest, StateTransitionsOnActivation) {
    master->add_slave(mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));
    ASSERT_NIL(master->initialize());

    EXPECT_EQ(master->slave_state(0), slave::State::INIT);

    ASSERT_NIL(master->activate());

    EXPECT_EQ(master->slave_state(0), slave::State::OP);
}

TEST_F(MasterTest, PartialStateTransition) {
    master->add_slave(mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));
    master->add_slave(mock::MockSlaveConfig(1, 0x1, 0x3, "Slave2"));
    master->add_slave(mock::MockSlaveConfig(2, 0x1, 0x4, "Slave3"));

    master->set_slave_transition_failure(1, slave::State::OP);

    ASSERT_NIL(master->initialize());
    ASSERT_NIL(master->activate());

    EXPECT_EQ(master->slave_state(0), slave::State::OP);
    EXPECT_EQ(master->slave_state(1), slave::State::SAFE_OP);
    EXPECT_EQ(master->slave_state(2), slave::State::OP);
    EXPECT_FALSE(master->all_slaves_operational());
    EXPECT_EQ(master->slaves_in_state(slave::State::OP), 2);
    EXPECT_EQ(master->slaves_in_state(slave::State::SAFE_OP), 1);
}

TEST_F(MasterTest, GracefulDeactivation) {
    master->add_slave(mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));
    ASSERT_NIL(master->initialize());
    ASSERT_NIL(master->activate());

    EXPECT_TRUE(master->is_activated());
    EXPECT_EQ(master->slave_state(0), slave::State::OP);

    master->deactivate();

    EXPECT_FALSE(master->is_activated());
    EXPECT_EQ(master->slave_state(0), slave::State::INIT);
    EXPECT_TRUE(master->was_called("deactivate"));
}

TEST_F(MasterTest, InterfaceNameAccessor) {
    EXPECT_EQ(master->interface_name(), "mock0");
}

TEST_F(MasterTest, ReceiveErrorInjection) {
    master->add_slave(mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));
    ASSERT_NIL(master->initialize());
    ASSERT_NIL(master->activate());

    master->inject_receive_error(xerrors::Error(CYCLIC_ERROR, "receive failed"));
    ASSERT_OCCURRED_AS(master->receive(), CYCLIC_ERROR);
}

TEST_F(MasterTest, SendErrorInjection) {
    master->add_slave(mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));
    ASSERT_NIL(master->initialize());
    ASSERT_NIL(master->activate());

    master->inject_send_error(xerrors::Error(CYCLIC_ERROR, "send failed"));
    ASSERT_OCCURRED_AS(master->send(), CYCLIC_ERROR);
}

TEST_F(MasterTest, ClearInjectedErrors) {
    master->inject_init_error(xerrors::Error(MASTER_INIT_ERROR, "error"));
    master->inject_activate_error(xerrors::Error(ACTIVATION_ERROR, "error"));
    master->inject_receive_error(xerrors::Error(CYCLIC_ERROR, "error"));
    master->inject_send_error(xerrors::Error(CYCLIC_ERROR, "error"));

    master->clear_injected_errors();
    master->add_slave(mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));

    ASSERT_NIL(master->initialize());
    ASSERT_NIL(master->activate());
    ASSERT_NIL(master->receive());
    ASSERT_NIL(master->send());
}

TEST_F(MasterTest, CallLogTracking) {
    master->add_slave(mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));

    ASSERT_NIL(master->initialize());
    ASSERT_NIL(master->activate());
    ASSERT_NIL(master->receive());
    ASSERT_NIL(master->send());
    master->deactivate();

    auto log = master->call_log();
    EXPECT_EQ(log.size(), 5);
    EXPECT_EQ(log[0], "initialize");
    EXPECT_EQ(log[1], "activate");
    EXPECT_EQ(log[2], "receive");
    EXPECT_EQ(log[3], "send");
    EXPECT_EQ(log[4], "deactivate");
}

TEST_F(MasterTest, CallLogClear) {
    master->add_slave(mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));
    ASSERT_NIL(master->initialize());

    EXPECT_TRUE(master->was_called("initialize"));

    master->clear_call_log();

    EXPECT_FALSE(master->was_called("initialize"));
    EXPECT_TRUE(master->call_log().empty());
}

TEST_F(MasterTest, SetSlaveStateDirectly) {
    master->add_slave(mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));
    ASSERT_NIL(master->initialize());

    EXPECT_EQ(master->slave_state(0), slave::State::INIT);

    master->set_slave_state(0, slave::State::PRE_OP);
    EXPECT_EQ(master->slave_state(0), slave::State::PRE_OP);

    master->set_slave_state(0, slave::State::SAFE_OP);
    EXPECT_EQ(master->slave_state(0), slave::State::SAFE_OP);
}

TEST_F(MasterTest, SlaveCountAccessor) {
    EXPECT_EQ(master->slave_count(), 0);

    master->add_slave(mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));
    EXPECT_EQ(master->slave_count(), 1);

    master->add_slave(mock::MockSlaveConfig(1, 0x1, 0x3, "Slave2"));
    EXPECT_EQ(master->slave_count(), 2);
}

TEST_F(MasterTest, HasSlaveInState) {
    master->add_slave(mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));
    master->add_slave(mock::MockSlaveConfig(1, 0x1, 0x3, "Slave2"));
    ASSERT_NIL(master->initialize());

    EXPECT_TRUE(master->has_slave_in_state(slave::State::INIT));
    EXPECT_FALSE(master->has_slave_in_state(slave::State::OP));

    ASSERT_NIL(master->activate());

    EXPECT_FALSE(master->has_slave_in_state(slave::State::INIT));
    EXPECT_TRUE(master->has_slave_in_state(slave::State::OP));
}

TEST_F(MasterTest, InputOutputDataReadWrite) {
    this->master->add_slave(mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));
    ASSERT_NIL(this->master->initialize());
    ASSERT_NIL(this->master->activate());

    uint32_t test_value = 0xDEADBEEF;
    this->master->set_input(0, test_value);

    uint32_t read_value;
    std::memcpy(&read_value, this->master->input_data().data(), sizeof(read_value));
    EXPECT_EQ(read_value, 0xDEADBEEF);

    uint16_t output_value = 0x1234;
    std::memcpy(
        this->master->output_data().data(),
        &output_value,
        sizeof(output_value)
    );
    EXPECT_EQ(this->master->get_output<uint16_t>(0), 0x1234);
}

}
