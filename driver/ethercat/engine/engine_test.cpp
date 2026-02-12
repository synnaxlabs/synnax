// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <thread>

#include "gtest/gtest.h"

#include "x/cpp/telem/frame.h"
#include "x/cpp/test/test.h"

#include "driver/ethercat/engine/engine.h"
#include "driver/ethercat/engine/pool.h"
#include "driver/ethercat/errors/errors.h"
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

TEST_F(EngineReadValueTest, ReaderBufferResizesAfterReconfigure) {
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
                 .name = "a",
                 .data_type = x::telem::UINT16_T},
                {.pdo_index = 0x1A00,
                 .index = 0x6000,
                 .sub_index = 2,
                 .bit_length = 32,
                 .is_input = true,
                 .name = "b",
                 .data_type = x::telem::UINT32_T},
            },
        }
    );
    this->create_engine();

    auto reader1 = ASSERT_NIL_P(this->engine->open_reader(
        {pdo::Entry(0, 0x6000, 1, 16, true, x::telem::UINT16_T)},
        x::telem::Rate(100)
    ));

    x::breaker::Breaker brk;
    brk.start();
    x::telem::Frame frame1(1, x::telem::Series(x::telem::UINT16_T, 1));
    ASSERT_NIL(reader1->read(brk, frame1));

    auto reader2 = ASSERT_NIL_P(this->engine->open_reader(
        {pdo::Entry(0, 0x6000, 2, 32, true, x::telem::UINT32_T)},
        x::telem::Rate(100)
    ));

    this->mock_master->set_input<uint16_t>(0, 0xABCD);

    auto read_u16 = [&]() -> uint16_t {
        x::telem::Frame f(1, x::telem::Series(x::telem::UINT16_T, 1));
        EXPECT_FALSE(reader1->read(brk, f));
        return f.series->at(0).at<uint16_t>(0);
    };
    ASSERT_EVENTUALLY_EQ(read_u16(), static_cast<uint16_t>(0xABCD));

    brk.stop();
}

TEST_F(EngineReadValueTest, ReadAfterReconfigureGetsCorrectValues) {
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
                 .name = "a",
                 .data_type = x::telem::UINT16_T},
                {.pdo_index = 0x1A00,
                 .index = 0x6000,
                 .sub_index = 2,
                 .bit_length = 32,
                 .is_input = true,
                 .name = "b",
                 .data_type = x::telem::INT32_T},
            },
        }
    );
    this->create_engine();

    auto reader1 = ASSERT_NIL_P(this->engine->open_reader(
        {pdo::Entry(0, 0x6000, 1, 16, true, x::telem::UINT16_T)},
        x::telem::Rate(100)
    ));

    this->mock_master->set_input<uint16_t>(0, 0x1234);

    x::breaker::Breaker brk;
    brk.start();
    x::telem::Frame frame1(1, x::telem::Series(x::telem::UINT16_T, 1));
    ASSERT_NIL(reader1->read(brk, frame1));
    EXPECT_EQ(frame1.series->at(0).at<uint16_t>(0), static_cast<uint16_t>(0x1234));

    auto reader2 = ASSERT_NIL_P(this->engine->open_reader(
        {pdo::Entry(0, 0x6000, 2, 32, true, x::telem::INT32_T)},
        x::telem::Rate(100)
    ));

    this->mock_master->set_input<uint16_t>(0, 0x5678);
    this->mock_master->set_input<int32_t>(2, 0xDEADBEEF);

    x::telem::Frame frame1b(1, x::telem::Series(x::telem::UINT16_T, 1));
    ASSERT_NIL(reader1->read(brk, frame1b));
    EXPECT_EQ(frame1b.series->at(0).at<uint16_t>(0), static_cast<uint16_t>(0x5678));

    x::telem::Frame frame2(1, x::telem::Series(x::telem::INT32_T, 1));
    ASSERT_NIL(reader2->read(brk, frame2));
    EXPECT_EQ(frame2.series->at(0).at<int32_t>(0), static_cast<int32_t>(0xDEADBEEF));

    brk.stop();
}

