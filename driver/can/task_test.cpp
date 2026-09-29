// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <chrono>
#include <cmath>
#include <cstdint>
#include <cstring>
#include <memory>
#include <numbers>
#include <string>
#include <unordered_map>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/bus/testutil/testutil.h"
#include "driver/bus/write.h"
#include "driver/can/factory.h"
#include "driver/can/loopback/loopback.h"
#include "driver/can/read.h"
#include "driver/can/scan_task.h"
#include "driver/can/slcan/slcan.h"
#include "driver/can/write.h"
#include "driver/pipeline/mock/pipeline.h"
#include "driver/task/task.h"

namespace driver::can {
using namespace bus::testutil;

namespace {
using Bytes = std::vector<std::uint8_t>;

constexpr std::uint32_t FAST_ID = 0x100;
constexpr std::uint32_t SLOW_ID = 0x18FF1234;
constexpr std::uint32_t SLOW_MASK = 0x1FFFFF00;
constexpr std::uint32_t UNKNOWN_ID = 0x7FF;
constexpr std::uint32_t COMMAND_ID = 0x200;
constexpr std::uint32_t HEARTBEAT_ID = 0x201;
const double SINE_SCALE = std::ldexp(1.0, -14);

/// @returns the raw sine of the fast frame with count n.
int sine(const int n) {
    return static_cast<int>(
        std::nearbyint(16384 * std::sin(2 * std::numbers::pi * n / 50))
    );
}

/// @returns the raw 12-bit temperature of the fast frame with count n.
int temperature(const int n) {
    return n % 400 - 200;
}

/// @returns the payload the vcan simulator sends in the fast frame with count n.
Bytes fast(const int n) {
    const auto count = static_cast<std::uint16_t>(n);
    const auto s = static_cast<std::uint16_t>(sine(n));
    const auto packed = static_cast<std::uint16_t>(
        (temperature(n) & 0xFFF) | (n % 16) << 12
    );
    const auto speed = static_cast<std::uint16_t>(n * 3);
    return {
        static_cast<std::uint8_t>(count),
        static_cast<std::uint8_t>(count >> 8),
        static_cast<std::uint8_t>(s),
        static_cast<std::uint8_t>(s >> 8),
        static_cast<std::uint8_t>(packed),
        static_cast<std::uint8_t>(packed >> 8),
        static_cast<std::uint8_t>(speed >> 8),
        static_cast<std::uint8_t>(speed),
    };
}

/// @returns the payload the vcan simulator sends in the slow frame with count n.
Bytes slow(const std::uint32_t n) {
    const float ratio = static_cast<float>(n) * 0.25f;
    Bytes out(8);
    std::memcpy(out.data(), &n, 4);
    std::memcpy(out.data() + 4, &ratio, 4);
    return out;
}

/// @returns a copy of the payload of frame.
Bytes payload(const Frame &frame) {
    return {frame.payload().begin(), frame.payload().end()};
}

/// @returns the timestamps of an index channel across frames, in order.
std::vector<std::int64_t>
times(const std::vector<x::telem::Frame> &frames, const synnax::channel::Key key) {
    std::vector<std::int64_t> out;
    for (const auto &fr: frames)
        for (const auto &[k, series]: fr) {
            if (k != key) continue;
            for (std::size_t i = 0; i < series.size(); i++)
                out.push_back(series.at<std::int64_t>(static_cast<int>(i)));
        }
    return out;
}

/// @returns every raw sample of the channel across frames, in order.
std::vector<std::string>
raws(const std::vector<x::telem::Frame> &frames, const synnax::channel::Key key) {
    std::vector<std::string> out;
    for (const auto &fr: frames)
        for (const auto &[k, series]: fr)
            if (k == key)
                for (auto &s: samples(series))
                    out.push_back(std::move(s));
    return out;
}

/// @returns the frames of the messages in cfg, failing the test when the bus cannot
/// carry one.
std::vector<Frame> message_frames(const bus::WriteConfig &cfg, const bool fd) {
    x::json::Parser parser(x::json::json::object());
    auto out = frames(parser, cfg, fd);
    EXPECT_TRUE(parser.ok()) << parser.error();
    return out;
}

/// @brief a bus that fails its next receive on request.
class FaultyBus final : public Bus {
    std::unique_ptr<Bus> inner;
    std::shared_ptr<std::atomic<bool>> fault;

public:
    FaultyBus(
        std::unique_ptr<Bus> inner,
        const synnax::can::Properties &props,
        std::shared_ptr<std::atomic<bool>> fault
    ):
        Bus(props.channel, props.fd, props.listen_only),
        inner(std::move(inner)),
        fault(std::move(fault)) {}

