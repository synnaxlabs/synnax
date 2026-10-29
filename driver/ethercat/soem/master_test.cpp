// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/esi/known_devices.h"
#include "driver/ethercat/soem/master.h"
#include "driver/ethercat/soem/mock_api.h"

namespace driver::ethercat::soem {

class SOEMMasterTest : public ::testing::Test {
protected:
    MockAPI *mock{};
    std::unique_ptr<Master> master;
    x::breaker::Breaker brk;

    void SetUp() override {
        auto api = std::make_unique<MockAPI>();
        this->mock = api.get();
        this->master = std::make_unique<Master>(std::move(api), "eth0");
        this->brk.start();
    }

    void TearDown() override { this->brk.stop(); }

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
        this->mock->state_change_requests.clear();
        this->mock->written_slaves.clear();
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

/// @brief it should return the socket check error and not start SOEM when the Driver
/// cannot open a raw socket.
TEST_F(SOEMMasterTest, InitializeSocketCheckFails) {
    this->add_default_slave();
    this->mock->check_socket_err = x::errors::Error(
        errors::PERMISSION_DENIED,
        "permission denied opening eth0"
    );
    ASSERT_OCCURRED_AS(this->master->initialize(), errors::PERMISSION_DENIED);
    EXPECT_FALSE(this->mock->init_called);
}

TEST_F(SOEMMasterTest, InitializeInitFails) {
    this->add_default_slave();
    this->mock->init_return = 0;
    const auto err = this->master->initialize();
    ASSERT_OCCURRED_AS(err, errors::MASTER_INIT_ERROR);
    EXPECT_FALSE(err.matches(errors::NO_SLAVES_FOUND));
    EXPECT_EQ(
        err.data,
        "cannot open eth0 for raw Ethernet access. Run the Driver as root (or with "
        "CAP_NET_RAW on Linux), and install Npcap on Windows."
    );
    EXPECT_TRUE(this->mock->close_called);
}

/// @brief it should return a no slaves error and close the socket when no slave
/// answers.
TEST_F(SOEMMasterTest, InitializeReportsAnInterfaceWithNoSlaves) {
    this->mock->config_init_return = 0;
    const auto err = this->master->initialize();
    ASSERT_OCCURRED_AS(err, errors::NO_SLAVES_FOUND);
    EXPECT_EQ(err.data, "no EtherCAT slaves found on eth0");
    EXPECT_TRUE(this->mock->close_called);
}

/// @brief it should return the number of slaves that answer on the bus.
TEST_F(SOEMMasterTest, CountSlavesReportsTheSlavesThatAnswer) {
    this->initialize_with_slave();
    this->mock->bus_slaves = 3;
    EXPECT_EQ(ASSERT_NIL_P(this->master->count_slaves()), 3);
    this->mock->bus_slaves = 0;
    EXPECT_EQ(ASSERT_NIL_P(this->master->count_slaves()), 0);
}

/// @brief it should return a master init error when it counts slaves before
/// initialization.
TEST_F(SOEMMasterTest, CountSlavesFailsBeforeInitialize) {
    ASSERT_OCCURRED_AS_P(this->master->count_slaves(), errors::MASTER_INIT_ERROR);
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
    this->master->set_slave_config(2, {.enabled = false});
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
    this->master->set_slave_config(2, {.enabled = false});
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

/// @brief it should return no distributed clock time when no slave has a distributed
/// clock.
TEST_F(SOEMMasterTest, DCTimeIsAbsentWithoutADistributedClock) {
    this->initialize_with_slave();
    this->activate_master();
    EXPECT_EQ(this->mock->configdc_count, 1);
    EXPECT_EQ(this->master->dc_time(), std::nullopt);
}

/// @brief it should convert the distributed clock time, which counts from the year
/// 2000, to a Unix timestamp.
TEST_F(SOEMMasterTest, DCTimeCountsFromTheDCEpoch) {
    this->initialize_with_slave();
    this->mock->hasdc = true;
    this->activate_master();
    this->mock->dc_time_value = 1'000'000'000;
    EXPECT_EQ(
        this->master->dc_time(),
        x::telem::TimeStamp(x::telem::SECOND * 946'684'801)
    );
}

////////////////////// Slave State //////////////////////

/// @brief it should report a slave in OP with one state read and no state writes.
TEST_F(SOEMMasterTest, CheckReportsOperationalSlave) {
    this->initialize_with_slave();
    this->activate_master();
    this->mock->slave_infos[0].state = EC_STATE_OPERATIONAL;
    const auto reads = this->mock->readstate_count;
    EXPECT_EQ(
        this->master->check(this->brk),
        (std::vector<slave::Health>{{.position = 1, .state = slave::State::OP}})
    );
    EXPECT_EQ(this->mock->readstate_count, reads + 1);
    EXPECT_TRUE(this->mock->written_slaves.empty());
}

/// @brief it should report the state a slave was in before the check tries to recover
/// it.
TEST_F(SOEMMasterTest, CheckReportsStateBeforeRecovery) {
    this->initialize_with_slave();
    this->activate_master();
    this->mock->slave_infos[0].state = EC_STATE_PRE_OP;
    EXPECT_EQ(this->master->check(this->brk)[0].state, slave::State::PRE_OP);
    EXPECT_EQ(this->master->check(this->brk)[0].state, slave::State::PRE_OP);
}

/// @brief it should report a slave in SAFE_OP with an error as faulted and acknowledge
/// the error.
TEST_F(SOEMMasterTest, CheckAcknowledgesSafeOpError) {
    this->initialize_with_slave();
    this->activate_master();
    this->mock->slave_infos[0].state = EC_STATE_SAFE_OP + EC_STATE_ERROR;
    this->mock->slave_infos[0].ALstatuscode = 0x001B;
    const auto health = this->master->check(this->brk);
    ASSERT_EQ(health.size(), 1);
    EXPECT_EQ(health[0].state, slave::State::SAFE_OP);
    EXPECT_TRUE(health[0].faulted);
    EXPECT_EQ(health[0].fault, ec_ALstatuscode2string(0x001B));
    EXPECT_EQ(
        this->mock->state_change_requests,
        (std::vector<std::pair<uint16_t, uint16_t>>{
            {1, EC_STATE_SAFE_OP + EC_STATE_ACK}
        })
    );
    EXPECT_EQ(this->mock->written_slaves, std::vector<uint16_t>{1});
}

/// @brief it should request OP for a slave in SAFE_OP.
TEST_F(SOEMMasterTest, CheckRequestsOpFromSafeOp) {
    this->initialize_with_slave();
    this->activate_master();
    this->mock->slave_infos[0].state = EC_STATE_SAFE_OP;
    this->master->check(this->brk);
    EXPECT_EQ(
        this->mock->state_change_requests,
        (std::vector<std::pair<uint16_t, uint16_t>>{{1, EC_STATE_OPERATIONAL}})
    );
    EXPECT_EQ(this->mock->written_slaves, std::vector<uint16_t>{1});
}

/// @brief it should reconfigure a slave that is below SAFE_OP.
TEST_F(SOEMMasterTest, CheckReconfiguresSlaveBelowSafeOp) {
    this->initialize_with_slave();
    this->activate_master();
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->master->check(this->brk);
    EXPECT_EQ(this->mock->reconfigured_slaves, std::vector<uint16_t>{1});
}

/// @brief it should not reconfigure a slave after the breaker stops.
TEST_F(SOEMMasterTest, CheckTakesNoRecoveryStepOnceBreakerStops) {
    this->initialize_with_slave();
    this->activate_master();
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->brk.stop();
    EXPECT_EQ(this->master->check(this->brk)[0].state, slave::State::INIT);
    EXPECT_TRUE(this->mock->reconfigured_slaves.empty());
}

/// @brief it should mark a slave that does not answer as lost after a failed recovery.
TEST_F(SOEMMasterTest, CheckMarksSilentSlaveLost) {
    this->initialize_with_slave();
    this->activate_master();
    this->mock->slave_infos[0].state = EC_STATE_NONE;
    this->mock->statecheck_overrides[EC_STATE_OPERATIONAL] = EC_STATE_NONE;
    this->mock->recover_return = 0;
    EXPECT_FALSE(this->master->check(this->brk)[0].lost);
    EXPECT_TRUE(this->mock->slave_infos[0].lost);
    EXPECT_TRUE(this->master->check(this->brk)[0].lost);
    EXPECT_EQ(this->mock->recovered_slaves, (std::vector<uint16_t>{1, 1}));
}

/// @brief it should recover a lost slave and clear its lost flag.
TEST_F(SOEMMasterTest, CheckRecoversLostSlave) {
    this->initialize_with_slave();
    this->activate_master();
    this->mock->slave_infos[0].state = EC_STATE_NONE;
    this->mock->slave_infos[0].lost = true;
    this->master->check(this->brk);
    EXPECT_EQ(this->mock->recovered_slaves, std::vector<uint16_t>{1});
    EXPECT_FALSE(this->mock->slave_infos[0].lost);
}

/// @brief it should clear the lost flag of a slave that answers again, without a
/// recovery.
TEST_F(SOEMMasterTest, CheckFindsLostSlaveThatAnswers) {
    this->initialize_with_slave();
    this->activate_master();
    this->mock->slave_infos[0].state = EC_STATE_OPERATIONAL;
    this->mock->slave_infos[0].lost = true;
    this->master->check(this->brk);
    EXPECT_TRUE(this->mock->recovered_slaves.empty());
    EXPECT_FALSE(this->mock->slave_infos[0].lost);
}

/// @brief it should leave a disabled slave out of the health report and not write its
/// state.
TEST_F(SOEMMasterTest, CheckSkipsDisabledSlave) {
    this->add_default_slave();
    this->add_default_slave();
    this->master->set_slave_config(2, {.enabled = false});
    ASSERT_NIL(this->master->initialize());
    this->activate_master();
    this->mock->slave_infos[1].state = EC_STATE_SAFE_OP;
    const auto health = this->master->check(this->brk);
    ASSERT_EQ(health.size(), 1);
    EXPECT_EQ(health[0].position, 1);
    EXPECT_TRUE(this->mock->written_slaves.empty());
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
    this->master->set_slave_config(1, {.enabled = false});
    this->master->set_slave_config(1, {});
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    ASSERT_NIL(this->master->activate());
    EXPECT_EQ(this->mock->slave_infos[0].group, 1);
}

////////////////////// PDO Layout //////////////////////

namespace {
/// @brief Beckhoff's vendor ID and the EL1008 product code, which the ESI registry
/// lists with eight 1-bit inputs.
constexpr uint32_t BECKHOFF = 0x00000002;
constexpr uint32_t EL1008 = 0x03F03052;

/// @brief returns a slave that answers CoE requests.
MockSlaveInfo coe_slave(const uint16_t input_bits) {
    MockSlaveInfo info;
    info.state = EC_STATE_PRE_OP;
    info.eep_man = 0x1234;
    info.eep_id = 0x5678;
    info.name = "TestSlave";
    info.mbx_proto = ECT_MBXPROT_COE;
    info.Ibits = input_bits;
    return info;
}

/// @brief serves one TxPDO, 0x1A00, holding a 1-bit entry, 7 bits of padding, and a
/// 16-bit entry.
void serve_padded_txpdo(MockAPI &mock) {
    mock.sdo_objects = {
        {{ECT_SDO_TXPDOASSIGN, 0}, 1},
        {{ECT_SDO_TXPDOASSIGN, 1}, 0x1A00},
        {{0x1A00, 0}, 3},
        {{0x1A00, 1}, 0x60000101},
        {{0x1A00, 2}, 0x00000007},
        {{0x1A00, 3}, 0x60001110},
        {{ECT_SDO_RXPDOASSIGN, 0}, 0},
    };
    mock.oe_entries = {
        {{0x6000, 1}, {"Underrange", 0x0001}},
        {{0x6000, 17}, {"Value", 0x0003}},
    };
}
}

/// @brief it should keep padding entries and the PDO name when it discovers PDOs
/// through CoE.
TEST_F(SOEMMasterTest, CoEDiscoveryKeepsPaddingAndPDOName) {
    this->add_slave(coe_slave(24));
    serve_padded_txpdo(*this->mock);
    this->mock->od_names[0x1A00] = "Channel 1";
    ASSERT_NIL(this->master->initialize());

    const auto inputs = this->master->slaves()[0].properties.input_pdos;
    ASSERT_EQ(inputs.size(), 3);
    EXPECT_EQ(inputs[0].name, "Underrange");
    EXPECT_EQ(inputs[0].pdo_name, "Channel 1");
    EXPECT_TRUE(inputs[1].is_padding());
    EXPECT_EQ(inputs[1].bit_length, 7);
    EXPECT_EQ(inputs[2].sub_index, 17);
    EXPECT_EQ(inputs[2].pdo_name, "Channel 1");
}

/// @brief it should leave the PDO name empty when the CoE object has no name.
TEST_F(SOEMMasterTest, CoEDiscoveryWithoutPDOName) {
    this->add_slave(coe_slave(24));
    serve_padded_txpdo(*this->mock);
    ASSERT_NIL(this->master->initialize());

    const auto inputs = this->master->slaves()[0].properties.input_pdos;
    ASSERT_EQ(inputs.size(), 3);
    EXPECT_FALSE(inputs[0].pdo_name.has_value());
}

/// @brief it should keep padding entries and the PDO name when it discovers PDOs
/// through the SII.
TEST_F(SOEMMasterTest, SIIDiscoveryKeepsPaddingAndPDOName) {
    MockSlaveInfo slave;
    slave.state = EC_STATE_PRE_OP;
    this->add_slave(slave);
    constexpr uint16_t start = 0x40;
    this->mock->sii_categories = {{ECT_SII_PDO, start}};
    // One TxPDO, 0x1A00 named by string 1, with 0x6000:01 (1 bit, string 2), 7 bits of
    // padding, and 0x6000:11 (INTEGER16, string 3).
    const std::vector<uint8_t> category = {
        16, 0,    0x00, 0x1A, 3, 3, 0, 1, 0, 0,    0x00, 0x60, 1, 2,    0x01, 1, 0,
        0,  0x00, 0x00, 0,    0, 0, 7, 0, 0, 0x00, 0x60, 17,   3, 0x03, 16,   0, 0,
    };
    for (size_t i = 0; i < category.size(); ++i)
        this->mock->sii_bytes[start + i] = category[i];
    this->mock->sii_strings = {{1, "Channel 1"}, {2, "Underrange"}, {3, "Value"}};

    ASSERT_NIL(this->master->initialize());

    const auto inputs = this->master->slaves()[0].properties.input_pdos;
    ASSERT_EQ(inputs.size(), 3);
    EXPECT_EQ(inputs[0].name, "Underrange");
    EXPECT_EQ(inputs[0].pdo_name, "Channel 1");
    EXPECT_TRUE(inputs[1].is_padding());
    EXPECT_EQ(inputs[1].bit_length, 7);
    EXPECT_EQ(inputs[2].name, "Value");
    EXPECT_EQ(inputs[2].pdo_name, "Channel 1");
}

/// @brief it should use the mapping read from the slave over the device registry.
TEST_F(SOEMMasterTest, LiveMappingTakesPrecedenceOverRegistry) {
    auto info = coe_slave(24);
    info.eep_man = BECKHOFF;
    info.eep_id = EL1008;
    this->add_slave(info);
    serve_padded_txpdo(*this->mock);
    ASSERT_NIL(this->master->initialize());

    const auto inputs = this->master->slaves()[0].properties.input_pdos;
    ASSERT_EQ(inputs.size(), 3);
    EXPECT_EQ(inputs[2].index, 0x6000);
    EXPECT_EQ(inputs[2].sub_index, 17);
}

/// @brief it should name the live entries of a known device from the device registry,
/// so a rescan keeps the names that automatic channels refer to.
TEST_F(SOEMMasterTest, RegistryNamesLiveEntriesOfAKnownDevice) {
    slave::Properties registry{};
    ASSERT_TRUE(esi::lookup_device_pdos(BECKHOFF, EL1008, 0, registry));
    auto info = coe_slave(24);
    info.eep_man = BECKHOFF;
    info.eep_id = EL1008;
    this->add_slave(info);
    serve_padded_txpdo(*this->mock);
    ASSERT_NIL(this->master->initialize());

    const auto inputs = this->master->slaves()[0].properties.input_pdos;
    ASSERT_EQ(inputs.size(), 3);
    EXPECT_EQ(inputs[0].name, registry.input_pdos.at(0).name);
    EXPECT_EQ(inputs[2].name, "Value");
}

/// @brief it should describe a slave from the device registry when the slave reports no
/// mapping.
TEST_F(SOEMMasterTest, RegistryDescribesSlaveWithoutLiveMapping) {
    MockSlaveInfo info;
    info.state = EC_STATE_PRE_OP;
    info.eep_man = BECKHOFF;
    info.eep_id = EL1008;
    this->add_slave(info);
    ASSERT_NIL(this->master->initialize());

    const auto slave = this->master->slaves()[0];
    EXPECT_EQ(slave.properties.input_pdos.size(), 8);
    EXPECT_TRUE(slave.status.pdo_discovery_error.empty());
}

/// @brief it should describe a slave from the device registry when the slave assigns a
/// PDO whose mapping it cannot read.
TEST_F(SOEMMasterTest, RegistryDescribesSlaveWithUnreadableMapping) {
    auto info = coe_slave(8);
    info.eep_man = BECKHOFF;
    info.eep_id = EL1008;
    this->add_slave(info);
    this->mock->sdo_objects = {
        {{ECT_SDO_TXPDOASSIGN, 0}, 1},
        {{ECT_SDO_TXPDOASSIGN, 1}, 0x1A00},
        {{ECT_SDO_RXPDOASSIGN, 0}, 0},
    };
    ASSERT_NIL(this->master->initialize());

    const auto slave = this->master->slaves()[0];
    EXPECT_EQ(slave.properties.input_pdos.size(), 8);
    EXPECT_TRUE(slave.status.pdo_discovery_error.empty());
}

/// @brief it should count padding when it computes entry offsets.
TEST_F(SOEMMasterTest, OffsetsCountPadding) {
    this->add_slave(coe_slave(24));
    serve_padded_txpdo(*this->mock);
    ASSERT_NIL(this->master->initialize());
    this->mock->ibytes = 3;
    ASSERT_NIL(this->master->activate());

    const auto offset = ASSERT_NIL_P(
        this->master->pdo_offset(pdo::Entry(1, 0x6000, 17, 16, true, x::telem::INT16_T))
    );
    EXPECT_EQ(offset.byte, 1);
    EXPECT_EQ(offset.bit, 0);
}

/// @brief it should return a mapping error for a slave whose discovered entries do not
/// match the bits it maps.
TEST_F(SOEMMasterTest, SlaveWhoseLayoutDisagreesWithMappingHasNoOffsets) {
    this->add_slave(coe_slave(16));
    serve_padded_txpdo(*this->mock);
    ASSERT_NIL(this->master->initialize());
    this->mock->ibytes = 2;
    ASSERT_NIL(this->master->activate());

    const auto err = this->master
                         ->pdo_offset(
                             pdo::Entry(1, 0x6000, 17, 16, true, x::telem::INT16_T)
                         )
                         .second;
    ASSERT_OCCURRED_AS(err, errors::PDO_MAPPING_ERROR);
    EXPECT_EQ(
        err.data,
        "slave 1 (TestSlave) maps 16 input and 0 output bits, but its discovered PDO "
        "entries hold 24 and 0"
    );
}

////////////////////// Startup Parameters //////////////////////

const slave::StartupParameter FILTER{
    .index = 0x8000,
    .sub_index = 0x15,
    .data_type = x::telem::UINT16_T,
    .value = 3,
    .name = "Filter",
};

const slave::StartupParameter RANGE{
    .index = 0x8000,
    .sub_index = 0x19,
    .data_type = x::telem::UINT8_T,
    .value = 2,
};

/// @brief it should write the startup parameters of a slave in their given order.
TEST_F(SOEMMasterTest, ActivateWritesStartupParametersInOrder) {
    this->initialize_with_slave();
    this->master->set_slave_config(1, {.startup_parameters = {FILTER, RANGE}});
    this->activate_master();
    EXPECT_EQ(
        this->mock->sdo_writes,
        (std::vector<MockAPI::SDOWrite>{
            {1, 0x8000, 0x15, {0x03, 0x00}},
            {1, 0x8000, 0x19, {0x02}},
        })
    );
}

/// @brief it should fail activation and stop writing when a slave aborts a startup
/// parameter.
TEST_F(SOEMMasterTest, ActivateFailsWhenSlaveRejectsStartupParameter) {
    this->initialize_with_slave();
    this->master->set_slave_config(1, {.startup_parameters = {FILTER, RANGE}});
    this->mock->sdo_write_aborts[{0x8000, 0x15}] = 0x06090031;
    const auto err = this->master->activate();
    ASSERT_OCCURRED_AS(err, errors::STARTUP_PARAMETER_ERROR);
    EXPECT_EQ(
        err.data,
        "slave 1 (TestSlave) rejected startup parameter 0x8000:15 (Filter): Value of "
        "parameter written too high"
    );
    EXPECT_EQ(this->mock->sdo_writes.size(), 1);
    EXPECT_TRUE(this->mock->state_change_requests.empty());
}

/// @brief it should fail activation when a startup parameter write gets no response.
TEST_F(SOEMMasterTest, ActivateFailsWhenStartupParameterGetsNoResponse) {
    this->initialize_with_slave();
    this->master->set_slave_config(1, {.startup_parameters = {RANGE}});
    this->mock->sdo_write_aborts[{0x8000, 0x19}] = 0;
    const auto err = this->master->activate();
    ASSERT_OCCURRED_AS(err, errors::STARTUP_PARAMETER_ERROR);
    EXPECT_EQ(
        err.data,
        "slave 1 (TestSlave) rejected startup parameter 0x8000:19: no response"
    );
}

/// @brief it should not blame an abort of another object for a failed startup parameter
/// write.
TEST_F(SOEMMasterTest, StartupFailureIgnoresAbortsOfOtherObjects) {
    this->initialize_with_slave();
    this->master->set_slave_config(1, {.startup_parameters = {RANGE}});
    this->mock->push_sdo_abort(1, 0x1C12, 0, 0x06020000);
    this->mock->sdo_write_aborts[{0x8000, 0x19}] = 0;
    const auto err = this->master->activate();
    ASSERT_OCCURRED_AS(err, errors::STARTUP_PARAMETER_ERROR);
    EXPECT_EQ(
        err.data,
        "slave 1 (TestSlave) rejected startup parameter 0x8000:19: no response"
    );
}

/// @brief it should write no startup parameters after a new config clears them.
TEST_F(SOEMMasterTest, ActivateWritesNothingAfterStartupParametersAreCleared) {
    this->initialize_with_slave();
    this->master->set_slave_config(1, {.startup_parameters = {FILTER}});
    this->master->set_slave_config(1, {});
    this->activate_master();
    EXPECT_TRUE(this->mock->sdo_writes.empty());
}

/// @brief it should not write the startup parameters of a disabled slave.
TEST_F(SOEMMasterTest, ActivateSkipsStartupParametersOfDisabledSlave) {
    this->initialize_with_slave();
    this->master->set_slave_config(
        1,
        {.enabled = false, .startup_parameters = {FILTER}}
    );
    this->activate_master();
    EXPECT_TRUE(this->mock->sdo_writes.empty());
}

/// @brief it should read the PDO layout after the startup parameters change the PDO
/// assignment.
TEST_F(SOEMMasterTest, ActivateReadsLayoutThatStartupParametersAssign) {
    this->add_slave(coe_slave(24));
    serve_padded_txpdo(*this->mock);
    this->mock->sdo_objects[{0x1A01, 0}] = 1;
    this->mock->sdo_objects[{0x1A01, 1}] = 0x60100110;
    this->mock->oe_entries[{0x6010, 1}] = {"Value", 0x0003};
    ASSERT_NIL(this->master->initialize());
    const slave::StartupParameter assign_1a01{
        .index = ECT_SDO_TXPDOASSIGN,
        .sub_index = 1,
        .data_type = x::telem::UINT16_T,
        .value = 0x1A01,
    };
    this->master->set_slave_config(1, {.startup_parameters = {assign_1a01}});
    this->mock->slave_infos[0].Ibits = 16;
    this->mock->ibytes = 2;
    ASSERT_NIL(this->master->activate());

    const auto offset = ASSERT_NIL_P(
        this->master->pdo_offset(pdo::Entry(1, 0x6010, 1, 16, true, x::telem::INT16_T))
    );
    EXPECT_EQ(offset.byte, 0);
    EXPECT_EQ(this->master->slaves()[0].properties.input_pdos.size(), 1);
}

/// @brief it should write the startup parameters again when it configures a slave that
/// fell back to INIT.
TEST_F(SOEMMasterTest, CheckWritesStartupParametersAgainOnReconfig) {
    this->initialize_with_slave();
    this->master->set_slave_config(1, {.startup_parameters = {FILTER}});
    this->activate_master();
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->master->check(this->brk);
    EXPECT_EQ(this->mock->sdo_writes.size(), 2);
    EXPECT_TRUE(this->mock->state_change_requests.empty());
}

/// @brief it should hold a slave in PRE-OP and report the failure when a startup
/// parameter write fails during reconfiguration.
TEST_F(SOEMMasterTest, CheckHoldsSlaveInPreOpWhenReconfigWriteFails) {
    this->initialize_with_slave();
    this->master->set_slave_config(1, {.startup_parameters = {FILTER}});
    this->activate_master();
    this->mock->sdo_write_aborts[{0x8000, 0x15}] = 0x06090031;
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->master->check(this->brk);
    EXPECT_EQ(
        this->mock->state_change_requests,
        (std::vector<std::pair<uint16_t, uint16_t>>{{1, EC_STATE_PRE_OP}})
    );
    this->mock->slave_infos[0].state = EC_STATE_PRE_OP;
    const auto health = this->master->check(this->brk);
    ASSERT_EQ(health.size(), 1);
    EXPECT_TRUE(health[0].faulted);
    EXPECT_EQ(
        health[0].fault,
        "rejected startup parameter 0x8000:15 (Filter): Value of parameter written "
        "too high"
    );
}

/// @brief it should not reconfigure a slave again after it rejects a startup parameter
/// during reconfiguration.
TEST_F(SOEMMasterTest, CheckStopsRecoveringSlaveThatRejectsStartupParameter) {
    this->initialize_with_slave();
    this->master->set_slave_config(1, {.startup_parameters = {FILTER}});
    this->activate_master();
    this->mock->sdo_write_aborts[{0x8000, 0x15}] = 0x06090031;
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->master->check(this->brk);
    this->mock->sdo_write_aborts.clear();
    this->mock->slave_infos[0].state = EC_STATE_PRE_OP;
    const auto health = this->master->check(this->brk);
    EXPECT_EQ(this->mock->reconfigured_slaves, std::vector<uint16_t>{1});
    ASSERT_EQ(health.size(), 1);
    EXPECT_TRUE(health[0].faulted);
}

/// @brief it should recover a slave that rejected a startup parameter once the master
/// activates again.
TEST_F(SOEMMasterTest, ActivateClearsStartupFailureFromRecovery) {
    this->initialize_with_slave();
    this->master->set_slave_config(1, {.startup_parameters = {FILTER}});
    this->activate_master();
    this->mock->sdo_write_aborts[{0x8000, 0x15}] = 0x06090031;
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->master->check(this->brk);
    this->mock->sdo_write_aborts.clear();
    this->master->deactivate();
    ASSERT_NIL(this->master->initialize());
    this->activate_master();
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->master->check(this->brk);
    EXPECT_EQ(this->mock->reconfigured_slaves, (std::vector<uint16_t>{1, 1}));
}
}
