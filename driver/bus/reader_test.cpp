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

#include "driver/bus/reader.h"
#include "driver/codec/framing/framing.h"
#include "driver/errors/errors.h"
#include "driver/icd/reader.h"
#include "driver/transport/errors.h"
#include "driver/transport/transport.h"

namespace driver::bus {
namespace {
const x::telem::TimeStamp ARRIVED(1000);

/// @brief returns the framer of framing.
std::unique_ptr<codec::framing::Framer> framer(const synnax::icd::Framing &framing) {
    return ASSERT_NIL_P(codec::framing::create(framing));
}

/// @brief returns a framer of 2-byte frames.
std::unique_ptr<codec::framing::Framer> pair_framer() {
    return framer(synnax::icd::FixedFraming{.length = 2});
}

/// @brief returns a framer of lines.
std::unique_ptr<codec::framing::Framer> line_framer() {
    return framer(synnax::icd::DelimitedFraming{});
}

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

/// @brief Config holds one message to read, and the channels of its fields.
struct Config {
    synnax::icd::Message message;
    synnax::icd::ReadMessage read;
    std::map<synnax::channel::Key, synnax::channel::Channel> channels;

    explicit Config(const synnax::icd::Payload &payload) {
        this->message.key = x::uuid::create();
        this->message.name = "frame";
        this->message.payload = payload;
        this->read.message = this->message.key;
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
        std::get<Payload>(this->message.payload).fields.push_back(f);
        this->read.bindings.push_back({.field = f.key, .channel = key});
        this->channels.insert_or_assign(key, channel(key, x::telem::FLOAT64_T, index));
    }

    [[nodiscard]] icd::Reader create() const {
        synnax::icd::ICD ic;
        ic.messages = {this->message};
        return ASSERT_NIL_P(icd::Reader::create({this->read}, ic, this->channels));
    }
};

/// @brief returns an unsigned binary field named name in the byte at offset.
synnax::icd::UnsignedBinaryField
byte_field(const std::string &name, const std::uint16_t offset) {
    synnax::icd::UnsignedBinaryField f;
    f.name = name;
    f.start_bit = offset * 8;
    return f;
}

/// @brief returns a reader of two-byte frames whose second byte v writes to channel 2,
/// and whose first byte is free.
Reader fixed() {
    Config cfg{synnax::icd::BinaryPayload{}};
    cfg.bind<synnax::icd::BinaryPayload>(byte_field("v", 1), 2);
    return ASSERT_NIL_P(Reader::create(pair_framer(), cfg.create()));
}

/// @brief returns a reader of lines that start with prefix and hold the value of
/// channel 2 after a comma.
Reader lines(const std::string &prefix = "") {
    Config cfg{synnax::icd::TextPayload{.prefix = prefix}};
    synnax::icd::DelimitedTextField v;
    v.name = "v";
    v.position = 1;
    cfg.bind<synnax::icd::TextPayload>(v, 2);
    return ASSERT_NIL_P(Reader::create(line_framer(), cfg.create()));
}

/// @brief Steady is a steady clock that a spec moves by hand.
struct Steady {
    std::chrono::steady_clock::time_point time;

