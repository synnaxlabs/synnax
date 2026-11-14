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
#include <future>
#include <map>
#include <memory>
#include <optional>
#include <string>
#include <thread>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/defer/defer.h"
#include "x/cpp/loop/testutil/testutil.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/thread/rt/rt.h"

#include "driver/ethercat/coe/coe.h"
#include "driver/ethercat/engine/engine.h"
#include "driver/ethercat/engine/pool.h"
#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/testutil/master.h"

namespace driver::ethercat::engine {
namespace {
/// @brief the DC registers of a slave whose SYNC0 cycle is the 10 ms bus period.
const dc::Registers SYNC0_AT_10MS{.assign_activate = 0x0300, .sync0_cycle = 10'000'000};

/// @brief the SOEM interface of a Windows adapter.
const std::string
    NPF_INTERFACE = "\\Device\\NPF_{126D8109-3F4B-4C1A-9E6D-2B7A0C5D8E11}";

/// @brief a loader for a network with no saved slave settings.
const SlaveConfigs NO_SLAVE_CONFIGS = [](const std::string &, const std::string &) {
    return std::pair{std::map<uint16_t, slave::Config>{}, x::errors::NIL};
};
const pdo::Entry IN_I16(0, 0x6000, 1, 16, true, x::telem::INT16_T);
const pdo::Entry IN_I32(0, 0x6000, 2, 32, true, x::telem::INT32_T);
const pdo::Entry IN_I24(0, 0x6000, 3, 24, true, x::telem::INT32_T);
const pdo::Entry IN_U4(0, 0x6000, 4, 4, true, x::telem::UINT8_T);
const pdo::Entry OUT_I16(0, 0x7000, 1, 16, false, x::telem::INT16_T);
const pdo::Entry OUT_I32(0, 0x7000, 2, 32, false, x::telem::INT32_T);
const pdo::Entry OUT_I24(0, 0x7000, 3, 24, false, x::telem::INT32_T);
const pdo::Entry OUT_U4(0, 0x7000, 4, 4, false, x::telem::UINT8_T);

/// @brief why a slave in the specs refuses SAFE-OP.
const slave::Refusal REFUSAL{.reason = "refused SAFE-OP: Inconsistent settings"};
/// @brief a refusal with slave::DC_SYNC_ADVICE.
const slave::Refusal DC_SYNC_REFUSAL{
    .reason = "refused SAFE-OP: Invalid DC SYNC configuration",
    .advice = slave::DC_SYNC_ADVICE,
};
/// @brief a refusal of a slave that kept the layout of its earlier mode.
const slave::Refusal STALE_LAYOUT_REFUSAL{
    .reason = "refused SAFE-OP: Invalid input configuration",
    .advice = slave::MODE_RESTART_ADVICE,
    .code = 0x001E,
};

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

/// @brief an engine configuration whose slave settings disable the slave at position.
Config disabling(const uint16_t position) {
    Config cfg;
    cfg.slave_configs = [position] {
        return std::pair{
            std::map<uint16_t, slave::Config>{{position, {.enabled = false}}},
            x::errors::NIL
        };
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
        this->master = std::make_shared<testutil::Master>();
        this->master->add_slave(io_slave(0));
        this->engine = std::make_shared<Engine>("eth0", this->master);
        this->brk.start();
    }

    void TearDown() override { this->brk.stop(); }

    /// @brief reads n samples of a single 16-bit entry and returns their times.
    x::telem::Series read_times(Engine::Reader &reader, const size_t n) {
        auto frame = frame_of({x::telem::INT16_T}, n);
        x::telem::Series times(x::telem::TIMESTAMP_T, n);
        const auto err = reader.read(this->brk, frame, times);
        EXPECT_NIL(err);
        return times;
    }
};

/// @brief runs the engine on a simulated clock, so the bus cycles as fast as the host
/// allows and each frame passes the reference clock at its exact simulated time.
class EngineLockTest : public EngineTest {
protected:
    /// @brief the frames a spec watches: 3 s at the 10 ms bus period, in which a
    /// distributed clock that runs 0.5% fast slips through one and a half periods.
    static constexpr size_t FRAMES = 300;
    const x::telem::TimeSpan period = x::telem::MILLISECOND * 10;
    const x::loop::HostClock clock = x::loop::fake_clock();

    void SetUp() override {
        this->master = std::make_shared<testutil::Master>(this->clock);
        this->master->add_slave(io_slave(0));
        Config cfg;
        cfg.host_clock = this->clock;
        this->engine = std::make_shared<Engine>("eth0", this->master, cfg);
        this->brk.start();
    }

    /// @brief returns the share of the recorded frames that passed the reference clock
    /// within a quarter period of half a period after a SYNC0 event, with SYNC0 events
    /// at shift after multiples of the period.
    [[nodiscard]] double
    share_in_phase(const x::telem::TimeSpan shift = x::telem::TimeSpan(0)) const {
        const auto frames = this->master->recorded_dc();
        EXPECT_EQ(frames.size(), FRAMES);
        int in_phase = 0;
        for (const auto &dc: frames) {
            const auto phase = ((dc - master::DC_EPOCH - shift) % this->period +
                                this->period) %
                               this->period;
            if ((phase - this->period / 2).abs() < this->period / 4) ++in_phase;
        }
        return static_cast<double>(in_phase) / static_cast<double>(frames.size());
    }
};

/// @brief it should hold an RT core while the bus runs and give it back when the bus
/// stops.
TEST_F(EngineTest, HoldsAnRTCoreOnlyWhileTheBusRuns) {
    const auto rt = std::make_shared<x::thread::rt::Manager>();
    if (rt->total_cores() == 0) GTEST_SKIP() << "the host has no RT cores";
    Config cfg;
    cfg.rt_manager = rt;
    const auto engine = std::make_shared<Engine>("eth0", this->master, cfg);
    EXPECT_EQ(rt->available_cores(), rt->total_cores());
    auto reader = ASSERT_NIL_P(engine->open_reader({IN_I16}, x::telem::Rate(100), 1));
    ASSERT_EVENTUALLY_EQ(rt->available_cores(), rt->total_cores() - 1);
    reader.reset();
    EXPECT_EQ(rt->available_cores(), rt->total_cores());
}

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

/// @brief it should hold each frame half a period after a SYNC0 event, with the SYNC0
/// shift, while the distributed clock runs 0.5% faster than the host clock.
TEST_F(EngineLockTest, LocksTheCycleToTheDistributedClock) {
    auto sync = SYNC0_AT_10MS;
    sync.sync0_shift = 5'000'000;
    this->master->set_dc_sync({sync});
    this->master->set_host_dc(1.005);
    this->master->record_dc(FRAMES);
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    EXPECT_GT(this->share_in_phase(x::telem::MILLISECOND * 5), 0.8);
}

/// @brief it should lock the cycle to SYNC0 events of 100 us or more, and not to a
/// faster SYNC0 that clocks hardware in a slave.
TEST_F(EngineLockTest, LocksTheCycleOnlyToSYNC0EventsThatMarkData) {
    this->master->set_dc_sync(
        {SYNC0_AT_10MS, {.assign_activate = 0x0300, .sync0_cycle = 1'000}}
    );
    this->master->set_host_dc(1.005);
    this->master->record_dc(FRAMES);
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    EXPECT_GT(this->share_in_phase(), 0.8);
}

/// @brief it should leave the cycle unlocked when no slave runs SYNC0.
TEST_F(EngineLockTest, LeavesTheCycleUnlockedWithoutSYNC0) {
    this->master->set_host_dc(1.005);
    this->master->record_dc(FRAMES);
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    EXPECT_LT(this->share_in_phase(), 0.8);
}

/// @brief it should lock the cycle to the distributed clock while the slaves move to OP
/// and the working counter falls short.
TEST_F(EngineLockTest, LocksTheCycleWhileTheSlavesMoveToOP) {
    this->master->set_dc_sync({SYNC0_AT_10MS});
    this->master->set_host_dc(1.005);
    // The slaves reach OP after 400 frames, so every recorded frame comes before OP.
    this->master->set_start_frames(400);
    this->master->record_dc(FRAMES);
    std::pair<std::unique_ptr<Engine::Reader>, x::errors::Error> res;
    std::thread start([&] {
        res = this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1);
    });
    EXPECT_GT(this->share_in_phase(), 0.8);
    start.join();
    ASSERT_NIL(res.second);
}

/// @brief it should not move the cycle on a frame with a short working counter after
/// the start, as the frame can miss the reference clock.
TEST_F(EngineLockTest, ShortFramesAfterTheStartDoNotMoveTheCycle) {
    this->master->set_dc_sync({SYNC0_AT_10MS});
    this->master->set_host_dc(1.005);
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    this->master->inject_receive_error(
        x::errors::Error(errors::WORKING_COUNTER_ERROR, "expected 3, got 1")
    );
    this->master->record_dc(FRAMES);
    EXPECT_LT(this->share_in_phase(), 0.8);
}

/// @brief it should return no samples and no error when the breaker stops.
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
        {{OUT_I16, 0}, {OUT_I32, 0}, {OUT_I24, 0}, {OUT_U4, 0}},
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
        this->engine->open_writer({{OUT_I16, 0}}, x::telem::Rate(100))
    );
    const auto err = writer->open_tx().write(1, static_cast<int16_t>(0x1234));
    ASSERT_OCCURRED_AS(err, errors::PDO_MAPPING_ERROR);
    EXPECT_EQ(err.data, "output index 1 is out of range for 1 registered entries");
}

/// @brief it should set the outputs of a closed writer to its stop values and hold them
/// while the bus keeps running.
TEST_F(EngineTest, ClosedWriterHoldsItsStopValuesWhileTheBusRuns) {
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 3)
    );
    auto writer = ASSERT_NIL_P(this->engine->open_writer(
        {{OUT_I16, static_cast<int16_t>(-7)}, {OUT_U4, static_cast<uint8_t>(0x5)}},
        x::telem::Rate(100)
    ));
    ASSERT_NIL(writer->open_tx().write(0, static_cast<int16_t>(0x1234)));
    ASSERT_EVENTUALLY_EQ(this->master->get_output<int16_t>(I16_BYTE), 0x1234);
    writer.reset();
    ASSERT_EVENTUALLY_EQ(this->master->get_output<int16_t>(I16_BYTE), -7);
    EXPECT_EQ(this->master->get_output<uint8_t>(U4_BYTE) & 0x0F, 0x5);
    EXPECT_EQ(this->read_times(*reader, 3).size(), 3);
    EXPECT_EQ(this->master->get_output<int16_t>(I16_BYTE), -7);
}

