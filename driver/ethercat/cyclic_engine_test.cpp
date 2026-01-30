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
    std::shared_ptr<ethercat::mock::Master> mock_master;
    std::unique_ptr<ethercat::CyclicEngine> engine;

    void SetUp() override {
        mock_master = std::make_shared<ethercat::mock::Master>("eth0");
        mock_master->add_slave(ethercat::mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));
        engine = std::make_unique<ethercat::CyclicEngine>(
            mock_master,
            ethercat::CyclicEngineConfig(telem::MILLISECOND * 10)
        );
    }
};

TEST_F(CyclicEngineTest, RegisterInputPDO) {
    ethercat::PDOEntry entry(0, 0x6000, 1, 16, true);
    auto handle = ASSERT_NIL_P(engine->register_input_pdo(entry));
    EXPECT_TRUE(handle.valid());
    EXPECT_TRUE(handle.is_input);
    EXPECT_EQ(handle.index, 0);
}

TEST_F(CyclicEngineTest, RegisterOutputPDO) {
    ethercat::PDOEntry entry(0, 0x7000, 1, 16, false);
    auto handle = ASSERT_NIL_P(engine->register_output_pdo(entry));
    EXPECT_TRUE(handle.valid());
    EXPECT_FALSE(handle.is_input);
    EXPECT_EQ(handle.index, 0);
}

TEST_F(CyclicEngineTest, RegisterMultiplePDOs) {
    ethercat::PDOEntry entry1(0, 0x6000, 1, 16, true);
    ethercat::PDOEntry entry2(0, 0x6000, 2, 32, true);

    auto handle1 = ASSERT_NIL_P(engine->register_input_pdo(entry1));
    EXPECT_EQ(handle1.index, 0);

    auto handle2 = ASSERT_NIL_P(engine->register_input_pdo(entry2));
    EXPECT_EQ(handle2.index, 1);

    ASSERT_NIL(engine->add_task());
    EXPECT_EQ(engine->get_actual_input_offset(handle1.index), 0);
    EXPECT_EQ(engine->get_actual_input_offset(handle2.index), 2);
    engine->remove_task();
}

TEST_F(CyclicEngineTest, ActivatesOnFirstTask) {
    ASSERT_NIL(engine->add_task());
    EXPECT_TRUE(mock_master->was_called("initialize"));
    EXPECT_TRUE(mock_master->was_called("activate"));
    EXPECT_TRUE(engine->is_running());
    engine->remove_task();
}

TEST_F(CyclicEngineTest, DeactivatesOnLastTask) {
    ASSERT_NIL(engine->add_task());
    ASSERT_NIL(engine->add_task());
    EXPECT_EQ(engine->get_task_count(), 2);

    engine->remove_task();
    EXPECT_TRUE(engine->is_running());
    EXPECT_EQ(engine->get_task_count(), 1);

    engine->remove_task();
    EXPECT_FALSE(engine->is_running());
    EXPECT_TRUE(mock_master->was_called("deactivate"));
}

TEST_F(CyclicEngineTest, WriteOutputStagesData) {
    ethercat::PDOEntry entry(0, 0x7000, 1, 16, false);
    auto handle = ASSERT_NIL_P(engine->register_output_pdo(entry));

    ASSERT_NIL(engine->add_task());

    const size_t actual_offset = engine->get_actual_output_offset(handle.index);
    uint16_t value = 0x1234;
    engine->write_output(actual_offset, &value, sizeof(value));

    engine->remove_task();
}

TEST_F(CyclicEngineTest, InitializeErrorPropagates) {
    mock_master->inject_init_error(
        xerrors::Error(ethercat::MASTER_INIT_ERROR, "init failed")
    );
    ASSERT_OCCURRED_AS(engine->add_task(), ethercat::MASTER_INIT_ERROR);
    EXPECT_FALSE(engine->is_running());
}

TEST_F(CyclicEngineTest, ActivateErrorPropagates) {
    mock_master->inject_activate_error(
        xerrors::Error(ethercat::ACTIVATION_ERROR, "activate failed")
    );
    ASSERT_OCCURRED_AS(engine->add_task(), ethercat::ACTIVATION_ERROR);
    EXPECT_FALSE(engine->is_running());
}

