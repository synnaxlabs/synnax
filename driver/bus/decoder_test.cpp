// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <map>
#include <set>
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/bus/decoder.h"
#include "driver/bus/testutil/testutil.h"

namespace driver::bus {
using namespace testutil;

namespace {
/// @returns a message whose field mux selects field a when it is 0 and field b when it
/// is 1.
synnax::icd::Message multiplexed() {
    auto mux = binary_field("mux", 0);
    auto a = binary_field("a", 8);
    a.multiplexor = mux.key;
    a.multiplex_values = {0};
    auto b = binary_field("b", 8);
    b.multiplexor = mux.key;
    b.multiplex_values = {1};
    return binary_message("status", {mux, a, b});
}

/// @brief resolves a config that reads every field of message. Field i goes to channel
/// i + 1, stamped on indexes[i], or on no index when that is zero.
ReadConfig grouped(
    const synnax::icd::Message &message,
    const std::vector<synnax::channel::Key> &indexes
) {
    ::synnax::bus::ReadConfig cfg;
    cfg.device = "dev";
    ::synnax::bus::ReadMessage rm{.message = message.key};
    std::vector<synnax::channel::Channel> channels;
    std::set<synnax::channel::Key> created;
    for (std::size_t i = 0; i < indexes.size(); i++) {
        const auto key = static_cast<synnax::channel::Key>(i + 1);
        if (indexes[i] != 0 && created.insert(indexes[i]).second)
            channels.push_back(index_channel(indexes[i]));
        channels.push_back(data_channel(key, indexes[i]));
        rm.fields.push_back(
            {.field = field_key(message, i), .channel = key, .index = indexes[i]}
        );
    }
    cfg.messages = {rm};
    x::json::Parser parser(x::json::json::object());
    auto [out, err] = ReadConfig::resolve(
        parser,
        cfg,
        {},
        std::nullopt,
        create_icd({message}),
        channels
    );
    EXPECT_TRUE(parser.ok()) << parser.error_json().dump();
    EXPECT_FALSE(err) << err;
    return std::move(out);
}

/// @brief the samples a flush wrote.
struct Flushed {
    /// @brief the values of each data channel.
    std::map<synnax::channel::Key, std::vector<double>> values;
    /// @brief the times of each index channel.
    std::map<synnax::channel::Key, std::vector<x::telem::TimeStamp>> times;
};

/// @returns what a flush of decoder writes, reading the channels in indexes as times.
Flushed flush(Decoder &decoder, const std::set<synnax::channel::Key> &indexes) {
    x::telem::Frame fr;
    decoder.flush(fr);
    Flushed out;
    for (const auto &[key, series]: fr)
        for (std::size_t i = 0; i < series.size(); i++)
            if (indexes.contains(key))
                out.times[key].push_back(
                    series.at<x::telem::TimeStamp>(static_cast<int>(i))
                );
            else
                out.values[key].push_back(series.at<double>(static_cast<int>(i)));
    return out;
}

const auto T = x::telem::TimeStamp(x::telem::SECOND);
}

TEST(ReadWarning, IsEmptyWithoutProblems) {
    EXPECT_TRUE(read_warning({}).empty());
}

TEST(ReadWarning, JoinsProblemsIntoTheDescription) {
    const auto warning = read_warning({"bad frame", "other"});
    EXPECT_EQ(warning.message, "Failed to read some data");
    EXPECT_EQ(warning.description, "bad frame; other");
}

TEST(Decoder, WritesASampleToEachGroupWhoseFieldsArePresent) {
    const auto cfg = grouped(multiplexed(), {100, 101, 102});
    Decoder decoder(cfg);
    decoder.decode(0, std::vector<std::uint8_t>{0, 7}, T);
    decoder.decode(0, std::vector<std::uint8_t>{1, 9}, T + x::telem::SECOND);
    decoder.decode(0, std::vector<std::uint8_t>{0, 8}, T + 2 * x::telem::SECOND);
    const auto out = flush(decoder, {100, 101, 102});
    EXPECT_EQ(out.values.at(1), (std::vector<double>{0, 1, 0}));
    EXPECT_EQ(out.values.at(2), (std::vector<double>{7, 8}));
    EXPECT_EQ(out.values.at(3), (std::vector<double>{9}));
    EXPECT_EQ(
        out.times.at(100),
        (std::vector{T, T + x::telem::SECOND, T + 2 * x::telem::SECOND})
    );
    EXPECT_EQ(out.times.at(101), (std::vector{T, T + 2 * x::telem::SECOND}));
    EXPECT_EQ(out.times.at(102), std::vector{T + x::telem::SECOND});
}

TEST(Decoder, WritesNothingForAGroupWhoseFieldsAreAbsent) {
    const auto cfg = grouped(multiplexed(), {100, 101, 102});
    Decoder decoder(cfg);
    decoder.decode(0, std::vector<std::uint8_t>{0, 7}, T);
    x::telem::Frame fr;
    decoder.flush(fr);
    EXPECT_TRUE(fr.contains(101));
    EXPECT_FALSE(fr.contains(102));
    EXPECT_FALSE(fr.contains(3));
}

TEST(Decoder, ListsEachIndexOnceInTheWriterConfig) {
    const auto cfg = grouped(multiplexed(), {100, 101, 101});
    const Decoder decoder(cfg);
    EXPECT_EQ(
        decoder.writer_config().channels,
        (std::vector<synnax::channel::Key>{100, 1, 101, 2, 3})
    );
}

TEST(Decoder, WritesFieldsWithoutAnIndexToTheirVirtualChannels) {
    const auto m = binary_message(
        "status",
        {binary_field("a", 0), binary_field("b", 8)}
    );
    const auto cfg = grouped(m, {0, 0});
    Decoder decoder(cfg);
    EXPECT_EQ(
        decoder.writer_config().channels,
        (std::vector<synnax::channel::Key>{1, 2})
    );
    decoder.decode(0, std::vector<std::uint8_t>{3, 4}, T);
    x::telem::Frame fr;
    decoder.flush(fr);
    EXPECT_EQ(fr.size(), 2);
    EXPECT_EQ(fr.at<double>(1, 0), 3);
    EXPECT_EQ(fr.at<double>(2, 0), 4);
}

TEST(Decoder, StampsStrictlyIncreasingTimesOnEachIndex) {
    const auto m = binary_message("status", {binary_field("a", 0)});
    const auto cfg = grouped(m, {100});
    Decoder decoder(cfg);
    decoder.decode(0, std::vector<std::uint8_t>{1}, T);
    decoder.decode(0, std::vector<std::uint8_t>{2}, T);
    decoder.decode(0, std::vector<std::uint8_t>{3}, T - x::telem::SECOND);
    auto out = flush(decoder, {100});
    EXPECT_EQ(
        out.times.at(100),
        (std::vector{T, T + x::telem::TimeSpan(1), T + x::telem::TimeSpan(2)})
    );
    decoder.decode(0, std::vector<std::uint8_t>{4}, T);
    out = flush(decoder, {100});
    EXPECT_EQ(out.times.at(100), std::vector{T + x::telem::TimeSpan(3)});
}

TEST(Decoder, WarnsWhenNoFrameArrivesForTenSecondsAfterStart) {
    const auto m = binary_message("status", {binary_field("a", 0)});
    const auto cfg = grouped(m, {100});
    Clock clock;
    Decoder decoder(cfg, clock.func());
    clock.advance(SILENCE - x::telem::TimeSpan(1));
    EXPECT_EQ(decoder.warning().description, "");
    clock.advance(x::telem::TimeSpan(1));
    EXPECT_EQ(decoder.warning().description, "no frames received since start");
    decoder.raw(std::vector<std::uint8_t>{1});
    decoder.decode(0, std::vector<std::uint8_t>{1}, T);
    EXPECT_EQ(decoder.warning().description, "");
}

TEST(Decoder, RestartsTheSilenceClockOnReset) {
    const auto m = binary_message("status", {binary_field("a", 0)});
    const auto cfg = grouped(m, {100});
    Clock clock;
    Decoder decoder(cfg, clock.func());
    decoder.warn("old");
    clock.advance(SILENCE);
    decoder.reset();
    EXPECT_EQ(decoder.warning().description, "");
    clock.advance(SILENCE);
    EXPECT_EQ(decoder.warning().description, "no frames received since start");
}

TEST(Decoder, WarnsWhenFramesArriveButNoneMatch) {
    const auto m = binary_message("status", {binary_field("a", 0)});
    const auto cfg = grouped(m, {100});
    Clock clock;
    Decoder decoder(cfg, clock.func());
    for (int i = 0; i < 10; i++) {
        decoder.raw(std::vector<std::uint8_t>{9});
        clock.advance(x::telem::SECOND);
    }
    EXPECT_EQ(
        decoder.warning().description,
        "frames arrive, but none match a message in the ICD"
    );
    decoder.raw(std::vector<std::uint8_t>{1});
    decoder.decode(0, std::vector<std::uint8_t>{1}, T);
    EXPECT_EQ(decoder.warning().description, "");
}

TEST(Decoder, StopsWarningAboutUnmatchedFramesWhenTheyStop) {
    const auto m = binary_message("status", {binary_field("a", 0)});
    const auto cfg = grouped(m, {100});
    Clock clock;
    Decoder decoder(cfg, clock.func());
    clock.advance(x::telem::SECOND);
    decoder.raw(std::vector<std::uint8_t>{9});
    clock.advance(SILENCE);
    EXPECT_EQ(decoder.warning().description, "");
}

TEST(Decoder, WarnsWhenAMessageMissesThreePeriods) {
    auto m = binary_message("status", {binary_field("a", 0)});
    m.period = 100 * x::telem::MILLISECOND;
    const auto cfg = grouped(m, {100});
    Clock clock;
    Decoder decoder(cfg, clock.func());
    decoder.raw(std::vector<std::uint8_t>{1});
    decoder.decode(0, std::vector<std::uint8_t>{1}, T);
    clock.advance(300 * x::telem::MILLISECOND - x::telem::TimeSpan(1));
    EXPECT_EQ(decoder.warning().description, "");
    clock.advance(x::telem::TimeSpan(1));
    EXPECT_EQ(
        decoder.warning().description,
        "status missed 3 periods (expected every 100ms)"
    );
    decoder.raw(std::vector<std::uint8_t>{1});
    decoder.decode(0, std::vector<std::uint8_t>{1}, T);
    EXPECT_EQ(decoder.warning().description, "");
}

TEST(Decoder, WarnsAboutMissedPeriodsBeforeAnyFrameArrives) {
    auto m = binary_message("status", {binary_field("a", 0)});
    m.period = 100 * x::telem::MILLISECOND;
    const auto cfg = grouped(m, {100});
    Clock clock;
    Decoder decoder(cfg, clock.func());
    clock.advance(300 * x::telem::MILLISECOND);
    EXPECT_EQ(
        decoder.warning().description,
        "status missed 3 periods (expected every 100ms)"
    );
    clock.advance(SILENCE);
    EXPECT_EQ(decoder.warning().description, "no frames received since start");
}

TEST(Decoder, ExpectsAPolledMessageEveryPollPeriod) {
    auto m = text_message("volt", {delimited_field("v", 0)});
    m.query = "VOLT?";
    const auto cfg = read_config(
        {m},
        std::nullopt,
        nullptr,
        {.rate = x::telem::Rate(10), .timeout = 20 * x::telem::MILLISECOND}
    );
    Clock clock;
    Decoder decoder(cfg, clock.func());
    clock.advance(300 * x::telem::MILLISECOND);
    decoder.raw(std::vector<std::uint8_t>{1});
    EXPECT_EQ(
        decoder.warning().description,
        "volt missed 3 periods (expected every 100ms)"
    );
}

TEST(Decoder, NamesAtMostThreeMessagesThatMissedPeriods) {
    std::vector<synnax::icd::Message> messages;
    for (const auto *name: {"a", "b", "c", "d", "e"}) {
        auto m = text_message(name, {delimited_field("v", 1)}, std::string(name) + ",");
        m.period = x::telem::SECOND;
        messages.push_back(m);
    }
    const auto cfg = read_config(messages, std::nullopt);
    Clock clock;
    Decoder decoder(cfg, clock.func());
    decoder.raw(std::vector<std::uint8_t>{1});
    clock.advance(3 * x::telem::SECOND);
    decoder.raw(std::vector<std::uint8_t>{1});
    EXPECT_EQ(
        decoder.warning().description,
        "a missed 3 periods (expected every 1s); b missed 3 periods (expected every "
        "1s); c missed 3 periods (expected every 1s); 2 more messages missed 3 periods"
    );
}

TEST(Decoder, HoldsAWarningForOneSecondAfterItLastOccurs) {
    const auto m = binary_message("status", {binary_field("a", 0)});
    const auto cfg = grouped(m, {100});
    Clock clock;
    Decoder decoder(cfg, clock.func());
    decoder.raw(std::vector<std::uint8_t>{1});
    decoder.warn("bad frame");
    decoder.warn("other");
    clock.advance(WARNING_HOLD / 2);
    decoder.warn("bad frame");
    EXPECT_EQ(decoder.warning().description, "bad frame; other");
    clock.advance(WARNING_HOLD / 2);
    EXPECT_EQ(decoder.warning().description, "bad frame");
    clock.advance(WARNING_HOLD / 2);
    EXPECT_EQ(decoder.warning().description, "");
}

TEST(Decoder, WarnsAboutADiscardedLateReply) {
    auto m = text_message("volt", {delimited_field("v", 0)});
    m.query = "VOLT?";
    const auto cfg = read_config(
        {m},
        std::nullopt,
        nullptr,
        {.rate = x::telem::Rate(1), .timeout = 20 * x::telem::MILLISECOND}
    );
    Clock clock;
    Decoder decoder(cfg, clock.func());
    decoder.raw(std::vector<std::uint8_t>{1});
    decoder.late(0);
    EXPECT_EQ(decoder.warning().description, "late reply to volt discarded");
    x::telem::Frame fr;
    decoder.flush(fr);
    EXPECT_TRUE(fr.empty());
}

TEST(Decoder, WarnsAndDropsAValueItsChannelCannotHold) {
    const auto m = binary_message(
        "status",
        {binary_field("a", 0, 16), binary_field("b", 16)}
    );
    const auto cfg = read_config(
        {m},
        std::nullopt,
        nullptr,
        {},
        {data_channel(1, 100, x::telem::INT8_T)}
    );
    Decoder decoder(cfg);
    decoder.decode(0, std::vector<std::uint8_t>{0x2C, 0x01, 7}, T);
    EXPECT_EQ(
        decoder.warning().description,
        "status: channel data_1 of data type int8 cannot hold 300"
    );
    x::telem::Frame fr;
    decoder.flush(fr);
    EXPECT_TRUE(fr.empty());
    decoder.decode(0, std::vector<std::uint8_t>{0x05, 0x00, 7}, T);
    decoder.flush(fr);
    EXPECT_EQ(fr.at<std::int8_t>(1, 0), 5);
}

TEST(Decoder, WarnsAndDropsAPayloadThatDoesNotDecode) {
    const auto m = binary_message("status", {binary_field("a", 8)});
    const auto cfg = grouped(m, {100});
    Decoder decoder(cfg);
    decoder.decode(0, std::vector<std::uint8_t>{1}, T);
    EXPECT_TRUE(
        decoder.warning().description.starts_with("status: payload of 1 bytes")
    );
    x::telem::Frame fr;
    decoder.flush(fr);
    EXPECT_TRUE(fr.empty());
}
}