/// @brief it should keep the latest command on an output that two writers share when a
/// third writer closes.
TEST_F(EngineTest, ClosingAWriterKeepsTheLatestCommandOnASharedOutput) {
    auto first = ASSERT_NIL_P(
        this->engine->open_writer({{OUT_I16, 0}}, x::telem::Rate(100))
    );
    auto second = ASSERT_NIL_P(
        this->engine->open_writer({{OUT_I16, 0}}, x::telem::Rate(100))
    );
    auto third = ASSERT_NIL_P(
        this->engine->open_writer({{OUT_I32, 0}}, x::telem::Rate(100))
    );
    ASSERT_NIL(second->open_tx().write(0, static_cast<int16_t>(10)));
    ASSERT_NIL(first->open_tx().write(0, static_cast<int16_t>(20)));
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

/// @brief it should set a shared output to the stop value of a writer that closes after
/// another writer commanded the output.
TEST_F(EngineTest, StopValueOutranksAnEarlierCommandOnASharedOutput) {
    auto first = ASSERT_NIL_P(
        this->engine->open_writer({{OUT_I16, 0}}, x::telem::Rate(100))
    );
    auto second = ASSERT_NIL_P(this->engine->open_writer(
        {{OUT_I16, static_cast<int16_t>(-7)}},
        x::telem::Rate(100)
    ));
    ASSERT_NIL(first->open_tx().write(0, static_cast<int16_t>(20)));
    ASSERT_NIL(second->open_tx().write(0, static_cast<int16_t>(10)));
    ASSERT_EVENTUALLY_EQ(this->master->get_output<int16_t>(I16_BYTE), 10);
    second.reset();
    ASSERT_EVENTUALLY_EQ(this->master->get_output<int16_t>(I16_BYTE), -7);
}

/// @brief it should keep the bits of another writer's entry when a writer closes on the
/// other half of the same byte.
TEST_F(EngineTest, ClosingAWriterKeepsTheOtherBitsOfItsByte) {
    const pdo::Entry low(1, 0x7000, 1, 4, false, x::telem::UINT8_T);
    const pdo::Entry high(1, 0x7000, 2, 4, false, x::telem::UINT8_T);
    this->master->add_slave({
        .position = 1,
        .vendor_id = 0x1,
        .product_code = 0x2,
        .name = "Slave1",
        .output_pdos = {pdo_of(low), pdo_of(high)},
    });
    auto low_writer = ASSERT_NIL_P(this->engine->open_writer(
        {{low, static_cast<uint8_t>(0x3)}},
        x::telem::Rate(100)
    ));
    const auto high_writer = ASSERT_NIL_P(this->engine->open_writer(
        {{high, static_cast<uint8_t>(0)}},
        x::telem::Rate(100)
    ));
    ASSERT_NIL(high_writer->open_tx().write(0, static_cast<uint8_t>(0xA)));
    ASSERT_EVENTUALLY_EQ(this->master->get_output<uint8_t>(SLAVE_IMAGE_SIZE), 0xA0);
    low_writer.reset();
    ASSERT_EVENTUALLY_EQ(this->master->get_output<uint8_t>(SLAVE_IMAGE_SIZE), 0xA3);
}

/// @brief it should send the stop values before the bus stops when the last writer
/// closes.
TEST_F(EngineTest, LastWriterToCloseSendsItsStopValuesBeforeTheBusStops) {
    auto writer = ASSERT_NIL_P(this->engine->open_writer(
        {{OUT_I16, static_cast<int16_t>(-7)}},
        x::telem::Rate(100)
    ));
    ASSERT_NIL(writer->open_tx().write(0, static_cast<int16_t>(0x1234)));
    ASSERT_EVENTUALLY_EQ(this->master->get_output<int16_t>(I16_BYTE), 0x1234);
    writer.reset();
    ASSERT_FALSE(this->engine->running());
    EXPECT_EQ(this->master->get_output<int16_t>(I16_BYTE), -7);
}

/// @brief it should keep the stop value a closed writer left on an output until a new
/// writer commands the output.
TEST_F(EngineTest, WriterKeepsTheStopValueAnotherWriterLeftUntilItCommands) {
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 3)
    );
    auto first = ASSERT_NIL_P(this->engine->open_writer(
        {{OUT_I16, static_cast<int16_t>(-7)}},
        x::telem::Rate(100)
    ));
    first.reset();
    ASSERT_EVENTUALLY_EQ(this->master->get_output<int16_t>(I16_BYTE), -7);
    auto writer = ASSERT_NIL_P(
        this->engine->open_writer({{OUT_I16, 0}}, x::telem::Rate(100))
    );
    EXPECT_EQ(this->read_times(*reader, 3).size(), 3);
    EXPECT_EQ(this->master->get_output<int16_t>(I16_BYTE), -7);
    ASSERT_NIL(writer->open_tx().write(0, static_cast<int16_t>(5)));
    ASSERT_EVENTUALLY_EQ(this->master->get_output<int16_t>(I16_BYTE), 5);
    writer.reset();
    ASSERT_EVENTUALLY_EQ(this->master->get_output<int16_t>(I16_BYTE), 0);
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
                    {{OUT_I16, 0}},
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

/// @brief it should keep the samples of a reader exactly one sample period apart when a
/// writer raises the bus rate between two of them.
TEST_F(EngineLockTest, ReaderKeepsExactSampleTimesWhenTheBusRateRises) {
    this->master->stall();
    // The writer starts the bus, so the reader joins a running bus at a known cycle.
    const auto slow = ASSERT_NIL_P(
        this->engine->open_writer({{OUT_I16, 0}}, x::telem::Rate(100))
    );
    // Destroyed before the reader and writers, which cannot stop a stalled bus.
    x::defer::defer resume([this] { this->master->resume(); });
    this->master->wait_until_stalled();
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(10), 3)
    );
    // The stalled cycle took its schedule before the reader joined.
    this->master->step();
    this->master->wait_until_stalled();
    // The stalled cycle samples, and the rate rises in the cycle after it.
    const auto fast = ASSERT_NIL_P(
        this->engine->open_writer({{OUT_I32, 0}}, x::telem::Rate(200))
    );
    this->master->resume();
    const auto times = this->read_times(*reader, 3);
    ASSERT_EQ(times.size(), 3);
    for (size_t i = 1; i < times.size(); ++i)
        EXPECT_EQ(
            x::telem::TimeSpan(times.at<int64_t>(i) - times.at<int64_t>(i - 1)),
            x::telem::MILLISECOND * 100
        );
}

/// @brief it should keep sample times in order through cycles that a lost slave makes
/// short, while the distributed clock runs ahead of the host clock.
TEST_F(EngineLockTest, ShortCyclesKeepSampleTimesInOrder) {
    constexpr size_t SAMPLES = 9;
    this->master->add_slave(io_slave(1));
    this->master->set_host_dc(1.005);
    // After 1000 frames, the mapped DC time runs about 30 ms ahead of the host clock.
    this->master->record_dc(1000);
    const auto writer = ASSERT_NIL_P(
        this->engine->open_writer({{OUT_I16, 0}}, x::telem::Rate(100))
    );
    ASSERT_EQ(this->master->recorded_dc().size(), 1000);
    this->master->stall();
    // Destroyed before the writer, which cannot stop a stalled bus.
    x::defer::defer resume([this] { this->master->resume(); });
    this->master->wait_until_stalled();
    this->master->set_slave_lost(1, true);
    ASSERT_EVENTUALLY_TRUE(
        this->engine->health().size() == 2 && this->engine->health()[1].lost
    );
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), SAMPLES)
    );
    // The stalled cycle took its schedule before the reader joined.
    this->master->step();
    for (size_t s = 0; s < SAMPLES; ++s) {
        this->master->wait_until_stalled();
        this->master->set_shortfall(
            s >= 3 && s < 6 ? std::optional<uint32_t>(3) : std::nullopt
        );
        this->master->step();
    }
    const auto times = this->read_times(*reader, SAMPLES);
    ASSERT_EQ(times.size(), SAMPLES);
    for (size_t i = 1; i < SAMPLES; ++i)
        EXPECT_GT(times.at<int64_t>(i), times.at<int64_t>(i - 1));
    EXPECT_EQ(reader->warning(), "");
}

/// @brief it should keep sample times in order when full cycles return after the
/// reference clock was lost, which leaves its distributed clock time behind.
TEST_F(EngineLockTest, FullCyclesAfterALostReferenceClockKeepTimesInOrder) {
    constexpr size_t SAMPLES = 9;
    this->master->add_slave(io_slave(1));
    this->master->set_dc_step(this->period);
    const auto writer = ASSERT_NIL_P(
        this->engine->open_writer({{OUT_I16, 0}}, x::telem::Rate(100))
    );
    this->master->stall();
    // Destroyed before the writer, which cannot stop a stalled bus.
    x::defer::defer resume([this] { this->master->resume(); });
    this->master->wait_until_stalled();
    this->master->set_slave_lost(1, true);
    ASSERT_EVENTUALLY_TRUE(
        this->engine->health().size() == 2 && this->engine->health()[1].lost
    );
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), SAMPLES)
    );
    // The stalled cycle took its schedule before the reader joined.
    this->master->step();
    for (size_t s = 0; s < SAMPLES; ++s) {
        this->master->wait_until_stalled();
        const bool lost = s >= 3 && s < 6;
        this->master->set_shortfall(lost ? std::optional<uint32_t>(3) : std::nullopt);
        // SOEM repeats the last DC time of a frame that misses the reference clock.
        this->master->set_dc_step(lost ? x::telem::TimeSpan(0) : this->period);
        this->master->step();
    }
    const auto times = this->read_times(*reader, SAMPLES);
    ASSERT_EQ(times.size(), SAMPLES);
    for (size_t i = 1; i < SAMPLES; ++i)
        EXPECT_GT(times.at<int64_t>(i), times.at<int64_t>(i - 1));
}

/// @brief it should decode each sample of reads that cross the end of the sample
/// buffer, for entries of different widths.
TEST_F(EngineLockTest, ReaderDecodesEachSampleAcrossTheBufferEnd) {
    // 210 samples pass through a buffer that holds 200, so the last read crosses its
    // end.
    constexpr size_t SAMPLES = 30;
    constexpr size_t READS = 7;
    // The writer runs the bus at the rate of the reader, so the reader joins a running
    // bus at a known cycle.
    const auto writer = ASSERT_NIL_P(
        this->engine->open_writer({{OUT_I16, 0}}, x::telem::Rate(100))
    );
    this->master->stall();
    // Destroyed before the writer, which cannot stop a stalled bus.
    x::defer::defer resume([this] { this->master->resume(); });
    this->master->wait_until_stalled();
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16, IN_I24, IN_U4}, x::telem::Rate(100), SAMPLES)
    );
    // The stalled cycle took its schedule before the reader joined.
    this->master->step();
    for (size_t r = 0; r < READS; ++r) {
        for (size_t s = 0; s < SAMPLES; ++s) {
            const auto k = static_cast<int32_t>(r * SAMPLES + s);
            this->master->wait_until_stalled();
            this->master->set_input<int16_t>(I16_BYTE, static_cast<int16_t>(k));
            this->master->set_input<int32_t>(I24_BYTE, -k);
            this->master->set_input<uint8_t>(U4_BYTE, static_cast<uint8_t>(k % 16));
            this->master->step();
        }
        auto frame = frame_of(
            {x::telem::INT16_T, x::telem::INT32_T, x::telem::UINT8_T},
            SAMPLES
        );
        x::telem::Series times(x::telem::TIMESTAMP_T, SAMPLES);
        ASSERT_NIL(reader->read(this->brk, frame, times));
        ASSERT_EQ(frame.series->at(0).size(), SAMPLES);
        for (size_t s = 0; s < SAMPLES; ++s) {
            const auto k = static_cast<int32_t>(r * SAMPLES + s);
            EXPECT_EQ(frame.series->at(0).at<int16_t>(s), k);
            EXPECT_EQ(frame.series->at(1).at<int32_t>(s), -k);
            EXPECT_EQ(frame.series->at(2).at<uint8_t>(s), k % 16);
        }
    }
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
        this->engine->open_writer({{OUT_I16, 0}}, x::telem::Rate(100))
    );
    ASSERT_NIL(writer->open_tx().write(0, static_cast<int16_t>(0x1234)));
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
    ASSERT_OCCURRED_AS(res.second, errors::PDO_NOT_MAPPED);
    EXPECT_EQ(res.second.data, "PDO 0x9999:01 is not in the process image of slave 0.");
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
        "PDO 0x6000:04 on slave 0 is 4 bits, not 32. Scan the network again."
    );
    EXPECT_FALSE(this->engine->running());
}