TEST_F(EngineReadValueTest, ReaderAfterRemovalAndReconfigure) {
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
                 .name = "a",
                 .data_type = x::telem::UINT16_T},
                {.pdo_index = 0x1A00,
                 .index = 0x6000,
                 .sub_index = 2,
                 .bit_length = 16,
                 .is_input = true,
                 .name = "b",
                 .data_type = x::telem::UINT16_T},
                {.pdo_index = 0x1A00,
                 .index = 0x6000,
                 .sub_index = 3,
                 .bit_length = 32,
                 .is_input = true,
                 .name = "c",
                 .data_type = x::telem::INT32_T},
            },
        }
    );
    this->create_engine();

    auto reader1 = ASSERT_NIL_P(this->engine->open_reader(
        {pdo::Entry(0, 0x6000, 1, 16, true, x::telem::UINT16_T)},
        x::telem::Rate(100)
    ));

    {
        auto reader2 = ASSERT_NIL_P(this->engine->open_reader(
            {pdo::Entry(0, 0x6000, 2, 16, true, x::telem::UINT16_T)},
            x::telem::Rate(100)
        ));
    }

    auto reader3 = ASSERT_NIL_P(this->engine->open_reader(
        {pdo::Entry(0, 0x6000, 3, 32, true, x::telem::INT32_T)},
        x::telem::Rate(100)
    ));

    this->mock_master->set_input<uint16_t>(0, 0xAAAA);
    this->mock_master->set_input<int32_t>(2, 0xBBBBCCCC);

    x::breaker::Breaker brk;
    brk.start();

    auto read_r1 = [&]() -> uint16_t {
        x::telem::Frame f(1, x::telem::Series(x::telem::UINT16_T, 1));
        EXPECT_FALSE(reader1->read(brk, f));
        return f.series->at(0).at<uint16_t>(0);
    };
    ASSERT_EVENTUALLY_EQ(read_r1(), static_cast<uint16_t>(0xAAAA));

    auto read_r3 = [&]() -> int32_t {
        x::telem::Frame f(1, x::telem::Series(x::telem::INT32_T, 1));
        EXPECT_FALSE(reader3->read(brk, f));
        return f.series->at(0).at<int32_t>(0);
    };
    ASSERT_EVENTUALLY_EQ(read_r3(), static_cast<int32_t>(0xBBBBCCCC));

    brk.stop();
}

TEST_F(EngineReadValueTest, ConcurrentOpenReaderAndRead) {
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
                 .name = "a",
                 .data_type = x::telem::UINT16_T},
                {.pdo_index = 0x1A00,
                 .index = 0x6000,
                 .sub_index = 2,
                 .bit_length = 32,
                 .is_input = true,
                 .name = "b",
                 .data_type = x::telem::UINT32_T},
            },
        }
    );
    this->create_engine();

    auto reader1 = ASSERT_NIL_P(this->engine->open_reader(
        {pdo::Entry(0, 0x6000, 1, 16, true, x::telem::UINT16_T)},
        x::telem::Rate(100)
    ));

    x::breaker::Breaker brk;
    brk.start();
    std::atomic<int> restarting_count{0};
    std::atomic<int> success_count{0};
    std::atomic<bool> done{false};

    std::thread reader_thread([&] {
        while (!done.load(std::memory_order_acquire)) {
            x::telem::Frame frame(1, x::telem::Series(x::telem::UINT16_T, 1));
            auto err = reader1->read(brk, frame);
            if (err.matches(errors::ENGINE_RESTARTING))
                restarting_count.fetch_add(1, std::memory_order_relaxed);
            else if (!err)
                success_count.fetch_add(1, std::memory_order_relaxed);
        }
    });

    auto reader2 = ASSERT_NIL_P(this->engine->open_reader(
        {pdo::Entry(0, 0x6000, 2, 32, true, x::telem::UINT32_T)},
        x::telem::Rate(100)
    ));

    ASSERT_EVENTUALLY_GE(success_count.load(std::memory_order_acquire), 3);

    done.store(true, std::memory_order_release);
    brk.stop();
    reader_thread.join();
}

