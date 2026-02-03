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
#include "driver/ethercat/engine/pool.h"
#include "driver/ethercat/mock/master.h"

class EngineTest : public ::testing::Test {
protected:
    std::shared_ptr<ethercat::mock::Master> mock_master;
    std::shared_ptr<ethercat::engine::Engine> engine;

    void SetUp() override {
        mock_master = std::make_shared<ethercat::mock::Master>("eth0");
        mock_master->add_slave(ethercat::mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));
        engine = std::make_shared<ethercat::engine::Engine>(mock_master);
    }
};

TEST_F(EngineTest, OpenReaderReturnsCorrectSize) {
    auto reader = ASSERT_NIL_P(engine->open_reader(
        {ethercat::PDOEntry(0, 0x6000, 1, 16, true)},
        telem::Rate(100)
    ));
    EXPECT_EQ(reader->size(), 2);
}

TEST_F(EngineTest, OpenWriterSucceeds) {
    auto writer = ASSERT_NIL_P(engine->open_writer(
        {ethercat::PDOEntry(0, 0x7000, 1, 16, false)},
        telem::Rate(100)
    ));
    writer->write(0, static_cast<uint16_t>(0x1234));
}

TEST_F(EngineTest, OpenReaderWithMultiplePDOs) {
    auto reader = ASSERT_NIL_P(engine->open_reader(
        {ethercat::PDOEntry(0, 0x6000, 1, 16, true),
         ethercat::PDOEntry(0, 0x6000, 2, 32, true)},
        telem::Rate(100)
    ));
    EXPECT_EQ(reader->size(), 6);
}

TEST_F(EngineTest, ReadReturnsData) {
    auto reader = ASSERT_NIL_P(engine->open_reader(
        {ethercat::PDOEntry(0, 0x6000, 1, 16, true)},
        telem::Rate(100)
    ));

    breaker::Breaker brk;
    brk.start();
    telem::Frame frame(1, telem::Series(telem::UINT16_T, 1));
    ASSERT_NIL(reader->read(brk, frame));
    EXPECT_EQ(frame.series->at(0).size(), 1);
    brk.stop();
}

TEST_F(EngineTest, ReadReturnsNilWhenBreakerStopped) {
    auto reader = ASSERT_NIL_P(engine->open_reader(
        {ethercat::PDOEntry(0, 0x6000, 1, 16, true)},
        telem::Rate(100)
    ));

    breaker::Breaker brk;
    telem::Frame frame(1, telem::Series(telem::UINT16_T, 1));
    ASSERT_NIL(reader->read(brk, frame));
}

TEST_F(EngineTest, WriteSucceeds) {
    auto writer = ASSERT_NIL_P(engine->open_writer(
        {ethercat::PDOEntry(0, 0x7000, 1, 16, false)},
        telem::Rate(100)
    ));
    writer->write(0, static_cast<uint16_t>(0x1234));
}

TEST_F(EngineTest, InitializeErrorPropagates) {
    mock_master->inject_init_error(
        xerrors::Error(ethercat::MASTER_INIT_ERROR, "init failed")
    );

    ASSERT_OCCURRED_AS(
        engine->open_reader({}, telem::Rate(100)).second,
        ethercat::MASTER_INIT_ERROR
    );
}

TEST_F(EngineTest, ActivateErrorPropagates) {
    mock_master->inject_activate_error(
        xerrors::Error(ethercat::ACTIVATION_ERROR, "activate failed")
    );

    ASSERT_OCCURRED_AS(
        engine->open_reader({}, telem::Rate(100)).second,
        ethercat::ACTIVATION_ERROR
    );
}

TEST_F(EngineTest, MultipleReadersCanRead) {
    auto reader1 = ASSERT_NIL_P(engine->open_reader(
        {ethercat::PDOEntry(0, 0x6000, 1, 16, true)},
        telem::Rate(100)
    ));

    auto reader2 = ASSERT_NIL_P(engine->open_reader(
        {ethercat::PDOEntry(0, 0x6000, 2, 32, true)},
        telem::Rate(100)
    ));

    breaker::Breaker brk;
    brk.start();

    telem::Frame frame1(1, telem::Series(telem::UINT16_T, 1));
    ASSERT_NIL(reader1->read(brk, frame1));

    telem::Frame frame2(2, telem::Series(telem::UINT32_T, 1));
    ASSERT_NIL(reader2->read(brk, frame2));

    brk.stop();
}