    [[nodiscard]] Clock clock() {
        return [this] { return this->time; };
    }
};

/// @brief returns a reader of two-byte frames, which sends the two-byte query at 10
/// Hz on clock.
Reader polling(Steady &clock) {
    Config cfg{synnax::icd::BinaryPayload{}};
    cfg.bind<synnax::icd::BinaryPayload>(byte_field("v", 1), 2);
    return ASSERT_NIL_P(
        Reader::create(
            pair_framer(),
            cfg.create(),
            Poll{.query = {0xAA, 0x01}, .rate = x::telem::Rate(10)},
            clock.clock()
        )
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
TEST(Reader, ReadsEveryFrameOfChunk) {
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
TEST(Reader, StampsFrameAfterLastFrame) {
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
TEST(Reader, KeepsGroupsAlignedWithTheirIndexes) {
    Config cfg{synnax::icd::BinaryPayload{}};
    auto mux = byte_field("mux", 0);
    mux.key = x::uuid::create();
    std::get<synnax::icd::BinaryPayload>(cfg.message.payload).fields.emplace_back(mux);
    auto a = byte_field("a", 1);
    a.multiplexor = mux.key;
    a.multiplex_values = {1};
    cfg.bind<synnax::icd::BinaryPayload>(a, 2, 1);
    auto b = byte_field("b", 1);
    b.multiplexor = mux.key;
    b.multiplex_values = {2};
    cfg.bind<synnax::icd::BinaryPayload>(b, 3, 4);
    auto r = ASSERT_NIL_P(Reader::create(pair_framer(), cfg.create()));
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
TEST(Reader, LeavesFramesPastBatchForNextRead) {
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
TEST(Reader, AddsNothingWithoutBytes) {
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
TEST(Reader, DropsPartialFrameWhenIdle) {
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
TEST(Reader, ReadsFramesAfterInvalidFrame) {
    Config cfg{synnax::icd::BinaryPayload{}};
    cfg.bind<synnax::icd::BinaryPayload>(byte_field("v", 1), 2);
    auto r = ASSERT_NIL_P(
        Reader::create(framer(synnax::icd::CobsFraming{}), cfg.create())
    );
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
TEST(Reader, DropsBytesOnTransportError) {
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

/// @brief it should warn about a frame that it cannot decode, and read the frames
/// after it.
TEST(Reader, WarnsAboutFrameItCannotDecode) {
    auto r = lines();
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
TEST(Reader, WarnsAboutFieldItCannotDecode) {
    Config cfg{synnax::icd::JSONPayload{}};
    synnax::icd::FloatJSONField v;
    v.name = "v";
    v.pointer = "/v";
    cfg.bind<synnax::icd::JSONPayload>(v, 2, 1);
    auto w = v;
    w.name = "w";
    w.pointer = "/w";
    cfg.bind<synnax::icd::JSONPayload>(w, 3, 4);
    auto r = ASSERT_NIL_P(Reader::create(line_framer(), cfg.create()));
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
TEST(Reader, SkipsFrameOfNoMessage) {
    auto r = lines("T");
    Memory t;
    t.push("X,1\nT,2\n");
    x::telem::Frame fr;
    const auto res = r.read(t, fr);
    ASSERT_NIL(res.error);
    ASSERT_TRUE(res.warning.empty());
    ASSERT_EQ(fr.series->at(1).values<double>(), (std::vector{2.0}));
}
/// @brief it should send the query at its first read and then once per period.
TEST(Reader, SendsQueryAtRate) {
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
TEST(Reader, WaitsNoLaterThanNextQuery) {
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
TEST(Reader, SkipsMissedQueries) {
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
TEST(Reader, FramesQuery) {
    Config cfg{synnax::icd::TextPayload{}};
    synnax::icd::DelimitedTextField v;
    v.name = "v";
    cfg.bind<synnax::icd::TextPayload>(v, 2);
    Steady clock;
    auto r = ASSERT_NIL_P(
        Reader::create(
            line_framer(),
            cfg.create(),
            Poll{.query = {'?'}, .rate = x::telem::Rate(10)},
            clock.clock()
        )
    );
    Memory t;
    x::telem::Frame fr;
    ASSERT_NIL(r.read(t, fr).error);
    ASSERT_EQ(t.writes, (std::vector<std::vector<std::uint8_t>>{{'?', '\n'}}));
}

/// @brief it should reject a query that the framing cannot hold.
TEST(Reader, RejectsQueryThatFramingCannotHold) {
    Config cfg{synnax::icd::BinaryPayload{}};
    cfg.bind<synnax::icd::BinaryPayload>(byte_field("v", 1), 2);
    const auto err = Reader::create(
                         pair_framer(),
                         cfg.create(),
                         Poll{.query = {1, 2, 3}, .rate = x::telem::Rate(10)}
    )
                         .second;
    ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
    ASSERT_EQ(
        err.data,
        "the framing cannot hold the poll query: the frame has 3 bytes, not the fixed "
        "length of 2"
    );
}

/// @brief it should refuse a poll rate that is not above 0 Hz, or whose period is
/// shorter than 1 ns.
TEST(Reader, RejectsPollRateOutOfRange) {
    for (const auto &[hz, text]: std::vector<std::pair<double, std::string>>{
             {0, "0 Hz"},
             {-1, "-1 Hz"},
             {2e9, "2e+09 Hz"},
         }) {
        Config cfg{synnax::icd::BinaryPayload{}};
        cfg.bind<synnax::icd::BinaryPayload>(byte_field("v", 1), 2);
        const auto err = Reader::create(
                             pair_framer(),
                             cfg.create(),
                             Poll{.query = {1, 2}, .rate = x::telem::Rate(hz)}
        )
                             .second;
        ASSERT_OCCURRED_AS(err, driver::errors::CONFIGURATION_ERROR);
        ASSERT_EQ(
            err.data,
            "the poll rate must be above 0 Hz and at most 1 GHz, not " + text
        );
    }
}

/// @brief it should keep a partial frame until reads have waited READ_TIMEOUT in all,
/// even when the next query cuts each wait short.
TEST(Reader, KeepsPartialFrameUntilReadTimeout) {
    Config cfg{synnax::icd::BinaryPayload{}};
    cfg.bind<synnax::icd::BinaryPayload>(byte_field("v", 1), 2);
    Steady clock;
    auto r = ASSERT_NIL_P(
        Reader::create(
            pair_framer(),
            cfg.create(),
            Poll{.query = {0xAA, 0x01}, .rate = x::telem::Rate(50)},
            clock.clock()
        )
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
TEST(Reader, DropsBytesWhenQueryFails) {
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
}