    std::pair<bool, x::errors::Error>
    receive(Frame &frame, const x::telem::TimeSpan timeout) override {
        if (this->fault->exchange(false)) return {false, TEMPORARY_HARDWARE_ERROR};
        return this->inner->receive(frame, timeout);
    }

    x::errors::Error close() override { return this->inner->close(); }

private:
    x::errors::Error transmit(const Frame &frame) override {
        return this->inner->send(frame);
    }
};

/// @brief a loopback backend that counts the buses it opens and can fault them.
class Counting final : public Backend {
public:
    std::shared_ptr<loopback::Backend> inner;
    std::atomic<int> opens = 0;
    std::shared_ptr<std::atomic<bool>> fault = std::make_shared<std::atomic<bool>>();

    explicit Counting(std::vector<std::string> channels = {}):
        inner(std::make_shared<loopback::Backend>(std::move(channels))) {}

    std::pair<std::vector<Channel>, x::errors::Error> scan() override {
        return this->inner->scan();
    }

    std::pair<std::unique_ptr<Bus>, x::errors::Error>
    open(const synnax::can::Properties &props) override {
        auto [bus, err] = this->inner->open(props);
        if (err) return {nullptr, err};
        ++this->opens;
        return {std::make_unique<FaultyBus>(std::move(bus), props, this->fault), err};
    }
};

/// @brief a bus whose received frames carry an adapter clock that reads one second at
/// the first frame and advances one second per frame.
class AdapterClockBus final : public Bus {
    std::unique_ptr<Bus> inner;
    std::int64_t received = 0;

public:
    AdapterClockBus(std::unique_ptr<Bus> inner, const synnax::can::Properties &props):
        Bus(props.channel, props.fd, props.listen_only), inner(std::move(inner)) {}

    std::pair<bool, x::errors::Error>
    receive(Frame &frame, const x::telem::TimeSpan timeout) override {
        auto res = this->inner->receive(frame, timeout);
        if (!res.first) return res;
        frame.time = x::telem::TimeStamp(++this->received * x::telem::SECOND);
        frame.clock = Clock::HARDWARE;
        return res;
    }

    x::errors::Error close() override { return this->inner->close(); }

private:
    x::errors::Error transmit(const Frame &frame) override {
        return this->inner->send(frame);
    }
};

/// @brief a loopback backend whose buses stamp frames with an adapter clock.
class AdapterClock final : public Backend {
public:
    std::shared_ptr<loopback::Backend> inner = std::make_shared<loopback::Backend>();

    std::pair<std::vector<Channel>, x::errors::Error> scan() override {
        return this->inner->scan();
    }

    std::pair<std::unique_ptr<Bus>, x::errors::Error>
    open(const synnax::can::Properties &props) override {
        auto [bus, err] = this->inner->open(props);
        if (err) return {nullptr, err};
        return {std::make_unique<AdapterClockBus>(std::move(bus), props), err};
    }
};

/// @brief runs CAN tasks on a loopback bus. The test plays the other node on the
/// bus through peer.
class CANTask : public ::testing::Test {
protected:
    std::shared_ptr<Counting> backend = std::make_shared<Counting>();
    std::shared_ptr<const Backends> backends = std::make_shared<const Backends>(
        Backends{{loopback::BACKEND, backend}}
    );
    std::shared_ptr<Links> links = std::make_shared<Links>();
    synnax::can::Properties props;
    std::unique_ptr<Bus> peer;
    std::shared_ptr<task::MockContext> ctx = std::make_shared<task::MockContext>(
        nullptr
    );
    synnax::task::Task task{.key = x::uuid::create(), .name = "can"};
    std::shared_ptr<std::vector<x::telem::Frame>>
        writes = std::make_shared<std::vector<x::telem::Frame>>();

    void SetUp() override {
        this->props.backend = loopback::BACKEND;
        this->props.channel = "can0";
        this->peer = ASSERT_NIL_P(this->backend->inner->open(this->props));
    }

    [[nodiscard]] Acquire acquire() const {
        return acquirer(this->links, "dev", this->backends, this->props);
    }