TEST_F(CyclicEngineTest, RegisterPDOWhileRunningTriggersRestart) {
    ethercat::PDOEntry entry1(0, 0x6000, 1, 16, true);
    ASSERT_NIL_P(engine->register_input_pdo(entry1));
    ASSERT_NIL(engine->add_task());
    EXPECT_TRUE(engine->is_running());

    mock_master->clear_call_log();

    ethercat::PDOEntry entry2(0, 0x6000, 2, 32, true);
    auto handle = ASSERT_NIL_P(engine->register_input_pdo(entry2));
    EXPECT_TRUE(handle.valid());

    EXPECT_TRUE(mock_master->was_called("deactivate"));
    EXPECT_TRUE(mock_master->was_called("initialize"));
    EXPECT_TRUE(mock_master->was_called("activate"));
    EXPECT_TRUE(engine->is_running());

    engine->remove_task();
}

TEST_F(CyclicEngineTest, WaitForInputsBreaker) {
    ethercat::PDOEntry entry(0, 0x6000, 1, 16, true);
    ASSERT_NIL_P(engine->register_input_pdo(entry));
    ASSERT_NIL(engine->add_task());

    std::atomic<bool> breaker{true};
    std::vector<uint8_t> buffer;

    ASSERT_OCCURRED_AS(
        engine->wait_for_inputs(buffer, breaker),
        ethercat::CYCLIC_ERROR
    );

    engine->remove_task();
}

TEST_F(CyclicEngineTest, WaitForInputsTimeout) {
    auto slow_master = std::make_shared<ethercat::mock::Master>("eth0");
    slow_master->add_slave(ethercat::mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));

    auto slow_engine = std::make_unique<ethercat::CyclicEngine>(
        slow_master,
        ethercat::CyclicEngineConfig(telem::SECOND * 10)
    );

    ethercat::PDOEntry entry(0, 0x6000, 1, 16, true);
    ASSERT_NIL_P(slow_engine->register_input_pdo(entry));
    ASSERT_NIL(slow_engine->add_task());

    std::atomic<bool> breaker{false};
    std::vector<uint8_t> buffer;

    slow_engine->remove_task();

    ASSERT_OCCURRED_AS(
        slow_engine->wait_for_inputs(buffer, breaker),
        ethercat::CYCLIC_ERROR
    );
}

TEST_F(CyclicEngineTest, WriteOutputBoundsCheck) {
    ethercat::PDOEntry entry(0, 0x7000, 1, 16, false);
    ASSERT_NIL_P(engine->register_output_pdo(entry));
    ASSERT_NIL(engine->add_task());

    uint64_t large_value = 0xDEADBEEFCAFEBABE;
    engine->write_output(1000, &large_value, sizeof(large_value));

    engine->remove_task();
}

TEST_F(CyclicEngineTest, CycleCountIncrement) {
    ASSERT_NIL(engine->add_task());

    uint64_t initial_count = engine->cycle_count();
    std::this_thread::sleep_for(std::chrono::milliseconds(50));
    uint64_t later_count = engine->cycle_count();

    EXPECT_GT(later_count, initial_count);

    engine->remove_task();
}

TEST_F(CyclicEngineTest, LastErrorCapture) {
    mock_master->inject_receive_error(
        xerrors::Error(ethercat::CYCLIC_ERROR, "receive failed")
    );

    ASSERT_NIL(engine->add_task());
    std::this_thread::sleep_for(std::chrono::milliseconds(50));

    ASSERT_OCCURRED_AS(engine->last_error(), ethercat::CYCLIC_ERROR);

    engine->remove_task();
}

TEST_F(CyclicEngineTest, CycleTimeAccessor) {
    EXPECT_EQ(engine->cycle_time(), telem::MILLISECOND * 10);
}

TEST_F(CyclicEngineTest, SlavesAccessor) {
    auto slaves = engine->slaves();
    ASSERT_EQ(slaves.size(), 1);
    EXPECT_EQ(slaves[0].name, "Slave1");
}

TEST_F(CyclicEngineTest, GetActualInputOffsetOutOfBounds) {
    ASSERT_NIL(engine->add_task());
    EXPECT_EQ(engine->get_actual_input_offset(999), 0);
    engine->remove_task();
}

