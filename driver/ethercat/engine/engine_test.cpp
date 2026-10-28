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
#include <thread>

#include "gtest/gtest.h"

#include "x/cpp/telem/frame.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/ethercat/engine/engine.h"
#include "driver/ethercat/engine/pool.h"
#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/testutil/master.h"

namespace driver::ethercat::engine {
namespace {
const pdo::Entry IN_I16(0, 0x6000, 1, 16, true, x::telem::INT16_T);
const pdo::Entry IN_I32(0, 0x6000, 2, 32, true, x::telem::INT32_T);
const pdo::Entry IN_I24(0, 0x6000, 3, 24, true, x::telem::INT32_T);
const pdo::Entry IN_U4(0, 0x6000, 4, 4, true, x::telem::UINT8_T);
const pdo::Entry OUT_I16(0, 0x7000, 1, 16, false, x::telem::INT16_T);
const pdo::Entry OUT_I32(0, 0x7000, 2, 32, false, x::telem::INT32_T);
const pdo::Entry OUT_I24(0, 0x7000, 3, 24, false, x::telem::INT32_T);
const pdo::Entry OUT_U4(0, 0x7000, 4, 4, false, x::telem::UINT8_T);

/// @brief byte offsets of the entries of io_slave in its slave's part of the image.
constexpr size_t I16_BYTE = 0;
constexpr size_t I32_BYTE = 2;
constexpr size_t I24_BYTE = 6;
constexpr size_t U4_BYTE = 9;
constexpr size_t SLAVE_IMAGE_SIZE = 10;

pdo::Properties pdo_of(const pdo::Entry &entry) {
    return {
        .pdo_index = static_cast<uint16_t>(entry.is_input ? 0x1A00 : 0x1600),
        .index = entry.index,
        .sub_index = entry.sub_index,
        .bit_length = entry.bit_length,
        .is_input = entry.is_input,
        .data_type = entry.data_type,
    };
}

/// @brief a slave that maps a 16-bit, a 32-bit, a 24-bit, and a 4-bit entry in each
/// direction.
slave::Properties io_slave(const uint16_t position) {
    return {
        .position = position,
        .vendor_id = 0x1,
        .product_code = 0x2,
        .name = "Slave" + std::to_string(position),
        .input_pdos = {pdo_of(IN_I16), pdo_of(IN_I32), pdo_of(IN_I24), pdo_of(IN_U4)},
        .output_pdos =
            {pdo_of(OUT_I16), pdo_of(OUT_I32), pdo_of(OUT_I24), pdo_of(OUT_U4)},
    };
}

pdo::Entry on_slave(pdo::Entry entry, const uint16_t position) {
    entry.slave_position = position;
    return entry;
}

x::telem::Frame frame_of(const std::vector<x::telem::DataType> &types, size_t n) {
    x::telem::Frame frame(types.size());
    for (const auto &dt: types)
        frame.series->push_back(x::telem::Series(dt, n));
    return frame;
}

/// @brief an engine configuration that stops trying to start the bus after two quick
/// retries.
Config quick_start_config() {
    Config cfg;
    cfg.start_retry = {
        .name = "ethercat_engine_test",
        .base_interval = x::telem::MILLISECOND,
        .max_retries = 2,
        .scale = 1,
        .max_interval = x::telem::MILLISECOND,
    };
    return cfg;
}
}

class EngineTest : public ::testing::Test {
protected:
    std::shared_ptr<testutil::Master> master;
    std::shared_ptr<Engine> engine;
    x::breaker::Breaker brk;

    void SetUp() override {
        this->master = std::make_shared<testutil::Master>("eth0");
        this->master->add_slave(io_slave(0));
        this->engine = std::make_shared<Engine>(this->master);
        this->brk.start();
    }

    void TearDown() override { this->brk.stop(); }