    std::unique_ptr<common::ReadTask> read_task(bus::ReadConfig cfg) {
        return std::make_unique<common::ReadTask>(
            this->task,
            this->ctx,
            x::breaker::default_config(this->task.name),
            std::make_unique<Source>(std::move(cfg), this->acquire()),
            std::make_shared<pipeline::mock::WriterFactory>(this->writes)
        );
    }

    std::unique_ptr<bus::Sink> sink(bus::WriteConfig cfg) {
        auto transmitter = std::make_unique<Transmitter>(
            message_frames(cfg, this->props.fd),
            this->acquire()
        );
        auto s = std::make_unique<bus::Sink>(std::move(cfg), std::move(transmitter));
        s->set_warning = [](const x::errors::Error &) {};
        return s;
    }

    void send(const std::uint32_t id, const bool extended, const Bytes &data) const {
        Frame frame{.id = id, .extended = extended};
        frame.length = static_cast<std::uint8_t>(data.size());
        std::ranges::copy(data, frame.data.begin());
        ASSERT_NIL(this->peer->send(frame));
    }

    /// @returns the frames the peer receives within window.
    std::vector<Frame> received(const x::telem::TimeSpan window) const {
        std::vector<Frame> out;
        const auto deadline = x::telem::TimeStamp::now() + window;
        Frame frame;
        for (auto now = x::telem::TimeStamp::now(); now < deadline;
             now = x::telem::TimeStamp::now()) {
            auto [got, err] = this->peer->receive(frame, deadline - now);
            EXPECT_FALSE(err) << err;
            if (got) out.push_back(frame);
        }
        return out;
    }

    /// @brief discards the frames the peer has received.
    void drain() const {
        Frame frame;
        while (ASSERT_NIL_P(this->peer->receive(frame, x::telem::TimeSpan::ZERO()))) {}
    }