TEST_F(CyclicEngineTest, GetActualOutputOffsetOutOfBounds) {
    ASSERT_NIL(engine->add_task());
    EXPECT_EQ(engine->get_actual_output_offset(999), 0);
    engine->remove_task();
}

TEST_F(CyclicEngineTest, MultipleSlavesPDORegistration) {
    auto multi_master = std::make_shared<ethercat::mock::Master>("eth0");
    multi_master->add_slave(ethercat::mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));
    multi_master->add_slave(ethercat::mock::MockSlaveConfig(1, 0x1, 0x3, "Slave2"));

    auto multi_engine = std::make_unique<ethercat::CyclicEngine>(
        multi_master,
        ethercat::CyclicEngineConfig(telem::MILLISECOND * 10)
    );

    ethercat::PDOEntry entry1(0, 0x6000, 1, 16, true);
    ethercat::PDOEntry entry2(1, 0x6000, 1, 32, true);

    auto handle1 = ASSERT_NIL_P(multi_engine->register_input_pdo(entry1));
    auto handle2 = ASSERT_NIL_P(multi_engine->register_input_pdo(entry2));

    ASSERT_NIL(multi_engine->add_task());

    EXPECT_EQ(multi_engine->get_actual_input_offset(handle1.index), 0);
    EXPECT_EQ(multi_engine->get_actual_input_offset(handle2.index), 4);

    multi_engine->remove_task();
}

TEST_F(CyclicEngineTest, ProcessErrorCapture) {
    mock_master->inject_process_error(
        xerrors::Error(ethercat::WORKING_COUNTER_ERROR, "wkc mismatch")
    );

    ASSERT_NIL(engine->add_task());
    std::this_thread::sleep_for(std::chrono::milliseconds(50));

    ASSERT_OCCURRED_AS(engine->last_error(), ethercat::WORKING_COUNTER_ERROR);

    engine->remove_task();
}

TEST_F(CyclicEngineTest, QueueErrorCapture) {
    mock_master->inject_queue_error(
        xerrors::Error(ethercat::CYCLIC_ERROR, "queue failed")
    );

    ASSERT_NIL(engine->add_task());
    std::this_thread::sleep_for(std::chrono::milliseconds(50));

    ASSERT_OCCURRED_AS(engine->last_error(), ethercat::CYCLIC_ERROR);

    engine->remove_task();
}

TEST_F(CyclicEngineTest, SendErrorCapture) {
    mock_master->inject_send_error(
        xerrors::Error(ethercat::CYCLIC_ERROR, "send failed")
    );

    ASSERT_NIL(engine->add_task());
    std::this_thread::sleep_for(std::chrono::milliseconds(50));

    ASSERT_OCCURRED_AS(engine->last_error(), ethercat::CYCLIC_ERROR);

    engine->remove_task();
}

TEST_F(CyclicEngineTest, WaitForInputsSuccess) {
    ethercat::PDOEntry entry(0, 0x6000, 1, 16, true);
    ASSERT_NIL_P(engine->register_input_pdo(entry));
    ASSERT_NIL(engine->add_task());

    std::atomic<bool> breaker{false};
    std::vector<uint8_t> buffer;

    ASSERT_NIL(engine->wait_for_inputs(buffer, breaker));

    engine->remove_task();
}

TEST_F(CyclicEngineTest, HandleBasedReadInput) {
    ethercat::PDOEntry entry(0, 0x6000, 1, 16, true);
    auto handle = ASSERT_NIL_P(engine->register_input_pdo(entry));
    EXPECT_TRUE(handle.valid());
    EXPECT_TRUE(handle.is_input);

    ASSERT_NIL(engine->add_task());

    std::atomic<bool> breaker{false};
    std::vector<uint8_t> buffer;
    ASSERT_NIL(engine->wait_for_inputs(buffer, breaker));

    uint16_t value = 0;
    ASSERT_NIL(engine->read_input(handle, &value, sizeof(value)));

    engine->remove_task();
}

TEST_F(CyclicEngineTest, HandleBasedWriteOutput) {
    ethercat::PDOEntry entry(0, 0x7000, 1, 16, false);
    auto handle = ASSERT_NIL_P(engine->register_output_pdo(entry));
    EXPECT_TRUE(handle.valid());
    EXPECT_FALSE(handle.is_input);

    ASSERT_NIL(engine->add_task());

    uint16_t value = 0x1234;
    engine->write_output(handle, &value, sizeof(value));

    engine->remove_task();
}