/// @brief it should keep the bus running when it rejects the entry of a new task.
TEST_F(EngineTest, RejectedEntryLeavesTheRunningBusUp) {
    const auto writer = ASSERT_NIL_P(
        this->engine->open_writer({{OUT_I16, 0}}, x::telem::Rate(100))
    );
    ASSERT_OCCURRED_AS_P(
        this->engine->open_writer(
            {{pdo::Entry(0, 0x9999, 1, 16, false, x::telem::INT16_T), 0}},
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
    this->engine = std::make_shared<Engine>("eth0", this->master, disabling(1));
    ASSERT_OCCURRED_AS_P(
        this->engine->open_reader({on_slave(IN_I16, 1)}, x::telem::Rate(100), 1),
        errors::PDO_MAPPING_ERROR
    );
}

/// @brief it should tell the user to restart every task on the network when the running
/// bus does not map an entry.
TEST_F(EngineTest, EntryOffTheRunningImageAsksForARestartOfTheTasks) {
    this->master->set_name("Ethernet 2");
    this->master->add_slave(io_slave(1));
    const auto disabled = std::make_shared<std::atomic<bool>>(true);
    Config cfg;
    cfg.slave_configs = [disabled] {
        std::map<uint16_t, slave::Config> configs;
        if (disabled->load()) configs[1] = {.enabled = false};
        return std::pair{configs, x::errors::NIL};
    };
    this->engine = std::make_shared<Engine>(NPF_INTERFACE, this->master, cfg);
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    disabled->store(false);
    const auto [late, err] = this->engine->open_reader(
        {on_slave(IN_I16, 1)},
        x::telem::Rate(100),
        1
    );
    ASSERT_OCCURRED_AS(err, errors::PDO_NOT_MAPPED);
    EXPECT_EQ(
        err.data,
        "PDO 0x6000:01 is not in the process image of slave 1. Stop every task on "
        "Ethernet 2, then start them again."
    );
    EXPECT_TRUE(this->engine->running());
}

/// @brief it should read the slave settings again each time the bus starts.
TEST_F(EngineTest, ReadsSlaveSettingsAtEachBusStart) {
    this->master->add_slave(io_slave(1));
    const auto disabled = std::make_shared<bool>(false);
    Config cfg;
    cfg.slave_configs = [disabled] {
        std::map<uint16_t, slave::Config> configs;
        if (*disabled) configs[1] = {.enabled = false};
        return std::pair{configs, x::errors::NIL};
    };
    this->engine = std::make_shared<Engine>("eth0", this->master, cfg);
    {
        const auto reader = ASSERT_NIL_P(
            this->engine->open_reader({on_slave(IN_I16, 1)}, x::telem::Rate(100), 1)
        );
    }
    ASSERT_FALSE(this->engine->running());
    *disabled = true;
    ASSERT_OCCURRED_AS_P(
        this->engine->open_reader({on_slave(IN_I16, 1)}, x::telem::Rate(100), 1),
        errors::PDO_MAPPING_ERROR
    );
}

/// @brief it should fail to start the bus with the error that kept the slave settings
/// from being read, before it initializes the master.
TEST_F(EngineTest, SlaveSettingsErrorFailsTheStart) {
    Config cfg;
    cfg.slave_configs = [] {
        return std::pair{
            std::map<uint16_t, slave::Config>{},
            x::errors::Error(errors::SLAVE_CONFIG_ERROR, "invalid settings")
        };
    };
    this->engine = std::make_shared<Engine>("eth0", this->master, cfg);
    const auto res = this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1);
    ASSERT_OCCURRED_AS(res.second, errors::SLAVE_CONFIG_ERROR);
    EXPECT_EQ(res.second.data, "invalid settings");
    EXPECT_EQ(this->master->call_count("scan"), 0);
    EXPECT_FALSE(this->engine->running());
}

/// @brief it should open a reader on a running bus when the slave settings cannot be
/// read, since only a bus start uses them.
TEST_F(EngineTest, SlaveSettingsErrorLeavesARunningBusOpen) {
    this->master->add_slave(io_slave(1));
    const auto failing = std::make_shared<bool>(false);
    Config cfg;
    cfg.slave_configs = [failing] {
        if (*failing)
            return std::pair{
                std::map<uint16_t, slave::Config>{},
                x::errors::Error(errors::SLAVE_CONFIG_ERROR, "invalid settings")
            };
        return std::pair{std::map<uint16_t, slave::Config>{}, x::errors::NIL};
    };
    this->engine = std::make_shared<Engine>("eth0", this->master, cfg);
    const auto first = ASSERT_NIL_P(
        this->engine->open_reader({on_slave(IN_I16, 1)}, x::telem::Rate(100), 1)
    );
    *failing = true;
    const auto second = ASSERT_NIL_P(
        this->engine->open_reader({on_slave(IN_I16, 1)}, x::telem::Rate(100), 1)
    );
    EXPECT_TRUE(this->engine->running());
    EXPECT_EQ(this->master->call_count("scan"), 1);
}

/// @brief it should read the slave settings again when the bus ran and stopped while an
/// open read them.
TEST_F(EngineTest, ReadsSlaveSettingsAgainWhenTheBusStopsDuringTheRead) {
    this->master->add_slave(io_slave(1));
    const auto calls = std::make_shared<std::atomic<int>>(0);
    const auto disabled = std::make_shared<std::atomic<bool>>(false);
    std::promise<void> release;
    Config cfg;
    cfg.slave_configs = [calls, disabled, released = release.get_future().share()] {
        std::map<uint16_t, slave::Config> configs;
        if (disabled->load()) configs[1] = {.enabled = false};
        if (calls->fetch_add(1) == 0) released.wait();
        return std::pair{configs, x::errors::NIL};
    };
    this->engine = std::make_shared<Engine>("eth0", this->master, cfg);
    std::pair<std::unique_ptr<Engine::Reader>, x::errors::Error> late;
    std::thread opening([&] {
        late = this->engine->open_reader({on_slave(IN_I16, 1)}, x::telem::Rate(100), 1);
    });
    ASSERT_EVENTUALLY_EQ(calls->load(), 1);
    {
        const auto reader = ASSERT_NIL_P(
            this->engine->open_reader({on_slave(IN_I16, 1)}, x::telem::Rate(100), 1)
        );
    }
    ASSERT_FALSE(this->engine->running());
    disabled->store(true);
    release.set_value();
    opening.join();
    ASSERT_OCCURRED_AS(late.second, errors::PDO_MAPPING_ERROR);
    EXPECT_EQ(calls->load(), 3);
}

/// @brief it should read the slave settings again each time a writer starts the bus.
TEST_F(EngineTest, WriterReadsSlaveSettingsAtEachBusStart) {
    this->master->add_slave(io_slave(1));
    const auto disabled = std::make_shared<bool>(false);
    Config cfg;
    cfg.slave_configs = [disabled] {
        std::map<uint16_t, slave::Config> configs;
        if (*disabled) configs[1] = {.enabled = false};
        return std::pair{configs, x::errors::NIL};
    };
    this->engine = std::make_shared<Engine>("eth0", this->master, cfg);
    {
        const auto writer = ASSERT_NIL_P(
            this->engine->open_writer({{on_slave(OUT_I16, 1), 0}}, x::telem::Rate(100))
        );
    }
    ASSERT_FALSE(this->engine->running());
    *disabled = true;
    ASSERT_OCCURRED_AS_P(
        this->engine->open_writer({{on_slave(OUT_I16, 1), 0}}, x::telem::Rate(100)),
        errors::PDO_MAPPING_ERROR
    );
}

/// @brief it should fail a writer's bus start with the error that kept the slave
/// settings from being read, before it initializes the master.
TEST_F(EngineTest, SlaveSettingsErrorFailsTheWriterStart) {
    Config cfg;
    cfg.slave_configs = [] {
        return std::pair{
            std::map<uint16_t, slave::Config>{},
            x::errors::Error(errors::SLAVE_CONFIG_ERROR, "invalid settings")
        };
    };
    this->engine = std::make_shared<Engine>("eth0", this->master, cfg);
    const auto res = this->engine->open_writer({{OUT_I16, 0}}, x::telem::Rate(100));
    ASSERT_OCCURRED_AS(res.second, errors::SLAVE_CONFIG_ERROR);
    EXPECT_EQ(res.second.data, "invalid settings");
    EXPECT_EQ(this->master->call_count("scan"), 0);
    EXPECT_FALSE(this->engine->running());
}

/// @brief it should open a writer on a running bus when the slave settings cannot be
/// read, since only a bus start uses them.
TEST_F(EngineTest, SlaveSettingsErrorLetsAWriterOpenOnARunningBus) {
    this->master->add_slave(io_slave(1));
    const auto failing = std::make_shared<bool>(false);
    Config cfg;
    cfg.slave_configs = [failing] {
        if (*failing)
            return std::pair{
                std::map<uint16_t, slave::Config>{},
                x::errors::Error(errors::SLAVE_CONFIG_ERROR, "invalid settings")
            };
        return std::pair{std::map<uint16_t, slave::Config>{}, x::errors::NIL};
    };
    this->engine = std::make_shared<Engine>("eth0", this->master, cfg);
    const auto first = ASSERT_NIL_P(
        this->engine->open_writer({{on_slave(OUT_I16, 1), 0}}, x::telem::Rate(100))
    );
    *failing = true;
    const auto second = ASSERT_NIL_P(
        this->engine->open_writer({{on_slave(OUT_I32, 1), 0}}, x::telem::Rate(100))
    );
    EXPECT_TRUE(this->engine->running());
    EXPECT_EQ(this->master->call_count("scan"), 1);
}

/// @brief it should pass the startup parameters in the slave settings to the master
/// when the bus starts.
TEST_F(EngineTest, StartupParametersReachTheMaster) {
    this->master->add_slave(io_slave(1));
    const std::vector<slave::StartupParameter> params{
        {.index = 0x8000,
         .sub_index = 0x15,
         .data_type = x::telem::UINT16_T,
         .value = 3}
    };
    Config cfg;
    cfg.slave_configs = [params] {
        return std::pair{
            std::map<uint16_t, slave::Config>{{1, {.startup_parameters = params}}},
            x::errors::NIL
        };
    };
    this->engine = std::make_shared<Engine>("eth0", this->master, cfg);
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({on_slave(IN_I16, 1)}, x::telem::Rate(100), 1)
    );
    EXPECT_EQ(this->master->config(1).startup_parameters, params);
}