    /// @brief waits for count frames with the identifier and payload, skipping others.
    /// @returns the time from the first match to the last, or nullopt on timeout.
    std::optional<x::telem::TimeSpan> wait_for(
        const std::uint32_t id,
        const Bytes &data,
        const int count,
        const x::telem::TimeSpan timeout = 3 * x::telem::SECOND
    ) const {
        const auto deadline = x::telem::TimeStamp::now() + timeout;
        std::optional<x::telem::TimeStamp> first;
        int seen = 0;
        Frame frame;
        for (auto now = x::telem::TimeStamp::now(); now < deadline;
             now = x::telem::TimeStamp::now()) {
            auto [got, err] = this->peer->receive(frame, deadline - now);
            EXPECT_FALSE(err) << err;
            if (!got || frame.id != id || payload(frame) != data) continue;
            if (!first) first = frame.time;
            if (++seen == count) return frame.time - *first;
        }
        return std::nullopt;
    }
};

/// @returns the library messages of the vcan read case.
std::vector<synnax::library::MessageEntry> read_messages() {
    auto sine_field = binary_field("sine", 16, 16);
    sine_field.signed_ = true;
    sine_field.scale = SINE_SCALE;
    auto temperature_field = binary_field("temperature", 32, 12);
    temperature_field.signed_ = true;
    temperature_field.scale = 0.5;
    temperature_field.offset = -10;
    auto speed_field = binary_field("speed", 55, 16);
    speed_field.byte_order = synnax::library::BYTE_ORDER_BIG_ENDIAN;
    auto fast_msg = binary_message(
        "fast",
        {binary_field("count", 0, 16),
         sine_field,
         temperature_field,
         binary_field("state", 44, 4),
         speed_field},
        synnax::library::CanIdentifier{.id = FAST_ID}
    );
    fast_msg.length = 8;
    auto ratio = binary_field("ratio", 32, 32);
    ratio.float_ = true;
    auto slow_msg = binary_message(
        "slow",
        {binary_field("count", 0, 32), ratio},
        synnax::library::CanIdentifier{
            .id = SLOW_ID & SLOW_MASK,
            .extended = true,
            .mask = SLOW_MASK,
        }
    );
    slow_msg.length = 8;
    return {fast_msg, slow_msg};
}

/// @brief resolves a CAN write config over messages. The fields of the messages map
/// in order to command channels numbered from 7.
bus::WriteConfig
write_config(const std::vector<synnax::library::MessageEntry> &messages) {
    ::synnax::bus::WriteConfig cfg;
    cfg.device = "dev";
    std::vector<synnax::channel::Channel> channels;
    synnax::channel::Key next = 7;
    for (const auto &m: messages) {
        ::synnax::bus::WriteMessage wm{.message = m.key};
        for (const auto &f: m.fields) {
            wm.fields.push_back({.field = key(f), .channel = next});
            channels.push_back(data_channel(next++, 0));
        }
        cfg.messages.push_back(wm);
    }
    x::json::Parser parser(x::json::json::object());
    auto out = bus::WriteConfig::resolve(
        parser,
        cfg,
        std::nullopt,
        library(messages),
        channels,
        bus::Medium::CAN
    );
    EXPECT_TRUE(parser.ok()) << parser.error_json().dump();
    return out;
}

/// @returns the command and heartbeat messages of the vcan write case.
std::vector<synnax::library::MessageEntry> write_messages() {
    auto setpoint = binary_field("setpoint", 0, 16);
    setpoint.scale = 0.5;
    auto trim = binary_field("trim", 16, 8);
    trim.signed_ = true;
    auto command = binary_message(
        "command",
        {setpoint, trim, binary_field("enabled", 24, 1)},
        synnax::library::CanIdentifier{.id = COMMAND_ID}
    );
    command.length = 8;
    auto heartbeat = binary_message(
        "heartbeat",
        {binary_field("beat", 0, 16)},
        synnax::library::CanIdentifier{.id = HEARTBEAT_ID}
    );
    heartbeat.length = 2;
    heartbeat.period = 100 * x::telem::MILLISECOND;
    return {command, heartbeat};
}

/// @returns a command frame for channels 7 to 10: setpoint, trim, enabled, and beat.
x::telem::Frame command(const std::vector<double> &values) {
    x::telem::Frame fr(values.size());
    for (std::size_t i = 0; i < values.size(); i++)
        fr.emplace(
            static_cast<synnax::channel::Key>(7 + i),
            x::telem::Series(values[i], x::telem::FLOAT64_T)
        );
    return fr;
}
}

/// @brief the flow of integration/tests/driver/can_read.py on a loopback bus.
TEST_F(CANTask, DecodesStandardAndMaskedExtendedFramesAndIgnoresUnknownOnes) {
    auto t = this->read_task(read_config(
        read_messages(),
        std::nullopt,
        [](auto &cfg) { cfg.raw = 50; },
        {},
        {data_channel(50, 0, x::telem::BYTES_T)},
        bus::Medium::CAN
    ));
    t->start("start");
    ASSERT_EVENTUALLY_EQ(this->backend->opens.load(), 1);
    constexpr int FRAMES = 50;
    for (int n = 0; n < FRAMES; n++) {
        this->send(FAST_ID, false, fast(n));
        if (n % 5 == 0) this->send(SLOW_ID, true, slow(n / 5));
        if (n % 10 == 0) this->send(UNKNOWN_ID, false, Bytes(8, 0xFF));
    }
    ASSERT_EVENTUALLY_EQ(values(*this->writes, 6).size(), FRAMES / 5);
    ASSERT_EVENTUALLY_EQ(values(*this->writes, 1).size(), FRAMES);
    t->stop("stop", true);
    std::vector<double> counts, sines, temps, states, speeds, slow_counts, ratios;
    for (int n = 0; n < FRAMES; n++) {
        counts.push_back(n);
        sines.push_back(sine(n) * SINE_SCALE);
        temps.push_back(temperature(n) * 0.5 - 10);
        states.push_back(n % 16);
        speeds.push_back(n * 3);
    }
    for (int n = 0; n < FRAMES / 5; n++) {
        slow_counts.push_back(n);
        ratios.push_back(n * 0.25);
    }
    EXPECT_EQ(values(*this->writes, 1), counts);
    EXPECT_EQ(values(*this->writes, 2), sines);
    EXPECT_EQ(values(*this->writes, 3), temps);
    EXPECT_EQ(values(*this->writes, 4), states);
    EXPECT_EQ(values(*this->writes, 5), speeds);
    EXPECT_EQ(values(*this->writes, 6), slow_counts);
    EXPECT_EQ(values(*this->writes, 7), ratios);
    for (const synnax::channel::Key index: {100, 101}) {
        const auto stamps = times(*this->writes, index);
        EXPECT_EQ(stamps.size(), index == 100 ? FRAMES : FRAMES / 5);
        EXPECT_TRUE(std::ranges::is_sorted(stamps));
    }
    const auto raw = raws(*this->writes, 50);
    ASSERT_EQ(raw.size(), FRAMES + FRAMES / 5 + FRAMES / 10);
    const auto first = fast(0);
    EXPECT_EQ(raw[0], std::string("\x00\x01\x00\x00", 4) + text(first));
    EXPECT_EQ(raw[1], std::string("\x34\x12\xFF\x98", 4) + text(slow(0)));
    EXPECT_EQ(raw[2], std::string("\xFF\x07\x00\x00", 4) + std::string(8, '\xFF'));
}

/// @brief the flow of integration/tests/driver/can_write.py on a loopback bus.
TEST_F(CANTask, SendsCommandsAndAPeriodicHeartbeatWithTheLatestValue) {
    auto s = this->sink(write_config(write_messages()));
    ASSERT_NIL(s->start());
    EXPECT_TRUE(this->received(300 * x::telem::MILLISECOND).empty());
    struct Round {
        std::vector<double> values;
        Bytes command;
        Bytes heartbeat;
    };
    const std::vector<Round> rounds = {
        {{12.5, -3, 1, 7},
         {0x19, 0x00, 0xFD, 0x01, 0x00, 0x00, 0x00, 0x00},
         {0x07, 0x00}},
        {{100, 42, 0, 9},
         {0xC8, 0x00, 0x2A, 0x00, 0x00, 0x00, 0x00, 0x00},
         {0x09, 0x00}},
    };
    for (const auto &r: rounds) {
        auto fr = command(r.values);
        ASSERT_NIL(s->write(fr));
        ASSERT_TRUE(this->wait_for(COMMAND_ID, r.command, 1).has_value());
        const auto span = this->wait_for(HEARTBEAT_ID, r.heartbeat, 5);
        ASSERT_TRUE(span.has_value());
        EXPECT_GE(*span, 300 * x::telem::MILLISECOND);
    }
    ASSERT_NIL(s->stop());
    this->drain();
    EXPECT_TRUE(this->received(300 * x::telem::MILLISECOND).empty());
}

TEST_F(CANTask, SendsNoHeartbeatUntilItsChannelHasAValue) {
    auto s = this->sink(write_config(write_messages()));
    ASSERT_NIL(s->start());
    x::telem::Frame fr(3);
    for (synnax::channel::Key k = 7; k <= 9; k++)
        fr.emplace(k, x::telem::Series(1.0, x::telem::FLOAT64_T));
    ASSERT_NIL(s->write(fr));
    const auto frames = this->received(400 * x::telem::MILLISECOND);
    ASSERT_EQ(frames.size(), 1);
    EXPECT_EQ(frames[0].id, COMMAND_ID);
    ASSERT_NIL(s->stop());
}

TEST_F(CANTask, SharesOneBusBetweenAReadAndAWriteTaskOnADevice) {
    auto reader = this->read_task(read_config(
        {binary_message(
            "status",
            {binary_field("v", 0)},
            synnax::library::CanIdentifier{.id = 0x10}
        )},
        std::nullopt,
        nullptr,
        {},
        {},
        bus::Medium::CAN
    ));
    auto cmd = binary_message(
        "cmd",
        {binary_field("v", 0)},
        synnax::library::CanIdentifier{.id = 0x20}
    );
    auto writer = this->sink(write_config({cmd}));
    reader->start("start");
    ASSERT_NIL(writer->start());
    ASSERT_EVENTUALLY_EQ(this->backend->opens.load(), 1);
    x::telem::Frame fr(7, x::telem::Series(5.0, x::telem::FLOAT64_T));
    ASSERT_NIL(writer->write(fr));
    ASSERT_TRUE(this->wait_for(0x20, {5}, 1).has_value());
    this->send(0x10, false, {42});
    ASSERT_EVENTUALLY_EQ(values(*this->writes, 1).size(), 1);
    ASSERT_NIL(writer->stop());
    reader->stop("stop", true);
    EXPECT_EQ(this->backend->opens.load(), 1);
    EXPECT_EQ(values(*this->writes, 1)[0], 42);
    auto [link, err] = this->acquire()();
    ASSERT_NIL(err);
    ASSERT_NIL(link->bus().second);
    EXPECT_EQ(this->backend->opens.load(), 2);
}

TEST_F(CANTask, ReopensTheBusAfterAReceiveFault) {
    auto t = this->read_task(read_config(
        {binary_message(
            "status",
            {binary_field("v", 0)},
            synnax::library::CanIdentifier{.id = 0x10}
        )},
        std::nullopt,
        nullptr,
        {},
        {},
        bus::Medium::CAN
    ));
    t->start("start");
    ASSERT_EVENTUALLY_EQ(this->backend->opens.load(), 1);
    this->backend->fault->store(true);
    ASSERT_EVENTUALLY_EQ_WITH_TIMEOUT(
        this->backend->opens.load(),
        2,
        std::chrono::seconds(5),
        std::chrono::milliseconds(10)
    );
    ASSERT_EVENTUALLY_TRUE([&] {
        this->send(0x10, false, {9});
        return !values(*this->writes, 1).empty();
    }());
    t->stop("stop", true);
    EXPECT_EQ(values(*this->writes, 1)[0], 9);
}

TEST_F(CANTask, MovesAdapterTimesOntoTheHostClock) {
    const auto adapter = std::make_shared<AdapterClock>();
    const auto backends = std::make_shared<const Backends>(
        Backends{{loopback::BACKEND, adapter}}
    );
    const auto peer = ASSERT_NIL_P(adapter->inner->open(this->props));
    Source source(
        read_config(
            {binary_message(
                "status",
                {binary_field("v", 0)},
                synnax::library::CanIdentifier{.id = 0x10}
            )},
            std::nullopt,
            nullptr,
            {},
            {},
            bus::Medium::CAN
        ),
        acquirer(this->links, "dev", backends, this->props)
    );
    ASSERT_NIL(source.start());
    const auto before = x::telem::TimeStamp::now();
    const Frame frame{.id = 0x10, .length = 1};
    for (int i = 0; i < 3; i++)
        ASSERT_NIL(peer->send(frame));
    x::breaker::Breaker breaker;
    std::vector<x::telem::Frame> frames;
    ASSERT_EVENTUALLY_TRUE([&] {
        x::telem::Frame fr;
        EXPECT_FALSE(source.read(breaker, fr).error);
        frames.push_back(std::move(fr));
        return times(frames, 100).size() == 3;
    }());
    ASSERT_NIL(source.stop());
    const auto stamps = times(frames, 100);
    EXPECT_GE(stamps[0], before.nanoseconds());
    EXPECT_LE(stamps[0], x::telem::TimeStamp::now().nanoseconds());
    EXPECT_EQ(stamps[1] - stamps[0], x::telem::SECOND.nanoseconds());
    EXPECT_EQ(stamps[2] - stamps[1], x::telem::SECOND.nanoseconds());
}

TEST_F(CANTask, SlewsAdapterTimesTowardTheHostClock) {
    const auto adapter = std::make_shared<AdapterClock>();
    const auto backends = std::make_shared<const Backends>(
        Backends{{loopback::BACKEND, adapter}}
    );
    const auto peer = ASSERT_NIL_P(adapter->inner->open(this->props));
    Source source(
        read_config(
            {binary_message(
                "status",
                {binary_field("v", 0)},
                synnax::library::CanIdentifier{.id = 0x10}
            )},
            std::nullopt,
            nullptr,
            {},
            {},
            bus::Medium::CAN
        ),
        acquirer(this->links, "dev", backends, this->props)
    );
    ASSERT_NIL(source.start());
    constexpr std::size_t count = 13;
    const Frame frame{.id = 0x10, .length = 1};
    for (std::size_t i = 0; i < count; i++)
        ASSERT_NIL(peer->send(frame));
    x::breaker::Breaker breaker;
    std::vector<x::telem::Frame> frames;
    ASSERT_EVENTUALLY_TRUE([&] {
        x::telem::Frame fr;
        EXPECT_FALSE(source.read(breaker, fr).error);
        frames.push_back(std::move(fr));
        return times(frames, 100).size() == count;
    }());
    ASSERT_NIL(source.stop());
    // The adapter clock gains a second per frame on the host clock. Once a window of
    // adapter time passes, the offset slews toward the host by 1/SLEW of each step.
    const auto stamps = times(frames, 100);
    const auto window = static_cast<std::size_t>(
        (Aligner::WINDOW / x::telem::SECOND).nanoseconds()
    );
    const auto slewed = x::telem::SECOND - x::telem::SECOND / Aligner::SLEW;
    for (std::size_t i = 1; i < count; i++)
        EXPECT_EQ(
            stamps[i] - stamps[i - 1],
            (i < window ? x::telem::SECOND : slewed).nanoseconds()
        ) << i;
}

TEST_F(CANTask, RejectsAFrameLongerThanTheBusCarries) {
    auto m = binary_message(
        "big",
        {binary_field("v", 0)},
        synnax::library::CanIdentifier{.id = 0x20}
    );
    Transmitter tx(message_frames(write_config({m}), false), this->acquire());
    ASSERT_NIL(tx.acquire());
    ASSERT_OCCURRED_AS(tx.send(0, Bytes(65, 0)), FRAME_ERROR);
    ASSERT_OCCURRED_AS(tx.send(0, Bytes(9, 0)), FRAME_ERROR);
    tx.release();
}

TEST(CANFrames, BuildsTheFrameOfEachMessage) {
    auto m = binary_message(
        "status",
        {binary_field("v", 0)},
        synnax::library::CanIdentifier{.id = 0x1ABCDEF0, .extended = true}
    );
    x::json::Parser parser(x::json::json::object());
    const auto built = frames(parser, write_config({m}), false);
    ASSERT_TRUE(parser.ok());
    ASSERT_EQ(built.size(), 1);
    EXPECT_EQ(built[0].id, 0x1ABCDEF0);
    EXPECT_TRUE(built[0].extended);
    EXPECT_FALSE(built[0].fd);
    EXPECT_EQ(built[0].length, 1);
}

TEST(CANFrames, RejectsAFrameTheBusCannotCarry) {
    auto m = binary_message(
        "status",
        {binary_field("v", 0)},
        synnax::library::CanIdentifier{.id = 0x20, .fd = true}
    );
    x::json::Parser parser(x::json::json::object());
    const auto built = frames(parser, write_config({m}), false);
    ASSERT_FALSE(parser.ok());
    EXPECT_NE(
        parser.error().data.find(
            "message status: a CAN FD frame cannot be sent on a classic CAN bus"
        ),
        std::string::npos
    );
}

TEST(CANScanner, ReportsEveryChannelAndKeepsTrackedDevices) {
    Backends backends{
        {loopback::BACKEND,
         std::make_shared<loopback::Backend>(
             std::vector<std::string>{"can0", "can 1"}
         )},
        {"pcan", std::make_shared<Unavailable>(UNSUPPORTED_ERROR)},
    };
    synnax::task::Task task{.key = x::uuid::create(), .name = "scan"};
    task.rack = 4;
    Scanner scanner(task, std::make_shared<const Backends>(std::move(backends)));
    synnax::can::Properties props;
    props.backend = loopback::BACKEND;
    props.channel = "can0";
    props.bitrate = 250000;
    const synnax::device::Device tracked{
        .key = "tracked",
        .rack = 4,
        .make = MAKE,
        .name = "Engine bus",
        .properties = props.to_json().get<x::json::json::object_t>(),
    };
    const std::unordered_map<std::string, synnax::device::Device> devices{
        {tracked.key, tracked}
    };
    auto devs = ASSERT_NIL_P(scanner.scan({.devices = &devices}));
    ASSERT_EQ(devs.size(), 2);
    EXPECT_EQ(devs[0].key, "tracked");
    EXPECT_EQ(devs[0].name, "Engine bus");
    EXPECT_EQ(devs[0].properties.at("bitrate"), 250000);
    EXPECT_EQ(devs[0].status->message, "Channel available");
    EXPECT_EQ(devs[1].key, "can_4_loopback_can_1");
    EXPECT_EQ(devs[1].make, MAKE);
    EXPECT_EQ(devs[1].model, loopback::BACKEND);
    EXPECT_EQ(devs[1].location, "can 1");
    EXPECT_EQ(devs[1].rack, 4);
    EXPECT_EQ(devs[1].properties.at("backend"), loopback::BACKEND);
    EXPECT_EQ(devs[1].properties.at("channel"), "can 1");
    EXPECT_EQ(devs[1].status->variant, synnax::status::VARIANT_SUCCESS);
}

TEST(CANScanner, CreatesNoDeviceForASerialPort) {
    Backends backends{
        {synnax::can::BACKEND_SLCAN, std::make_shared<slcan::Backend>()},
    };
    synnax::task::Task task{.key = x::uuid::create(), .name = "scan"};
    Scanner scanner(task, std::make_shared<const Backends>(std::move(backends)));
    const std::unordered_map<std::string, synnax::device::Device> devices;
    EXPECT_TRUE(ASSERT_NIL_P(scanner.scan({.devices = &devices})).empty());
}
}