TEST_F(EngineTest, MultipleSlavesPDORegistration) {
    auto multi_master = std::make_shared<ethercat::mock::Master>("eth0");
    multi_master->add_slave(ethercat::mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));
    multi_master->add_slave(ethercat::mock::MockSlaveConfig(1, 0x1, 0x3, "Slave2"));

    auto multi_engine = std::make_shared<ethercat::engine::Engine>(multi_master);

    auto reader = ASSERT_NIL_P(multi_engine->open_reader(
        {ethercat::PDOEntry(0, 0x6000, 1, 16, true),
         ethercat::PDOEntry(1, 0x6000, 1, 32, true)},
        telem::Rate(100)
    ));

    EXPECT_EQ(reader->size(), 6);
}

TEST_F(EngineTest, MixedReadersAndWriters) {
    auto reader = ASSERT_NIL_P(engine->open_reader(
        {ethercat::PDOEntry(0, 0x6000, 1, 16, true)},
        telem::Rate(100)
    ));

    auto writer = ASSERT_NIL_P(engine->open_writer(
        {ethercat::PDOEntry(0, 0x7000, 1, 16, false)},
        telem::Rate(100)
    ));

    breaker::Breaker brk;
    brk.start();
    telem::Frame frame(1, telem::Series(telem::UINT16_T, 1));
    ASSERT_NIL(reader->read(brk, frame));

    writer->write(0, static_cast<uint16_t>(0x5678));
    brk.stop();
}

TEST_F(EngineTest, ReadAfterReconfigure) {
    auto reader1 = ASSERT_NIL_P(engine->open_reader(
        {ethercat::PDOEntry(0, 0x6000, 1, 16, true)},
        telem::Rate(100)
    ));

    breaker::Breaker brk;
    brk.start();

    telem::Frame frame1(1, telem::Series(telem::UINT16_T, 1));
    ASSERT_NIL(reader1->read(brk, frame1));

    auto reader2 = ASSERT_NIL_P(engine->open_reader(
        {ethercat::PDOEntry(0, 0x6000, 2, 32, true)},
        telem::Rate(100)
    ));

    telem::Frame frame2(2, telem::Series(telem::UINT32_T, 1));

    ASSERT_NIL(reader1->read(brk, frame1));
    ASSERT_NIL(reader2->read(brk, frame2));
    brk.stop();
}

TEST_F(EngineTest, WriteTypeConversionFloatToInt16) {
    auto writer = ASSERT_NIL_P(engine->open_writer(
        {ethercat::PDOEntry(0, 0x7000, 1, 16, false, telem::INT16_T)},
        telem::Rate(100)
    ));
    writer->write(0, 42.7f);
    ASSERT_EVENTUALLY_EQ(
        this->mock_master->get_output<int16_t>(0),
        static_cast<int16_t>(42)
    );
}

TEST_F(EngineTest, WriteTypeConversionInt64ToInt32) {
    auto writer = ASSERT_NIL_P(engine->open_writer(
        {ethercat::PDOEntry(0, 0x7000, 1, 32, false, telem::INT32_T)},
        telem::Rate(100)
    ));
    writer->write(0, static_cast<int64_t>(0x12345678));
    ASSERT_EVENTUALLY_EQ(
        this->mock_master->get_output<int32_t>(0),
        static_cast<int32_t>(0x12345678)
    );
}

TEST_F(EngineTest, WriteSubByteSingleByte) {
    auto writer = ASSERT_NIL_P(engine->open_writer(
        {ethercat::PDOEntry(0, 0x7000, 1, 4, false, telem::UINT8_T)},
        telem::Rate(100)
    ));
    writer->write(0, static_cast<uint8_t>(0x0F));
    ASSERT_EVENTUALLY_EQ(
        static_cast<uint8_t>(this->mock_master->get_output<uint8_t>(0) & 0x0F),
        static_cast<uint8_t>(0x0F)
    );
}

TEST_F(EngineTest, Write24BitNoOffset) {
    auto writer = ASSERT_NIL_P(engine->open_writer(
        {ethercat::PDOEntry(0, 0x7000, 1, 24, false, telem::INT32_T)},
        telem::Rate(100)
    ));
    writer->write(0, static_cast<int32_t>(0x123456));
    ASSERT_EVENTUALLY_EQ(
        this->mock_master->get_output<uint8_t>(0),
        static_cast<uint8_t>(0x56)
    );
    EXPECT_EQ(this->mock_master->get_output<uint8_t>(1), static_cast<uint8_t>(0x34));
    EXPECT_EQ(this->mock_master->get_output<uint8_t>(2), static_cast<uint8_t>(0x12));
}

