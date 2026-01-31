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

#include "driver/ethercat/loop/loop.h"
#include "driver/ethercat/mock/master.h"

class LoopTest : public ::testing::Test {
protected:
    std::shared_ptr<ethercat::mock::Master> mock_master;
    std::shared_ptr<ethercat::Loop> loop;

    void SetUp() override {
        mock_master = std::make_shared<ethercat::mock::Master>("eth0");
        mock_master->add_slave(ethercat::mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));
        loop = std::make_shared<ethercat::Loop>(
            mock_master,
            ethercat::LoopConfig(telem::MILLISECOND * 10)
        );
    }
};

TEST_F(LoopTest, OpenReaderReturnsCorrectSize) {
    auto reader = ASSERT_NIL_P(
        loop->open_reader({ethercat::PDOEntry(0, 0x6000, 1, 16, true)})
    );
    EXPECT_EQ(reader.size(), 2); // 16 bits = 2 bytes
}

TEST_F(LoopTest, OpenWriterSucceeds) {
    auto writer = ASSERT_NIL_P(
        loop->open_writer({ethercat::PDOEntry(0, 0x7000, 1, 16, false)})
    );
    // Writer created successfully - can write data
    uint16_t value = 0x1234;
    writer.write(0, &value, sizeof(value));
}

TEST_F(LoopTest, OpenReaderWithMultiplePDOs) {
    auto reader = ASSERT_NIL_P(loop->open_reader(
        {ethercat::PDOEntry(0, 0x6000, 1, 16, true),
         ethercat::PDOEntry(0, 0x6000, 2, 32, true)}
    ));
    EXPECT_EQ(reader.size(), 6); // 2 + 4 bytes
}

TEST_F(LoopTest, ReadReturnsData) {
    auto reader = ASSERT_NIL_P(
        loop->open_reader({ethercat::PDOEntry(0, 0x6000, 1, 16, true)})
    );

    std::atomic<bool> stopped{false};
    std::vector<uint8_t> buffer;
    ASSERT_NIL(reader.read(buffer, stopped));
    EXPECT_GE(buffer.size(), reader.size());
}

TEST_F(LoopTest, ReadReturnsErrorWhenStopped) {
    auto reader = ASSERT_NIL_P(
        loop->open_reader({ethercat::PDOEntry(0, 0x6000, 1, 16, true)})
    );

    std::atomic<bool> stopped{true};
    std::vector<uint8_t> buffer;
    ASSERT_OCCURRED_AS(reader.read(buffer, stopped), ethercat::CYCLIC_ERROR);
}

TEST_F(LoopTest, WriteSucceeds) {
    auto writer = ASSERT_NIL_P(
        loop->open_writer({ethercat::PDOEntry(0, 0x7000, 1, 16, false)})
    );

    uint16_t value = 0x1234;
    writer.write(0, &value, sizeof(value));
}

TEST_F(LoopTest, InitializeErrorPropagates) {
    mock_master->inject_init_error(
        xerrors::Error(ethercat::MASTER_INIT_ERROR, "init failed")
    );

    ASSERT_OCCURRED_AS(loop->open_reader({}).second, ethercat::MASTER_INIT_ERROR);
}

TEST_F(LoopTest, ActivateErrorPropagates) {
    mock_master->inject_activate_error(
        xerrors::Error(ethercat::ACTIVATION_ERROR, "activate failed")
    );

    ASSERT_OCCURRED_AS(loop->open_reader({}).second, ethercat::ACTIVATION_ERROR);
}

TEST_F(LoopTest, MultipleReadersCanRead) {
    auto reader1 = ASSERT_NIL_P(
        loop->open_reader({ethercat::PDOEntry(0, 0x6000, 1, 16, true)})
    );

    auto reader2 = ASSERT_NIL_P(
        loop->open_reader({ethercat::PDOEntry(0, 0x6000, 2, 32, true)})
    );

    std::atomic<bool> stopped{false};
    std::vector<uint8_t> buffer;

    ASSERT_NIL(reader1.read(buffer, stopped));
    ASSERT_NIL(reader2.read(buffer, stopped));
}

TEST_F(LoopTest, MultipleSlavesPDORegistration) {
    auto multi_master = std::make_shared<ethercat::mock::Master>("eth0");
    multi_master->add_slave(ethercat::mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));
    multi_master->add_slave(ethercat::mock::MockSlaveConfig(1, 0x1, 0x3, "Slave2"));

    auto multi_loop = std::make_shared<ethercat::Loop>(
        multi_master,
        ethercat::LoopConfig(telem::MILLISECOND * 10)
    );

    auto reader = ASSERT_NIL_P(multi_loop->open_reader(
        {ethercat::PDOEntry(0, 0x6000, 1, 16, true),
         ethercat::PDOEntry(1, 0x6000, 1, 32, true)}
    ));

    EXPECT_EQ(reader.size(), 6); // 2 + 4 bytes
}

TEST_F(LoopTest, MixedReadersAndWriters) {
    auto reader = ASSERT_NIL_P(
        loop->open_reader({ethercat::PDOEntry(0, 0x6000, 1, 16, true)})
    );

    auto writer = ASSERT_NIL_P(
        loop->open_writer({ethercat::PDOEntry(0, 0x7000, 1, 16, false)})
    );

    std::atomic<bool> stopped{false};
    std::vector<uint8_t> buffer;
    ASSERT_NIL(reader.read(buffer, stopped));

    uint16_t write_value = 0x5678;
    writer.write(0, &write_value, sizeof(write_value));
}

TEST_F(LoopTest, ReaderMoveSemantics) {
    auto reader1 = ASSERT_NIL_P(
        loop->open_reader({ethercat::PDOEntry(0, 0x6000, 1, 16, true)})
    );

    ethercat::Loop::Reader reader2 = std::move(reader1);
    EXPECT_EQ(reader2.size(), 2);

    std::atomic<bool> stopped{false};
    std::vector<uint8_t> buffer;
    ASSERT_NIL(reader2.read(buffer, stopped));
}

TEST_F(LoopTest, WriterMoveSemantics) {
    auto writer1 = ASSERT_NIL_P(
        loop->open_writer({ethercat::PDOEntry(0, 0x7000, 1, 16, false)})
    );

    ethercat::Loop::Writer writer2 = std::move(writer1);

    uint16_t value = 0x1234;
    writer2.write(0, &value, sizeof(value));
}

TEST_F(LoopTest, ReadAfterReconfigure) {
    auto reader1 = ASSERT_NIL_P(
        loop->open_reader({ethercat::PDOEntry(0, 0x6000, 1, 16, true)})
    );

    std::atomic<bool> stopped{false};
    std::vector<uint8_t> buffer;
    ASSERT_NIL(reader1.read(buffer, stopped));

    auto reader2 = ASSERT_NIL_P(
        loop->open_reader({ethercat::PDOEntry(0, 0x6000, 2, 32, true)})
    );

    ASSERT_NIL(reader1.read(buffer, stopped));
    ASSERT_NIL(reader2.read(buffer, stopped));
}