    /// @brief reads n samples of a single 16-bit entry and returns their times.
    x::telem::Series read_times(Engine::Reader &reader, const size_t n) {
        auto frame = frame_of({x::telem::INT16_T}, n);
        x::telem::Series times(x::telem::TIMESTAMP_T, n);
        const auto err = reader.read(this->brk, frame, times);
        EXPECT_FALSE(err) << err.message();
        return times;
    }
};

/// @brief it should decode 16-bit, 32-bit, 24-bit, and 4-bit input entries into samples
/// of their data types.
TEST_F(EngineTest, ReadDecodesEachEntryType) {
    this->master->set_input<int16_t>(I16_BYTE, 0x1234);
    this->master->set_input<int32_t>(I32_BYTE, static_cast<int32_t>(0xDEADBEEF));
    this->master->set_input<uint8_t>(I24_BYTE, 0xFE);
    this->master->set_input<uint8_t>(I24_BYTE + 1, 0xFF);
    this->master->set_input<uint8_t>(I24_BYTE + 2, 0xFF);
    this->master->set_input<uint8_t>(U4_BYTE, 0xAF);
    const auto reader = ASSERT_NIL_P(this->engine->open_reader(
        {IN_I16, IN_I32, IN_I24, IN_U4},
        x::telem::Rate(100),
        1
    ));
    auto frame = frame_of(
        {x::telem::INT16_T, x::telem::INT32_T, x::telem::INT32_T, x::telem::UINT8_T},
        1
    );
    x::telem::Series times(x::telem::TIMESTAMP_T, 1);
    ASSERT_NIL(reader->read(this->brk, frame, times));
    EXPECT_EQ(frame.series->at(0).at<int16_t>(0), 0x1234);
    EXPECT_EQ(frame.series->at(1).at<int32_t>(0), static_cast<int32_t>(0xDEADBEEF));
    EXPECT_EQ(frame.series->at(2).at<int32_t>(0), -2);
    EXPECT_EQ(frame.series->at(3).at<uint8_t>(0), 0x0F);
    EXPECT_EQ(times.size(), 1);
}

/// @brief it should read entries from more than one slave into one frame.
TEST_F(EngineTest, ReadCollectsEntriesFromSeveralSlaves) {
    this->master->add_slave(io_slave(1));
    this->master->set_input<int16_t>(I16_BYTE, 1);
    this->master->set_input<int16_t>(SLAVE_IMAGE_SIZE + I16_BYTE, 2);
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16, on_slave(IN_I16, 1)}, x::telem::Rate(100), 1)
    );
    auto frame = frame_of({x::telem::INT16_T, x::telem::INT16_T}, 1);
    x::telem::Series times(x::telem::TIMESTAMP_T, 1);
    ASSERT_NIL(reader->read(this->brk, frame, times));
    EXPECT_EQ(frame.series->at(0).at<int16_t>(0), 1);
    EXPECT_EQ(frame.series->at(1).at<int16_t>(0), 2);
}

/// @brief it should return one sample for each sample period, each with the time of its
/// cycle.
TEST_F(EngineTest, ReadReturnsOneSamplePerPeriodWithItsCycleTime) {
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 10)
    );
    const auto times = this->read_times(*reader, 10);
    ASSERT_EQ(times.size(), 10);
    for (size_t i = 1; i < times.size(); ++i)
        EXPECT_GT(times.at<int64_t>(i), times.at<int64_t>(i - 1));
    const auto span = x::telem::TimeSpan(times.at<int64_t>(9) - times.at<int64_t>(0));
    EXPECT_GE(span, x::telem::MILLISECOND * 45);
}

/// @brief it should stamp samples near the host time, spaced by the distributed clock.
TEST_F(EngineTest, ReadStampsSamplesWithTheDistributedClockSpacing) {
    this->master->set_dc_step(x::telem::MILLISECOND * 10);
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 10)
    );
    const auto times = this->read_times(*reader, 10);
    ASSERT_EQ(times.size(), 10);
    EXPECT_LT(
        x::telem::TimeStamp::now() - x::telem::TimeStamp(times.at<int64_t>(0)),
        x::telem::SECOND
    );
    for (size_t i = 1; i < times.size(); ++i) {
        const auto spacing = x::telem::TimeSpan(
            times.at<int64_t>(i) - times.at<int64_t>(i - 1)
        );
        EXPECT_GT(spacing, x::telem::MICROSECOND * 9'900);
        EXPECT_LT(spacing, x::telem::MICROSECOND * 10'100);
    }
}

TEST_F(EngineTest, ReadReturnsNilWhenBreakerStopped) {
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    this->brk.stop();
    auto frame = frame_of({x::telem::INT16_T}, 1);
    x::telem::Series times(x::telem::TIMESTAMP_T, 1);
    ASSERT_NIL(reader->read(this->brk, frame, times));
    EXPECT_EQ(frame.series->at(0).size(), 0);
}

/// @brief it should give a slower reader samples at its own rate while a faster reader
/// sets the bus rate.
TEST_F(EngineTest, SlowerReaderGetsSamplesAtItsOwnRate) {
    const auto fast = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(1000), 1)
    );
    const auto slow = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 5)
    );
    ASSERT_EQ(this->engine->cycle_rate(), x::telem::Rate(1000));
    const auto times = this->read_times(*slow, 5);
    ASSERT_EQ(times.size(), 5);
    const auto span = x::telem::TimeSpan(times.at<int64_t>(4) - times.at<int64_t>(0));
    EXPECT_GE(span, x::telem::MILLISECOND * 20);
}