namespace {
/// @brief sets the measurement of input 1 of a 6xSTG at the given position.
Settings measurement(const uint16_t position, const double value) {
    return {
        {position,
         {{.index = 0x8040,
           .sub_index = 1,
           .data_type = x::telem::UINT16_T,
           .value = value}}}
    };
}
}

/// @brief it should write a reader's settings in PRE-OP when it starts the bus.
TEST_F(EngineTest, WritesReaderSettingsAsStartupParametersOnStart) {
    const auto settings = measurement(0, 6);
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1, settings)
    );
    EXPECT_EQ(this->master->config(0).startup_parameters, settings.at(0));
    EXPECT_TRUE(this->master->written().empty());
}

/// @brief it should write a reader's settings to a bus that another task already runs.
TEST_F(EngineTest, WritesReaderSettingsToARunningBus) {
    const auto first = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    const auto settings = measurement(0, 6);
    const auto second = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1, settings)
    );
    const auto written = this->master->written();
    ASSERT_EQ(written.size(), 1);
    EXPECT_EQ(written[0].first, 0);
    EXPECT_EQ(written[0].second, settings.at(0));
}

/// @brief a 6xSTG that holds the mode in which DewesoftX sets it up.
const x::json::json SHARED_SETTINGS{{"mode", {{"shared", true}}}};

/// @brief it should write no settings to a slave that another program sets up when the
/// bus starts.
TEST_F(EngineTest, WritesNoSettingsToASharedSlaveOnStart) {
    auto shared = io_slave(1);
    shared.settings = SHARED_SETTINGS;
    this->master->add_slave(shared);
    Config cfg;
    cfg.slave_configs = [] {
        return std::pair{
            std::map<uint16_t, slave::Config>{
                {1, {.startup_parameters = measurement(1, 3).at(1)}}
            },
            x::errors::NIL
        };
    };
    this->engine = std::make_shared<Engine>("eth0", this->master, cfg);
    const auto reader = ASSERT_NIL_P(this->engine->open_reader(
        {on_slave(IN_I16, 1)},
        x::telem::Rate(100),
        1,
        measurement(1, 6)
    ));
    EXPECT_TRUE(this->master->config(1).startup_parameters.empty());
}

/// @brief it should write no settings to a slave that another program sets up on a
/// running bus.
TEST_F(EngineTest, WritesNoSettingsToASharedSlaveOnARunningBus) {
    auto shared = io_slave(1);
    shared.settings = SHARED_SETTINGS;
    this->master->add_slave(shared);
    const auto first = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    const auto second = ASSERT_NIL_P(this->engine->open_reader(
        {on_slave(IN_I16, 1)},
        x::telem::Rate(100),
        1,
        measurement(1, 6)
    ));
    EXPECT_TRUE(this->master->written().empty());
}

/// @brief it should not check the settings of a slave that another program sets up
/// against those of other readers.
TEST_F(EngineTest, SharedSlaveSettingsDoNotConflict) {
    auto shared = io_slave(1);
    shared.settings = SHARED_SETTINGS;
    this->master->add_slave(shared);
    const auto entries = std::vector{on_slave(IN_I16, 1)};
    const auto first = ASSERT_NIL_P(
        this->engine->open_reader(entries, x::telem::Rate(100), 1, measurement(1, 6))
    );
    const auto second = ASSERT_NIL_P(
        this->engine->open_reader(entries, x::telem::Rate(100), 1, measurement(1, 0))
    );
    EXPECT_TRUE(this->master->written().empty());
}

/// @brief it should not write the settings of a reader that the running bus cannot map.
TEST_F(EngineTest, UnmappedReaderLeavesARunningBusSettingsAlone) {
    const auto first = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    ASSERT_OCCURRED_AS_P(
        this->engine->open_reader(
            {pdo::Entry(0, 0x9999, 1, 16, true, x::telem::INT16_T)},
            x::telem::Rate(100),
            1,
            measurement(0, 6)
        ),
        errors::PDO_MAPPING_ERROR
    );
    EXPECT_TRUE(this->master->written().empty());
}

/// @brief it should refuse a reader that sets an object to another value than an open
/// reader does.
TEST_F(EngineTest, RefusesAReaderWhoseSettingsConflict) {
    const auto first = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1, measurement(0, 6))
    );
    ASSERT_OCCURRED_AS_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1, measurement(0, 0)),
        errors::SETTING_CONFLICT
    );
    const auto same = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1, measurement(0, 6))
    );
}

/// @brief it should refuse a reader whose settings conflict with a reader that opened
/// while it read the slave settings.
TEST_F(EngineTest, RefusesAConflictingReaderThatOpenedDuringItsSettingsRead) {
    const auto calls = std::make_shared<std::atomic<int>>(0);
    std::promise<void> release;
    Config cfg;
    cfg.slave_configs = [calls, released = release.get_future().share()] {
        if (calls->fetch_add(1) == 0) released.wait();
        return std::pair{std::map<uint16_t, slave::Config>{}, x::errors::NIL};
    };
    this->engine = std::make_shared<Engine>("eth0", this->master, cfg);
    std::pair<std::unique_ptr<Engine::Reader>, x::errors::Error> late;
    std::thread opening([&] {
        late = this->engine
                   ->open_reader({IN_I16}, x::telem::Rate(100), 1, measurement(0, 0));
    });
    ASSERT_EVENTUALLY_EQ(calls->load(), 1);
    const auto first = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1, measurement(0, 6))
    );
    release.set_value();
    opening.join();
    ASSERT_OCCURRED_AS(late.second, errors::SETTING_CONFLICT);
    EXPECT_TRUE(this->master->written().empty());
}

/// @brief it should let a reader set an object that a closed reader set to another
/// value.
TEST_F(EngineTest, ReleasesTheSettingsOfAClosedReader) {
    auto keep = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    auto first = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1, measurement(0, 6))
    );
    first.reset();
    const auto second = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1, measurement(0, 0))
    );
    EXPECT_EQ(this->master->config(0).startup_parameters, measurement(0, 0).at(0));
}

/// @brief it should fail to open a reader whose setting a slave on the running bus
/// rejects.
TEST_F(EngineTest, FailsAReaderWhoseSettingTheSlaveRejects) {
    const auto first = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    this->master->inject_write_error(
        x::errors::Error(errors::STARTUP_PARAMETER_ERROR, "rejected 0x8040:01")
    );
    ASSERT_OCCURRED_AS_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1, measurement(0, 6)),
        errors::STARTUP_PARAMETER_ERROR
    );
}

/// @brief it should return the error that kept the master from initializing once the
/// retries run out.
TEST_F(EngineTest, InitializeErrorPropagates) {
    this->engine = std::make_shared<Engine>("eth0", this->master, quick_start_config());
    this->master->inject_scan_error(
        x::errors::Error(errors::MASTER_INIT_ERROR, "init failed")
    );
    const auto res = this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1);
    ASSERT_OCCURRED_AS(res.second, errors::MASTER_INIT_ERROR);
    EXPECT_EQ(res.second.data, "Init failed.");
    EXPECT_EQ(this->master->call_count("scan"), 3);
    EXPECT_FALSE(this->engine->running());
}

/// @brief it should deactivate the master after each failed activation, and return the
/// activation error once the retries run out.
TEST_F(EngineTest, ActivationErrorPropagates) {
    this->engine = std::make_shared<Engine>("eth0", this->master, quick_start_config());
    this->master->inject_activate_error(
        x::errors::Error(errors::ACTIVATION_ERROR, "activate failed")
    );
    const auto res = this->engine->open_writer({{OUT_I16, 0}}, x::telem::Rate(100));
    ASSERT_OCCURRED_AS(res.second, errors::ACTIVATION_ERROR);
    EXPECT_EQ(res.second.data, "Activate failed.");
    EXPECT_EQ(this->master->call_count("activate"), 3);
    EXPECT_EQ(this->master->call_count("deactivate"), 3);
    EXPECT_FALSE(this->engine->running());
}

/// @brief it should discover the slaves again while the bus is stopped, and not while
/// it runs.
TEST_F(EngineTest, RediscoverOnlyWhileStopped) {
    ASSERT_NIL_P(this->engine->rediscover());
    EXPECT_EQ(this->master->call_count("deactivate"), 1);
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    ASSERT_NIL_P(this->engine->rediscover());
    EXPECT_EQ(this->master->call_count("deactivate"), 1);
}

/// @brief it should activate the master with the period of the bus rate.
TEST_F(EngineTest, ActivatesTheMasterWithTheBusPeriod) {
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    EXPECT_EQ(this->master->plan().period, x::telem::MILLISECOND * 10);
}

/// @brief it should return at once, without a retry, when the config of a slave cannot
/// run.
TEST_F(EngineTest, SlaveConfigErrorSkipsRetries) {
    this->engine = std::make_shared<Engine>("eth0", this->master, quick_start_config());
    this->master->inject_activate_error(
        x::errors::Error(errors::SLAVE_CONFIG_ERROR, "SYNC1 cycle is over the limit")
    );
    ASSERT_OCCURRED_AS_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1),
        errors::SLAVE_CONFIG_ERROR
    );
    EXPECT_EQ(this->master->call_count("activate"), 1);
    EXPECT_FALSE(this->engine->running());
}

/// @brief it should return at once, without a retry, when a slave aborts a setting.
TEST_F(EngineTest, RejectedSettingSkipsRetries) {
    this->engine = std::make_shared<Engine>("eth0", this->master, quick_start_config());
    this->master->inject_activate_error(
        x::errors::Error(errors::STARTUP_PARAMETER_REJECTED, "rejected 0x8040:1C")
    );
    ASSERT_OCCURRED_AS_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1),
        errors::STARTUP_PARAMETER_REJECTED
    );
    EXPECT_EQ(this->master->call_count("activate"), 1);
    EXPECT_FALSE(this->engine->running());
}

/// @brief it should retry a setting write that got no response, as from a slave whose
/// mailbox is slow after a power restart.
TEST_F(EngineTest, SettingWithNoResponseIsRetried) {
    this->engine = std::make_shared<Engine>("eth0", this->master, quick_start_config());
    this->master->inject_activate_error(
        x::errors::Error(errors::STARTUP_PARAMETER_ERROR, "0x8040:1C: no response")
    );
    ASSERT_OCCURRED_AS_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1),
        errors::STARTUP_PARAMETER_ERROR
    );
    EXPECT_EQ(this->master->call_count("activate"), 3);
}

/// @brief it should return at once, without a retry, when a slave refuses a state
/// change.
TEST_F(EngineTest, StateChangeRefusalSkipsRetries) {
    this->engine = std::make_shared<Engine>("eth0", this->master, quick_start_config());
    this->master->inject_activate_error(
        x::errors::Error(errors::STATE_CHANGE_REFUSED, "inconsistent settings")
    );
    const auto res = this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1);
    ASSERT_OCCURRED_AS(res.second, errors::STATE_CHANGE_REFUSED);
    EXPECT_EQ(res.second.data, "Inconsistent settings.");
    EXPECT_EQ(this->master->call_count("activate"), 1);
    EXPECT_EQ(this->master->call_count("deactivate"), 1);
    EXPECT_FALSE(this->engine->running());
}

/// @brief it should start the bus again without a slave that refuses a state.
TEST_F(EngineTest, StartsWithoutASlaveThatRefusesAState) {
    this->master->add_slave(io_slave(1));
    this->master->refuse(1, REFUSAL);
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    EXPECT_EQ(this->master->call_count("activate"), 2);
    EXPECT_EQ(this->read_times(*reader, 1).size(), 1);
}

