// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <cstddef>
#include <cstdint>
#include <deque>
#include <map>
#include <memory>
#include <optional>
#include <span>
#include <string>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/channel/types.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/codec/framing/framing.h"
#include "driver/errors/errors.h"
#include "driver/icd/reader.h"
#include "driver/icd/receiver.h"
#include "driver/transport/errors.h"
#include "driver/transport/transport.h"

namespace driver::icd {
namespace {
const x::telem::TimeStamp ARRIVED(1000);

/// @brief Memory is a transport whose reads return the chunks that a spec queues, and
/// whose writes keep their bytes.
class Memory final : public transport::Transport {
    std::vector<std::uint8_t> held;

public:
    /// @brief chunks holds the result of each read to come. An empty chunk is a
    /// timeout.
    std::deque<std::pair<std::vector<std::uint8_t>, x::errors::Error>> chunks;
    /// @brief time is the arrival time of each chunk.
    x::telem::TimeStamp time = ARRIVED;
    /// @brief timeouts holds the timeout of each read.
    std::vector<x::telem::TimeSpan> timeouts;
    /// @brief writes holds the bytes of each write.
    std::vector<std::vector<std::uint8_t>> writes;
    /// @brief write_err is the error of each write, which then keeps no bytes.
    x::errors::Error write_err;

    std::pair<transport::Chunk, x::errors::Error>
    read(const x::telem::TimeSpan timeout) override {
        this->timeouts.push_back(timeout);
        if (this->chunks.empty()) return {{.time = this->time}, x::errors::NIL};
        auto [bytes, err] = std::move(this->chunks.front());
        this->chunks.pop_front();
        this->held = std::move(bytes);
        return {{.data = this->held, .time = this->time}, err};
    }

    x::errors::Error write(
        const std::span<const std::uint8_t> data,
        x::telem::TimeSpan /*timeout*/
    ) override {
        if (this->write_err) return this->write_err;
        this->writes.emplace_back(data.begin(), data.end());
        return x::errors::NIL;
    }

    void push(std::vector<std::uint8_t> bytes) {
        this->chunks.emplace_back(std::move(bytes), x::errors::NIL);
    }

    void push(const std::string &text) {
        this->push(std::vector<std::uint8_t>(text.begin(), text.end()));
    }
};

/// @brief returns a channel of dt with key that index indexes.
synnax::channel::Channel channel(
    const synnax::channel::Key key,
    const x::telem::DataType &dt,
    const synnax::channel::Key index
) {
    synnax::channel::Channel ch;
    ch.key = key;
    ch.data_type = dt;
    ch.index = index;
    ch.is_index = dt == x::telem::TIMESTAMP_T;
    return ch;
}

/// @brief Config holds an ICD with one message to read, and the channels of its fields.
struct Config {
    synnax::icd::ICD ic;
    synnax::icd::ReadMessage read;
    std::map<synnax::channel::Key, synnax::channel::Channel> channels;

    Config(
        const synnax::icd::Payload &payload,
        std::optional<synnax::icd::Framing> framing,
        const std::string &name = "device"
    ) {
        this->ic.key = x::uuid::create();
        this->ic.name = name;
        this->ic.framing = std::move(framing);
        synnax::icd::Message m;
        m.key = x::uuid::create();
        m.name = "frame";
        m.payload = payload;
        this->ic.messages = {m};
        this->read.message = m.key;
    }

    /// @brief adds f to the message and binds it to a float64 channel with key that
    /// index indexes.
    template<typename Payload, typename Field>
    void bind(
        Field f,
        const synnax::channel::Key key,
        const synnax::channel::Key index = 1
    ) {
        f.key = x::uuid::create();
        std::get<Payload>(this->ic.messages.front().payload).fields.push_back(f);
        this->read.bindings.push_back({.field = f.key, .channel = key});
        this->channels.insert_or_assign(key, channel(key, x::telem::FLOAT64_T, index));
    }

