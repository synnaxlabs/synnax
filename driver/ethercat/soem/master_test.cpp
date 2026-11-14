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
#include <map>
#include <optional>
#include <string>
#include <thread>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/loop/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/ethercat/coe/coe.h"
#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/esi/known_devices.h"
#include "driver/ethercat/esi/settings.h"
#include "driver/ethercat/master/contract.h"
#include "driver/ethercat/soem/master.h"
#include "driver/ethercat/soem/mock_api.h"
#include "driver/ethercat/testutil/stg.h"

namespace driver::ethercat::soem {
namespace {
/// @brief the bus cycle the specs activate the master with.
const auto PERIOD = x::telem::MILLISECOND;

/// @brief a bus cycle that sends nothing. The mock API needs no process data to reach
/// OP.
void no_cycle(const master::Image &) {}

/// @brief the only mode of a module that runs in DC mode alone: SYNC0 of 1 us and SYNC1
/// of 1000 bus cycles.
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
    /// @brief the settings of each slave that activate() starts the master with.
    std::map<uint16_t, slave::Config> configs;
    /// @brief the process image of the latest activate().
    master::Image image;
    /// @brief the result of the latest activate().
    master::Activation activation;
    /// @brief the time from the send of a frame to its receive in cycle.
    std::chrono::microseconds cycle_gap{100};

    void SetUp() override {
        // A loaded Windows host can take tens of milliseconds to cycle two frames.
        this->create_master(x::telem::SECOND);
        this->brk.start();
    }

    /// @brief replaces the master with one on a new mock that gives the DC clocks
    /// dc_settle_timeout to settle.
    void create_master(const x::telem::TimeSpan dc_settle_timeout) {
        auto api = std::make_unique<MockAPI>();
        this->mock = api.get();
        this->master = std::make_unique<Master>(
            std::move(api),
            "eth0",
            Config{
                .pre_op_timeout = x::telem::MILLISECOND * 100,
                .state_change_timeout = x::telem::MILLISECOND * 100,
                .init_timeout = x::telem::MILLISECOND * 100,
                .dc_settle_timeout = dc_settle_timeout,
            }
        );
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
        ASSERT_NIL_P(this->master->scan());
    }

    /// @brief writes modes to the SII DC category of every slave.
    void set_op_modes(const std::vector<dc::OpMode> &modes) {
        for (auto &slave: this->mock->slave_infos)
            set_op_modes(slave, modes);
    }