/// @brief it should fail the start without a retry when every slave with process data
/// refuses a state, so the bus does not run with only a coupler.
TEST_F(EngineTest, StartFailsWhenOnlyASlaveWithoutProcessDataIsLeft) {
    this->master->add_slave({.position = 1, .name = "Coupler"});
    this->master->refuse(0, REFUSAL);
    auto res = this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1);
    EXPECT_EQ(
        res.second.data,
        "Failed to reach SAFE-OP: refused SAFE-OP: Inconsistent settings."
    );
    ASSERT_OCCURRED_AS_P(std::move(res), errors::STATE_CHANGE_REFUSED);
    EXPECT_EQ(this->master->call_count("activate"), 1);
}

/// @brief it should tell the user how to fix the DC settings of a left-out slave that
/// refused its DC sync configuration.
TEST_F(EngineTest, ReaderOnASlaveThatRefusedDCSyncGetsAdvice) {
    this->master->add_slave(io_slave(1));
    this->master->refuse(1, DC_SYNC_REFUSAL);
    auto res = this->engine->open_reader({on_slave(IN_I16, 1)}, x::telem::Rate(100), 1);
    EXPECT_EQ(
        res.second.data,
        "Slave 1 (Slave1) is not on the network: it refused SAFE-OP: Invalid DC SYNC "
        "configuration. Fix the device, then stop every task on the network to try it "
        "again. To fix \"Invalid DC SYNC configuration\", right-click the device, "
        "select Configure, and pick another sync mode."
    );
    ASSERT_OCCURRED_AS_P(std::move(res), errors::STATE_CHANGE_REFUSED);
}

/// @brief it should fail a reader on a slave that the bus started without, and stop the
/// bus that the reader started.
TEST_F(EngineTest, ReaderOnALeftOutSlaveFails) {
    this->master->add_slave(io_slave(1));
    this->master->refuse(1, REFUSAL);
    auto res = this->engine->open_reader({on_slave(IN_I16, 1)}, x::telem::Rate(100), 1);
    EXPECT_EQ(
        res.second.data,
        "Slave 1 (Slave1) is not on the network: it refused SAFE-OP: Inconsistent "
        "settings. Fix the device, then stop every task on the network to try it "
        "again."
    );
    ASSERT_OCCURRED_AS_P(std::move(res), errors::STATE_CHANGE_REFUSED);
    EXPECT_FALSE(this->engine->running());
}

/// @brief it should give each left-out slave and its reason in the error when the start
/// then fails.
TEST_F(EngineTest, FailedStartGivesTheLeftOutSlaves) {
    this->engine = std::make_shared<Engine>("eth0", this->master, quick_start_config());
    this->master->add_slave(io_slave(1));
    this->master->refuse(1, REFUSAL);
    this->master->inject_state_change_error(
        x::errors::Error(errors::STATE_CHANGE_ERROR, "failed to reach OP")
    );
    auto res = this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1);
    EXPECT_EQ(
        res.second.data,
        "Failed to reach OP. The Driver left out slave 1 because it refused SAFE-OP: "
        "Inconsistent settings."
    );
    ASSERT_OCCURRED_AS_P(std::move(res), errors::STATE_CHANGE_ERROR);
    EXPECT_FALSE(this->engine->running());
}

/// @brief it should end the error of a failed start with the DC sync advice once when a
/// left-out slave needs it.
TEST_F(EngineTest, FailedStartGivesTheDCSyncAdviceOfALeftOutSlave) {
    this->engine = std::make_shared<Engine>("eth0", this->master, quick_start_config());
    this->master->add_slave(io_slave(1));
    this->master->refuse(1, DC_SYNC_REFUSAL);
    this->master->inject_state_change_error(
        x::errors::Error(errors::STATE_CHANGE_ERROR, "failed to reach OP")
    );
    auto res = this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1);
    EXPECT_EQ(
        res.second.data,
        "Failed to reach OP. The Driver left out slave 1 because it refused SAFE-OP: "
        "Invalid DC SYNC configuration. " +
            std::string(slave::DC_SYNC_ADVICE)
    );
    ASSERT_OCCURRED_AS_P(std::move(res), errors::STATE_CHANGE_ERROR);
}

/// @brief it should give the DC sync advice once, at the end, when two left-out slaves
/// report "Invalid DC SYNC configuration".
TEST_F(EngineTest, FailedStartGivesTheDCSyncAdviceOnce) {
    this->engine = std::make_shared<Engine>("eth0", this->master, quick_start_config());
    this->master->add_slave(io_slave(1));
    this->master->add_slave(io_slave(2));
    this->master->refuse(1, DC_SYNC_REFUSAL);
    this->master->refuse(2, DC_SYNC_REFUSAL);
    this->master->inject_state_change_error(
        x::errors::Error(errors::STATE_CHANGE_ERROR, "failed to reach OP")
    );
    auto res = this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1);
    EXPECT_EQ(
        res.second.data,
        "Failed to reach OP. The Driver left out slave 1 because it refused SAFE-OP: "
        "Invalid DC SYNC configuration. The Driver left out slave 2 because it "
        "refused SAFE-OP: Invalid DC SYNC configuration. " +
            std::string(slave::DC_SYNC_ADVICE)
    );
    ASSERT_OCCURRED_AS_P(std::move(res), errors::STATE_CHANGE_ERROR);
}

/// @brief it should end the error of a failed start with each distinct advice of the
/// left-out slaves, in the order of the slaves.
TEST_F(EngineTest, FailedStartGivesEachDistinctAdvice) {
    this->engine = std::make_shared<Engine>("eth0", this->master, quick_start_config());
    this->master->add_slave(io_slave(1));
    this->master->add_slave(io_slave(2));
    this->master->add_slave(io_slave(3));
    this->master->refuse(1, DC_SYNC_REFUSAL);
    this->master->refuse(
        2,
        {.reason = "refused SAFE-OP: Invalid output configuration",
         .advice = slave::MODE_RESTART_ADVICE}
    );
    this->master->refuse(3, DC_SYNC_REFUSAL);
    this->master->inject_state_change_error(
        x::errors::Error(errors::STATE_CHANGE_ERROR, "failed to reach OP")
    );
    auto res = this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1);
    EXPECT_EQ(
        res.second.data,
        "Failed to reach OP. The Driver left out slave 1 because it refused SAFE-OP: "
        "Invalid DC SYNC configuration. The Driver left out slave 2 because it "
        "refused SAFE-OP: Invalid output configuration. The Driver left out slave 3 "
        "because it refused SAFE-OP: Invalid DC SYNC configuration. " +
            std::string(slave::DC_SYNC_ADVICE) + " " +
            std::string(slave::MODE_RESTART_ADVICE)
    );
    ASSERT_OCCURRED_AS_P(std::move(res), errors::STATE_CHANGE_ERROR);
}

/// @brief it should end the error of a failed start with the DC sync advice when every
/// slave refuses SAFE-OP with "Invalid DC SYNC configuration".
TEST_F(EngineTest, StartFailsWithTheDCSyncAdviceWhenEverySlaveRefuses) {
    this->master->refuse(0, DC_SYNC_REFUSAL);
    auto res = this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1);
    EXPECT_EQ(
        res.second.data,
        "Failed to reach SAFE-OP: refused SAFE-OP: Invalid DC SYNC configuration. " +
            std::string(slave::DC_SYNC_ADVICE)
    );
    ASSERT_OCCURRED_AS_P(std::move(res), errors::STATE_CHANGE_REFUSED);
}

/// @brief it should try a left-out slave again the next time the bus starts.
TEST_F(EngineTest, TriesALeftOutSlaveAgainAtTheNextStart) {
    this->master->add_slave(io_slave(1));
    this->master->refuse(1, REFUSAL);
    auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    reader.reset();
    this->master->clear_injected_errors();
    const auto other = ASSERT_NIL_P(
        this->engine->open_reader({on_slave(IN_I16, 1)}, x::telem::Rate(100), 1)
    );
    EXPECT_EQ(this->read_times(*other, 1).size(), 1);
}

/// @brief it should start again with a slave that refused its layout once, as after a
/// mode change, instead of leaving it out.
TEST_F(EngineTest, StartsAgainWithASlaveThatRefusedAStaleLayout) {
    this->master->add_slave(io_slave(1));
    this->master->refuse(1, STALE_LAYOUT_REFUSAL, 1);
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({on_slave(IN_I16, 1)}, x::telem::Rate(100), 1)
    );
    EXPECT_EQ(this->master->call_count("activate"), 2);
    EXPECT_TRUE(this->master->plan().slaves[1].enabled);
    EXPECT_EQ(this->read_times(*reader, 1).size(), 1);
}

/// @brief it should start again with the only slave on the bus when it refuses its
/// layout once.
TEST_F(EngineTest, StartsAgainWithTheOnlySlaveWhenItRefusesAStaleLayout) {
    this->master->refuse(0, STALE_LAYOUT_REFUSAL, 1);
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    EXPECT_EQ(this->master->call_count("activate"), 2);
    EXPECT_EQ(this->read_times(*reader, 1).size(), 1);
}

/// @brief it should leave out a slave that refuses its layout again after the reset.
TEST_F(EngineTest, LeavesOutASlaveThatRefusesAStaleLayoutTwice) {
    this->master->add_slave(io_slave(1));
    this->master->refuse(1, STALE_LAYOUT_REFUSAL);
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    EXPECT_EQ(this->master->call_count("activate"), 3);
    EXPECT_FALSE(this->master->plan().slaves[1].enabled);
    auto res = this->engine->open_reader({on_slave(IN_I16, 1)}, x::telem::Rate(100), 1);
    EXPECT_EQ(
        res.second.data,
        "Slave 1 (Slave1) is not on the network: it refused SAFE-OP: Invalid input "
        "configuration. Fix the device, then stop every task on the network to try it "
        "again. If the mode of the device changed, restart its power."
    );
    ASSERT_OCCURRED_AS_P(std::move(res), errors::STATE_CHANGE_REFUSED);
}

/// @brief it should leave out a slave with another refusal in the same pass that resets
/// a slave with a stale layout.
TEST_F(EngineTest, LeavesOutAnotherRefusalWhileItResetsAStaleLayout) {
    this->master->add_slave(io_slave(1));
    this->master->add_slave(io_slave(2));
    this->master->refuse(1, STALE_LAYOUT_REFUSAL, 1);
    this->master->refuse(2, REFUSAL);
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({on_slave(IN_I16, 1)}, x::telem::Rate(100), 1)
    );
    EXPECT_EQ(this->master->call_count("activate"), 2);
    auto plan = this->master->plan();
    EXPECT_TRUE(plan.slaves[1].enabled);
    EXPECT_FALSE(plan.slaves[2].enabled);
    EXPECT_EQ(this->read_times(*reader, 1).size(), 1);
}

/// @brief it should cycle the bus while the slaves move to OP.
TEST_F(EngineTest, StartCyclesTheBusBeforeTheSlavesReachOP) {
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    const auto log = this->master->call_log();
    const auto op = std::ranges::find(log, "op");
    ASSERT_NE(op, log.end());
    EXPECT_LT(std::ranges::find(log, "send"), op);
}