TEST_F(CyclicEngineTest, InvalidHandleReadInput) {
    ASSERT_NIL(engine->add_task());

    ethercat::PDOHandle invalid_handle = ethercat::PDOHandle::invalid();
    uint16_t value = 0;
    ASSERT_OCCURRED_AS(
        engine->read_input(invalid_handle, &value, sizeof(value)),
        ethercat::PDO_MAPPING_ERROR
    );

    engine->remove_task();
}

TEST_F(CyclicEngineTest, RestartPreservesOutputBuffer) {
    ethercat::PDOEntry output_entry(0, 0x7000, 1, 16, false);
    auto output_handle = ASSERT_NIL_P(engine->register_output_pdo(output_entry));

    ASSERT_NIL(engine->add_task());

    uint16_t value = 0xABCD;
    const size_t offset = engine->get_actual_output_offset(output_handle.index);
    engine->write_output(offset, &value, sizeof(value));

    mock_master->clear_call_log();

    ethercat::PDOEntry input_entry(0, 0x6000, 1, 32, true);
    ASSERT_NIL_P(engine->register_input_pdo(input_entry));

    EXPECT_TRUE(mock_master->was_called("deactivate"));
    EXPECT_TRUE(mock_master->was_called("activate"));
    EXPECT_TRUE(engine->is_running());

    engine->remove_task();
}

TEST_F(CyclicEngineTest, SecondTaskStartsWhileFirstRunning) {
    ethercat::PDOEntry entry1(0, 0x6000, 1, 16, true);
    ASSERT_NIL_P(engine->register_input_pdo(entry1));
    ASSERT_NIL(engine->add_task());
    EXPECT_TRUE(engine->is_running());
    EXPECT_EQ(engine->get_task_count(), 1);

    ASSERT_NIL(engine->add_task());
    EXPECT_EQ(engine->get_task_count(), 2);

    engine->remove_task();
    EXPECT_EQ(engine->get_task_count(), 1);
    EXPECT_TRUE(engine->is_running());

    engine->remove_task();
    EXPECT_EQ(engine->get_task_count(), 0);
    EXPECT_FALSE(engine->is_running());
}

TEST_F(CyclicEngineTest, InitCallCountIncrementsOnRestart) {
    ethercat::PDOEntry entry1(0, 0x6000, 1, 16, true);
    ASSERT_NIL_P(engine->register_input_pdo(entry1));
    ASSERT_NIL(engine->add_task());

    size_t initial_count = mock_master->init_call_count();
    EXPECT_EQ(initial_count, 1);

    ethercat::PDOEntry entry2(0, 0x6000, 2, 32, true);
    ASSERT_NIL_P(engine->register_input_pdo(entry2));

    EXPECT_EQ(mock_master->init_call_count(), 2);

    engine->remove_task();
}

TEST_F(CyclicEngineTest, RegisterOutputPDOWhileRunning) {
    ethercat::PDOEntry input_entry(0, 0x6000, 1, 16, true);
    ASSERT_NIL_P(engine->register_input_pdo(input_entry));
    ASSERT_NIL(engine->add_task());

    mock_master->clear_call_log();

    ethercat::PDOEntry output_entry(0, 0x7000, 1, 16, false);
    auto handle = ASSERT_NIL_P(engine->register_output_pdo(output_entry));
    EXPECT_TRUE(handle.valid());
    EXPECT_FALSE(handle.is_input);

    EXPECT_TRUE(mock_master->was_called("deactivate"));
    EXPECT_TRUE(mock_master->was_called("initialize"));
    EXPECT_TRUE(mock_master->was_called("activate"));

    engine->remove_task();
}

