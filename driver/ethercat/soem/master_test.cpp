// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <atomic>
#include <chrono>
#include <cstdint>
#include <cstdlib>
#include <functional>
#include <optional>
#include <thread>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/esi/known_devices.h"
#include "driver/ethercat/soem/master.h"
#include "driver/ethercat/soem/mock_api.h"

namespace driver::ethercat::soem {
namespace {
/// @brief the bus cycle the specs activate the master with.
const auto PERIOD = x::telem::MILLISECOND;

/// @brief a bus cycle that sends nothing. The mock API needs no process data to reach
/// OP.
void no_cycle() {}

/// @brief returns the little-endian bytes of value.
template<typename T>
std::vector<uint8_t> bytes(const T value) {
    std::vector<uint8_t> out(sizeof(T));
    for (size_t i = 0; i < sizeof(T); ++i)
        out[i] = static_cast<uint8_t>(static_cast<uint64_t>(value) >> (8 * i));
    return out;
}

/// @brief the only mode of a module that runs in DC mode alone: SYNC0 of 1 us and
/// SYNC1 of 1000 bus cycles.
const dc::OpMode DC_ONLY{
    .name = "DC",
    .assign_activate = 0x0700,
    .sync0_cycle = 1000,
    .sync1_factor = -1000,
};
const dc::OpMode SM{.name = "SM-Synchron", .sync0_factor = 1};
/// @brief a DC mode with a SYNC0 of 1 s and a SYNC1 at each SYNC0 event.
const dc::OpMode DC_SECOND{
    .name = "DC",
    .assign_activate = 0x0700,
    .sync0_cycle = 1000000000,
};
}

class SOEMMasterTest : public ::testing::Test {
protected:
    MockAPI *mock{};
    std::unique_ptr<Master> master;
    x::breaker::Breaker brk;
    /// @brief the bus cycle that activate() starts the master with.
    x::telem::TimeSpan period = PERIOD;
    /// @brief the time from the send of a frame to its receive in cycle.
    std::chrono::microseconds cycle_gap{100};

