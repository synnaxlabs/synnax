// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <map>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/loop/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/ethercat/coe/coe.h"
#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/esi/known_devices.h"
#include "driver/ethercat/esi/settings.h"
#include "driver/ethercat/igh/master.h"
#include "driver/ethercat/igh/mock_api.h"
#include "driver/ethercat/master/contract.h"
#include "driver/ethercat/telem/telem.h"
#include "driver/ethercat/testutil/stg.h"

namespace driver::ethercat::igh {
namespace {
/// @brief the bus cycle the specs activate the master with.
const auto PERIOD = x::telem::MILLISECOND;

using Registered = std::vector<std::pair<uint16_t, uint8_t>>;
using PDOs = std::vector<MockAPI::PDO>;

/// @brief assigns PDO 0x1A01 to the inputs in place of PDO 0x1A00.
const slave::StartupParameter ASSIGN_INPUTS{
    .index = 0x1C13,
    .sub_index = 0x01,
    .data_type = x::telem::UINT16_T,
    .value = 0x1A01,
};

const slave::StartupParameter FILTER{
    .index = 0x8000,
    .sub_index = 0x15,
    .data_type = x::telem::UINT16_T,
    .value = 3,
    .name = "Filter",
};

/// @brief makes slave map 0x7000:01 through PDO 0x1600 on its outputs and 0x6000:01
/// through PDO 0x1A00 on its inputs.
void create_io(MockAPI::Slave &slave) {
    slave.syncs = {
        {EC_DIR_OUTPUT, {}},
        {EC_DIR_INPUT, {}},
        {EC_DIR_OUTPUT, {{0x1600, {{0x7000, 0x01, 16}}}}},
        {EC_DIR_INPUT, {{0x1A00, {{0x6000, 0x01, 32}}}}},
    };
}

/// @brief makes slave map 0x6000:01 through PDO 0x1A00 on its inputs, and also offer
/// PDO 0x1A01, which maps 0x6010:01.
void create_assignable_layout(MockAPI::Slave &slave) {
    slave.syncs = {
        {EC_DIR_OUTPUT, {}},
        {EC_DIR_INPUT, {}},
        {EC_DIR_OUTPUT, {}},
        {EC_DIR_INPUT, {{0x1A00, {{0x6000, 0x01, 16}}}}},
    };
    slave.objects = {
        {{0x1C00, 0}, 4},
        {{0x1C00, 1}, 1},
        {{0x1C00, 2}, 2},
        {{0x1C00, 3}, 3},
        {{0x1C00, 4}, 4},
        {{0x1C12, 0}, 0},
        {{0x1C13, 0}, 1},
        {{0x1C13, 1}, 0x1A00},
        {{0x1A00, 0}, 1},
        {{0x1A00, 1}, 0x60000110},
        {{0x1A01, 0}, 1},
        {{0x1A01, 1}, 0x60100120},
    };
}

/// @brief returns the entry 0x6010:01 of the slave at position 1.
pdo::Entry assigned_entry() {
    return {
        .slave_position = 1,
        .index = 0x6010,
        .sub_index = 0x01,
        .bit_length = 32,
        .is_input = true,
    };
}
}

class IGHMasterTest : public ::testing::Test {
protected:
    std::shared_ptr<MockAPI> api = std::make_shared<MockAPI>();
    Master master{this->api, 0, {.clock = x::loop::fake_clock()}};
    x::breaker::Breaker brk;
    /// @brief the settings of each slave that activate() starts the master with.
    std::map<uint16_t, slave::Config> configs;
    /// @brief the process image of the latest activate().
    master::Image image;

    void SetUp() override { this->brk.start(); }

    void TearDown() override { this->brk.stop(); }

    /// @brief activates the master with configs, and keeps the image it gives.
    master::Activation activate() {
        return this->master.activate(
            {.period = PERIOD, .slaves = this->configs},
            [this](const master::Image &image) { this->image = image; }
        );
    }

