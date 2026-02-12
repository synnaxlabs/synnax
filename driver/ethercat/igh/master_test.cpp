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
#include "driver/ethercat/igh/master.h"
#include "driver/ethercat/igh/mock_api.h"

namespace driver::ethercat::igh {

class IGHMasterTest : public ::testing::Test {
protected:
    std::shared_ptr<MockAPI> mock_api;
    std::unique_ptr<Master> master_instance;

    void SetUp() override {
        this->mock_api = std::make_shared<MockAPI>();
        this->master_instance = std::make_unique<Master>(this->mock_api, 0);
    }

    void add_default_slave(
        uint16_t position = 0,
        uint32_t vendor = 0xBEEF,
        uint32_t product = 0xCAFE
    ) {
        this->mock_api->add_slave(position, vendor, product, 1, 100, "TestSlave");
    }

    void initialize_with_slave() {
        this->add_default_slave();
        ASSERT_NIL(this->master_instance->initialize());
    }

    void activate_with_domains(size_t output_sz = 16, size_t input_sz = 16) {
        this->mock_api->set_output_domain_size(output_sz);
        this->mock_api->set_input_domain_size(input_sz);
        ASSERT_NIL(this->master_instance->activate());
    }
};

TEST_F(IGHMasterTest, InitializeHappyPath) {
    this->add_default_slave();
    ASSERT_NIL(this->master_instance->initialize());
    auto slaves = this->master_instance->slaves();
    EXPECT_EQ(slaves.size(), 1u);
}

TEST_F(IGHMasterTest, InitializeRequestMasterFails) {
    this->mock_api->set_request_master_result(nullptr);
    ASSERT_OCCURRED_AS(this->master_instance->initialize(), errors::MASTER_INIT_ERROR);
}

TEST_F(IGHMasterTest, InitializeMasterInfoFails) {
    this->mock_api->set_master_info_result(-1);
    ASSERT_OCCURRED_AS(this->master_instance->initialize(), errors::MASTER_INIT_ERROR);
    EXPECT_TRUE(this->mock_api->release_master_called());
}

TEST_F(IGHMasterTest, InitializeOutputDomainFails) {
    this->add_default_slave();
    this->mock_api->set_create_domain_fails_output(true);
    ASSERT_OCCURRED_AS(this->master_instance->initialize(), errors::MASTER_INIT_ERROR);
    EXPECT_TRUE(this->mock_api->release_master_called());
}

TEST_F(IGHMasterTest, InitializeInputDomainFails) {
    this->add_default_slave();
    this->mock_api->set_create_domain_fails_input(true);
    ASSERT_OCCURRED_AS(this->master_instance->initialize(), errors::MASTER_INIT_ERROR);
    EXPECT_TRUE(this->mock_api->release_master_called());
}

TEST_F(IGHMasterTest, InitializeIdempotent) {
    this->add_default_slave();
    ASSERT_NIL(this->master_instance->initialize());
    ASSERT_NIL(this->master_instance->initialize());
}

TEST_F(IGHMasterTest, InitializePopulatesSlaveInfo) {
    this->mock_api->add_slave(0, 0x1234, 0x5678, 3, 42, "EL2008");
    ASSERT_NIL(this->master_instance->initialize());
    auto slaves = this->master_instance->slaves();
    ASSERT_EQ(slaves.size(), 1u);
    EXPECT_EQ(slaves[0].properties.vendor_id, 0x1234u);
    EXPECT_EQ(slaves[0].properties.product_code, 0x5678u);
    EXPECT_EQ(slaves[0].properties.revision, 3u);
    EXPECT_EQ(slaves[0].properties.serial, 42u);
    EXPECT_EQ(slaves[0].properties.name, "EL2008");
    EXPECT_EQ(slaves[0].properties.position, 0u);
}

TEST_F(IGHMasterTest, InitializeMultipleSlaves) {
    this->mock_api->add_slave(0, 0x1111, 0x2222, 1, 10, "Slave0");
    this->mock_api->add_slave(1, 0x3333, 0x4444, 2, 20, "Slave1");
    ASSERT_NIL(this->master_instance->initialize());
    auto slaves = this->master_instance->slaves();
    ASSERT_EQ(slaves.size(), 2u);
    EXPECT_EQ(slaves[0].properties.name, "Slave0");
    EXPECT_EQ(slaves[1].properties.name, "Slave1");
}

TEST_F(IGHMasterTest, RegisterPDOsAfterActivationFails) {
    this->initialize_with_slave();
    this->activate_with_domains();
    std::vector<pdo::Entry> entries = {{0, 0x6000, 0x01, 16, true, x::telem::UINT16_T}};
    ASSERT_OCCURRED_AS(
        this->master_instance->register_pdos(entries),
        errors::PDO_MAPPING_ERROR
    );
}

TEST_F(IGHMasterTest, RegisterPDOsSlaveConfigFails) {
    this->initialize_with_slave();
    this->mock_api->set_slave_config_fails(true);
    std::vector<pdo::Entry> entries = {{0, 0x6000, 0x01, 16, true, x::telem::UINT16_T}};
    ASSERT_OCCURRED_AS(
        this->master_instance->register_pdos(entries),
        errors::PDO_MAPPING_ERROR
    );
}

TEST_F(IGHMasterTest, RegisterPDOsDisabledSlaveReturnsError) {
    this->initialize_with_slave();
    this->master_instance->set_slave_enabled(0, false);
    std::vector<pdo::Entry> entries = {{0, 0x6000, 0x01, 16, true, x::telem::UINT16_T}};
    ASSERT_OCCURRED_AS(
        this->master_instance->register_pdos(entries),
        errors::PDO_MAPPING_ERROR
    );
}

TEST_F(IGHMasterTest, ActivateHappyPath) {
    this->initialize_with_slave();
    this->mock_api->set_output_domain_size(8);
    this->mock_api->set_input_domain_size(8);
    ASSERT_NIL(this->master_instance->activate());
}

TEST_F(IGHMasterTest, ActivateNotInitializedFails) {
    ASSERT_OCCURRED_AS(this->master_instance->activate(), errors::ACTIVATION_ERROR);
}

TEST_F(IGHMasterTest, ActivateIdempotent) {
    this->initialize_with_slave();
    this->activate_with_domains();
    ASSERT_NIL(this->master_instance->activate());
}

TEST_F(IGHMasterTest, ActivateMasterActivateFails) {
    this->initialize_with_slave();
    this->mock_api->set_activate_result(-1);
    ASSERT_OCCURRED_AS(this->master_instance->activate(), errors::ACTIVATION_ERROR);
}

TEST_F(IGHMasterTest, ActivateOutputDomainDataNullWithNonzeroSize) {
    this->initialize_with_slave();
    this->mock_api->set_output_domain_size(0);
    this->mock_api->set_input_domain_size(8);
    ASSERT_NIL(this->master_instance->activate());
}

TEST_F(IGHMasterTest, ActivateInputDomainDataNullWithNonzeroSize) {
    this->initialize_with_slave();
    this->mock_api->set_output_domain_size(8);
    this->mock_api->set_input_domain_size(0);
    ASSERT_NIL(this->master_instance->activate());
}

TEST_F(IGHMasterTest, ActivateZeroDomainSizesSucceeds) {
    this->initialize_with_slave();
    this->mock_api->set_output_domain_size(0);
    this->mock_api->set_input_domain_size(0);
    ASSERT_NIL(this->master_instance->activate());
}

TEST_F(IGHMasterTest, ReceiveNotActivatedFails) {
    ASSERT_OCCURRED_AS(this->master_instance->receive(), errors::CYCLIC_ERROR);
}

TEST_F(IGHMasterTest, ReceiveWcZeroReturnsError) {
    this->initialize_with_slave();
    this->activate_with_domains();
    ec_domain_state_t zero_state{};
    zero_state.wc_state = EC_WC_ZERO;
    this->mock_api->set_output_domain_state(zero_state);
    this->mock_api->set_input_domain_state(zero_state);
    ASSERT_OCCURRED_AS(this->master_instance->receive(), errors::WORKING_COUNTER_ERROR);
}

TEST_F(IGHMasterTest, ReceiveWcIncompleteReturnsNil) {
    this->initialize_with_slave();
    this->activate_with_domains();
    ec_domain_state_t incomplete{};
    incomplete.wc_state = EC_WC_INCOMPLETE;
    this->mock_api->set_output_domain_state(incomplete);
    this->mock_api->set_input_domain_state(incomplete);
    ASSERT_NIL(this->master_instance->receive());
}

TEST_F(IGHMasterTest, ReceiveWcCompleteReturnsNil) {
    this->initialize_with_slave();
    this->activate_with_domains();
    ec_domain_state_t complete{};
    complete.wc_state = EC_WC_COMPLETE;
    this->mock_api->set_output_domain_state(complete);
    this->mock_api->set_input_domain_state(complete);
    ASSERT_NIL(this->master_instance->receive());
}

TEST_F(IGHMasterTest, ReceiveOutputZeroInputComplete) {
    this->initialize_with_slave();
    this->activate_with_domains();
    ec_domain_state_t zero_state{};
    zero_state.wc_state = EC_WC_ZERO;
    ec_domain_state_t complete{};
    complete.wc_state = EC_WC_COMPLETE;
    this->mock_api->set_output_domain_state(zero_state);
    this->mock_api->set_input_domain_state(complete);
    ASSERT_OCCURRED_AS(this->master_instance->receive(), errors::WORKING_COUNTER_ERROR);
}

TEST_F(IGHMasterTest, SendNotActivatedFails) {
    ASSERT_OCCURRED_AS(this->master_instance->send(), errors::CYCLIC_ERROR);
}

TEST_F(IGHMasterTest, SendSucceeds) {
    this->initialize_with_slave();
    this->activate_with_domains();
    ASSERT_NIL(this->master_instance->send());
}

TEST_F(IGHMasterTest, InputDataBeforeActivation) {
    auto data = this->master_instance->input_data();
    EXPECT_TRUE(data.empty());
}

TEST_F(IGHMasterTest, OutputDataBeforeActivation) {
    auto data = this->master_instance->output_data();
    EXPECT_TRUE(data.empty());
}

TEST_F(IGHMasterTest, InputDataAfterActivation) {
    this->initialize_with_slave();
    this->mock_api->set_input_domain_size(8);
    this->mock_api->set_output_domain_size(8);
    ASSERT_NIL(this->master_instance->activate());
    auto data = this->master_instance->input_data();
    EXPECT_EQ(data.size(), 8u);
}

TEST_F(IGHMasterTest, OutputDataAfterActivation) {
    this->initialize_with_slave();
    this->mock_api->set_output_domain_size(4);
    this->mock_api->set_input_domain_size(4);
    ASSERT_NIL(this->master_instance->activate());
    auto data = this->master_instance->output_data();
    EXPECT_EQ(data.size(), 4u);
}

TEST_F(IGHMasterTest, SlaveStateInvalidPosition) {
    this->initialize_with_slave();
    EXPECT_EQ(this->master_instance->slave_state(99), slave::State::UNKNOWN);
}

TEST_F(IGHMasterTest, SlaveStateNoConfig) {
    this->initialize_with_slave();
    EXPECT_EQ(this->master_instance->slave_state(0), slave::State::UNKNOWN);
}

TEST_F(IGHMasterTest, SlaveStateReturnsCorrectState) {
    this->initialize_with_slave();
    this->master_instance->get_or_create_slave_config(0);
    ec_slave_config_state_t op_state{};
    op_state.al_state = 0x08;
    op_state.online = 1;
    op_state.operational = 1;
    this->mock_api->set_slave_config_state(0, op_state);
    EXPECT_EQ(this->master_instance->slave_state(0), slave::State::OP);
}

TEST_F(IGHMasterTest, SlaveStatePreOp) {
    this->initialize_with_slave();
    this->master_instance->get_or_create_slave_config(0);
    ec_slave_config_state_t preop_state{};
    preop_state.al_state = 0x02;
    this->mock_api->set_slave_config_state(0, preop_state);
    EXPECT_EQ(this->master_instance->slave_state(0), slave::State::PRE_OP);
}

TEST_F(IGHMasterTest, AllSlavesOperationalTrue) {
    this->initialize_with_slave();
    this->master_instance->get_or_create_slave_config(0);
    this->activate_with_domains();
    this->mock_api->set_all_slaves_al_state(0x08);
    EXPECT_TRUE(this->master_instance->all_slaves_operational());
}

TEST_F(IGHMasterTest, AllSlavesOperationalFalse) {
    this->initialize_with_slave();
    this->master_instance->get_or_create_slave_config(0);
    this->activate_with_domains();
    this->mock_api->set_all_slaves_al_state(0x04);
    EXPECT_FALSE(this->master_instance->all_slaves_operational());
}

TEST_F(IGHMasterTest, AllSlavesOperationalNotActivated) {
    this->initialize_with_slave();
    EXPECT_FALSE(this->master_instance->all_slaves_operational());
}

TEST_F(IGHMasterTest, AllSlavesOperationalSkipsDisabled) {
    this->mock_api->add_slave(0, 0x1111, 0x2222, 1, 1, "S0");
    this->mock_api->add_slave(1, 0x3333, 0x4444, 1, 2, "S1");
    ASSERT_NIL(this->master_instance->initialize());
    this->master_instance->get_or_create_slave_config(0);
    this->master_instance->get_or_create_slave_config(1);
    this->master_instance->set_slave_enabled(1, false);
    this->activate_with_domains();
    this->mock_api->set_all_slaves_al_state(0x08);
    EXPECT_TRUE(this->master_instance->all_slaves_operational());
}

TEST_F(IGHMasterTest, DeactivateResetsState) {
    this->initialize_with_slave();
    this->activate_with_domains();
    this->master_instance->deactivate();
    EXPECT_TRUE(this->master_instance->input_data().empty());
    EXPECT_TRUE(this->master_instance->output_data().empty());
    EXPECT_TRUE(this->master_instance->slaves().empty());
}

TEST_F(IGHMasterTest, DeactivateCallsReleaseMaster) {
    this->initialize_with_slave();
    this->activate_with_domains();
    this->master_instance->deactivate();
    EXPECT_TRUE(this->mock_api->release_master_called());
}

TEST_F(IGHMasterTest, DeactivateNotInitializedIsNoOp) {
    this->master_instance->deactivate();
    EXPECT_FALSE(this->mock_api->release_master_called());
}

TEST_F(IGHMasterTest, InterfaceName) {
    EXPECT_EQ(this->master_instance->interface_name(), "igh:0");
    Master m2(this->mock_api, 3);
    EXPECT_EQ(m2.interface_name(), "igh:3");
}

TEST_F(IGHMasterTest, DeactivateCallsMasterDeactivate) {
    this->initialize_with_slave();
    this->activate_with_domains();
    this->master_instance->deactivate();
    EXPECT_TRUE(this->mock_api->master_deactivate_called());
}

TEST_F(IGHMasterTest, SetSlaveEnabledDisable) {
    this->add_default_slave();
    ASSERT_NIL(this->master_instance->initialize());
    this->master_instance->set_slave_enabled(0, false);
    std::vector<pdo::Entry> entries = {{0, 0x6000, 0x01, 16, true, x::telem::UINT16_T}};
    ASSERT_OCCURRED_AS(
        this->master_instance->register_pdos(entries),
        errors::PDO_MAPPING_ERROR
    );
}

TEST_F(IGHMasterTest, SetSlaveEnabledReEnable) {
    this->add_default_slave();
    ASSERT_NIL(this->master_instance->initialize());
    this->master_instance->set_slave_enabled(0, false);
    this->master_instance->set_slave_enabled(0, true);
    auto *sc = this->master_instance->get_or_create_slave_config(0);
    EXPECT_NE(sc, nullptr);
}

}