TEST_F(EngineTest, WriteAfterReaderReconfigureWithOffsetShift) {
    // Writer gets offset 0 initially. Then we shift output offsets by 4 bytes
    // before opening a reader (which triggers reconfigure). The writer must
    // refresh its cached offset from 0 → 4 or it writes to the wrong location.
    auto writer = ASSERT_NIL_P(engine->open_writer(
        {pdo::Entry(0, 0x7000, 1, 16, false, x::telem::INT16_T)},
        x::telem::Rate(100)
    ));
    writer->write(0, static_cast<int16_t>(0x1234));
    ASSERT_EVENTUALLY_EQ(
        this->mock_master->get_output<int16_t>(0),
        static_cast<int16_t>(0x1234)
    );

    // Simulate real master behavior: offsets shift after reconfigure.
    this->mock_master->set_output_padding(4);

    auto reader = ASSERT_NIL_P(
        engine->open_reader({pdo::Entry(0, 0x6000, 1, 16, true)}, x::telem::Rate(100))
    );

    // PDO is now at byte 4 (shifted by padding). With stale offsets this
    // would write to byte 0 and the value at byte 4 would remain zero.
    writer->write(0, static_cast<int16_t>(0x5678));
    ASSERT_EVENTUALLY_EQ(
        this->mock_master->get_output<int16_t>(4),
        static_cast<int16_t>(0x5678)
    );
}

TEST_F(EngineTest, WriteAfterWriterReconfigureWithOffsetShift) {
    auto writer1 = ASSERT_NIL_P(engine->open_writer(
        {pdo::Entry(0, 0x7000, 1, 16, false, x::telem::INT16_T)},
        x::telem::Rate(100)
    ));
    writer1->write(0, static_cast<int16_t>(0x1234));
    ASSERT_EVENTUALLY_EQ(
        this->mock_master->get_output<int16_t>(0),
        static_cast<int16_t>(0x1234)
    );

    // Shift output offsets before opening second writer (triggers reconfigure).
    this->mock_master->set_output_padding(4);

    auto writer2 = ASSERT_NIL_P(engine->open_writer(
        {pdo::Entry(0, 0x7000, 2, 32, false, x::telem::INT32_T)},
        x::telem::Rate(100)
    ));

    // writer1's PDO shifted from byte 0 to byte 4, writer2's at byte 6.
    writer1->write(0, static_cast<int16_t>(0x5678));
    ASSERT_EVENTUALLY_EQ(
        this->mock_master->get_output<int16_t>(4),
        static_cast<int16_t>(0x5678)
    );

    writer2->write(0, static_cast<int32_t>(0xDEADBEEF));
    ASSERT_EVENTUALLY_EQ(
        this->mock_master->get_output<int32_t>(6),
        static_cast<int32_t>(0xDEADBEEF)
    );
}