/// @brief it should encode values into 16-bit, 32-bit, 24-bit, and 4-bit output
/// entries.
TEST_F(EngineTest, WriteEncodesEachEntryType) {
    const auto writer = ASSERT_NIL_P(this->engine->open_writer(
        {OUT_I16, OUT_I32, OUT_I24, OUT_U4},
        x::telem::Rate(100)
    ));
    {
        const auto tx = writer->open_tx();
        ASSERT_NIL(tx.write(0, 42.7f));
        ASSERT_NIL(tx.write(1, static_cast<int64_t>(0x12345678)));
        ASSERT_NIL(tx.write(2, static_cast<int32_t>(-1)));
        ASSERT_NIL(tx.write(3, static_cast<uint8_t>(0x0F)));
    }
    ASSERT_EVENTUALLY_EQ(this->master->get_output<int16_t>(I16_BYTE), 42);
    EXPECT_EQ(this->master->get_output<int32_t>(I32_BYTE), 0x12345678);
    EXPECT_EQ(this->master->get_output<uint8_t>(I24_BYTE), 0xFF);
    EXPECT_EQ(this->master->get_output<uint8_t>(I24_BYTE + 1), 0xFF);
    EXPECT_EQ(this->master->get_output<uint8_t>(I24_BYTE + 2), 0xFF);
    EXPECT_EQ(this->master->get_output<uint8_t>(U4_BYTE) & 0x0F, 0x0F);
}

/// @brief it should return a PDO mapping error when a write names an index past the
/// writer's entries.
TEST_F(EngineTest, WriteToUnknownIndexReturnsError) {
    const auto writer = ASSERT_NIL_P(
        this->engine->open_writer({OUT_I16}, x::telem::Rate(100))
    );
    const auto err = writer->write(1, static_cast<int16_t>(0x1234));
    ASSERT_OCCURRED_AS(err, errors::PDO_MAPPING_ERROR);
    EXPECT_EQ(err.data, "output index 1 is out of range for 1 registered entries");
}

/// @brief it should set the outputs of a closed writer to zero while the bus keeps
/// running.
TEST_F(EngineTest, ClosedWriterReturnsItsOutputsToZero) {
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    auto writer = ASSERT_NIL_P(
        this->engine->open_writer({OUT_I16}, x::telem::Rate(100))
    );
    ASSERT_NIL(writer->write(0, static_cast<int16_t>(0x1234)));
    ASSERT_EVENTUALLY_EQ(this->master->get_output<int16_t>(I16_BYTE), 0x1234);
    writer.reset();
    ASSERT_TRUE(this->engine->running());
    ASSERT_EVENTUALLY_EQ(this->master->get_output<int16_t>(I16_BYTE), 0);
}

/// @brief it should keep the latest command on an output that two writers share when a
/// third writer closes.
TEST_F(EngineTest, ClosingAWriterKeepsTheLatestCommandOnASharedOutput) {
    auto first = ASSERT_NIL_P(
        this->engine->open_writer({OUT_I16}, x::telem::Rate(100))
    );
    auto second = ASSERT_NIL_P(
        this->engine->open_writer({OUT_I16}, x::telem::Rate(100))
    );
    auto third = ASSERT_NIL_P(
        this->engine->open_writer({OUT_I32}, x::telem::Rate(100))
    );
    ASSERT_NIL(second->write(0, static_cast<int16_t>(10)));
    ASSERT_NIL(first->write(0, static_cast<int16_t>(20)));
    ASSERT_EVENTUALLY_EQ(this->master->get_output<int16_t>(I16_BYTE), 20);

    third.reset();
    // The second sample comes from a cycle after the first one wrote its outputs.
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    for (int i = 0; i < 2; ++i) {
        auto frame = frame_of({x::telem::INT16_T}, 1);
        x::telem::Series times(x::telem::TIMESTAMP_T, 1);
        ASSERT_NIL(reader->read(this->brk, frame, times));
    }
    EXPECT_EQ(this->master->get_output<int16_t>(I16_BYTE), 20);
}

/// @brief it should send zero outputs before the bus stops when the last writer closes.
TEST_F(EngineTest, LastWriterToCloseReturnsItsOutputsToZero) {
    auto writer = ASSERT_NIL_P(
        this->engine->open_writer({OUT_I16}, x::telem::Rate(100))
    );
    ASSERT_NIL(writer->write(0, static_cast<int16_t>(0x1234)));
    ASSERT_EVENTUALLY_EQ(this->master->get_output<int16_t>(I16_BYTE), 0x1234);
    writer.reset();
    ASSERT_FALSE(this->engine->running());
    EXPECT_EQ(this->master->get_output<int16_t>(I16_BYTE), 0);
}