TEST_F(CyclicEngineTest, TwoTasksWithDifferentPDOsBothWorkAfterRestart) {
    ethercat::PDOEntry task1_entry(0, 0x6000, 1, 16, true);
    auto task1_handle = ASSERT_NIL_P(engine->register_input_pdo(task1_entry));
    ASSERT_NIL(engine->add_task());

    std::atomic<bool> breaker{false};
    std::vector<uint8_t> buffer;
    ASSERT_NIL(engine->wait_for_inputs(buffer, breaker));

    uint16_t task1_value = 0;
    ASSERT_NIL(engine->read_input(task1_handle, &task1_value, sizeof(task1_value)));

    ethercat::PDOEntry task2_entry(0, 0x6000, 2, 32, true);
    auto task2_handle = ASSERT_NIL_P(engine->register_input_pdo(task2_entry));
    ASSERT_NIL(engine->add_task());

    ASSERT_NIL(engine->wait_for_inputs(buffer, breaker));

    ASSERT_NIL(engine->read_input(task1_handle, &task1_value, sizeof(task1_value)));

    uint32_t task2_value = 0;
    ASSERT_NIL(engine->read_input(task2_handle, &task2_value, sizeof(task2_value)));

    engine->remove_task();
    EXPECT_TRUE(engine->is_running());
    EXPECT_EQ(engine->get_task_count(), 1);

    engine->remove_task();
    EXPECT_FALSE(engine->is_running());
}

TEST_F(CyclicEngineTest, HandleIndexStableAfterRestart) {
    ethercat::PDOEntry entry1(0, 0x6000, 1, 16, true);
    auto handle1 = ASSERT_NIL_P(engine->register_input_pdo(entry1));
    EXPECT_EQ(handle1.index, 0);

    ASSERT_NIL(engine->add_task());
    size_t offset_before = engine->get_actual_input_offset(handle1.index);

    ethercat::PDOEntry entry2(0, 0x6000, 2, 32, true);
    auto handle2 = ASSERT_NIL_P(engine->register_input_pdo(entry2));
    EXPECT_EQ(handle2.index, 1);

    EXPECT_EQ(handle1.index, 0);

    size_t offset_after = engine->get_actual_input_offset(handle1.index);
    EXPECT_EQ(offset_before, offset_after);

    engine->remove_task();
}

TEST_F(CyclicEngineTest, FirstTaskContinuesReadingAfterSecondTaskTriggersRestart) {
    ethercat::PDOEntry entry1(0, 0x6000, 1, 16, true);
    auto handle1 = ASSERT_NIL_P(engine->register_input_pdo(entry1));
    ASSERT_NIL(engine->add_task());

    std::atomic<bool> breaker{false};
    std::vector<uint8_t> buffer;

    for (int i = 0; i < 3; i++) {
        ASSERT_NIL(engine->wait_for_inputs(buffer, breaker));
        uint16_t value = 0;
        ASSERT_NIL(engine->read_input(handle1, &value, sizeof(value)));
    }

    ethercat::PDOEntry entry2(0, 0x6000, 2, 32, true);
    auto handle2 = ASSERT_NIL_P(engine->register_input_pdo(entry2));
    ASSERT_NIL(engine->add_task());

    for (int i = 0; i < 3; i++) {
        ASSERT_NIL(engine->wait_for_inputs(buffer, breaker));
        uint16_t value1 = 0;
        uint32_t value2 = 0;
        ASSERT_NIL(engine->read_input(handle1, &value1, sizeof(value1)));
        ASSERT_NIL(engine->read_input(handle2, &value2, sizeof(value2)));
    }

    engine->remove_task();
    engine->remove_task();
}

TEST_F(CyclicEngineTest, TaskStopsWhileAnotherContinues) {
    ethercat::PDOEntry entry(0, 0x6000, 1, 16, true);
    auto handle = ASSERT_NIL_P(engine->register_input_pdo(entry));

    ASSERT_NIL(engine->add_task());
    ASSERT_NIL(engine->add_task());
    EXPECT_EQ(engine->get_task_count(), 2);

    engine->remove_task();
    EXPECT_EQ(engine->get_task_count(), 1);
    EXPECT_TRUE(engine->is_running());

    std::atomic<bool> breaker{false};
    std::vector<uint8_t> buffer;
    ASSERT_NIL(engine->wait_for_inputs(buffer, breaker));

    uint16_t value = 0;
    ASSERT_NIL(engine->read_input(handle, &value, sizeof(value)));

    engine->remove_task();
    EXPECT_FALSE(engine->is_running());
}