TEST_F(EngineReadValueTest, MonotonicReadUnderChurn) {
    this->mock_master->add_slave(
        slave::Properties{
            .position = 0,
            .vendor_id = 0x1,
            .product_code = 0x2,
            .name = "Slave1",
            .input_pdos =
                {
                    {.pdo_index = 0x1A00,
                     .index = 0x6000,
                     .sub_index = 1,
                     .bit_length = 16,
                     .is_input = true,
                     .name = "a",
                     .data_type = x::telem::UINT16_T},
                    {.pdo_index = 0x1A00,
                     .index = 0x6000,
                     .sub_index = 2,
                     .bit_length = 32,
                     .is_input = true,
                     .name = "b",
                     .data_type = x::telem::UINT32_T},
                    {.pdo_index = 0x1A00,
                     .index = 0x6000,
                     .sub_index = 3,
                     .bit_length = 16,
                     .is_input = true,
                     .name = "c",
                     .data_type = x::telem::UINT16_T},
                },
            .output_pdos = {
                {.pdo_index = 0x1600,
                 .index = 0x7000,
                 .sub_index = 1,
                 .bit_length = 16,
                 .is_input = false,
                 .name = "out_a",
                 .data_type = x::telem::INT16_T},
                {.pdo_index = 0x1600,
                 .index = 0x7000,
                 .sub_index = 2,
                 .bit_length = 32,
                 .is_input = false,
                 .name = "out_b",
                 .data_type = x::telem::INT32_T},
            },
        }
    );
    this->create_engine();

    auto persistent_reader = ASSERT_NIL_P(this->engine->open_reader(
        {pdo::Entry(0, 0x6000, 1, 16, true, x::telem::UINT16_T),
         pdo::Entry(0, 0x6000, 2, 32, true, x::telem::UINT32_T)},
        x::telem::Rate(100)
    ));

    x::breaker::Breaker brk;
    brk.start();

    std::atomic<uint32_t> counter{1};
    std::atomic<bool> done{false};
    std::atomic<int> monotonic_violations{0};
    std::atomic<int> zero_after_nonzero{0};
    std::atomic<int> success_count{0};

    std::thread producer([&] {
        while (!done.load(std::memory_order_acquire)) {
            auto val = counter.fetch_add(1, std::memory_order_relaxed);
            this->mock_master->set_input<uint32_t>(2, val);
            std::this_thread::sleep_for(std::chrono::milliseconds(1));
        }
    });

    std::thread reader_thread([&] {
        uint32_t prev = 0;
        bool seen_nonzero = false;
        while (!done.load(std::memory_order_acquire)) {
            x::telem::Frame frame(2);
            frame.series->push_back(x::telem::Series(x::telem::UINT16_T, 1));
            frame.series->push_back(x::telem::Series(x::telem::UINT32_T, 1));
            auto err = persistent_reader->read(brk, frame);
            if (err || !brk.running()) continue;
            auto val = frame.series->at(1).at<uint32_t>(0);
            if (val != 0) seen_nonzero = true;
            if (seen_nonzero && val == 0)
                zero_after_nonzero.fetch_add(1, std::memory_order_relaxed);
            if (val < prev)
                monotonic_violations.fetch_add(1, std::memory_order_relaxed);
            if (val >= prev && val != 0) prev = val;
            success_count.fetch_add(1, std::memory_order_relaxed);
        }
    });

    constexpr int CYCLES = 10;

    std::thread reader_churn([&] {
        for (int i = 0; i < CYCLES; i++) {
            auto r = ASSERT_NIL_P(this->engine->open_reader(
                {pdo::Entry(0, 0x6000, 3, 16, true, x::telem::UINT16_T)},
                x::telem::Rate(100)
            ));
        }
    });

    std::thread writer_churn([&] {
        for (int i = 0; i < CYCLES; i++) {
            auto w = ASSERT_NIL_P(this->engine->open_writer(
                {pdo::Entry(0, 0x7000, 2, 32, false, x::telem::INT32_T)},
                x::telem::Rate(100)
            ));
        }
    });

    std::thread mixed_churn([&] {
        for (int i = 0; i < CYCLES; i++) {
            if (i % 2 == 0) {
                auto r = ASSERT_NIL_P(this->engine->open_reader(
                    {pdo::Entry(0, 0x6000, 1, 16, true, x::telem::UINT16_T)},
                    x::telem::Rate(100)
                ));
            } else {
                auto w = ASSERT_NIL_P(this->engine->open_writer(
                    {pdo::Entry(0, 0x7000, 1, 16, false, x::telem::INT16_T)},
                    x::telem::Rate(100)
                ));
            }
        }
    });

    reader_churn.join();
    writer_churn.join();
    mixed_churn.join();

    ASSERT_EVENTUALLY_GE(success_count.load(std::memory_order_acquire), 50);

    done.store(true, std::memory_order_release);
    brk.stop();
    producer.join();
    reader_thread.join();

    EXPECT_EQ(monotonic_violations.load(std::memory_order_relaxed), 0);
    EXPECT_EQ(zero_after_nonzero.load(std::memory_order_relaxed), 0);
}

TEST_F(EngineTest, WriterOffsetIntegrityUnderChurn) {
    auto writer = ASSERT_NIL_P(engine->open_writer(
        {pdo::Entry(0, 0x7000, 1, 16, false, x::telem::INT16_T)},
        x::telem::Rate(100)
    ));

    std::atomic<bool> done{false};
    std::thread writer_thread([&] {
        while (!done.load(std::memory_order_acquire)) {
            writer->write(0, static_cast<int16_t>(0x1234));
            std::this_thread::sleep_for(std::chrono::milliseconds(1));
        }
    });

    ASSERT_EVENTUALLY_EQ(
        this->mock_master->get_output<int16_t>(0),
        static_cast<int16_t>(0x1234)
    );

    this->mock_master->set_output_padding(4);
    {
        auto r = ASSERT_NIL_P(engine->open_reader(
            {pdo::Entry(0, 0x6000, 1, 16, true)},
            x::telem::Rate(100)
        ));
    }
    ASSERT_EVENTUALLY_EQ(
        this->mock_master->get_output<int16_t>(4),
        static_cast<int16_t>(0x1234)
    );

    this->mock_master->set_output_padding(8);
    {
        auto r = ASSERT_NIL_P(engine->open_reader(
            {pdo::Entry(0, 0x6000, 1, 16, true)},
            x::telem::Rate(100)
        ));
    }
    ASSERT_EVENTUALLY_EQ(
        this->mock_master->get_output<int16_t>(8),
        static_cast<int16_t>(0x1234)
    );

    this->mock_master->set_output_padding(0);
    {
        auto r = ASSERT_NIL_P(engine->open_reader(
            {pdo::Entry(0, 0x6000, 1, 16, true)},
            x::telem::Rate(100)
        ));
    }
    ASSERT_EVENTUALLY_EQ(
        this->mock_master->get_output<int16_t>(0),
        static_cast<int16_t>(0x1234)
    );

    done.store(true, std::memory_order_release);
    writer_thread.join();
}