/// @brief it should keep the sample spacing of a reader while writers open and close
/// and change the bus rate.
TEST_F(EngineTest, ReaderKeepsItsSpacingWhileTheBusRateChanges) {
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(20), 20)
    );
    std::atomic done = false;
    std::thread churn([&] {
        while (!done.load()) {
            {
                const auto writer = this->engine->open_writer(
                    {OUT_I16},
                    x::telem::Rate(1000)
                );
                std::this_thread::sleep_for(std::chrono::milliseconds(30));
            }
            std::this_thread::sleep_for(std::chrono::milliseconds(30));
        }
    });
    const auto times = this->read_times(*reader, 20);
    done.store(true);
    churn.join();
    ASSERT_EQ(times.size(), 20);
    for (size_t i = 1; i < times.size(); ++i)
        EXPECT_GE(
            x::telem::TimeSpan(times.at<int64_t>(i) - times.at<int64_t>(i - 1)),
            x::telem::MILLISECOND * 25
        );
}

/// @brief it should reject a reader with a sample rate below 0.001 Hz and not start the
/// bus.
TEST_F(EngineTest, ReadRateBelowOneMillihertzIsRejected) {
    ASSERT_OCCURRED_AS_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(0.0001), 1),
        errors::RATE_MISMATCH
    );
    EXPECT_FALSE(this->engine->running());
}

/// @brief it should reject a reader with a sample rate between two 0.001 Hz steps,
/// instead of rounding its rate.
TEST_F(EngineTest, ReadRateBetweenMillihertzStepsIsRejected) {
    ASSERT_OCCURRED_AS_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(0.0014), 1),
        errors::RATE_MISMATCH
    );
    EXPECT_FALSE(this->engine->running());
}

/// @brief it should accept a reader rate whose float holds a rounding error below one
/// part per million, such as 0.251 Hz.
TEST_F(EngineTest, ReadRateWithFloatRoundingErrorOpens) {
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(0.251), 1)
    );
    EXPECT_EQ(this->engine->cycle_rate(), x::telem::Rate(100.149));
}

/// @brief it should open a task without a restart of the bus or a change to the outputs
/// of other tasks.
TEST_F(EngineTest, OpeningATaskKeepsTheBusRunning) {
    const auto writer = ASSERT_NIL_P(
        this->engine->open_writer({OUT_I16}, x::telem::Rate(100))
    );
    ASSERT_NIL(writer->write(0, static_cast<int16_t>(0x1234)));
    ASSERT_EVENTUALLY_EQ(this->master->get_output<int16_t>(I16_BYTE), 0x1234);
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    EXPECT_EQ(this->master->call_count("activate"), 1);
    EXPECT_EQ(this->master->call_count("deactivate"), 0);
    EXPECT_EQ(this->master->get_output<int16_t>(I16_BYTE), 0x1234);
    EXPECT_EQ(this->read_times(*reader, 1).size(), 1);
}

/// @brief it should close a task without a restart of the bus or a gap in the samples
/// of other tasks.
TEST_F(EngineTest, ClosingATaskKeepsTheBusRunning) {
    const auto kept = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    auto closed = ASSERT_NIL_P(
        this->engine->open_reader({IN_I32}, x::telem::Rate(100), 1)
    );
    closed.reset();
    EXPECT_TRUE(this->engine->running());
    EXPECT_EQ(this->master->call_count("activate"), 1);
    EXPECT_EQ(this->master->call_count("deactivate"), 0);
    EXPECT_EQ(this->read_times(*kept, 1).size(), 1);
}

/// @brief it should stop the bus and deactivate the master when the last task closes.
TEST_F(EngineTest, BusStopsWhenTheLastTaskCloses) {
    auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    reader.reset();
    EXPECT_FALSE(this->engine->running());
    EXPECT_EQ(this->engine->cycle_rate(), x::telem::Rate(0));
    EXPECT_EQ(this->master->call_count("deactivate"), 1);
}

/// @brief it should reject an entry that is not in the process image and not start the
/// bus.
TEST_F(EngineTest, UnmappedEntryIsRejected) {
    const auto res = this->engine->open_reader(
        {pdo::Entry(0, 0x9999, 1, 16, true, x::telem::INT16_T)},
        x::telem::Rate(100),
        1
    );
    ASSERT_OCCURRED_AS(res.second, errors::PDO_MAPPING_ERROR);
    EXPECT_EQ(res.second.data, "PDO 0x9999:01 is not in the process image of slave 0");
    EXPECT_FALSE(this->engine->running());
}

/// @brief it should reject an entry that is longer than the PDO the slave maps.
TEST_F(EngineTest, EntryLongerThanItsPDOIsRejected) {
    const auto res = this->engine->open_reader(
        {pdo::Entry(0, 0x6000, 4, 32, true, x::telem::UINT32_T)},
        x::telem::Rate(100),
        1
    );
    ASSERT_OCCURRED_AS(res.second, errors::PDO_MAPPING_ERROR);
    EXPECT_EQ(
        res.second.data,
        "PDO 0x6000:04 on slave 0 is 4 bits, not 32. Scan the network again"
    );
    EXPECT_FALSE(this->engine->running());
}

