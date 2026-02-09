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
#include "x/cpp/test/test.h"

#include "driver/ethercat/engine/engine.h"
#include "driver/ethercat/engine/pool.h"
#include "driver/ethercat/mock/master.h"

namespace driver::ethercat::engine {
class EngineTest : public ::testing::Test {
protected:
    std::shared_ptr<mock::Master> mock_master;
    std::shared_ptr<Engine> engine;

    void SetUp() override {
        mock_master = std::make_shared<mock::Master>("eth0");
        mock_master->add_slave(
            slave::Properties{
                .position = 0,
                .vendor_id = 0x1,
                .product_code = 0x2,
                .name = "Slave1"
            }
        );
        engine = std::make_shared<Engine>(mock_master);
    }
};

TEST_F(EngineTest, OpenReaderReturnsCorrectSize) {
    auto reader = ASSERT_NIL_P(
        engine->open_reader({pdo::Entry(0, 0x6000, 1, 16, true)}, x::telem::Rate(100))
    );
    EXPECT_EQ(reader->size(), 2);
}

TEST_F(EngineTest, OpenWriterSucceeds) {
    auto writer = ASSERT_NIL_P(
        engine->open_writer({pdo::Entry(0, 0x7000, 1, 16, false)}, x::telem::Rate(100))
    );
    writer->write(0, static_cast<uint16_t>(0x1234));
}

TEST_F(EngineTest, OpenReaderWithMultiplePDOs) {
    auto reader = ASSERT_NIL_P(engine->open_reader(
        {pdo::Entry(0, 0x6000, 1, 16, true), pdo::Entry(0, 0x6000, 2, 32, true)},
        x::telem::Rate(100)
    ));
    EXPECT_EQ(reader->size(), 6);
}

TEST_F(EngineTest, ReadReturnsData) {
    auto reader = ASSERT_NIL_P(
        engine->open_reader({pdo::Entry(0, 0x6000, 1, 16, true)}, x::telem::Rate(100))
    );

    x::breaker::Breaker brk;
    brk.start();
    x::telem::Frame frame(1, x::telem::Series(x::telem::UINT16_T, 1));
    ASSERT_NIL(reader->read(brk, frame));
    EXPECT_EQ(frame.series->at(0).size(), 1);
    brk.stop();
}

TEST_F(EngineTest, ReadReturnsNilWhenBreakerStopped) {
    auto reader = ASSERT_NIL_P(
        engine->open_reader({pdo::Entry(0, 0x6000, 1, 16, true)}, x::telem::Rate(100))
    );

    x::breaker::Breaker brk;
    x::telem::Frame frame(1, x::telem::Series(x::telem::UINT16_T, 1));
    ASSERT_NIL(reader->read(brk, frame));
}

TEST_F(EngineTest, WriteSucceeds) {
    auto writer = ASSERT_NIL_P(
        engine->open_writer({pdo::Entry(0, 0x7000, 1, 16, false)}, x::telem::Rate(100))
    );
    writer->write(0, static_cast<uint16_t>(0x1234));
}

TEST_F(EngineTest, InitializeErrorPropagates) {
    mock_master->inject_init_error(
        x::errors::Error(errors::MASTER_INIT_ERROR, "init failed")
    );

    ASSERT_OCCURRED_AS(
        engine->open_reader({}, x::telem::Rate(100)).second,
        errors::MASTER_INIT_ERROR
    );
}

TEST_F(EngineTest, ActivateErrorPropagates) {
    mock_master->inject_activate_error(
        x::errors::Error(errors::ACTIVATION_ERROR, "activate failed")
    );

    ASSERT_OCCURRED_AS(
        engine->open_reader({}, x::telem::Rate(100)).second,
        errors::ACTIVATION_ERROR
    );
}

TEST_F(EngineTest, MultipleReadersCanRead) {
    auto reader1 = ASSERT_NIL_P(
        engine->open_reader({pdo::Entry(0, 0x6000, 1, 16, true)}, x::telem::Rate(100))
    );

    auto reader2 = ASSERT_NIL_P(
        engine->open_reader({pdo::Entry(0, 0x6000, 2, 32, true)}, x::telem::Rate(100))
    );

    x::breaker::Breaker brk;
    brk.start();

    x::telem::Frame frame1(1, x::telem::Series(x::telem::UINT16_T, 1));
    ASSERT_NIL(reader1->read(brk, frame1));

    x::telem::Frame frame2(2, x::telem::Series(x::telem::UINT32_T, 1));
    ASSERT_NIL(reader2->read(brk, frame2));

    brk.stop();
}

TEST_F(EngineTest, MultipleSlavesPDORegistration) {
    auto multi_master = std::make_shared<mock::Master>("eth0");
    multi_master->add_slave(
        slave::Properties{
            .position = 0,
            .vendor_id = 0x1,
            .product_code = 0x2,
            .name = "Slave1"
        }
    );
    multi_master->add_slave(
        slave::Properties{
            .position = 1,
            .vendor_id = 0x1,
            .product_code = 0x3,
            .name = "Slave2"
        }
    );

    auto multi_engine = std::make_shared<Engine>(multi_master);

    auto reader = ASSERT_NIL_P(multi_engine->open_reader(
        {pdo::Entry(0, 0x6000, 1, 16, true), pdo::Entry(1, 0x6000, 1, 32, true)},
        x::telem::Rate(100)
    ));

    EXPECT_EQ(reader->size(), 6);
}

TEST_F(EngineTest, MixedReadersAndWriters) {
    auto reader = ASSERT_NIL_P(
        engine->open_reader({pdo::Entry(0, 0x6000, 1, 16, true)}, x::telem::Rate(100))
    );

    auto writer = ASSERT_NIL_P(
        engine->open_writer({pdo::Entry(0, 0x7000, 1, 16, false)}, x::telem::Rate(100))
    );

    x::breaker::Breaker brk;
    brk.start();
    x::telem::Frame frame(1, x::telem::Series(x::telem::UINT16_T, 1));
    ASSERT_NIL(reader->read(brk, frame));

    writer->write(0, static_cast<uint16_t>(0x5678));
    brk.stop();
}

TEST_F(EngineTest, ReadAfterReconfigure) {
    auto reader1 = ASSERT_NIL_P(
        engine->open_reader({pdo::Entry(0, 0x6000, 1, 16, true)}, x::telem::Rate(100))
    );

    x::breaker::Breaker brk;
    brk.start();

    x::telem::Frame frame1(1, x::telem::Series(x::telem::UINT16_T, 1));
    ASSERT_NIL(reader1->read(brk, frame1));

    auto reader2 = ASSERT_NIL_P(
        engine->open_reader({pdo::Entry(0, 0x6000, 2, 32, true)}, x::telem::Rate(100))
    );

    x::telem::Frame frame2(2, x::telem::Series(x::telem::UINT32_T, 1));

    ASSERT_NIL(reader1->read(brk, frame1));
    ASSERT_NIL(reader2->read(brk, frame2));
    brk.stop();
}

TEST_F(EngineTest, WriteTypeConversionFloatToInt16) {
    auto writer = ASSERT_NIL_P(engine->open_writer(
        {pdo::Entry(0, 0x7000, 1, 16, false, x::telem::INT16_T)},
        x::telem::Rate(100)
    ));
    writer->write(0, 42.7f);
    ASSERT_EVENTUALLY_EQ(
        this->mock_master->get_output<int16_t>(0),
        static_cast<int16_t>(42)
    );
}

TEST_F(EngineTest, WriteTypeConversionInt64ToInt32) {
    auto writer = ASSERT_NIL_P(engine->open_writer(
        {pdo::Entry(0, 0x7000, 1, 32, false, x::telem::INT32_T)},
        x::telem::Rate(100)
    ));
    writer->write(0, static_cast<int64_t>(0x12345678));
    ASSERT_EVENTUALLY_EQ(
        this->mock_master->get_output<int32_t>(0),
        static_cast<int32_t>(0x12345678)
    );
}

TEST_F(EngineTest, WriteSubByteSingleByte) {
    auto writer = ASSERT_NIL_P(engine->open_writer(
        {pdo::Entry(0, 0x7000, 1, 4, false, x::telem::UINT8_T)},
        x::telem::Rate(100)
    ));
    writer->write(0, static_cast<uint8_t>(0x0F));
    ASSERT_EVENTUALLY_EQ(
        static_cast<uint8_t>(this->mock_master->get_output<uint8_t>(0) & 0x0F),
        static_cast<uint8_t>(0x0F)
    );
}

TEST_F(EngineTest, Write24BitNoOffset) {
    auto writer = ASSERT_NIL_P(engine->open_writer(
        {pdo::Entry(0, 0x7000, 1, 24, false, x::telem::INT32_T)},
        x::telem::Rate(100)
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
        {pdo::Entry(0, 0x7000, 1, 24, false, x::telem::INT32_T)},
        x::telem::Rate(100)
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
    std::shared_ptr<mock::Master> mock_master;
    std::shared_ptr<Engine> engine;

    void SetUp() override { mock_master = std::make_shared<mock::Master>("eth0"); }

    void create_engine() { engine = std::make_shared<Engine>(mock_master); }
};

TEST_F(EngineReadValueTest, ReadValueInt16) {
    this->mock_master->add_slave(
        slave::Properties{
            .position = 0,
            .vendor_id = 0x1,
            .product_code = 0x2,
            .name = "Slave1",
            .input_pdos = {
                {.pdo_index = 0x1A00,
                 .index = 0x6000,
                 .sub_index = 1,
                 .bit_length = 16,
                 .is_input = true,
                 .name = "status_word",
                 .data_type = x::telem::INT16_T}
            },
        }
    );
    this->create_engine();

    auto reader = ASSERT_NIL_P(this->engine->open_reader(
        {pdo::Entry(0, 0x6000, 1, 16, true, x::telem::INT16_T)},
        x::telem::Rate(100)
    ));

    this->mock_master->set_input<int16_t>(0, 0x1234);

    x::breaker::Breaker brk;
    brk.start();
    x::telem::Frame frame(1, x::telem::Series(x::telem::INT16_T, 1));
    ASSERT_NIL(reader->read(brk, frame));
    brk.stop();

    ASSERT_EQ(frame.series->at(0).size(), 1);
    EXPECT_EQ(frame.series->at(0).at<int16_t>(0), static_cast<int16_t>(0x1234));
}

TEST_F(EngineReadValueTest, ReadValueInt32) {
    this->mock_master->add_slave(
        slave::Properties{
            .position = 0,
            .vendor_id = 0x1,
            .product_code = 0x2,
            .name = "Slave1",
            .input_pdos = {
                {.pdo_index = 0x1A00,
                 .index = 0x6000,
                 .sub_index = 1,
                 .bit_length = 32,
                 .is_input = true,
                 .name = "position",
                 .data_type = x::telem::INT32_T}
            },
        }
    );
    this->create_engine();

    auto reader = ASSERT_NIL_P(this->engine->open_reader(
        {pdo::Entry(0, 0x6000, 1, 32, true, x::telem::INT32_T)},
        x::telem::Rate(100)
    ));

    this->mock_master->set_input<int32_t>(0, 0x12345678);

    x::breaker::Breaker brk;
    brk.start();
    x::telem::Frame frame(1, x::telem::Series(x::telem::INT32_T, 1));
    ASSERT_NIL(reader->read(brk, frame));
    brk.stop();

    ASSERT_EQ(frame.series->at(0).size(), 1);
    EXPECT_EQ(frame.series->at(0).at<int32_t>(0), static_cast<int32_t>(0x12345678));
}

TEST_F(EngineReadValueTest, ReadValueMultiplePDOs) {
    this->mock_master->add_slave(
        slave::Properties{
            .position = 0,
            .vendor_id = 0x1,
            .product_code = 0x2,
            .name = "Slave1",
            .input_pdos = {
                {.pdo_index = 0x1A00,
                 .index = 0x6000,
                 .sub_index = 1,
                 .bit_length = 16,
                 .is_input = true,
                 .name = "status_word",
                 .data_type = x::telem::INT16_T},
                {.pdo_index = 0x1A00,
                 .index = 0x6000,
                 .sub_index = 2,
                 .bit_length = 32,
                 .is_input = true,
                 .name = "position",
                 .data_type = x::telem::INT32_T},
            },
        }
    );
    this->create_engine();

    auto reader = ASSERT_NIL_P(this->engine->open_reader(
        {pdo::Entry(0, 0x6000, 1, 16, true, x::telem::INT16_T),
         pdo::Entry(0, 0x6000, 2, 32, true, x::telem::INT32_T)},
        x::telem::Rate(100)
    ));

    this->mock_master->set_input<int16_t>(0, 0x1234);
    this->mock_master->set_input<int32_t>(2, 0xDEADBEEF);

    x::breaker::Breaker brk;
    brk.start();
    x::telem::Frame frame(2);
    frame.series->push_back(x::telem::Series(x::telem::INT16_T, 1));
    frame.series->push_back(x::telem::Series(x::telem::INT32_T, 1));
    ASSERT_NIL(reader->read(brk, frame));
    brk.stop();

    ASSERT_EQ(frame.series->at(0).size(), 1);
    ASSERT_EQ(frame.series->at(1).size(), 1);
    EXPECT_EQ(frame.series->at(0).at<int16_t>(0), static_cast<int16_t>(0x1234));
    EXPECT_EQ(frame.series->at(1).at<int32_t>(0), static_cast<int32_t>(0xDEADBEEF));
}

TEST_F(EngineReadValueTest, ReadValue24BitPositive) {
    this->mock_master->add_slave(
        slave::Properties{
            .position = 0,
            .vendor_id = 0x1,
            .product_code = 0x2,
            .name = "Slave1",
            .input_pdos = {
                {.pdo_index = 0x1A00,
                 .index = 0x6000,
                 .sub_index = 1,
                 .bit_length = 24,
                 .is_input = true,
                 .name = "position_24bit",
                 .data_type = x::telem::INT32_T}
            },
        }
    );
    this->create_engine();

    auto reader = ASSERT_NIL_P(this->engine->open_reader(
        {pdo::Entry(0, 0x6000, 1, 24, true, x::telem::INT32_T)},
        x::telem::Rate(100)
    ));

    this->mock_master->set_input<uint8_t>(0, 0x56);
    this->mock_master->set_input<uint8_t>(1, 0x34);
    this->mock_master->set_input<uint8_t>(2, 0x12);

    x::breaker::Breaker brk;
    brk.start();
    x::telem::Frame frame(1, x::telem::Series(x::telem::INT32_T, 1));
    ASSERT_NIL(reader->read(brk, frame));
    brk.stop();

    ASSERT_EQ(frame.series->at(0).size(), 1);
    EXPECT_EQ(frame.series->at(0).at<int32_t>(0), static_cast<int32_t>(0x123456));
}

TEST_F(EngineReadValueTest, ReadValue24BitNegative) {
    this->mock_master->add_slave(
        slave::Properties{
            .position = 0,
            .vendor_id = 0x1,
            .product_code = 0x2,
            .name = "Slave1",
            .input_pdos = {
                {.pdo_index = 0x1A00,
                 .index = 0x6000,
                 .sub_index = 1,
                 .bit_length = 24,
                 .is_input = true,
                 .name = "position_24bit",
                 .data_type = x::telem::INT32_T}
            },
        }
    );
    this->create_engine();

    auto reader = ASSERT_NIL_P(this->engine->open_reader(
        {pdo::Entry(0, 0x6000, 1, 24, true, x::telem::INT32_T)},
        x::telem::Rate(100)
    ));

    this->mock_master->set_input<uint8_t>(0, 0xFF);
    this->mock_master->set_input<uint8_t>(1, 0xFF);
    this->mock_master->set_input<uint8_t>(2, 0xFF);

    x::breaker::Breaker brk;
    brk.start();
    x::telem::Frame frame(1, x::telem::Series(x::telem::INT32_T, 1));
    ASSERT_NIL(reader->read(brk, frame));
    brk.stop();

    ASSERT_EQ(frame.series->at(0).size(), 1);
    EXPECT_EQ(frame.series->at(0).at<int32_t>(0), static_cast<int32_t>(-1));
}

TEST_F(EngineReadValueTest, ReadValueSubByte4Bit) {
    this->mock_master->add_slave(
        slave::Properties{
            .position = 0,
            .vendor_id = 0x1,
            .product_code = 0x2,
            .name = "Slave1",
            .input_pdos = {
                {.pdo_index = 0x1A00,
                 .index = 0x6000,
                 .sub_index = 1,
                 .bit_length = 4,
                 .is_input = true,
                 .name = "nibble",
                 .data_type = x::telem::UINT8_T}
            },
        }
    );
    this->create_engine();

    auto reader = ASSERT_NIL_P(this->engine->open_reader(
        {pdo::Entry(0, 0x6000, 1, 4, true, x::telem::UINT8_T)},
        x::telem::Rate(100)
    ));

    this->mock_master->set_input<uint8_t>(0, 0xAF);

    x::breaker::Breaker brk;
    brk.start();
    x::telem::Frame frame(1, x::telem::Series(x::telem::UINT8_T, 1));
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
    auto multi_master = std::make_shared<mock::Master>("eth0");
    multi_master->add_slave(
        slave::Properties{
            .position = 0,
            .vendor_id = 0x1,
            .product_code = 0x2,
            .name = "Slave1"
        }
    );
    multi_master->add_slave(
        slave::Properties{
            .position = 1,
            .vendor_id = 0x3,
            .product_code = 0x4,
            .name = "Slave2"
        }
    );
    auto multi_engine = std::make_shared<Engine>(multi_master);

    ASSERT_NIL(multi_engine->ensure_initialized());

    auto slaves = multi_engine->slaves();
    ASSERT_EQ(slaves.size(), 2);
    EXPECT_EQ(slaves[0].properties.position, 0);
    EXPECT_EQ(slaves[1].properties.position, 1);
}

TEST_F(EngineTest, InterfaceNameReturnsCorrect) {
    EXPECT_EQ(this->engine->interface_name(), "eth0");
}

TEST(PoolTest, DiscoverSlavesCreatesEngine) {
    auto mock_master = std::make_shared<mock::Master>("eth0");
    mock_master->add_slave(
        slave::Properties{
            .position = 0,
            .vendor_id = 0x1,
            .product_code = 0x2,
            .name = "Slave1"
        }
    );
    mock_master->add_slave(
        slave::Properties{
            .position = 1,
            .vendor_id = 0x3,
            .product_code = 0x4,
            .name = "Slave2"
        }
    );

    auto manager = std::make_unique<mock::Manager>();
    manager->configure("eth0", mock_master);

    Pool pool(std::move(manager));

    auto slaves = ASSERT_NIL_P(pool.discover_slaves("eth0"));
    ASSERT_EQ(slaves.size(), 2);
    EXPECT_EQ(slaves[0].properties.position, 0);
    EXPECT_EQ(slaves[1].properties.position, 1);
}

TEST(PoolTest, DiscoverSlavesReturnsFromRunningEngine) {
    auto mock_master = std::make_shared<mock::Master>("eth0");
    mock_master->add_slave(
        slave::Properties{
            .position = 0,
            .vendor_id = 0x1,
            .product_code = 0x2,
            .name = "Slave1"
        }
    );
    mock_master->add_slave(
        slave::Properties{
            .position = 1,
            .vendor_id = 0x3,
            .product_code = 0x4,
            .name = "Slave2"
        }
    );

    auto manager = std::make_unique<mock::Manager>();
    manager->configure("eth0", mock_master);

    Pool pool(std::move(manager));

    auto engine = ASSERT_NIL_P(pool.acquire("eth0"));
    auto reader = ASSERT_NIL_P(
        engine->open_reader({pdo::Entry(0, 0x6000, 1, 16, true)}, x::telem::Rate(100))
    );

    EXPECT_TRUE(pool.is_active("eth0"));

    auto slaves = ASSERT_NIL_P(pool.discover_slaves("eth0"));
    ASSERT_EQ(slaves.size(), 2);
    EXPECT_EQ(slaves[0].properties.position, 0);
    EXPECT_EQ(slaves[1].properties.position, 1);
}

TEST(PoolTest, DiscoverSlavesInitErrorNotCached) {
    auto mock_master = std::make_shared<mock::Master>("eth0");
    mock_master->inject_init_error(
        x::errors::Error(errors::MASTER_INIT_ERROR, "no interface")
    );

    auto manager = std::make_unique<mock::Manager>();
    manager->configure("eth0", mock_master);

    Pool pool(std::move(manager));

    auto slaves = ASSERT_OCCURRED_AS_P(
        pool.discover_slaves("eth0"),
        errors::MASTER_INIT_ERROR
    );
    EXPECT_TRUE(slaves.empty());
    EXPECT_FALSE(pool.is_active("eth0"));
}
}