/// @brief it should run no slave recovery while the slaves move to OP, and stop the
/// cycle before it deactivates the master when a slave refuses a state.
TEST_F(EngineTest, RefusedStartStopsTheCycleBeforeDeactivating) {
    this->engine = std::make_shared<Engine>("eth0", this->master, quick_start_config());
    this->master->inject_state_change_error(
        x::errors::Error(errors::STATE_CHANGE_REFUSED, "invalid DC SYNC configuration")
    );
    ASSERT_OCCURRED_AS_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1),
        errors::STATE_CHANGE_REFUSED
    );
    const auto log = this->master->call_log();
    EXPECT_GE(std::ranges::count(log, "send"), 1);
    EXPECT_EQ(std::ranges::count(log, "check"), 0);
    EXPECT_EQ(std::ranges::count(log, "deactivate"), 1);
    EXPECT_EQ(log.back(), "deactivate");
    EXPECT_FALSE(this->engine->running());
    EXPECT_EQ(this->engine->cycle_rate(), x::telem::Rate(0));
}

/// @brief it should stop the cycle and retry when the slaves do not reach OP in time.
TEST_F(EngineTest, StartRetriesAfterTheSlavesMissOP) {
    this->engine = std::make_shared<Engine>("eth0", this->master, quick_start_config());
    this->master->inject_state_change_error(
        x::errors::Error(errors::STATE_CHANGE_ERROR, "failed to reach OP")
    );
    ASSERT_OCCURRED_AS_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1),
        errors::STATE_CHANGE_ERROR
    );
    EXPECT_EQ(this->master->call_count("activate"), 3);
    EXPECT_EQ(this->master->call_count("deactivate"), 3);
    EXPECT_EQ(this->master->call_log().back(), "deactivate");
    EXPECT_FALSE(this->engine->running());
}

/// @brief it should stop retrying a failed start when one more try would end after the
/// start timeout, and return the error of the last try.
TEST_F(EngineTest, StartStopsRetryingBeforeTheStartTimeout) {
    const auto clock = x::loop::fake_clock();
    this->master = std::make_shared<testutil::Master>(clock);
    this->master->add_slave(io_slave(0));
    this->master->set_activate_duration(x::telem::SECOND * 25);
    this->master->inject_activate_error(
        x::errors::Error(errors::ACTIVATION_ERROR, "activate failed")
    );
    auto cfg = quick_start_config();
    cfg.start_retry.max_retries = 10;
    cfg.host_clock = clock;
    cfg.start_timeout = x::telem::MINUTE;
    this->engine = std::make_shared<Engine>("eth0", this->master, cfg);
    const auto res = this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1);
    ASSERT_OCCURRED_AS(res.second, errors::ACTIVATION_ERROR);
    EXPECT_EQ(res.second.data, "Activate failed.");
    // The second try ends at 50 s, so a third would end at 75 s.
    EXPECT_EQ(this->master->call_count("activate"), 2);
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
    EXPECT_EQ(err.data, "Every bus cycle failed: expected 3, got 1.");
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
    EXPECT_EQ(err.data, "No inputs from eth0 within 220ms.");
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

/// @brief it should name each slave that is out of OP in the warning of cycles that
/// failed for a wrong working counter.
TEST_F(EngineTest, WorkingCounterWarningNamesTheSlavesOutOfOP) {
    this->master->add_slave(io_slave(1));
    this->master->add_slave(io_slave(2));
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    this->master->set_slave_lost(1, true);
    this->master->set_slave_transition_failure(2, slave::State::OP);
    this->master->fault_slave(2, slave::State::SAFE_OP, "Sync manager watchdog");
    ASSERT_EVENTUALLY_TRUE(
        this->engine->health().size() == 3 && this->engine->health()[1].lost &&
        this->engine->health()[2].faulted
    );
    const auto receives = this->master->call_count("receive");
    this->master->inject_receive_error(
        x::errors::Error(
            errors::WORKING_COUNTER_ERROR,
            "working counter mismatch: expected 9, got 3"
        )
    );
    ASSERT_EVENTUALLY_GE(this->master->call_count("receive"), receives + 3);
    this->master->clear_injected_errors();
    EXPECT_EQ(this->read_times(*reader, 1).size(), 1);
    EXPECT_EQ(
        reader->warning(),
        "Bus cycles failed: working counter mismatch: expected 9, got 3. Slave 1 "
        "(Slave1) is not in OP: not responding. Slave 2 (Slave2) is not in OP: SAFE-OP "
        "+ ERROR: Sync manager watchdog."
    );
}

/// @brief it should drop the samples of only the readers of a lost slave while the
/// slave explains the short working counter, name the slave, and read it again when it
/// returns.
TEST_F(EngineTest, ShortCycleDropsOnlyTheSamplesOfALostSlave) {
    this->master->add_slave(io_slave(1));
    const auto kept = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    const auto lost = ASSERT_NIL_P(
        this->engine->open_reader({on_slave(IN_I16, 1)}, x::telem::Rate(100), 1)
    );
    this->master->set_slave_lost(1, true);
    ASSERT_EVENTUALLY_TRUE(
        this->engine->health().size() == 2 && this->engine->health()[1].lost
    );
    this->master->set_shortfall(3);
    const auto short_since = x::telem::TimeStamp::now();
    ASSERT_EVENTUALLY_TRUE(
        x::telem::TimeStamp(this->read_times(*kept, 1).at<int64_t>(0)) > short_since
    );
    EXPECT_EQ(kept->warning(), "");
    ASSERT_EVENTUALLY_EQ(this->read_times(*lost, 1).size(), 0);
    EXPECT_EQ(lost->warning(), "Lost contact with slave 1 (Slave1).");
    this->master->set_shortfall(std::nullopt);
    this->master->set_slave_lost(1, false);
    ASSERT_EVENTUALLY_EQ(this->read_times(*lost, 1).size(), 1);
}

/// @brief it should tell a writer once when its slave leaves OP and once when it
/// returns, and end the return warning at the next transaction.
TEST_F(EngineTest, WriterWarnsWhileItsSlaveIsOutOfOP) {
    this->master->add_slave(io_slave(1));
    std::atomic<int> changes{0};
    const auto writer = ASSERT_NIL_P(this->engine->open_writer(
        {{on_slave(OUT_I16, 1), 0}},
        x::telem::Rate(100),
        [&changes] { ++changes; }
    ));
    EXPECT_EQ(writer->warning(), "");
    this->master->set_slave_lost(1, true);
    ASSERT_EVENTUALLY_EQ(changes.load(), 1);
    EXPECT_EQ(writer->warning(), "Lost contact with slave 1 (Slave1).");
    this->master->set_slave_lost(1, false);
    ASSERT_EVENTUALLY_EQ(changes.load(), 2);
    EXPECT_EQ(
        writer->warning(),
        "Slave 1 (Slave1) returned and took the last commanded values."
    );
    { const auto tx = writer->open_tx(); }
    EXPECT_EQ(writer->warning(), "");
    EXPECT_EQ(changes.load(), 2);
}

/// @brief it should not tell a writer that another slave left OP.
TEST_F(EngineTest, WriterIgnoresAnotherSlaveOutOfOP) {
    this->master->add_slave(io_slave(1));
    std::atomic<int> changes{0};
    const auto writer = ASSERT_NIL_P(
        this->engine->open_writer({{OUT_I16, 0}}, x::telem::Rate(100), [&changes] {
            ++changes;
        })
    );
    this->master->set_slave_lost(1, true);
    ASSERT_EVENTUALLY_TRUE(
        this->engine->health().size() == 2 && this->engine->health()[1].lost
    );
    EXPECT_EQ(writer->warning(), "");
    EXPECT_EQ(changes.load(), 0);
}

/// @brief it should name a writer's slave that responds out of OP by its state.
TEST_F(EngineTest, WriterNamesAResponsiveSlaveOutOfOP) {
    this->master->add_slave(io_slave(1));
    std::atomic<int> changes{0};
    const auto writer = ASSERT_NIL_P(this->engine->open_writer(
        {{on_slave(OUT_I16, 1), 0}},
        x::telem::Rate(100),
        [&changes] { ++changes; }
    ));
    this->master->set_slave_transition_failure(1, slave::State::OP);
    this->master->set_slave_state(1, slave::State::SAFE_OP);
    ASSERT_EVENTUALLY_EQ(changes.load(), 1);
    EXPECT_EQ(writer->warning(), "Slave 1 (Slave1) is not in OP.");
    this->master->set_slave_lost(1, true);
    ASSERT_EVENTUALLY_EQ(changes.load(), 2);
    EXPECT_EQ(writer->warning(), "Lost contact with slave 1 (Slave1).");
}

/// @brief it should warn a writer that opens while its slave is out of OP.
TEST_F(EngineTest, WriterOpenedWhileItsSlaveIsOutOfOPWarns) {
    this->master->add_slave(io_slave(1));
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    this->master->set_slave_lost(1, true);
    ASSERT_EVENTUALLY_TRUE(
        this->engine->health().size() == 2 && this->engine->health()[1].lost
    );
    const auto writer = ASSERT_NIL_P(
        this->engine->open_writer({{on_slave(OUT_I16, 1), 0}}, x::telem::Rate(100))
    );
    EXPECT_EQ(writer->warning(), "Lost contact with slave 1 (Slave1).");
}

/// @brief it should name a slave that responds out of OP by its state, not as lost.
TEST_F(EngineTest, ShortCycleNamesAResponsiveSlaveOutOfOP) {
    this->master->add_slave(io_slave(1));
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({on_slave(IN_I16, 1)}, x::telem::Rate(100), 1)
    );
    this->master->set_slave_transition_failure(1, slave::State::OP);
    this->master->set_slave_state(1, slave::State::SAFE_OP);
    ASSERT_EVENTUALLY_TRUE(
        this->engine->health().size() == 2 &&
        this->engine->health()[1].state == slave::State::SAFE_OP
    );
    this->master->set_shortfall(2);
    ASSERT_EVENTUALLY_EQ(this->read_times(*reader, 1).size(), 0);
    EXPECT_EQ(reader->warning(), "Slave 1 (Slave1) is not in OP.");
}

/// @brief it should drop the samples of every reader when the working counter falls
/// further short than the slaves out of OP explain.
TEST_F(EngineTest, ShortCycleBeyondTheLostSlavesFailsEveryReader) {
    this->master->add_slave(io_slave(1));
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    this->master->set_slave_lost(1, true);
    ASSERT_EVENTUALLY_TRUE(
        this->engine->health().size() == 2 && this->engine->health()[1].lost
    );
    this->master->set_shortfall(4);
    auto frame = frame_of({x::telem::INT16_T}, 1);
    x::telem::Series times(x::telem::TIMESTAMP_T, 1);
    ASSERT_EVENTUALLY_TRUE(
        reader->read(this->brk, frame, times).matches(errors::CYCLE_FAILED)
    );
    const auto err = reader->read(this->brk, frame, times);
    ASSERT_OCCURRED_AS(err, errors::CYCLE_FAILED);
    EXPECT_EQ(
        err.data,
        "Every bus cycle failed: short by 4. Slave 1 (Slave1) is not in OP: not "
        "responding."
    );
}

