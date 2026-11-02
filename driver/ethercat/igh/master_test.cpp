// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/esi/known_devices.h"
#include "driver/ethercat/igh/master.h"
#include "driver/ethercat/igh/mock_api.h"

namespace driver::ethercat::igh {

class IGHMasterTest : public ::testing::Test {
protected:
    std::shared_ptr<MockAPI> mock_api;
    std::unique_ptr<Master> master_instance;
    x::breaker::Breaker brk;

    void SetUp() override {
        this->mock_api = std::make_shared<MockAPI>();
        this->master_instance = std::make_unique<Master>(this->mock_api, 0);
        this->brk.start();
    }

    void TearDown() override { this->brk.stop(); }

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

/// @brief it should return the device check error when the Driver cannot open the
/// device of the master.
TEST_F(IGHMasterTest, InitializeDeviceCheckFails) {
    this->add_default_slave();
    this->mock_api->set_check_device_err(
        x::errors::Error(
            errors::PERMISSION_DENIED,
            "permission denied opening /dev/EtherCAT0"
        )
    );
    ASSERT_OCCURRED_AS(this->master_instance->initialize(), errors::PERMISSION_DENIED);
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

/// @brief it should return the number of slaves on the bus.
TEST_F(IGHMasterTest, CountSlavesReportsTheSlavesOnTheBus) {
    this->initialize_with_slave();
    this->add_default_slave(1);
    EXPECT_EQ(ASSERT_NIL_P(this->master_instance->count_slaves()), 2);
}

/// @brief it should return a master init error when it counts slaves before
/// initialization.
TEST_F(IGHMasterTest, CountSlavesFailsBeforeInitialize) {
    ASSERT_OCCURRED_AS_P(
        this->master_instance->count_slaves(),
        errors::MASTER_INIT_ERROR
    );
}

/// @brief it should configure each enabled slave on activation.
TEST_F(IGHMasterTest, ActivateConfiguresEveryEnabledSlave) {
    this->mock_api->add_slave(0, 0x1111, 0x2222, 1, 1, "S0");
    this->mock_api->add_slave(1, 0x3333, 0x4444, 1, 2, "S1");
    ASSERT_NIL(this->master_instance->initialize());
    this->activate_with_domains();
    EXPECT_EQ(this->mock_api->slave_config_count(), 2u);
}

/// @brief it should not configure a disabled slave on activation.
TEST_F(IGHMasterTest, ActivateSkipsDisabledSlave) {
    this->mock_api->add_slave(0, 0x1111, 0x2222, 1, 1, "S0");
    this->mock_api->add_slave(1, 0x3333, 0x4444, 1, 2, "S1");
    ASSERT_NIL(this->master_instance->initialize());
    this->master_instance->set_slave_enabled(1, false);
    this->activate_with_domains();
    EXPECT_EQ(this->mock_api->slave_config_count(), 1u);
}

/// @brief it should return an activation error that names the slave when its
/// configuration fails.
TEST_F(IGHMasterTest, ActivateSlaveConfigFails) {
    this->initialize_with_slave();
    this->mock_api->set_slave_config_fails(true);
    const auto err = this->master_instance->activate();
    ASSERT_OCCURRED_AS(err, errors::ACTIVATION_ERROR);
    EXPECT_EQ(err.data, "failed to configure slave 0 (TestSlave)");
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

/// @brief it should return a working counter error when the domains report an
/// incomplete working counter.
TEST_F(IGHMasterTest, ReceiveWcIncompleteReturnsWorkingCounterError) {
    this->initialize_with_slave();
    this->activate_with_domains();
    ec_domain_state_t incomplete{};
    incomplete.wc_state = EC_WC_INCOMPLETE;
    incomplete.working_counter = 1;
    this->mock_api->set_output_domain_state(incomplete);
    this->mock_api->set_input_domain_state(incomplete);
    const auto err = this->master_instance->receive();
    ASSERT_OCCURRED_AS(err, errors::WORKING_COUNTER_ERROR);
    EXPECT_EQ(err.data, "incomplete working counter: output 1, input 1");
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

/// @brief it should not count the zero working counter of an empty output domain on a
/// bus with only input slaves.
TEST_F(IGHMasterTest, ReceiveIgnoresEmptyOutputDomain) {
    this->initialize_with_slave();
    this->activate_with_domains(0, 16);
    ec_domain_state_t zero_state{};
    zero_state.wc_state = EC_WC_ZERO;
    ec_domain_state_t complete{};
    complete.wc_state = EC_WC_COMPLETE;
    this->mock_api->set_output_domain_state(zero_state);
    this->mock_api->set_input_domain_state(complete);
    ASSERT_NIL(this->master_instance->receive());
}

/// @brief it should not count the zero working counter of an empty input domain on a
/// bus with only output slaves.
TEST_F(IGHMasterTest, ReceiveIgnoresEmptyInputDomain) {
    this->initialize_with_slave();
    this->activate_with_domains(16, 0);
    ec_domain_state_t zero_state{};
    zero_state.wc_state = EC_WC_ZERO;
    ec_domain_state_t complete{};
    complete.wc_state = EC_WC_COMPLETE;
    this->mock_api->set_output_domain_state(complete);
    this->mock_api->set_input_domain_state(zero_state);
    ASSERT_NIL(this->master_instance->receive());
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

/// @brief it should report the state of each configured slave.
TEST_F(IGHMasterTest, CheckReportsEachConfiguredSlave) {
    this->initialize_with_slave();
    this->activate_with_domains();
    ec_slave_config_state_t op{};
    op.al_state = 0x08;
    op.online = 1;
    op.operational = 1;
    this->mock_api->set_slave_config_state(0, op);
    const auto health = this->master_instance->check(this->brk);
    ASSERT_EQ(health.size(), 1u);
    EXPECT_EQ(health[0], (slave::Health{.position = 0, .state = slave::State::OP}));
}

/// @brief it should report a slave that is offline as lost.
TEST_F(IGHMasterTest, CheckReportsOfflineSlaveAsLost) {
    this->initialize_with_slave();
    this->activate_with_domains();
    ec_slave_config_state_t offline{};
    offline.al_state = 0x01;
    this->mock_api->set_slave_config_state(0, offline);
    const auto health = this->master_instance->check(this->brk);
    ASSERT_EQ(health.size(), 1u);
    EXPECT_TRUE(health[0].lost);
}

/// @brief it should report a slave with its error flag set as faulted.
TEST_F(IGHMasterTest, CheckReportsErrorFlagAsFaulted) {
    this->initialize_with_slave();
    this->activate_with_domains();
    ec_slave_config_state_t safe_op{};
    safe_op.al_state = 0x04;
    safe_op.online = 1;
    this->mock_api->set_slave_config_state(0, safe_op);
    this->mock_api->set_slave_error_flag(0, 1);
    const auto health = this->master_instance->check(this->brk);
    ASSERT_EQ(health.size(), 1u);
    EXPECT_EQ(health[0].state, slave::State::SAFE_OP);
    EXPECT_TRUE(health[0].faulted);
}

/// @brief it should leave a disabled slave out of the health report.
TEST_F(IGHMasterTest, CheckSkipsDisabledSlave) {
    this->mock_api->add_slave(0, 0x1111, 0x2222, 1, 1, "S0");
    this->mock_api->add_slave(1, 0x3333, 0x4444, 1, 2, "S1");
    ASSERT_NIL(this->master_instance->initialize());
    this->master_instance->set_slave_enabled(1, false);
    this->activate_with_domains();
    const auto health = this->master_instance->check(this->brk);
    ASSERT_EQ(health.size(), 1u);
    EXPECT_EQ(health[0].position, 0);
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

TEST_F(IGHMasterTest, SetSlaveEnabledReEnable) {
    this->add_default_slave();
    ASSERT_NIL(this->master_instance->initialize());
    this->master_instance->set_slave_enabled(0, false);
    this->master_instance->set_slave_enabled(0, true);
    this->activate_with_domains();
    EXPECT_EQ(this->mock_api->slave_config_count(), 1u);
}

/// @brief it should discover padding entries but not register them with the master.
TEST_F(IGHMasterTest, PaddingIsDiscoveredButNotRegistered) {
    this->mock_api->fake_syncs = {
        {EC_DIR_INPUT,
         {{0x1A00, {{0x6000, 0x01, 1}, {0x0000, 0x00, 7}, {0x6000, 0x11, 16}}}}},
    };
    this->initialize_with_slave();

    const auto inputs = this->master_instance->slaves()[0].properties.input_pdos;
    ASSERT_EQ(inputs.size(), 3);
    EXPECT_TRUE(inputs[1].is_padding());
    EXPECT_EQ(inputs[1].bit_length, 7);

    this->activate_with_domains();
    const std::vector<std::pair<uint16_t, uint8_t>> registered = {
        {0x6000, 0x01},
        {0x6000, 0x11},
    };
    EXPECT_EQ(this->mock_api->registered_entries, registered);
}

/// @brief it should fill the names and data types of live entries from the device
/// registry, so a rescan keeps the names that automatic channels refer to.
TEST_F(IGHMasterTest, RegistryFillsNamesAndDataTypesOfLiveEntries) {
    constexpr uint32_t beckhoff = 0x00000002;
    constexpr uint32_t el3102 = 0x0C1E3052;
    slave::Properties registry{};
    ASSERT_TRUE(esi::lookup_device_pdos(beckhoff, el3102, 1, registry));
    const auto known = std::ranges::find_if(registry.input_pdos, [](const auto &p) {
        return !p.is_padding() && p.bit_length == 16;
    });
    ASSERT_NE(known, registry.input_pdos.end());
    this->mock_api->fake_syncs = {
        {EC_DIR_INPUT, {{0x1A00, {{known->index, known->sub_index, 16}}}}},
    };
    this->add_default_slave(0, beckhoff, el3102);
    ASSERT_NIL(this->master_instance->initialize());

    const auto inputs = this->master_instance->slaves()[0].properties.input_pdos;
    ASSERT_EQ(inputs.size(), 1);
    EXPECT_EQ(inputs[0].name, known->name);
    EXPECT_EQ(inputs[0].data_type, known->data_type);
}

}