    /// @brief returns the entry that reads the message, and sends poll.
    [[nodiscard]] synnax::icd::ReadEntry
    entry(std::optional<synnax::icd::Poll> poll = std::nullopt) const {
        return {.icd = this->ic.key, .messages = {this->read}, .poll = std::move(poll)};
    }
};

/// @brief returns the receiver of the entry of cfg on a stream.
Receiver create(
    const Config &cfg,
    std::optional<synnax::icd::Poll> poll = std::nullopt,
    transport::Clock now = std::chrono::steady_clock::now
) {
    return ASSERT_NIL_P(
        Receiver::create(
            {cfg.entry(std::move(poll))},
            {cfg.ic},
            cfg.channels,
            false,
            std::move(now)
        )
    );
}

/// @brief returns the error of creating a receiver of entries on a stream.
x::errors::Error refused(
    const std::vector<synnax::icd::ReadEntry> &entries,
    const std::vector<synnax::icd::ICD> &icds,
    const std::map<synnax::channel::Key, synnax::channel::Channel> &channels = {}
) {
    return Receiver::create(entries, icds, channels, false).second;
}

/// @brief returns an unsigned binary field named name in the byte at offset.
synnax::icd::UnsignedBinaryField
byte_field(const std::string &name, const std::uint16_t offset) {
    synnax::icd::UnsignedBinaryField f;
    f.name = name;
    f.start_bit = offset * 8;
    return f;
}

/// @brief returns the config of two-byte frames whose second byte v writes to channel
/// 2, and whose first byte is free.
Config pairs(
    std::optional<synnax::icd::Framing> framing = synnax::icd::FixedFraming{.length = 2}
) {
    Config cfg{synnax::icd::BinaryPayload{}, std::move(framing)};
    cfg.bind<synnax::icd::BinaryPayload>(byte_field("v", 1), 2);
    return cfg;
}

/// @brief returns the config of lines that start with prefix and hold the value of
/// channel key after a comma, on index channel index.
Config lines(
    const std::string &prefix = "",
    const synnax::channel::Key key = 2,
    const synnax::channel::Key index = 1,
    const std::string &name = "device"
) {
    Config cfg{
        synnax::icd::TextPayload{.prefix = prefix},
        synnax::icd::DelimitedFraming{},
        name
    };
    synnax::icd::DelimitedTextField v;
    v.name = "v";
    v.position = 1;
    cfg.bind<synnax::icd::TextPayload>(v, key, index);
    return cfg;
}

/// @brief returns a receiver of two-byte frames.
Receiver fixed() {
    return create(pairs());
}

/// @brief Steady is a steady clock that a spec moves by hand.
struct Steady {
    std::chrono::steady_clock::time_point time;