/// @brief it should drop the samples of every reader when the working counter falls
/// short by less than the slaves out of OP miss.
TEST_F(EngineTest, ShortCycleBelowTheLostSlavesFailsEveryReader) {
    this->master->add_slave(io_slave(1));
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    this->master->set_slave_lost(1, true);
    ASSERT_EVENTUALLY_TRUE(
        this->engine->health().size() == 2 && this->engine->health()[1].lost
    );
    this->master->set_shortfall(1);
    auto frame = frame_of({x::telem::INT16_T}, 1);
    x::telem::Series times(x::telem::TIMESTAMP_T, 1);
    ASSERT_EVENTUALLY_TRUE(
        reader->read(this->brk, frame, times).matches(errors::CYCLE_FAILED)
    );
}

/// @brief it should drop the samples of every reader when a slave in OP falls short
/// beside a slave in SAFE-OP, which misses only its outputs, so that no reader takes
/// the stale inputs of the slave in OP.
TEST_F(EngineTest, ShortCycleBeyondASafeOpSlaveFailsEveryReader) {
    this->master->add_slave(io_slave(1));
    auto inputs_only = io_slave(2);
    inputs_only.output_pdos.clear();
    this->master->add_slave(inputs_only);
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({on_slave(IN_I16, 2)}, x::telem::Rate(100), 1)
    );
    this->master->set_slave_transition_failure(1, slave::State::OP);
    this->master->set_slave_state(1, slave::State::SAFE_OP);
    ASSERT_EVENTUALLY_TRUE(
        this->engine->health().size() == 3 &&
        this->engine->health()[1].state == slave::State::SAFE_OP
    );
    this->master->set_shortfall(3);
    auto frame = frame_of({x::telem::INT16_T}, 1);
    x::telem::Series times(x::telem::TIMESTAMP_T, 1);
    ASSERT_EVENTUALLY_TRUE(
        reader->read(this->brk, frame, times).matches(errors::CYCLE_FAILED)
    );
}

/// @brief it should drop the samples of every reader when the working counter falls
/// short while every slave is in OP.
TEST_F(EngineTest, ShortCycleWithEverySlaveInOPFailsEveryReader) {
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    this->master->set_shortfall(1);
    auto frame = frame_of({x::telem::INT16_T}, 1);
    x::telem::Series times(x::telem::TIMESTAMP_T, 1);
    ASSERT_EVENTUALLY_TRUE(
        reader->read(this->brk, frame, times).matches(errors::CYCLE_FAILED)
    );
    const auto err = reader->read(this->brk, frame, times);
    ASSERT_OCCURRED_AS(err, errors::CYCLE_FAILED);
    EXPECT_EQ(err.data, "Every bus cycle failed: short by 1.");
}

/// @brief it should name no slave in the warning of cycles that failed for another
/// reason.
TEST_F(EngineTest, OtherCycleWarningNamesNoSlave) {
    this->master->add_slave(io_slave(1));
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    this->master->set_slave_lost(1, true);
    ASSERT_EVENTUALLY_TRUE(
        this->engine->health().size() == 2 && this->engine->health()[1].lost
    );
    const auto receives = this->master->call_count("receive");
    this->master->inject_receive_error(
        x::errors::Error(errors::CYCLIC_ERROR, "process data receive failed")
    );
    ASSERT_EVENTUALLY_GE(this->master->call_count("receive"), receives + 3);
    this->master->clear_injected_errors();
    EXPECT_EQ(this->read_times(*reader, 1).size(), 1);
    EXPECT_EQ(reader->warning(), "Bus cycles failed: process data receive failed.");
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
        this->engine->open_writer({{OUT_I16, 0}}, x::telem::Rate(250))
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
        "No bus rate at or below 10000 Hz on eth0 is a multiple of every read task "
        "sample rate (997 Hz, 300 Hz). Choose sample rates with a smaller common "
        "multiple."
    );
    EXPECT_EQ(this->engine->cycle_rate(), x::telem::Rate(997));
}

/// @brief it should reject a writer faster than the maximum bus rate and not start the
/// bus.
TEST_F(EngineTest, WriterAboveTheMaximumRateIsRejected) {
    const auto res = this->engine->open_writer({{OUT_I16, 0}}, x::telem::Rate(20000));
    ASSERT_OCCURRED_AS(res.second, errors::RATE_MISMATCH);
    EXPECT_EQ(
        res.second.data,
        "A write task on eth0 runs at 20000 Hz, above the maximum bus rate of 10000 "
        "Hz."
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
        this->engine->open_writer({{OUT_I16, 0}}, x::telem::Rate(1000))
    );
    ASSERT_EQ(this->engine->cycle_rate(), x::telem::Rate(1000));
    writer.reset();
    EXPECT_EQ(this->engine->cycle_rate(), x::telem::Rate(100));
    const auto times = this->read_times(*reader, 5);
    ASSERT_EQ(times.size(), 5);
    const auto span = x::telem::TimeSpan(times.at<int64_t>(4) - times.at<int64_t>(0));
    EXPECT_GE(span, x::telem::MILLISECOND * 20);
}

/// @brief it should refuse a task that needs another bus rate while a slave runs DC
/// sync, until every task on the port stops.
TEST_F(EngineTest, RefusesANewRateWhileASlaveRunsDCSync) {
    this->master->set_dc_sync({SYNC0_AT_10MS});
    auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    const auto res = this->engine->open_writer({{OUT_I16, 0}}, x::telem::Rate(250));
    ASSERT_OCCURRED_AS(res.second, errors::RATE_MISMATCH);
    EXPECT_EQ(
        res.second.data,
        "The bus on eth0 cycles at 100 Hz while a slave runs DC sync, and the rate "
        "cannot change until every task on the port stops. Use a sample rate that 100 "
        "Hz is a multiple of, or an execution rate of at most 100 Hz."
    );
    EXPECT_EQ(this->engine->cycle_rate(), x::telem::Rate(100));
    reader.reset();
    const auto writer = ASSERT_NIL_P(
        this->engine->open_writer({{OUT_I16, 0}}, x::telem::Rate(250))
    );
    EXPECT_EQ(this->engine->cycle_rate(), x::telem::Rate(250));
}

/// @brief it should refuse a reader whose sample rate the fixed bus rate does not serve
/// while a slave runs DC sync.
TEST_F(EngineTest, RefusesAReaderAtANewRateWhileASlaveRunsDCSync) {
    this->master->set_dc_sync({SYNC0_AT_10MS});
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    const auto res = this->engine->open_reader({IN_I32}, x::telem::Rate(300), 1);
    ASSERT_OCCURRED_AS(res.second, errors::RATE_MISMATCH);
    EXPECT_EQ(
        res.second.data,
        "The bus on eth0 cycles at 100 Hz while a slave runs DC sync, and the rate "
        "cannot change until every task on the port stops. Use a sample rate that 100 "
        "Hz is a multiple of, or an execution rate of at most 100 Hz."
    );
    EXPECT_EQ(this->engine->cycle_rate(), x::telem::Rate(100));
}

/// @brief it should keep the bus rate when a faster task closes and when a slower task
/// opens while a slave runs DC sync.
TEST_F(EngineTest, KeepsTheRateWhileASlaveRunsDCSync) {
    this->master->set_dc_sync({SYNC0_AT_10MS});
    auto writer = ASSERT_NIL_P(
        this->engine->open_writer({{OUT_I16, 0}}, x::telem::Rate(1000))
    );
    const auto slow = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    ASSERT_EQ(this->engine->cycle_rate(), x::telem::Rate(1000));
    writer.reset();
    EXPECT_EQ(this->engine->cycle_rate(), x::telem::Rate(1000));
    const auto other = ASSERT_NIL_P(
        this->engine->open_reader({IN_I32}, x::telem::Rate(200), 1)
    );
    EXPECT_EQ(this->engine->cycle_rate(), x::telem::Rate(1000));
}