    void SetUp() override {
        auto api = std::make_unique<MockAPI>();
        this->mock = api.get();
        this->master = std::make_unique<Master>(
            std::move(api),
            "eth0",
            Config{
                .pre_op_timeout = x::telem::MILLISECOND * 100,
                .state_change_timeout = x::telem::MILLISECOND * 100,
                .dc_settle_timeout = x::telem::MILLISECOND * 50,
            }
        );
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

    /// @brief writes modes to the SII DC category that every slave reads.
    void set_op_modes(const std::vector<dc::OpMode> &modes) {
        constexpr uint16_t start = 0x200;
        this->mock->sii_categories = {{60, start}};
        const auto put = [this](const size_t addr, const uint64_t value, const int n) {
            for (int i = 0; i < n; ++i)
                this->mock
                    ->sii_bytes[static_cast<uint16_t>(addr + i)] = static_cast<uint8_t>(
                    value >> (8 * i)
                );
        };
        put(start, modes.size() * 12, 2);
        for (size_t i = 0; i < modes.size(); ++i) {
            const auto &m = modes[i];
            const size_t a = start + 2 + i * 24;
            put(a, m.sync0_cycle, 4);
            put(a + 4, static_cast<uint32_t>(m.sync0_shift), 4);
            put(a + 8, static_cast<uint32_t>(m.sync1_shift), 4);
            put(a + 12, static_cast<uint16_t>(m.sync1_factor), 2);
            put(a + 14, m.assign_activate, 2);
            put(a + 16, static_cast<uint16_t>(m.sync0_factor), 2);
            if (m.name.empty()) continue;
            put(a + 18, i + 1, 1);
            this->mock->sii_strings[static_cast<uint16_t>(i + 1)] = m.name;
        }
    }

    /// @brief initializes the master with one DC slave named 16xAO that lists modes,
    /// and gives it a DC system time of 5 s.
    void initialize_with_dc_slave(const std::vector<dc::OpMode> &modes) {
        MockSlaveInfo slave;
        slave.state = EC_STATE_PRE_OP;
        slave.hasdc = true;
        slave.name = "16xAO";
        this->add_slave(slave);
        this->set_op_modes(modes);
        this->mock->register_values[{1, 0x0910}] = bytes<uint64_t>(5'000'000'000);
        ASSERT_NIL(this->master->initialize());
        this->mock->obytes = 2;
        this->mock->ibytes = 4;
        this->mock->outputs_wkc = 1;
        this->mock->inputs_wkc = 1;
        this->mock->receive_return = 3;
        this->mock->state_change_requests.clear();
    }

    /// @brief exchanges process data until cycling turns false. Like the engine, it
    /// receives each frame cycle_gap after it sends it.
    void cycle(const std::atomic<bool> &cycling) {
        while (cycling.load()) {
            static_cast<void>(this->master->send());
            std::this_thread::sleep_for(this->cycle_gap);
            static_cast<void>(this->master->receive());
        }
    }

    /// @brief activates the master with a bus cycle that exchanges process data on its
    /// own thread until activate returns. Calls on_start as the cycle starts.
    x::errors::Error activate(const std::function<void()> &on_start = nullptr) {
        std::atomic<bool> cycling = true;
        std::thread cycle;
        auto err = this->master->activate(this->period, [&] {
            if (on_start) on_start();
            cycle = std::thread([&] { this->cycle(cycling); });
        });
        cycling = false;
        if (cycle.joinable()) cycle.join();
        return err;
    }

    /// @brief checks the slaves while a bus cycle exchanges process data on its own
    /// thread.
    std::vector<slave::Health> check_while_cycling() {
        std::atomic<bool> cycling = true;
        std::thread cycle([&] { this->cycle(cycling); });
        auto health = this->master->check(this->brk);
        cycling = false;
        cycle.join();
        return health;
    }

    /// @brief activates the master with two DC slaves named 16xAO that list modes. The
    /// first is the reference clock, and the second is 300 ns behind it.
    void activate_two_dc_slaves(const std::vector<dc::OpMode> &modes) {
        MockSlaveInfo slave;
        slave.state = EC_STATE_PRE_OP;
        slave.hasdc = true;
        slave.name = "16xAO";
        this->add_slave(slave);
        slave.pdelay = 300;
        this->add_slave(slave);
        this->set_op_modes(modes);
        ASSERT_NIL(this->master->initialize());
        this->mock->hasdc = true;
        this->activate_master();
        this->mock->register_writes.clear();
    }

    /// @brief returns the offset (0x0920) that the master wrote to a slave.
    int64_t written_offset(const uint16_t slave) const {
        const auto it = std::ranges::find_if(
            this->mock->register_writes,
            [slave](const RegisterWrite &w) {
                return w.slave == slave && w.address == 0x0920;
            }
        );
        EXPECT_NE(it, this->mock->register_writes.end());
        if (it == this->mock->register_writes.end()) return 0;
        int64_t value = 0;
        for (size_t i = 0; i < it->data.size(); ++i)
            value |= static_cast<int64_t>(it->data[i]) << (8 * i);
        return value;
    }

    void activate_master() {
        this->mock->obytes = 2;
        this->mock->ibytes = 4;
        this->mock->outputs_wkc = 1;
        this->mock->inputs_wkc = 1;
        this->mock->receive_return = 3;
        ASSERT_NIL(this->activate());
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

/// @brief it should return a no slaves error that asks for a cable and power check, and
/// close the socket, when no slave answers.
TEST_F(SOEMMasterTest, InitializeReportsAnInterfaceWithNoSlaves) {
    this->mock->config_init_return = 0;
    const auto err = this->master->initialize();
    ASSERT_OCCURRED_AS(err, errors::NO_SLAVES_FOUND);
    EXPECT_EQ(
        err.data,
        "no EtherCAT slaves found on eth0. Check the cable and the power of the first "
        "device."
    );
    EXPECT_TRUE(this->mock->close_called);
}

/// @brief it should name the DewesoftX filter and the Windows name of the adapter when
/// the filter is bound to an interface with no slaves.
TEST_F(SOEMMasterTest, InitializeNamesTheDewesoftFilterOnAnInterfaceWithNoSlaves) {
    this->mock->config_init_return = 0;
    this->mock->filtered_interfaces.insert("eth0");
    this->mock->adapter_names["eth0"] = "Ethernet 2";
    const auto err = this->master->initialize();
    ASSERT_OCCURRED_AS(err, errors::NO_SLAVES_FOUND);
    EXPECT_EQ(
        err.data,
        "no EtherCAT slaves found on eth0. The DewesoftX \"EthercatDAQ Filter\" is on "
        "adapter \"Ethernet 2\". After DewesoftX starts, the filter blocks EtherCAT "
        "replies, also after DewesoftX closes. Clear it in the properties of "
        "\"Ethernet 2\" (Settings > Network & internet > Advanced network settings > "
        "More network adapter options, then right-click the adapter > Properties)."
    );
    EXPECT_TRUE(this->mock->close_called);
}

/// @brief it should name the interface when the Windows name of an adapter with the
/// DewesoftX filter cannot be read.
TEST_F(SOEMMasterTest, InitializeNamesTheInterfaceWhenTheFilteredAdapterHasNoName) {
    this->mock->config_init_return = 0;
    this->mock->filtered_interfaces.insert("eth0");
    const auto err = this->master->initialize();
    ASSERT_OCCURRED_AS(err, errors::NO_SLAVES_FOUND);
    EXPECT_EQ(
        err.data,
        "no EtherCAT slaves found on eth0. The DewesoftX \"EthercatDAQ Filter\" is on "
        "adapter \"eth0\". After DewesoftX starts, the filter blocks EtherCAT replies, "
        "also after DewesoftX closes. Clear it in the properties of \"eth0\" (Settings "
        "> Network & internet > Advanced network settings > More network adapter "
        "options, then right-click the adapter > Properties)."
    );
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

/// @brief it should wait for the slaves to reach PRE-OP before it reads PDOs from the
/// CoE object dictionary.
TEST_F(SOEMMasterTest, CoEDiscoveryWaitsForPreOp) {
    MockSlaveInfo slave;
    slave.state = EC_STATE_INIT;
    slave.bus_state = EC_STATE_INIT;
    slave.mbx_proto = ECT_MBXPROT_COE;
    this->add_slave(slave);
    this->mock->sdo_objects = {
        {{ECT_SDO_TXPDOASSIGN, 0}, 1},
        {{ECT_SDO_TXPDOASSIGN, 1}, 0x1A00},
        {{0x1A00, 0}, 1},
        {{0x1A00, 1}, 0x60000110},
        {{ECT_SDO_RXPDOASSIGN, 0}, 0},
    };

    ASSERT_NIL(this->master->initialize());

    const auto slaves = this->master->slaves();
    ASSERT_EQ(slaves.size(), 1);
    EXPECT_TRUE(slaves[0].status.pdo_discovery_error.empty());
    const auto &inputs = slaves[0].properties.input_pdos;
    ASSERT_EQ(inputs.size(), 1);
    EXPECT_EQ(inputs[0].index, 0x6000);
    EXPECT_EQ(inputs[0].sub_index, 1);
    EXPECT_EQ(inputs[0].bit_length, 16);
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

/// @brief it should give the EEPROM back to the slave after it reads PDOs from the SII.
TEST_F(SOEMMasterTest, SIIDiscoveryGivesTheEEPROMBackToTheSlave) {
    MockSlaveInfo slave;
    slave.state = EC_STATE_PRE_OP;
    this->add_slave(slave);
    constexpr uint16_t start = 0x40;
    this->mock->sii_categories = {{ECT_SII_PDO, start}};
    // One TxPDO, 0x1A00, with entry 0x6000:01 (INTEGER16).
    const std::vector<uint8_t> category =
        {8, 0, 0x00, 0x1A, 1, 3, 0, 0, 0, 0, 0x00, 0x60, 1, 0, 0x03, 16, 0, 0};
    for (size_t i = 0; i < category.size(); ++i)
        this->mock->sii_bytes[start + i] = category[i];

    ASSERT_NIL(this->master->initialize());

    ASSERT_EQ(this->master->slaves()[0].properties.input_pdos.size(), 1);
    EXPECT_TRUE(this->mock->eeprom_held.empty());
}

/// @brief it should read the synchronization modes a slave lists in its SII DC
/// category, in SII order.
TEST_F(SOEMMasterTest, InitializeReadsOpModesFromSII) {
    MockSlaveInfo slave;
    slave.state = EC_STATE_PRE_OP;
    this->add_slave(slave);
    constexpr uint16_t start = 0x200;
    this->mock->sii_categories = {{60, start}};
    // The DC category of a slave on the test bench: SM, DC with a 1 s SYNC0, and DC
    // with a 1 us SYNC0 and a SYNC1 of 1000 bus cycles.
    const std::vector<uint8_t> category = {
        36, 0, 0,    0,    0, 0,    0,    0, 0, 0, 0, 0,    0,    0,    0,
        0,  0, 0,    1,    0, 3,    0,    0, 0, 0, 0, 0,    0xCA, 0x9A, 0x3B,
        0,  0, 0,    0,    0, 0,    0,    0, 0, 0, 0, 0x07, 0,    0,    4,
        0,  0, 0,    0,    0, 0xE8, 0x03, 0, 0, 0, 0, 0,    0,    0,    0,
        0,  0, 0x18, 0xFC, 0, 0x07, 0,    0, 5, 0, 0, 0,    0,    0,
    };
    for (size_t i = 0; i < category.size(); ++i)
        this->mock->sii_bytes[start + i] = category[i];
    this->mock->sii_strings = {{3, "SM-Synchron"}, {4, "DC"}, {5, "DC-bus1"}};

    ASSERT_NIL(this->master->initialize());

    const std::vector<dc::OpMode> expected = {
        {.name = "SM-Synchron", .sync0_factor = 1},
        {.name = "DC", .assign_activate = 0x0700, .sync0_cycle = 1000000000},
        {.name = "DC-bus1",
         .assign_activate = 0x0700,
         .sync0_cycle = 1000,
         .sync1_factor = -1000},
    };
    EXPECT_EQ(this->master->slaves()[0].properties.op_modes, expected);
}

/// @brief it should give the EEPROM back to the slave after it reads the OpModes.
TEST_F(SOEMMasterTest, InitializeGivesTheEEPROMBackAfterReadingOpModes) {
    MockSlaveInfo slave;
    slave.state = EC_STATE_PRE_OP;
    this->add_slave(slave);
    constexpr uint16_t start = 0x200;
    this->mock->sii_categories = {{60, start}};
    // One SM mode with every field 0.
    std::vector<uint8_t> category(26, 0);
    category[0] = 12;
    for (size_t i = 0; i < category.size(); ++i)
        this->mock->sii_bytes[start + i] = category[i];

    ASSERT_NIL(this->master->initialize());

    ASSERT_EQ(this->master->slaves()[0].properties.op_modes.size(), 1);
    EXPECT_TRUE(this->mock->eeprom_held.empty());
}

/// @brief it should read negative shift times and leave the name of a mode empty when
/// the SII gives none.
TEST_F(SOEMMasterTest, InitializeReadsSignedShiftsAndUnnamedOpModes) {
    MockSlaveInfo slave;
    slave.state = EC_STATE_PRE_OP;
    this->add_slave(slave);
    constexpr uint16_t start = 0x200;
    this->mock->sii_categories = {{60, start}};
    // One DC mode: SYNC0 = bus cycle, SYNC0 shift -100 us, SYNC1 shift 25 us, no name.
    const std::vector<uint8_t> category = {
        12, 0,    0,    0,    0,    0,    0x60, 0x79, 0xFE, 0xFF, 0xA8, 0x61, 0,
        0,  0x01, 0x00, 0x00, 0x03, 0x01, 0x00, 0,    0,    0,    0,    0,    0,
    };
    for (size_t i = 0; i < category.size(); ++i)
        this->mock->sii_bytes[start + i] = category[i];

    ASSERT_NIL(this->master->initialize());

    const std::vector<dc::OpMode> expected = {{
        .assign_activate = 0x0300,
        .sync0_factor = 1,
        .sync0_shift = -100000,
        .sync1_factor = 1,
        .sync1_shift = 25000,
    }};
    EXPECT_EQ(this->master->slaves()[0].properties.op_modes, expected);
}

/// @brief it should give a slave no synchronization modes when its SII has no DC
/// category.
TEST_F(SOEMMasterTest, InitializeGivesNoOpModesWithoutDCCategory) {
    this->initialize_with_slave();
    EXPECT_TRUE(this->master->slaves()[0].properties.op_modes.empty());
}

////////////////////// Activation //////////////////////

TEST_F(SOEMMasterTest, ActivateHappyPath) {
    this->initialize_with_slave();
    this->activate_master();
}

TEST_F(SOEMMasterTest, ActivateNotInitialized) {
    ASSERT_OCCURRED_AS(this->activate(), errors::ACTIVATION_ERROR);
}

TEST_F(SOEMMasterTest, ActivateAlreadyActivated) {
    this->initialize_with_slave();
    this->activate_master();
    ASSERT_OCCURRED_AS(this->activate(), errors::ACTIVATION_ERROR);
}

TEST_F(SOEMMasterTest, ActivateConfigMapFails) {
    this->initialize_with_slave();
    this->mock->config_map_return = 0;
    ASSERT_OCCURRED_AS(this->activate(), errors::ACTIVATION_ERROR);
}

/// @brief it should return a refusal with the AL status as soon as a slave refuses
/// SAFE-OP, without waiting for the state change timeout.
TEST_F(SOEMMasterTest, ActivateSafeOpRefusalReturnsAtOnce) {
    this->initialize_with_slave();
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    this->mock->statecheck_overrides[EC_STATE_SAFE_OP] = EC_STATE_PRE_OP +
                                                         EC_STATE_ERROR;
    this->mock->slave_infos[0].ALstatuscode = 0x001D;
    this->mock->readstate_count = 0;
    const auto err = this->activate();
    ASSERT_OCCURRED_AS(err, errors::STATE_CHANGE_REFUSED);
    EXPECT_EQ(
        err.data,
        "failed to reach SAFE-OP: slave 1 (TestSlave) is in PRE-OP: Invalid output "
        "configuration"
    );
    EXPECT_EQ(this->mock->readstate_count, 1);
}

/// @brief it should not report a refusal or its AL status when a slave stays in PRE-OP
/// without the error flag, even if an old AL status code remains.
TEST_F(SOEMMasterTest, ActivateSafeOpTimeoutIsNotARefusal) {
    this->initialize_with_slave();
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    this->mock->statecheck_overrides[EC_STATE_SAFE_OP] = EC_STATE_PRE_OP;
    this->mock->slave_infos[0].ALstatuscode = 0x001D;
    const auto err = this->activate();
    ASSERT_OCCURRED_AS(err, errors::STATE_CHANGE_ERROR);
    EXPECT_FALSE(err.matches(errors::STATE_CHANGE_REFUSED));
    EXPECT_EQ(err.data, "failed to reach SAFE-OP: slave 1 (TestSlave) is in PRE-OP");
}

/// @brief it should name only the slave that refused SAFE-OP, from one read of the
/// slave states.
TEST_F(SOEMMasterTest, ActivateNamesOnlyTheRefusingSlave) {
    MockSlaveInfo pending;
    pending.state = EC_STATE_PRE_OP;
    pending.name = "PendingSlave";
    this->add_slave(pending);
    MockSlaveInfo refusing;
    refusing.state = EC_STATE_PRE_OP;
    refusing.name = "RefusingSlave";
    refusing.ALstatuscode = 0x0026;
    this->add_slave(refusing);
    ASSERT_NIL(this->master->initialize());
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    this->mock->statecheck_overrides[EC_STATE_SAFE_OP] = EC_STATE_PRE_OP +
                                                         EC_STATE_ERROR;
    this->mock->readstate_count = 0;
    const auto err = this->activate();
    ASSERT_OCCURRED_AS(err, errors::STATE_CHANGE_REFUSED);
    EXPECT_EQ(
        err.data,
        "failed to reach SAFE-OP: slave 2 (RefusingSlave) is in PRE-OP: Inconsistent "
        "settings"
    );
    EXPECT_EQ(this->mock->readstate_count, 1);
}

/// @brief it should add the DC sync advice once when slaves refuse SAFE-OP with
/// "Invalid DC SYNC configuration".
TEST_F(SOEMMasterTest, ActivateDCSyncRefusalAddsAdvice) {
    MockSlaveInfo first;
    first.state = EC_STATE_PRE_OP;
    first.name = "FirstSlave";
    first.ALstatuscode = 0x0030;
    this->add_slave(first);
    MockSlaveInfo second = first;
    second.name = "SecondSlave";
    this->add_slave(second);
    ASSERT_NIL(this->master->initialize());
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    this->mock->statecheck_overrides[EC_STATE_SAFE_OP] = EC_STATE_PRE_OP +
                                                         EC_STATE_ERROR;
    const auto err = this->activate();
    ASSERT_OCCURRED_AS(err, errors::STATE_CHANGE_REFUSED);
    EXPECT_EQ(
        err.data,
        "failed to reach SAFE-OP: slave 1 (FirstSlave) is in PRE-OP: Invalid DC SYNC "
        "configuration; slave 2 (SecondSlave) is in PRE-OP: Invalid DC SYNC "
        "configuration. A slave that reports \"Invalid DC SYNC configuration\" "
        "expects a synchronization mode that the Driver did not set up. The Driver "
        "uses SM sync when a slave lists an SM mode, and DC sync when it lists only DC "
        "modes. Set the slave to the mode that the Driver uses (in DewesoftX for "
        "Dewesoft modules), or disable the slave so the rest of the bus can run."
    );
}

/// @brief it should return a refusal as soon as a slave refuses OP.
TEST_F(SOEMMasterTest, ActivateOpRefusalReturnsAtOnce) {
    this->initialize_with_slave();
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    this->mock->statecheck_overrides[EC_STATE_OPERATIONAL] = EC_STATE_SAFE_OP +
                                                             EC_STATE_ERROR;
    this->mock->slave_infos[0].ALstatuscode = 0x0025;
    this->mock->readstate_count = 0;
    const auto err = this->activate();
    ASSERT_OCCURRED_AS(err, errors::STATE_CHANGE_REFUSED);
    EXPECT_EQ(
        err.data,
        "failed to reach OP: slave 1 (TestSlave) is in SAFE-OP: Invalid output mapping"
    );
    // One read each for SAFE-OP and OP.
    EXPECT_EQ(this->mock->readstate_count, 2);
}

/// @brief it should return a timing fault at once, with its AL status, as an error that
/// is not a refusal, so that the start is retried.
TEST_F(SOEMMasterTest, ActivateOpTimingFaultIsNotARefusal) {
    this->initialize_with_slave();
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    this->mock->statecheck_overrides[EC_STATE_OPERATIONAL] = EC_STATE_SAFE_OP +
                                                             EC_STATE_ERROR;
    this->mock->slave_infos[0].ALstatuscode = 0x001B;
    this->mock->readstate_count = 0;
    const auto err = this->activate();
    ASSERT_OCCURRED_AS(err, errors::STATE_CHANGE_ERROR);
    EXPECT_FALSE(err.matches(errors::STATE_CHANGE_REFUSED));
    EXPECT_EQ(
        err.data,
        "failed to reach OP: slave 1 (TestSlave) is in SAFE-OP: Sync manager watchdog"
    );
    EXPECT_EQ(this->mock->readstate_count, 2);
}

/// @brief it should not count a slave that asks to go back to PRE-OP as a refusal,
/// because the retry takes it back through INIT.
TEST_F(SOEMMasterTest, ActivateSafeOpRestartRequestIsNotARefusal) {
    this->initialize_with_slave();
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    this->mock->statecheck_overrides[EC_STATE_SAFE_OP] = EC_STATE_PRE_OP +
                                                         EC_STATE_ERROR;
    this->mock->slave_infos[0].ALstatuscode = 0x0022;
    const auto err = this->activate();
    ASSERT_OCCURRED_AS(err, errors::STATE_CHANGE_ERROR);
    EXPECT_FALSE(err.matches(errors::STATE_CHANGE_REFUSED));
    EXPECT_EQ(
        err.data,
        "failed to reach SAFE-OP: slave 1 (TestSlave) is in PRE-OP: Slave needs PREOP"
    );
}

TEST_F(SOEMMasterTest, ActivateOpTransitionFails) {
    this->initialize_with_slave();
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    this->mock->statecheck_overrides[EC_STATE_OPERATIONAL] = EC_STATE_SAFE_OP;
    ASSERT_OCCURRED_AS(this->activate(), errors::STATE_CHANGE_ERROR);
}

/// @brief it should start the bus cycle once send() works and before it requests
/// SAFE-OP.
TEST_F(SOEMMasterTest, ActivateStartsTheCycleBeforeSAFEOP) {
    this->initialize_with_slave();
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    this->mock->statecheck_pending_polls[EC_STATE_OPERATIONAL] = 3;
    std::optional<std::vector<std::pair<uint16_t, uint16_t>>> requests;
    ASSERT_NIL(this->activate([&] {
        ASSERT_NIL(this->master->send());
        requests = this->mock->state_change_requests;
    }));
    ASSERT_TRUE(requests.has_value());
    EXPECT_TRUE(requests.value().empty());
}

/// @brief it should exchange no process data until every slave reaches SAFE-OP, and
/// send the reference clock time until then.
TEST_F(SOEMMasterTest, ActivateHoldsProcessDataUntilSAFEOP) {
    this->initialize_with_slave();
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    this->mock->hasdc = true;
    this->mock->statecheck_pending_polls[EC_STATE_SAFE_OP] = 3;
    ASSERT_NIL(this->activate());
    EXPECT_GT(this->mock->sync_dc_count, 0);
    ASSERT_FALSE(this->mock->send_states.empty());
    for (const auto state: this->mock->send_states)
        EXPECT_GE(state, EC_STATE_SAFE_OP);
}

/// @brief it should report a cycle before SAFE-OP as a working counter shortfall, since
/// it exchanges no process data.
TEST_F(SOEMMasterTest, ReceiveBeforeSAFEOPReportsAShortfall) {
    this->initialize_with_slave();
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    x::errors::Error err;
    int receives = -1;
    ASSERT_NIL(this->activate([&] {
        ASSERT_NIL(this->master->send());
        err = this->master->receive();
        receives = this->mock->receive_count;
    }));
    ASSERT_OCCURRED_AS(err, errors::WORKING_COUNTER_ERROR);
    EXPECT_EQ(err.data, "no process data passes the slaves until they reach SAFE-OP");
    EXPECT_EQ(receives, 0);
}

/// @brief it should fail a send before SAFE-OP when the reference clock frame does not
/// come back.
TEST_F(SOEMMasterTest, SendFailsWhenTheReferenceClockFrameIsLost) {
    this->initialize_with_slave();
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    this->mock->hasdc = true;
    this->mock->sync_dc_return = 0;
    x::errors::Error err;
    static_cast<void>(this->master->activate(PERIOD, [&] {
        err = this->master->send();
    }));
    ASSERT_OCCURRED_AS(err, errors::CYCLIC_ERROR);
    EXPECT_EQ(err.data, "reference clock frame failed");
}

/// @brief it should map the slaves with no SAFE-OP request from SOEM, so the slaves
/// enter SAFE-OP only once the bus cycles, and let the next config_init request PRE-OP.
TEST_F(SOEMMasterTest, ActivateMapsTheSlavesWithoutASAFEOPRequest) {
    this->initialize_with_slave();
    this->activate_master();
    EXPECT_EQ(this->mock->manual_at_map, true);
    EXPECT_FALSE(this->mock->manual_state_change);
}

/// @brief it should fail activation without a request for OP when no frame passes the
/// slaves in SAFE-OP, since a slave with outputs refuses OP until one does.
TEST_F(SOEMMasterTest, ActivateRequestsOPOnlyAfterAFramePassesInSAFEOP) {
    this->initialize_with_slave();
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    this->mock->state_change_requests.clear();
    const auto err = this->master->activate(PERIOD, no_cycle);
    ASSERT_OCCURRED_AS(err, errors::STATE_CHANGE_ERROR);
    EXPECT_EQ(
        err.data,
        "failed to reach OP: no process data passed the slaves in SAFE-OP"
    );
    EXPECT_EQ(
        this->mock->state_change_requests,
        (std::vector<std::pair<uint16_t, uint16_t>>{{1, EC_STATE_SAFE_OP}})
    );
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
    ASSERT_NIL(this->activate());
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
    ASSERT_NIL(this->activate());
    const std::vector<std::pair<uint16_t, uint16_t>> expected = {
        {1, EC_STATE_SAFE_OP},
        {1, EC_STATE_OPERATIONAL},
    };
    EXPECT_EQ(this->mock->state_change_requests, expected);
}

/// @brief it should write zero to 0x0980 and then to 0x0981 on each enabled slave with
/// a distributed clock, before it maps the slaves and so requests SAFE-OP.
TEST_F(SOEMMasterTest, ActivateClearsDCActivationBeforeMapping) {
    MockSlaveInfo dc;
    dc.state = EC_STATE_PRE_OP;
    dc.hasdc = true;
    this->add_slave(dc);
    MockSlaveInfo plain;
    plain.state = EC_STATE_PRE_OP;
    this->add_slave(plain);
    this->add_slave(dc);
    ASSERT_NIL(this->master->initialize());
    this->master->set_slave_enabled(3, false);
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    ASSERT_NIL(this->activate());
    const std::vector<RegisterWrite> expected = {
        {.slave = 1, .address = 0x0980, .data = {0}},
        {.slave = 1, .address = 0x0981, .data = {0}},
    };
    EXPECT_EQ(this->mock->register_writes, expected);
}

/// @brief it should fail activation, before it maps the slaves, when a slave does not
/// take the write that clears its DC activation.
TEST_F(SOEMMasterTest, ActivateFailsWhenDCActivationCannotBeCleared) {
    MockSlaveInfo dc;
    dc.state = EC_STATE_PRE_OP;
    dc.hasdc = true;
    dc.name = "EL3104";
    this->add_slave(dc);
    ASSERT_NIL(this->master->initialize());
    this->mock->write_register_return = 0;
    const auto err = this->activate();
    ASSERT_OCCURRED_AS(err, errors::ACTIVATION_ERROR);
    EXPECT_EQ(err.data, "slave 1 (EL3104) did not take a write to DC register 0x0980");
    EXPECT_EQ(this->mock->mapped_group, std::nullopt);
}

/// @brief it should start SYNC0 and SYNC1 on a slave that lists only a DC mode, before
/// it requests SAFE-OP, with the first SYNC0 event on a whole SYNC0 cycle at least
/// 100 ms after the system time of the slave.
TEST_F(SOEMMasterTest, ActivateStartsSyncOnASlaveWithOnlyADCMode) {
    this->initialize_with_dc_slave({DC_ONLY});
    ASSERT_NIL(this->activate());
    const std::vector<RegisterWrite> expected = {
        {.slave = 1, .address = 0x0980, .data = {0}},
        {.slave = 1, .address = 0x0981, .data = {0}},
        {.slave = 1, .address = 0x0980, .data = {0}, .mapped = true},
        {.slave = 1, .address = 0x09A0, .data = bytes<uint32_t>(1000), .mapped = true},
        {.slave = 1,
         .address = 0x09A4,
         .data = bytes<uint32_t>(999'999'000),
         .mapped = true},
        {.slave = 1,
         .address = 0x0990,
         .data = bytes<uint64_t>(5'100'001'000),
         .mapped = true},
        {.slave = 1, .address = 0x0981, .data = {0x07}, .mapped = true},
    };
    EXPECT_EQ(this->mock->register_writes, expected);
    EXPECT_EQ(this->mock->requests_before_write, std::vector<size_t>(7, 0));
}

/// @brief it should add the SYNC0 shift of the mode to the start time.
TEST_F(SOEMMasterTest, ActivateShiftsTheFirstSYNC0Event) {
    auto mode = DC_SECOND;
    mode.sync0_shift = 250'000;
    this->initialize_with_dc_slave({mode});
    ASSERT_NIL(this->activate());
    ASSERT_EQ(this->mock->register_writes.size(), 7);
    EXPECT_EQ(this->mock->register_writes[5].data, bytes<uint64_t>(6'000'250'000));
}

/// @brief it should put the first SYNC0 event at least 100 ms after the system time of
/// the slave when the shift of the mode is negative.
TEST_F(SOEMMasterTest, ActivateKeepsTheFirstSYNC0EventAheadWithANegativeShift) {
    this->initialize_with_dc_slave({{
        .name = "DC",
        .assign_activate = 0x0300,
        .sync0_cycle = 1'000'000,
        .sync0_shift = -200'000'000,
    }});
    ASSERT_NIL(this->activate());
    ASSERT_EQ(this->mock->register_writes.size(), 7);
    EXPECT_EQ(this->mock->register_writes[5].data, bytes<uint64_t>(5'101'000'000));
}

/// @brief it should leave DC sync off on a slave that lists an SM mode.
TEST_F(SOEMMasterTest, ActivateLeavesSyncOffOnASlaveWithAnSMMode) {
    this->initialize_with_dc_slave({SM, DC_SECOND});
    ASSERT_NIL(this->activate());
    const std::vector<RegisterWrite> expected = {
        {.slave = 1, .address = 0x0980, .data = {0}},
        {.slave = 1, .address = 0x0981, .data = {0}},
    };
    EXPECT_EQ(this->mock->register_writes, expected);
}

/// @brief it should leave DC sync off on a disabled slave.
TEST_F(SOEMMasterTest, ActivateLeavesSyncOffOnADisabledSlave) {
    this->initialize_with_dc_slave({DC_ONLY});
    this->master->set_slave_enabled(1, false);
    ASSERT_NIL(this->activate());
    EXPECT_TRUE(this->mock->register_writes.empty());
}

/// @brief it should fail activation before it maps the slaves when the mode of a slave
/// gives values its DC registers cannot hold at the bus cycle.
TEST_F(SOEMMasterTest, ActivateFailsWhenTheDCModeCannotRunAtThePeriod) {
    this->initialize_with_dc_slave({DC_ONLY});
    this->period = x::telem::MILLISECOND * 10;
    const auto err = this->activate();
    ASSERT_OCCURRED_AS(err, errors::SLAVE_CONFIG_ERROR);
    EXPECT_EQ(
        err.data,
        "slave 1 (16xAO) cannot run its DC mode at a bus cycle of " +
            this->period.to_string() + ": " +
            dc::resolve(DC_ONLY, this->period).second.data
    );
    EXPECT_EQ(this->mock->mapped_group, std::nullopt);
}

/// @brief it should write only 0x0980 on a slave whose mode gives the sync unit to the
/// PDI.
TEST_F(SOEMMasterTest, ActivateLeavesASyncUnitThePDIOwns) {
    this->initialize_with_dc_slave(
        {{.name = "DC", .assign_activate = 0x0301, .sync0_factor = 1}}
    );
    ASSERT_NIL(this->activate());
    ASSERT_EQ(this->mock->register_writes.size(), 3);
    EXPECT_EQ(
        this->mock->register_writes[2],
        (RegisterWrite{.slave = 1, .address = 0x0980, .data = {0x01}, .mapped = true})
    );
}

/// @brief it should fail activation, before it starts SYNC0 or requests SAFE-OP, when
/// the clock of a slave stays more than 10 us off the reference clock.
TEST_F(SOEMMasterTest, ActivateFailsWhenADCClockDoesNotSettle) {
    this->initialize_with_dc_slave({DC_ONLY});
    this->mock->register_values[{1, 0x092C}] = bytes<uint32_t>(20'000);
    const auto err = this->activate();
    ASSERT_OCCURRED_AS(err, errors::ACTIVATION_ERROR);
    EXPECT_EQ(
        err.data,
        "the clock of slave 1 (16xAO) is " + x::telem::TimeSpan(20'000).to_string() +
            " off the reference clock after " +
            (x::telem::MILLISECOND * 50).to_string() + ". It must be within " +
            (x::telem::MICROSECOND * 10).to_string() + " before SYNC0 starts."
    );
    EXPECT_EQ(this->mock->register_writes.size(), 2);
    EXPECT_TRUE(this->mock->state_change_requests.empty());
}

/// @brief it should fail activation, before it starts SYNC0 or requests SAFE-OP, when
/// no frame carries the reference clock to the slaves, since the clock difference of a
/// slave keeps its old value until one does.
TEST_F(SOEMMasterTest, ActivateReadsTheClockDifferenceOnlyAfterAFrame) {
    this->initialize_with_dc_slave({DC_ONLY});
    const auto err = this->master->activate(PERIOD, no_cycle);
    ASSERT_OCCURRED_AS(err, errors::ACTIVATION_ERROR);
    EXPECT_EQ(
        err.data,
        "no frame carried the reference clock to the slaves within " +
            (x::telem::MILLISECOND * 50).to_string()
    );
    EXPECT_EQ(this->mock->register_writes.size(), 2);
    EXPECT_TRUE(this->mock->state_change_requests.empty());
}

/// @brief it should read bit 31 of the system time difference as its sign.
TEST_F(SOEMMasterTest, ActivateReadsTheSignOfTheClockDifference) {
    this->initialize_with_dc_slave({DC_ONLY});
    this->mock->register_values[{1, 0x092C}] = bytes<uint32_t>(0x80000000 | 5'000);
    ASSERT_NIL(this->activate());
    EXPECT_EQ(this->mock->register_writes.size(), 7);
}

/// @brief it should fail activation when a slave does not answer a read of its clock
/// difference.
TEST_F(SOEMMasterTest, ActivateFailsWhenADCClockCannotBeRead) {
    this->initialize_with_dc_slave({DC_ONLY});
    this->mock->read_register_return = 0;
    const auto err = this->activate();
    ASSERT_OCCURRED_AS(err, errors::ACTIVATION_ERROR);
    EXPECT_EQ(err.data, "slave 1 (16xAO) did not answer a read of DC register 0x092C");
}

/// @brief it should turn DC sync off when it stops the bus.
TEST_F(SOEMMasterTest, DeactivateClearsDCActivation) {
    this->initialize_with_dc_slave({DC_ONLY});
    ASSERT_NIL(this->activate());
    this->mock->register_writes.clear();
    this->master->deactivate();
    const std::vector<RegisterWrite> expected = {
        {.slave = 1, .address = 0x0980, .data = {0}, .mapped = true},
        {.slave = 1, .address = 0x0981, .data = {0}, .mapped = true},
    };
    EXPECT_EQ(this->mock->register_writes, expected);
}

/// @brief it should turn DC sync off on every slave when it stops the bus, also after
/// a slave fails a write.
TEST_F(SOEMMasterTest, DeactivateClearsDCActivationPastAFailedSlave) {
    MockSlaveInfo dc;
    dc.state = EC_STATE_PRE_OP;
    dc.hasdc = true;
    this->add_slave(dc);
    this->add_slave(dc);
    ASSERT_NIL(this->master->initialize());
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    ASSERT_NIL(this->activate());
    this->mock->register_writes.clear();
    this->mock->unwritable_slaves = {1};
    this->master->deactivate();
    const std::vector<RegisterWrite> expected = {
        {.slave = 1, .address = 0x0980, .data = {0}, .mapped = true},
        {.slave = 2, .address = 0x0980, .data = {0}, .mapped = true},
        {.slave = 2, .address = 0x0981, .data = {0}, .mapped = true},
    };
    EXPECT_EQ(this->mock->register_writes, expected);
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
    EXPECT_EQ(this->master->check(this->brk)[0].state, slave::State::SAFE_OP);
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
    EXPECT_EQ(health[0].al_status, ec_ALstatuscode2string(0x001B));
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
    this->master->set_slave_enabled(2, false);
    ASSERT_NIL(this->master->initialize());
    this->activate_master();
    this->mock->slave_infos[1].state = EC_STATE_SAFE_OP;
    const auto health = this->master->check(this->brk);
    ASSERT_EQ(health.size(), 1);
    EXPECT_EQ(health[0].position, 1);
    EXPECT_TRUE(this->mock->written_slaves.empty());
}

/// @brief it should set the propagation delay and the offset of a DC slave that comes
/// back from a power loss against the reference clock, then start SYNC0 on it, all
/// before SOEM requests SAFE-OP.
TEST_F(SOEMMasterTest, CheckRestoresTheClockOfADCSlaveThatLostPower) {
    this->activate_two_dc_slaves({DC_ONLY});
    this->mock->slave_infos[0].state = EC_STATE_OPERATIONAL;
    this->mock->slave_infos[1].state = EC_STATE_INIT;
    this->mock->register_values[{1, 0x0918}] = bytes<int64_t>(1'000'000);
    this->mock->register_values[{1, 0x0920}] = bytes<int64_t>(9'000'000'000);
    this->mock->register_values[{2, 0x0918}] = bytes<int64_t>(2'000);
    this->mock->register_values[{2, 0x0910}] = bytes<uint64_t>(5'000'000'000);
    this->check_while_cycling();
    const std::vector<RegisterWrite> expected = {
        {.slave = 2, .address = 0x0980, .data = {0}, .mapped = true},
        {.slave = 2, .address = 0x0981, .data = {0}, .mapped = true},
        {.slave = 2, .address = 0x0928, .data = bytes<int32_t>(300), .mapped = true},
        {.slave = 2,
         .address = 0x0920,
         .data = bytes<int64_t>(1'000'000 + 9'000'000'000 + 300 - 2'000),
         .mapped = true},
        {.slave = 2, .address = 0x0980, .data = {0}, .mapped = true},
        {.slave = 2, .address = 0x09A0, .data = bytes<uint32_t>(1000), .mapped = true},
        {.slave = 2,
         .address = 0x09A4,
         .data = bytes<uint32_t>(999'999'000),
         .mapped = true},
        {.slave = 2,
         .address = 0x0990,
         .data = bytes<uint64_t>(5'100'001'000),
         .mapped = true},
        {.slave = 2, .address = 0x0981, .data = {0x07}, .mapped = true},
    };
    EXPECT_EQ(this->mock->register_writes, expected);
    EXPECT_EQ(this->mock->latch_count, 1);
    EXPECT_EQ(this->mock->reconfigured_slaves, std::vector<uint16_t>{2});
    EXPECT_TRUE(this->mock->written_slaves.empty());
}

/// @brief it should set the clock of a reference clock that comes back from a power
/// loss against the next DC slave, and leave SYNC0 off on a slave in SM mode.
TEST_F(SOEMMasterTest, CheckRestoresTheReferenceClockFromTheNextDCSlave) {
    this->activate_two_dc_slaves({SM});
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->mock->slave_infos[1].state = EC_STATE_SAFE_OP + EC_STATE_ERROR;
    this->mock->register_values[{1, 0x0918}] = bytes<int64_t>(2'000);
    this->mock->register_values[{2, 0x0918}] = bytes<int64_t>(1'000'000);
    this->mock->register_values[{2, 0x0920}] = bytes<int64_t>(9'000'000'000);
    this->check_while_cycling();
    const std::vector<RegisterWrite> expected = {
        {.slave = 1, .address = 0x0980, .data = {0}, .mapped = true},
        {.slave = 1, .address = 0x0981, .data = {0}, .mapped = true},
        {.slave = 1, .address = 0x0928, .data = bytes<int32_t>(0), .mapped = true},
        {.slave = 1,
         .address = 0x0920,
         .data = bytes<int64_t>(1'000'000 + 9'000'000'000 - 300 - 2'000),
         .mapped = true},
    };
    EXPECT_EQ(this->mock->register_writes, expected);
}

/// @brief it should set the clock of the only DC slave, when it comes back from a power
/// loss, to the DC time of the latest good frame plus the host time since its send.
TEST_F(SOEMMasterTest, CheckRestoresTheOnlyDCSlaveFromTheDCTimeOfTheLatestFrame) {
    this->initialize_with_dc_slave({DC_ONLY});
    this->mock->hasdc = true;
    this->mock->host_dc_offset = 7'000'000'000;
    this->cycle_gap = std::chrono::milliseconds(10);
    this->activate_master();
    this->mock->register_writes.clear();
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->mock->register_values[{1, 0x0918}] = bytes<int64_t>(2'000);
    this->check_while_cycling();
    const auto error = this->written_offset(1) -
                       (this->mock->latched_at + 7'000'000'000 - 2'000);
    EXPECT_LT(std::abs(error), (5 * x::telem::MILLISECOND).nanoseconds());
    EXPECT_EQ(
        this->mock->register_writes.back(),
        (RegisterWrite{.slave = 1, .address = 0x0981, .data = {0x07}, .mapped = true})
    );
    EXPECT_TRUE(this->mock->state_change_requests.empty());
}

/// @brief it should keep the DC time of the latest frame when the only DC slave, which
/// maps no PDOs, comes back from a power loss with its clock reset, since its frames
/// keep a matching working counter.
TEST_F(SOEMMasterTest, CheckIgnoresTheResetClockOfAReturningReferenceClock) {
    this->initialize_with_dc_slave({DC_ONLY});
    this->mock->hasdc = true;
    this->mock->dc_time_value = 7'000'000'000;
    this->activate_master();
    this->mock->register_writes.clear();
    this->mock->dc_time_value = 1'000;
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->mock->register_values[{1, 0x0918}] = bytes<int64_t>(2'000);
    this->check_while_cycling();
    const auto error = this->written_offset(1) - (7'000'000'000 - 2'000);
    EXPECT_LT(std::abs(error), x::telem::SECOND.nanoseconds());
}

/// @brief it should set the clock of the first DC slave from the latest frame when the
/// whole bus comes back from a power loss, then set the next one from the first, and
/// hold none of them in PRE-OP.
TEST_F(SOEMMasterTest, CheckRestoresEveryDCSlaveAfterTheWholeBusLostPower) {
    this->mock->dc_time_value = 7'000'000'000;
    this->activate_two_dc_slaves({DC_ONLY});
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->mock->slave_infos[1].state = EC_STATE_INIT;
    this->mock->register_values[{1, 0x0918}] = bytes<int64_t>(2'000);
    this->mock->register_values[{2, 0x0918}] = bytes<int64_t>(5'000);
    this->check_while_cycling();
    EXPECT_EQ(this->mock->reconfigured_slaves, (std::vector<uint16_t>{1, 2}));
    const auto first = this->written_offset(1);
    const auto error = first - (7'000'000'000 - 2'000);
    EXPECT_LT(std::abs(error), x::telem::SECOND.nanoseconds());
    EXPECT_EQ(this->written_offset(2), 2'000 + first + 300 - 5'000);
    EXPECT_TRUE(this->mock->state_change_requests.empty());
}

/// @brief it should hold a DC slave that comes back from a power loss in PRE-OP, and
/// report why, when the master cannot restore its clock.
TEST_F(SOEMMasterTest, CheckHoldsADCSlaveInPreOpWhenItsClockCannotBeRestored) {
    this->initialize_with_dc_slave({DC_ONLY});
    this->activate_master();
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->mock->latch_return = 0;
    this->master->check(this->brk);
    EXPECT_EQ(
        this->mock->state_change_requests,
        (std::vector<std::pair<uint16_t, uint16_t>>{{1, EC_STATE_PRE_OP}})
    );
    EXPECT_EQ(this->mock->written_slaves, std::vector<uint16_t>{1});
    const auto health = this->master->check(this->brk);
    ASSERT_EQ(health.size(), 1);
    EXPECT_TRUE(health[0].faulted);
    EXPECT_EQ(
        health[0].al_status,
        "Failed to restore the distributed clock: no slave latched its receive time"
    );
}

/// @brief it should stop reporting a restore failure once the clock of the slave is
/// restored.
TEST_F(SOEMMasterTest, CheckClearsTheRestoreFailureOnceTheClockIsRestored) {
    this->activate_two_dc_slaves({DC_ONLY});
    this->mock->slave_infos[0].state = EC_STATE_OPERATIONAL;
    this->mock->slave_infos[1].state = EC_STATE_INIT;
    this->mock->latch_return = 0;
    this->master->check(this->brk);
    EXPECT_EQ(
        this->master->check(this->brk)[1].al_status,
        "Failed to restore the distributed clock: no slave latched its receive time"
    );
    this->mock->latch_return = 1;
    this->check_while_cycling();
    const auto health = this->master->check(this->brk);
    EXPECT_FALSE(health[1].faulted);
    EXPECT_TRUE(health[1].al_status.empty());
}

/// @brief it should set the clock of a DC slave that comes back from a power loss from
/// the DC time of the latest frame with a matching working counter.
TEST_F(SOEMMasterTest, CheckIgnoresTheDCTimeOfAFrameWithAWorkingCounterMismatch) {
    this->initialize_with_dc_slave({DC_ONLY});
    this->mock->hasdc = true;
    this->mock->host_dc_offset = 7'000'000'000;
    this->activate_master();
    this->mock->receive_return = 2;
    this->mock->host_dc_offset = 9'000'000'000;
    ASSERT_NIL(this->master->send());
    ASSERT_OCCURRED_AS(this->master->receive(), errors::WORKING_COUNTER_ERROR);
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->mock->register_values[{1, 0x0918}] = bytes<int64_t>(2'000);
    this->master->check(this->brk);
    const auto error = this->written_offset(1) -
                       (this->mock->latched_at + 7'000'000'000 - 2'000);
    EXPECT_LT(std::abs(error), x::telem::SECOND.nanoseconds());
}

/// @brief it should leave the clock registers of a slave without DC alone when it
/// reconfigures the slave.
TEST_F(SOEMMasterTest, CheckRestoresNoClockOnASlaveWithoutDC) {
    this->initialize_with_slave();
    this->activate_master();
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->master->check(this->brk);
    EXPECT_EQ(this->mock->reconfigured_slaves, std::vector<uint16_t>{1});
    EXPECT_TRUE(this->mock->register_writes.empty());
    EXPECT_EQ(this->mock->latch_count, 0);
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

/// @brief it should wait for INIT when a slave still shows the error flag of a fault in
/// SAFE-OP after the INIT request.
TEST_F(SOEMMasterTest, DeactivateWaitsForInitAfterFault) {
    this->initialize_with_slave();
    this->activate_master();
    this->mock->slave_infos[0].state = EC_STATE_SAFE_OP + EC_STATE_ERROR;
    this->mock->slave_infos[0].ALstatuscode = 0x001B;
    this->mock->statecheck_pending_polls[EC_STATE_INIT] = 2;
    this->mock->readstate_count = 0;
    this->master->deactivate();
    EXPECT_EQ(this->mock->slave_infos[0].state, EC_STATE_INIT);
    EXPECT_EQ(this->mock->readstate_count, 3);
}

/// @brief it should not set the clock of a DC slave that comes back from a power loss
/// from a DC time that the master saved before it deactivated.
TEST_F(SOEMMasterTest, DeactivateDropsTheSavedDCTime) {
    this->initialize_with_dc_slave({DC_ONLY});
    this->mock->hasdc = true;
    this->mock->host_dc_offset = 7'000'000'000;
    this->activate_master();
    this->master->deactivate();
    ASSERT_NIL(this->master->initialize());
    this->mock->receive_return = 2;
    ASSERT_NIL(this->activate());
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->master->check(this->brk);
    const auto health = this->master->check(this->brk);
    ASSERT_EQ(health.size(), 1);
    EXPECT_EQ(
        health[0].al_status,
        "Failed to restore the distributed clock: no frame has carried the DC time to "
        "set the clock from"
    );
}

/// @brief it should stop reporting a restore failure once the master deactivates.
TEST_F(SOEMMasterTest, DeactivateClearsTheRestoreFailures) {
    this->initialize_with_dc_slave({DC_ONLY});
    this->activate_master();
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->mock->latch_return = 0;
    this->master->check(this->brk);
    ASSERT_TRUE(this->master->check(this->brk)[0].faulted);
    this->master->deactivate();
    this->mock->latch_return = 1;
    ASSERT_NIL(this->master->initialize());
    this->activate_master();
    const auto health = this->master->check(this->brk);
    ASSERT_EQ(health.size(), 1);
    EXPECT_FALSE(health[0].faulted);
    EXPECT_TRUE(health[0].al_status.empty());
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
    ASSERT_NIL(this->activate());
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
    ASSERT_NIL(this->activate());

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
    ASSERT_NIL(this->activate());

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

}
