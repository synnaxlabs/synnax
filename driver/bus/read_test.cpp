// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <functional>
#include <map>
#include <memory>
#include <set>
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/bus/read.h"
#include "driver/bus/testutil/testutil.h"
#include "driver/codec/framing/framing.h"

namespace driver::bus {
using namespace testutil;

namespace {
/// @brief what a sequence of reads produced.
struct Output {
    /// @brief every value written to each channel, in order.
    std::map<synnax::channel::Key, std::vector<double>> values;
    /// @brief every time stamped on each index channel, in order.
    std::map<synnax::channel::Key, std::vector<x::telem::TimeStamp>> times;
    /// @brief every raw frame, in order.
    std::vector<std::string> raw;
    /// @brief the last warning returned.
    std::string warning;
    /// @brief the last error returned.
    x::errors::Error error;
};

/// @brief a source over an in-memory wire, with the read loop a pipeline would run.
class Harness {
public:
    std::shared_ptr<Wire> wire = std::make_shared<Wire>();
    std::unique_ptr<Source> source;
    Output out;
    std::set<synnax::channel::Key> indexes;
    synnax::channel::Key raw = 0;
    x::breaker::Breaker breaker{x::breaker::Config{.name = "test"}};
    bool started = false;

    explicit Harness(ReadConfig cfg) {
        for (const auto &m: cfg.messages)
            if (m.index != 0) this->indexes.insert(m.index);
        this->raw = cfg.raw;
        this->source = std::make_unique<Source>(std::move(cfg), acquire(this->wire));
        this->breaker.start();
    }

    ~Harness() { this->breaker.stop(); }

    /// @brief starts the source, as a pipeline does before its first read.
    x::errors::Error start() {
        this->started = true;
        return this->source->start();
    }

    /// @brief reads once, recording what came back. Starts the source first when no
    /// test has.
    void read() {
        if (!this->started) { ASSERT_NIL(this->start()); }
        x::telem::Frame fr;
        const auto res = this->source->read(this->breaker, fr);
        this->out.warning = res.warning;
        this->out.error = res.error;
        for (const auto &[key, series]: fr) {
            if (key == this->raw) {
                for (const auto &s: samples(series))
                    this->out.raw.push_back(s);
                continue;
            }
            for (std::size_t i = 0; i < series.size(); i++)
                if (this->indexes.contains(key))
                    this->out.times[key].push_back(
                        series.at<x::telem::TimeStamp>(static_cast<int>(i))
                    );
                else
                    this->out.values[key].push_back(
                        series.at<double>(static_cast<int>(i))
                    );
        }
    }

    /// @brief reads until done returns true or 200 reads pass.
    bool read_until(const std::function<bool(const Output &)> &done) {
        for (int i = 0; i < 200; i++) {
            this->read();
            if (done(this->out)) return true;
        }
        return false;
    }