TEST_F(EngineTest, Write24BitSignedNegative) {
    auto writer = ASSERT_NIL_P(engine->open_writer(
        {ethercat::PDOEntry(0, 0x7000, 1, 24, false, telem::INT32_T)},
        telem::Rate(100)
    ));
    writer->write(0, static_cast<int32_t>(-1));
    ASSERT_EVENTUALLY_EQ(
        this->mock_master->get_output<uint8_t>(0),
        static_cast<uint8_t>(0xFF)
    );
    EXPECT_EQ(this->mock_master->get_output<uint8_t>(1), static_cast<uint8_t>(0xFF));
    EXPECT_EQ(this->mock_master->get_output<uint8_t>(2), static_cast<uint8_t>(0xFF));
}

class EngineReadValueTest : public ::testing::Test {
protected:
    std::shared_ptr<ethercat::mock::Master> mock_master;
    std::shared_ptr<ethercat::engine::Engine> engine;

    void SetUp() override {
        mock_master = std::make_shared<ethercat::mock::Master>("eth0");
    }

    void create_engine() {
        engine = std::make_shared<ethercat::engine::Engine>(mock_master);
    }
};

TEST_F(EngineReadValueTest, ReadValueInt16) {
    ethercat::PDOEntryInfo pdo_info;
    pdo_info.pdo_index = 0x1A00;
    pdo_info.index = 0x6000;
    pdo_info.subindex = 1;
    pdo_info.bit_length = 16;
    pdo_info.is_input = true;
    pdo_info.name = "status_word";
    pdo_info.data_type = telem::INT16_T;

    this->mock_master->add_slave(
        ethercat::mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1")
            .with_input_pdos({pdo_info})
    );
    this->create_engine();

    auto reader = ASSERT_NIL_P(this->engine->open_reader(
        {ethercat::PDOEntry(0, 0x6000, 1, 16, true, telem::INT16_T)},
        telem::Rate(100)
    ));

    this->mock_master->set_input<int16_t>(0, 0x1234);

    breaker::Breaker brk;
    brk.start();
    telem::Frame frame(1, telem::Series(telem::INT16_T, 1));
    ASSERT_NIL(reader->read(brk, frame));
    brk.stop();

    ASSERT_EQ(frame.series->at(0).size(), 1);
    EXPECT_EQ(frame.series->at(0).at<int16_t>(0), static_cast<int16_t>(0x1234));
}

TEST_F(EngineReadValueTest, ReadValueInt32) {
    ethercat::PDOEntryInfo pdo_info;
    pdo_info.pdo_index = 0x1A00;
    pdo_info.index = 0x6000;
    pdo_info.subindex = 1;
    pdo_info.bit_length = 32;
    pdo_info.is_input = true;
    pdo_info.name = "position";
    pdo_info.data_type = telem::INT32_T;

    this->mock_master->add_slave(
        ethercat::mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1")
            .with_input_pdos({pdo_info})
    );
    this->create_engine();

    auto reader = ASSERT_NIL_P(this->engine->open_reader(
        {ethercat::PDOEntry(0, 0x6000, 1, 32, true, telem::INT32_T)},
        telem::Rate(100)
    ));

    this->mock_master->set_input<int32_t>(0, 0x12345678);

    breaker::Breaker brk;
    brk.start();
    telem::Frame frame(1, telem::Series(telem::INT32_T, 1));
    ASSERT_NIL(reader->read(brk, frame));
    brk.stop();

    ASSERT_EQ(frame.series->at(0).size(), 1);
    EXPECT_EQ(frame.series->at(0).at<int32_t>(0), static_cast<int32_t>(0x12345678));
}

