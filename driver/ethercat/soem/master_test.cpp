// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/soem/master.h"
#include "driver/ethercat/soem/mock_api.h"

namespace driver::ethercat::soem {

class SOEMMasterTest : public ::testing::Test {
protected:
    MockAPI *mock{};
    std::unique_ptr<Master> master;

    void SetUp() override {
        auto api = std::make_unique<MockAPI>();
        this->mock = api.get();
        this->master = std::make_unique<Master>(std::move(api), "eth0");
    }

    void add_slave(MockSlaveInfo info = {}) {
        this->mock->slave_infos.push_back(std::move(info));
    }

    void add_default_slave() {
        MockSlaveInfo slave;
        slave.state = EC_STATE_PRE_OP;
        slave.eep_man = 0x1234;
        slave.eep_id = 0x5678;
        slave.eep_rev = 0x0001;
        slave.eep_ser = 0xABCD;
        slave.name = "TestSlave";
        slave.Ibits = 32;
        slave.Obits = 16;
        this->add_slave(slave);
    }

    void initialize_with_slave() {
        this->add_default_slave();
        ASSERT_NIL(this->master->initialize());
    }

    void activate_master() {
        this->mock->obytes = 2;
        this->mock->ibytes = 4;
        this->mock->outputs_wkc = 1;
        this->mock->inputs_wkc = 1;
        this->mock->receive_return = 3;
        ASSERT_NIL(this->master->activate());
    }
};

////////////////////// Initialization //////////////////////

TEST_F(SOEMMasterTest, InitializeHappyPath) {
    this->initialize_with_slave();
    auto slaves = this->master->slaves();
    ASSERT_EQ(slaves.size(), 1);
    EXPECT_EQ(slaves[0].properties.vendor_id, 0x1234);
    EXPECT_EQ(slaves[0].properties.product_code, 0x5678);
    EXPECT_EQ(slaves[0].properties.revision, 0x0001);
    EXPECT_EQ(slaves[0].properties.serial, 0xABCD);
    EXPECT_EQ(slaves[0].properties.name, "TestSlave");
    EXPECT_EQ(slaves[0].properties.input_bits, 32);
    EXPECT_EQ(slaves[0].properties.output_bits, 16);
}

TEST_F(SOEMMasterTest, InitializeInitFails) {
    this->add_default_slave();
    this->mock->init_return = 0;
    ASSERT_OCCURRED_AS(this->master->initialize(), errors::MASTER_INIT_ERROR);
    EXPECT_TRUE(this->mock->close_called);
}

TEST_F(SOEMMasterTest, InitializeConfigInitFails) {
    this->add_default_slave();
    this->mock->config_init_return = 0;
    ASSERT_OCCURRED_AS(this->master->initialize(), errors::MASTER_INIT_ERROR);
    EXPECT_TRUE(this->mock->close_called);
}

TEST_F(SOEMMasterTest, InitializeIdempotent) {
    this->initialize_with_slave();
    ASSERT_NIL(this->master->initialize());
}

/// @brief it should name and type each PDO entry from the CoE object dictionary of the
/// slave.
TEST_F(SOEMMasterTest, CoEDiscoveryUsesEntryNamesAndDataTypes) {
    MockSlaveInfo slave;
    slave.state = EC_STATE_PRE_OP;
    slave.eep_man = 0x1234;
    slave.eep_id = 0x5678;
    slave.mbx_proto = ECT_MBXPROT_COE;
    this->add_slave(slave);
    this->mock->sdo_objects = {
        {{ECT_SDO_TXPDOASSIGN, 0}, 1},
        {{ECT_SDO_TXPDOASSIGN, 1}, 0x1A00},
        {{0x1A00, 0}, 2},
        {{0x1A00, 1}, 0x60000110},
        {{0x1A00, 2}, 0x60000220},
        {{ECT_SDO_RXPDOASSIGN, 0}, 0},
    };
    this->mock->oe_entries = {
        {{0x6000, 1}, {"Status", 0x0003}},
        {{0x6000, 2}, {"Value", 0x0008}},
    };

    ASSERT_NIL(this->master->initialize());

    const auto slaves = this->master->slaves();
    ASSERT_EQ(slaves.size(), 1);
    const auto &inputs = slaves[0].properties.input_pdos;
    ASSERT_EQ(inputs.size(), 2);
    EXPECT_EQ(inputs[0].name, "Status");
    EXPECT_EQ(inputs[0].data_type, x::telem::INT16_T);
    EXPECT_EQ(inputs[1].name, "Value");
    EXPECT_EQ(inputs[1].data_type, x::telem::FLOAT32_T);
}

/// @brief it should type each PDO entry from the data type in the SII of the slave.
TEST_F(SOEMMasterTest, SIIDiscoveryUsesEntryDataTypes) {
    MockSlaveInfo slave;
    slave.state = EC_STATE_PRE_OP;
    this->add_slave(slave);
    constexpr uint16_t start = 0x40;
    this->mock->sii_categories = {{ECT_SII_PDO, start}};
    // One TxPDO, 0x1A00, with entries 0x6000:01 (INTEGER16) and 0x6000:02 (REAL32).
    const std::vector<uint8_t> category = {
        12, 0,    0x00, 0x1A, 2, 3,    0,    0, 0, 0,    0x00, 0x60, 1,
        0,  0x03, 16,   0,    0, 0x00, 0x60, 2, 0, 0x08, 32,   0,    0,
    };
    for (size_t i = 0; i < category.size(); ++i)
        this->mock->sii_bytes[start + i] = category[i];

    ASSERT_NIL(this->master->initialize());

    const auto slaves = this->master->slaves();
    ASSERT_EQ(slaves.size(), 1);
    const auto &inputs = slaves[0].properties.input_pdos;
    ASSERT_EQ(inputs.size(), 2);
    EXPECT_EQ(inputs[0].index, 0x6000);
    EXPECT_EQ(inputs[0].sub_index, 1);
    EXPECT_EQ(inputs[0].data_type, x::telem::INT16_T);
    EXPECT_EQ(inputs[1].sub_index, 2);
    EXPECT_EQ(inputs[1].data_type, x::telem::FLOAT32_T);
}

////////////////////// Activation //////////////////////

TEST_F(SOEMMasterTest, ActivateHappyPath) {
    this->initialize_with_slave();
    this->activate_master();
}

TEST_F(SOEMMasterTest, ActivateNotInitialized) {
    ASSERT_OCCURRED_AS(this->master->activate(), errors::ACTIVATION_ERROR);
}

TEST_F(SOEMMasterTest, ActivateAlreadyActivated) {
    this->initialize_with_slave();
    this->activate_master();
    ASSERT_OCCURRED_AS(this->master->activate(), errors::ACTIVATION_ERROR);
}

TEST_F(SOEMMasterTest, ActivateConfigMapFails) {
    this->initialize_with_slave();
    this->mock->config_map_return = 0;
    ASSERT_OCCURRED_AS(this->master->activate(), errors::ACTIVATION_ERROR);
}

TEST_F(SOEMMasterTest, ActivateSafeOpTransitionFails) {
    this->initialize_with_slave();
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    this->mock->statecheck_overrides[EC_STATE_SAFE_OP] = EC_STATE_PRE_OP;
    this->mock->slave_infos[0].ALstatuscode = 0x001D;
    const auto err = this->master->activate();
    ASSERT_OCCURRED_AS(err, errors::STATE_CHANGE_ERROR);
    EXPECT_EQ(
        err.data,
        "failed to reach SAFE-OP: slave 1 (TestSlave) is in PRE-OP: Invalid output "
        "configuration"
    );
}

TEST_F(SOEMMasterTest, ActivateOpTransitionFails) {
    this->initialize_with_slave();
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    this->mock->statecheck_overrides[EC_STATE_OPERATIONAL] = EC_STATE_SAFE_OP;
    ASSERT_OCCURRED_AS(this->master->activate(), errors::STATE_CHANGE_ERROR);
}

/// @brief it should exchange process data while it waits for the slaves to reach OP.
TEST_F(SOEMMasterTest, ActivateExchangesProcessDataUntilOperational) {
    this->initialize_with_slave();
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    this->mock->statecheck_pending_polls[EC_STATE_OPERATIONAL] = 3;
    ASSERT_NIL(this->master->activate());
    EXPECT_EQ(this->mock->send_count, 5);
    EXPECT_EQ(this->mock->receive_count, 5);
}

TEST_F(SOEMMasterTest, ActivateDisabledSlavesSetGroup) {
    MockSlaveInfo slave1;
    slave1.state = EC_STATE_PRE_OP;
    slave1.name = "ActiveSlave";
    this->add_slave(slave1);

    MockSlaveInfo slave2;
    slave2.state = EC_STATE_PRE_OP;
    slave2.name = "DisabledSlave";
    this->add_slave(slave2);

    ASSERT_NIL(this->master->initialize());
    this->master->set_slave_enabled(2, false);
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    ASSERT_NIL(this->master->activate());
    EXPECT_EQ(this->mock->slave_infos[0].group, 1);
    EXPECT_EQ(this->mock->slave_infos[1].group, 2);
    EXPECT_EQ(this->mock->mapped_group, 1);
}

/// @brief it should not request state changes for a disabled slave.
TEST_F(SOEMMasterTest, DisabledSlaveStaysOutOfStateTransitions) {
    MockSlaveInfo slave1;
    slave1.state = EC_STATE_PRE_OP;
    this->add_slave(slave1);
    MockSlaveInfo slave2;
    slave2.state = EC_STATE_PRE_OP;
    this->add_slave(slave2);

    ASSERT_NIL(this->master->initialize());
    this->master->set_slave_enabled(2, false);
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    ASSERT_NIL(this->master->activate());
    const std::vector<std::pair<uint16_t, uint16_t>> expected = {
        {1, EC_STATE_SAFE_OP},
        {1, EC_STATE_OPERATIONAL},
    };
    EXPECT_EQ(this->mock->state_change_requests, expected);
}

/// @brief it should send and receive process data for the group of enabled slaves only.
TEST_F(SOEMMasterTest, CyclicExchangeUsesEnabledGroupOnly) {
    this->initialize_with_slave();
    this->activate_master();
    this->mock->processdata_groups.clear();
    ASSERT_NIL(this->master->send());
    ASSERT_NIL(this->master->receive());
    EXPECT_EQ(this->mock->processdata_groups, std::vector<uint8_t>({1, 1}));
}

////////////////////// Receive //////////////////////

TEST_F(SOEMMasterTest, ReceiveNotActivated) {
    ASSERT_OCCURRED_AS(this->master->receive(), errors::CYCLIC_ERROR);
}

TEST_F(SOEMMasterTest, ReceiveNegativeWkc) {
    this->initialize_with_slave();
    this->activate_master();
    this->mock->receive_return = -1;
    ASSERT_OCCURRED_AS(this->master->receive(), errors::CYCLIC_ERROR);
}

TEST_F(SOEMMasterTest, ReceiveWkcMismatch) {
    this->initialize_with_slave();
    this->activate_master();
    this->mock->receive_return = 1;
    const auto err = this->master->receive();
    ASSERT_OCCURRED_AS(err, errors::WORKING_COUNTER_ERROR);
    EXPECT_EQ(err.data, "working counter mismatch: expected 3, got 1");
}

TEST_F(SOEMMasterTest, ReceiveWkcMatch) {
    this->initialize_with_slave();
    this->activate_master();
    ASSERT_NIL(this->master->receive());
}

////////////////////// Send //////////////////////

TEST_F(SOEMMasterTest, SendNotActivated) {
    ASSERT_OCCURRED_AS(this->master->send(), errors::CYCLIC_ERROR);
}

TEST_F(SOEMMasterTest, SendReturnsZero) {
    this->initialize_with_slave();
    this->activate_master();
    this->mock->send_return = 0;
    ASSERT_OCCURRED_AS(this->master->send(), errors::CYCLIC_ERROR);
}

TEST_F(SOEMMasterTest, SendSucceeds) {
    this->initialize_with_slave();
    this->activate_master();
    ASSERT_NIL(this->master->send());
}

////////////////////// Data Access //////////////////////

TEST_F(SOEMMasterTest, OutputDataBeforeActivation) {
    this->initialize_with_slave();
    EXPECT_TRUE(this->master->output_data().empty());
}

TEST_F(SOEMMasterTest, InputDataBeforeActivation) {
    this->initialize_with_slave();
    EXPECT_TRUE(this->master->input_data().empty());
}

TEST_F(SOEMMasterTest, OutputDataAfterActivation) {
    this->initialize_with_slave();
    this->activate_master();
    EXPECT_EQ(this->master->output_data().size(), 2);
}

TEST_F(SOEMMasterTest, InputDataAfterActivation) {
    this->initialize_with_slave();
    this->activate_master();
    EXPECT_EQ(this->master->input_data().size(), 4);
}

////////////////////// Slave State //////////////////////

TEST_F(SOEMMasterTest, SlaveStateInvalidPosition) {
    this->initialize_with_slave();
    EXPECT_EQ(this->master->slave_state(0), slave::State::UNKNOWN);
    EXPECT_EQ(this->master->slave_state(999), slave::State::UNKNOWN);
}

TEST_F(SOEMMasterTest, SlaveStateCorrectConversion) {
    this->initialize_with_slave();
    this->mock->slave_infos[0].state = EC_STATE_OPERATIONAL;
    EXPECT_EQ(this->master->slave_state(1), slave::State::OP);
    this->mock->slave_infos[0].state = EC_STATE_SAFE_OP;
    EXPECT_EQ(this->master->slave_state(1), slave::State::SAFE_OP);
    this->mock->slave_infos[0].state = EC_STATE_PRE_OP;
    EXPECT_EQ(this->master->slave_state(1), slave::State::PRE_OP);
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    EXPECT_EQ(this->master->slave_state(1), slave::State::INIT);
}

TEST_F(SOEMMasterTest, AllSlavesOperationalTrue) {
    this->initialize_with_slave();
    this->activate_master();
    this->mock->slave_infos[0].state = EC_STATE_OPERATIONAL;
    EXPECT_TRUE(this->master->all_slaves_operational());
}

TEST_F(SOEMMasterTest, AllSlavesOperationalFalse) {
    this->initialize_with_slave();
    this->activate_master();
    this->mock->slave_infos[0].state = EC_STATE_SAFE_OP;
    EXPECT_FALSE(this->master->all_slaves_operational());
}

TEST_F(SOEMMasterTest, AllSlavesOperationalNotActivated) {
    this->initialize_with_slave();
    EXPECT_FALSE(this->master->all_slaves_operational());
}

////////////////////// Deactivation //////////////////////

TEST_F(SOEMMasterTest, DeactivateResetsState) {
    this->initialize_with_slave();
    this->activate_master();
    this->master->deactivate();
    EXPECT_TRUE(this->mock->close_called);
    EXPECT_TRUE(this->master->slaves().empty());
    EXPECT_TRUE(this->master->output_data().empty());
    EXPECT_TRUE(this->master->input_data().empty());
}

TEST_F(SOEMMasterTest, DeactivateRequestsInitState) {
    this->initialize_with_slave();
    this->activate_master();
    this->mock->state_change_requests.clear();
    this->master->deactivate();
    bool found_init = false;
    for (const auto &[pos, state]: this->mock->state_change_requests)
        if (state == EC_STATE_INIT) found_init = true;
    EXPECT_TRUE(found_init);
}

TEST_F(SOEMMasterTest, DeactivateNotInitializedIsNoop) {
    this->master->deactivate();
    EXPECT_FALSE(this->mock->close_called);
}

////////////////////// Interface Name //////////////////////

TEST_F(SOEMMasterTest, InterfaceName) {
    EXPECT_EQ(this->master->interface_name(), "eth0");
}

////////////////////// Slave Enable/Disable //////////////////////

TEST_F(SOEMMasterTest, SetSlaveDisabledThenReEnabled) {
    this->initialize_with_slave();
    this->master->set_slave_enabled(1, false);
    this->master->set_slave_enabled(1, true);
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    ASSERT_NIL(this->master->activate());
    EXPECT_EQ(this->mock->slave_infos[0].group, 1);
}

}