    /// @returns the values written to key.
    std::vector<double> values(const synnax::channel::Key key) {
        return this->out.values[key];
    }
};

const ::synnax::bus::Framing NEWLINE = ::synnax::bus::DelimiterFraming{};
}

TEST(Source, DecodesABinaryMessageIntoItsChannels) {
    const auto m = binary_message(
        "status",
        {binary_field("a", 0), binary_field("b", 8, 16)}
    );
    Harness h(read_config({m}, ::synnax::bus::FixedFraming{.length = 3}));
    const auto before = x::telem::TimeStamp::now();
    h.wire->push({0x05, 0x34, 0x12});
    ASSERT_TRUE(h.read_until([](const Output &o) { return o.values.contains(1); }));
    EXPECT_EQ(h.values(1), std::vector<double>{5});
    EXPECT_EQ(h.values(2), std::vector<double>{0x1234});
    ASSERT_EQ(h.out.times[100].size(), 1);
    EXPECT_GE(h.out.times[100][0], before);
    EXPECT_TRUE(h.out.warning.empty());
}

TEST(Source, DecodesSeveralMessagesMatchedByTheirIdentifiers) {
    const auto a = text_message(
        "a",
        {delimited_field("x", 1), delimited_field("y", 2)},
        synnax::library::TokenIdentifier{.prefix = "A,"}
    );
    const auto b = text_message(
        "b",
        {delimited_field("z", 1)},
        synnax::library::TokenIdentifier{.prefix = "B,"}
    );
    Harness h(read_config({a, b}, NEWLINE));
    h.wire->push(bytes("A,1,2\nB,3\nA,4,5\n"));
    ASSERT_TRUE(h.read_until([](const Output &o) { return o.values.contains(3); }));
    EXPECT_EQ(h.values(1), (std::vector<double>{1, 4}));
    EXPECT_EQ(h.values(2), (std::vector<double>{2, 5}));
    EXPECT_EQ(h.values(3), std::vector<double>{3});
    const auto &times = h.out.times[100];
    ASSERT_EQ(times.size(), 2);
    EXPECT_LT(times[0], times[1]);
    EXPECT_EQ(h.out.times[101].size(), 1);
}

TEST(Source, IgnoresDisabledMessages) {
    const auto a = text_message(
        "a",
        {delimited_field("x", 1)},
        synnax::library::TokenIdentifier{.prefix = "A,"}
    );
    const auto b = text_message(
        "b",
        {delimited_field("z", 1)},
        synnax::library::TokenIdentifier{.prefix = "B,"}
    );
    Harness h(read_config({a, b}, NEWLINE, [](auto &cfg) {
        cfg.messages[1].disabled = true;
    }));
    const auto keys = h.source->writer_config().channels;
    EXPECT_EQ(keys, (std::vector<synnax::channel::Key>{100, 1}));
    h.wire->push(bytes("B,3\nA,1\n"));
    ASSERT_TRUE(h.read_until([](const Output &o) { return o.values.contains(1); }));
    EXPECT_FALSE(h.out.values.contains(2));
}

TEST(Source, WritesEveryFrameToTheRawChannel) {
    const auto a = text_message(
        "a",
        {delimited_field("x", 1)},
        synnax::library::TokenIdentifier{.prefix = "A,"}
    );
    auto raw = data_channel(50, 0, x::telem::BYTES_T);
    Harness h(read_config({a}, NEWLINE, [](auto &cfg) { cfg.raw = 50; }, {}, {raw}));
    h.wire->push(bytes("A,1\nunknown\n"));
    ASSERT_TRUE(h.read_until([](const Output &o) { return o.raw.size() == 2; }));
    EXPECT_EQ(h.out.raw, (std::vector<std::string>{"A,1", "unknown"}));
    EXPECT_EQ(h.values(1), std::vector<double>{1});
}

TEST(Source, ResynchronizesAfterGarbage) {
    const auto m = binary_message("status", {binary_field("v", 24)});
    Harness h(
        read_config({m}, ::synnax::bus::SyncFraming{.sync = "AA55", .length_offset = 2})
    );
    h.wire->push({0x01, 0x02, 0xAA, 0x03, 0xAA, 0x55, 0x01, 0x07});
    ASSERT_TRUE(h.read_until([](const Output &o) { return o.values.contains(1); }));
    EXPECT_EQ(h.values(1), std::vector<double>{7});
    EXPECT_NE(h.out.warning.find("dropped"), std::string::npos);
}

TEST(Source, ReportsChecksumFailures) {
    const ::synnax::bus::Framing framing = ::synnax::bus::SyncFraming{
        .sync = "AA55",
        .length_offset = 2,
        .length_adjustment = 2,
        .checksum = ::synnax::bus::CHECKSUM_CRC_16_MODBUS,
    };
    const auto m = binary_message("status", {binary_field("v", 24)});
    Harness h(read_config({m}, framing));
    auto [framer, err] = codec::framing::create(framing);
    ASSERT_NIL(err);
    std::vector<std::uint8_t> good;
    ASSERT_NIL(framer->encode(std::vector<std::uint8_t>{0xAA, 0x55, 0x00, 0x09}, good));
    auto bad = good;
    bad[3] ^= 0xFF;
    h.wire->push(bad);
    ASSERT_TRUE(h.read_until([](const Output &o) { return !o.warning.empty(); }));
    EXPECT_FALSE(h.out.values.contains(1));
    h.wire->push(good);
    ASSERT_TRUE(h.read_until([](const Output &o) { return o.values.contains(1); }));
    EXPECT_EQ(h.values(1), std::vector<double>{9});
}

TEST(Source, WarnsAndDropsAShortPayload) {
    const auto m = binary_message("status", {binary_field("v", 16)});
    Harness h(read_config({m}, NEWLINE));
    h.wire->push(bytes("ab\n"));
    ASSERT_TRUE(h.read_until([](const Output &o) { return !o.warning.empty(); }));
    EXPECT_TRUE(h.out.warning.starts_with("status: payload of 2 bytes"));
    EXPECT_FALSE(h.out.values.contains(1));
}

TEST(Source, WarnsWhenATextFieldDoesNotParse) {
    const auto m = text_message("status", {delimited_field("v", 0)});
    Harness h(read_config({m}, NEWLINE));
    h.wire->push(bytes("abc\n"));
    ASSERT_TRUE(h.read_until([](const Output &o) { return !o.warning.empty(); }));
    EXPECT_EQ(h.out.warning, "status: 1 fields did not parse as numbers");
    EXPECT_FALSE(h.out.values.contains(1));
}

TEST(Source, ReportsARepeatedWarningOnce) {
    const auto m = text_message("status", {delimited_field("v", 0)});
    Harness h(read_config({m}, NEWLINE));
    h.wire->push(bytes("abc\nabc\nabc\n"));
    ASSERT_TRUE(h.read_until([](const Output &o) { return !o.warning.empty(); }));
    EXPECT_EQ(h.out.warning, "status: 1 fields did not parse as numbers");
}

TEST(Source, ReconnectsAndDropsThePartialFrameAfterTheTransportFails) {
    const auto m = text_message(
        "a",
        {delimited_field("x", 1)},
        synnax::library::TokenIdentifier{.prefix = "A,"}
    );
    Harness h(read_config({m}, NEWLINE));
    ASSERT_NIL(h.start());
    h.wire->push(bytes("A,1"));
    h.read();
    h.wire->fail(transport::UNREACHABLE_ERROR);
    h.read();
    ASSERT_OCCURRED_AS(h.out.error, transport::UNREACHABLE_ERROR);
    h.wire->push(bytes("2\nA,3\n"));
    ASSERT_TRUE(h.read_until([](const Output &o) { return o.values.contains(1); }));
    EXPECT_EQ(h.values(1), std::vector<double>{3});
    EXPECT_EQ(h.wire->opens, 2);
}

TEST(Source, StartsWhileTheDeviceIsUnreachable) {
    const auto m = text_message("a", {delimited_field("x", 0)});
    Harness h(read_config({m}, NEWLINE));
    h.wire->open_errs = {transport::UNREACHABLE_ERROR, transport::UNREACHABLE_ERROR};
    ASSERT_NIL(h.start());
    h.read();
    ASSERT_OCCURRED_AS(h.out.error, transport::UNREACHABLE_ERROR);
    h.wire->push(bytes("4\n"));
    ASSERT_TRUE(h.read_until([](const Output &o) { return o.values.contains(1); }));
    EXPECT_EQ(h.values(1), std::vector<double>{4});
}

TEST(Source, FailsToStartWithInvalidDeviceProperties) {
    const auto m = text_message("a", {delimited_field("x", 0)});
    Harness h(read_config({m}, NEWLINE));
    h.wire->open_errs = {transport::CONFIG_ERROR};
    ASSERT_OCCURRED_AS(h.start(), transport::CONFIG_ERROR);
}

TEST(Source, PollsEachQueryAndDecodesItsReplyInOrder) {
    auto volt = text_message("volt", {delimited_field("v", 0)});
    volt.query = "VOLT?";
    auto curr = text_message("curr", {delimited_field("i", 0)});
    curr.query = "CURR?";
    Harness h(read_config(
        {volt, curr},
        NEWLINE,
        nullptr,
        {.rate = x::telem::Rate(50), .timeout = 500 * x::telem::MILLISECOND}
    ));
    h.wire->on_write = [](Wire &w, const std::span<const std::uint8_t> data) {
        if (text(data) == "VOLT?\n") w.reads.push_back(bytes("1.5\n"));
        if (text(data) == "CURR?\n") w.reads.push_back(bytes("0.25\n"));
    };
    ASSERT_NIL(h.start());
    ASSERT_TRUE(h.read_until([](const Output &o) {
        return o.values.contains(1) && o.values.contains(2);
    }));
    EXPECT_EQ(h.values(1)[0], 1.5);
    EXPECT_EQ(h.values(2)[0], 0.25);
    const auto writes = h.wire->written();
    ASSERT_GE(writes.size(), 2);
    EXPECT_EQ(text(writes[0]), "VOLT?\n");
    EXPECT_EQ(text(writes[1]), "CURR?\n");
    EXPECT_TRUE(h.out.warning.empty());
}

TEST(Source, DoesNotTakeAStreamedFrameAsTheReply) {
    auto polled = text_message(
        "polled",
        {delimited_field("v", 1)},
        synnax::library::TokenIdentifier{.prefix = "R,"}
    );
    polled.query = "Q";
    const auto streamed = text_message(
        "streamed",
        {delimited_field("v", 1)},
        synnax::library::TokenIdentifier{.prefix = "S,"}
    );
    Harness h(read_config(
        {polled, streamed},
        NEWLINE,
        nullptr,
        {.rate = x::telem::Rate(20), .timeout = 500 * x::telem::MILLISECOND}
    ));
    h.wire->on_write = [](Wire &w, std::span<const std::uint8_t>) {
        w.reads.push_back(bytes("S,9\nR,1\n"));
    };
    ASSERT_NIL(h.start());
    ASSERT_TRUE(h.read_until([](const Output &o) {
        return o.values.contains(1) && o.values.contains(2);
    }));
    EXPECT_EQ(h.values(1)[0], 1);
    EXPECT_EQ(h.values(2)[0], 9);
}

TEST(Source, WarnsOnAMissedReplyAndRecoversWhenRepliesResume) {
    auto volt = text_message("volt", {delimited_field("v", 0)});
    volt.query = "VOLT?";
    Harness h(read_config(
        {volt},
        NEWLINE,
        nullptr,
        {.rate = x::telem::Rate(20), .timeout = 20 * x::telem::MILLISECOND}
    ));
    ASSERT_NIL(h.start());
    ASSERT_TRUE(h.read_until([](const Output &o) { return !o.warning.empty(); }));
    EXPECT_EQ(h.out.warning, "no reply to the query of volt within 20ms");
    {
        std::lock_guard lock(h.wire->mu);
        h.wire->on_write = [](Wire &w, std::span<const std::uint8_t>) {
            w.reads.push_back(bytes("2\n"));
        };
    }
    ASSERT_TRUE(h.read_until([](const Output &o) {
        return o.values.contains(1) && o.warning.empty();
    }));
    EXPECT_EQ(h.values(1).back(), 2);
}

TEST(Source, DiscardsALateReplyBeforeTheNextQuery) {
    auto volt = text_message("volt", {delimited_field("v", 0)});
    volt.query = "VOLT?";
    Harness h(read_config(
        {volt},
        NEWLINE,
        nullptr,
        {.rate = x::telem::Rate(1000), .timeout = 20 * x::telem::MILLISECOND}
    ));
    ASSERT_NIL(h.start());
    h.read();
    ASSERT_EQ(h.out.warning, "no reply to the query of volt within 20ms");
    {
        std::lock_guard lock(h.wire->mu);
        h.wire->reads.push_back(bytes("1\n7"));
        h.wire->on_write = [](Wire &w, std::span<const std::uint8_t>) {
            w.reads.push_back(bytes("2\n"));
        };
    }
    ASSERT_TRUE(h.read_until([](const Output &o) { return o.values.contains(1); }));
    EXPECT_EQ(h.values(1), std::vector<double>{2});
}

TEST(Source, DecodesEachDatagramAsOneFrame) {
    const auto m = binary_message(
        "status",
        {binary_field("a", 0), binary_field("b", 8)}
    );
    Harness h(read_config({m}, std::nullopt));
    h.wire->push({0x01, 0x02});
    h.wire->push({0x03, 0x04});
    ASSERT_TRUE(h.read_until([](const Output &o) {
        return o.values.contains(1) && o.values.at(1).size() == 2;
    }));
    EXPECT_EQ(h.values(1), (std::vector<double>{1, 3}));
    EXPECT_EQ(h.values(2), (std::vector<double>{2, 4}));
}
}