/// @brief it should keep the bus running when it rejects the entry of a new task.
TEST_F(EngineTest, RejectedEntryLeavesTheRunningBusUp) {
    const auto writer = ASSERT_NIL_P(
        this->engine->open_writer({OUT_I16}, x::telem::Rate(100))
    );
    ASSERT_OCCURRED_AS_P(
        this->engine->open_writer(
            {pdo::Entry(0, 0x9999, 1, 16, false, x::telem::INT16_T)},
            x::telem::Rate(100)
        ),
        errors::PDO_MAPPING_ERROR
    );
    EXPECT_TRUE(this->engine->running());
    EXPECT_EQ(this->master->call_count("activate"), 1);
}

/// @brief it should reject an entry on a disabled slave, because the slave is not in
/// the process image.
TEST_F(EngineTest, DisabledSlaveIsNotMapped) {
    this->master->add_slave(io_slave(1));
    this->engine->set_slave_enabled(1, false);
    ASSERT_OCCURRED_AS_P(
        this->engine->open_reader({on_slave(IN_I16, 1)}, x::telem::Rate(100), 1),
        errors::PDO_MAPPING_ERROR
    );
}

/// @brief it should return the error that kept the master from initializing once the
/// retries run out.
TEST_F(EngineTest, InitializeErrorPropagates) {
    this->engine = std::make_shared<Engine>(this->master, quick_start_config());
    this->master->inject_init_error(
        x::errors::Error(errors::MASTER_INIT_ERROR, "init failed")
    );
    const auto res = this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1);
    ASSERT_OCCURRED_AS(res.second, errors::MASTER_INIT_ERROR);
    EXPECT_EQ(res.second.data, "init failed");
    EXPECT_EQ(this->master->init_call_count(), 3);
    EXPECT_FALSE(this->engine->running());
}

/// @brief it should deactivate the master after each failed activation, and return the
/// activation error once the retries run out.
TEST_F(EngineTest, ActivationErrorPropagates) {
    this->engine = std::make_shared<Engine>(this->master, quick_start_config());
    this->master->inject_activate_error(
        x::errors::Error(errors::ACTIVATION_ERROR, "activate failed")
    );
    const auto res = this->engine->open_writer({OUT_I16}, x::telem::Rate(100));
    ASSERT_OCCURRED_AS(res.second, errors::ACTIVATION_ERROR);
    EXPECT_EQ(res.second.data, "activate failed");
    EXPECT_EQ(this->master->call_count("activate"), 3);
    EXPECT_EQ(this->master->call_count("deactivate"), 3);
    EXPECT_FALSE(this->engine->running());
}

/// @brief it should retry activation of the master until it succeeds, then open the
/// task.
TEST_F(EngineTest, StartRetriesUntilActivationSucceeds) {
    this->master->inject_activate_error(
        x::errors::Error(errors::ACTIVATION_ERROR, "activate failed")
    );
    std::thread recover([this] {
        while (this->master->call_count("activate") < 2)
            std::this_thread::sleep_for(std::chrono::milliseconds(1));
        this->master->clear_injected_errors();
    });
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    recover.join();
    EXPECT_GE(this->master->call_count("activate"), 2);
    EXPECT_TRUE(this->engine->running());
}

/// @brief it should return a temporary error while bus cycles fail, and return samples
/// again when the cycles recover.
TEST_F(EngineTest, FailedCyclesReturnATemporaryError) {
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    auto frame = frame_of({x::telem::INT16_T}, 1);
    x::telem::Series times(x::telem::TIMESTAMP_T, 1);
    this->master->inject_receive_error(
        x::errors::Error(errors::WORKING_COUNTER_ERROR, "expected 3, got 1")
    );
    ASSERT_EVENTUALLY_TRUE(
        reader->read(this->brk, frame, times).matches(errors::CYCLE_FAILED)
    );
    const auto err = reader->read(this->brk, frame, times);
    ASSERT_OCCURRED_AS(err, errors::CYCLE_FAILED);
    ASSERT_TRUE(err.matches(driver::errors::TEMPORARY_HARDWARE_ERROR));
    EXPECT_EQ(err.data, "every bus cycle failed: expected 3, got 1");
    this->master->clear_injected_errors();
    ASSERT_EVENTUALLY_NIL(reader->read(this->brk, frame, times));
}

