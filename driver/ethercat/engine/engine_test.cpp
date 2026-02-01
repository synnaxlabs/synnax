// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/telem/frame.h"
#include "x/cpp/xtest/xtest.h"

#include "driver/ethercat/engine/engine.h"
#include "driver/ethercat/mock/master.h"

class EngineTest : public ::testing::Test {
protected:
    std::shared_ptr<ethercat::mock::Master> mock_master;
    std::shared_ptr<ethercat::engine::Engine> engine;

    void SetUp() override {
        mock_master = std::make_shared<ethercat::mock::Master>("eth0");
        mock_master->add_slave(ethercat::mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));
        engine = std::make_shared<ethercat::engine::Engine>(
            mock_master,
            ethercat::engine::Config(telem::MILLISECOND * 10)
        );
    }
};

TEST_F(EngineTest, OpenReaderReturnsCorrectSize) {
    auto reader = ASSERT_NIL_P(
        engine->open_reader({ethercat::PDOEntry(0, 0x6000, 1, 16, true)})
    );
    EXPECT_EQ(reader->size(), 2);
}

TEST_F(EngineTest, OpenWriterSucceeds) {
    auto writer = ASSERT_NIL_P(
        engine->open_writer({ethercat::PDOEntry(0, 0x7000, 1, 16, false)})
    );
    writer->write(0, static_cast<uint16_t>(0x1234));
}

TEST_F(EngineTest, OpenReaderWithMultiplePDOs) {
    auto reader = ASSERT_NIL_P(engine->open_reader(
        {ethercat::PDOEntry(0, 0x6000, 1, 16, true),
         ethercat::PDOEntry(0, 0x6000, 2, 32, true)}
    ));
    EXPECT_EQ(reader->size(), 6);
}

TEST_F(EngineTest, ReadReturnsData) {
    auto reader = ASSERT_NIL_P(
        engine->open_reader({ethercat::PDOEntry(0, 0x6000, 1, 16, true)})
    );

    breaker::Breaker brk;
    brk.start();
    telem::Frame frame;
    frame.emplace(1, telem::Series(telem::UINT16_T, 1));
    ASSERT_NIL(reader->read(brk, frame));
    EXPECT_EQ(frame.series->at(0).size(), 1);
    brk.stop();
}

TEST_F(EngineTest, ReadReturnsNilWhenBreakerStopped) {
    auto reader = ASSERT_NIL_P(
        engine->open_reader({ethercat::PDOEntry(0, 0x6000, 1, 16, true)})
    );

    breaker::Breaker brk;
    // Breaker not started, so running() returns false - this simulates user stop
    telem::Frame frame;
    frame.emplace(1, telem::Series(telem::UINT16_T, 1));
    // User-commanded stop should return NIL, not an error
    ASSERT_NIL(reader->read(brk, frame));
}

TEST_F(EngineTest, WriteSucceeds) {
    auto writer = ASSERT_NIL_P(
        engine->open_writer({ethercat::PDOEntry(0, 0x7000, 1, 16, false)})
    );
    writer->write(0, static_cast<uint16_t>(0x1234));
}

TEST_F(EngineTest, InitializeErrorPropagates) {
    mock_master->inject_init_error(
        xerrors::Error(ethercat::MASTER_INIT_ERROR, "init failed")
    );

    ASSERT_OCCURRED_AS(engine->open_reader({}).second, ethercat::MASTER_INIT_ERROR);
}

TEST_F(EngineTest, ActivateErrorPropagates) {
    mock_master->inject_activate_error(
        xerrors::Error(ethercat::ACTIVATION_ERROR, "activate failed")
    );

    ASSERT_OCCURRED_AS(engine->open_reader({}).second, ethercat::ACTIVATION_ERROR);
}

TEST_F(EngineTest, MultipleReadersCanRead) {
    auto reader1 = ASSERT_NIL_P(
        engine->open_reader({ethercat::PDOEntry(0, 0x6000, 1, 16, true)})
    );

    auto reader2 = ASSERT_NIL_P(
        engine->open_reader({ethercat::PDOEntry(0, 0x6000, 2, 32, true)})
    );

    breaker::Breaker brk;
    brk.start();

    telem::Frame frame1;
    frame1.emplace(1, telem::Series(telem::UINT16_T, 1));
    ASSERT_NIL(reader1->read(brk, frame1));

    telem::Frame frame2;
    frame2.emplace(2, telem::Series(telem::UINT32_T, 1));
    ASSERT_NIL(reader2->read(brk, frame2));

    brk.stop();
}

