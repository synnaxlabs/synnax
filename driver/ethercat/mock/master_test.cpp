// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <gtest/gtest.h>

#include "x/cpp/test/test.h"

#include "driver/ethercat/mock/master.h"

namespace driver::ethercat::mock {

TEST(MasterConstruction, DefaultInterfaceName) {
    Master master;
    EXPECT_EQ(master.interface_name(), "mock0");
}

TEST(MasterConstruction, CustomInterfaceName) {
    Master master("eth1");
    EXPECT_EQ(master.interface_name(), "eth1");
}

TEST(MasterSlaveManagement, AddSlaveIncreasesCount) {
    Master master;
    EXPECT_EQ(master.slave_count(), 0);
    master.add_slave(slave::Properties{.position = 0});
    EXPECT_EQ(master.slave_count(), 1);
    master.add_slave(slave::Properties{.position = 1});
    EXPECT_EQ(master.slave_count(), 2);
}

TEST(MasterSlaveManagement, AddSlaveInitializesStateToInit) {
    Master master;
    master.add_slave(slave::Properties{.position = 0});
    EXPECT_EQ(master.slave_state(0), slave::State::INIT);
}

TEST(MasterSlaveManagement, AddSlaveSetsDiscoveredTrue) {
    Master master;
    master.add_slave(slave::Properties{.position = 0});
    auto slaves = master.slaves();
    ASSERT_EQ(slaves.size(), 1);
    EXPECT_TRUE(slaves[0].status.pdos_discovered);
}

TEST(MasterSlaveManagement, SlavesReturnsAddedSlaves) {
    Master master;
    master.add_slave(
        slave::Properties{
            .position = 0,
            .vendor_id = 0x100,
            .product_code = 0x200,
            .name = "Slave0"
        }
    );
    master.add_slave(
        slave::Properties{
            .position = 1,
            .vendor_id = 0x101,
            .product_code = 0x201,
            .name = "Slave1"
        }
    );

    auto slaves = master.slaves();
    ASSERT_EQ(slaves.size(), 2);
    EXPECT_EQ(slaves[0].properties.position, 0);
    EXPECT_EQ(slaves[0].properties.vendor_id, 0x100u);
    EXPECT_EQ(slaves[0].properties.name, "Slave0");
    EXPECT_EQ(slaves[1].properties.position, 1);
    EXPECT_EQ(slaves[1].properties.vendor_id, 0x101u);
    EXPECT_EQ(slaves[1].properties.name, "Slave1");
}

TEST(MasterSlaveManagement, SetSlaveStateUpdatesState) {
    Master master;
    master.add_slave(slave::Properties{.position = 0});
    master.set_slave_state(0, slave::State::PRE_OP);
    EXPECT_EQ(master.slave_state(0), slave::State::PRE_OP);

    auto slaves = master.slaves();
    EXPECT_EQ(slaves[0].status.state, slave::State::PRE_OP);
}

TEST(MasterSlaveManagement, SlaveStateReturnsUnknownForInvalidPosition) {
    Master master;
    EXPECT_EQ(master.slave_state(99), slave::State::UNKNOWN);
}

TEST(MasterInitialization, InitializeSetsInitializedTrue) {
    Master master;
    EXPECT_FALSE(master.is_initialized());
    ASSERT_NIL(master.initialize());
    EXPECT_TRUE(master.is_initialized());
}

TEST(MasterInitialization, InitializeLogsCall) {
    Master master;
    ASSERT_NIL(master.initialize());
    EXPECT_TRUE(master.was_called("initialize"));
}

TEST(MasterInitialization, InitializeIncrementsCallCount) {
    Master master;
    EXPECT_EQ(master.init_call_count(), 0);
    ASSERT_NIL(master.initialize());
    EXPECT_EQ(master.init_call_count(), 1);
    master.deactivate();
    ASSERT_NIL(master.initialize());
    EXPECT_EQ(master.init_call_count(), 2);
}

TEST(MasterInitialization, InitializeWithInjectedError) {
    Master master;
    master.inject_init_error(x::errors::Error(errors::MASTER_INIT_ERROR, "test error"));
    ASSERT_OCCURRED_AS(master.initialize(), errors::MASTER_INIT_ERROR);
}

TEST(MasterInitialization, InitializeWithInjectedErrorDoesNotSetInitialized) {
    Master master;
    master.inject_init_error(x::errors::Error(errors::MASTER_INIT_ERROR, "test error"));
    auto err = master.initialize();
    EXPECT_FALSE(master.is_initialized());
}

TEST(MasterActivation, ActivateRequiresInitialization) {
    Master master;
    ASSERT_OCCURRED_AS(master.activate(), errors::ACTIVATION_ERROR);
}

TEST(MasterActivation, ActivateSetsActivatedTrue) {
    Master master;
    ASSERT_NIL(master.initialize());
    EXPECT_FALSE(master.is_activated());
    ASSERT_NIL(master.activate());
    EXPECT_TRUE(master.is_activated());
}

TEST(MasterActivation, ActivateLogsCall) {
    Master master;
    ASSERT_NIL(master.initialize());
    ASSERT_NIL(master.activate());
    EXPECT_TRUE(master.was_called("activate"));
}

TEST(MasterActivation, ActivateTransitionsSlavesToOp) {
    Master master;
    master.add_slave(slave::Properties{.position = 0});
    master.add_slave(slave::Properties{.position = 1});
    ASSERT_NIL(master.initialize());
    ASSERT_NIL(master.activate());
    EXPECT_EQ(master.slave_state(0), slave::State::OP);
    EXPECT_EQ(master.slave_state(1), slave::State::OP);
}

TEST(MasterActivation, ActivateWithInjectedError) {
    Master master;
    ASSERT_NIL(master.initialize());
    master.inject_activate_error(
        x::errors::Error(errors::ACTIVATION_ERROR, "test error")
    );
    ASSERT_OCCURRED_AS(master.activate(), errors::ACTIVATION_ERROR);
}

TEST(MasterActivation, ActivateWithTransitionFailure) {
    Master master;
    master.add_slave(slave::Properties{.position = 0});
    master.add_slave(slave::Properties{.position = 1});
    master.set_slave_transition_failure(1, slave::State::OP);
    ASSERT_NIL(master.initialize());
    ASSERT_NIL(master.activate());
    EXPECT_EQ(master.slave_state(0), slave::State::OP);
    EXPECT_EQ(master.slave_state(1), slave::State::SAFE_OP);
}

TEST(MasterActivation, ActivateAllocatesIoMaps) {
    Master master;
    master.add_slave(
        slave::Properties{
            .position = 0,
            .input_pdos =
                {{.pdo_index = 0x1A00,
                  .index = 0x6000,
                  .sub_index = 1,
                  .bit_length = 16,
                  .is_input = true,
                  .data_type = x::telem::INT16_T}},
            .output_pdos = {
                {.pdo_index = 0x1600,
                 .index = 0x7000,
                 .sub_index = 1,
                 .bit_length = 32,
                 .is_input = false,
                 .data_type = x::telem::INT32_T}
            }
        }
    );
    ASSERT_NIL(master.initialize());
    ASSERT_NIL(master.activate());
    EXPECT_GE(master.input_data().size(), 2);
    EXPECT_GE(master.output_data().size(), 4);
}

TEST(MasterDeactivation, DeactivateResetsState) {
    Master master;
    ASSERT_NIL(master.initialize());
    ASSERT_NIL(master.activate());
    master.deactivate();
    EXPECT_FALSE(master.is_initialized());
    EXPECT_FALSE(master.is_activated());
}

TEST(MasterDeactivation, DeactivateLogsCall) {
    Master master;
    ASSERT_NIL(master.initialize());
    ASSERT_NIL(master.activate());
    master.deactivate();
    EXPECT_TRUE(master.was_called("deactivate"));
}

TEST(MasterDeactivation, DeactivateTransitionsSlavesToInit) {
    Master master;
    master.add_slave(slave::Properties{.position = 0});
    ASSERT_NIL(master.initialize());
    ASSERT_NIL(master.activate());
    EXPECT_EQ(master.slave_state(0), slave::State::OP);
    master.deactivate();
    EXPECT_EQ(master.slave_state(0), slave::State::INIT);
}

TEST(MasterDeactivation, DeactivateClearsIoMaps) {
    Master master;
    master.add_slave(slave::Properties{.position = 0});
    ASSERT_NIL(master.initialize());
    ASSERT_NIL(master.activate());
    master.deactivate();
    EXPECT_TRUE(master.input_data().empty());
    EXPECT_TRUE(master.output_data().empty());
}

TEST(MasterSendReceive, SendLogsCall) {
    Master master;
    ASSERT_NIL(master.initialize());
    ASSERT_NIL(master.activate());
    ASSERT_NIL(master.send());
    EXPECT_TRUE(master.was_called("send"));
}

TEST(MasterSendReceive, SendWithInjectedError) {
    Master master;
    ASSERT_NIL(master.initialize());
    ASSERT_NIL(master.activate());
    master.inject_send_error(x::errors::Error(errors::MASTER_INIT_ERROR, "send error"));
    ASSERT_OCCURRED_AS(master.send(), errors::MASTER_INIT_ERROR);
}

TEST(MasterSendReceive, ReceiveLogsCall) {
    Master master;
    ASSERT_NIL(master.initialize());
    ASSERT_NIL(master.activate());
    ASSERT_NIL(master.receive());
    EXPECT_TRUE(master.was_called("receive"));
}

TEST(MasterSendReceive, ReceiveWithInjectedError) {
    Master master;
    ASSERT_NIL(master.initialize());
    ASSERT_NIL(master.activate());
    master.inject_receive_error(
        x::errors::Error(errors::MASTER_INIT_ERROR, "receive error")
    );
    ASSERT_OCCURRED_AS(master.receive(), errors::MASTER_INIT_ERROR);
}

TEST(MasterPdoRegistration, RegisterPdosLogsCall) {
    Master master;
    ASSERT_NIL(master.register_pdos({}));
    EXPECT_TRUE(master.was_called("register_pdos"));
}

TEST(MasterPdoRegistration, PdoOffsetReturnsCorrectOffset) {
    Master master;
    std::vector<pdo::Entry> entries = {
        pdo::Entry(0, 0x6000, 1, 16, true, x::telem::INT16_T),
        pdo::Entry(0, 0x6000, 2, 32, true, x::telem::INT32_T),
    };
    ASSERT_NIL(master.register_pdos(entries));
    ASSERT_NIL(master.initialize());
    ASSERT_NIL(master.activate());

    auto offset1 = master.pdo_offset(entries[0]);
    auto offset2 = master.pdo_offset(entries[1]);
    EXPECT_EQ(offset1.byte, 0);
    EXPECT_EQ(offset2.byte, 2);
}

TEST(MasterPdoRegistration, PdoOffsetReturnsEmptyForUnregistered) {
    Master master;
    ASSERT_NIL(master.initialize());
    ASSERT_NIL(master.activate());
    pdo::Entry unknown(0, 0x9999, 1, 16, true, x::telem::INT16_T);
    auto offset = master.pdo_offset(unknown);
    EXPECT_EQ(offset.byte, 0);
    EXPECT_EQ(offset.bit, 0);
}

TEST(MasterIoMap, SetInputWritesToInputMap) {
    Master master;
    master.add_slave(slave::Properties{.position = 0});
    ASSERT_NIL(master.initialize());
    ASSERT_NIL(master.activate());
    master.set_input<int16_t>(0, 0x1234);
    auto data = master.input_data();
    ASSERT_GE(data.size(), 2);
    int16_t value;
    std::memcpy(&value, data.data(), sizeof(value));
    EXPECT_EQ(value, 0x1234);
}

TEST(MasterIoMap, GetOutputReadsFromOutputMap) {
    Master master;
    master.add_slave(slave::Properties{.position = 0});
    ASSERT_NIL(master.initialize());
    ASSERT_NIL(master.activate());
    auto data = master.output_data();
    ASSERT_GE(data.size(), 4);
    int32_t write_val = 0xDEADBEEF;
    std::memcpy(data.data(), &write_val, sizeof(write_val));
    EXPECT_EQ(master.get_output<int32_t>(0), 0xDEADBEEF);
}

TEST(MasterIoMap, InputDataReturnsEmptyBeforeActivation) {
    Master master;
    EXPECT_TRUE(master.input_data().empty());
}

TEST(MasterIoMap, OutputDataReturnsEmptyBeforeActivation) {
    Master master;
    EXPECT_TRUE(master.output_data().empty());
}

TEST(MasterAllSlavesOperational, ReturnsFalseWhenEmpty) {
    Master master;
    EXPECT_FALSE(master.all_slaves_operational());
}

TEST(MasterAllSlavesOperational, ReturnsFalseWhenNotAllOp) {
    Master master;
    master.add_slave(slave::Properties{.position = 0});
    master.add_slave(slave::Properties{.position = 1});
    master.set_slave_transition_failure(1, slave::State::OP);
    ASSERT_NIL(master.initialize());
    ASSERT_NIL(master.activate());
    EXPECT_FALSE(master.all_slaves_operational());
}

TEST(MasterAllSlavesOperational, ReturnsTrueWhenAllOp) {
    Master master;
    master.add_slave(slave::Properties{.position = 0});
    master.add_slave(slave::Properties{.position = 1});
    ASSERT_NIL(master.initialize());
    ASSERT_NIL(master.activate());
    EXPECT_TRUE(master.all_slaves_operational());
}

TEST(MasterCallLog, WasCalledReturnsTrueForCalledMethod) {
    Master master;
    ASSERT_NIL(master.initialize());
    EXPECT_TRUE(master.was_called("initialize"));
}

TEST(MasterCallLog, WasCalledReturnsFalseForUncalledMethod) {
    Master master;
    EXPECT_FALSE(master.was_called("initialize"));
}

TEST(MasterCallLog, ClearCallLogRemovesEntries) {
    Master master;
    ASSERT_NIL(master.initialize());
    EXPECT_TRUE(master.was_called("initialize"));
    master.clear_call_log();
    EXPECT_FALSE(master.was_called("initialize"));
}

TEST(MasterErrorInjection, ClearInjectedErrorsResetsAllErrors) {
    Master master;
    master.inject_init_error(x::errors::Error(errors::MASTER_INIT_ERROR, "init"));
    master.inject_activate_error(
        x::errors::Error(errors::ACTIVATION_ERROR, "activate")
    );
    master.inject_send_error(x::errors::Error(errors::MASTER_INIT_ERROR, "send"));
    master.inject_receive_error(x::errors::Error(errors::MASTER_INIT_ERROR, "receive"));
    master.clear_injected_errors();
    ASSERT_NIL(master.initialize());
    ASSERT_NIL(master.activate());
    ASSERT_NIL(master.send());
    ASSERT_NIL(master.receive());
}

TEST(MasterSlaveStates, HasSlaveInState) {
    Master master;
    master.add_slave(slave::Properties{.position = 0});
    master.add_slave(slave::Properties{.position = 1});
    EXPECT_TRUE(master.has_slave_in_state(slave::State::INIT));
    EXPECT_FALSE(master.has_slave_in_state(slave::State::OP));
}

TEST(MasterSlaveStates, SlavesInState) {
    Master master;
    master.add_slave(slave::Properties{.position = 0});
    master.add_slave(slave::Properties{.position = 1});
    EXPECT_EQ(master.slaves_in_state(slave::State::INIT), 2);
    EXPECT_EQ(master.slaves_in_state(slave::State::OP), 0);
}

TEST(MasterInitCallCount, ResetInitCallCount) {
    Master master;
    ASSERT_NIL(master.initialize());
    EXPECT_EQ(master.init_call_count(), 1);
    master.reset_init_call_count();
    EXPECT_EQ(master.init_call_count(), 0);
}

TEST(ManagerEnumerate, ReturnsConfiguredMasters) {
    Manager manager;
    auto master1 = std::make_shared<Master>("eth0");
    auto master2 = std::make_shared<Master>("eth1");
    manager.configure("eth0", master1);
    manager.configure("eth1", master2);

    auto infos = manager.enumerate();
    ASSERT_EQ(infos.size(), 2);
    EXPECT_EQ(infos[0].key, "eth0");
    EXPECT_EQ(infos[1].key, "eth1");
}

TEST(ManagerCreate, ReturnsConfiguredMaster) {
    Manager manager;
    auto master = std::make_shared<Master>("eth0");
    manager.configure("eth0", master);

    auto [result, err] = manager.create("eth0");
    ASSERT_NIL(err);
    EXPECT_EQ(result, master);
}

TEST(ManagerCreate, ReturnsErrorForUnconfigured) {
    Manager manager;
    ASSERT_OCCURRED_AS_P(manager.create("unknown"), errors::MASTER_INIT_ERROR);
}

}