/// @brief it should return an error when the bus delivers no samples within twice the
/// duration of a read, and return samples again once the bus resumes.
TEST_F(EngineTest, ReadTimesOutWhenTheBusStalls) {
    this->master->stall();
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    auto frame = frame_of({x::telem::INT16_T}, 1);
    x::telem::Series times(x::telem::TIMESTAMP_T, 1);
    const auto err = reader->read(this->brk, frame, times);
    this->master->resume();
    ASSERT_OCCURRED_AS(err, errors::CYCLE_OVERRUN);
    EXPECT_EQ(err.data, "no inputs from eth0 within 220ms");
    EXPECT_EQ(this->read_times(*reader, 1).size(), 1);
}

/// @brief it should report the samples that failed cycles dropped as a warning.
TEST_F(EngineTest, DroppedSamplesFromFailedCyclesAreAWarning) {
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    const auto receives = this->master->call_count("receive");
    this->master->inject_receive_error(
        x::errors::Error(errors::WORKING_COUNTER_ERROR, "expected 3, got 1")
    );
    ASSERT_EVENTUALLY_GE(this->master->call_count("receive"), receives + 3);
    this->master->clear_injected_errors();
    EXPECT_EQ(this->read_times(*reader, 1).size(), 1);
    EXPECT_EQ(reader->warning(), "Bus cycles failed: expected 3, got 1.");
    EXPECT_EQ(reader->warning(), "");
}

/// @brief it should report the samples a reader dropped because it fell behind the bus
/// as a warning.
TEST_F(EngineTest, ReaderThatFallsBehindGetsAWarning) {
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    std::string warning;
    ASSERT_EVENTUALLY_TRUE_WITH_TIMEOUT(
        !(warning = reader->warning()).empty(),
        std::chrono::seconds(6),
        std::chrono::milliseconds(100)
    );
    EXPECT_EQ(
        warning,
        "The task cannot keep up with the bus. Lower the stream rate for the task."
    );
    EXPECT_EQ(this->read_times(*reader, 1).size(), 1);
}

/// @brief it should run the bus at the minimum rate of 100 Hz when every task is
/// slower.
TEST_F(EngineTest, SlowTasksRunTheBusAtTheMinimumRate) {
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(10), 1)
    );
    EXPECT_EQ(this->engine->cycle_rate(), x::telem::Rate(100));
}

/// @brief it should open a 0.1 Hz reader on a 100 Hz bus, despite the float precision
/// of a rate.
TEST_F(EngineTest, TenthHzReaderOpens) {
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(0.1), 1)
    );
    EXPECT_EQ(this->engine->cycle_rate(), x::telem::Rate(100));
}

/// @brief it should run the bus at the least common multiple of the read rates.
TEST_F(EngineTest, BusRateIsTheCommonMultipleOfReadRates) {
    const auto a = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(300), 1)
    );
    const auto b = ASSERT_NIL_P(
        this->engine->open_reader({IN_I32}, x::telem::Rate(200), 1)
    );
    EXPECT_EQ(this->engine->cycle_rate(), x::telem::Rate(600));
}

/// @brief it should run the bus at the smallest multiple of the read rates that is at
/// least the rate of the fastest writer.
TEST_F(EngineTest, BusRateCoversTheFastestWriter) {
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    const auto writer = ASSERT_NIL_P(
        this->engine->open_writer({OUT_I16}, x::telem::Rate(250))
    );
    EXPECT_EQ(this->engine->cycle_rate(), x::telem::Rate(300));
}

/// @brief it should reject a reader when no bus rate up to the maximum is a multiple of
/// every read rate.
TEST_F(EngineTest, ReadRatesWithoutASmallCommonMultipleAreRejected) {
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(997), 1)
    );
    const auto res = this->engine->open_reader({IN_I32}, x::telem::Rate(300), 1);
    ASSERT_OCCURRED_AS(res.second, errors::RATE_MISMATCH);
    EXPECT_EQ(
        res.second.data,
        "no bus rate at or below 10000 Hz on eth0 is a multiple of every read task "
        "sample rate (997 Hz, 300 Hz). Choose sample rates with a smaller common "
        "multiple."
    );
    EXPECT_EQ(this->engine->cycle_rate(), x::telem::Rate(997));
}

/// @brief it should reject a writer faster than the maximum bus rate and not start the
/// bus.
TEST_F(EngineTest, WriterAboveTheMaximumRateIsRejected) {
    const auto res = this->engine->open_writer({OUT_I16}, x::telem::Rate(20000));
    ASSERT_OCCURRED_AS(res.second, errors::RATE_MISMATCH);
    EXPECT_EQ(
        res.second.data,
        "a write task on eth0 runs at 20000 Hz, above the maximum bus rate of 10000 Hz"
    );
    EXPECT_FALSE(this->engine->running());
}