TEST_F(EngineReadValueTest, ReadValueMultiplePDOs) {
    ethercat::PDOEntryInfo pdo1;
    pdo1.pdo_index = 0x1A00;
    pdo1.index = 0x6000;
    pdo1.subindex = 1;
    pdo1.bit_length = 16;
    pdo1.is_input = true;
    pdo1.name = "status_word";
    pdo1.data_type = telem::INT16_T;

    ethercat::PDOEntryInfo pdo2;
    pdo2.pdo_index = 0x1A00;
    pdo2.index = 0x6000;
    pdo2.subindex = 2;
    pdo2.bit_length = 32;
    pdo2.is_input = true;
    pdo2.name = "position";
    pdo2.data_type = telem::INT32_T;

    this->mock_master->add_slave(
        ethercat::mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1")
            .with_input_pdos({pdo1, pdo2})
    );
    this->create_engine();

    auto reader = ASSERT_NIL_P(this->engine->open_reader(
        {ethercat::PDOEntry(0, 0x6000, 1, 16, true, telem::INT16_T),
         ethercat::PDOEntry(0, 0x6000, 2, 32, true, telem::INT32_T)},
        telem::Rate(100)
    ));

    this->mock_master->set_input<int16_t>(0, 0x1234);
    this->mock_master->set_input<int32_t>(2, 0xDEADBEEF);

    breaker::Breaker brk;
    brk.start();
    telem::Frame frame(2);
    frame.series->push_back(telem::Series(telem::INT16_T, 1));
    frame.series->push_back(telem::Series(telem::INT32_T, 1));
    ASSERT_NIL(reader->read(brk, frame));
    brk.stop();

    ASSERT_EQ(frame.series->at(0).size(), 1);
    ASSERT_EQ(frame.series->at(1).size(), 1);
    EXPECT_EQ(frame.series->at(0).at<int16_t>(0), static_cast<int16_t>(0x1234));
    EXPECT_EQ(frame.series->at(1).at<int32_t>(0), static_cast<int32_t>(0xDEADBEEF));
}

TEST_F(EngineReadValueTest, ReadValue24BitPositive) {
    ethercat::PDOEntryInfo pdo_info;
    pdo_info.pdo_index = 0x1A00;
    pdo_info.index = 0x6000;
    pdo_info.subindex = 1;
    pdo_info.bit_length = 24;
    pdo_info.is_input = true;
    pdo_info.name = "position_24bit";
    pdo_info.data_type = telem::INT32_T;

    this->mock_master->add_slave(
        ethercat::mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1")
            .with_input_pdos({pdo_info})
    );
    this->create_engine();

    auto reader = ASSERT_NIL_P(this->engine->open_reader(
        {ethercat::PDOEntry(0, 0x6000, 1, 24, true, telem::INT32_T)},
        telem::Rate(100)
    ));

    this->mock_master->set_input<uint8_t>(0, 0x56);
    this->mock_master->set_input<uint8_t>(1, 0x34);
    this->mock_master->set_input<uint8_t>(2, 0x12);

    breaker::Breaker brk;
    brk.start();
    telem::Frame frame(1, telem::Series(telem::INT32_T, 1));
    ASSERT_NIL(reader->read(brk, frame));
    brk.stop();

    ASSERT_EQ(frame.series->at(0).size(), 1);
    EXPECT_EQ(frame.series->at(0).at<int32_t>(0), static_cast<int32_t>(0x123456));
}

TEST_F(EngineReadValueTest, ReadValue24BitNegative) {
    ethercat::PDOEntryInfo pdo_info;
    pdo_info.pdo_index = 0x1A00;
    pdo_info.index = 0x6000;
    pdo_info.subindex = 1;
    pdo_info.bit_length = 24;
    pdo_info.is_input = true;
    pdo_info.name = "position_24bit";
    pdo_info.data_type = telem::INT32_T;

    this->mock_master->add_slave(
        ethercat::mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1")
            .with_input_pdos({pdo_info})
    );
    this->create_engine();

    auto reader = ASSERT_NIL_P(this->engine->open_reader(
        {ethercat::PDOEntry(0, 0x6000, 1, 24, true, telem::INT32_T)},
        telem::Rate(100)
    ));

    this->mock_master->set_input<uint8_t>(0, 0xFF);
    this->mock_master->set_input<uint8_t>(1, 0xFF);
    this->mock_master->set_input<uint8_t>(2, 0xFF);

    breaker::Breaker brk;
    brk.start();
    telem::Frame frame(1, telem::Series(telem::INT32_T, 1));
    ASSERT_NIL(reader->read(brk, frame));
    brk.stop();

    ASSERT_EQ(frame.series->at(0).size(), 1);
    EXPECT_EQ(frame.series->at(0).at<int32_t>(0), static_cast<int32_t>(-1));
}

