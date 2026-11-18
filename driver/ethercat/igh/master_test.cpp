// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cerrno>
#include <format>
#include <map>
#include <string>
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

/// @brief returns a CoE dictionary that holds 0x6000:07, a counter, and 0x6030:01, a
/// voltage, as IgH reads them from an IOLITE 6xSTG.
std::vector<MockAPI::DictionaryObject> create_dictionary() {
    return {
        {.info = {.index = 0x6000, .max_sub_index = 8, .name = "DI Inputs"},
         .entries = {{7, {.name = "BufCounter", .data_type = 0x0007}}}},
        {.info = {.index = 0x6030, .max_sub_index = 6, .name = "UnBuf Inputs"},
         .entries = {{1, {.name = "u1", .data_type = 0x0008}}}},
    };
}

/// @brief makes slave map the counter and the voltage of create_dictionary() through
/// PDO 0x1A00 on its inputs.
void create_dictionary_layout(MockAPI::Slave &slave) {
    slave.syncs = {
        {EC_DIR_INPUT, {{0x1A00, {{0x6000, 0x07, 32}, {0x6030, 0x01, 32}}}}}
    };
}

/// @brief returns a dictionary of count objects, each with one entry.
std::vector<MockAPI::DictionaryObject> create_objects(const uint16_t count) {
    std::vector<MockAPI::DictionaryObject> objects;
    objects.reserve(count);
    for (uint16_t i = 0; i < count; ++i)
        objects.push_back(
            {.info = {.index = static_cast<uint16_t>(0x2000 + i), .name = "Object"},
             .entries = {{0, {.name = "Value", .data_type = 0x0007}}}}
        );
    return objects;
}

/// @brief returns an SII that lists CoE in the mailbox protocols, with a general
/// category that says whether the slave gives SDO information.
std::vector<uint16_t> create_coe_sii(const bool sdo_info) {
    std::vector<uint16_t> words(0x40, 0);
    words[0x1C] = 0x0004;
    std::vector<uint16_t> general(16, 0);
    if (sdo_info) general[2] = 0x0200;
    words.push_back(30);
    words.push_back(static_cast<uint16_t>(general.size()));
    words.insert(words.end(), general.begin(), general.end());
    words.push_back(0xFFFF);
    return words;
}

/// @brief a mode that runs on the sync manager events.
const dc::OpMode FREE_RUN{.name = "FreeRun"};

/// @brief a mode that runs SYNC0 once in each bus cycle, 50 µs after the cycle starts.
const dc::OpMode DC_SYNC{
    .name = "DC",
    .assign_activate = 0x0300,
    .sync0_factor = 1,
    .sync0_shift = 50'000,
};

/// @brief returns an SII that lists modes, as the strings and DC categories hold them.
std::vector<uint16_t> create_sii(const std::vector<dc::OpMode> &modes) {
    std::vector<uint8_t> strings{static_cast<uint8_t>(modes.size())};
    std::vector<uint8_t> dc;
    const auto put = [&dc](const uint32_t value, const int bytes) {
        for (int i = 0; i < bytes; i++)
            dc.push_back(static_cast<uint8_t>(value >> (8 * i)));
    };
    for (size_t i = 0; i < modes.size(); i++) {
        const auto &mode = modes[i];
        strings.push_back(static_cast<uint8_t>(mode.name.size()));
        strings.insert(strings.end(), mode.name.begin(), mode.name.end());
        put(mode.sync0_cycle, 4);
        put(static_cast<uint32_t>(mode.sync0_shift), 4);
        put(static_cast<uint32_t>(mode.sync1_shift), 4);
        put(static_cast<uint16_t>(mode.sync1_factor), 2);
        put(mode.assign_activate, 2);
        put(static_cast<uint16_t>(mode.sync0_factor), 2);
        put(i + 1, 1);
        put(0, 5);
    }
    std::vector<uint16_t> words(0x40, 0);
    for (auto [type, bytes]: {std::pair{10, strings}, std::pair{60, dc}}) {
        if (bytes.size() % 2 != 0) bytes.push_back(0);
        words.push_back(static_cast<uint16_t>(type));
        words.push_back(static_cast<uint16_t>(bytes.size() / 2));
        for (size_t i = 0; i < bytes.size(); i += 2)
            words.push_back(static_cast<uint16_t>(bytes[i] | (bytes[i + 1] << 8)));
    }
    words.push_back(0xFFFF);
    return words;
}