/// @brief it should slow the bus down when the fastest task closes, and keep the sample
/// spacing of the open readers.
TEST_F(EngineTest, BusSlowsDownWhenAFasterTaskCloses) {
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 5)
    );
    auto writer = ASSERT_NIL_P(
        this->engine->open_writer({OUT_I16}, x::telem::Rate(1000))
    );
    ASSERT_EQ(this->engine->cycle_rate(), x::telem::Rate(1000));
    writer.reset();
    EXPECT_EQ(this->engine->cycle_rate(), x::telem::Rate(100));
    const auto times = this->read_times(*reader, 5);
    ASSERT_EQ(times.size(), 5);
    const auto span = x::telem::TimeSpan(times.at<int64_t>(4) - times.at<int64_t>(0));
    EXPECT_GE(span, x::telem::MILLISECOND * 20);
}

/// @brief it should report the health of each enabled slave and leave out disabled
/// slaves.
TEST_F(EngineTest, HealthReportsEachEnabledSlave) {
    this->master->add_slave(io_slave(1));
    this->engine->set_slave_enabled(1, false);
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    ASSERT_EVENTUALLY_EQ(this->engine->health().size(), 1);
    EXPECT_EQ(
        this->engine->health()[0],
        (slave::Health{.position = 0, .state = slave::State::OP})
    );
}

/// @brief it should report a faulted slave in its health, then return the slave to OP
/// while the bus keeps running.
TEST_F(EngineTest, FaultedSlaveReturnsToOpWhileTheBusRuns) {
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    this->master->set_slave_transition_failure(0, slave::State::OP);
    this->master->fault_slave(0, slave::State::SAFE_OP, "Sync manager watchdog");
    const slave::Health faulted{
        .position = 0,
        .state = slave::State::SAFE_OP,
        .faulted = true,
        .al_status = "Sync manager watchdog",
    };
    ASSERT_EVENTUALLY_TRUE(
        !this->engine->health().empty() && this->engine->health()[0] == faulted
    );
    EXPECT_TRUE(this->engine->running());
    EXPECT_EQ(this->read_times(*reader, 1).size(), 1);
    this->master->clear_transition_failures();
    ASSERT_EVENTUALLY_TRUE(
        !this->engine->health().empty() && this->engine->health()[0].operational()
    );
    EXPECT_EQ(this->master->call_count("activate"), 1);
}

/// @brief it should report no health when the bus stops.
TEST_F(EngineTest, HealthIsEmptyWhenTheBusStops) {
    auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    ASSERT_EVENTUALLY_EQ(this->engine->health().size(), 1);
    reader.reset();
    EXPECT_TRUE(this->engine->health().empty());
}

/// @brief it should keep the outputs and samples of running tasks correct while other
/// tasks open and close.
TEST_F(EngineTest, TasksOpenAndCloseWithoutDisturbingRunningOnes) {
    const auto writer = ASSERT_NIL_P(
        this->engine->open_writer({OUT_I16}, x::telem::Rate(100))
    );
    ASSERT_NIL(writer->write(0, static_cast<int16_t>(0x1234)));
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I32}, x::telem::Rate(100), 1)
    );
    ASSERT_EVENTUALLY_EQ(this->master->get_output<int16_t>(I16_BYTE), 0x1234);

    std::atomic<bool> done{false};
    std::thread producer([&] {
        for (int32_t i = 1; !done.load(); ++i) {
            this->master->set_input<int32_t>(I32_BYTE, i);
            std::this_thread::sleep_for(std::chrono::microseconds(500));
        }
    });
    std::atomic<int> read_errors{0};
    std::atomic<int> decreases{0};
    std::atomic<int> reads{0};
    std::thread consumer([&] {
        int32_t last = 0;
        auto frame = frame_of({x::telem::INT32_T}, 1);
        x::telem::Series times(x::telem::TIMESTAMP_T, 1);
        while (!done.load()) {
            frame.series->at(0).clear();
            times.clear();
            if (reader->read(this->brk, frame, times)) {
                read_errors++;
                continue;
            }
            const auto value = frame.series->at(0).at<int32_t>(0);
            if (value < last) decreases++;
            last = value;
            reads++;
        }
    });

    int output_changes = 0;
    for (int i = 0; i < 50; ++i) {
        const auto churn_reader = ASSERT_NIL_P(this->engine->open_reader(
            {IN_I16, IN_I24},
            x::telem::Rate(i % 2 == 0 ? 200 : 50),
            1
        ));
        const auto churn_writer = ASSERT_NIL_P(
            this->engine->open_writer({OUT_I32}, x::telem::Rate(400))
        );
        ASSERT_NIL(churn_writer->write(0, static_cast<int32_t>(i)));
        if (this->master->get_output<int16_t>(I16_BYTE) != 0x1234) output_changes++;
        std::this_thread::sleep_for(std::chrono::milliseconds(5));
    }
    done.store(true);
    producer.join();
    consumer.join();

    EXPECT_EQ(read_errors.load(), 0);
    EXPECT_EQ(decreases.load(), 0);
    EXPECT_GT(reads.load(), 10);
    EXPECT_EQ(output_changes, 0);
    EXPECT_EQ(this->master->call_count("activate"), 1);
    EXPECT_EQ(this->engine->cycle_rate(), x::telem::Rate(100));
}