    /// @brief writes modes to the SII DC category of slave.
    static void
    set_op_modes(MockSlaveInfo &slave, const std::vector<dc::OpMode> &modes) {
        constexpr uint16_t start = 0x200;
        slave.sii_categories = {{60, start}};
        const auto put = [&slave](
                             const size_t addr,
                             const uint64_t value,
                             const int n
                         ) {
            for (int i = 0; i < n; ++i)
                slave.sii_bytes[static_cast<uint16_t>(addr + i)] = static_cast<uint8_t>(
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
            slave.sii_strings[static_cast<uint16_t>(i + 1)] = m.name;
        }
    }

    /// @brief initializes the master with one DC slave named 16xAO that lists modes,
    /// and gives it a DC system time of 5 s.
    void initialize_with_dc_slave(const std::vector<dc::OpMode> &modes) {
        MockSlaveInfo slave;
        slave.state = EC_STATE_PRE_OP;
        slave.name = "16xAO";
        this->initialize_with_dc_slave(slave, modes);
    }

    /// @brief initializes the master with slave as its one DC slave, which lists modes,
    /// and gives it a DC system time of 5 s.
    void initialize_with_dc_slave(
        MockSlaveInfo slave,
        const std::vector<dc::OpMode> &modes
    ) {
        slave.hasdc = true;
        this->add_slave(slave);
        this->set_op_modes(modes);
        this->mock->register_values[{1, 0x0910}] = 5'000'000'000;
        ASSERT_NIL_P(this->master->scan());
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
        this->activation = this->master->activate(
            {.period = this->period, .slaves = this->configs},
            [&](const master::Image &image) {
                this->image = image;
                if (on_start) on_start();
                cycle = std::thread([&] { this->cycle(cycling); });
            }
        );
        cycling = false;
        if (cycle.joinable()) cycle.join();
        return this->activation.err;
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

    /// @brief checks a DC slave that came back from a power loss with a clock that does
    /// not settle in 10 s, and stops the breaker 100 ms into the check. A bus cycle
    /// exchanges process data during the check when cycling is true.
    /// @returns how long the check took.
    std::chrono::steady_clock::duration stop_during_restore(const bool cycling) {
        this->create_master(x::telem::SECOND * 10);
        this->initialize_with_dc_slave({DC_ONLY});
        this->activate_master();
        this->mock->slave_infos[0].state = EC_STATE_INIT;
        this->mock->register_values[{1, 0x092C}] = 20'000;
        std::atomic<bool> running = cycling;
        std::thread cycle([&] { this->cycle(running); });
        std::thread stop([this] {
            std::this_thread::sleep_for(std::chrono::milliseconds(100));
            this->brk.stop();
        });
        const auto start = std::chrono::steady_clock::now();
        this->master->check(this->brk);
        const auto elapsed = std::chrono::steady_clock::now() - start;
        stop.join();
        running = false;
        cycle.join();
        return elapsed;
    }

    /// @brief activates the master with two DC slaves named 16xAO that list modes. The
    /// first is the reference clock, and the second is 300 ns behind it.
    void activate_two_dc_slaves(
        const std::vector<dc::OpMode> &modes,
        std::map<uint16_t, slave::Config> configs = {}
    ) {
        MockSlaveInfo slave;
        slave.state = EC_STATE_PRE_OP;
        slave.hasdc = true;
        slave.name = "16xAO";
        this->add_slave(slave);
        slave.pdelay = 300;
        this->add_slave(slave);
        this->set_op_modes(modes);
        ASSERT_NIL_P(this->master->scan());
        this->configs = std::move(configs);
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
        return it->value;
    }

    /// @brief initializes the master with a slave that stays in PRE-OP without an error
    /// and a slave that refuses SAFE-OP with the AL status code, by default 0x0026
    /// ("Inconsistent settings").
    void add_pending_and_refusing_slaves(const uint16_t code = 0x0026) {
        MockSlaveInfo pending;
        pending.state = EC_STATE_PRE_OP;
        pending.name = "PendingSlave";
        pending.Ibits = 16;
        this->add_slave(pending);
        MockSlaveInfo refusing;
        refusing.state = EC_STATE_PRE_OP;
        refusing.name = "RefusingSlave";
        refusing.ALstatuscode = code;
        refusing.Ibits = 16;
        this->add_slave(refusing);
        ASSERT_NIL_P(this->master->scan());
        this->mock->obytes = 2;
        this->mock->ibytes = 4;
        this->mock->statecheck_overrides[EC_STATE_SAFE_OP] = EC_STATE_PRE_OP +
                                                             EC_STATE_ERROR;
    }

    /// @brief initializes the master with two slaves in PRE-OP named First and Second.
    void add_two_slaves() {
        for (const auto *name: {"First", "Second"}) {
            MockSlaveInfo slave;
            slave.state = EC_STATE_PRE_OP;
            slave.name = name;
            this->add_slave(slave);
        }
        ASSERT_NIL_P(this->master->scan());
        this->mock->obytes = 2;
        this->mock->ibytes = 4;
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

/// @brief it should read the identity and name of each slave on the bus.
TEST_F(SOEMMasterTest, ScanHappyPath) {
    this->initialize_with_slave();
    auto slaves = this->master->scan().first;
    ASSERT_EQ(slaves.size(), 1);
    EXPECT_EQ(slaves[0].properties.vendor_id, 0x1234);
    EXPECT_EQ(slaves[0].properties.product_code, 0x5678);
    EXPECT_EQ(slaves[0].properties.revision, 0x0001);
    EXPECT_EQ(slaves[0].properties.serial, 0xABCD);
    EXPECT_EQ(slaves[0].properties.name, "TestSlave");
}

/// @brief it should return the socket check error and not start SOEM when the Driver
/// cannot open a raw socket.
TEST_F(SOEMMasterTest, ScanSocketCheckFails) {
    this->add_default_slave();
    this->mock->check_socket_err = x::errors::Error(
        errors::PERMISSION_DENIED,
        "permission denied opening eth0"
    );
    ASSERT_OCCURRED_AS_P(this->master->scan(), errors::PERMISSION_DENIED);
    EXPECT_FALSE(this->mock->init_called);
}

/// @brief it should tell the user how to get raw Ethernet access when the socket fails
/// to open, and close the socket.
TEST_F(SOEMMasterTest, ScanInitFails) {
    this->add_default_slave();
    this->mock->init_return = 0;
    const auto err = this->master->scan().second;
    ASSERT_OCCURRED_AS(err, errors::MASTER_INIT_ERROR);
    EXPECT_FALSE(err.matches(errors::NO_SLAVES_FOUND));
    EXPECT_EQ(
        err.data,
        "Cannot open eth0 for raw Ethernet access. Run the Driver as root (or with "
        "CAP_NET_RAW on Linux), and install Npcap on Windows."
    );
    EXPECT_TRUE(this->mock->close_called);
}

/// @brief it should return a no slaves error that asks for a cable and power check, and
/// close the socket, when no slave answers.
TEST_F(SOEMMasterTest, ScanReportsAnInterfaceWithNoSlaves) {
    this->mock->config_init_return = 0;
    const auto err = this->master->scan().second;
    ASSERT_OCCURRED_AS(err, errors::NO_SLAVES_FOUND);
    EXPECT_FALSE(err.matches(errors::REPLIES_BLOCKED));
    EXPECT_EQ(
        err.data,
        "No EtherCAT slaves found on eth0. Check the cable and the power of the first "
        "device."
    );
    EXPECT_TRUE(this->mock->close_called);
}

/// @brief it should return a blocked replies error that names the DewesoftX filter and
/// the Windows name of the adapter when the filter is bound to an interface with no
/// slaves.
TEST_F(SOEMMasterTest, ScanNamesTheDewesoftFilterOnAnInterfaceWithNoSlaves) {
    this->mock->config_init_return = 0;
    this->mock->filtered_interfaces.insert("eth0");
    this->mock->adapter_names["eth0"] = "Ethernet 2";
    const auto err = this->master->scan().second;
    ASSERT_OCCURRED_AS(err, errors::REPLIES_BLOCKED);
    EXPECT_EQ(
        err.data,
        "No EtherCAT slaves found on Ethernet 2. The DewesoftX \"EthercatDAQ Filter\" "
        "on this adapter blocks EtherCAT replies. Clear it in the properties of "
        "\"Ethernet 2\" (Settings > Network & internet > Advanced network settings > "
        "More network adapter options, then right-click the adapter > Properties)."
    );
    EXPECT_TRUE(this->mock->close_called);
}

/// @brief it should name the interface when the Windows name of an adapter with the
/// DewesoftX filter cannot be read.
TEST_F(SOEMMasterTest, ScanNamesTheInterfaceWhenTheFilteredAdapterHasNoName) {
    this->mock->config_init_return = 0;
    this->mock->filtered_interfaces.insert("eth0");
    const auto err = this->master->scan().second;
    ASSERT_OCCURRED_AS(err, errors::REPLIES_BLOCKED);
    EXPECT_EQ(
        err.data,
        "No EtherCAT slaves found on eth0. The DewesoftX \"EthercatDAQ Filter\" on "
        "this adapter blocks EtherCAT replies. Clear it in the properties of \"eth0\" "
        "(Settings > Network & internet > Advanced network settings > More network "
        "adapter options, then right-click the adapter > Properties)."
    );
}

/// @brief it should take the Windows name of the adapter as its name, or the interface
/// name when the adapter has none.
TEST_F(SOEMMasterTest, NameIsTheWindowsNameOfTheAdapter) {
    EXPECT_EQ(this->master->name(), "eth0");
    this->mock->adapter_names["eth0"] = "Ethernet 2";
    EXPECT_EQ(this->master->name(), "Ethernet 2");
}

/// @brief it should call the interface by the Windows name of its adapter when it does
/// not open.
TEST_F(SOEMMasterTest, ScanInitFailureNamesTheWindowsAdapter) {
    this->add_default_slave();
    this->mock->init_return = 0;
    this->mock->adapter_names["eth0"] = "Ethernet 2";
    const auto err = this->master->scan().second;
    ASSERT_OCCURRED_AS(err, errors::MASTER_INIT_ERROR);
    EXPECT_EQ(
        err.data,
        "Cannot open Ethernet 2 for raw Ethernet access. Run the Driver as root (or "
        "with CAP_NET_RAW on Linux), and install Npcap on Windows."
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

/// @brief it should name and type each PDO entry from the CoE object dictionary of the
/// slave.
TEST_F(SOEMMasterTest, CoEDiscoveryUsesEntryNamesAndDataTypes) {
    MockSlaveInfo slave;
    slave.state = EC_STATE_PRE_OP;
    slave.eep_man = 0x1234;
    slave.eep_id = 0x5678;
    slave.mbx_proto = ECT_MBXPROT_COE;
    this->add_slave(slave);
    this->mock->slave_infos[0].sdo_objects = {
        {{0x1C00, 0}, 4},
        {{0x1C00, 3}, 3},
        {{0x1C00, 4}, 4},
        {{ECT_SDO_TXPDOASSIGN, 0}, 1},
        {{ECT_SDO_TXPDOASSIGN, 1}, 0x1A00},
        {{0x1A00, 0}, 2},
        {{0x1A00, 1}, 0x60000110},
        {{0x1A00, 2}, 0x60000220},
        {{ECT_SDO_RXPDOASSIGN, 0}, 0},
    };
    this->mock->slave_infos[0].oe_entries = {
        {{0x6000, 1}, {"Status", 0x0003}},
        {{0x6000, 2}, {"Value", 0x0008}},
    };

    ASSERT_NIL_P(this->master->scan());

    const auto slaves = this->master->scan().first;
    ASSERT_EQ(slaves.size(), 1);
    const auto &inputs = slaves[0].properties.input_pdos;
    ASSERT_EQ(inputs.size(), 2);
    EXPECT_EQ(inputs[0].name, "Status");
    EXPECT_EQ(inputs[0].data_type, x::telem::INT16_T);
    EXPECT_EQ(inputs[1].name, "Value");
    EXPECT_EQ(inputs[1].data_type, x::telem::FLOAT32_T);
}

/// @brief it should read the PDO layout of each slave from that slave's own object
/// dictionary.
TEST_F(SOEMMasterTest, CoEDiscoveryReadsEachSlaveFromItsOwnDictionary) {
    for (const uint32_t mapping: {0x60000110U, 0x60100120U}) {
        MockSlaveInfo slave;
        slave.state = EC_STATE_PRE_OP;
        slave.mbx_proto = ECT_MBXPROT_COE;
        slave.sdo_objects = {
            {{0x1C00, 0}, 4},
            {{0x1C00, 3}, 3},
            {{0x1C00, 4}, 4},
            {{ECT_SDO_RXPDOASSIGN, 0}, 0},
            {{ECT_SDO_TXPDOASSIGN, 0}, 1},
            {{ECT_SDO_TXPDOASSIGN, 1}, 0x1A00},
            {{0x1A00, 0}, 1},
            {{0x1A00, 1}, mapping},
        };
        this->add_slave(slave);
    }

    const auto slaves = ASSERT_NIL_P(this->master->scan());

    ASSERT_EQ(slaves.size(), 2);
    ASSERT_EQ(slaves[0].properties.input_pdos.size(), 1);
    EXPECT_EQ(slaves[0].properties.input_pdos[0].index, 0x6000);
    EXPECT_EQ(slaves[0].properties.input_pdos[0].bit_length, 16);
    ASSERT_EQ(slaves[1].properties.input_pdos.size(), 1);
    EXPECT_EQ(slaves[1].properties.input_pdos[0].index, 0x6010);
    EXPECT_EQ(slaves[1].properties.input_pdos[0].bit_length, 32);
}

/// @brief it should read the PDO layout of a slave that supports complete access with
/// one read of each whole object.
TEST_F(SOEMMasterTest, CoEDiscoveryReadsWholeObjectsWithCompleteAccess) {
    MockSlaveInfo slave;
    slave.state = EC_STATE_PRE_OP;
    slave.mbx_proto = ECT_MBXPROT_COE;
    slave.complete_access = true;
    slave.complete_objects = {
        {0x1C00, {4, 0, 1, 2, 3, 4}},
        {ECT_SDO_RXPDOASSIGN, {0, 0}},
        {ECT_SDO_TXPDOASSIGN, {1, 0, 0x00, 0x1A}},
        {0x1A00, {2, 0, 0x10, 0x01, 0x00, 0x60, 0x20, 0x02, 0x00, 0x60}},
    };
    this->add_slave(slave);

    const auto slaves = ASSERT_NIL_P(this->master->scan());

    ASSERT_EQ(slaves.size(), 1);
    const auto &inputs = slaves[0].properties.input_pdos;
    ASSERT_EQ(inputs.size(), 2);
    EXPECT_EQ(inputs[0].index, 0x6000);
    EXPECT_EQ(inputs[0].sub_index, 1);
    EXPECT_EQ(inputs[0].bit_length, 16);
    EXPECT_EQ(inputs[1].sub_index, 2);
    EXPECT_EQ(inputs[1].bit_length, 32);
}

namespace {
/// @brief returns a slave of model 0x1234:0x5678 at revision rev with one TxPDO,
/// 0x1A00, that maps 0x6000:01.
MockSlaveInfo create_named_slave(const uint32_t rev) {
    MockSlaveInfo slave;
    slave.state = EC_STATE_PRE_OP;
    slave.eep_man = 0x1234;
    slave.eep_id = 0x5678;
    slave.eep_rev = rev;
    slave.mbx_proto = ECT_MBXPROT_COE;
    slave.sdo_objects = {
        {{0x1C00, 0}, 4},
        {{0x1C00, 3}, 3},
        {{0x1C00, 4}, 4},
        {{ECT_SDO_RXPDOASSIGN, 0}, 0},
        {{ECT_SDO_TXPDOASSIGN, 0}, 1},
        {{ECT_SDO_TXPDOASSIGN, 1}, 0x1A00},
        {{0x1A00, 0}, 1},
        {{0x1A00, 1}, 0x60000110},
    };
    return slave;
}

/// @brief makes a slave list the given module identities in its detected module list.
void set_modules(MockSlaveInfo &slave, const std::vector<uint32_t> &modules) {
    slave.sdo_objects[{0xF050, 0}] = static_cast<uint32_t>(modules.size());
    for (size_t i = 0; i < modules.size(); ++i)
        slave.sdo_objects[{0xF050, static_cast<uint8_t>(i + 1)}] = modules[i];
}
}

/// @brief it should give each slave of a model the names and data types that it read
/// from the first one.
TEST_F(SOEMMasterTest, CoEDiscoveryReadsTheNamesOfAModelOnce) {
    this->add_slave(create_named_slave(1));
    this->add_slave(create_named_slave(1));
    this->mock->slave_infos[0].od_names = {{0x1A00, "Inputs"}};
    this->mock->slave_infos[0].oe_entries = {{{0x6000, 1}, {"Status", 0x0006}}};

    const auto slaves = ASSERT_NIL_P(this->master->scan());

    ASSERT_EQ(slaves.size(), 2);
    for (const auto &s: slaves) {
        ASSERT_EQ(s.properties.input_pdos.size(), 1);
        EXPECT_EQ(s.properties.input_pdos[0].name, "Status");
        EXPECT_EQ(s.properties.input_pdos[0].data_type, x::telem::UINT16_T);
        EXPECT_EQ(s.properties.input_pdos[0].pdo_name, "Inputs");
    }
}

/// @brief it should read the names and data types of a slave whose model has another
/// revision.
TEST_F(SOEMMasterTest, CoEDiscoveryReadsTheNamesOfEachRevision) {
    this->add_slave(create_named_slave(1));
    this->add_slave(create_named_slave(2));
    this->mock->slave_infos[0].oe_entries = {{{0x6000, 1}, {"Status", 0x0006}}};
    this->mock->slave_infos[1].oe_entries = {{{0x6000, 1}, {"Value", 0x0008}}};

    const auto slaves = ASSERT_NIL_P(this->master->scan());

    ASSERT_EQ(slaves.size(), 2);
    ASSERT_EQ(slaves[1].properties.input_pdos.size(), 1);
    EXPECT_EQ(slaves[1].properties.input_pdos[0].name, "Value");
    EXPECT_EQ(slaves[1].properties.input_pdos[0].data_type, x::telem::FLOAT32_T);
}

/// @brief it should read the names and data types of a slave of a model whose modules
/// differ from those of the first slave.
TEST_F(SOEMMasterTest, CoEDiscoveryReadsTheNamesOfEachModuleList) {
    this->add_slave(create_named_slave(1));
    this->add_slave(create_named_slave(1));
    set_modules(this->mock->slave_infos[0], {0x1001});
    set_modules(this->mock->slave_infos[1], {0x1002});
    this->mock->slave_infos[0].oe_entries = {{{0x6000, 1}, {"Status", 0x0006}}};
    this->mock->slave_infos[1].oe_entries = {{{0x6000, 1}, {"Value", 0x0003}}};

    const auto slaves = ASSERT_NIL_P(this->master->scan());

    ASSERT_EQ(slaves.size(), 2);
    ASSERT_EQ(slaves[1].properties.input_pdos.size(), 1);
    EXPECT_EQ(slaves[1].properties.input_pdos[0].name, "Value");
    EXPECT_EQ(slaves[1].properties.input_pdos[0].data_type, x::telem::INT16_T);
}

/// @brief it should read the names and data types of a slave again when its modules
/// change between scans.
TEST_F(SOEMMasterTest, CoEDiscoveryReadsTheNamesAgainAfterAModuleChange) {
    this->add_slave(create_named_slave(1));
    set_modules(this->mock->slave_infos[0], {0x1001});
    this->mock->slave_infos[0].oe_entries = {{{0x6000, 1}, {"Status", 0x0006}}};
    ASSERT_NIL_P(this->master->scan());
    this->master->deactivate();
    set_modules(this->mock->slave_infos[0], {0x1002});
    this->mock->slave_infos[0].oe_entries = {{{0x6000, 1}, {"Value", 0x0003}}};

    const auto slaves = ASSERT_NIL_P(this->master->scan());

    ASSERT_EQ(slaves.size(), 1);
    ASSERT_EQ(slaves[0].properties.input_pdos.size(), 1);
    EXPECT_EQ(slaves[0].properties.input_pdos[0].name, "Value");
    EXPECT_EQ(slaves[0].properties.input_pdos[0].data_type, x::telem::INT16_T);
}

/// @brief it should read the names and data types of a slave that does not give its
/// detected modules, also when another slave of its model came first.
TEST_F(SOEMMasterTest, CoEDiscoveryReadsTheNamesOfASlaveWithUnknownModules) {
    this->add_slave(create_named_slave(1));
    this->add_slave(create_named_slave(1));
    this->mock->slave_infos[1].sdo_read_aborts = {{{0xF050, 0}, 0x06010000}};
    this->mock->slave_infos[0].oe_entries = {{{0x6000, 1}, {"Status", 0x0006}}};
    this->mock->slave_infos[1].oe_entries = {{{0x6000, 1}, {"Value", 0x0003}}};

    const auto slaves = ASSERT_NIL_P(this->master->scan());

    ASSERT_EQ(slaves.size(), 2);
    ASSERT_EQ(slaves[1].properties.input_pdos.size(), 1);
    EXPECT_EQ(slaves[1].properties.input_pdos[0].name, "Value");
    EXPECT_EQ(slaves[1].properties.input_pdos[0].data_type, x::telem::INT16_T);
}

/// @brief it should give each slave of a model the synchronization modes that it read
/// from the SII of the first one.
TEST_F(SOEMMasterTest, ScanReadsTheOpModesOfAModelOnce) {
    this->add_slave(create_named_slave(1));
    this->add_slave(create_named_slave(1));
    set_op_modes(this->mock->slave_infos[0], {DC_SECOND});

    const auto slaves = ASSERT_NIL_P(this->master->scan());

    ASSERT_EQ(slaves.size(), 2);
    ASSERT_EQ(slaves[1].properties.op_modes.size(), 1);
    EXPECT_EQ(slaves[1].properties.op_modes[0].name, DC_SECOND.name);
}

/// @brief it should read the SII of the next slave of a model when the SII of the first
/// one gives no synchronization modes.
TEST_F(SOEMMasterTest, ScanReadsTheOpModesAgainAfterAReadFindsNone) {
    this->add_slave(create_named_slave(1));
    this->add_slave(create_named_slave(1));
    set_op_modes(this->mock->slave_infos[1], {DC_SECOND});

    const auto slaves = ASSERT_NIL_P(this->master->scan());

    ASSERT_EQ(slaves.size(), 2);
    EXPECT_TRUE(slaves[0].properties.op_modes.empty());
    ASSERT_EQ(slaves[1].properties.op_modes.size(), 1);
    EXPECT_EQ(slaves[1].properties.op_modes[0].name, DC_SECOND.name);
}

/// @brief it should wait for the slaves to reach PRE-OP before it reads PDOs from the
/// CoE object dictionary.
TEST_F(SOEMMasterTest, CoEDiscoveryWaitsForPreOp) {
    MockSlaveInfo slave;
    slave.state = EC_STATE_INIT;
    slave.bus_state = EC_STATE_INIT;
    slave.mbx_proto = ECT_MBXPROT_COE;
    this->add_slave(slave);
    this->mock->slave_infos[0].sdo_objects = {
        {{0x1C00, 0}, 4},
        {{0x1C00, 3}, 3},
        {{0x1C00, 4}, 4},
        {{ECT_SDO_TXPDOASSIGN, 0}, 1},
        {{ECT_SDO_TXPDOASSIGN, 1}, 0x1A00},
        {{0x1A00, 0}, 1},
        {{0x1A00, 1}, 0x60000110},
        {{ECT_SDO_RXPDOASSIGN, 0}, 0},
    };

    ASSERT_NIL_P(this->master->scan());

    const auto slaves = this->master->scan().first;
    ASSERT_EQ(slaves.size(), 1);
    EXPECT_TRUE(slaves[0].status.pdo_discovery_error.empty());
    const auto &inputs = slaves[0].properties.input_pdos;
    ASSERT_EQ(inputs.size(), 1);
    EXPECT_EQ(inputs[0].index, 0x6000);
    EXPECT_EQ(inputs[0].sub_index, 1);
    EXPECT_EQ(inputs[0].bit_length, 16);
}

/// @brief it should read the PDOs of a CoE slave from its SII when the slave has no
/// sync manager type object, as SOEM then maps it from its SII.
TEST_F(SOEMMasterTest, CoEDiscoveryReadsTheSIIWithoutSyncManagerTypes) {
    MockSlaveInfo slave;
    slave.state = EC_STATE_PRE_OP;
    slave.mbx_proto = ECT_MBXPROT_COE;
    this->add_slave(slave);
    constexpr uint16_t start = 0x40;
    this->mock->slave_infos[0].sii_categories = {{ECT_SII_PDO, start}};
    // One TxPDO, 0x1A00, with entry 0x6000:01 (INTEGER16).
    const std::vector<uint8_t> category =
        {8, 0, 0x00, 0x1A, 1, 3, 0, 0, 0, 0, 0x00, 0x60, 1, 0, 0x03, 16, 0, 0};
    for (size_t i = 0; i < category.size(); ++i)
        this->mock->slave_infos[0].sii_bytes[start + i] = category[i];

    const auto slaves = ASSERT_NIL_P(this->master->scan());

    ASSERT_EQ(slaves[0].properties.input_pdos.size(), 1);
    EXPECT_EQ(slaves[0].properties.input_pdos[0].index, 0x6000);
    EXPECT_TRUE(slaves[0].status.pdo_discovery_error.empty());
}

/// @brief it should read the PDOs in the SII of a CoE slave that fails to give its sync
/// manager count, as SOEM maps the slave from its SII, and keep the error of the CoE
/// read.
TEST_F(SOEMMasterTest, CoEDiscoveryFallsBackToTheSIIWhenTheSyncManagerCountFails) {
    MockSlaveInfo slave;
    slave.state = EC_STATE_PRE_OP;
    slave.mbx_proto = ECT_MBXPROT_COE;
    this->add_slave(slave);
    this->mock->slave_infos[0].sdo_read_aborts = {{{0x1C00, 0}, 0x06010000}};
    constexpr uint16_t start = 0x40;
    this->mock->slave_infos[0].sii_categories = {{ECT_SII_PDO, start}};
    const std::vector<uint8_t> category =
        {8, 0, 0x00, 0x1A, 1, 3, 0, 0, 0, 0, 0x00, 0x60, 1, 0, 0x03, 16, 0, 0};
    for (size_t i = 0; i < category.size(); ++i)
        this->mock->slave_infos[0].sii_bytes[start + i] = category[i];

    const auto slaves = ASSERT_NIL_P(this->master->scan());

    ASSERT_EQ(slaves[0].properties.input_pdos.size(), 1);
    EXPECT_EQ(slaves[0].properties.input_pdos[0].index, 0x6000);
    EXPECT_EQ(
        slaves[0].status.pdo_discovery_error,
        "failed to read the PDO layout: failed to read 0x1C00:00: Unsupported access "
        "to an object, so the Driver uses the PDOs in its SII"
    );
}

/// @brief it should report why a CoE slave failed to give its sync manager count when
/// its SII lists no PDOs either.
TEST_F(SOEMMasterTest, CoEDiscoveryReportsAFailedReadWithoutSIIPDOs) {
    MockSlaveInfo slave;
    slave.state = EC_STATE_PRE_OP;
    slave.mbx_proto = ECT_MBXPROT_COE;
    this->add_slave(slave);
    this->mock->slave_infos[0].sdo_read_aborts = {{{0x1C00, 0}, 0x06010000}};

    const auto slaves = ASSERT_NIL_P(this->master->scan());

    EXPECT_TRUE(slaves[0].properties.input_pdos.empty());
    EXPECT_EQ(
        slaves[0].status.pdo_discovery_error,
        "failed to read the PDO layout: failed to read 0x1C00:00: Unsupported access "
        "to an object"
    );
}

/// @brief it should read the PDOs in the SII of a CoE slave whose failed reads leave no
/// CoE PDOs, as SOEM maps the slave from its SII.
TEST_F(SOEMMasterTest, CoEDiscoveryFallsBackToTheSIIWhenItReadsNoCoEPDO) {
    MockSlaveInfo slave;
    slave.state = EC_STATE_PRE_OP;
    slave.mbx_proto = ECT_MBXPROT_COE;
    this->add_slave(slave);
    this->mock->slave_infos[0].sdo_objects = {
        {{0x1C00, 0}, 4},
        {{0x1C00, 3}, 3},
        {{0x1C00, 4}, 4},
        {{ECT_SDO_RXPDOASSIGN, 0}, 0},
        {{ECT_SDO_TXPDOASSIGN, 0}, 1},
        {{ECT_SDO_TXPDOASSIGN, 1}, 0x1A00},
    };
    constexpr uint16_t start = 0x40;
    this->mock->slave_infos[0].sii_categories = {{ECT_SII_PDO, start}};
    const std::vector<uint8_t> category =
        {8, 0, 0x00, 0x1A, 1, 3, 0, 0, 0, 0, 0x00, 0x60, 1, 0, 0x03, 16, 0, 0};
    for (size_t i = 0; i < category.size(); ++i)
        this->mock->slave_infos[0].sii_bytes[start + i] = category[i];

    const auto slaves = ASSERT_NIL_P(this->master->scan());

    ASSERT_EQ(slaves[0].properties.input_pdos.size(), 1);
    EXPECT_EQ(slaves[0].properties.input_pdos[0].index, 0x6000);
    EXPECT_EQ(
        slaves[0].status.pdo_discovery_error,
        "failed to read the PDO layout: failed to read 0x1A00:00: Object does not "
        "exist in the object dictionary, so the Driver uses the PDOs in its SII"
    );
}

/// @brief it should keep the CoE PDOs it reads around a failed read, without the PDOs
/// in the SII, as SOEM does.
TEST_F(SOEMMasterTest, CoEDiscoveryKeepsTheCoEPDOsAroundAFailedRead) {
    MockSlaveInfo slave;
    slave.state = EC_STATE_PRE_OP;
    slave.mbx_proto = ECT_MBXPROT_COE;
    this->add_slave(slave);
    this->mock->slave_infos[0].sdo_objects = {
        {{0x1C00, 0}, 4},
        {{0x1C00, 3}, 3},
        {{0x1C00, 4}, 4},
        {{ECT_SDO_RXPDOASSIGN, 0}, 1},
        {{ECT_SDO_RXPDOASSIGN, 1}, 0x1600},
        {{0x1600, 0}, 1},
        {{0x1600, 1}, 0x70000110},
    };
    this->mock->slave_infos[0].sdo_read_aborts = {
        {{ECT_SDO_TXPDOASSIGN, 0}, 0x06010000}
    };
    constexpr uint16_t start = 0x40;
    this->mock->slave_infos[0].sii_categories = {{ECT_SII_PDO, start}};
    const std::vector<uint8_t> category =
        {8, 0, 0x00, 0x1A, 1, 3, 0, 0, 0, 0, 0x00, 0x60, 1, 0, 0x03, 16, 0, 0};
    for (size_t i = 0; i < category.size(); ++i)
        this->mock->slave_infos[0].sii_bytes[start + i] = category[i];

    const auto slaves = ASSERT_NIL_P(this->master->scan());

    EXPECT_TRUE(slaves[0].properties.input_pdos.empty());
    ASSERT_EQ(slaves[0].properties.output_pdos.size(), 1);
    EXPECT_EQ(slaves[0].properties.output_pdos[0].index, 0x7000);
    EXPECT_EQ(
        slaves[0].status.pdo_discovery_error,
        "failed to read the PDO layout: failed to read 0x1C13:00: Unsupported access "
        "to an object, so the Driver uses the PDOs it read"
    );
}

/// @brief it should reject a CoE slave whose SII lists PDOs on a sync manager with no
/// PDO assign object while its other sync managers have assigned PDOs, as SOEM leaves
/// the fixed sync manager unmapped.
TEST_F(SOEMMasterTest, CoEDiscoveryRejectsAFixedSyncManagerNextToAnAssignedOne) {
    MockSlaveInfo slave;
    slave.state = EC_STATE_PRE_OP;
    slave.mbx_proto = ECT_MBXPROT_COE;
    this->add_slave(slave);
    this->mock->slave_infos[0].sdo_objects = {
        {{0x1C00, 0}, 4},
        {{0x1C00, 3}, 3},
        {{0x1C00, 4}, 4},
        {{ECT_SDO_TXPDOASSIGN, 0}, 1},
        {{ECT_SDO_TXPDOASSIGN, 1}, 0x1A00},
        {{0x1A00, 0}, 1},
        {{0x1A00, 1}, 0x60000110},
    };
    constexpr uint16_t start = 0x40;
    this->mock->slave_infos[0].sii_categories = {{ECT_SII_PDO + 1, start}};
    // One RxPDO, 0x1600 on SM2, with entry 0x7000:01 (INTEGER16).
    const std::vector<uint8_t> category =
        {8, 0, 0x00, 0x16, 1, 2, 0, 0, 0, 0, 0x00, 0x70, 1, 0, 0x03, 16, 0, 0};
    for (size_t i = 0; i < category.size(); ++i)
        this->mock->slave_infos[0].sii_bytes[start + i] = category[i];

    const auto slaves = ASSERT_NIL_P(this->master->scan());

    EXPECT_TRUE(slaves[0].properties.input_pdos.empty());
    EXPECT_TRUE(slaves[0].properties.output_pdos.empty());
    EXPECT_EQ(
        slaves[0].status.pdo_discovery_error,
        "SOEM cannot map this slave because sync manager 2 has fixed PDOs and the "
        "other sync managers have assigned PDOs. Use IgH for this slave"
    );
    EXPECT_TRUE(this->mock->eeprom_held.empty());
}

/// @brief it should keep the assigned PDOs of a CoE slave when its SII lists no PDOs on
/// the sync manager that has no PDO assign object.
TEST_F(SOEMMasterTest, CoEDiscoveryKeepsAFixedSyncManagerWithNoPDOs) {
    MockSlaveInfo slave;
    slave.state = EC_STATE_PRE_OP;
    slave.mbx_proto = ECT_MBXPROT_COE;
    this->add_slave(slave);
    this->mock->slave_infos[0].sdo_objects = {
        {{0x1C00, 0}, 4},
        {{0x1C00, 3}, 3},
        {{0x1C00, 4}, 4},
        {{ECT_SDO_TXPDOASSIGN, 0}, 1},
        {{ECT_SDO_TXPDOASSIGN, 1}, 0x1A00},
        {{0x1A00, 0}, 1},
        {{0x1A00, 1}, 0x60000110},
    };

    const auto slaves = ASSERT_NIL_P(this->master->scan());

    ASSERT_EQ(slaves[0].properties.input_pdos.size(), 1);
    EXPECT_EQ(slaves[0].properties.input_pdos[0].index, 0x6000);
    EXPECT_TRUE(slaves[0].status.pdo_discovery_error.empty());
}

/// @brief it should give the EEPROM back to the slave after it reads PDOs from the SII.
TEST_F(SOEMMasterTest, SIIDiscoveryGivesTheEEPROMBackToTheSlave) {
    MockSlaveInfo slave;
    slave.state = EC_STATE_PRE_OP;
    this->add_slave(slave);
    constexpr uint16_t start = 0x40;
    this->mock->slave_infos[0].sii_categories = {{ECT_SII_PDO, start}};
    // One TxPDO, 0x1A00, with entry 0x6000:01 (INTEGER16).
    const std::vector<uint8_t> category =
        {8, 0, 0x00, 0x1A, 1, 3, 0, 0, 0, 0, 0x00, 0x60, 1, 0, 0x03, 16, 0, 0};
    for (size_t i = 0; i < category.size(); ++i)
        this->mock->slave_infos[0].sii_bytes[start + i] = category[i];

    ASSERT_NIL_P(this->master->scan());

    ASSERT_EQ(this->master->scan().first[0].properties.input_pdos.size(), 1);
    EXPECT_TRUE(this->mock->eeprom_held.empty());
}

/// @brief it should give the EEPROM back to the slave after it reads the OpModes.
TEST_F(SOEMMasterTest, ScanGivesTheEEPROMBackAfterReadingOpModes) {
    MockSlaveInfo slave;
    slave.state = EC_STATE_PRE_OP;
    this->add_slave(slave);
    constexpr uint16_t start = 0x200;
    this->mock->slave_infos[0].sii_categories = {{60, start}};
    // One SM mode with every field 0.
    std::vector<uint8_t> category(26, 0);
    category[0] = 12;
    for (size_t i = 0; i < category.size(); ++i)
        this->mock->slave_infos[0].sii_bytes[start + i] = category[i];

    ASSERT_NIL_P(this->master->scan());

    ASSERT_EQ(this->master->scan().first[0].properties.op_modes.size(), 1);
    EXPECT_TRUE(this->mock->eeprom_held.empty());
}

/// @brief it should give a slave no synchronization modes when its SII has no DC
/// category.
TEST_F(SOEMMasterTest, ScanGivesNoOpModesWithoutDCCategory) {
    this->initialize_with_slave();
    EXPECT_TRUE(this->master->scan().first[0].properties.op_modes.empty());
}

/// @brief returns a standalone 6xSTG in PRE-OP that holds the objects of the 6xSTG on
/// the station and lists the mailbox protocols in mbx_proto.
MockSlaveInfo create_stg(const uint16_t mbx_proto) {
    MockSlaveInfo slave;
    slave.state = EC_STATE_PRE_OP;
    slave.eep_man = testutil::DEWESOFT;
    slave.eep_id = testutil::IOLITE_6XSTG;
    slave.name = "6xSTG";
    slave.mbx_proto = mbx_proto;
    slave.sdo_objects = testutil::stg_objects(0);
    return slave;
}

/// @brief it should store the settings that the catalog describes for a slave, read
/// from that slave's own object dictionary.
TEST_F(SOEMMasterTest, ScanStoresTheSettingsThatTheCatalogDescribes) {
    this->add_default_slave();
    this->add_slave(create_stg(ECT_MBXPROT_COE));
    const auto slaves = ASSERT_NIL_P(this->master->scan());
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
TEST_F(SOEMMasterTest, ScanStoresTheSettingsOfTheSlaveRevision) {
    MockSlaveInfo slave;
    slave.state = EC_STATE_PRE_OP;
    slave.eep_man = testutil::DEWESOFT;
    slave.eep_id = testutil::IOLITE_DIO;
    slave.eep_rev = 0x00050000;
    slave.name = "32xDO";
    slave.mbx_proto = ECT_MBXPROT_COE;
    slave.sdo_objects = {{{0x8020, 1}, 0}, {{0x8020, 9}, 0}};
    this->add_slave(slave);
    const auto slaves = ASSERT_NIL_P(this->master->scan());
    ASSERT_TRUE(slaves[0].properties.settings.has_value());
    const auto &device = slaves[0].properties.settings.value().at("device");
    ASSERT_EQ(device.size(), 1);
    EXPECT_EQ(device[0]["name"], "Outputs on connection loss");
}

/// @brief it should report why a setting of a slave could not be read, and keep the
/// other settings.
TEST_F(SOEMMasterTest, ScanReportsAFailedSettingsRead) {
    auto stg = create_stg(ECT_MBXPROT_COE);
    stg.sdo_objects.erase({0x8050, 3});
    this->add_slave(stg);
    const auto slaves = ASSERT_NIL_P(this->master->scan());
    ASSERT_TRUE(slaves[0].properties.settings.has_value());
    EXPECT_TRUE(slaves[0].properties.settings.value().contains("mode"));
    EXPECT_EQ(
        slaves[0].status.settings_error,
        "failed to read 0x8050:03: Object does not exist in the object dictionary"
    );
}

/// @brief it should read no settings from a slave that does not list CoE.
TEST_F(SOEMMasterTest, ScanReadsNoSettingsFromASlaveWithoutCoE) {
    this->add_slave(create_stg(0));
    const auto slaves = ASSERT_NIL_P(this->master->scan());
    EXPECT_FALSE(slaves[0].properties.settings.has_value());
}

////////////////////// Activation //////////////////////

/// @brief it should activate a scanned bus.
TEST_F(SOEMMasterTest, ActivateHappyPath) {
    this->initialize_with_slave();
    this->activate_master();
}

/// @brief it should give each process data sync manager of the enabled slaves a length
/// only from its PDOs, since a slave can refuse SAFE-OP for a sync manager that is on
/// with no PDOs.
TEST_F(SOEMMasterTest, ActivateClearsProcessDataSMsBeforeTheMapping) {
    this->initialize_with_slave();
    this->activate_master();
    EXPECT_TRUE(this->mock->sms_cleared_at_map);
}

/// @brief it should fail to activate when SOEM cannot map the process data.
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

/// @brief it should wait for a state change on the clock of its config, so a fake clock
/// ends a one-minute timeout at once.
TEST_F(SOEMMasterTest, ActivateWaitsForAStateChangeOnTheConfigClock) {
    auto api = std::make_unique<MockAPI>();
    this->mock = api.get();
    this->master = std::make_unique<Master>(
        std::move(api),
        "eth0",
        Config{
            .pre_op_timeout = x::telem::MILLISECOND * 100,
            .state_change_timeout = x::telem::MINUTE,
            .clock = x::loop::fake_clock(),
        }
    );
    this->initialize_with_slave();
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    this->mock->statecheck_overrides[EC_STATE_SAFE_OP] = EC_STATE_PRE_OP;
    const auto start = std::chrono::steady_clock::now();
    ASSERT_OCCURRED_AS(this->activate(), errors::STATE_CHANGE_ERROR);
    EXPECT_LT(std::chrono::steady_clock::now() - start, std::chrono::seconds(10));
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
    this->add_pending_and_refusing_slaves();
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

/// @brief it should record why a slave refused SAFE-OP when another slave did not
/// refuse it.
TEST_F(SOEMMasterTest, ActivateRecordsEachSlaveThatRefusesAState) {
    this->add_pending_and_refusing_slaves();
    ASSERT_OCCURRED_AS(this->activate(), errors::STATE_CHANGE_REFUSED);
    const std::map<uint16_t, slave::Refusal> expected = {
        {2, {.reason = "refused SAFE-OP: Inconsistent settings", .code = 0x0026}},
    };
    EXPECT_EQ(this->activation.refusals, expected);
}

/// @brief it should refuse a used slave with no identity once the image shows the
/// other slaves, without mapping the slave or checking its chosen mode.
TEST_F(SOEMMasterTest, ActivateRefusesAUsedSlaveWithNoIdentity) {
    MockSlaveInfo unidentified;
    unidentified.state = EC_STATE_PRE_OP;
    unidentified.eep_man = 0;
    unidentified.eep_id = 0;
    unidentified.name = "? M:00000000 I:00000000";
    unidentified.Ibits = 16;
    this->add_slave(unidentified);
    this->add_default_slave();
    ASSERT_NIL_P(this->master->scan());
    this->configs = {
        {1, {.op_mode = dc::Choice{.index = 2, .assign_activate = 0x0700}}}
    };
    const auto err = this->activate();
    ASSERT_OCCURRED_AS(err, errors::ACTIVATION_ERROR);
    EXPECT_EQ(
        err.data,
        "slave 1 (? M:00000000 I:00000000) gave no identity when the Driver read its "
        "EEPROM"
    );
    const std::map<uint16_t, slave::Refusal> expected = {
        {1, {.reason = "gave no identity when the Driver read its EEPROM"}},
    };
    EXPECT_EQ(this->activation.refusals, expected);
    EXPECT_EQ(this->image.slaves.size(), 2);
    EXPECT_EQ(this->mock->context.slavelist[1].Ibits, 0);
    EXPECT_EQ(this->mock->context.slavelist[2].Ibits, 32);
    EXPECT_TRUE(this->mock->state_change_requests.empty());
}

/// @brief it should refuse a used slave with no identity when it is the only slave
/// with process data.
TEST_F(SOEMMasterTest, ActivateRefusesTheOnlySlaveWithIOWhenItHasNoIdentity) {
    MockSlaveInfo unidentified;
    unidentified.state = EC_STATE_PRE_OP;
    unidentified.eep_man = 0;
    unidentified.eep_id = 0;
    unidentified.name = "? M:00000000 I:00000000";
    unidentified.Ibits = 16;
    this->add_slave(unidentified);
    ASSERT_NIL_P(this->master->scan());
    const auto err = this->activate();
    ASSERT_OCCURRED_AS(err, errors::ACTIVATION_ERROR);
    EXPECT_EQ(
        err.data,
        "slave 1 (? M:00000000 I:00000000) gave no identity when the Driver read its "
        "EEPROM"
    );
    EXPECT_EQ(this->activation.refusals.size(), 1);
    EXPECT_TRUE(this->activation.refusals.contains(1));
}

/// @brief it should start a bus with a slave that has no identity when no task uses
/// the slave.
TEST_F(SOEMMasterTest, ActivateStartsPastAnUnusedSlaveWithNoIdentity) {
    MockSlaveInfo unidentified;
    unidentified.state = EC_STATE_PRE_OP;
    unidentified.eep_man = 0;
    unidentified.eep_id = 0;
    this->add_slave(unidentified);
    this->add_default_slave();
    ASSERT_NIL_P(this->master->scan());
    this->configs = {{1, {.enabled = false}}};
    ASSERT_NIL(this->activate());
    EXPECT_TRUE(this->activation.refusals.empty());
}

/// @brief it should mark the refusal of a slave that reports "Invalid DC SYNC
/// configuration", so that each message about it gives the DC sync advice.
TEST_F(SOEMMasterTest, ActivateMarksARefusalWithAnInvalidDCSync) {
    this->add_pending_and_refusing_slaves(0x0030);
    ASSERT_OCCURRED_AS(this->activate(), errors::STATE_CHANGE_REFUSED);
    const std::map<uint16_t, slave::Refusal> expected = {
        {2,
         {.reason = "refused SAFE-OP: Invalid DC SYNC configuration",
          .advice = slave::DC_SYNC_ADVICE,
          .code = 0x0030}},
    };
    EXPECT_EQ(this->activation.refusals, expected);
}

/// @brief it should give the mode restart advice and the AL status code with the
/// refusal of a slave that reports an invalid input or output configuration.
TEST_F(SOEMMasterTest, ActivateGivesTheModeRestartAdviceWithAnInvalidConfiguration) {
    for (const uint16_t code: {0x001D, 0x001E}) {
        this->create_master(x::telem::SECOND);
        this->add_pending_and_refusing_slaves(code);
        ASSERT_OCCURRED_AS(this->activate(), errors::STATE_CHANGE_REFUSED);
        EXPECT_EQ(this->activation.refusals.at(2).advice, slave::MODE_RESTART_ADVICE);
        EXPECT_EQ(this->activation.refusals.at(2).code, code);
    }
}

/// @brief it should record why a slave refused OP after SAFE-OP when another slave with
/// process data reached OP.
TEST_F(SOEMMasterTest, ActivateRecordsASlaveThatRefusesOp) {
    for (const auto *name: {"RunningSlave", "RefusingSlave"}) {
        MockSlaveInfo slave;
        slave.state = EC_STATE_PRE_OP;
        slave.name = name;
        slave.Ibits = 16;
        this->add_slave(slave);
    }
    ASSERT_NIL_P(this->master->scan());
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    this->mock->inputs_wkc = 1;
    this->mock->receive_return = 3;
    this->mock->slave_infos[1].ALstatuscode = 0x0026;
    this->mock->slave_infos[1].bus_state = EC_STATE_SAFE_OP + EC_STATE_ERROR;
    const auto err = this->activate();
    ASSERT_OCCURRED_AS(err, errors::STATE_CHANGE_REFUSED);
    EXPECT_EQ(
        err.data,
        "failed to reach OP: slave 2 (RefusingSlave) is in SAFE-OP: Inconsistent "
        "settings"
    );
    EXPECT_EQ(this->mock->slave_infos[0].state, EC_STATE_OPERATIONAL);
    const std::map<uint16_t, slave::Refusal> expected = {
        {2, {.reason = "refused OP: Inconsistent settings", .code = 0x0026}},
    };
    EXPECT_EQ(this->activation.refusals, expected);
}

/// @brief it should forget the refusals of the last start when a new start begins.
TEST_F(SOEMMasterTest, ActivateForgetsTheRefusalsOfTheLastStart) {
    this->add_pending_and_refusing_slaves();
    ASSERT_OCCURRED_AS(this->activate(), errors::STATE_CHANGE_REFUSED);
    this->master->deactivate();
    ASSERT_NIL_P(this->master->scan());
    this->configs = {{2, {.enabled = false}}};
    this->mock->slave_infos[0].ALstatuscode = 0x001D;
    ASSERT_OCCURRED_AS(this->activate(), errors::STATE_CHANGE_REFUSED);
    const std::map<uint16_t, slave::Refusal> expected = {
        {1,
         {.reason = "refused SAFE-OP: Invalid output configuration",
          .advice = slave::MODE_RESTART_ADVICE,
          .code = 0x001D}},
    };
    EXPECT_EQ(this->activation.refusals, expected);
}

/// @brief it should report that slaves refused SAFE-OP with "Invalid DC SYNC
/// configuration", and leave the advice out of its error.
TEST_F(SOEMMasterTest, ActivateReportsADCSyncRefusal) {
    MockSlaveInfo first;
    first.state = EC_STATE_PRE_OP;
    first.name = "FirstSlave";
    first.ALstatuscode = 0x0030;
    this->add_slave(first);
    MockSlaveInfo second = first;
    second.name = "SecondSlave";
    this->add_slave(second);
    ASSERT_NIL_P(this->master->scan());
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
        "configuration"
    );
    EXPECT_EQ(this->activation.refusals.at(1).advice, slave::DC_SYNC_ADVICE);
    EXPECT_EQ(this->activation.refusals.at(2).advice, slave::DC_SYNC_ADVICE);
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

/// @brief it should fail to activate when a slave does not reach OP.
TEST_F(SOEMMasterTest, ActivateOpTransitionFails) {
    this->initialize_with_slave();
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    this->mock->statecheck_overrides[EC_STATE_OPERATIONAL] = EC_STATE_SAFE_OP;
    ASSERT_OCCURRED_AS(this->activate(), errors::STATE_CHANGE_ERROR);
}

/// @brief it should start the bus cycle once send() works and before it requests SAFE-
/// OP.
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
    this->mock->context.slavelist[1].hasdc = 1;
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

/// @brief it should fail a send before SAFE-OP and the receive after it when the
/// reference clock frame does not come back, so no caller reads a stale DC time.
TEST_F(SOEMMasterTest, SendAndReceiveFailWhenTheReferenceClockFrameIsLost) {
    this->initialize_with_slave();
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    this->mock->context.slavelist[1].hasdc = 1;
    this->mock->sync_dc_return = 0;
    x::errors::Error send_err;
    x::errors::Error receive_err;
    static_cast<void>(this->master->activate({.period = PERIOD}, [&](const auto &) {
        send_err = this->master->send();
        receive_err = this->master->receive();
    }));
    ASSERT_OCCURRED_AS(send_err, errors::CYCLIC_ERROR);
    EXPECT_EQ(send_err.data, "reference clock frame failed");
    ASSERT_OCCURRED_AS(receive_err, errors::CYCLIC_ERROR);
    EXPECT_EQ(receive_err.data, "reference clock frame failed");
}

/// @brief it should map the slaves with no SAFE-OP request from SOEM, so the slaves
/// enter SAFE-OP only once the bus cycles, and let the next config_init request PRE-OP.
TEST_F(SOEMMasterTest, ActivateMapsTheSlavesWithoutASAFEOPRequest) {
    this->initialize_with_slave();
    this->activate_master();
    EXPECT_EQ(this->mock->manual_at_map, true);
    EXPECT_EQ(this->mock->context.manualstatechange, 0);
}

/// @brief it should fail activation without a request for OP when no frame passes the
/// slaves in SAFE-OP, since a slave with outputs refuses OP until one does.
TEST_F(SOEMMasterTest, ActivateRequestsOPOnlyAfterAFramePassesInSAFEOP) {
    this->initialize_with_slave();
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    this->mock->state_change_requests.clear();
    const auto err = this->master->activate({.period = PERIOD}, no_cycle).err;
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

/// @brief it should put a disabled slave in a group that does not exchange process
/// data.
TEST_F(SOEMMasterTest, ActivateDisabledSlavesSetGroup) {
    MockSlaveInfo slave1;
    slave1.state = EC_STATE_PRE_OP;
    slave1.name = "ActiveSlave";
    this->add_slave(slave1);

    MockSlaveInfo slave2;
    slave2.state = EC_STATE_PRE_OP;
    slave2.name = "DisabledSlave";
    this->add_slave(slave2);

    ASSERT_NIL_P(this->master->scan());
    this->configs = {{2, {.enabled = false}}};
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    ASSERT_NIL(this->activate());
    EXPECT_EQ(this->mock->context.slavelist[1].group, 1);
    EXPECT_EQ(this->mock->context.slavelist[2].group, 2);
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

    ASSERT_NIL_P(this->master->scan());
    this->configs = {{2, {.enabled = false}}};
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
    ASSERT_NIL_P(this->master->scan());
    this->configs = {{3, {.enabled = false}}};
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    ASSERT_NIL(this->activate());
    const std::vector<RegisterWrite> expected = {
        {.slave = 1, .address = 0x0980, .value = 0, .size = 1},
        {.slave = 1, .address = 0x0981, .value = 0, .size = 1},
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
    ASSERT_NIL_P(this->master->scan());
    this->mock->write_register_return = 0;
    const auto err = this->activate();
    ASSERT_OCCURRED_AS(err, errors::ACTIVATION_ERROR);
    EXPECT_EQ(err.data, "slave 1 (EL3104) refused a write to DC register 0x0980");
    EXPECT_EQ(this->mock->mapped_group, std::nullopt);
}

/// @brief it should tell a DC register write that got no reply from one that the slave
/// refused.
TEST_F(SOEMMasterTest, ActivateReportsADCWriteThatGotNoReply) {
    MockSlaveInfo dc;
    dc.state = EC_STATE_PRE_OP;
    dc.hasdc = true;
    dc.name = "EL3104";
    this->add_slave(dc);
    ASSERT_NIL_P(this->master->scan());
    this->mock->write_register_return = EC_NOFRAME;
    const auto err = this->activate();
    ASSERT_OCCURRED_AS(err, errors::ACTIVATION_ERROR);
    EXPECT_EQ(
        err.data,
        "slave 1 (EL3104) did not reply to a write to DC register 0x0980"
    );
}

/// @brief it should give each read and write of a DC register the SOEM timeout that
/// sends the frame up to three times.
TEST_F(SOEMMasterTest, ActivateGivesEachDCRegisterAccessThreeTries) {
    this->initialize_with_dc_slave({DC_ONLY});
    ASSERT_NIL(this->activate());
    ASSERT_FALSE(this->mock->register_timeouts.empty());
    // SOEM sends a frame again only when its timeout is longer than EC_TIMEOUTRET.
    EXPECT_EQ(EC_TIMEOUTRET3, EC_TIMEOUTRET * 3);
    EXPECT_TRUE(std::ranges::all_of(this->mock->register_timeouts, [](const int t) {
        return t == EC_TIMEOUTRET3;
    }));
}

/// @brief it should start SYNC0 and SYNC1 on a slave that lists only a DC mode, before
/// it requests SAFE-OP, with the first SYNC0 event on a whole SYNC0 cycle at least 100
/// ms after the system time of the slave.
TEST_F(SOEMMasterTest, ActivateStartsSyncOnASlaveWithOnlyADCMode) {
    this->initialize_with_dc_slave({DC_ONLY});
    ASSERT_NIL(this->activate());
    const std::vector<RegisterWrite> expected = {
        {.slave = 1, .address = 0x0980, .value = 0, .size = 1},
        {.slave = 1, .address = 0x0981, .value = 0, .size = 1},
        {.slave = 1, .address = 0x0980, .value = 0, .size = 1, .mapped = true},
        {.slave = 1, .address = 0x09A0, .value = 1000, .size = 4, .mapped = true},
        {.slave = 1,
         .address = 0x09A4,
         .value = 999'999'000,
         .size = 4,
         .mapped = true},
        {.slave = 1,
         .address = 0x0990,
         .value = 5'100'001'000,
         .size = 8,
         .mapped = true},
        {.slave = 1, .address = 0x0981, .value = 0x07, .size = 1, .mapped = true},
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
    EXPECT_EQ(this->mock->register_writes[5].value, 6'000'250'000);
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
    EXPECT_EQ(this->mock->register_writes[5].value, 5'101'000'000);
}

/// @brief it should start the cyclic unit 100 ms after the system time of the slave
/// when the mode leaves SYNC0 off, since there is no SYNC0 cycle to align to.
TEST_F(SOEMMasterTest, ActivateStartsTheCyclicUnitOfAModeWithSYNC0Off) {
    this->initialize_with_dc_slave({{.name = "DC", .assign_activate = 0x0100}});
    ASSERT_NIL(this->activate());
    ASSERT_EQ(this->mock->register_writes.size(), 7);
    EXPECT_EQ(this->mock->register_writes[3].value, 0);
    EXPECT_EQ(this->mock->register_writes[5].value, 5'100'000'000);
    EXPECT_EQ(this->mock->register_writes[6].value, 0x01);
}

/// @brief it should leave DC sync off on a slave that lists an SM mode.
TEST_F(SOEMMasterTest, ActivateLeavesSyncOffOnASlaveWithAnSMMode) {
    this->initialize_with_dc_slave({SM, DC_SECOND});
    ASSERT_NIL(this->activate());
    const std::vector<RegisterWrite> expected = {
        {.slave = 1, .address = 0x0980, .value = 0, .size = 1},
        {.slave = 1, .address = 0x0981, .value = 0, .size = 1},
    };
    EXPECT_EQ(this->mock->register_writes, expected);
}

/// @brief it should run the DC mode that the catalog gives for the mode a slave holds
/// when no mode is chosen, in place of the SM mode that it would pick.
TEST_F(SOEMMasterTest, ActivateRunsTheCatalogModeWhenNoModeIsChosen) {
    this->initialize_with_dc_slave(
        create_stg(ECT_MBXPROT_COE),
        {SM, {.name = "DC", .assign_activate = 0x0300}}
    );
    const auto catalog = this->master->scan().first.at(0).properties.catalog_op_mode();
    ASSERT_TRUE(catalog.has_value());
    EXPECT_EQ(catalog.value().assign_activate, 0x0300);
    ASSERT_NIL(this->activate());
    ASSERT_EQ(this->mock->register_writes.size(), 7);
    EXPECT_EQ(this->mock->register_writes[3].address, 0x09A0);
    EXPECT_EQ(this->mock->register_writes[3].value, 1000);
    EXPECT_EQ(this->mock->register_writes[4].address, 0x09A4);
    EXPECT_EQ(this->mock->register_writes[4].value, 999'999'000);
    EXPECT_EQ(this->mock->register_writes[6].address, 0x0981);
    EXPECT_EQ(this->mock->register_writes[6].value, 0x07);
}

/// @brief it should leave DC sync off on a disabled slave.
TEST_F(SOEMMasterTest, ActivateLeavesSyncOffOnADisabledSlave) {
    this->initialize_with_dc_slave({DC_ONLY});
    this->configs = {{1, {.enabled = false}}};
    ASSERT_NIL(this->activate());
    EXPECT_TRUE(this->mock->register_writes.empty());
}

/// @brief it should leave DC sync off on a disabled slave, even when the slave cannot
/// use its chosen mode.
TEST_F(SOEMMasterTest, ActivateLeavesSyncOffOnADisabledSlaveWithAChosenMode) {
    this->initialize_with_dc_slave({DC_ONLY});
    this->configs = {
        {1, {.enabled = false, .op_mode = dc::Choice{.assign_activate = 0x0700}}}
    };
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

/// @brief it should fail activation before it writes a DC register when the SYNC0
/// events of the slaves do not line up with the bus cycle, and name each slave with its
/// SYNC0 cycle.
TEST_F(SOEMMasterTest, ActivateFailsWhenSYNC0EventsDoNotLineUpWithTheBusCycle) {
    MockSlaveInfo slave;
    slave.state = EC_STATE_PRE_OP;
    slave.hasdc = true;
    slave.name = "16xAO";
    this->add_slave(slave);
    slave.name = "16xLV";
    this->add_slave(slave);
    this->set_op_modes(
        {{.name = "DC", .assign_activate = 0x0300, .sync0_cycle = 150'000}}
    );
    ASSERT_NIL_P(this->master->scan());
    const auto err = this->activate();
    ASSERT_OCCURRED_AS(err, errors::SLAVE_CONFIG_ERROR);
    EXPECT_EQ(
        err.data,
        "SYNC0 events do not line up with the bus cycle of 1ms: slave 1 (16xAO) every "
        "150us, slave 2 (16xLV) every 150us. Change the bus rate or the SYNC0 cycles "
        "so that a span of 100us or more divides the bus cycle and each SYNC0 cycle."
    );
    EXPECT_TRUE(this->mock->register_writes.empty());
}

/// @brief it should run the chosen DC mode of a slave that also lists an SM mode, with
/// the chosen cycles.
TEST_F(SOEMMasterTest, ActivateRunsTheChosenModeWithItsCycles) {
    this->initialize_with_dc_slave({SM, DC_SECOND});
    const dc::Choice choice{
        .index = 1,
        .assign_activate = 0x0700,
        .cycles = {.sync1 = x::telem::SECOND * 2},
    };
    this->configs = {{1, {.op_mode = choice}}};
    ASSERT_NIL(this->activate());
    EXPECT_EQ(
        this->image.dc_sync,
        std::vector{dc::resolve(DC_SECOND, PERIOD, choice.cycles).first}
    );
}

/// @brief it should fail activation before it writes a DC register when the slave does
/// not list its chosen mode, and list the modes it has.
TEST_F(SOEMMasterTest, ActivateFailsWhenTheSlaveDoesNotListItsChosenMode) {
    this->initialize_with_dc_slave({SM, DC_SECOND});
    this->configs = {
        {1, {.op_mode = dc::Choice{.index = 2, .assign_activate = 0x0700}}}
    };
    const auto err = this->activate();
    ASSERT_OCCURRED_AS(err, errors::SLAVE_CONFIG_ERROR);
    EXPECT_EQ(
        err.data,
        "slave 1 (16xAO) cannot use its chosen mode: the slave does not list mode 2 "
        "with AssignActivate 0x0700. It lists 0 \"SM-Synchron\" (AssignActivate "
        "0x0000), 1 \"DC\" (AssignActivate 0x0700)."
    );
    EXPECT_TRUE(this->mock->register_writes.empty());
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
        (RegisterWrite{
            .slave = 1,
            .address = 0x0980,
            .value = 0x01,
            .size = 1,
            .mapped = true
        })
    );
}

/// @brief it should leave a sync unit that the PDI owns out of the SYNC0 line-up check,
/// since the master does not start its SYNC0 events.
TEST_F(SOEMMasterTest, ActivateIgnoresTheSYNC0CycleOfASyncUnitThePDIOwns) {
    this->initialize_with_dc_slave(
        {{.name = "DC", .assign_activate = 0x0301, .sync0_cycle = 150'000}}
    );
    ASSERT_NIL(this->activate());
}

/// @brief it should fail activation, before it starts SYNC0 or requests SAFE-OP, when
/// the clock of a slave stays more than 10 us off the reference clock.
TEST_F(SOEMMasterTest, ActivateFailsWhenADCClockDoesNotSettle) {
    this->initialize_with_dc_slave({DC_ONLY});
    this->mock->register_values[{1, 0x092C}] = 20'000;
    const auto err = this->activate();
    ASSERT_OCCURRED_AS(err, errors::ACTIVATION_ERROR);
    EXPECT_EQ(
        err.data,
        "the clock of slave 1 (16xAO) is " + x::telem::TimeSpan(20'000).to_string() +
            " off the reference clock after " + x::telem::SECOND.to_string() +
            ". It must be within " + (x::telem::MICROSECOND * 10).to_string() +
            " before SYNC0 starts."
    );
    EXPECT_EQ(this->mock->register_writes.size(), 2);
    EXPECT_TRUE(this->mock->state_change_requests.empty());
}

/// @brief it should fail activation, before it starts SYNC0 or requests SAFE-OP, when
/// no frame carries the reference clock to the slaves, since the clock difference of a
/// slave keeps its old value until one does.
TEST_F(SOEMMasterTest, ActivateReadsTheClockDifferenceOnlyAfterAFrame) {
    this->create_master(x::telem::MILLISECOND * 50);
    this->initialize_with_dc_slave({DC_ONLY});
    const auto err = this->master->activate({.period = PERIOD}, no_cycle).err;
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
    this->mock->register_values[{1, 0x092C}] = 0x80000000 | 5'000;
    ASSERT_NIL(this->activate());
    EXPECT_EQ(this->mock->register_writes.size(), 7);
}

/// @brief it should fail activation when a slave refuses a read of its clock
/// difference.
TEST_F(SOEMMasterTest, ActivateFailsWhenADCClockCannotBeRead) {
    this->initialize_with_dc_slave({DC_ONLY});
    this->mock->read_register_return = 0;
    const auto err = this->activate();
    ASSERT_OCCURRED_AS(err, errors::ACTIVATION_ERROR);
    EXPECT_EQ(err.data, "slave 1 (16xAO) refused a read of DC register 0x092C");
}

/// @brief it should turn DC sync off when it stops the bus.
TEST_F(SOEMMasterTest, DeactivateClearsDCActivation) {
    this->initialize_with_dc_slave({DC_ONLY});
    ASSERT_NIL(this->activate());
    this->mock->register_writes.clear();
    this->master->deactivate();
    const std::vector<RegisterWrite> expected = {
        {.slave = 1, .address = 0x0980, .value = 0, .size = 1, .mapped = true},
        {.slave = 1, .address = 0x0981, .value = 0, .size = 1, .mapped = true},
    };
    EXPECT_EQ(this->mock->register_writes, expected);
}

/// @brief it should turn DC sync off on every slave when it stops the bus, also after a
/// slave fails a write.
TEST_F(SOEMMasterTest, DeactivateClearsDCActivationPastAFailedSlave) {
    MockSlaveInfo dc;
    dc.state = EC_STATE_PRE_OP;
    dc.hasdc = true;
    this->add_slave(dc);
    this->add_slave(dc);
    ASSERT_NIL_P(this->master->scan());
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    ASSERT_NIL(this->activate());
    this->mock->register_writes.clear();
    this->mock->unwritable_slaves = {1};
    this->master->deactivate();
    const std::vector<RegisterWrite> expected = {
        {.slave = 1, .address = 0x0980, .value = 0, .size = 1, .mapped = true},
        {.slave = 2, .address = 0x0980, .value = 0, .size = 1, .mapped = true},
        {.slave = 2, .address = 0x0981, .value = 0, .size = 1, .mapped = true},
    };
    EXPECT_EQ(this->mock->register_writes, expected);
}

/// @brief it should report the DC registers of each slave whose SYNC0 events it starts,
/// until it stops the bus.
TEST_F(SOEMMasterTest, DCSyncReportsEachSlaveThatRunsSYNC0) {
    MockSlaveInfo dc;
    dc.state = EC_STATE_PRE_OP;
    dc.hasdc = true;
    this->add_slave(dc);
    this->add_slave(dc);
    this->set_op_modes({DC_SECOND});
    ASSERT_NIL_P(this->master->scan());
    this->mock->obytes = 2;
    this->mock->ibytes = 4;
    this->mock->outputs_wkc = 2;
    this->mock->inputs_wkc = 2;
    this->mock->receive_return = 6;
    ASSERT_NIL(this->activate());
    const auto regs = dc::resolve(DC_SECOND, PERIOD).first;
    EXPECT_EQ(this->image.dc_sync, (std::vector{regs, regs}));
}

/// @brief it should report no DC sync when the PDI owns the sync unit of the slave.
TEST_F(SOEMMasterTest, DCSyncIsEmptyWhenThePDIOwnsTheSyncUnit) {
    this->initialize_with_dc_slave(
        {{.name = "DC", .assign_activate = 0x0301, .sync0_factor = 1}}
    );
    ASSERT_NIL(this->activate());
    EXPECT_TRUE(this->image.dc_sync.empty());
}

/// @brief it should report no DC sync when the slave runs in an SM mode.
TEST_F(SOEMMasterTest, DCSyncIsEmptyInSMMode) {
    this->initialize_with_dc_slave({SM, DC_SECOND});
    ASSERT_NIL(this->activate());
    EXPECT_TRUE(this->image.dc_sync.empty());
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

/// @brief it should fail a receive when no frame comes back.
TEST_F(SOEMMasterTest, ReceiveNegativeWkc) {
    this->initialize_with_slave();
    this->activate_master();
    this->mock->receive_return = -1;
    ASSERT_OCCURRED_AS(this->master->receive(), errors::CYCLIC_ERROR);
}

/// @brief it should fail a receive whose working counter is not the expected count.
TEST_F(SOEMMasterTest, ReceiveWkcMismatch) {
    this->initialize_with_slave();
    this->activate_master();
    this->mock->receive_return = 1;
    const auto err = this->master->receive();
    ASSERT_OCCURRED_AS(err, errors::WORKING_COUNTER_ERROR);
    EXPECT_EQ(err.data, "working counter mismatch: expected 3, got 1");
    EXPECT_EQ(this->master->shortfall(), 2);
}

/// @brief it should report no shortfall for a working counter above the expected count.
TEST_F(SOEMMasterTest, ReceiveReportsNoShortfallAboveTheExpectedCount) {
    this->initialize_with_slave();
    this->activate_master();
    this->mock->receive_return = 4;
    ASSERT_OCCURRED_AS(this->master->receive(), errors::WORKING_COUNTER_ERROR);
    EXPECT_EQ(this->master->shortfall(), std::nullopt);
}

/// @brief it should give each enabled slave a working counter share of 2 for its
/// outputs and 1 for its inputs, and none to a slave without process data.
TEST_F(SOEMMasterTest, ImageGivesTheWorkingCounterShareOfEachSlave) {
    this->add_default_slave();
    const auto add = [this](const uint16_t ibits, const uint16_t obits) {
        MockSlaveInfo slave;
        slave.state = EC_STATE_PRE_OP;
        slave.Ibits = ibits;
        slave.Obits = obits;
        this->add_slave(slave);
    };
    add(16, 0);
    add(0, 8);
    add(0, 0);
    ASSERT_NIL_P(this->master->scan());
    this->activate_master();
    EXPECT_EQ(
        this->image.working_counter_shares,
        (std::map<uint16_t, uint32_t>{{1, 3}, {2, 1}, {3, 2}})
    );
}

/// @brief it should accept a receive whose working counter is the expected count.
TEST_F(SOEMMasterTest, ReceiveWkcMatch) {
    this->initialize_with_slave();
    this->activate_master();
    ASSERT_NIL(this->master->receive());
}

////////////////////// Send //////////////////////

/// @brief it should fail a send that SOEM cannot put on the wire.
TEST_F(SOEMMasterTest, SendReturnsZero) {
    this->initialize_with_slave();
    this->activate_master();
    this->mock->send_return = 0;
    ASSERT_OCCURRED_AS(this->master->send(), errors::CYCLIC_ERROR);
}

/// @brief it should send the process data of an active bus.
TEST_F(SOEMMasterTest, SendSucceeds) {
    this->initialize_with_slave();
    this->activate_master();
    ASSERT_NIL(this->master->send());
}

////////////////////// Data Access //////////////////////

/// @brief it should give the cycle the outputs and inputs of the enabled slaves.
TEST_F(SOEMMasterTest, ActivateGivesTheProcessImage) {
    this->initialize_with_slave();
    this->activate_master();
    EXPECT_EQ(this->image.outputs.size(), 2);
    EXPECT_EQ(this->image.inputs.size(), 4);
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
    this->mock->context.slavelist[1].hasdc = 1;
    this->activate_master();
    this->mock->context.DCtime = 1'000'000'000;
    EXPECT_EQ(
        this->master->dc_time(),
        x::telem::TimeStamp(x::telem::SECOND * 946'684'801)
    );
}

/// @brief it should read the reference clock time in the frames of the enabled slaves
/// when the reference slave, the first slave with a distributed clock, is disabled.
TEST_F(SOEMMasterTest, DCTimeIsPresentWhenTheReferenceSlaveIsDisabled) {
    MockSlaveInfo reference;
    reference.state = EC_STATE_PRE_OP;
    reference.hasdc = true;
    this->add_slave(reference);
    MockSlaveInfo enabled;
    enabled.state = EC_STATE_PRE_OP;
    enabled.hasdc = true;
    this->add_slave(enabled);
    ASSERT_NIL_P(this->master->scan());
    this->configs = {{1, {.enabled = false}}};
    this->activate_master();
    this->mock->context.DCtime = 1'000'000'000;
    EXPECT_EQ(this->mock->context.grouplist[1].hasdc, 1);
    EXPECT_EQ(this->mock->context.grouplist[1].DCnext, 1);
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

/// @brief it should report a slave that does not answer a second read as lost.
TEST_F(SOEMMasterTest, CheckMarksSilentSlaveLost) {
    this->initialize_with_slave();
    this->activate_master();
    this->mock->slave_infos[0].state = EC_STATE_NONE;
    this->mock->statecheck_overrides[EC_STATE_OPERATIONAL] = EC_STATE_NONE;
    this->mock->recover_return = 0;
    EXPECT_TRUE(this->master->check(this->brk)[0].lost);
    EXPECT_TRUE(this->master->check(this->brk)[0].lost);
    EXPECT_EQ(this->mock->recovered_slaves, (std::vector<uint16_t>{1, 1}));
}

/// @brief it should report each slave in OP when a state read gets no reply.
TEST_F(SOEMMasterTest, CheckKeepsStatesOnDroppedStateRead) {
    this->add_default_slave();
    this->add_default_slave();
    ASSERT_NIL_P(this->master->scan());
    this->activate_master();
    this->mock->dropped_state_reads = 1;
    const auto health = this->master->check(this->brk);
    ASSERT_EQ(health.size(), 2);
    for (const auto &h: health) {
        EXPECT_EQ(h.state, slave::State::OP);
        EXPECT_FALSE(h.lost);
    }
    EXPECT_TRUE(this->mock->reconfigured_slaves.empty());
    EXPECT_TRUE(this->mock->recovered_slaves.empty());
}

/// @brief it should report the state a slave reads again when a state read gets no
/// reply.
TEST_F(SOEMMasterTest, CheckReportsTheStateReadAgainAfterDroppedStateRead) {
    this->initialize_with_slave();
    this->activate_master();
    this->mock->dropped_state_reads = 1;
    this->mock->statecheck_overrides[EC_STATE_OPERATIONAL] = EC_STATE_SAFE_OP +
                                                             EC_STATE_ERROR;
    const auto health = this->master->check(this->brk);
    ASSERT_EQ(health.size(), 1);
    EXPECT_EQ(health[0].state, slave::State::SAFE_OP);
    EXPECT_TRUE(health[0].faulted);
    EXPECT_FALSE(health[0].lost);
}

/// @brief it should read a slave again after a dropped state read once the breaker
/// stops.
TEST_F(SOEMMasterTest, CheckReadsSlaveAgainOnceBreakerStops) {
    this->initialize_with_slave();
    this->activate_master();
    this->mock->dropped_state_reads = 1;
    this->brk.stop();
    EXPECT_EQ(this->master->check(this->brk)[0].state, slave::State::OP);
}

/// @brief it should recover a lost slave and clear its lost flag.
TEST_F(SOEMMasterTest, CheckRecoversLostSlave) {
    this->initialize_with_slave();
    this->activate_master();
    this->mock->slave_infos[0].state = EC_STATE_NONE;
    this->mock->context.slavelist[1].islost = 1;
    this->master->check(this->brk);
    EXPECT_EQ(this->mock->recovered_slaves, std::vector<uint16_t>{1});
    EXPECT_FALSE(this->mock->context.slavelist[1].islost);
}

/// @brief it should clear the lost flag of a slave that answers again, without a
/// recovery.
TEST_F(SOEMMasterTest, CheckFindsLostSlaveThatAnswers) {
    this->initialize_with_slave();
    this->activate_master();
    this->mock->slave_infos[0].state = EC_STATE_OPERATIONAL;
    this->mock->context.slavelist[1].islost = 1;
    this->master->check(this->brk);
    EXPECT_TRUE(this->mock->recovered_slaves.empty());
    EXPECT_FALSE(this->mock->context.slavelist[1].islost);
}

/// @brief it should leave a disabled slave out of the health report and not write its
/// state.
TEST_F(SOEMMasterTest, CheckSkipsDisabledSlave) {
    this->add_default_slave();
    this->add_default_slave();
    this->configs = {{2, {.enabled = false}}};
    ASSERT_NIL_P(this->master->scan());
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
    this->mock->register_values[{1, 0x0918}] = 1'000'000;
    this->mock->register_values[{1, 0x0920}] = 9'000'000'000;
    this->mock->register_values[{2, 0x0918}] = 2'000;
    this->mock->register_values[{2, 0x0910}] = 5'000'000'000;
    this->check_while_cycling();
    const std::vector<RegisterWrite> expected = {
        {.slave = 2, .address = 0x0980, .value = 0, .size = 1, .mapped = true},
        {.slave = 2, .address = 0x0981, .value = 0, .size = 1, .mapped = true},
        {.slave = 2, .address = 0x0928, .value = 300, .size = 4, .mapped = true},
        {.slave = 2,
         .address = 0x0920,
         .value = 1'000'000 + 9'000'000'000 + 300 - 2'000,
         .size = 8,
         .mapped = true},
        {.slave = 2, .address = 0x0980, .value = 0, .size = 1, .mapped = true},
        {.slave = 2, .address = 0x09A0, .value = 1000, .size = 4, .mapped = true},
        {.slave = 2,
         .address = 0x09A4,
         .value = 999'999'000,
         .size = 4,
         .mapped = true},
        {.slave = 2,
         .address = 0x0990,
         .value = 5'100'001'000,
         .size = 8,
         .mapped = true},
        {.slave = 2, .address = 0x0981, .value = 0x07, .size = 1, .mapped = true},
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
    this->mock->register_values[{1, 0x0918}] = 2'000;
    this->mock->register_values[{2, 0x0918}] = 1'000'000;
    this->mock->register_values[{2, 0x0920}] = 9'000'000'000;
    this->check_while_cycling();
    const std::vector<RegisterWrite> expected = {
        {.slave = 1, .address = 0x0980, .value = 0, .size = 1, .mapped = true},
        {.slave = 1, .address = 0x0981, .value = 0, .size = 1, .mapped = true},
        {.slave = 1, .address = 0x0928, .value = 0, .size = 4, .mapped = true},
        {.slave = 1,
         .address = 0x0920,
         .value = 1'000'000 + 9'000'000'000 - 300 - 2'000,
         .size = 8,
         .mapped = true},
    };
    EXPECT_EQ(this->mock->register_writes, expected);
}

/// @brief it should set the clock of the only DC slave, when it comes back from a power
/// loss, to the DC time of the latest good frame plus the host time since its send.
TEST_F(SOEMMasterTest, CheckRestoresTheOnlyDCSlaveFromTheDCTimeOfTheLatestFrame) {
    this->initialize_with_dc_slave({DC_ONLY});
    this->mock->host_dc_offset = 7'000'000'000;
    this->cycle_gap = std::chrono::milliseconds(10);
    this->activate_master();
    this->mock->register_writes.clear();
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->mock->register_values[{1, 0x0918}] = 2'000;
    this->check_while_cycling();
    const auto error = this->written_offset(1) -
                       (this->mock->latched_at + 7'000'000'000 - 2'000);
    EXPECT_LT(std::abs(error), (5 * x::telem::MILLISECOND).nanoseconds());
    EXPECT_EQ(
        this->mock->register_writes.back(),
        (RegisterWrite{
            .slave = 1,
            .address = 0x0981,
            .value = 0x07,
            .size = 1,
            .mapped = true
        })
    );
    EXPECT_TRUE(this->mock->state_change_requests.empty());
}

/// @brief it should keep the DC time of the latest frame when the only DC slave, which
/// maps no PDOs, comes back from a power loss with its clock reset, since its frames
/// keep a matching working counter.
TEST_F(SOEMMasterTest, CheckIgnoresTheResetClockOfAReturningReferenceClock) {
    this->initialize_with_dc_slave({DC_ONLY});
    this->mock->context.DCtime = 7'000'000'000;
    this->activate_master();
    this->mock->register_writes.clear();
    this->mock->context.DCtime = 1'000;
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->mock->register_values[{1, 0x0918}] = 2'000;
    this->check_while_cycling();
    const auto error = this->written_offset(1) - (7'000'000'000 - 2'000);
    EXPECT_LT(std::abs(error), x::telem::SECOND.nanoseconds());
}

/// @brief it should set the clock of the first DC slave from the latest frame when the
/// whole bus comes back from a power loss, then set the next one from the first, and
/// hold none of them in PRE-OP.
TEST_F(SOEMMasterTest, CheckRestoresEveryDCSlaveAfterTheWholeBusLostPower) {
    this->mock->context.DCtime = 7'000'000'000;
    this->activate_two_dc_slaves({DC_ONLY});
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->mock->slave_infos[1].state = EC_STATE_INIT;
    this->mock->register_values[{1, 0x0918}] = 2'000;
    this->mock->register_values[{2, 0x0918}] = 5'000;
    this->check_while_cycling();
    EXPECT_EQ(this->mock->reconfigured_slaves, (std::vector<uint16_t>{1, 2}));
    const auto first = this->written_offset(1);
    const auto error = first - (7'000'000'000 - 2'000);
    EXPECT_LT(std::abs(error), x::telem::SECOND.nanoseconds());
    EXPECT_EQ(this->written_offset(2), 2'000 + first + 300 - 5'000);
    EXPECT_TRUE(this->mock->state_change_requests.empty());
}

/// @brief it should set the clock of a DC slave from the latest frame, and not from a
/// slave whose restore failed in the same check, when the whole bus comes back from a
/// power loss.
TEST_F(SOEMMasterTest, CheckSetsNoClockFromASlaveWhoseRestoreFailed) {
    this->mock->context.DCtime = 7'000'000'000;
    this->activate_two_dc_slaves({DC_ONLY});
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->mock->slave_infos[1].state = EC_STATE_INIT;
    this->mock->unwritable_slaves.insert(1);
    this->mock->register_values[{1, 0x0918}] = 2'000;
    this->mock->register_values[{2, 0x0918}] = 5'000;
    this->check_while_cycling();
    EXPECT_EQ(this->mock->reconfigured_slaves, (std::vector<uint16_t>{1, 2}));
    const auto error = this->written_offset(2) - (7'000'000'000 - 5'000);
    EXPECT_LT(std::abs(error), x::telem::SECOND.nanoseconds());
}

/// @brief it should send a DC slave that comes back from a power loss to PRE-OP from
/// SAFE-OP, and report why, when the master cannot restore its clock.
TEST_F(SOEMMasterTest, CheckHoldsADCSlaveInPreOpWhenItsClockCannotBeRestored) {
    this->initialize_with_dc_slave({DC_ONLY});
    this->activate_master();
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->mock->latch_return = 0;
    this->master->check(this->brk);
    EXPECT_EQ(this->mock->slave_infos[0].state, EC_STATE_SAFE_OP);
    EXPECT_TRUE(this->mock->state_change_requests.empty());
    const auto health = this->master->check(this->brk);
    EXPECT_EQ(
        this->mock->state_change_requests,
        (std::vector<std::pair<uint16_t, uint16_t>>{{1, EC_STATE_PRE_OP}})
    );
    EXPECT_EQ(this->mock->written_slaves, std::vector<uint16_t>{1});
    ASSERT_EQ(health.size(), 1);
    EXPECT_TRUE(health[0].faulted);
    EXPECT_EQ(health[0].dc_restore_error, "no slave latched its receive time");
}

/// @brief it should tell a latch frame that got no reply from a latch that no slave
/// took.
TEST_F(SOEMMasterTest, CheckReportsALatchFrameThatGotNoReply) {
    this->initialize_with_dc_slave({DC_ONLY});
    this->activate_master();
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->mock->latch_return = EC_NOFRAME;
    this->master->check(this->brk);
    const auto health = this->master->check(this->brk);
    ASSERT_EQ(health.size(), 1);
    EXPECT_EQ(
        health[0].dc_restore_error,
        "the frame that latches the receive times got no reply"
    );
}

/// @brief it should request PRE-OP and never OP on each check of a slave whose clock
/// restore failed while it stays in SAFE-OP, as when a PRE-OP request gets lost.
TEST_F(SOEMMasterTest, CheckKeepsASlaveWhoseClockRestoreFailedOutOfOp) {
    this->initialize_with_dc_slave({DC_ONLY});
    this->activate_master();
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->mock->latch_return = 0;
    this->master->check(this->brk);
    this->mock->slave_infos[0].bus_state = EC_STATE_SAFE_OP;
    this->master->check(this->brk);
    this->master->check(this->brk);
    EXPECT_EQ(
        this->mock->state_change_requests,
        (std::vector<std::pair<uint16_t, uint16_t>>{
            {1, EC_STATE_PRE_OP},
            {1, EC_STATE_PRE_OP},
        })
    );
}

/// @brief it should fail the clock restore of a DC slave that comes back from a power
/// loss, and not set its clock from the host time, when a DC slave that kept running
/// does not answer the read of its clock.
TEST_F(SOEMMasterTest, CheckFailsTheRestoreWhenARunningDCSlaveDoesNotAnswer) {
    this->mock->host_dc_offset = 7'000'000'000;
    this->activate_two_dc_slaves({DC_ONLY});
    this->mock->slave_infos[0].state = EC_STATE_OPERATIONAL;
    this->mock->slave_infos[1].state = EC_STATE_INIT;
    this->mock->unanswered_reads.insert({1, 0x0918});
    this->check_while_cycling();
    EXPECT_TRUE(
        std::ranges::none_of(this->mock->register_writes, [](const RegisterWrite &w) {
            return w.address == 0x0920;
        })
    );
    EXPECT_EQ(
        this->master->check(this->brk)[1].dc_restore_error,
        "slave 1 (16xAO) did not reply to a read of DC register 0x0918"
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
        this->master->check(this->brk)[1].dc_restore_error,
        "no slave latched its receive time"
    );
    this->mock->latch_return = 1;
    this->check_while_cycling();
    const auto health = this->master->check(this->brk);
    EXPECT_FALSE(health[1].faulted);
    EXPECT_TRUE(health[1].dc_restore_error.empty());
}

/// @brief it should set the clock of a DC slave that comes back from a power loss from
/// the DC time of the latest frame with a matching working counter.
TEST_F(SOEMMasterTest, CheckIgnoresTheDCTimeOfAFrameWithAWorkingCounterMismatch) {
    this->initialize_with_dc_slave({DC_ONLY});
    this->mock->host_dc_offset = 7'000'000'000;
    this->activate_master();
    this->mock->receive_return = 2;
    this->mock->host_dc_offset = 9'000'000'000;
    ASSERT_NIL(this->master->send());
    ASSERT_OCCURRED_AS(this->master->receive(), errors::WORKING_COUNTER_ERROR);
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->mock->register_values[{1, 0x0918}] = 2'000;
    this->check_while_cycling();
    const auto error = this->written_offset(1) -
                       (this->mock->latched_at + 7'000'000'000 - 2'000);
    EXPECT_LT(std::abs(error), x::telem::SECOND.nanoseconds());
}

/// @brief it should end the restore of a DC slave whose clock does not settle once the
/// breaker stops, well before the settle timeout.
TEST_F(SOEMMasterTest, CheckEndsAClockRestoreOnceTheBreakerStops) {
    EXPECT_LT(this->stop_during_restore(true), std::chrono::seconds(1));
    EXPECT_EQ(
        this->master->check(this->brk)[0].dc_restore_error,
        "the bus stopped before the clocks "
        "settled"
    );
}

/// @brief it should end the restore of a DC slave that waits for a frame once the
/// breaker stops, well before the settle timeout.
TEST_F(SOEMMasterTest, CheckEndsAFrameWaitOfAClockRestoreOnceTheBreakerStops) {
    EXPECT_LT(this->stop_during_restore(false), std::chrono::seconds(1));
    EXPECT_EQ(
        this->master->check(this->brk)[0].dc_restore_error,
        "the bus stopped before the clocks "
        "settled"
    );
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

/// @brief it should close the socket on deactivate.
TEST_F(SOEMMasterTest, DeactivateResetsState) {
    this->initialize_with_slave();
    this->activate_master();
    this->master->deactivate();
    EXPECT_TRUE(this->mock->close_called);
}

/// @brief it should request INIT from the slaves on deactivate.
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

/// @brief it should give the slaves init_timeout, not state_change_timeout, to return
/// to INIT.
TEST_F(SOEMMasterTest, DeactivateWaitsTheInitTimeoutForInit) {
    auto api = std::make_unique<MockAPI>();
    this->mock = api.get();
    this->master = std::make_unique<Master>(
        std::move(api),
        "eth0",
        Config{
            .pre_op_timeout = x::telem::MILLISECOND * 100,
            .state_change_timeout = x::telem::MILLISECOND * 100,
            .init_timeout = x::telem::MILLISECOND * 300,
        }
    );
    this->initialize_with_slave();
    this->activate_master();
    this->mock->statecheck_overrides[EC_STATE_INIT] = EC_STATE_SAFE_OP;
    const auto start = std::chrono::steady_clock::now();
    this->master->deactivate();
    EXPECT_GE(std::chrono::steady_clock::now() - start, std::chrono::milliseconds(300));
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
    this->mock->host_dc_offset = 7'000'000'000;
    this->activate_master();
    this->master->deactivate();
    ASSERT_NIL_P(this->master->scan());
    this->mock->receive_return = 2;
    ASSERT_NIL(this->activate());
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->master->check(this->brk);
    const auto health = this->master->check(this->brk);
    ASSERT_EQ(health.size(), 1);
    EXPECT_EQ(
        health[0].dc_restore_error,
        "no frame has carried the DC time to "
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
    ASSERT_NIL_P(this->master->scan());
    this->activate_master();
    const auto health = this->master->check(this->brk);
    ASSERT_EQ(health.size(), 1);
    EXPECT_FALSE(health[0].faulted);
    EXPECT_TRUE(health[0].dc_restore_error.empty());
}

/// @brief it should do nothing on a deactivate before a scan.
TEST_F(SOEMMasterTest, DeactivateNotInitializedIsNoop) {
    this->master->deactivate();
    EXPECT_FALSE(this->mock->close_called);
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
    mock.slave_infos[0].sdo_objects = {
        {{0x1C00, 0}, 4},
        {{0x1C00, 3}, 3},
        {{0x1C00, 4}, 4},
        {{ECT_SDO_TXPDOASSIGN, 0}, 1},
        {{ECT_SDO_TXPDOASSIGN, 1}, 0x1A00},
        {{0x1A00, 0}, 3},
        {{0x1A00, 1}, 0x60000101},
        {{0x1A00, 2}, 0x00000007},
        {{0x1A00, 3}, 0x60001110},
        {{ECT_SDO_RXPDOASSIGN, 0}, 0},
    };
    mock.slave_infos[0].oe_entries = {
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
    this->mock->slave_infos[0].od_names[0x1A00] = "Channel 1";
    ASSERT_NIL_P(this->master->scan());

    const auto inputs = this->master->scan().first[0].properties.input_pdos;
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
    ASSERT_NIL_P(this->master->scan());

    const auto inputs = this->master->scan().first[0].properties.input_pdos;
    ASSERT_EQ(inputs.size(), 3);
    EXPECT_FALSE(inputs[0].pdo_name.has_value());
}

/// @brief it should use the mapping read from the slave over the device registry.
TEST_F(SOEMMasterTest, LiveMappingTakesPrecedenceOverRegistry) {
    auto info = coe_slave(24);
    info.eep_man = BECKHOFF;
    info.eep_id = EL1008;
    this->add_slave(info);
    serve_padded_txpdo(*this->mock);
    ASSERT_NIL_P(this->master->scan());

    const auto inputs = this->master->scan().first[0].properties.input_pdos;
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
    ASSERT_NIL_P(this->master->scan());

    const auto inputs = this->master->scan().first[0].properties.input_pdos;
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
    ASSERT_NIL_P(this->master->scan());

    const auto slave = this->master->scan().first[0];
    EXPECT_EQ(slave.properties.input_pdos.size(), 8);
    EXPECT_TRUE(slave.status.pdo_discovery_error.empty());
}

/// @brief it should describe a slave from the device registry when the slave assigns a
/// PDO whose mapping it cannot read, and keep the failed read in its status so the
/// scanner discovers it again.
TEST_F(SOEMMasterTest, RegistryDescribesSlaveWithUnreadableMapping) {
    auto info = coe_slave(8);
    info.eep_man = BECKHOFF;
    info.eep_id = EL1008;
    this->add_slave(info);
    this->mock->slave_infos[0].sdo_objects = {
        {{0x1C00, 0}, 4},
        {{0x1C00, 3}, 3},
        {{0x1C00, 4}, 4},
        {{ECT_SDO_TXPDOASSIGN, 0}, 1},
        {{ECT_SDO_TXPDOASSIGN, 1}, 0x1A00},
        {{ECT_SDO_RXPDOASSIGN, 0}, 0},
    };
    ASSERT_NIL_P(this->master->scan());

    const auto slave = this->master->scan().first[0];
    EXPECT_EQ(slave.properties.input_pdos.size(), 8);
    EXPECT_EQ(
        slave.status.pdo_discovery_error,
        "failed to read the PDO layout: failed to read 0x1A00:00: Object does not "
        "exist in the object dictionary"
    );
}

/// @brief it should count padding when it computes entry offsets.
TEST_F(SOEMMasterTest, OffsetsCountPadding) {
    this->add_slave(coe_slave(24));
    serve_padded_txpdo(*this->mock);
    ASSERT_NIL_P(this->master->scan());
    this->mock->ibytes = 3;
    ASSERT_NIL(this->activate());

    const auto offset = ASSERT_NIL_P(
        this->image.offset(pdo::Entry(1, 0x6000, 17, 16, true, x::telem::INT16_T))
    );
    EXPECT_EQ(offset.byte, 1);
    EXPECT_EQ(offset.bit, 0);
}

namespace {
/// @brief returns a slave with one 4-bit input entry, 0x6000:01, and one 4-bit output
/// entry, 0x7000:01, whose inputs follow one byte of outputs in the IOmap.
MockSlaveInfo nibble_slave() {
    auto info = coe_slave(4);
    info.Obits = 4;
    info.input_offset = 1;
    info.sdo_objects = {
        {{0x1C00, 0}, 4},
        {{0x1C00, 3}, 3},
        {{0x1C00, 4}, 4},
        {{ECT_SDO_TXPDOASSIGN, 0}, 1},
        {{ECT_SDO_TXPDOASSIGN, 1}, 0x1A00},
        {{0x1A00, 0}, 1},
        {{0x1A00, 1}, 0x60000104},
        {{ECT_SDO_RXPDOASSIGN, 0}, 1},
        {{ECT_SDO_RXPDOASSIGN, 1}, 0x1600},
        {{0x1600, 0}, 1},
        {{0x1600, 1}, 0x70000104},
    };
    return info;
}
}

/// @brief it should place the entries of a slave that SOEM packs into the byte of the
/// slave before it at the start bit SOEM gives the slave.
TEST_F(SOEMMasterTest, OffsetsFollowTheStartBitOfAPackedSlave) {
    this->add_slave(nibble_slave());
    auto packed = nibble_slave();
    packed.input_start_bit = 4;
    packed.output_start_bit = 4;
    this->add_slave(packed);
    ASSERT_NIL_P(this->master->scan());
    this->mock->ibytes = 1;
    this->mock->obytes = 1;
    ASSERT_NIL(this->activate());

    const auto input = ASSERT_NIL_P(
        this->image.offset(pdo::Entry(2, 0x6000, 1, 4, true, x::telem::UINT8_T))
    );
    EXPECT_EQ(input.byte, 0);
    EXPECT_EQ(input.bit, 4);
    const auto output = ASSERT_NIL_P(
        this->image.offset(pdo::Entry(2, 0x7000, 1, 4, false, x::telem::UINT8_T))
    );
    EXPECT_EQ(output.byte, 0);
    EXPECT_EQ(output.bit, 4);
}

/// @brief it should return a mapping error for a slave whose discovered entries do not
/// match the bits it maps.
TEST_F(SOEMMasterTest, SlaveWhoseLayoutDisagreesWithMappingHasNoOffsets) {
    this->add_slave(coe_slave(16));
    serve_padded_txpdo(*this->mock);
    ASSERT_NIL_P(this->master->scan());
    this->mock->ibytes = 2;
    ASSERT_NIL(this->activate());

    const auto err = this->image
                         .offset(pdo::Entry(1, 0x6000, 17, 16, true, x::telem::INT16_T))
                         .second;
    ASSERT_OCCURRED_AS(err, errors::PDO_MAPPING_ERROR);
    EXPECT_EQ(
        err.data,
        "Slave 1 (TestSlave) maps 16 input and 0 output bits, but its discovered PDO "
        "entries hold 24 and 0."
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
    this->configs = {{1, {.startup_parameters = {FILTER, RANGE}}}};
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
    this->configs = {{1, {.startup_parameters = {FILTER, RANGE}}}};
    this->mock->slave_infos[0].sdo_write_aborts[{0x8000, 0x15}] = 0x06090031;
    const auto err = this->activate();
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
    this->configs = {{1, {.startup_parameters = {RANGE}}}};
    this->mock->slave_infos[0].sdo_write_aborts[{0x8000, 0x19}] = 0;
    const auto err = this->activate();
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
    this->configs = {{1, {.startup_parameters = {RANGE}}}};
    this->mock->push_sdo_abort(1, 0x1C12, 0, 0x06020000);
    this->mock->slave_infos[0].sdo_write_aborts[{0x8000, 0x19}] = 0;
    const auto err = this->activate();
    ASSERT_OCCURRED_AS(err, errors::STARTUP_PARAMETER_ERROR);
    EXPECT_EQ(
        err.data,
        "slave 1 (TestSlave) rejected startup parameter 0x8000:19: no response"
    );
}

/// @brief it should write no startup parameters after a new config clears them.
TEST_F(SOEMMasterTest, ActivateWritesNothingAfterStartupParametersAreCleared) {
    this->initialize_with_slave();
    this->configs = {{1, {.startup_parameters = {FILTER}}}};
    this->configs = {{1, {}}};
    this->activate_master();
    EXPECT_TRUE(this->mock->sdo_writes.empty());
}

/// @brief it should write only the parameters of the configs that replace the old ones.
TEST_F(SOEMMasterTest, ActivateWritesOnlyReplacedStartupParameters) {
    this->initialize_with_slave();
    this->configs = {{1, {.startup_parameters = {FILTER}}}};
    this->configs = {{1, {.startup_parameters = {RANGE}}}};
    this->activate_master();
    ASSERT_EQ(this->mock->sdo_writes.size(), 1);
    EXPECT_EQ(this->mock->sdo_writes[0].index, RANGE.index);
}

/// @brief it should write nothing to a slave left out of the replacement configs.
TEST_F(SOEMMasterTest, ActivateWritesNothingToSlaveLeftOutOfReplacement) {
    this->initialize_with_slave();
    this->configs = {{1, {.startup_parameters = {FILTER}}}};
    this->configs = {};
    this->activate_master();
    EXPECT_TRUE(this->mock->sdo_writes.empty());
}

/// @brief it should not write the startup parameters of a disabled slave.
TEST_F(SOEMMasterTest, ActivateSkipsStartupParametersOfDisabledSlave) {
    this->initialize_with_slave();
    this->configs = {{1, {.enabled = false, .startup_parameters = {FILTER}}}};
    this->activate_master();
    EXPECT_TRUE(this->mock->sdo_writes.empty());
}

/// @brief it should read the PDO layout after the startup parameters change the PDO
/// assignment.
TEST_F(SOEMMasterTest, ActivateReadsLayoutThatStartupParametersAssign) {
    this->add_slave(coe_slave(24));
    serve_padded_txpdo(*this->mock);
    this->mock->slave_infos[0].sdo_objects[{0x1A01, 0}] = 1;
    this->mock->slave_infos[0].sdo_objects[{0x1A01, 1}] = 0x60100110;
    this->mock->slave_infos[0].oe_entries[{0x6010, 1}] = {"Value", 0x0003};
    ASSERT_NIL_P(this->master->scan());
    const slave::StartupParameter assign_1a01{
        .index = ECT_SDO_TXPDOASSIGN,
        .sub_index = 1,
        .data_type = x::telem::UINT16_T,
        .value = 0x1A01,
    };
    this->configs = {{1, {.startup_parameters = {assign_1a01}}}};
    this->mock->slave_infos[0].Ibits = 16;
    this->mock->ibytes = 2;
    ASSERT_NIL(this->activate());

    const auto offset = ASSERT_NIL_P(
        this->image.offset(pdo::Entry(1, 0x6010, 1, 16, true, x::telem::INT16_T))
    );
    EXPECT_EQ(offset.byte, 0);
    EXPECT_EQ(this->image.slaves[0].properties.input_pdos.size(), 1);
}

/// @brief it should write the startup parameters again when it configures a slave that
/// fell back to INIT.
TEST_F(SOEMMasterTest, CheckWritesStartupParametersAgainOnReconfig) {
    this->initialize_with_slave();
    this->configs = {{1, {.startup_parameters = {FILTER}}}};
    this->activate_master();
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->master->check(this->brk);
    EXPECT_EQ(this->mock->sdo_writes.size(), 2);
    EXPECT_TRUE(this->mock->state_change_requests.empty());
}

/// @brief it should write parameters to a slave on the running bus, and write them
/// again with its startup parameters when the slave recovers.
TEST_F(SOEMMasterTest, WriteReachesTheRunningBusAndReplaysOnRecovery) {
    this->initialize_with_slave();
    this->configs = {{1, {.startup_parameters = {FILTER}}}};
    this->activate_master();
    this->mock->sdo_writes.clear();
    const auto range = slave::StartupParameter{
        .index = 0x8000,
        .sub_index = 0x15,
        .data_type = x::telem::UINT16_T,
        .value = 5,
    };
    ASSERT_NIL(this->master->write(1, {range, RANGE}));
    EXPECT_EQ(
        this->mock->sdo_writes,
        (std::vector<MockAPI::SDOWrite>{
            {1, 0x8000, 0x15, {0x05, 0x00}},
            {1, 0x8000, 0x19, {0x02}},
        })
    );
    this->mock->sdo_writes.clear();
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->master->check(this->brk);
    EXPECT_EQ(
        this->mock->sdo_writes,
        (std::vector<MockAPI::SDOWrite>{
            {1, 0x8000, 0x15, {0x05, 0x00}},
            {1, 0x8000, 0x19, {0x02}},
        })
    );
}

/// @brief it should write parameters to a slave when no task runs the bus.
TEST_F(SOEMMasterTest, WriteReachesASlaveOnAStoppedBus) {
    this->initialize_with_slave();
    ASSERT_NIL(this->master->write(1, {RANGE}));
    EXPECT_EQ(
        this->mock->sdo_writes,
        (std::vector<MockAPI::SDOWrite>{{1, 0x8000, 0x19, {0x02}}})
    );
}

/// @brief it should return the reason a slave rejects a parameter on the running bus,
/// and leave the slave on the bus without a fault.
TEST_F(SOEMMasterTest, WriteReturnsTheRejectionWithoutFaultingTheSlave) {
    this->initialize_with_slave();
    this->activate_master();
    this->mock->slave_infos[0].sdo_write_aborts[{0x8000, 0x15}] = 0x06090031;
    const auto err = this->master->write(1, {FILTER});
    ASSERT_OCCURRED_AS(err, errors::STARTUP_PARAMETER_ERROR);
    EXPECT_EQ(
        err.data,
        "Slave 1 (TestSlave) rejected startup parameter 0x8000:15 (Filter): Value of "
        "parameter written too high."
    );
    const auto health = this->master->check(this->brk);
    ASSERT_EQ(health.size(), 1);
    EXPECT_FALSE(health[0].faulted);
    EXPECT_TRUE(health[0].startup_error.empty());
}

/// @brief it should read the value that a slave on the running bus holds in an object.
TEST_F(SOEMMasterTest, ReadReturnsTheValueOfAnObjectOnTheRunningBus) {
    this->initialize_with_slave();
    this->activate_master();
    this->mock->slave_infos[0].sdo_objects[{0x8040, 0x15}] = 0x40000000;
    EXPECT_EQ(ASSERT_NIL_P(this->master->read(1, 0x8040, 0x15)), 0x40000000);
}

/// @brief it should return the abort of a read of an object that the slave does not
/// have.
TEST_F(SOEMMasterTest, ReadReturnsTheAbortOfAMissingObject) {
    this->initialize_with_slave();
    const auto [value, err] = this->master->read(1, 0x8040, 0x15);
    ASSERT_OCCURRED_AS(err, coe::OBJECT_MISSING);
    EXPECT_EQ(
        err.data,
        "failed to read 0x8040:15: Object does not exist in the object dictionary"
    );
}

/// @brief it should hold a slave in PRE-OP and report the failure when a startup
/// parameter write fails during reconfiguration.
TEST_F(SOEMMasterTest, CheckHoldsSlaveInPreOpWhenReconfigWriteFails) {
    this->initialize_with_slave();
    this->configs = {{1, {.startup_parameters = {FILTER}}}};
    this->activate_master();
    this->mock->slave_infos[0].sdo_write_aborts[{0x8000, 0x15}] = 0x06090031;
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
        health[0].startup_error,
        "rejected startup parameter 0x8000:15 (Filter): Value of parameter written "
        "too high"
    );
}

/// @brief it should write the startup parameters of a DC slave that comes back from a
/// power loss, then restore its clock, in one reconfiguration.
TEST_F(SOEMMasterTest, CheckWritesStartupParametersThenRestoresTheClock) {
    this->activate_two_dc_slaves({DC_ONLY}, {{2, {.startup_parameters = {FILTER}}}});
    this->mock->sdo_writes.clear();
    this->mock->slave_infos[0].state = EC_STATE_OPERATIONAL;
    this->mock->slave_infos[1].state = EC_STATE_INIT;
    this->mock->register_values[{1, 0x0918}] = 1'000'000;
    this->mock->register_values[{1, 0x0920}] = 9'000'000'000;
    this->mock->register_values[{2, 0x0918}] = 2'000;
    this->mock->register_values[{2, 0x0910}] = 5'000'000'000;
    this->check_while_cycling();
    const std::vector<MockAPI::SDOWrite> expected{{2, 0x8000, 0x15, {0x03, 0x00}}};
    EXPECT_EQ(this->mock->sdo_writes, expected);
    ASSERT_FALSE(this->mock->register_writes.empty());
    EXPECT_EQ(
        this->mock->register_writes.back(),
        (RegisterWrite{
            .slave = 2,
            .address = 0x0981,
            .value = 0x07,
            .size = 1,
            .mapped = true
        })
    );
    EXPECT_EQ(this->mock->reconfigured_slaves, std::vector<uint16_t>{2});
}

/// @brief it should leave the clock of a DC slave alone when the slave rejects a
/// startup parameter on its way back from a power loss, and report the rejection.
TEST_F(SOEMMasterTest, CheckRestoresNoClockOnASlaveThatRejectsAStartupParameter) {
    this->activate_two_dc_slaves({DC_ONLY}, {{2, {.startup_parameters = {FILTER}}}});
    this->mock->slave_infos[1].sdo_write_aborts[{0x8000, 0x15}] = 0x06090031;
    this->mock->slave_infos[0].state = EC_STATE_OPERATIONAL;
    this->mock->slave_infos[1].state = EC_STATE_INIT;
    this->check_while_cycling();
    EXPECT_TRUE(this->mock->register_writes.empty());
    const auto health = this->master->check(this->brk);
    ASSERT_EQ(health.size(), 2);
    EXPECT_TRUE(health[1].faulted);
    EXPECT_EQ(
        health[1].startup_error,
        "rejected startup parameter 0x8000:15 (Filter): Value of parameter written "
        "too high"
    );
    EXPECT_TRUE(health[1].dc_restore_error.empty());
}

/// @brief it should not reconfigure a slave again after it rejects a startup parameter
/// during reconfiguration.
TEST_F(SOEMMasterTest, CheckStopsRecoveringSlaveThatRejectsStartupParameter) {
    this->initialize_with_slave();
    this->configs = {{1, {.startup_parameters = {FILTER}}}};
    this->activate_master();
    this->mock->slave_infos[0].sdo_write_aborts[{0x8000, 0x15}] = 0x06090031;
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->master->check(this->brk);
    this->mock->slave_infos[0].sdo_write_aborts.clear();
    this->mock->slave_infos[0].state = EC_STATE_PRE_OP;
    const auto health = this->master->check(this->brk);
    EXPECT_EQ(this->mock->reconfigured_slaves, std::vector<uint16_t>{1});
    ASSERT_EQ(health.size(), 1);
    EXPECT_TRUE(health[0].faulted);
}

/// @brief it should clear the lost flag of a slave that rejected a startup parameter
/// once it answers again.
TEST_F(SOEMMasterTest, CheckClearsLostFlagOfSlaveThatRejectsStartupParameter) {
    this->initialize_with_slave();
    this->configs = {{1, {.startup_parameters = {FILTER}}}};
    this->activate_master();
    this->mock->slave_infos[0].sdo_write_aborts[{0x8000, 0x15}] = 0x06090031;
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->master->check(this->brk);
    this->mock->slave_infos[0].bus_state = EC_STATE_NONE;
    this->mock->statecheck_overrides[EC_STATE_OPERATIONAL] = EC_STATE_NONE;
    EXPECT_TRUE(this->master->check(this->brk)[0].lost);
    this->mock->slave_infos[0].bus_state = EC_STATE_PRE_OP;
    EXPECT_FALSE(this->master->check(this->brk)[0].lost);
}

/// @brief it should recover a slave that rejected a startup parameter once the master
/// activates again.
TEST_F(SOEMMasterTest, ActivateClearsStartupFailureFromRecovery) {
    this->initialize_with_slave();
    this->configs = {{1, {.startup_parameters = {FILTER}}}};
    this->activate_master();
    this->mock->slave_infos[0].sdo_write_aborts[{0x8000, 0x15}] = 0x06090031;
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->master->check(this->brk);
    this->mock->slave_infos[0].sdo_write_aborts.clear();
    this->master->deactivate();
    ASSERT_NIL_P(this->master->scan());
    this->activate_master();
    this->mock->slave_infos[0].state = EC_STATE_INIT;
    this->master->check(this->brk);
    EXPECT_EQ(this->mock->reconfigured_slaves, (std::vector<uint16_t>{1, 1}));
}

namespace {
/// @brief a SOEM master on its mock for the master contract.
class Bus {
    MockAPI *api;
    Master m;

    explicit Bus(std::unique_ptr<MockAPI> api):
        api(api.get()),
        m(std::move(api),
          "eth0",
          {
              .pre_op_timeout = x::telem::MILLISECOND * 100,
              .state_change_timeout = x::telem::MILLISECOND * 100,
              .init_timeout = x::telem::MILLISECOND * 100,
              .dc_settle_timeout = x::telem::MILLISECOND * 50,
          }) {}

public:
    Bus(): Bus(std::make_unique<MockAPI>()) {}

    Master &master() { return this->m; }

    void add_slave() {
        MockSlaveInfo slave;
        slave.state = EC_STATE_PRE_OP;
        slave.name = "Slave " + std::to_string(this->api->slave_infos.size() + 1);
        slave.mbx_proto = ECT_MBXPROT_COE;
        slave.Ibits = 16;
        slave.input_offset = this->api->ibytes;
        slave.sdo_objects = {
            {{0x1C00, 0}, 4},
            {{0x1C00, 3}, 3},
            {{0x1C00, 4}, 4},
            {{ECT_SDO_RXPDOASSIGN, 0}, 0},
            {{ECT_SDO_TXPDOASSIGN, 0}, 1},
            {{ECT_SDO_TXPDOASSIGN, 1}, 0x1A00},
            {{0x1A00, 0}, 1},
            {{0x1A00, 1}, 0x60000110},
        };
        this->api->slave_infos.push_back(std::move(slave));
        this->api->ibytes += 2;
    }

    void refuse(const uint16_t position) {
        this->api->slave_infos[position - 1].ALstatuscode = 0x0026;
        this->api->statecheck_overrides[EC_STATE_SAFE_OP] = EC_STATE_PRE_OP +
                                                            EC_STATE_ERROR;
    }
};
}
}

namespace driver::ethercat::master::contract {
INSTANTIATE_TYPED_TEST_SUITE_P(SOEM, Contract, soem::Bus);
}