/// @brief makes slave a slave with a distributed clock that lists modes.
void create_dc(MockAPI::Slave &slave, const std::vector<dc::OpMode> &modes) {
    slave.info.ports[0].receive_time = 1;
    slave.sii = create_sii(modes);
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
    x::loop::HostClock clock = x::loop::fake_clock();
    Master master{this->api, 0, {.clock = this->clock}};
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

/// @brief it should fail when the master cannot be opened.
TEST_F(IGHMasterTest, ScanFailsWhenTheMasterCannotBeOpened) {
    this->api->open_master_fails = true;
    const auto err = this->master.scan().second;
    ASSERT_OCCURRED_AS(err, errors::MASTER_INIT_ERROR);
    EXPECT_EQ(
        err.data,
        "failed to open igh:0. Check that libethercat comes from the EtherLab version "
        "of the ec_master kernel module"
    );
}

/// @brief it should scan without a reservation, so IgH keeps scanning the bus, and
/// reserve the master at activation.
TEST_F(IGHMasterTest, ScanLeavesTheMasterUnreservedUntilActivation) {
    this->api->add_slave();
    ASSERT_NIL_P(this->master.scan());
    EXPECT_FALSE(this->api->reserved);
    ASSERT_NIL(this->activate().err);
    EXPECT_TRUE(this->api->reserved);
}

/// @brief it should wait for a scan that IgH runs before it reads the slaves.
TEST_F(IGHMasterTest, ScanWaitsForTheScanOfIgH) {
    this->api->add_slave();
    this->api->scan_reads = 3;
    EXPECT_EQ(ASSERT_NIL_P(this->master.scan()).size(), 1);
    EXPECT_EQ(this->api->scan_reads, 0);
}

/// @brief it should fail, and release the master, when IgH does not finish its scan.
TEST_F(IGHMasterTest, ScanFailsWhenIgHKeepsScanning) {
    this->api->add_slave();
    this->api->scan_reads = 100000;
    const auto err = this->master.scan().second;
    ASSERT_OCCURRED_AS(err, errors::MASTER_INIT_ERROR);
    EXPECT_EQ(err.data, "IgH did not finish its scan of the bus within 10s");
    EXPECT_TRUE(this->api->released);
}

/// @brief it should fail, and release the master, when no slave answers.
TEST_F(IGHMasterTest, ScanFailsWhenNoSlaveAnswers) {
    const auto err = this->master.scan().second;
    ASSERT_OCCURRED_AS(err, errors::NO_SLAVES_FOUND);
    EXPECT_EQ(
        err.data,
        "No EtherCAT modules found on igh:0. Check the cable and the power of the first "
        "module."
    );
    EXPECT_TRUE(this->api->released);
}

/// @brief it should tell a network port with no link from slaves that do not answer.
TEST_F(IGHMasterTest, ScanFailsWhenThePortHasNoLink) {
    this->api->link_up = false;
    const auto err = this->master.scan().second;
    ASSERT_OCCURRED_AS(err, errors::NO_LINK);
    EXPECT_EQ(
        err.data,
        "No EtherCAT modules found on igh:0. Its network port has no link. Check the "
        "cable to the first module, and that MASTER0_DEVICE in the IgH configuration "
        "names the port."
    );
}

/// @brief it should release the master when it cannot read the master info.
TEST_F(IGHMasterTest, ScanFailsWhenTheMasterInfoCannotBeRead) {
    this->api->master_info_result = -1;
    const auto err = this->master.scan().second;
    ASSERT_OCCURRED_AS(err, errors::MASTER_INIT_ERROR);
    EXPECT_EQ(err.data, "failed to get master info");
    EXPECT_TRUE(this->api->released);
}

/// @brief it should fail activation when it cannot create the domain of a slave.
TEST_F(IGHMasterTest, ActivateFailsWhenADomainCannotBeCreated) {
    create_io(this->api->add_slave());
    this->api->domain_fails = true;
    const auto err = this->scan_and_activate();
    ASSERT_OCCURRED_AS(err, errors::ACTIVATION_ERROR);
    EXPECT_EQ(err.data, "failed to create the domain of module 1 (TestSlave)");
}

/// @brief it should fail to initialize, and release the master, when it cannot read a
/// slave.
TEST_F(IGHMasterTest, ScanFailsWhenASlaveCannotBeRead) {
    this->api->add_slave();
    this->api->add_slave().unreadable = true;
    const auto err = this->master.scan().second;
    ASSERT_OCCURRED_AS(err, errors::MASTER_INIT_ERROR);
    EXPECT_EQ(err.data, "failed to read module 2: Input/output error");
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

/// @brief it should name each entry and give its data type from the CoE dictionary
/// that IgH read from the slave, the final object of the dictionary included.
TEST_F(IGHMasterTest, ScanNamesEntriesFromTheDictionaryOfTheSlave) {
    auto &slave = this->api->add_slave();
    create_dictionary_layout(slave);
    slave.dictionary = create_dictionary();
    const auto slaves = ASSERT_NIL_P(this->master.scan());
    const auto &inputs = slaves[0].properties.input_pdos;
    ASSERT_EQ(inputs.size(), 2);
    EXPECT_EQ(inputs[0].name, "BufCounter");
    EXPECT_EQ(inputs[0].data_type, x::telem::UINT32_T);
    EXPECT_EQ(inputs[1].name, "u1");
    EXPECT_EQ(inputs[1].data_type, x::telem::FLOAT32_T);
}

/// @brief it should keep the generic name of an entry of the object that IgH still
/// reads, with no lookup that IgH would write to the kernel log.
TEST_F(IGHMasterTest, ScanKeepsTheNameOfAnEntryThatIgHHasNotRead) {
    auto &slave = this->api->add_slave();
    create_dictionary_layout(slave);
    slave.dictionary = create_dictionary();
    slave.dictionary[1].entries.clear();
    slave.dictionary.push_back({.info = {.index = 0xF050, .max_sub_index = 4}});
    const auto slaves = ASSERT_NIL_P(this->master.scan());
    const auto &inputs = slaves[0].properties.input_pdos;
    ASSERT_EQ(inputs.size(), 2);
    EXPECT_EQ(inputs[0].name, "BufCounter");
    EXPECT_EQ(inputs[1].name, "Input (uint32) 0x6030:01");
    EXPECT_EQ(slave.entry_reads, 1);
}

/// @brief it should name an entry at a later scan once IgH has read the dictionary.
TEST_F(IGHMasterTest, ScanNamesAnEntryOnceIgHHasReadIt) {
    auto &slave = this->api->add_slave();
    create_dictionary_layout(slave);
    const auto first = ASSERT_NIL_P(this->master.scan());
    EXPECT_EQ(first[0].properties.input_pdos[0].name, "Input (uint32) 0x6000:07");
    slave.dictionary = create_dictionary();
    const auto second = ASSERT_NIL_P(this->master.scan());
    EXPECT_EQ(second[0].properties.input_pdos[0].name, "BufCounter");
}

/// @brief it should read the CoE objects of a slave once IgH has read its dictionary,
/// the final entries included, as IgH mixes up the replies otherwise.
TEST_F(IGHMasterTest, ScanWaitsForIgHToReadTheDictionaryOfASlave) {
    auto &slave = this->api->add_slave();
    slave.sii = create_coe_sii(true);
    create_dictionary_layout(slave);
    slave.dictionary = create_dictionary();
    slave.dictionary[1].entries[6] = {.name = "u6", .data_type = 0x0008};
    slave.dictionary_read = 1;
    slave.wide_objects[{0x1018, 4}] = 7;
    const auto started = this->clock.steady();
    const auto slaves = ASSERT_NIL_P(this->master.scan());
    EXPECT_LT(this->clock.steady() - started, std::chrono::seconds(1));
    EXPECT_EQ(slaves[0].properties.serial, 7);
    EXPECT_EQ(slaves[0].status.serial_error, "");
    EXPECT_EQ(slaves[0].properties.input_pdos[1].name, "u1");
}

/// @brief it should keep waiting while IgH reads more of the dictionary, however long
/// the whole read takes.
TEST_F(IGHMasterTest, ScanWaitsWhileIgHKeepsReadingTheDictionary) {
    auto &slave = this->api->add_slave();
    slave.sii = create_coe_sii(true);
    slave.dictionary = create_objects(100);
    slave.dictionary_read = 1;
    slave.wide_objects[{0x1018, 4}] = 7;
    const auto started = this->clock.steady();
    const auto slaves = ASSERT_NIL_P(this->master.scan());
    EXPECT_GT(this->clock.steady() - started, std::chrono::seconds(5));
    EXPECT_EQ(slaves[0].properties.serial, 7);
    EXPECT_EQ(slaves[0].status.serial_error, "");
}

/// @brief it should read the CoE objects of a slave once IgH has read no more of its
/// dictionary for 5 s.
TEST_F(IGHMasterTest, ScanStopsWaitingOnceIgHStopsReadingTheDictionary) {
    auto &slave = this->api->add_slave();
    slave.sii = create_coe_sii(true);
    slave.dictionary = create_objects(3);
    slave.dictionary_read = 2;
    slave.dictionary_stalled = true;
    slave.wide_objects[{0x1018, 4}] = 7;
    const auto started = this->clock.steady();
    const auto slaves = ASSERT_NIL_P(this->master.scan());
    const auto waited = this->clock.steady() - started;
    EXPECT_GE(waited, std::chrono::seconds(5));
    EXPECT_LT(waited, std::chrono::seconds(6));
    EXPECT_EQ(slaves[0].properties.serial, 7);
}

/// @brief it should count an object whose description holds no name as described
/// once it has sub-indices or IgH describes a later object.
TEST_F(IGHMasterTest, ScanWaitsPastObjectsWithNoName) {
    auto &slave = this->api->add_slave();
    slave.sii = create_coe_sii(true);
    slave.dictionary = create_objects(3);
    slave.dictionary[1].info.name.clear();
    slave.dictionary[2] = {
        .info = {.index = 0x2002, .max_sub_index = 1},
        .entries = {{1, {.name = "Value", .data_type = 0x0007}}},
    };
    slave.dictionary_read = 1;
    slave.wide_objects[{0x1018, 4}] = 7;
    const auto started = this->clock.steady();
    const auto slaves = ASSERT_NIL_P(this->master.scan());
    EXPECT_LT(this->clock.steady() - started, std::chrono::seconds(1));
    EXPECT_EQ(slaves[0].properties.serial, 7);
}

/// @brief it should look up a final entry that IgH does not have less and less often,
/// as IgH logs each miss.
TEST_F(IGHMasterTest, ScanBacksOffLookupsOfAMissingFinalEntry) {
    auto &slave = this->api->add_slave();
    slave.sii = create_coe_sii(true);
    slave.dictionary = create_objects(1);
    slave.dictionary[0].info.max_sub_index = 2;
    ASSERT_NIL_P(this->master.scan());
    EXPECT_LE(slave.entry_reads, 7);
}

/// @brief it should not wait for a dictionary that IgH does not read: that of a slave
/// with no CoE, or with no SDO information.
TEST_F(IGHMasterTest, ScanDoesNotWaitForADictionaryThatIgHDoesNotRead) {
    this->api->add_slave();
    this->api->add_slave().sii = create_coe_sii(false);
    const auto started = this->clock.steady();
    ASSERT_NIL_P(this->master.scan());
    EXPECT_LT(this->clock.steady() - started, std::chrono::seconds(1));
}

/// @brief it should keep the name and data type that the registry gives a known
/// slave at each scan, as SOEM does.
TEST_F(IGHMasterTest, ScanPrefersTheRegistryToTheDictionary) {
    constexpr uint32_t beckhoff = 0x00000002;
    constexpr uint32_t el3102 = 0x0C1E3052;
    slave::Properties registry{};
    ASSERT_TRUE(esi::lookup_device_pdos(beckhoff, el3102, 1, registry));
    const auto known = std::ranges::find_if(registry.input_pdos, [](const auto &p) {
        return !p.is_padding() && p.bit_length == 16;
    });
    ASSERT_NE(known, registry.input_pdos.end());
    auto &slave = this->api->add_slave("EL3102", beckhoff, el3102);
    slave.syncs = {
        {EC_DIR_INPUT, {{0x1A00, {{known->index, known->sub_index, 16}}}}},
    };
    slave.dictionary = {
        {.info =
             {.index = known->index, .max_sub_index = known->sub_index, .name = "AI"},
         .entries = {{known->sub_index, {.name = "Other", .data_type = 0x0006}}}},
    };
    ASSERT_NIL_P(this->master.scan());
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

/// @brief it should read the 8-byte CoE serial number of a slave whose SII serial
/// number is 0, and keep a nonzero SII serial number.
TEST_F(IGHMasterTest, ScanReadsTheCoESerialWhenTheSIISerialIsZero) {
    this->api->add_slave().wide_objects[{0x1018, 4}] = 0x000000DB21014087;
    auto &sii = this->api->add_slave();
    sii.info.serial_number = 42;
    sii.wide_objects[{0x1018, 4}] = 7;
    const auto slaves = ASSERT_NIL_P(this->master.scan());
    EXPECT_EQ(slaves[0].properties.serial, 0x000000DB21014087);
    EXPECT_EQ(slaves[1].properties.serial, 42);
    EXPECT_FALSE(slaves[0].serial_unread());
}

/// @brief it should count a CoE serial number that the Core cannot store exactly as
/// none.
TEST_F(IGHMasterTest, ScanCountsATooLargeCoESerialAsNone) {
    this->api->add_slave().wide_objects[{0x1018, 4}] = uint64_t{1} << 53;
    const auto slaves = ASSERT_NIL_P(this->master.scan());
    EXPECT_EQ(slaves[0].properties.serial, 0);
    EXPECT_FALSE(slaves[0].serial_unread());
}

/// @brief it should report why a slave gave no CoE serial number when a later read can
/// get it, and leave a slave that refuses the read for good with none.
TEST_F(IGHMasterTest, ScanReportsACoESerialReadWithNoReply) {
    this->api->add_slave().upload_errors[{0x1018, 4}] = -EIO;
    this->api->add_slave().upload_errors[{0x1018, 4}] = -EPROTONOSUPPORT;
    const auto slaves = ASSERT_NIL_P(this->master.scan());
    EXPECT_EQ(slaves[0].status.serial_error, "failed to read 0x1018:04: no response");
    EXPECT_TRUE(slaves[0].serial_unread());
    EXPECT_TRUE(slaves[1].status.serial_error.empty());
    EXPECT_FALSE(slaves[1].serial_unread());
}

/// @brief it should give serial number 0 to a slave with the model and CoE serial
/// number of a slave before it.
TEST_F(IGHMasterTest, ScanClearsADuplicateCoESerial) {
    this->api->add_slave().wide_objects[{0x1018, 4}] = 9;
    this->api->add_slave().wide_objects[{0x1018, 4}] = 9;
    const auto slaves = ASSERT_NIL_P(this->master.scan());
    EXPECT_EQ(slaves[0].properties.serial, 9);
    EXPECT_EQ(slaves[1].properties.serial, 0);
}

/// @brief it should report the AL state of each slave, and read nothing through the
/// mailbox of a slave below PRE-OP.
TEST_F(IGHMasterTest, ScanReadsOnlyTheSIIOfAStartingSlave) {
    for (const uint8_t state: {EC_AL_STATE_INIT, EC_AL_STATE_SAFEOP}) {
        auto &slave = this->api->add_slave(
            "6xSTG",
            testutil::DEWESOFT,
            testutil::IOLITE_6XSTG
        );
        slave.info.al_state = state;
        slave.objects = testutil::stg_objects(0);
        slave.wide_objects[{0x1018, 4}] = 9;
    }
    const auto slaves = ASSERT_NIL_P(this->master.scan());
    EXPECT_EQ(slaves[0].status.state, slave::State::INIT);
    EXPECT_TRUE(slaves[0].status.starting);
    EXPECT_EQ(slaves[0].properties.serial, 0);
    EXPECT_FALSE(slaves[0].properties.settings.has_value());
    EXPECT_TRUE(slaves[0].status.settings_error.empty());
    EXPECT_EQ(slaves[1].status.state, slave::State::SAFE_OP);
    EXPECT_FALSE(slaves[1].status.starting);
    EXPECT_EQ(slaves[1].properties.serial, 9);
    EXPECT_TRUE(slaves[1].properties.settings.has_value());
}

/// @brief it should read the CoE objects of each slave that was starting at the scan
/// once it reaches PRE-OP, disabled ones included, and start without one that stays in
/// INIT.
TEST_F(IGHMasterTest, ActivateReadsAStartingSlaveOnceItReachesPreOP) {
    for (const int reads: {1, -1}) {
        auto &slave = this->api->add_slave(
            "6xSTG",
            testutil::DEWESOFT,
            testutil::IOLITE_6XSTG
        );
        slave.info.al_state = EC_AL_STATE_INIT;
        slave.reads_to_pre_op = reads;
        slave.objects = testutil::stg_objects(0);
        slave.wide_objects[{0x1018, 4}] = 9;
    }
    auto &disabled = this->api->add_slave();
    disabled.info.al_state = EC_AL_STATE_INIT;
    disabled.reads_to_pre_op = 1;
    disabled.wide_objects[{0x1018, 4}] = 10;
    this->configs = {{1, {}}, {2, {}}, {3, {.enabled = false}}};
    ASSERT_NIL_P(this->master.scan());
    const auto started = this->clock.steady();
    ASSERT_NIL(this->activate().err);
    EXPECT_GE(this->clock.steady() - started, std::chrono::seconds(2));
    const auto &slaves = this->image.slaves;
    EXPECT_EQ(slaves[0].status.state, slave::State::PRE_OP);
    EXPECT_FALSE(slaves[0].status.starting);
    EXPECT_EQ(slaves[0].properties.serial, 9);
    EXPECT_TRUE(slaves[0].properties.settings.has_value());
    EXPECT_TRUE(slaves[1].status.starting);
    EXPECT_FALSE(slaves[1].properties.settings.has_value());
    EXPECT_FALSE(slaves[2].status.starting);
    EXPECT_EQ(slaves[2].properties.serial, 10);
}

/// @brief it should wait for IgH to read the dictionary of a slave that reaches PRE-OP
/// after the scan before it reads the CoE objects of the slave.
TEST_F(IGHMasterTest, ActivateWaitsForTheDictionaryOfASlaveThatReachesPreOP) {
    auto &slave = this->api->add_slave();
    slave.info.al_state = EC_AL_STATE_INIT;
    slave.reads_to_pre_op = 1;
    slave.sii = create_coe_sii(true);
    slave.dictionary = create_objects(20);
    slave.dictionary_read = 1;
    slave.wide_objects[{0x1018, 4}] = 7;
    this->configs = {{1, {}}};
    ASSERT_NIL(this->scan_and_activate());
    EXPECT_EQ(this->image.slaves[0].properties.serial, 7);
    EXPECT_EQ(this->image.slaves[0].status.serial_error, "");
}

/// @brief it should give the reason for each way an SDO read fails.
TEST_F(IGHMasterTest, ReadGivesTheReasonForEachFailure) {
    auto &slave = this->api->add_slave();
    slave.upload_errors = {
        {{0x8000, 1}, -EIO},
        {{0x8000, 2}, -EPROTO},
        {{0x8000, 3}, -EPROTONOSUPPORT},
    };
    slave.wide_objects[{0x8000, 4}] = 1;
    ASSERT_NIL_P(this->master.scan());
    const std::vector<std::pair<uint8_t, std::string>> reasons{
        {1, "no response"},
        {2, "the module refused the mailbox request"},
        {3, "the module refused the mailbox request"},
        {4, "the reply is larger than the read allows"},
        {5, "Object does not exist in the object dictionary"},
    };
    for (const auto &[sub_index, reason]: reasons) {
        const auto err = this->master.read(1, 0x8000, sub_index).second;
        ASSERT_OCCURRED_AS(err, errors::SDO_READ_ERROR);
        EXPECT_EQ(
            err.data,
            std::format("failed to read 0x8000:{:02X}: {}", sub_index, reason)
        );
    }
    EXPECT_TRUE(this->master.read(1, 0x8000, 1).second.matches(coe::NO_REPLY));
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

/// @brief it should report a change while IgH scans the bus, as the slaves changed.
TEST_F(IGHMasterTest, ChangedWhileIgHScansTheBus) {
    this->api->add_slave();
    ASSERT_NIL_P(this->master.scan());
    this->api->scan_reads = 1;
    EXPECT_TRUE(ASSERT_NIL_P(this->master.changed()).stale);
    EXPECT_FALSE(ASSERT_NIL_P(this->master.changed()).stale);
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

/// @brief it should tell why it cannot reserve the master.
TEST_F(IGHMasterTest, ActivateFailsWhenTheMasterCannotBeReserved) {
    this->api->add_slave();
    ASSERT_NIL_P(this->master.scan());
    const std::vector<std::pair<int, std::string>> cases{
        {-EBUSY, "another program uses igh:0. Stop it, then start the task again"},
        {-ENODEV,
         "no network port is attached to igh:0. Set MASTER0_DEVICE in the IgH "
         "configuration to the MAC address of the port, then restart IgH with 'sudo "
         "systemctl restart ethercat'"},
        {-EINTR, "failed to reserve igh:0: Interrupted system call"},
    };
    for (const auto &[ret, msg]: cases) {
        this->api->reserve_result = ret;
        const auto err = this->activate().err;
        ASSERT_OCCURRED_AS(err, errors::ACTIVATION_ERROR);
        EXPECT_EQ(err.data, msg);
    }
}

/// @brief it should fail activation, before it configures a slave, when the slaves
/// changed between the scan and the reservation, so that the caller scans again.
TEST_F(IGHMasterTest, ActivateFailsWhenTheSlavesChangedSinceTheScan) {
    this->api->add_slave("EL3102", 0x2, 0x0C1E3052);
    ASSERT_NIL_P(this->master.scan());
    this->api->slaves[0].info.product_code = 0x03F03052;
    const auto err = this->activate().err;
    ASSERT_OCCURRED_AS(err, errors::ACTIVATION_ERROR);
    EXPECT_EQ(err.data, "the modules on igh:0 changed since the last scan");
    EXPECT_FALSE(this->api->slaves[0].configured);
}

/// @brief it should refuse a used slave with no identity once the image shows the other
/// slaves, without configuring the slave or checking its chosen mode.
TEST_F(IGHMasterTest, ActivateRefusesAUsedSlaveWithNoIdentity) {
    this->api->add_slave("Unknown", 0, 0);
    create_io(this->api->add_slave());
    ASSERT_NIL_P(this->master.scan());
    this->configs = {
        {1, {.op_mode = dc::Choice{.index = 2, .assign_activate = 0x0700}}}
    };
    const auto activation = this->activate();
    ASSERT_OCCURRED_AS(activation.err, errors::ACTIVATION_ERROR);
    EXPECT_EQ(
        activation.err.data,
        "module 1 (Unknown) gave no identity when IgH read its EEPROM"
    );
    const std::map<uint16_t, slave::Refusal> refusals{
        {1, {.reason = "gave no identity when IgH read its EEPROM"}},
    };
    EXPECT_EQ(activation.refusals, refusals);
    EXPECT_FALSE(this->api->slaves[0].configured);
    EXPECT_FALSE(this->image.offsets.empty());
}

/// @brief it should refuse a used slave with no identity before it starts the cycle
/// when no other slave has process data.
TEST_F(IGHMasterTest, ActivateRefusesTheOnlySlaveWithIOWhenItHasNoIdentity) {
    create_io(this->api->add_slave("Unknown", 0, 0));
    ASSERT_NIL_P(this->master.scan());
    bool started = false;
    const auto activation = this->master.activate(
        {.period = PERIOD},
        [&started](const master::Image &) { started = true; }
    );
    ASSERT_OCCURRED_AS(activation.err, errors::ACTIVATION_ERROR);
    EXPECT_TRUE(activation.refusals.contains(1));
    EXPECT_FALSE(started);
}

/// @brief it should start a bus with a slave that has no identity when no task uses the
/// slave.
TEST_F(IGHMasterTest, ActivateStartsPastAnUnusedSlaveWithNoIdentity) {
    this->api->add_slave("Unknown", 0, 0);
    create_io(this->api->add_slave());
    ASSERT_NIL_P(this->master.scan());
    this->configs = {{1, {.enabled = false}}};
    const auto activation = this->activate();
    ASSERT_NIL(activation.err);
    EXPECT_TRUE(activation.refusals.empty());
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
    EXPECT_EQ(err.data, "failed to configure module 1 (TestSlave)");
}

/// @brief it should give the cycle the domains, with the offset of each entry.
TEST_F(IGHMasterTest, ActivateGivesTheProcessImage) {
    create_io(this->api->add_slave());
    create_io(this->api->add_slave());
    ASSERT_NIL(this->scan_and_activate());
    EXPECT_EQ(this->api->domains.size(), 2);
    EXPECT_EQ(this->image.outputs.size(), 12);
    EXPECT_EQ(this->image.inputs.size(), 12);
    const auto output = ASSERT_NIL_P(this->image.offset(
        {.slave_position = 2,
         .index = 0x7000,
         .sub_index = 0x01,
         .bit_length = 16,
         .is_input = false}
    ));
    EXPECT_EQ(output.byte, 6);
    const auto input = ASSERT_NIL_P(this->image.offset(
        {.slave_position = 2,
         .index = 0x6000,
         .sub_index = 0x01,
         .bit_length = 32,
         .is_input = true}
    ));
    EXPECT_EQ(input.byte, 8);
}

/// @brief it should create no domain for a slave with no process data.
TEST_F(IGHMasterTest, ActivateCreatesNoDomainForASlaveWithNoProcessData) {
    this->api->add_slave();
    create_io(this->api->add_slave());
    ASSERT_NIL(this->scan_and_activate());
    ASSERT_EQ(this->api->domains.size(), 1);
    EXPECT_EQ(this->api->slaves[1].domain, 0);
    EXPECT_EQ(this->image.working_counter_shares.count(1), 0);
    ASSERT_NIL(this->master.receive());
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
/// refusal of its next state, after one reset and before the OP timeout.
TEST_F(IGHMasterTest, ActivateReturnsTheRefusalOfEachSlave) {
    this->api->add_slave("EL3102");
    auto &pre_op = this->api->add_slave("EL4002");
    pre_op.reaches = EC_AL_STATE_PREOP;
    pre_op.refuses = true;
    auto &safe_op = this->api->add_slave("EL1008");
    safe_op.reaches = EC_AL_STATE_SAFEOP;
    safe_op.refuses = true;
    ASSERT_NIL_P(this->master.scan());
    const auto started = this->clock.steady();
    const auto activation = this->activate();
    EXPECT_LT(this->clock.steady() - started, std::chrono::seconds(1));
    EXPECT_EQ(this->api->resets, 1);
    ASSERT_OCCURRED_AS(activation.err, errors::STATE_CHANGE_REFUSED);
    EXPECT_EQ(
        activation.err.data,
        "failed to reach OP: module 2 (EL4002) is in PRE-OP with an AL error; module "
        "3 (EL1008) is in SAFE-OP with an AL error"
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
        "failed to reach OP within 10s: module 1 (TestSlave) is in SAFE-OP"
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
    const auto activation = this->master.activate(
        {.period = PERIOD},
        [this](const master::Image &) { this->api->slaves[0].unreadable = true; }
    );
    ASSERT_OCCURRED_AS(activation.err, errors::STATE_CHANGE_ERROR);
    EXPECT_EQ(
        activation.err.data,
        "failed to reach OP within 10s: module 1 (TestSlave): failed to read its "
        "state: Input/output error"
    );
}

/// @brief it should reset the master once when every slave below OP refuses, and start
/// when the reset brings each one to OP.
TEST_F(IGHMasterTest, ActivateTriesARefusingSlaveAgainAfterAReset) {
    auto &slave = this->api->add_slave();
    slave.reaches = EC_AL_STATE_SAFEOP;
    slave.refuses = true;
    slave.recovers_on_reset = true;
    ASSERT_NIL(this->scan_and_activate());
    EXPECT_EQ(this->api->resets, 1);
}

/// @brief it should not take an error flag that a reset has yet to clear as a second
/// refusal.
TEST_F(IGHMasterTest, ActivateWaitsForAResetToClearTheErrorFlags) {
    auto &slave = this->api->add_slave();
    slave.reaches = EC_AL_STATE_SAFEOP;
    slave.refuses = true;
    slave.recovers_on_reset = true;
    this->api->reset_lag = 20;
    ASSERT_NIL(this->scan_and_activate());
    EXPECT_EQ(this->api->resets, 1);
}

/// @brief it should wait for a slave below OP with no error flag even when another
/// slave refuses.
TEST_F(IGHMasterTest, ActivateWaitsForASlaveBelowOPWithNoErrorFlag) {
    auto &refusing = this->api->add_slave("EL4002");
    refusing.reaches = EC_AL_STATE_PREOP;
    refusing.refuses = true;
    this->api->add_slave("EL1008").reaches = EC_AL_STATE_SAFEOP;
    ASSERT_NIL_P(this->master.scan());
    const auto started = this->clock.steady();
    const auto activation = this->activate();
    EXPECT_GE(this->clock.steady() - started, std::chrono::seconds(10));
    EXPECT_EQ(this->api->resets, 0);
    ASSERT_OCCURRED_AS(activation.err, errors::STATE_CHANGE_REFUSED);
    EXPECT_EQ(
        activation.err.data,
        "failed to reach OP within 10s: module 1 (EL4002) is in PRE-OP with an AL "
        "error; module 2 (EL1008) is in SAFE-OP"
    );
    const std::map<uint16_t, slave::Refusal> refusals{
        {1, {.reason = "refused SAFE-OP"}},
    };
    EXPECT_EQ(activation.refusals, refusals);
}

/// @brief it should return the refusals at once when the reset fails.
TEST_F(IGHMasterTest, ActivateReturnsTheRefusalsWhenTheResetFails) {
    auto &slave = this->api->add_slave();
    slave.reaches = EC_AL_STATE_PREOP;
    slave.refuses = true;
    slave.recovers_on_reset = true;
    this->api->reset_result = -EIO;
    ASSERT_NIL_P(this->master.scan());
    const auto activation = this->activate();
    EXPECT_EQ(this->api->resets, 1);
    ASSERT_OCCURRED_AS(activation.err, errors::STATE_CHANGE_REFUSED);
    EXPECT_EQ(
        activation.err.data,
        "failed to reach OP: module 1 (TestSlave) is in PRE-OP with an AL error"
    );
}

/// @brief it should read the modes of each slave from its SII, and read none from a SII
/// with no categories.
TEST_F(IGHMasterTest, ScanReadsTheModesOfEachSlaveFromItsSII) {
    create_dc(this->api->add_slave(), {FREE_RUN, DC_SYNC});
    this->api->add_slave();
    const auto slaves = ASSERT_NIL_P(this->master.scan());
    EXPECT_EQ(slaves[0].properties.op_modes, (std::vector{FREE_RUN, DC_SYNC}));
    EXPECT_TRUE(slaves[1].properties.op_modes.empty());
}

/// @brief it should fail the scan when it cannot read the SII of a slave.
TEST_F(IGHMasterTest, ScanFailsWhenTheSIICannotBeRead) {
    this->api->add_slave();
    this->api->add_slave().sii_result = -EIO;
    const auto err = this->master.scan().second;
    ASSERT_OCCURRED_AS(err, errors::MASTER_INIT_ERROR);
    EXPECT_EQ(err.data, "failed to read the SII of module 2: Input/output error");
}

/// @brief it should fail activation on a chosen mode that the slave does not list,
/// before it configures any slave.
TEST_F(IGHMasterTest, ActivateFailsOnAChosenModeTheSlaveDoesNotList) {
    this->api->add_slave();
    create_dc(this->api->add_slave(), {FREE_RUN, DC_SYNC});
    this->configs = {
        {2, {.op_mode = dc::Choice{.index = 1, .assign_activate = 0x0700}}}
    };
    const auto err = this->scan_and_activate();
    ASSERT_OCCURRED_AS(err, errors::SLAVE_CONFIG_ERROR);
    EXPECT_EQ(
        err.data,
        "module 2 (TestSlave) cannot use its chosen mode: the module does not list "
        "mode 1 with AssignActivate 0x0700. It lists 0 \"FreeRun\" (AssignActivate "
        "0x0000), 1 \"DC\" (AssignActivate 0x0300)."
    );
    EXPECT_FALSE(this->api->slaves[0].configured);
}

/// @brief it should fail activation on a chosen cycle for an SM mode, as the SOEM
/// master does.
TEST_F(IGHMasterTest, ActivateFailsOnAChosenCycleForAnSMMode) {
    create_dc(this->api->add_slave(), {FREE_RUN, DC_SYNC});
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
        "module 1 (TestSlave) cannot use its chosen mode: mode \"FreeRun\" "
        "(AssignActivate 0x0000) does not run SYNC0, so it cannot use a SYNC0 cycle "
        "of 1ms"
    );
    EXPECT_FALSE(this->api->slaves[0].configured);
}

/// @brief it should run a slave in a chosen SM mode, with no DC sync.
TEST_F(IGHMasterTest, ActivateRunsAChosenSMMode) {
    create_dc(this->api->add_slave(), {DC_SYNC, FREE_RUN});
    this->configs = {{1, {.op_mode = dc::Choice{.index = 1, .assign_activate = 0}}}};
    ASSERT_NIL(this->scan_and_activate());
    EXPECT_TRUE(this->api->slaves[0].configured);
    EXPECT_FALSE(this->api->slaves[0].dc);
    EXPECT_TRUE(this->image.dc_sync.empty());
    EXPECT_FALSE(this->image.reference_clock);
}

/// @brief it should set up the DC sync of each slave in a DC mode, and give the first
/// slave with a distributed clock as the reference clock.
TEST_F(IGHMasterTest, ActivateSetsUpTheDCSyncOfEachSlaveInADCMode) {
    this->api->add_slave();
    create_dc(this->api->add_slave(), {FREE_RUN, DC_SYNC});
    create_dc(this->api->add_slave(), {DC_SYNC});
    ASSERT_NIL(this->scan_and_activate());
    EXPECT_FALSE(this->api->slaves[1].dc);
    EXPECT_EQ(
        this->api->slaves[2].dc,
        (MockAPI::DC{
            .assign_activate = 0x0300,
            .sync0_cycle = 1'000'000,
            .sync0_shift = 50'000,
            .sync1_cycle = 0,
            .sync1_shift = 0,
        })
    );
    EXPECT_EQ(
        this->image.dc_sync,
        (std::vector<dc::Registers>{{
            .assign_activate = 0x0300,
            .sync0_cycle = 1'000'000,
            .sync0_shift = 50'000,
        }})
    );
    EXPECT_EQ(this->image.reference_clock, 2);
}

/// @brief it should give IgH a first application time on whole SYNC0 cycles of each
/// slave, at or before the time of the first cycle, so that the SYNC0 events start on
/// whole cycles after the DC epoch.
TEST_F(IGHMasterTest, ActivateStartsTheSYNC0EventsOnWholeCycles) {
    create_dc(this->api->add_slave(), {DC_SYNC});
    auto slow = DC_SYNC;
    slow.sync0_factor = 3;
    create_dc(this->api->add_slave(), {slow});
    ASSERT_NIL(this->scan_and_activate());
    ASSERT_NIL(this->master.send());
    ASSERT_EQ(this->api->app_times.size(), 2);
    EXPECT_EQ(this->api->app_times[0] % 3'000'000, 0);
    EXPECT_LE(this->api->app_times[0], this->api->app_times[1]);
    EXPECT_LT(this->api->app_times[1] - this->api->app_times[0], 3'000'000);
}

/// @brief it should fail activation, before it configures any slave, when SYNC1 of a
/// slave does not come a whole number of SYNC0 cycles after SYNC0, since IgH would
/// start its SYNC0 events off the grid.
TEST_F(IGHMasterTest, ActivateFailsOnASYNC1ShiftOffTheSYNC0Grid) {
    auto shifted = DC_SYNC;
    shifted.assign_activate = 0x0700;
    shifted.sync1_factor = 1;
    shifted.sync1_shift = 250'000;
    create_dc(this->api->add_slave(), {shifted});
    const auto err = this->scan_and_activate();
    ASSERT_OCCURRED_AS(err, errors::SLAVE_CONFIG_ERROR);
    EXPECT_EQ(
        err.data,
        "module 1 (TestSlave) cannot run its DC mode on the IgH master: SYNC1 comes "
        "250us after SYNC0, which is not a whole number of SYNC0 cycles of 1ms"
    );
    EXPECT_FALSE(this->api->slaves[0].configured);
}

/// @brief it should fail activation, before it configures any slave, when the SYNC0
/// cycles of the slaves have no common multiple that IgH can start them on.
TEST_F(IGHMasterTest, ActivateFailsOnSYNC0CyclesWithNoCommonStart) {
    for (const int64_t cycles: {42'949, 42'947, 42'943}) {
        create_dc(this->api->add_slave(), {DC_SYNC});
        this->configs[static_cast<uint16_t>(this->api->slaves.size())] = {
            .op_mode = dc::Choice{
                .index = 0,
                .assign_activate = DC_SYNC.assign_activate,
                .cycles = {.sync0 = x::telem::MICROSECOND * 100 * cycles},
            },
        };
    }
    const auto err = this->scan_and_activate();
    ASSERT_OCCURRED_AS(err, errors::SLAVE_CONFIG_ERROR);
    EXPECT_EQ(
        err.data,
        "the SYNC0 cycles of the modules have no common multiple that the IgH master "
        "can start them on"
    );
    EXPECT_FALSE(this->api->slaves[0].configured);
}

/// @brief it should fail activation when IgH cannot set up the DC sync of a slave.
TEST_F(IGHMasterTest, ActivateFailsWhenTheDCSyncCannotBeSetUp) {
    create_dc(this->api->add_slave(), {DC_SYNC});
    this->api->dc_config_result = -ENOMEM;
    const auto err = this->scan_and_activate();
    ASSERT_OCCURRED_AS(err, errors::ACTIVATION_ERROR);
    EXPECT_EQ(
        err.data,
        "failed to set up the DC sync of module 1 (TestSlave): Cannot allocate memory"
    );
}

/// @brief it should fail a send when IgH does not take the application time.
TEST_F(IGHMasterTest, SendFailsWhenTheApplicationTimeIsNotTaken) {
    create_dc(this->api->add_slave(), {DC_SYNC});
    ASSERT_NIL(this->scan_and_activate());
    this->api->app_time_result = -EPERM;
    const auto err = this->master.send();
    ASSERT_OCCURRED_AS(err, errors::CYCLIC_ERROR);
    EXPECT_EQ(err.data, "failed to give the application time: Operation not permitted");
}

/// @brief it should give the application time in each cycle, and sync the reference
/// clock to it and the other clocks to the reference clock.
TEST_F(IGHMasterTest, CycleSyncsTheClocksToTheApplicationTime) {
    create_dc(this->api->add_slave(), {DC_SYNC});
    ASSERT_NIL(this->scan_and_activate());
    ASSERT_NIL(this->master.send());
    ASSERT_EQ(this->api->app_times.size(), 2);
    EXPECT_EQ(this->api->reference_syncs, std::vector{this->api->app_times[1]});
    EXPECT_EQ(this->api->slave_syncs, 1);
}

/// @brief it should sync no clocks and read no DC time when no slave runs a DC mode.
TEST_F(IGHMasterTest, CycleSyncsNoClocksWithoutADCMode) {
    auto &slave = this->api->add_slave();
    create_io(slave);
    create_dc(slave, {FREE_RUN, DC_SYNC});
    ASSERT_NIL(this->scan_and_activate());
    ASSERT_NIL(this->master.send());
    ASSERT_NIL(this->master.receive());
    EXPECT_TRUE(this->api->app_times.empty());
    EXPECT_EQ(this->api->slave_syncs, 0);
    EXPECT_FALSE(this->master.dc_time());
}

/// @brief it should go on while IgH has no reference clock, as during a scan of the
/// bus.
TEST_F(IGHMasterTest, SendGoesOnWithoutAReferenceClock) {
    create_dc(this->api->add_slave(), {DC_SYNC});
    ASSERT_NIL(this->scan_and_activate());
    this->api->clock_sync_result = -ENXIO;
    ASSERT_NIL(this->master.send());
}

/// @brief it should fail a send when IgH cannot sync the clocks.
TEST_F(IGHMasterTest, SendFailsWhenTheClocksCannotSync) {
    create_dc(this->api->add_slave(), {DC_SYNC});
    ASSERT_NIL(this->scan_and_activate());
    this->api->clock_sync_result = -EIO;
    const auto err = this->master.send();
    ASSERT_OCCURRED_AS(err, errors::CYCLIC_ERROR);
    EXPECT_EQ(err.data, "failed to sync the distributed clocks: Input/output error");
}

/// @brief it should read the DC time once a slave gives a working counter, as the time
/// nearest the application time of the frame with the low 32 bits that the reference
/// clock gave.
TEST_F(IGHMasterTest, DCTimeReadsTheReferenceClockOnceASlaveAnswers) {
    auto &slave = this->api->add_slave();
    create_io(slave);
    create_dc(slave, {DC_SYNC});
    ASSERT_NIL(this->scan_and_activate());
    this->api->domain_of(slave).state = {.working_counter = 0, .wc_state = EC_WC_ZERO};
    ASSERT_NIL(this->master.send());
    ASSERT_OCCURRED_AS(this->master.receive(), errors::WORKING_COUNTER_ERROR);
    EXPECT_FALSE(this->master.dc_time());
    this->api->domain_of(slave).state = {
        .working_counter = 3,
        .wc_state = EC_WC_COMPLETE
    };
    for (const int64_t ahead: {500, -1000}) {
        ASSERT_NIL(this->master.send());
        const auto sent = this->api->app_times.back();
        this->api->reference_clock_time = static_cast<uint32_t>(sent + ahead);
        ASSERT_NIL(this->master.receive());
        EXPECT_EQ(
            this->master.dc_time(),
            master::DC_EPOCH + x::telem::TimeSpan(static_cast<int64_t>(sent) + ahead)
        );
    }
}

/// @brief it should give no DC time for a frame that did not reach the reference clock,
/// or while IgH has no reference clock during a scan of the bus.
TEST_F(IGHMasterTest, DCTimeGivesNoneForAFrameWithNoTime) {
    auto &slave = this->api->add_slave();
    create_io(slave);
    create_dc(slave, {DC_SYNC});
    ASSERT_NIL(this->scan_and_activate());
    for (const int ret: {-EIO, -ENXIO}) {
        this->api->reference_clock_result = 0;
        ASSERT_NIL(this->master.send());
        ASSERT_NIL(this->master.receive());
        ASSERT_TRUE(this->master.dc_time());
        this->api->reference_clock_result = ret;
        ASSERT_NIL(this->master.send());
        ASSERT_NIL(this->master.receive());
        EXPECT_FALSE(this->master.dc_time());
    }
}

/// @brief it should find the reference clock again on a scan after a failed one, as the
/// bus can change in between.
TEST_F(IGHMasterTest, ScanFindsTheReferenceClockAgainAfterAFailedScan) {
    this->api->add_slave();
    this->api->add_slave();
    auto &slaves = this->api->slaves;
    slaves[0].info.ports[0].receive_time = 1;
    slaves[1].sii_result = -EIO;
    ASSERT_OCCURRED_AS_P(this->master.scan(), errors::MASTER_INIT_ERROR);
    slaves[0].info.ports[0].receive_time = 0;
    slaves[1].sii_result = 0;
    create_dc(slaves[1], {DC_SYNC});
    ASSERT_NIL(this->scan_and_activate());
    EXPECT_EQ(this->image.reference_clock, 2);
}

/// @brief it should fail a receive when IgH cannot read the reference clock for a
/// reason other than a missed frame or a scan of the bus.
TEST_F(IGHMasterTest, ReceiveFailsWhenTheReferenceClockCannotBeRead) {
    auto &slave = this->api->add_slave();
    create_io(slave);
    create_dc(slave, {DC_SYNC});
    ASSERT_NIL(this->scan_and_activate());
    this->api->reference_clock_result = -EFAULT;
    ASSERT_NIL(this->master.send());
    const auto err = this->master.receive();
    ASSERT_OCCURRED_AS(err, errors::CYCLIC_ERROR);
    EXPECT_EQ(err.data, "failed to read the reference clock: Bad address");
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
            " of module 1 (EL3102): No such file or directory"
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
    ASSERT_OCCURRED_AS(err, errors::STARTUP_PARAMETER_REJECTED);
    EXPECT_EQ(
        err.data,
        "module 1 (TestSlave) rejected startup parameter 0x8000:15 (Filter): Value of "
        "parameter written too high"
    );
    EXPECT_TRUE(this->api->slaves[0].startup_sdos.empty());
}

/// @brief it should report no response, not a rejection, when a slave does not answer a
/// startup parameter write.
TEST_F(IGHMasterTest, ActivateFailsWhenASlaveDoesNotAnswerAStartupParameter) {
    this->api->add_slave().download_aborts[{0x8000, 0x15}] = 0;
    this->configs = {{1, {.startup_parameters = {FILTER}}}};
    const auto err = this->scan_and_activate();
    ASSERT_OCCURRED_AS(err, errors::STARTUP_PARAMETER_ERROR);
    EXPECT_FALSE(err.matches(errors::STARTUP_PARAMETER_REJECTED));
    EXPECT_EQ(
        err.data,
        "module 1 (TestSlave) failed to write startup parameter 0x8000:15 (Filter): no "
        "response"
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
        "failed to add startup parameter 0x8000:15 (Filter) to module 1 (TestSlave)"
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
        "module 1 (TestSlave) rejected startup parameter 0x8000:15 (Filter): Value of "
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
        "the IgH master cannot change 0x8000:15 (Filter) of module 1 (TestSlave) while "
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
        "module 1 (TestSlave) failed to read startup parameter 0x8000:15 (Filter): "
        "Object does not exist in the object dictionary"
    );
}

/// @brief it should refuse to write to a position with no slave on the bus.
TEST_F(IGHMasterTest, WriteRejectsAPositionWithNoSlave) {
    this->api->add_slave();
    ASSERT_NIL(this->scan_and_activate());
    const auto err = this->master.write(2, {FILTER});
    ASSERT_OCCURRED_AS(err, errors::STARTUP_PARAMETER_ERROR);
    EXPECT_EQ(err.data, "no module at position 2");
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

/// @brief it should name the entries of the PDO a startup parameter assigns from the
/// CoE dictionary that IgH read from the slave.
TEST_F(IGHMasterTest, ActivateNamesTheEntriesAStartupParameterAssigns) {
    auto &slave = this->api->add_slave();
    create_assignable_layout(slave);
    slave.dictionary = {
        {.info = {.index = 0x6010, .max_sub_index = 1, .name = "Strain"},
         .entries = {{1, {.name = "Value", .data_type = 0x0008}}}},
    };
    this->configs = {{1, {.startup_parameters = {ASSIGN_INPUTS}}}};
    ASSERT_NIL(this->scan_and_activate());
    const auto &inputs = this->image.slaves[0].properties.input_pdos;
    ASSERT_EQ(inputs.size(), 1);
    EXPECT_EQ(inputs[0].name, "Value");
    EXPECT_EQ(inputs[0].data_type, x::telem::FLOAT32_T);
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
        "failed to read the PDO layout of module 1 (TestSlave) after its startup "
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

/// @brief it should fail when no slave answers its domain, and report no slave as
/// inputless, since no frame came back.
TEST_F(IGHMasterTest, ReceiveFailsWhenNoSlaveAnswers) {
    create_io(this->api->add_slave());
    create_io(this->api->add_slave());
    ASSERT_NIL(this->scan_and_activate());
    for (const auto &slave: this->api->slaves)
        this->api->domain_of(slave).state.wc_state = EC_WC_ZERO;
    const auto err = this->master.receive();
    ASSERT_OCCURRED_AS(err, errors::WORKING_COUNTER_ERROR);
    EXPECT_EQ(err.data, "no modules responded");
    EXPECT_EQ(this->master.shortfall(), 6);
    EXPECT_TRUE(this->master.inputless().empty());
}

/// @brief it should report the shortfall of a slave that gives its inputs but does not
/// take its outputs, and not report it as inputless.
TEST_F(IGHMasterTest, ReceiveReportsThePartOfASlaveThatFellShort) {
    create_io(this->api->add_slave());
    ASSERT_NIL(this->scan_and_activate());
    this->api->domain_of(this->api->slaves[0]).state = {
        .working_counter = 1,
        .wc_state = EC_WC_INCOMPLETE,
    };
    const auto err = this->master.receive();
    ASSERT_OCCURRED_AS(err, errors::WORKING_COUNTER_ERROR);
    EXPECT_EQ(err.data, "incomplete working counter: 1 of 3");
    EXPECT_EQ(this->master.shortfall(), 2);
    EXPECT_TRUE(this->master.inputless().empty());
}

/// @brief it should report each slave whose domain gave nothing as inputless, one with
/// only outputs too, and count its whole share in the shortfall.
TEST_F(IGHMasterTest, ReceiveReportsEachSilentSlaveAsInputless) {
    create_io(this->api->add_slave());
    create_io(this->api->add_slave());
    this->api->add_slave().syncs = {{EC_DIR_OUTPUT, {{0x1600, {{0x7000, 0x01, 8}}}}}};
    ASSERT_NIL(this->scan_and_activate());
    this->api->domain_of(this->api->slaves[1]).state.wc_state = EC_WC_ZERO;
    this->api->domain_of(this->api->slaves[2]).state.wc_state = EC_WC_ZERO;
    const auto err = this->master.receive();
    ASSERT_OCCURRED_AS(err, errors::WORKING_COUNTER_ERROR);
    EXPECT_EQ(err.data, "incomplete working counter: 3 of 7");
    EXPECT_EQ(this->master.shortfall(), 4);
    EXPECT_EQ(
        std::vector(this->master.inputless().begin(), this->master.inputless().end()),
        (std::vector<uint16_t>{2, 3})
    );
    this->api->domain_of(this->api->slaves[1]).state.wc_state = EC_WC_COMPLETE;
    this->api->domain_of(this->api->slaves[2]).state.wc_state = EC_WC_COMPLETE;
    ASSERT_NIL(this->master.receive());
    EXPECT_EQ(this->master.shortfall(), std::nullopt);
    EXPECT_TRUE(this->master.inputless().empty());
}

/// @brief it should give each enabled slave the working counter share that its own
/// domain expects: 2 for its outputs and 1 for its inputs in one LRW datagram, or 1 for
/// the one direction it maps.
TEST_F(IGHMasterTest, ImageGivesTheWorkingCounterShareOfEachSlave) {
    create_io(this->api->add_slave());
    this->api->add_slave().syncs = {{EC_DIR_INPUT, {{0x1A00, {{0x6000, 0x01, 16}}}}}};
    this->api->add_slave().syncs = {{EC_DIR_OUTPUT, {{0x1600, {{0x7000, 0x01, 8}}}}}};
    ASSERT_NIL(this->scan_and_activate());
    EXPECT_EQ(
        this->image.working_counter_shares,
        (std::map<uint16_t, master::WorkingCounterShare>{
            {1, {.outputs = 2, .inputs = 1}},
            {2, {.inputs = 1}},
            {3, {.outputs = 1}},
        })
    );
}

/// @brief it should copy the memory of each domain into the image inputs on receive,
/// and copy the image outputs into each domain before it queues the domain.
TEST_F(IGHMasterTest, CycleCopiesTheProcessDataOfEachDomain) {
    create_io(this->api->add_slave());
    create_io(this->api->add_slave());
    ASSERT_NIL(this->scan_and_activate());
    auto &second = this->api->domain_of(this->api->slaves[1]);
    second.data[2] = 0xAB;
    ASSERT_NIL(this->master.receive());
    EXPECT_EQ(this->image.inputs[8], 0xAB);
    this->image.outputs[6] = 0xCD;
    ASSERT_NIL(this->master.send());
    EXPECT_EQ(second.data[0], 0xCD);
    EXPECT_EQ(this->api->queued, (std::vector<size_t>{0, 1}));
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

/// @brief it should keep the health of a slave whose domain answers, or that has no
/// domain, while IgH scans the bus again, and report a slave whose domain gives nothing
/// as lost.
TEST_F(IGHMasterTest, CheckKeepsTheHealthOfAnAnsweringSlaveDuringAScan) {
    create_io(this->api->add_slave());
    create_io(this->api->add_slave());
    this->api->add_slave();
    ASSERT_NIL(this->scan_and_activate());
    const auto before = this->master.check(this->brk);
    this->api->scanning = true;
    this->api->domain_of(this->api->slaves[1]).state.wc_state = EC_WC_ZERO;
    const auto health = this->master.check(this->brk);
    ASSERT_EQ(health.size(), 3);
    EXPECT_EQ(health[0], before[0]);
    EXPECT_FALSE(health[0].lost);
    EXPECT_EQ(health[0].state, slave::State::OP);
    EXPECT_TRUE(health[1].lost);
    EXPECT_EQ(health[2], before[2]);
}

/// @brief it should reset the master when the latest check found a faulted slave, so
/// IgH configures the slave again.
TEST_F(IGHMasterTest, RecoverResetsTheMasterForAFaultedSlave) {
    auto &slave = this->api->add_slave();
    ASSERT_NIL(this->scan_and_activate());
    slave.info.al_state = EC_AL_STATE_SAFEOP;
    slave.info.error_flag = 1;
    ASSERT_TRUE(this->master.check(this->brk)[0].faulted);
    this->master.recover(this->brk);
    EXPECT_EQ(this->api->resets, 1);
}

/// @brief it should not reset the master when no slave is faulted.
TEST_F(IGHMasterTest, RecoverDoesNothingWithoutAFaultedSlave) {
    this->api->add_slave();
    ASSERT_NIL(this->scan_and_activate());
    ASSERT_FALSE(this->master.check(this->brk)[0].faulted);
    this->master.recover(this->brk);
    EXPECT_EQ(this->api->resets, 0);
}

/// @brief it should wait longer before each next reset while a slave stays faulted.
TEST_F(IGHMasterTest, RecoverBacksOffWhileASlaveStaysFaulted) {
    auto &slave = this->api->add_slave();
    ASSERT_NIL(this->scan_and_activate());
    slave.reaches = EC_AL_STATE_SAFEOP;
    slave.refuses = true;
    slave.info.error_flag = 1;
    const auto check_and_recover = [this] {
        ASSERT_TRUE(this->master.check(this->brk)[0].faulted);
        this->master.recover(this->brk);
    };
    check_and_recover();
    check_and_recover();
    EXPECT_EQ(this->api->resets, 1);
    this->clock.sleep(x::telem::SECOND);
    check_and_recover();
    EXPECT_EQ(this->api->resets, 2);
    this->clock.sleep(x::telem::SECOND);
    check_and_recover();
    EXPECT_EQ(this->api->resets, 2);
    this->clock.sleep(x::telem::SECOND);
    check_and_recover();
    EXPECT_EQ(this->api->resets, 3);
}

/// @brief it should keep backing off while a faulted slave is configured again after a
/// reset, which clears its AL error flag until it refuses again.
TEST_F(IGHMasterTest, RecoverBacksOffWhileAFaultedSlaveIsConfiguredAgain) {
    auto &slave = this->api->add_slave();
    ASSERT_NIL(this->scan_and_activate());
    slave.reaches = EC_AL_STATE_SAFEOP;
    slave.refuses = true;
    slave.info.error_flag = 1;
    slave.reads_to_reconfigure = 1;
    ASSERT_TRUE(this->master.check(this->brk)[0].faulted);
    this->master.recover(this->brk);
    EXPECT_EQ(this->api->resets, 1);
    ASSERT_FALSE(this->master.check(this->brk)[0].faulted);
    ASSERT_TRUE(this->master.check(this->brk)[0].faulted);
    this->master.recover(this->brk);
    EXPECT_EQ(this->api->resets, 1);
}

/// @brief it should start the backoff over once a check finds every slave in OP.
TEST_F(IGHMasterTest, RecoverStartsTheBackoffOverOnceEverySlaveIsInOP) {
    auto &slave = this->api->add_slave();
    ASSERT_NIL(this->scan_and_activate());
    slave.reaches = EC_AL_STATE_SAFEOP;
    slave.refuses = true;
    slave.info.error_flag = 1;
    for (int i = 0; i < 2; i++) {
        ASSERT_TRUE(this->master.check(this->brk)[0].faulted);
        this->master.recover(this->brk);
        this->clock.sleep(x::telem::SECOND);
    }
    EXPECT_EQ(this->api->resets, 2);
    slave.recovers_on_reset = true;
    ASSERT_FALSE(this->master.check(this->brk)[0].faulted);
    slave.info.error_flag = 1;
    ASSERT_TRUE(this->master.check(this->brk)[0].faulted);
    this->master.recover(this->brk);
    EXPECT_EQ(this->api->resets, 3);
}

/// @brief it should back off after a failed reset as after one that worked.
TEST_F(IGHMasterTest, RecoverBacksOffAfterAFailedReset) {
    auto &slave = this->api->add_slave();
    ASSERT_NIL(this->scan_and_activate());
    slave.info.error_flag = 1;
    this->api->reset_result = -EIO;
    for (int i = 0; i < 2; i++) {
        ASSERT_TRUE(this->master.check(this->brk)[0].faulted);
        this->master.recover(this->brk);
    }
    EXPECT_EQ(this->api->resets, 1);
}

/// @brief it should not reset the master once the breaker stops.
TEST_F(IGHMasterTest, RecoverDoesNothingOnceTheBreakerStops) {
    auto &slave = this->api->add_slave();
    ASSERT_NIL(this->scan_and_activate());
    slave.info.error_flag = 1;
    ASSERT_TRUE(this->master.check(this->brk)[0].faulted);
    this->brk.stop();
    this->master.recover(this->brk);
    EXPECT_EQ(this->api->resets, 0);
}

/// @brief it should keep the reset backoff while IgH scans the bus and a faulted slave
/// keeps its health.
TEST_F(IGHMasterTest, CheckKeepsTheResetBackoffThroughAScan) {
    auto &slave = this->api->add_slave();
    create_io(slave);
    ASSERT_NIL(this->scan_and_activate());
    slave.reaches = EC_AL_STATE_SAFEOP;
    slave.refuses = true;
    slave.info.error_flag = 1;
    ASSERT_TRUE(this->master.check(this->brk)[0].faulted);
    this->master.recover(this->brk);
    EXPECT_EQ(this->api->resets, 1);
    this->api->scanning = true;
    ASSERT_TRUE(this->master.check(this->brk)[0].faulted);
    this->api->scanning = false;
    ASSERT_TRUE(this->master.check(this->brk)[0].faulted);
    this->master.recover(this->brk);
    EXPECT_EQ(this->api->resets, 1);
}

/// @brief it should start the reset backoff over after a deactivate.
TEST_F(IGHMasterTest, DeactivateStartsTheResetBackoffOver) {
    auto &slave = this->api->add_slave();
    for (int i = 1; i <= 2; i++) {
        ASSERT_NIL(this->scan_and_activate());
        slave.info.al_state = EC_AL_STATE_SAFEOP;
        slave.info.error_flag = 1;
        ASSERT_TRUE(this->master.check(this->brk)[0].faulted);
        this->master.recover(this->brk);
        EXPECT_EQ(this->api->resets, i);
        this->master.deactivate();
    }
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