TEST_F(EngineReadValueTest, ReadValueSubByte4Bit) {
    ethercat::PDOEntryInfo pdo_info;
    pdo_info.pdo_index = 0x1A00;
    pdo_info.index = 0x6000;
    pdo_info.subindex = 1;
    pdo_info.bit_length = 4;
    pdo_info.is_input = true;
    pdo_info.name = "nibble";
    pdo_info.data_type = telem::UINT8_T;

    this->mock_master->add_slave(
        ethercat::mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1")
            .with_input_pdos({pdo_info})
    );
    this->create_engine();

    auto reader = ASSERT_NIL_P(this->engine->open_reader(
        {ethercat::PDOEntry(0, 0x6000, 1, 4, true, telem::UINT8_T)},
        telem::Rate(100)
    ));

    this->mock_master->set_input<uint8_t>(0, 0xAF);

    breaker::Breaker brk;
    brk.start();
    telem::Frame frame(1, telem::Series(telem::UINT8_T, 1));
    ASSERT_NIL(reader->read(brk, frame));
    brk.stop();

    ASSERT_EQ(frame.series->at(0).size(), 1);
    EXPECT_EQ(frame.series->at(0).at<uint8_t>(0), static_cast<uint8_t>(0x0F));
}

TEST_F(EngineTest, EnsureInitializedIdempotent) {
    ASSERT_NIL(this->engine->ensure_initialized());
    ASSERT_NIL(this->engine->ensure_initialized());
}

TEST_F(EngineTest, SlavesReturnsDiscoveredSlaves) {
    auto multi_master = std::make_shared<ethercat::mock::Master>("eth0");
    multi_master->add_slave(ethercat::mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));
    multi_master->add_slave(ethercat::mock::MockSlaveConfig(1, 0x3, 0x4, "Slave2"));
    auto multi_engine = std::make_shared<ethercat::engine::Engine>(multi_master);

    ASSERT_NIL(multi_engine->ensure_initialized());

    auto slaves = multi_engine->slaves();
    ASSERT_EQ(slaves.size(), 2);
    EXPECT_EQ(slaves[0].position, 0);
    EXPECT_EQ(slaves[1].position, 1);
}

TEST_F(EngineTest, InterfaceNameReturnsCorrect) {
    EXPECT_EQ(this->engine->interface_name(), "eth0");
}

TEST(PoolTest, DiscoverSlavesCreatesEngine) {
    auto mock_master = std::make_shared<ethercat::mock::Master>("eth0");
    mock_master->add_slave(ethercat::mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));
    mock_master->add_slave(ethercat::mock::MockSlaveConfig(1, 0x3, 0x4, "Slave2"));

    auto manager = std::make_unique<ethercat::mock::Manager>();
    manager->configure("eth0", mock_master);

    ethercat::engine::Pool pool(std::move(manager));

    auto slaves = ASSERT_NIL_P(pool.discover_slaves("eth0"));
    ASSERT_EQ(slaves.size(), 2);
    EXPECT_EQ(slaves[0].position, 0);
    EXPECT_EQ(slaves[1].position, 1);
}

TEST(PoolTest, DiscoverSlavesReturnsFromRunningEngine) {
    auto mock_master = std::make_shared<ethercat::mock::Master>("eth0");
    mock_master->add_slave(ethercat::mock::MockSlaveConfig(0, 0x1, 0x2, "Slave1"));
    mock_master->add_slave(ethercat::mock::MockSlaveConfig(1, 0x3, 0x4, "Slave2"));

    auto manager = std::make_unique<ethercat::mock::Manager>();
    manager->configure("eth0", mock_master);

    ethercat::engine::Pool pool(std::move(manager));

    auto engine = ASSERT_NIL_P(pool.acquire("eth0"));
    auto reader = ASSERT_NIL_P(engine->open_reader(
        {ethercat::PDOEntry(0, 0x6000, 1, 16, true)},
        telem::Rate(100)
    ));

    EXPECT_TRUE(pool.is_active("eth0"));

    auto slaves = ASSERT_NIL_P(pool.discover_slaves("eth0"));
    ASSERT_EQ(slaves.size(), 2);
    EXPECT_EQ(slaves[0].position, 0);
    EXPECT_EQ(slaves[1].position, 1);
}

TEST(PoolTest, DiscoverSlavesInitErrorNotCached) {
    auto mock_master = std::make_shared<ethercat::mock::Master>("eth0");
    mock_master->inject_init_error(
        xerrors::Error(ethercat::MASTER_INIT_ERROR, "no interface")
    );

    auto manager = std::make_unique<ethercat::mock::Manager>();
    manager->configure("eth0", mock_master);

    ethercat::engine::Pool pool(std::move(manager));

    auto slaves = ASSERT_OCCURRED_AS_P(
        pool.discover_slaves("eth0"),
        ethercat::MASTER_INIT_ERROR
    );
    EXPECT_TRUE(slaves.empty());
    EXPECT_FALSE(pool.is_active("eth0"));
}