/// @brief it should keep the bus rate while a slave runs a SYNC0 too fast to mark
/// process data, since its SYNC1 cycle comes from the bus period.
TEST_F(EngineTest, KeepsTheRateWhileASlaveRunsAFastSYNC0) {
    this->master->set_dc_sync(
        {{.assign_activate = 0x0700, .sync0_cycle = 1'000, .sync1_cycle = 9'999'000}}
    );
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    ASSERT_OCCURRED_AS_P(
        this->engine->open_writer({{OUT_I16, 0}}, x::telem::Rate(250)),
        errors::RATE_MISMATCH
    );
    EXPECT_EQ(this->engine->cycle_rate(), x::telem::Rate(100));
}

/// @brief it should report the health of each enabled slave and leave out disabled
/// slaves.
TEST_F(EngineTest, HealthReportsEachEnabledSlave) {
    this->master->add_slave(io_slave(1));
    this->engine = std::make_shared<Engine>("eth0", this->master, disabling(1));
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    ASSERT_EVENTUALLY_EQ(this->engine->health().size(), 1);
    EXPECT_EQ(
        this->engine->health()[0],
        (slave::Health{.position = 0, .state = slave::State::OP})
    );
}

/// @brief it should report a left-out slave in its health with why the bus started
/// without it.
TEST_F(EngineTest, HealthReportsALeftOutSlave) {
    this->master->add_slave(io_slave(1));
    this->master->refuse(1, REFUSAL);
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    ASSERT_EVENTUALLY_EQ(this->engine->health().size(), 2);
    EXPECT_EQ(
        this->engine->health()[1],
        (slave::Health{.position = 1, .refusal = REFUSAL})
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
        this->engine->open_writer({{OUT_I16, 0}}, x::telem::Rate(100))
    );
    ASSERT_NIL(writer->open_tx().write(0, static_cast<int16_t>(0x1234)));
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
            this->engine->open_writer({{OUT_I32, 0}}, x::telem::Rate(400))
        );
        ASSERT_NIL(churn_writer->open_tx().write(0, static_cast<int32_t>(i)));
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

/// @brief it should keep the discovered slaves and not reinitialize the master when the
/// slave count is unchanged.
TEST_F(EngineTest, DiscoverKeepsDiscoveryWhenTheBusIsUnchanged) {
    ASSERT_NIL_P(this->engine->discover());
    ASSERT_NIL_P(this->engine->discover());
    const auto calls = this->master->call_log();
    EXPECT_EQ(std::ranges::count(calls, "deactivate"), 0);
}

/// @brief it should reinitialize the master and discover again when the slave count
/// changes.
TEST_F(EngineTest, DiscoverRediscoversWhenASlaveIsPluggedIn) {
    ASSERT_NIL_P(this->engine->discover());
    this->master->set_bus_slaves(2);
    ASSERT_NIL_P(this->engine->discover());
    EXPECT_EQ(
        this->master->call_log(),
        (std::vector<std::string>{
            "scan",
            "count_slaves",
            "scan",
            "count_slaves",
            "deactivate",
            "scan",
        })
    );
}

/// @brief it should not count or discover slaves while the bus runs.
TEST_F(EngineTest, DiscoverDoesNotRediscoverWhileTheBusRuns) {
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    this->master->set_bus_slaves(2);
    this->master->clear_call_log();
    ASSERT_NIL_P(this->engine->discover());
    const auto calls = this->master->call_log();
    EXPECT_EQ(std::ranges::count(calls, "count_slaves"), 0);
}

/// @brief it should return the slaves that the master discovers.
TEST_F(EngineTest, DiscoverReturnsTheSlavesOnTheBus) {
    this->master->add_slave(io_slave(1));
    const auto slaves = ASSERT_NIL_P(this->engine->discover());
    ASSERT_EQ(slaves.size(), 2);
    EXPECT_EQ(slaves[0].properties.position, 0);
    EXPECT_EQ(slaves[1].properties.position, 1);
}

/// @brief the writes that switch a 6xSTG to standalone mode and save the mode.
const std::vector<slave::StartupParameter> STANDALONE{
    {.index = 0x8020, .sub_index = 9, .data_type = x::telem::UINT16_T, .value = 0},
    {.index = 0x8020, .sub_index = 2, .data_type = x::telem::UINT16_T, .value = 2},
};

/// @brief picks STANDALONE for any slave that configure finds.
std::pair<std::vector<slave::StartupParameter>, x::errors::Error>
standalone(const slave::DiscoveryResult &) {
    return {STANDALONE, x::errors::NIL};
}

/// @brief it should write to a slave on a stopped bus, then discover the slaves again.
TEST_F(EngineTest, ConfigureWritesOnAStoppedBusAndDiscoversAgain) {
    ASSERT_NIL(this->engine->configure(io_slave(0), standalone));
    EXPECT_EQ(
        this->master->call_log(),
        (std::vector<std::string>{"deactivate", "scan", "write", "deactivate", "scan"})
    );
    EXPECT_EQ(
        this->master->written(),
        (std::vector<std::pair<uint16_t, std::vector<slave::StartupParameter>>>{
            {0, STANDALONE},
        })
    );
}

/// @brief it should scan the bus again before it looks for the slave, so a slave that
/// restarted since the last scan is not looked up in that old scan.
TEST_F(EngineTest, ConfigureScansTheBusAgainBeforeItLooksForTheSlave) {
    ASSERT_NIL_P(this->engine->discover());
    auto restarted = io_slave(0);
    restarted.serial = 5;
    this->master->replace_slave(0, restarted);
    ASSERT_NIL(this->engine->configure(restarted, standalone));
    EXPECT_EQ(
        this->master->written(),
        (std::vector<std::pair<uint16_t, std::vector<slave::StartupParameter>>>{
            {0, STANDALONE},
        })
    );
}

/// @brief it should write to the slave at its position in the new scan, not at the
/// position that the caller saw.
TEST_F(EngineTest, ConfigureFindsTheSlaveInItsOwnScan) {
    auto first = io_slave(0);
    first.serial = 7;
    auto second = io_slave(1);
    second.serial = 8;
    this->master = std::make_shared<testutil::Master>();
    this->master->add_slave(first);
    this->master->add_slave(second);
    this->engine = std::make_shared<Engine>("eth0", this->master);
    auto seen = second;
    seen.position = 0;
    ASSERT_NIL(this->engine->configure(seen, standalone));
    EXPECT_EQ(
        this->master->written(),
        (std::vector<std::pair<uint16_t, std::vector<slave::StartupParameter>>>{
            {1, STANDALONE},
        })
    );
}

/// @brief it should refuse to write when the slave is no longer on the bus.
TEST_F(EngineTest, ConfigureRefusesASlaveThatIsNotOnTheBus) {
    auto gone = io_slave(0);
    gone.serial = 9;
    const auto err = this->engine->configure(gone, standalone);
    ASSERT_OCCURRED_AS(err, errors::SLAVE_CONFIG_ERROR);
    EXPECT_EQ(err.data, "Slave0 is not on eth0. Check its power and cabling.");
    EXPECT_EQ(this->master->call_count("write"), 0);
}

/// @brief it should refuse to write to a slave while a task runs the bus.
TEST_F(EngineTest, ConfigureRefusesWhileTheBusRuns) {
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    this->master->clear_call_log();
    const auto err = this->engine->configure(io_slave(0), standalone);
    ASSERT_OCCURRED_AS(err, errors::SLAVE_CONFIG_ERROR);
    EXPECT_EQ(err.data, "A task uses eth0. Stop the tasks that use it first.");
    EXPECT_EQ(this->master->call_count("write"), 0);
}

/// @brief it should refuse at once to write while a task starts the bus, without
/// waiting for the start to end.
TEST_F(EngineTest, ConfigureRefusesAtOnceWhileATaskStartsTheBus) {
    this->master->hold_activate();
    std::thread opener([this] {
        auto [reader, err] = this->engine
                                 ->open_reader({IN_I16}, x::telem::Rate(100), 1);
    });
    this->master->wait_until_activating();
    const auto err = this->engine->configure(io_slave(0), standalone);
    this->master->release_activate();
    opener.join();
    ASSERT_OCCURRED_AS(err, errors::SLAVE_CONFIG_ERROR);
    EXPECT_EQ(
        err.data,
        "A task is starting on eth0. Try again when the start ends, or stop the task."
    );
    EXPECT_EQ(this->master->call_count("write"), 0);
}

/// @brief it should write nothing and scan only once when the parameters for the slave
/// are empty.
TEST_F(EngineTest, ConfigureWritesNothingForNoParameters) {
    ASSERT_NIL(this->engine->configure(io_slave(0), [](const auto &) {
        return std::pair{std::vector<slave::StartupParameter>{}, x::errors::NIL};
    }));
    EXPECT_EQ(
        this->master->call_log(),
        (std::vector<std::string>{"deactivate", "scan"})
    );
}

/// @brief it should call the network by the name of its master, not by its interface.
TEST_F(EngineTest, MessagesUseTheNameOfTheMaster) {
    this->master->set_name("Ethernet 2");
    this->engine = std::make_shared<Engine>(NPF_INTERFACE, this->master);
    EXPECT_EQ(this->engine->name(), "Ethernet 2");
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    const auto err = this->engine->configure(io_slave(0), standalone);
    ASSERT_OCCURRED_AS(err, errors::SLAVE_CONFIG_ERROR);
    EXPECT_EQ(err.data, "A task uses Ethernet 2. Stop the tasks that use it first.");
}

/// @brief it should return the error of a write that the slave rejects.
TEST_F(EngineTest, ConfigureReturnsTheErrorOfTheWrite) {
    this->master->inject_write_error(
        x::errors::Error(errors::STARTUP_PARAMETER_ERROR, "rejected")
    );
    ASSERT_OCCURRED_AS(
        this->engine->configure(io_slave(0), standalone),
        errors::STARTUP_PARAMETER_ERROR
    );
}

/// @brief it should read an object of a slave while a task runs the bus.
TEST_F(EngineTest, ReadsAnObjectWhileTheBusRuns) {
    this->master->set_object(0, 0x8040, 0x1c, 0x40a00000);
    const auto reader = ASSERT_NIL_P(
        this->engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    this->read_times(*reader, 1);
    EXPECT_EQ(ASSERT_NIL_P(this->engine->read(0, 0x8040, 0x1c)), 0x40a00000);
}

/// @brief it should return the abort of an object that the slave does not have.
TEST_F(EngineTest, ReadReturnsTheAbortOfAMissingObject) {
    ASSERT_OCCURRED_AS_P(this->engine->read(0, 0x8040, 0x1d), coe::OBJECT_MISSING);
}

/// @brief it should create an engine for an interface on the first discovery.
TEST(PoolTest, DiscoverSlavesCreatesEngine) {
    auto master = std::make_shared<testutil::Master>();
    master->add_slave(io_slave(0));
    master->add_slave(io_slave(1));
    auto manager = std::make_unique<testutil::Manager>();
    manager->configure("eth0", master);
    Pool pool(std::move(manager), NO_SLAVE_CONFIGS);
    const auto slaves = ASSERT_NIL_P(pool.discover("eth0"));
    ASSERT_EQ(slaves.size(), 2);
    EXPECT_EQ(slaves[0].properties.position, 0);
    EXPECT_EQ(slaves[1].properties.position, 1);
}

/// @brief it should take no RT core for a network that it only scans.
TEST(PoolTest, DiscoverSlavesTakesNoRTCore) {
    const auto rt = std::make_shared<x::thread::rt::Manager>();
    if (rt->total_cores() == 0) GTEST_SKIP() << "the host has no RT cores";
    auto master = std::make_shared<testutil::Master>();
    master->add_slave(io_slave(0));
    auto manager = std::make_unique<testutil::Manager>();
    manager->configure("eth0", master);
    Pool pool(std::move(manager), NO_SLAVE_CONFIGS, rt);
    ASSERT_NIL_P(pool.discover("eth0"));
    EXPECT_EQ(rt->available_cores(), rt->total_cores());
}

/// @brief it should return the slaves of a running engine without a new discovery.
TEST(PoolTest, DiscoverSlavesReturnsFromRunningEngine) {
    auto master = std::make_shared<testutil::Master>();
    master->add_slave(io_slave(0));
    master->add_slave(io_slave(1));
    auto manager = std::make_unique<testutil::Manager>();
    manager->configure("eth0", master);
    Pool pool(std::move(manager), NO_SLAVE_CONFIGS);
    const auto engine = ASSERT_NIL_P(pool.acquire("eth0"));
    const auto reader = ASSERT_NIL_P(
        engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    EXPECT_TRUE(engine->running());
    const auto slaves = ASSERT_NIL_P(pool.discover("eth0"));
    ASSERT_EQ(slaves.size(), 2);
    EXPECT_EQ(slaves[0].properties.position, 0);
    EXPECT_EQ(slaves[1].properties.position, 1);
}

/// @brief it should discover again after a failed discovery, not return the failure.
TEST(PoolTest, DiscoverSlavesInitErrorNotCached) {
    auto master = std::make_shared<testutil::Master>();
    master->inject_scan_error(
        x::errors::Error(errors::MASTER_INIT_ERROR, "no interface")
    );
    auto manager = std::make_unique<testutil::Manager>();
    manager->configure("eth0", master);
    Pool pool(std::move(manager), NO_SLAVE_CONFIGS);
    ASSERT_OCCURRED_AS_P(pool.discover("eth0"), errors::MASTER_INIT_ERROR);
    master->clear_injected_errors();
    master->add_slave(io_slave(0));
    EXPECT_EQ(ASSERT_NIL_P(pool.discover("eth0")).size(), 1);
}

/// @brief it should give the slave settings loader the interface and the name of its
/// master.
TEST(PoolTest, SlaveSettingsLoaderGetsTheNameOfTheNetwork) {
    auto master = std::make_shared<testutil::Master>();
    master->add_slave(io_slave(0));
    master->set_name("Ethernet 2");
    auto manager = std::make_unique<testutil::Manager>();
    manager->configure(NPF_INTERFACE, master);
    std::vector<std::pair<std::string, std::string>> loads;
    Pool pool(
        std::move(manager),
        [&loads](const std::string &network, const std::string &name) {
            loads.emplace_back(network, name);
            return std::pair{std::map<uint16_t, slave::Config>{}, x::errors::NIL};
        }
    );
    const auto engine = ASSERT_NIL_P(pool.acquire(NPF_INTERFACE));
    const auto reader = ASSERT_NIL_P(
        engine->open_reader({IN_I16}, x::telem::Rate(100), 1)
    );
    EXPECT_EQ(
        loads,
        (std::vector<std::pair<std::string, std::string>>{
            {NPF_INTERFACE, "Ethernet 2"},
        })
    );
}
}