TEST_F(CyclicEngineTest, WriteTaskStartsWhileReadTaskRunning) {
    ethercat::PDOEntry input_entry(0, 0x6000, 1, 16, true);
    auto input_handle = ASSERT_NIL_P(engine->register_input_pdo(input_entry));
    ASSERT_NIL(engine->add_task());

    std::atomic<bool> breaker{false};
    std::vector<uint8_t> buffer;
    ASSERT_NIL(engine->wait_for_inputs(buffer, breaker));

    ethercat::PDOEntry output_entry(0, 0x7000, 1, 16, false);
    auto output_handle = ASSERT_NIL_P(engine->register_output_pdo(output_entry));
    ASSERT_NIL(engine->add_task());

    ASSERT_NIL(engine->wait_for_inputs(buffer, breaker));

    uint16_t read_value = 0;
    ASSERT_NIL(engine->read_input(input_handle, &read_value, sizeof(read_value)));

    uint16_t write_value = 0x5678;
    engine->write_output(output_handle, &write_value, sizeof(write_value));

    engine->remove_task();
    engine->remove_task();
}

TEST_F(CyclicEngineTest, MultipleRestartsInSequence) {
    ethercat::PDOEntry entry1(0, 0x6000, 1, 8, true);
    auto handle1 = ASSERT_NIL_P(engine->register_input_pdo(entry1));
    ASSERT_NIL(engine->add_task());
    EXPECT_EQ(mock_master->init_call_count(), 1);

    ethercat::PDOEntry entry2(0, 0x6000, 2, 8, true);
    auto handle2 = ASSERT_NIL_P(engine->register_input_pdo(entry2));
    EXPECT_EQ(mock_master->init_call_count(), 2);

    ethercat::PDOEntry entry3(0, 0x6000, 3, 8, true);
    auto handle3 = ASSERT_NIL_P(engine->register_input_pdo(entry3));
    EXPECT_EQ(mock_master->init_call_count(), 3);

    EXPECT_TRUE(engine->is_running());
    EXPECT_EQ(handle1.index, 0);
    EXPECT_EQ(handle2.index, 1);
    EXPECT_EQ(handle3.index, 2);

    std::atomic<bool> breaker{false};
    std::vector<uint8_t> buffer;
    ASSERT_NIL(engine->wait_for_inputs(buffer, breaker));

    uint8_t v1 = 0, v2 = 0, v3 = 0;
    ASSERT_NIL(engine->read_input(handle1, &v1, sizeof(v1)));
    ASSERT_NIL(engine->read_input(handle2, &v2, sizeof(v2)));
    ASSERT_NIL(engine->read_input(handle3, &v3, sizeof(v3)));

    engine->remove_task();
}

TEST_F(CyclicEngineTest, ReadAndWriteTasksWithSeparatePDOs) {
    auto multi_master = std::make_shared<ethercat::mock::Master>("eth0");
    multi_master->add_slave(ethercat::mock::MockSlaveConfig(0, 0x1, 0x2, "InputSlave"));
    multi_master->add_slave(
        ethercat::mock::MockSlaveConfig(1, 0x1, 0x3, "OutputSlave")
    );

    auto multi_engine = std::make_unique<ethercat::CyclicEngine>(
        multi_master,
        ethercat::CyclicEngineConfig(telem::MILLISECOND * 10)
    );

    ethercat::PDOEntry input_entry(0, 0x6000, 1, 16, true);
    auto input_handle = ASSERT_NIL_P(multi_engine->register_input_pdo(input_entry));
    ASSERT_NIL(multi_engine->add_task());

    ethercat::PDOEntry output_entry(1, 0x7000, 1, 16, false);
    auto output_handle = ASSERT_NIL_P(multi_engine->register_output_pdo(output_entry));
    ASSERT_NIL(multi_engine->add_task());

    EXPECT_TRUE(multi_engine->is_running());
    EXPECT_EQ(multi_engine->get_task_count(), 2);

    std::atomic<bool> breaker{false};
    std::vector<uint8_t> buffer;
    ASSERT_NIL(multi_engine->wait_for_inputs(buffer, breaker));

    uint16_t read_value = 0;
    ASSERT_NIL(multi_engine->read_input(input_handle, &read_value, sizeof(read_value)));

    uint16_t write_value = 0x9ABC;
    multi_engine->write_output(output_handle, &write_value, sizeof(write_value));

    multi_engine->remove_task();
    EXPECT_TRUE(multi_engine->is_running());

    multi_engine->remove_task();
    EXPECT_FALSE(multi_engine->is_running());
}