    [[nodiscard]] transport::Clock clock() {
        return [this] { return this->time; };
    }
};

/// @brief returns a receiver of two-byte frames, which sends the two-byte query at 10
/// Hz on clock.
Receiver polling(Steady &clock) {
    return create(
        pairs(),
        synnax::icd::Poll{.query = {0xAA, 0x01}, .rate = x::telem::Rate(10)},
        clock.clock()
    );
}

std::vector<std::int64_t> stamps(const std::vector<std::int64_t> &offsets) {
    std::vector<std::int64_t> out;
    out.reserve(offsets.size());
    for (const auto o: offsets)
        out.push_back(ARRIVED.nanoseconds() + o);
    return out;
}
}

/// @brief it should read every frame of a chunk, and stamp each one 1 ns after the
/// last.
TEST(Receiver, ReadsEveryFrameOfChunk) {
    auto r = fixed();
    Memory t;
    t.push(std::vector<std::uint8_t>{0, 7, 0, 8, 0, 9});
    x::telem::Frame fr;
    const auto res = r.read(t, fr);
    ASSERT_NIL(res.error);
    ASSERT_TRUE(res.warning.empty());
    ASSERT_EQ(*fr.channels, (std::vector<std::uint32_t>{1, 2}));
    ASSERT_EQ(fr.series->at(0).values<std::int64_t>(), stamps({0, 1, 2}));
    ASSERT_EQ(fr.series->at(1).values<double>(), (std::vector{7.0, 8.0, 9.0}));
}

/// @brief it should stamp a frame 1 ns after the last frame when its bytes arrived
/// earlier.
TEST(Receiver, StampsFrameAfterLastFrame) {
    auto r = fixed();
    Memory t;
    t.push(std::vector<std::uint8_t>{0, 7});
    t.push(std::vector<std::uint8_t>{0, 8});
    x::telem::Frame first;
    ASSERT_NIL(r.read(t, first).error);
    t.time = ARRIVED - x::telem::SECOND;
    x::telem::Frame second;
    ASSERT_NIL(r.read(t, second).error);
    ASSERT_EQ(second.series->at(0).values<std::int64_t>(), stamps({1}));
}

/// @brief it should keep each index group aligned with its own index when the
/// multiplexors select the groups in different frames.
TEST(Receiver, KeepsGroupsAlignedWithTheirIndexes) {
    Config cfg{synnax::icd::BinaryPayload{}, synnax::icd::FixedFraming{.length = 2}};
    auto mux = byte_field("mux", 0);
    mux.key = x::uuid::create();
    std::get<synnax::icd::BinaryPayload>(cfg.ic.messages.front().payload)
        .fields.emplace_back(mux);
    auto a = byte_field("a", 1);
    a.multiplexor = mux.key;
    a.multiplex_values = {1};
    cfg.bind<synnax::icd::BinaryPayload>(a, 2, 1);
    auto b = byte_field("b", 1);
    b.multiplexor = mux.key;
    b.multiplex_values = {2};
    cfg.bind<synnax::icd::BinaryPayload>(b, 3, 4);
    auto r = create(cfg);
    Memory t;
    t.push(std::vector<std::uint8_t>{1, 7, 2, 9, 1, 8});
    x::telem::Frame fr;
    ASSERT_NIL(r.read(t, fr).error);
    ASSERT_EQ(*fr.channels, (std::vector<std::uint32_t>{1, 2, 4, 3}));
    ASSERT_EQ(fr.series->at(0).values<std::int64_t>(), stamps({0, 2}));
    ASSERT_EQ(fr.series->at(1).values<double>(), (std::vector{7.0, 8.0}));
    ASSERT_EQ(fr.series->at(2).values<std::int64_t>(), stamps({1}));
    ASSERT_EQ(fr.series->at(3).values<double>(), (std::vector{9.0}));
}

/// @brief it should leave the frames past BATCH for the next read, which reads them
/// without waiting for the transport.
TEST(Receiver, LeavesFramesPastBatchForNextRead) {
    auto r = fixed();
    Memory t;
    std::vector<std::uint8_t> chunk;
    for (std::size_t i = 0; i <= BATCH; i++)
        chunk.insert(chunk.end(), {0, static_cast<std::uint8_t>(i % 256)});
    t.push(chunk);
    x::telem::Frame first;
    ASSERT_NIL(r.read(t, first).error);
    ASSERT_EQ(first.series->at(1).size(), BATCH);
    x::telem::Frame second;
    ASSERT_NIL(r.read(t, second).error);
    ASSERT_EQ(t.timeouts.size(), 1U);
    ASSERT_EQ(
        second.series->at(0).values<std::int64_t>(),
        stamps({static_cast<std::int64_t>(BATCH)})
    );
    ASSERT_EQ(
        second.series->at(1).values<double>(),
        (std::vector{static_cast<double>(BATCH % 256)})
    );
}

/// @brief it should add nothing to the frame when no bytes arrive.
TEST(Receiver, AddsNothingWithoutBytes) {
    auto r = fixed();
    Memory t;
    x::telem::Frame fr;
    const auto res = r.read(t, fr);
    ASSERT_NIL(res.error);
    ASSERT_TRUE(res.warning.empty());
    ASSERT_TRUE(fr.empty());
}

/// @brief it should drop a partial frame with a warning when no bytes arrive, so that
/// the next frame starts in step.
TEST(Receiver, DropsPartialFrameWhenIdle) {
    auto r = fixed();
    Memory t;
    t.push(std::vector<std::uint8_t>{0});
    x::telem::Frame fr;
    ASSERT_NIL(r.read(t, fr).error);
    const auto res = r.read(t, fr);
    ASSERT_NIL(res.error);
    ASSERT_EQ(res.warning.message, "Dropped invalid data");
    ASSERT_FALSE(res.warning.description.empty());
    t.push(std::vector<std::uint8_t>{0, 7});
    ASSERT_NIL(r.read(t, fr).error);
    ASSERT_EQ(fr.series->at(1).values<double>(), (std::vector{7.0}));
}

/// @brief it should warn about a frame that the framing drops, and read the frames
/// after it.
TEST(Receiver, ReadsFramesAfterInvalidFrame) {
    auto r = create(pairs(synnax::icd::CobsFraming{}));
    Memory t;
    t.push(std::vector<std::uint8_t>{0x05, 0x01, 0x00, 0x01, 0x02, 0x07, 0x00});
    x::telem::Frame fr;
    const auto res = r.read(t, fr);
    ASSERT_NIL(res.error);
    ASSERT_EQ(res.warning.message, "Dropped invalid data");
    ASSERT_EQ(res.warning.description, "dropped a frame with invalid COBS bytes");
    ASSERT_EQ(fr.series->at(1).values<double>(), (std::vector{7.0}));
}

/// @brief it should return the error of the transport, and drop the bytes it holds.
TEST(Receiver, DropsBytesOnTransportError) {
    auto r = fixed();
    Memory t;
    t.push(std::vector<std::uint8_t>{0});
    t.chunks.emplace_back(
        std::vector<std::uint8_t>{},
        x::errors::Error(transport::UNREACHABLE_ERROR, "the port closed")
    );
    t.push(std::vector<std::uint8_t>{0, 7});
    x::telem::Frame fr;
    ASSERT_NIL(r.read(t, fr).error);
    ASSERT_OCCURRED_AS(r.read(t, fr).error, transport::UNREACHABLE_ERROR);
    ASSERT_NIL(r.read(t, fr).error);
    ASSERT_EQ(fr.series->at(1).values<double>(), (std::vector{7.0}));
}

/// @brief it should warn about a frame that it cannot decode, and read the frames after
/// it.
TEST(Receiver, WarnsAboutFrameItCannotDecode) {
    auto r = create(lines());
    Memory t;
    t.push("oven\noven,21.5\n");
    x::telem::Frame fr;
    const auto res = r.read(t, fr);
    ASSERT_NIL(res.error);
    ASSERT_EQ(res.warning.message, "Failed to decode message");
    ASSERT_EQ(res.warning.description, "field v: no value at position 1");
    ASSERT_EQ(fr.series->at(1).values<double>(), (std::vector{21.5}));
}

/// @brief it should write the other index groups of a frame with a field that does not
/// decode, and warn about the field.
TEST(Receiver, WarnsAboutFieldItCannotDecode) {
    Config cfg{synnax::icd::JSONPayload{}, synnax::icd::DelimitedFraming{}};
    synnax::icd::FloatJSONField v;
    v.name = "v";
    v.pointer = "/v";
    cfg.bind<synnax::icd::JSONPayload>(v, 2, 1);
    auto w = v;
    w.name = "w";
    w.pointer = "/w";
    cfg.bind<synnax::icd::JSONPayload>(w, 3, 4);
    auto r = create(cfg);
    Memory t;
    t.push(std::string(R"({"v": 1, "w": {}})") + "\n");
    x::telem::Frame fr;
    const auto res = r.read(t, fr);
    ASSERT_NIL(res.error);
    ASSERT_EQ(res.warning.message, "Failed to decode message");
    ASSERT_EQ(
        res.warning.description,
        "field w: expected a number or boolean, got an object"
    );
    ASSERT_EQ(*fr.channels, (std::vector<std::uint32_t>{1, 2}));
    ASSERT_EQ(fr.series->at(1).values<double>(), (std::vector{1.0}));
}

/// @brief it should skip a frame that belongs to no message without a warning.
TEST(Receiver, SkipsFrameOfNoMessage) {
    auto r = create(lines("T"));
    Memory t;
    t.push("X,1\nT,2\n");
    x::telem::Frame fr;
    const auto res = r.read(t, fr);
    ASSERT_NIL(res.error);
    ASSERT_TRUE(res.warning.empty());
    ASSERT_EQ(fr.series->at(1).values<double>(), (std::vector{2.0}));
}
/// @brief it should send the query at its first read and then once per period.
TEST(Receiver, SendsQueryAtRate) {
    Steady clock;
    auto r = polling(clock);
    Memory t;
    x::telem::Frame fr;
    const std::vector<std::vector<std::uint8_t>> one{{0xAA, 0x01}};
    ASSERT_NIL(r.read(t, fr).error);
    ASSERT_EQ(t.writes, one);
    clock.time += std::chrono::milliseconds(50);
    ASSERT_NIL(r.read(t, fr).error);
    ASSERT_EQ(t.writes, one);
    clock.time += std::chrono::milliseconds(50);
    ASSERT_NIL(r.read(t, fr).error);
    ASSERT_EQ(t.writes.size(), 2U);
}

/// @brief it should wait for bytes no later than the time of the next query.
TEST(Receiver, WaitsNoLaterThanNextQuery) {
    Steady clock;
    auto r = polling(clock);
    Memory t;
    x::telem::Frame fr;
    ASSERT_NIL(r.read(t, fr).error);
    clock.time += std::chrono::milliseconds(80);
    ASSERT_NIL(r.read(t, fr).error);
    ASSERT_EQ(t.timeouts, (std::vector{READ_TIMEOUT, 20 * x::telem::MILLISECOND}));
}

/// @brief it should send one query after a stall, not one for each missed period.
TEST(Receiver, SkipsMissedQueries) {
    Steady clock;
    auto r = polling(clock);
    Memory t;
    x::telem::Frame fr;
    ASSERT_NIL(r.read(t, fr).error);
    clock.time += std::chrono::milliseconds(350);
    ASSERT_NIL(r.read(t, fr).error);
    ASSERT_NIL(r.read(t, fr).error);
    ASSERT_EQ(t.writes.size(), 2U);
    ASSERT_EQ(t.timeouts.back(), READ_TIMEOUT);
}

/// @brief it should send the query in the framing of the reader.
TEST(Receiver, FramesQuery) {
    Steady clock;
    auto r = create(
        lines(),
        synnax::icd::Poll{.query = {'?'}, .rate = x::telem::Rate(10)},
        clock.clock()
    );
    Memory t;
    x::telem::Frame fr;
    ASSERT_NIL(r.read(t, fr).error);
    ASSERT_EQ(t.writes, (std::vector<std::vector<std::uint8_t>>{{'?', '\n'}}));
}

/// @brief it should reject a query that the framing cannot hold.
TEST(Receiver, RejectsQueryThatFramingCannotHold) {
    const auto cfg = pairs();
    const auto err = refused(
        {cfg.entry(synnax::icd::Poll{.query = {1, 2, 3}, .rate = x::telem::Rate(10)})},
        {cfg.ic},
        cfg.channels
    );
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(
        err.data,
        "entry 0: the framing cannot hold the poll query: the frame has 3 bytes, not "
        "the fixed length of 2"
    );
}

/// @brief it should refuse a poll rate that is not above 0 Hz, or whose period is
/// shorter than 1 ns.
TEST(Receiver, RejectsPollRateOutOfRange) {
    for (const auto &[hz, text]: std::vector<std::pair<double, std::string>>{
             {0, "0 Hz"},
             {-1, "-1 Hz"},
             {2e9, "2e+09 Hz"},
         }) {
        const auto cfg = pairs();
        const auto err = refused(
            {cfg.entry(synnax::icd::Poll{.query = {1, 2}, .rate = x::telem::Rate(hz)})},
            {cfg.ic},
            cfg.channels
        );
        ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
        ASSERT_EQ(
            err.data,
            "entry 0: the poll rate must be above 0 Hz and at most 1 GHz, not " + text
        );
    }
}

/// @brief it should keep a partial frame until reads have waited READ_TIMEOUT in all,
/// even when the next query cuts each wait short.
TEST(Receiver, KeepsPartialFrameUntilReadTimeout) {
    Steady clock;
    auto r = create(
        pairs(),
        synnax::icd::Poll{.query = {0xAA, 0x01}, .rate = x::telem::Rate(50)},
        clock.clock()
    );
    Memory t;
    t.push(std::vector<std::uint8_t>{0});
    x::telem::Frame fr;
    ASSERT_NIL(r.read(t, fr).error);
    clock.time += std::chrono::milliseconds(20);
    ASSERT_TRUE(r.read(t, fr).warning.empty());
    clock.time += std::chrono::milliseconds(20);
    ASSERT_TRUE(r.read(t, fr).warning.empty());
    clock.time += std::chrono::milliseconds(20);
    ASSERT_EQ(r.read(t, fr).warning.message, "Dropped invalid data");
}

/// @brief it should return the error of a query that fails to send, and drop the bytes
/// in the framer.
TEST(Receiver, DropsBytesWhenQueryFails) {
    Steady clock;
    auto r = polling(clock);
    Memory t;
    t.push(std::vector<std::uint8_t>{0});
    x::telem::Frame fr;
    ASSERT_NIL(r.read(t, fr).error);
    t.write_err = x::errors::Error(transport::UNREACHABLE_ERROR, "the port closed");
    clock.time += std::chrono::milliseconds(100);
    ASSERT_OCCURRED_AS(r.read(t, fr).error, transport::UNREACHABLE_ERROR);
    t.write_err = x::errors::NIL;
    t.push(std::vector<std::uint8_t>{0, 7});
    clock.time += std::chrono::milliseconds(100);
    ASSERT_NIL(r.read(t, fr).error);
    ASSERT_EQ(fr.series->at(1).values<double>(), (std::vector{7.0}));
}

/// @brief returns channels a merged with channels b.
std::map<synnax::channel::Key, synnax::channel::Channel> merged(
    std::map<synnax::channel::Key, synnax::channel::Channel> a,
    const std::map<synnax::channel::Key, synnax::channel::Channel> &b
) {
    a.insert(b.begin(), b.end());
    return a;
}

/// @brief it should give each frame to the entry with a message that matches it.
TEST(Receiver, GivesFrameToEntryWithMatchingMessage) {
    const auto a = lines("A", 2, 1, "first");
    const auto b = lines("B", 3, 4, "second");
    auto r = ASSERT_NIL_P(
        Receiver::create(
            {a.entry(), b.entry()},
            {a.ic, b.ic},
            merged(a.channels, b.channels),
            false
        )
    );
    Memory t;
    t.push("B,2\nA,1\n");
    x::telem::Frame fr;
    const auto res = r.read(t, fr);
    ASSERT_NIL(res.error);
    ASSERT_TRUE(res.warning.empty());
    ASSERT_EQ(*fr.channels, (std::vector<std::uint32_t>{4, 3, 1, 2}));
    ASSERT_EQ(fr.series->at(1).values<double>(), (std::vector{2.0}));
    ASSERT_EQ(fr.series->at(3).values<double>(), (std::vector{1.0}));
}

/// @brief it should give a frame that two entries match to the first one.
TEST(Receiver, GivesFrameToFirstMatchingEntry) {
    const auto a = lines("T", 2, 1, "first");
    const auto b = lines("T", 3, 4, "second");
    auto r = ASSERT_NIL_P(
        Receiver::create(
            {a.entry(), b.entry()},
            {a.ic, b.ic},
            merged(a.channels, b.channels),
            false
        )
    );
    Memory t;
    t.push("T,5\n");
    x::telem::Frame fr;
    ASSERT_NIL(r.read(t, fr).error);
    ASSERT_EQ(*fr.channels, (std::vector<std::uint32_t>{1, 2}));
    ASSERT_EQ(fr.series->at(1).values<double>(), (std::vector{5.0}));
}

/// @brief it should give a frame that an earlier entry cannot decode to a later entry
/// that matches it, with no warning.
TEST(Receiver, GivesFrameThatEarlierEntryCannotDecodeToLaterEntry) {
    Config a{synnax::icd::JSONPayload{}, synnax::icd::DelimitedFraming{}, "first"};
    synnax::icd::FloatJSONField v;
    v.name = "v";
    v.pointer = "/v";
    a.bind<synnax::icd::JSONPayload>(v, 2, 1);
    const auto b = lines("T", 3, 4, "second");
    auto r = ASSERT_NIL_P(
        Receiver::create(
            {a.entry(), b.entry()},
            {a.ic, b.ic},
            merged(a.channels, b.channels),
            false
        )
    );
    Memory t;
    t.push("T,5\n");
    x::telem::Frame fr;
    const auto res = r.read(t, fr);
    ASSERT_NIL(res.error);
    ASSERT_TRUE(res.warning.empty());
    ASSERT_EQ(*fr.channels, (std::vector<std::uint32_t>{4, 3}));
    ASSERT_EQ(fr.series->at(1).values<double>(), (std::vector{5.0}));
}

/// @brief it should read two entries that read different messages of one ICD.
TEST(Receiver, ReadsEntriesOfOneICDWithDifferentMessages) {
    auto a = lines("A", 2, 1);
    const auto b = lines("B", 3, 4);
    a.ic.messages.push_back(b.ic.messages.front());
    const synnax::icd::ReadEntry second{.icd = a.ic.key, .messages = {b.read}};
    auto r = ASSERT_NIL_P(
        Receiver::create(
            {a.entry(), second},
            {a.ic},
            merged(a.channels, b.channels),
            false
        )
    );
    Memory t;
    t.push("B,2\n");
    x::telem::Frame fr;
    ASSERT_NIL(r.read(t, fr).error);
    ASSERT_EQ(*fr.channels, (std::vector<std::uint32_t>{4, 3}));
    ASSERT_EQ(fr.series->at(1).values<double>(), (std::vector{2.0}));
}

/// @brief it should send the query of each entry on its own period, and wait for bytes
/// no later than the next query of any entry.
TEST(Receiver, SendsQueryOfEachEntryOnItsOwnPeriod) {
    const auto a = lines("A", 2, 1, "first");
    const auto b = lines("B", 3, 4, "second");
    Steady clock;
    auto r = ASSERT_NIL_P(
        Receiver::create(
            {a.entry(synnax::icd::Poll{.query = {'a'}, .rate = x::telem::Rate(10)}),
             b.entry(synnax::icd::Poll{.query = {'b'}, .rate = x::telem::Rate(4)})},
            {a.ic, b.ic},
            merged(a.channels, b.channels),
            false,
            clock.clock()
        )
    );
    Memory t;
    x::telem::Frame fr;
    ASSERT_NIL(r.read(t, fr).error);
    clock.time += std::chrono::milliseconds(100);
    ASSERT_NIL(r.read(t, fr).error);
    clock.time += std::chrono::milliseconds(140);
    ASSERT_NIL(r.read(t, fr).error);
    clock.time += std::chrono::milliseconds(10);
    ASSERT_NIL(r.read(t, fr).error);
    const std::vector<std::vector<std::uint8_t>> sent{
        {'a', '\n'},
        {'b', '\n'},
        {'a', '\n'},
        {'a', '\n'},
        {'b', '\n'},
    };
    ASSERT_EQ(t.writes, sent);
    ASSERT_EQ(
        t.timeouts,
        (std::vector{
            READ_TIMEOUT,
            READ_TIMEOUT,
            10 * x::telem::MILLISECOND,
            READ_TIMEOUT,
        })
    );
}

/// @brief it should refuse two entries that read one message of one ICD, because a
/// frame of it could belong to either.
TEST(Receiver, RejectsEntriesThatReadOneMessage) {
    const auto cfg = pairs();
    const auto err = refused({cfg.entry(), cfg.entry()}, {cfg.ic}, cfg.channels);
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(err.data, "entries 0 and 1 both read message frame of ICD device");
}

/// @brief it should refuse a stream entry whose ICD has no framing.
TEST(Receiver, RejectsStreamICDWithoutFraming) {
    const auto cfg = pairs(std::nullopt);
    const auto err = refused({cfg.entry()}, {cfg.ic}, cfg.channels);
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(err.data, "ICD device has no framing to split the stream with");
}

/// @brief it should refuse entries whose ICDs frame the stream differently.
TEST(Receiver, RejectsICDsWithDifferentFramings) {
    const auto a = lines("A", 2, 1, "first");
    auto b = lines("B", 3, 4, "second");
    b.ic.framing = synnax::icd::DelimitedFraming{.delimiter = "\r\n"};
    const auto err = refused(
        {a.entry(), b.entry()},
        {a.ic, b.ic},
        merged(a.channels, b.channels)
    );
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(err.data, "ICDs first and second frame the stream differently");
}

/// @brief it should refuse an entry whose ICD does not exist.
TEST(Receiver, RejectsMissingICD) {
    const auto cfg = pairs();
    const auto err = refused({cfg.entry()}, {}, cfg.channels);
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(err.data, "entry 0: ICD " + cfg.ic.key.to_string() + " does not exist");
}

/// @brief it should name the entry whose reader fails to create.
TEST(Receiver, NamesEntryWhoseReaderFails) {
    const auto cfg = pairs();
    const auto err = refused({cfg.entry()}, {cfg.ic});
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(err.data, "entry 0: message frame: channel 2 does not exist");
}

/// @brief it should read each datagram with no framing as one message, and send the
/// query unframed.
TEST(Receiver, ReadsEachDatagramAsOneMessage) {
    const auto cfg = pairs(std::nullopt);
    Steady clock;
    auto r = ASSERT_NIL_P(
        Receiver::create(
            {cfg.entry(synnax::icd::Poll{.query = {0xAA}, .rate = x::telem::Rate(10)})},
            {cfg.ic},
            cfg.channels,
            true,
            clock.clock()
        )
    );
    Memory t;
    t.push(std::vector<std::uint8_t>{0, 7});
    t.push(std::vector<std::uint8_t>{0, 8});
    x::telem::Frame fr;
    ASSERT_NIL(r.read(t, fr).error);
    ASSERT_NIL(r.read(t, fr).error);
    ASSERT_EQ(t.writes, (std::vector<std::vector<std::uint8_t>>{{0xAA}}));
    ASSERT_EQ(fr.series->at(1).values<double>(), (std::vector{7.0, 8.0}));
}

/// @brief it should frame the bytes inside each datagram, and drop a partial frame at
/// the end of a datagram with a warning.
TEST(Receiver, FramesInsideEachDatagram) {
    const auto cfg = pairs();
    auto r = ASSERT_NIL_P(
        Receiver::create({cfg.entry()}, {cfg.ic}, cfg.channels, true)
    );
    Memory t;
    t.push(std::vector<std::uint8_t>{0, 7, 0});
    t.push(std::vector<std::uint8_t>{0, 9});
    x::telem::Frame fr;
    const auto res = r.read(t, fr);
    ASSERT_NIL(res.error);
    ASSERT_EQ(res.warning.message, "Dropped invalid data");
    ASSERT_NIL(r.read(t, fr).error);
    ASSERT_EQ(fr.series->at(1).values<double>(), (std::vector{7.0, 9.0}));
}

/// @brief it should read every frame of a datagram that holds more than BATCH frames
/// before it drops the partial frame at its end.
TEST(Receiver, ReadsDatagramPastBatch) {
    const auto cfg = pairs();
    auto r = ASSERT_NIL_P(
        Receiver::create({cfg.entry()}, {cfg.ic}, cfg.channels, true)
    );
    Memory t;
    std::vector<std::uint8_t> datagram;
    for (std::size_t i = 0; i <= BATCH; i++)
        datagram.insert(datagram.end(), {0, 7});
    datagram.push_back(0);
    t.push(datagram);
    x::telem::Frame first;
    const auto res = r.read(t, first);
    ASSERT_NIL(res.error);
    ASSERT_TRUE(res.warning.empty());
    ASSERT_EQ(first.series->at(1).size(), BATCH);
    x::telem::Frame second;
    ASSERT_EQ(r.read(t, second).warning.message, "Dropped invalid data");
    ASSERT_EQ(second.series->at(1).values<double>(), (std::vector{7.0}));
}

/// @brief it should read nothing and send nothing with no entries.
TEST(Receiver, ReadsNothingWithoutEntries) {
    auto r = ASSERT_NIL_P(Receiver::create({}, {}, {}, false));
    Memory t;
    t.push("1,2\n");
    x::telem::Frame fr;
    ASSERT_NIL(r.read(t, fr).error);
    ASSERT_NIL(r.read(t, fr).error);
    ASSERT_TRUE(fr.empty());
    ASSERT_TRUE(t.writes.empty());
}
}