    /// @brief scans the bus and activates the master with configs.
    x::errors::Error scan_and_activate() {
        if (auto err = this->master.scan().second) return err;
        return this->activate().err;
    }
};

/// @brief it should read the identity and name of a slave, and count positions from 1.
TEST_F(IGHMasterTest, ScanReadsEachSlave) {
    auto &first = this->api->add_slave("EL2008", 0x1234, 0x5678);
    first.info.revision_number = 3;
    first.info.serial_number = 42;
    this->api->add_slave("EL1008");
    const auto slaves = ASSERT_NIL_P(this->master.scan());
    ASSERT_EQ(slaves.size(), 2);
    const auto &props = slaves[0].properties;
    EXPECT_EQ(props.position, 1);
    EXPECT_EQ(props.vendor_id, 0x1234);
    EXPECT_EQ(props.product_code, 0x5678);
    EXPECT_EQ(props.revision, 3);
    EXPECT_EQ(props.serial, 42);
    EXPECT_EQ(props.name, "EL2008");
    EXPECT_EQ(slaves[1].properties.position, 2);
    EXPECT_EQ(slaves[1].properties.name, "EL1008");
}

/// @brief it should read the PDO layout of each slave from that slave.
TEST_F(IGHMasterTest, ScanReadsTheLayoutOfEachSlave) {
    this->api->add_slave().syncs = {{EC_DIR_INPUT, {{0x1A00, {{0x6000, 0x01, 16}}}}}};
    this->api->add_slave().syncs = {{EC_DIR_OUTPUT, {{0x1600, {{0x7000, 0x01, 8}}}}}};
    const auto slaves = ASSERT_NIL_P(this->master.scan());
    ASSERT_EQ(slaves[0].properties.input_pdos.size(), 1);
    EXPECT_EQ(slaves[0].properties.input_pdos[0].index, 0x6000);
    EXPECT_TRUE(slaves[0].properties.output_pdos.empty());
    EXPECT_TRUE(slaves[1].properties.input_pdos.empty());
    ASSERT_EQ(slaves[1].properties.output_pdos.size(), 1);
    EXPECT_EQ(slaves[1].properties.output_pdos[0].index, 0x7000);
}

/// @brief it should return the slaves it found without a new discovery.
TEST_F(IGHMasterTest, ScanReturnsTheSlavesItFound) {
    this->api->add_slave();
    ASSERT_NIL_P(this->master.scan());
    this->api->add_slave();
    EXPECT_EQ(ASSERT_NIL_P(this->master.scan()).size(), 1);
}

/// @brief it should name the network of each slave after the master index.
TEST_F(IGHMasterTest, ScanNamesTheNetworkAfterTheMasterIndex) {
    this->api->add_slave();
    Master master(this->api, 3);
    const auto slaves = ASSERT_NIL_P(master.scan());
    ASSERT_EQ(slaves.size(), 1);
    EXPECT_EQ(slaves[0].properties.network, "igh:3");
}

/// @brief it should return the device check error when the Driver cannot open the
/// device of the master.
TEST_F(IGHMasterTest, ScanFailsWhenTheDeviceCheckFails) {
    this->api->check_device_err = x::errors::Error(
        errors::PERMISSION_DENIED,
        "permission denied opening /dev/EtherCAT0"
    );
    ASSERT_OCCURRED_AS_P(this->master.scan(), errors::PERMISSION_DENIED);
}

/// @brief it should fail when the kernel module gives no master.
TEST_F(IGHMasterTest, ScanFailsWhenTheMasterIsMissing) {
    this->api->request_master_fails = true;
    const auto err = this->master.scan().second;
    ASSERT_OCCURRED_AS(err, errors::MASTER_INIT_ERROR);
    EXPECT_EQ(err.data, "IgH master not available - is kernel module loaded?");
}

/// @brief it should release the master when it cannot read the master info.
TEST_F(IGHMasterTest, ScanFailsWhenTheMasterInfoCannotBeRead) {
    this->api->master_info_result = -1;
    const auto err = this->master.scan().second;
    ASSERT_OCCURRED_AS(err, errors::MASTER_INIT_ERROR);
    EXPECT_EQ(err.data, "failed to get master info");
    EXPECT_TRUE(this->api->released);
}

/// @brief it should release the master when it cannot create the output domain.
TEST_F(IGHMasterTest, ScanFailsWhenTheOutputDomainFails) {
    this->api->add_slave();
    this->api->output_domain_fails = true;
    const auto err = this->master.scan().second;
    ASSERT_OCCURRED_AS(err, errors::MASTER_INIT_ERROR);
    EXPECT_EQ(err.data, "failed to create output domain");
    EXPECT_TRUE(this->api->released);
}

/// @brief it should release the master when it cannot create the input domain.
TEST_F(IGHMasterTest, ScanFailsWhenTheInputDomainFails) {
    this->api->add_slave();
    this->api->input_domain_fails = true;
    const auto err = this->master.scan().second;
    ASSERT_OCCURRED_AS(err, errors::MASTER_INIT_ERROR);
    EXPECT_EQ(err.data, "failed to create input domain");
    EXPECT_TRUE(this->api->released);
}

/// @brief it should fail to initialize, and release the master, when it cannot read a
/// slave.
TEST_F(IGHMasterTest, ScanFailsWhenASlaveCannotBeRead) {
    this->api->add_slave();
    this->api->add_slave().unreadable = true;
    const auto err = this->master.scan().second;
    ASSERT_OCCURRED_AS(err, errors::MASTER_INIT_ERROR);
    EXPECT_EQ(err.data, "failed to read slave 2: Input/output error");
    EXPECT_TRUE(this->api->released);
}

/// @brief it should report why it failed to read the layout of a slave, and give the
/// slave no PDOs.
TEST_F(IGHMasterTest, ScanReportsAFailedLayoutRead) {
    auto &slave = this->api->add_slave();
    create_io(slave);
    slave.layout_unreadable = true;
    const auto slaves = ASSERT_NIL_P(this->master.scan());
    ASSERT_EQ(slaves.size(), 1);
    EXPECT_TRUE(slaves[0].properties.input_pdos.empty());
    EXPECT_TRUE(slaves[0].properties.output_pdos.empty());
    EXPECT_EQ(
        slaves[0].status.pdo_discovery_error,
        "failed to read the PDO layout: failed to read PDO 0 of sync manager 2: "
        "Input/output error"
    );
}

/// @brief it should discover padding entries but not register them with the master.
TEST_F(IGHMasterTest, ScanDiscoversPaddingThatActivateDoesNotRegister) {
    auto &slave = this->api->add_slave();
    slave.syncs = {
        {EC_DIR_INPUT,
         {{0x1A00, {{0x6000, 0x01, 1}, {0x0000, 0x00, 7}, {0x6000, 0x11, 16}}}}},
    };
    const auto slaves = ASSERT_NIL_P(this->master.scan());
    const auto &inputs = slaves[0].properties.input_pdos;
    ASSERT_EQ(inputs.size(), 3);
    EXPECT_TRUE(inputs[1].is_padding());
    EXPECT_EQ(inputs[1].bit_length, 7);
    ASSERT_NIL(this->activate().err);
    EXPECT_EQ(slave.registered, (Registered{{0x6000, 0x01}, {0x6000, 0x11}}));
}

/// @brief it should fill the names and data types of live entries from the device
/// registry, so a rescan keeps the names that automatic channels refer to.
TEST_F(IGHMasterTest, ScanFillsNamesAndDataTypesFromTheRegistry) {
    constexpr uint32_t beckhoff = 0x00000002;
    constexpr uint32_t el3102 = 0x0C1E3052;
    slave::Properties registry{};
    ASSERT_TRUE(esi::lookup_device_pdos(beckhoff, el3102, 1, registry));
    const auto known = std::ranges::find_if(registry.input_pdos, [](const auto &p) {
        return !p.is_padding() && p.bit_length == 16;
    });
    ASSERT_NE(known, registry.input_pdos.end());
    this->api->add_slave("EL3102", beckhoff, el3102).syncs = {
        {EC_DIR_INPUT, {{0x1A00, {{known->index, known->sub_index, 16}}}}},
    };
    const auto slaves = ASSERT_NIL_P(this->master.scan());
    const auto &inputs = slaves[0].properties.input_pdos;
    ASSERT_EQ(inputs.size(), 1);
    EXPECT_EQ(inputs[0].name, known->name);
    EXPECT_EQ(inputs[0].data_type, known->data_type);
}

/// @brief it should store the settings that the catalog describes for a slave, read
/// from that slave's own object dictionary.
TEST_F(IGHMasterTest, ScanStoresTheSettingsThatTheCatalogDescribes) {
    this->api->add_slave();
    this->api->add_slave("6xSTG", testutil::DEWESOFT, testutil::IOLITE_6XSTG)
        .objects = testutil::stg_objects(0);
    const auto slaves = ASSERT_NIL_P(this->master.scan());
    const auto expected = ASSERT_NIL_P(
        esi::read_settings(
            testutil::DEWESOFT,
            testutil::IOLITE_6XSTG,
            slaves[1].properties.revision,
            slaves[1].properties.op_modes,
            testutil::read_of(testutil::stg_objects(0))
        )
    );
    ASSERT_TRUE(expected.has_value());
    EXPECT_FALSE(slaves[0].properties.settings.has_value());
    EXPECT_EQ(slaves[1].properties.settings, expected);
}

/// @brief it should store the settings that the catalog describes for the revision of
/// the slave.
TEST_F(IGHMasterTest, ScanStoresTheSettingsOfTheSlaveRevision) {
    auto &slave = this->api
                      ->add_slave("32xDO", testutil::DEWESOFT, testutil::IOLITE_DIO);
    slave.info.revision_number = 0x00050000;
    slave.objects = {{{0x8020, 1}, 0}, {{0x8020, 9}, 0}};
    const auto slaves = ASSERT_NIL_P(this->master.scan());
    ASSERT_TRUE(slaves[0].properties.settings.has_value());
    const auto &device = slaves[0].properties.settings.value().at("device");
    ASSERT_EQ(device.size(), 1);
    EXPECT_EQ(device[0]["name"], "Outputs on connection loss");
}

/// @brief it should report why a setting of a slave could not be read, and keep the
/// other settings.
TEST_F(IGHMasterTest, ScanReportsAFailedSettingsRead) {
    auto objects = testutil::stg_objects(0);
    objects.erase({0x8050, 3});
    this->api->add_slave("6xSTG", testutil::DEWESOFT, testutil::IOLITE_6XSTG)
        .objects = objects;
    const auto slaves = ASSERT_NIL_P(this->master.scan());
    ASSERT_TRUE(slaves[0].properties.settings.has_value());
    EXPECT_TRUE(slaves[0].properties.settings.value().contains("mode"));
    EXPECT_EQ(
        slaves[0].status.settings_error,
        "failed to read 0x8050:03: Object does not exist in the object dictionary"
    );
}

/// @brief it should report a change once IgH finds another model at a position.
TEST_F(IGHMasterTest, ChangedOnceAnotherModelIsAtAPosition) {
    this->api->add_slave("EL3102", 0x2, 0x0C1E3052);
    ASSERT_NIL_P(this->master.scan());
    EXPECT_FALSE(ASSERT_NIL_P(this->master.changed()).stale);
    this->api->slaves[0].info.product_code = 0x03F03052;
    const auto change = ASSERT_NIL_P(this->master.changed());
    EXPECT_TRUE(change.stale);
    EXPECT_TRUE(change.restarted.empty());
    EXPECT_FALSE(this->master.restarts_visible());
}

/// @brief it should fail to compare the bus with the scan when IgH cannot read a slave.
TEST_F(IGHMasterTest, ChangedFailsWhenASlaveIsUnreadable) {
    this->api->add_slave();
    ASSERT_NIL_P(this->master.scan());
    this->api->slaves[0].unreadable = true;
    ASSERT_OCCURRED_AS_P(this->master.changed(), errors::MASTER_INIT_ERROR);
}

/// @brief it should fail activation when the IgH master does not activate.
TEST_F(IGHMasterTest, ActivateFailsWhenTheMasterDoesNotActivate) {
    this->api->add_slave();
    this->api->activate_result = -1;
    const auto err = this->scan_and_activate();
    ASSERT_OCCURRED_AS(err, errors::ACTIVATION_ERROR);
    EXPECT_EQ(err.data, "ecrt_master_activate failed");
}

/// @brief it should configure each enabled slave and skip a disabled one.
TEST_F(IGHMasterTest, ActivateConfiguresEachEnabledSlave) {
    this->api->add_slave();
    this->api->add_slave();
    this->api->add_slave();
    this->configs = {{2, {.enabled = false}}};
    ASSERT_NIL(this->scan_and_activate());
    EXPECT_TRUE(this->api->slaves[0].configured);
    EXPECT_FALSE(this->api->slaves[1].configured);
    EXPECT_TRUE(this->api->slaves[2].configured);
}

/// @brief it should return an activation error that names the slave when its
/// configuration fails.
TEST_F(IGHMasterTest, ActivateFailsWhenASlaveCannotBeConfigured) {
    this->api->add_slave();
    this->api->slave_config_fails = true;
    const auto err = this->scan_and_activate();
    ASSERT_OCCURRED_AS(err, errors::ACTIVATION_ERROR);
    EXPECT_EQ(err.data, "failed to configure slave 1 (TestSlave)");
}

/// @brief it should give the cycle the domains, with the offset of each entry.
TEST_F(IGHMasterTest, ActivateGivesTheProcessImage) {
    create_io(this->api->add_slave());
    ASSERT_NIL(this->scan_and_activate());
    EXPECT_EQ(this->image.outputs.size(), 2);
    EXPECT_EQ(this->image.inputs.size(), 4);
    ASSERT_NIL_P(this->image.offset(
        {.slave_position = 1,
         .index = 0x7000,
         .sub_index = 0x01,
         .bit_length = 16,
         .is_input = false}
    ));
    ASSERT_NIL_P(this->image.offset(
        {.slave_position = 1,
         .index = 0x6000,
         .sub_index = 0x01,
         .bit_length = 32,
         .is_input = true}
    ));
}

/// @brief it should start a bus whose slaves have no process data.
TEST_F(IGHMasterTest, ActivateStartsABusWithNoProcessData) {
    this->api->add_slave();
    ASSERT_NIL(this->scan_and_activate());
    EXPECT_TRUE(this->image.inputs.empty());
    EXPECT_TRUE(this->image.outputs.empty());
}

/// @brief it should start the bus cycle once send() works.
TEST_F(IGHMasterTest, ActivateStartsTheCycleOnceSendWorks) {
    this->api->add_slave();
    ASSERT_NIL_P(this->master.scan());
    bool started = false;
    ASSERT_NIL(this->master
                   .activate(
                       {.period = PERIOD},
                       [&](const master::Image &) {
                           ASSERT_NIL(this->master.send());
                           started = true;
                       }
                   )
                   .err);
    EXPECT_TRUE(started);
}

/// @brief it should return once every enabled slave reaches OP.
TEST_F(IGHMasterTest, ActivateWaitsForEverySlaveToReachOP) {
    this->api->add_slave();
    this->api->add_slave().reads_to_reach = 5;
    ASSERT_NIL(this->scan_and_activate());
    const auto health = this->master.check(this->brk);
    ASSERT_EQ(health.size(), 2);
    EXPECT_EQ(health[1], (slave::Health{.position = 2, .state = slave::State::OP}));
}

/// @brief it should return each slave that has its AL error flag set below OP as a
/// refusal of its next state.
TEST_F(IGHMasterTest, ActivateReturnsTheRefusalOfEachSlave) {
    this->api->add_slave("EL3102");
    auto &pre_op = this->api->add_slave("EL4002");
    pre_op.reaches = EC_AL_STATE_PREOP;
    pre_op.refuses = true;
    auto &safe_op = this->api->add_slave("EL1008");
    safe_op.reaches = EC_AL_STATE_SAFEOP;
    safe_op.refuses = true;
    ASSERT_NIL_P(this->master.scan());
    const auto activation = this->activate();
    ASSERT_OCCURRED_AS(activation.err, errors::STATE_CHANGE_REFUSED);
    EXPECT_EQ(
        activation.err.data,
        "failed to reach OP within 10s: slave 2 (EL4002) is in PRE-OP with an AL "
        "error; slave 3 (EL1008) is in SAFE-OP with an AL error"
    );
    const std::map<uint16_t, slave::Refusal> refusals{
        {2, {.reason = "refused SAFE-OP"}},
        {3, {.reason = "refused OP"}},
    };
    EXPECT_EQ(activation.refusals, refusals);
}

/// @brief it should fail without refusals when a slave stays below OP with no AL
/// error.
TEST_F(IGHMasterTest, ActivateFailsWhenASlaveStaysBelowOP) {
    this->api->add_slave().reaches = EC_AL_STATE_SAFEOP;
    ASSERT_NIL_P(this->master.scan());
    const auto activation = this->activate();
    ASSERT_OCCURRED_AS(activation.err, errors::STATE_CHANGE_ERROR);
    EXPECT_EQ(
        activation.err.data,
        "failed to reach OP within 10s: slave 1 (TestSlave) is in SAFE-OP"
    );
    EXPECT_TRUE(activation.refusals.empty());
}

/// @brief it should start when a slave whose configuration state it cannot read is in
/// OP.
TEST_F(IGHMasterTest, ActivateStartsWhenAnUnreadableSlaveIsInOP) {
    this->api->add_slave().state_unreadable = true;
    ASSERT_NIL_P(this->master.scan());
    this->api->slaves[0].info.al_state = EC_AL_STATE_OP;
    const auto activation = this->activate();
    ASSERT_NIL(activation.err);
    EXPECT_TRUE(activation.refusals.empty());
}

/// @brief it should name the slave whose state it failed to read when a start times
/// out.
TEST_F(IGHMasterTest, ActivateReportsASlaveWhoseStateItCannotRead) {
    this->api->add_slave().reaches = EC_AL_STATE_SAFEOP;
    ASSERT_NIL_P(this->master.scan());
    this->api->slaves[0].unreadable = true;
    const auto activation = this->activate();
    ASSERT_OCCURRED_AS(activation.err, errors::STATE_CHANGE_ERROR);
    EXPECT_EQ(
        activation.err.data,
        "failed to reach OP within 10s: slave 1 (TestSlave): failed to read its "
        "state: Input/output error"
    );
}

/// @brief it should fail activation on a chosen DC mode before it configures any slave,
/// as the IgH master does not run DC sync.
TEST_F(IGHMasterTest, ActivateFailsOnAChosenDCMode) {
    this->api->add_slave();
    this->api->add_slave();
    this->configs = {
        {2, {.op_mode = dc::Choice{.index = 1, .assign_activate = 0x0700}}}
    };
    const auto err = this->scan_and_activate();
    ASSERT_OCCURRED_AS(err, errors::SLAVE_CONFIG_ERROR);
    EXPECT_EQ(
        err.data,
        "slave 2 (TestSlave) cannot use its chosen mode: the IgH master does not run "
        "DC sync"
    );
    EXPECT_FALSE(this->api->slaves[0].configured);
}

/// @brief it should fail activation on a chosen cycle for an SM mode, as the SOEM
/// master does.
TEST_F(IGHMasterTest, ActivateFailsOnAChosenCycleForAnSMMode) {
    this->api->add_slave();
    this->configs = {{
        1,
        {.op_mode = dc::Choice{
             .index = 0,
             .assign_activate = 0,
             .cycles = {.sync0 = x::telem::MILLISECOND},
         }},
    }};
    const auto err = this->scan_and_activate();
    ASSERT_OCCURRED_AS(err, errors::SLAVE_CONFIG_ERROR);
    EXPECT_EQ(
        err.data,
        "slave 1 (TestSlave) cannot use its chosen mode: mode (AssignActivate 0x0000) "
        "does not run SYNC0, so it cannot use a SYNC0 cycle of 1ms"
    );
    EXPECT_FALSE(this->api->slaves[0].configured);
}

/// @brief it should run a slave in a chosen SM mode.
TEST_F(IGHMasterTest, ActivateRunsAChosenSMMode) {
    this->api->add_slave();
    this->configs = {{1, {.op_mode = dc::Choice{.index = 0, .assign_activate = 0}}}};
    ASSERT_NIL(this->scan_and_activate());
    EXPECT_TRUE(this->api->slaves[0].configured);
}

/// @brief it should start the bus when the master rejects the entries of a slave whose
/// layout comes from the device registry, and return the reason only for those
/// entries.
TEST_F(IGHMasterTest, ActivateReturnsWhyTheMasterRejectedAnEntry) {
    constexpr uint32_t beckhoff = 0x00000002;
    constexpr uint32_t el3102 = 0x0C1E3052;
    this->api->add_slave("EL3102", beckhoff, el3102);
    create_io(this->api->add_slave());
    ASSERT_NIL(this->scan_and_activate());
    const auto &registry = this->image.slaves[0].properties.input_pdos;
    const auto entry = std::ranges::find_if(registry, [](const auto &p) {
        return !p.is_padding();
    });
    ASSERT_NE(entry, registry.end());
    const auto rejected = this->image.offset(
        pdo::Entry(1, entry->index, entry->sub_index, entry->bit_length, true, {})
    );
    ASSERT_OCCURRED_AS_P(rejected, errors::PDO_MAPPING_ERROR);
    EXPECT_EQ(
        rejected.second.data,
        "the IgH master rejected PDO " +
            telem::format_index_sub_index(entry->index, entry->sub_index) +
            " of slave 1 (EL3102): No such file or directory"
    );
    ASSERT_NIL_P(this->image.offset(pdo::Entry(2, 0x6000, 1, 32, true, {})));
}

/// @brief it should write each startup parameter on activation and also add it to the
/// config of the slave.
TEST_F(IGHMasterTest, ActivateWritesAndConfiguresStartupParameters) {
    this->api->add_slave();
    this->configs = {{1, {.startup_parameters = {FILTER}}}};
    ASSERT_NIL(this->scan_and_activate());
    const std::vector<MockAPI::SDO> expected{{0x8000, 0x15, {0x03, 0x00}}};
    EXPECT_EQ(this->api->slaves[0].downloads, expected);
    EXPECT_EQ(this->api->slaves[0].startup_sdos, expected);
}

/// @brief it should fail activation with a startup parameter error that names the
/// parameter and the abort reason.
TEST_F(IGHMasterTest, ActivateFailsWhenASlaveRejectsAStartupParameter) {
    this->api->add_slave().download_aborts[{0x8000, 0x15}] = 0x06090031;
    this->configs = {{1, {.startup_parameters = {FILTER}}}};
    const auto err = this->scan_and_activate();
    ASSERT_OCCURRED_AS(err, errors::STARTUP_PARAMETER_ERROR);
    EXPECT_EQ(
        err.data,
        "slave 1 (TestSlave) rejected startup parameter 0x8000:15 (Filter): Value of "
        "parameter written too high"
    );
    EXPECT_TRUE(this->api->slaves[0].startup_sdos.empty());
}

/// @brief it should report no response when a slave does not answer a startup
/// parameter write.
TEST_F(IGHMasterTest, ActivateFailsWhenASlaveDoesNotAnswerAStartupParameter) {
    this->api->add_slave().download_aborts[{0x8000, 0x15}] = 0;
    this->configs = {{1, {.startup_parameters = {FILTER}}}};
    const auto err = this->scan_and_activate();
    ASSERT_OCCURRED_AS(err, errors::STARTUP_PARAMETER_ERROR);
    EXPECT_EQ(
        err.data,
        "slave 1 (TestSlave) rejected startup parameter 0x8000:15 (Filter): no response"
    );
}

/// @brief it should fail activation when the master cannot add a startup parameter to
/// the config of the slave.
TEST_F(IGHMasterTest, ActivateFailsWhenAStartupParameterCannotBeConfigured) {
    this->api->add_slave();
    this->api->config_sdo_fails = true;
    this->configs = {{1, {.startup_parameters = {FILTER}}}};
    const auto err = this->scan_and_activate();
    ASSERT_OCCURRED_AS(err, errors::ACTIVATION_ERROR);
    EXPECT_EQ(
        err.data,
        "failed to add startup parameter 0x8000:15 (Filter) to slave 1 (TestSlave)"
    );
}

/// @brief it should not write the startup parameters of a disabled slave.
TEST_F(IGHMasterTest, ActivateSkipsTheStartupParametersOfADisabledSlave) {
    this->api->add_slave();
    this->configs = {{1, {.enabled = false, .startup_parameters = {FILTER}}}};
    ASSERT_NIL(this->scan_and_activate());
    EXPECT_TRUE(this->api->slaves[0].downloads.empty());
}

/// @brief it should write each parameter to a slave before activation.
TEST_F(IGHMasterTest, WriteDownloadsEachParameterBeforeActivation) {
    this->api->add_slave();
    ASSERT_NIL_P(this->master.scan());
    ASSERT_NIL(this->master.write(1, {FILTER}));
    const std::vector<MockAPI::SDO> expected{{0x8000, 0x15, {0x03, 0x00}}};
    EXPECT_EQ(this->api->slaves[0].downloads, expected);
    EXPECT_TRUE(this->api->slaves[0].startup_sdos.empty());
}

/// @brief it should return the reason a slave rejects a parameter before activation.
TEST_F(IGHMasterTest, WriteReturnsWhyASlaveRejectedAParameter) {
    this->api->add_slave().download_aborts[{0x8000, 0x15}] = 0x06090031;
    ASSERT_NIL_P(this->master.scan());
    const auto err = this->master.write(1, {FILTER});
    ASSERT_OCCURRED_AS(err, errors::STARTUP_PARAMETER_ERROR);
    EXPECT_EQ(
        err.data,
        "slave 1 (TestSlave) rejected startup parameter 0x8000:15 (Filter): Value of "
        "parameter written too high"
    );
}

/// @brief it should name the master by its index.
TEST_F(IGHMasterTest, NameIsTheIndexOfTheMaster) {
    EXPECT_EQ(this->master.name(), "igh:0");
}

/// @brief it should read the value that a slave holds in an object, counting positions
/// from 1.
TEST_F(IGHMasterTest, ReadReturnsTheValueOfAnObject) {
    this->api->add_slave();
    this->api->add_slave().objects[{0x8040, 0x15}] = 0x40000000;
    ASSERT_NIL(this->scan_and_activate());
    EXPECT_EQ(ASSERT_NIL_P(this->master.read(2, 0x8040, 0x15)), 0x40000000);
}

/// @brief it should return the abort of a read of an object that the slave does not
/// have.
TEST_F(IGHMasterTest, ReadReturnsTheAbortOfAMissingObject) {
    this->api->add_slave();
    ASSERT_NIL(this->scan_and_activate());
    ASSERT_OCCURRED_AS_P(this->master.read(1, 0x8040, 0x15), coe::OBJECT_MISSING);
}

/// @brief it should accept a value that the slave holds on the running bus, and write
/// nothing.
TEST_F(IGHMasterTest, WriteAcceptsAValueThatTheSlaveHolds) {
    this->api->add_slave().objects[{0x8000, 0x15}] = 3;
    ASSERT_NIL(this->scan_and_activate());
    ASSERT_NIL(this->master.write(1, {FILTER}));
    EXPECT_TRUE(this->api->slaves[0].downloads.empty());
}

/// @brief it should refuse a value that the slave does not hold on the running bus,
/// since IgH writes its startup parameters again each time the slave restarts.
TEST_F(IGHMasterTest, WriteRefusesAValueThatTheSlaveDoesNotHold) {
    this->api->add_slave();
    this->configs = {{1, {.startup_parameters = {FILTER}}}};
    ASSERT_NIL(this->scan_and_activate());
    auto changed = FILTER;
    changed.value = 4;
    const auto err = this->master.write(1, {changed});
    ASSERT_OCCURRED_AS(err, errors::SETTING_CONFLICT);
    EXPECT_EQ(
        err.data,
        "the IgH master cannot change 0x8000:15 (Filter) of slave 1 (TestSlave) while "
        "other tasks run the bus. Stop them, then start this task first"
    );
    const std::vector<MockAPI::SDO> expected{{0x8000, 0x15, {0x03, 0x00}}};
    EXPECT_EQ(this->api->slaves[0].downloads, expected);
}

/// @brief it should return the reason a slave on the running bus fails to read a
/// parameter.
TEST_F(IGHMasterTest, WriteReturnsWhyASlaveFailedToReadAParameter) {
    this->api->add_slave();
    ASSERT_NIL(this->scan_and_activate());
    const auto err = this->master.write(1, {FILTER});
    ASSERT_OCCURRED_AS(err, errors::STARTUP_PARAMETER_ERROR);
    EXPECT_EQ(
        err.data,
        "slave 1 (TestSlave) failed to read startup parameter 0x8000:15 (Filter): "
        "Object does not exist in the object dictionary"
    );
}

/// @brief it should refuse to write to a position with no slave on the bus.
TEST_F(IGHMasterTest, WriteRejectsAPositionWithNoSlave) {
    this->api->add_slave();
    ASSERT_NIL(this->scan_and_activate());
    const auto err = this->master.write(2, {FILTER});
    ASSERT_OCCURRED_AS(err, errors::STARTUP_PARAMETER_ERROR);
    EXPECT_EQ(err.data, "no slave at position 2");
}

/// @brief it should register the entries of the PDO a startup parameter assigns, and
/// configure that layout on the slave.
TEST_F(IGHMasterTest, ActivateRegistersTheEntriesAStartupParameterAssigns) {
    auto &slave = this->api->add_slave();
    create_assignable_layout(slave);
    this->configs = {{1, {.startup_parameters = {ASSIGN_INPUTS}}}};
    ASSERT_NIL(this->scan_and_activate());
    EXPECT_EQ(slave.registered, (Registered{{0x6010, 0x01}}));
    EXPECT_EQ(slave.assigned.at(3), (PDOs{{0x1A01, {{0x6010, 0x01, 32}}}}));
    const auto &inputs = this->image.slaves[0].properties.input_pdos;
    ASSERT_EQ(inputs.size(), 1);
    EXPECT_EQ(inputs[0].index, 0x6010);
    EXPECT_EQ(inputs[0].bit_length, 32);
    ASSERT_NIL_P(this->image.offset(assigned_entry()));
}

/// @brief it should read the layout of each slave from that slave's own object
/// dictionary.
TEST_F(IGHMasterTest, ActivateReadsTheLayoutOfEachSlaveFromItsOwnDictionary) {
    create_assignable_layout(this->api->add_slave());
    create_assignable_layout(this->api->add_slave());
    this->configs = {
        {1, {.startup_parameters = {ASSIGN_INPUTS}}},
        {2, {.startup_parameters = {FILTER}}},
    };
    ASSERT_NIL(this->scan_and_activate());
    EXPECT_EQ(
        this->api->slaves[0].assigned.at(3),
        (PDOs{{0x1A01, {{0x6010, 0x01, 32}}}})
    );
    EXPECT_EQ(
        this->api->slaves[1].assigned.at(3),
        (PDOs{{0x1A00, {{0x6000, 0x01, 16}}}})
    );
}

/// @brief it should keep the scanned layout of a slave that has no sync manager type
/// object.
TEST_F(IGHMasterTest, ActivateKeepsTheScannedLayoutWithoutSyncManagerTypes) {
    auto &slave = this->api->add_slave();
    create_assignable_layout(slave);
    slave.objects.erase({0x1C00, 0});
    this->configs = {{1, {.startup_parameters = {ASSIGN_INPUTS}}}};
    ASSERT_NIL(this->scan_and_activate());
    EXPECT_EQ(slave.registered, (Registered{{0x6000, 0x01}}));
    EXPECT_TRUE(slave.assigned.empty());
}

/// @brief it should clear the assignment of a sync manager that assigns no PDOs.
TEST_F(IGHMasterTest, ActivateClearsAnEmptyAssignment) {
    auto &slave = this->api->add_slave();
    create_assignable_layout(slave);
    slave.syncs[2].pdos = {{0x1600, {{0x7000, 0x01, 8}}}};
    this->configs = {{1, {.startup_parameters = {ASSIGN_INPUTS}}}};
    ASSERT_NIL(this->scan_and_activate());
    EXPECT_TRUE(slave.assigned.at(2).empty());
    EXPECT_EQ(slave.registered, (Registered{{0x6010, 0x01}}));
}

/// @brief it should clear the mapping of an assigned PDO that maps no entries.
TEST_F(IGHMasterTest, ActivateClearsAnEmptyMapping) {
    auto &slave = this->api->add_slave();
    create_assignable_layout(slave);
    slave.objects[{0x1A00, 0}] = 0;
    this->configs = {{1, {.startup_parameters = {FILTER}}}};
    ASSERT_NIL(this->scan_and_activate());
    EXPECT_EQ(slave.assigned.at(3), (PDOs{{0x1A00, {}}}));
    EXPECT_TRUE(slave.registered.empty());
    EXPECT_TRUE(this->image.slaves[0].properties.input_pdos.empty());
}

/// @brief it should keep the scanned PDOs of a sync manager without an assign object.
TEST_F(IGHMasterTest, ActivateKeepsTheScannedPDOsWithoutAnAssignObject) {
    auto &slave = this->api->add_slave();
    create_assignable_layout(slave);
    slave.syncs[2].pdos = {{0x1600, {{0x7000, 0x01, 8}}}};
    slave.objects.erase({0x1C12, 0});
    this->configs = {{1, {.startup_parameters = {ASSIGN_INPUTS}}}};
    ASSERT_NIL(this->scan_and_activate());
    EXPECT_EQ(slave.registered, (Registered{{0x7000, 0x01}, {0x6010, 0x01}}));
    EXPECT_FALSE(slave.assigned.contains(2));
}

/// @brief it should keep the scanned PDOs of a sync manager without an assign object
/// when it reads another sync manager of the same direction.
TEST_F(IGHMasterTest, ActivateKeepsTheScannedPDOsOfAFixedSyncManager) {
    auto &slave = this->api->add_slave();
    create_assignable_layout(slave);
    slave.syncs.push_back({EC_DIR_OUTPUT, {{0x1600, {{0x7000, 0x01, 8}}}}});
    slave.objects[{0x1C00, 0}] = 5;
    slave.objects[{0x1C00, 5}] = 3;
    slave.objects[{0x1C12, 0}] = 1;
    slave.objects[{0x1C12, 1}] = 0x1601;
    slave.objects[{0x1601, 0}] = 1;
    slave.objects[{0x1601, 1}] = 0x70100108;
    this->configs = {{1, {.startup_parameters = {ASSIGN_INPUTS}}}};
    ASSERT_NIL(this->scan_and_activate());
    EXPECT_EQ(
        slave.registered,
        (Registered{{0x7010, 0x01}, {0x7000, 0x01}, {0x6010, 0x01}})
    );
    EXPECT_EQ(slave.assigned.at(2), (PDOs{{0x1601, {{0x7010, 0x01, 8}}}}));
    EXPECT_FALSE(slave.assigned.contains(4));
}

/// @brief it should fail activation when it cannot read the layout after the startup
/// parameters.
TEST_F(IGHMasterTest, ActivateFailsWhenTheLayoutCannotBeRead) {
    auto &slave = this->api->add_slave();
    create_assignable_layout(slave);
    slave.objects.erase({0x1A01, 0});
    this->configs = {{1, {.startup_parameters = {ASSIGN_INPUTS}}}};
    const auto err = this->scan_and_activate();
    ASSERT_OCCURRED_AS(err, errors::PDO_DISCOVERY_ERROR);
    EXPECT_EQ(
        err.data,
        "failed to read the PDO layout of slave 1 (TestSlave) after its startup "
        "parameters: failed to read 0x1A01:00: Object does not exist in the "
        "object dictionary"
    );
    EXPECT_TRUE(slave.registered.empty());
}

/// @brief it should receive when both domains report a complete working counter.
TEST_F(IGHMasterTest, ReceiveSucceedsOnACompleteWorkingCounter) {
    create_io(this->api->add_slave());
    ASSERT_NIL(this->scan_and_activate());
    ASSERT_NIL(this->master.receive());
}

/// @brief it should fail when no slave answers a domain.
TEST_F(IGHMasterTest, ReceiveFailsWhenNoSlaveAnswers) {
    create_io(this->api->add_slave());
    ASSERT_NIL(this->scan_and_activate());
    this->api->outputs.state.wc_state = EC_WC_ZERO;
    const auto err = this->master.receive();
    ASSERT_OCCURRED_AS(err, errors::WORKING_COUNTER_ERROR);
    EXPECT_EQ(err.data, "no slaves responded");
}

/// @brief it should return a working counter error when a domain reports an incomplete
/// working counter.
TEST_F(IGHMasterTest, ReceiveFailsOnAnIncompleteWorkingCounter) {
    create_io(this->api->add_slave());
    ASSERT_NIL(this->scan_and_activate());
    this->api->outputs.state = {.working_counter = 1, .wc_state = EC_WC_INCOMPLETE};
    this->api->inputs.state = {.working_counter = 2, .wc_state = EC_WC_COMPLETE};
    const auto err = this->master.receive();
    ASSERT_OCCURRED_AS(err, errors::WORKING_COUNTER_ERROR);
    EXPECT_EQ(err.data, "incomplete working counter: output 1, input 2");
}

/// @brief it should report no shortfall before a complete cycle gives the full working
/// counter.
TEST_F(IGHMasterTest, ReceiveReportsNoShortfallBeforeACompleteCycle) {
    create_io(this->api->add_slave());
    ASSERT_NIL(this->scan_and_activate());
    this->api->outputs.state = {.working_counter = 0, .wc_state = EC_WC_ZERO};
    ASSERT_OCCURRED_AS(this->master.receive(), errors::WORKING_COUNTER_ERROR);
    EXPECT_EQ(this->master.shortfall(), std::nullopt);
}

/// @brief it should report how far the working counter fell below the latest complete
/// cycle.
TEST_F(IGHMasterTest, ReceiveReportsTheShortfallBelowTheLatestCompleteCycle) {
    create_io(this->api->add_slave());
    ASSERT_NIL(this->scan_and_activate());
    this->api->outputs.state = {.working_counter = 1, .wc_state = EC_WC_COMPLETE};
    this->api->inputs.state = {.working_counter = 1, .wc_state = EC_WC_COMPLETE};
    ASSERT_NIL(this->master.receive());
    this->api->outputs.state = {.working_counter = 0, .wc_state = EC_WC_ZERO};
    ASSERT_OCCURRED_AS(this->master.receive(), errors::WORKING_COUNTER_ERROR);
    EXPECT_EQ(this->master.shortfall(), 1);
}

/// @brief it should give each enabled slave a working counter share of 1 for each
/// domain that holds one of its entries.
TEST_F(IGHMasterTest, ImageGivesTheWorkingCounterShareOfEachSlave) {
    create_io(this->api->add_slave());
    this->api->add_slave().syncs = {{EC_DIR_INPUT, {{0x1A00, {{0x6000, 0x01, 16}}}}}};
    ASSERT_NIL(this->scan_and_activate());
    EXPECT_EQ(
        this->image.working_counter_shares,
        (std::map<uint16_t, uint32_t>{{1, 2}, {2, 1}})
    );
}

/// @brief it should not count the zero working counter of an empty domain, which
/// exchanges no data.
TEST_F(IGHMasterTest, ReceiveIgnoresAnEmptyDomain) {
    this->api->add_slave().syncs = {{EC_DIR_INPUT, {{0x1A00, {{0x6000, 0x01, 16}}}}}};
    ASSERT_NIL(this->scan_and_activate());
    this->api->outputs.state.wc_state = EC_WC_ZERO;
    ASSERT_NIL(this->master.receive());
}

/// @brief it should fail when the master fails to receive frames.
TEST_F(IGHMasterTest, ReceiveFailsWhenTheMasterCannotReceive) {
    create_io(this->api->add_slave());
    ASSERT_NIL(this->scan_and_activate());
    this->api->receive_result = -EIO;
    const auto err = this->master.receive();
    ASSERT_OCCURRED_AS(err, errors::CYCLIC_ERROR);
    EXPECT_EQ(err.data, "failed to receive frames: Input/output error");
}

/// @brief it should fail when the master fails to send frames.
TEST_F(IGHMasterTest, SendFailsWhenTheMasterCannotSend) {
    create_io(this->api->add_slave());
    ASSERT_NIL(this->scan_and_activate());
    this->api->send_result = -EIO;
    const auto err = this->master.send();
    ASSERT_OCCURRED_AS(err, errors::CYCLIC_ERROR);
    EXPECT_EQ(err.data, "failed to send frames: Input/output error");
}

/// @brief it should fail when the master fails to process a domain.
TEST_F(IGHMasterTest, ReceiveFailsWhenADomainCannotBeProcessed) {
    create_io(this->api->add_slave());
    ASSERT_NIL(this->scan_and_activate());
    this->api->domain_process_result = -EIO;
    const auto err = this->master.receive();
    ASSERT_OCCURRED_AS(err, errors::CYCLIC_ERROR);
    EXPECT_EQ(err.data, "failed to process a domain: Input/output error");
}

/// @brief it should fail when the master fails to read the state of a domain.
TEST_F(IGHMasterTest, ReceiveFailsWhenADomainStateCannotBeRead) {
    create_io(this->api->add_slave());
    ASSERT_NIL(this->scan_and_activate());
    this->api->domain_state_result = -EIO;
    const auto err = this->master.receive();
    ASSERT_OCCURRED_AS(err, errors::CYCLIC_ERROR);
    EXPECT_EQ(err.data, "failed to read a domain state: Input/output error");
}

/// @brief it should fail when the master fails to queue a domain.
TEST_F(IGHMasterTest, SendFailsWhenADomainCannotBeQueued) {
    create_io(this->api->add_slave());
    ASSERT_NIL(this->scan_and_activate());
    this->api->domain_queue_result = -EIO;
    const auto err = this->master.send();
    ASSERT_OCCURRED_AS(err, errors::CYCLIC_ERROR);
    EXPECT_EQ(err.data, "failed to queue a domain: Input/output error");
}

/// @brief it should report a slave whose state or info it cannot read as lost.
TEST_F(IGHMasterTest, CheckReportsASlaveItCannotReadAsLost) {
    this->api->add_slave();
    this->api->add_slave();
    ASSERT_NIL(this->scan_and_activate());
    this->api->slaves[0].state_unreadable = true;
    this->api->slaves[1].unreadable = true;
    const auto health = this->master.check(this->brk);
    ASSERT_EQ(health.size(), 2);
    EXPECT_TRUE(health[0].lost);
    EXPECT_TRUE(health[1].lost);
}

/// @brief it should report a slave that is offline as lost.
TEST_F(IGHMasterTest, CheckReportsAnOfflineSlaveAsLost) {
    this->api->add_slave();
    ASSERT_NIL(this->scan_and_activate());
    this->api->slaves[0].online = false;
    const auto health = this->master.check(this->brk);
    ASSERT_EQ(health.size(), 1);
    EXPECT_TRUE(health[0].lost);
}

/// @brief it should report a slave with its error flag set as faulted.
TEST_F(IGHMasterTest, CheckReportsTheErrorFlagAsFaulted) {
    this->api->add_slave();
    ASSERT_NIL(this->scan_and_activate());
    this->api->slaves[0].info.al_state = EC_AL_STATE_SAFEOP;
    this->api->slaves[0].info.error_flag = 1;
    const auto health = this->master.check(this->brk);
    ASSERT_EQ(health.size(), 1);
    EXPECT_EQ(health[0].state, slave::State::SAFE_OP);
    EXPECT_TRUE(health[0].faulted);
}

/// @brief it should deactivate and release the master.
TEST_F(IGHMasterTest, DeactivateReleasesTheMaster) {
    this->api->add_slave();
    ASSERT_NIL(this->scan_and_activate());
    this->master.deactivate();
    EXPECT_TRUE(this->api->deactivated);
    EXPECT_TRUE(this->api->released);
}

/// @brief it should do nothing when the master never scanned.
TEST_F(IGHMasterTest, DeactivateDoesNothingBeforeScan) {
    this->master.deactivate();
    EXPECT_FALSE(this->api->released);
}
/// @brief it should create a master only for a key of the form igh:N.
TEST(IGHManagerTest, CreateRejectsAKeyThatIsNotOfTheFormIghN) {
    Manager manager(std::make_shared<MockAPI>());
    for (const std::string key: {"igh:", "igh:x", "igh:1x", "igh:-1", "eth0"}) {
        auto res = manager.create(key);
        EXPECT_EQ(
            res.second.data,
            "invalid IgH master key '" + key + "': expected format 'igh:N'"
        );
        ASSERT_OCCURRED_AS_P(std::move(res), errors::MASTER_INIT_ERROR);
    }
    EXPECT_NE(ASSERT_NIL_P(manager.create("igh:2")), nullptr);
}

namespace {
/// @brief an IgH master on its mock for the master contract.
class Bus {
    std::shared_ptr<MockAPI> api = std::make_shared<MockAPI>();
    Master m{this->api, 0, {.clock = x::loop::fake_clock()}};

public:
    Master &master() { return this->m; }

    void add_slave() {
        const auto name = "Slave " + std::to_string(this->api->slaves.size() + 1);
        this->api->add_slave(name).syncs = {
            {EC_DIR_INPUT, {{0x1A00, {{0x6000, 0x01, 16}}}}},
        };
    }

    void refuse(const uint16_t position) {
        auto &slave = this->api->slaves[position - 1];
        slave.reaches = EC_AL_STATE_PREOP;
        slave.refuses = true;
    }
};
}
}

namespace driver::ethercat::master::contract {
INSTANTIATE_TYPED_TEST_SUITE_P(IgH, Contract, igh::Bus);
}
