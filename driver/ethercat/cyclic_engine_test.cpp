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

#include "driver/ethercat/cyclic_engine.h"
#include "driver/ethercat/mock/master.h"

class CyclicEngineTest : public ::testing::Test {
protected:
    std::shared_ptr<ethercat::mock::MockMaster> mock_master;
    std::unique_ptr<ethercat::CyclicEngine> engine;

    void SetUp() override {
        mock_master = std::make_shared<ethercat::mock::MockMaster>("eth0");
        mock_master->add_slave(ethercat::mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));
        engine = std::make_unique<ethercat::CyclicEngine>(
            mock_master,
            ethercat::CyclicEngineConfig(telem::MILLISECOND * 10)
        );
    }
};

TEST_F(CyclicEngineTest, RegisterInputPDO) {
    ethercat::PDOEntry entry(0, 0x6000, 1, 16, true);
    auto [reg_index, err] = engine->register_input_pdo(entry);
    ASSERT_NIL(err);
    EXPECT_EQ(reg_index, 0);  // First registration = index 0
}

TEST_F(CyclicEngineTest, RegisterOutputPDO) {
    ethercat::PDOEntry entry(0, 0x7000, 1, 16, false);
    auto [reg_index, err] = engine->register_output_pdo(entry);
    ASSERT_NIL(err);
    EXPECT_EQ(reg_index, 0);  // First registration = index 0
}

TEST_F(CyclicEngineTest, RegisterMultiplePDOs) {
    ethercat::PDOEntry entry1(0, 0x6000, 1, 16, true);
    ethercat::PDOEntry entry2(0, 0x6000, 2, 32, true);

    auto [reg1, err1] = engine->register_input_pdo(entry1);
    ASSERT_NIL(err1);
    EXPECT_EQ(reg1, 0);  // First registration = index 0

    auto [reg2, err2] = engine->register_input_pdo(entry2);
    ASSERT_NIL(err2);
    EXPECT_EQ(reg2, 1);  // Second registration = index 1

    // Verify actual offsets after activation
    ASSERT_NIL(engine->add_task());
    // Offsets are relative to slave's input region (0 and 2 bytes respectively)
    EXPECT_EQ(engine->get_actual_input_offset(0), 0);
    EXPECT_EQ(engine->get_actual_input_offset(1), 2);
    engine->remove_task();
}

TEST_F(CyclicEngineTest, ActivatesOnFirstTask) {
    ASSERT_NIL(engine->add_task());
    EXPECT_TRUE(mock_master->was_called("initialize"));
    EXPECT_TRUE(mock_master->was_called("activate"));
    EXPECT_TRUE(engine->running());
    engine->remove_task();
}

TEST_F(CyclicEngineTest, DeactivatesOnLastTask) {
    ASSERT_NIL(engine->add_task());
    ASSERT_NIL(engine->add_task());
    EXPECT_EQ(engine->task_count(), 2);

    engine->remove_task();
    EXPECT_TRUE(engine->running());
    EXPECT_EQ(engine->task_count(), 1);

    engine->remove_task();
    EXPECT_FALSE(engine->running());
    EXPECT_TRUE(mock_master->was_called("deactivate"));
}

TEST_F(CyclicEngineTest, WriteOutputStagesData) {
    ethercat::PDOEntry entry(0, 0x7000, 1, 16, false);
    auto [reg_index, err] = engine->register_output_pdo(entry);
    ASSERT_NIL(err);

    ASSERT_NIL(engine->add_task());

    // Get actual offset after activation
    const size_t actual_offset = engine->get_actual_output_offset(reg_index);

    uint16_t value = 0x1234;
    engine->write_output(actual_offset, &value, sizeof(value));

    engine->remove_task();
}

TEST_F(CyclicEngineTest, InitializeErrorPropagates) {
    mock_master->inject_init_error(
        xerrors::Error(ethercat::MASTER_INIT_ERROR, "init failed")
    );
    auto err = engine->add_task();
    ASSERT_OCCURRED_AS(err, ethercat::MASTER_INIT_ERROR);
    EXPECT_FALSE(engine->running());
}

TEST_F(CyclicEngineTest, ActivateErrorPropagates) {
    mock_master->inject_activate_error(
        xerrors::Error(ethercat::ACTIVATION_ERROR, "activate failed")
    );
    auto err = engine->add_task();
    ASSERT_OCCURRED_AS(err, ethercat::ACTIVATION_ERROR);
    EXPECT_FALSE(engine->running());
}

TEST_F(CyclicEngineTest, CannotRegisterPDOWhileRunning) {
    ASSERT_NIL(engine->add_task());
    ethercat::PDOEntry entry(0, 0x6000, 1, 16, true);
    auto [offset, err] = engine->register_input_pdo(entry);
    ASSERT_OCCURRED_AS(err, ethercat::PDO_MAPPING_ERROR);
    engine->remove_task();
}