TEST_F(EngineTest, EnsureInitializedIdempotent) {
    ASSERT_NIL(this->engine->ensure_initialized());
    ASSERT_NIL(this->engine->ensure_initialized());
}

/// @brief it should keep the discovered slaves and not reinitialize the master when the
/// slave count is unchanged.
TEST_F(EngineTest, EnsureInitializedKeepsDiscoveryWhenTheBusIsUnchanged) {
    ASSERT_NIL(this->engine->ensure_initialized());
    ASSERT_NIL(this->engine->ensure_initialized());
    const auto calls = this->master->call_log();
    EXPECT_EQ(std::ranges::count(calls, "deactivate"), 0);
}

/// @brief it should reinitialize the master and discover again when the slave count
/// changes.
TEST_F(EngineTest, EnsureInitializedRediscoversWhenASlaveIsPluggedIn) {
    ASSERT_NIL(this->engine->ensure_initialized());
    this->master->set_bus_slaves(2);
    ASSERT_NIL(this->engine->ensure_initialized());
    EXPECT_EQ(
        this->master->call_log(),
        (std::vector<std::string>{
            "initialize",
            "count_slaves",
            "initialize",
            "count_slaves",
            "deactivate",
            "initialize",
        })
    );
}

/// @brief it should not count or discover slaves while the bus runs.
TEST_F(EngineTest, EnsureInitializedDoesNotRediscoverWhileTheBusRuns) {
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    this->master->set_bus_slaves(2);
    this->master->clear_call_log();
    ASSERT_NIL(this->engine->ensure_initialized());
    const auto calls = this->master->call_log();
    EXPECT_EQ(std::ranges::count(calls, "count_slaves"), 0);
}

TEST_F(EngineTest, SlavesReturnsDiscoveredSlaves) {
    this->master->add_slave(io_slave(1));
    ASSERT_NIL(this->engine->ensure_initialized());
    const auto slaves = this->engine->slaves();
    ASSERT_EQ(slaves.size(), 2);
    EXPECT_EQ(slaves[0].properties.position, 0);
    EXPECT_EQ(slaves[1].properties.position, 1);
}

TEST_F(EngineTest, InterfaceNameReturnsCorrect) {
    EXPECT_EQ(this->engine->interface_name(), "eth0");
}

TEST(PoolTest, DiscoverSlavesCreatesEngine) {
    auto master = std::make_shared<testutil::Master>("eth0");
    master->add_slave(io_slave(0));
    master->add_slave(io_slave(1));
    auto manager = std::make_unique<testutil::Manager>();
    manager->configure("eth0", master);
    Pool pool(std::move(manager));
    const auto slaves = ASSERT_NIL_P(pool.discover_slaves("eth0"));
    ASSERT_EQ(slaves.size(), 2);
    EXPECT_EQ(slaves[0].properties.position, 0);
    EXPECT_EQ(slaves[1].properties.position, 1);
}

TEST(PoolTest, DiscoverSlavesReturnsFromRunningEngine) {
    auto master = std::make_shared<testutil::Master>("eth0");
    master->add_slave(io_slave(0));
    master->add_slave(io_slave(1));
    auto manager = std::make_unique<testutil::Manager>();
    manager->configure("eth0", master);
    Pool pool(std::move(manager));
    const auto engine = ASSERT_NIL_P(pool.acquire("eth0"));
    const auto reader = ASSERT_NIL_P(
        engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    EXPECT_TRUE(pool.is_active("eth0"));
    const auto slaves = ASSERT_NIL_P(pool.discover_slaves("eth0"));
    ASSERT_EQ(slaves.size(), 2);
    EXPECT_EQ(slaves[0].properties.position, 0);
    EXPECT_EQ(slaves[1].properties.position, 1);
}

TEST(PoolTest, DiscoverSlavesInitErrorNotCached) {
    auto master = std::make_shared<testutil::Master>("eth0");
    master->inject_init_error(
        x::errors::Error(errors::MASTER_INIT_ERROR, "no interface")
    );
    auto manager = std::make_unique<testutil::Manager>();
    manager->configure("eth0", master);
    Pool pool(std::move(manager));
    const auto slaves = ASSERT_OCCURRED_AS_P(
        pool.discover_slaves("eth0"),
        errors::MASTER_INIT_ERROR
    );
    EXPECT_TRUE(slaves.empty());
    EXPECT_FALSE(pool.is_active("eth0"));
}
}