TEST_F(EngineReadValueTest, MultiReaderReconfigureConsistency) {
    this->mock_master->add_slave(
        slave::Properties{
            .position = 0,
            .vendor_id = 0x1,
            .product_code = 0x2,
            .name = "Slave1",
            .input_pdos =
                {
                    {.pdo_index = 0x1A00,
                     .index = 0x6000,
                     .sub_index = 1,
                     .bit_length = 16,
                     .is_input = true,
                     .name = "a",
                     .data_type = x::telem::UINT16_T},
                    {.pdo_index = 0x1A00,
                     .index = 0x6000,
                     .sub_index = 2,
                     .bit_length = 32,
                     .is_input = true,
                     .name = "b",
                     .data_type = x::telem::UINT32_T},
                    {.pdo_index = 0x1A00,
                     .index = 0x6000,
                     .sub_index = 3,
                     .bit_length = 16,
                     .is_input = true,
                     .name = "c",
                     .data_type = x::telem::UINT16_T},
                },
            .output_pdos = {
                {.pdo_index = 0x1600,
                 .index = 0x7000,
                 .sub_index = 1,
                 .bit_length = 16,
                 .is_input = false,
                 .name = "out_a",
                 .data_type = x::telem::INT16_T},
            },
        }
    );
    this->create_engine();

    auto reader_a = ASSERT_NIL_P(this->engine->open_reader(
        {pdo::Entry(0, 0x6000, 1, 16, true, x::telem::UINT16_T),
         pdo::Entry(0, 0x6000, 2, 32, true, x::telem::UINT32_T)},
        x::telem::Rate(100)
    ));

    auto reader_b = ASSERT_NIL_P(this->engine->open_reader(
        {pdo::Entry(0, 0x6000, 3, 16, true, x::telem::UINT16_T)},
        x::telem::Rate(100)
    ));

    x::breaker::Breaker brk;
    brk.start();

    for (int cycle = 0; cycle < 3; cycle++) {
        auto val_a = static_cast<uint16_t>(0x1000 + cycle);
        auto val_b = static_cast<uint32_t>(0xAA000000 + cycle);
        auto val_c = static_cast<uint16_t>(0x2000 + cycle);

        this->mock_master->set_input<uint16_t>(0, val_a);
        this->mock_master->set_input<uint32_t>(2, val_b);
        this->mock_master->set_input<uint16_t>(6, val_c);

        auto read_a_0 = [&]() -> uint16_t {
            x::telem::Frame f(2);
            f.series->push_back(x::telem::Series(x::telem::UINT16_T, 1));
            f.series->push_back(x::telem::Series(x::telem::UINT32_T, 1));
            EXPECT_FALSE(reader_a->read(brk, f));
            return f.series->at(0).at<uint16_t>(0);
        };
        ASSERT_EVENTUALLY_EQ(read_a_0(), val_a);

        {
            x::telem::Frame fa(2);
            fa.series->push_back(x::telem::Series(x::telem::UINT16_T, 1));
            fa.series->push_back(x::telem::Series(x::telem::UINT32_T, 1));
            ASSERT_NIL(reader_a->read(brk, fa));
            EXPECT_EQ(fa.series->at(1).at<uint32_t>(0), val_b);
        }

        {
            auto read_b = [&]() -> uint16_t {
                x::telem::Frame fb(1, x::telem::Series(x::telem::UINT16_T, 1));
                EXPECT_FALSE(reader_b->read(brk, fb));
                return fb.series->at(0).at<uint16_t>(0);
            };
            ASSERT_EVENTUALLY_EQ(read_b(), val_c);
        }

        {
            auto transient = ASSERT_NIL_P(this->engine->open_writer(
                {pdo::Entry(0, 0x7000, 1, 16, false, x::telem::INT16_T)},
                x::telem::Rate(100)
            ));
        }
    }

    brk.stop();
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