TEST_F(EngineTest, MultipleSlavesPDORegistration) {
    auto multi_master = std::make_shared<ethercat::mock::Master>("eth0");
    multi_master->add_slave(ethercat::mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));
    multi_master->add_slave(ethercat::mock::MockSlaveConfig(1, 0x1, 0x3, "Slave2"));

    auto multi_engine = std::make_shared<ethercat::engine::Engine>(
        multi_master,
        ethercat::engine::Config(telem::MILLISECOND * 10)
    );

    auto reader = ASSERT_NIL_P(multi_engine->open_reader(
        {ethercat::PDOEntry(0, 0x6000, 1, 16, true),
         ethercat::PDOEntry(1, 0x6000, 1, 32, true)}
    ));

    EXPECT_EQ(reader->size(), 6);
}

TEST_F(EngineTest, MixedReadersAndWriters) {
    auto reader = ASSERT_NIL_P(
        engine->open_reader({ethercat::PDOEntry(0, 0x6000, 1, 16, true)})
    );

    auto writer = ASSERT_NIL_P(
        engine->open_writer({ethercat::PDOEntry(0, 0x7000, 1, 16, false)})
    );

    breaker::Breaker brk;
    brk.start();
    telem::Frame frame;
    frame.emplace(1, telem::Series(telem::UINT16_T, 1));
    ASSERT_NIL(reader->read(brk, frame));

    writer->write(0, static_cast<uint16_t>(0x5678));
    brk.stop();
}

TEST_F(EngineTest, ReadAfterReconfigure) {
    auto reader1 = ASSERT_NIL_P(
        engine->open_reader({ethercat::PDOEntry(0, 0x6000, 1, 16, true)})
    );

    breaker::Breaker brk;
    brk.start();

    telem::Frame frame1;
    frame1.emplace(1, telem::Series(telem::UINT16_T, 1));
    ASSERT_NIL(reader1->read(brk, frame1));

    auto reader2 = ASSERT_NIL_P(
        engine->open_reader({ethercat::PDOEntry(0, 0x6000, 2, 32, true)})
    );

    telem::Frame frame2;
    frame2.emplace(2, telem::Series(telem::UINT32_T, 1));

    ASSERT_NIL(reader1->read(brk, frame1));
    ASSERT_NIL(reader2->read(brk, frame2));
    brk.stop();
}

TEST_F(EngineTest, WriteTypeConversionFloatToInt16) {
    auto writer = ASSERT_NIL_P(engine->open_writer(
        {ethercat::PDOEntry(0, 0x7000, 1, 16, false, telem::INT16_T)}
    ));
    writer->write(0, 42.7f);
    auto outputs = mock_master->output_data();
    int16_t written_value = static_cast<int16_t>(outputs[0]) |
                            (static_cast<int16_t>(outputs[1]) << 8);
    EXPECT_EQ(written_value, 42);
}

TEST_F(EngineTest, WriteTypeConversionInt64ToInt32) {
    auto writer = ASSERT_NIL_P(engine->open_writer(
        {ethercat::PDOEntry(0, 0x7000, 1, 32, false, telem::INT32_T)}
    ));
    writer->write(0, static_cast<int64_t>(0x12345678));
    auto outputs = mock_master->output_data();
    int32_t written_value = static_cast<int32_t>(outputs[0]) |
                            (static_cast<int32_t>(outputs[1]) << 8) |
                            (static_cast<int32_t>(outputs[2]) << 16) |
                            (static_cast<int32_t>(outputs[3]) << 24);
    EXPECT_EQ(written_value, 0x12345678);
}

TEST_F(EngineTest, WriteSubByteSingleByte) {
    auto writer = ASSERT_NIL_P(engine->open_writer(
        {ethercat::PDOEntry(0, 0x7000, 1, 4, false, telem::UINT8_T)}
    ));
    writer->write(0, static_cast<uint8_t>(0x0F));
    auto outputs = mock_master->output_data();
    EXPECT_EQ(outputs[0] & 0x0F, 0x0F);
}

TEST_F(EngineTest, Write24BitNoOffset) {
    auto writer = ASSERT_NIL_P(engine->open_writer(
        {ethercat::PDOEntry(0, 0x7000, 1, 24, false, telem::INT32_T)}
    ));
    writer->write(0, static_cast<int32_t>(0x123456));
    auto outputs = mock_master->output_data();
    EXPECT_EQ(outputs[0], 0x56);
    EXPECT_EQ(outputs[1], 0x34);
    EXPECT_EQ(outputs[2], 0x12);
}

TEST_F(EngineTest, Write24BitSignedNegative) {
    auto writer = ASSERT_NIL_P(engine->open_writer(
        {ethercat::PDOEntry(0, 0x7000, 1, 24, false, telem::INT32_T)}
    ));
    writer->write(0, static_cast<int32_t>(-1));
    auto outputs = mock_master->output_data();
    EXPECT_EQ(outputs[0], 0xFF);
    EXPECT_EQ(outputs[1], 0xFF);
    EXPECT_EQ(outputs[2], 0xFF);
}
