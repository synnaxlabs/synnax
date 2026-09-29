// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/bus/decoder.h"
#include "driver/bus/testutil/testutil.h"

namespace driver::bus {
using namespace testutil;

namespace {
using Bytes = std::vector<std::uint8_t>;

synnax::channel::Channel raw_channel() {
    auto ch = data_channel(50, 0, x::telem::BYTES_T);
    ch.is_virtual = true;
    return ch;
}

std::vector<x::telem::Frame> flush(Decoder &d) {
    std::vector<x::telem::Frame> frames;
    d.flush(frames.emplace_back());
    return frames;
}

std::vector<std::int64_t>
times(const std::vector<x::telem::Frame> &frames, const synnax::channel::Key key) {
    std::vector<std::int64_t> out;
    for (const auto &fr: frames)
        for (const auto &[k, series]: fr)
            if (k == key)
                for (std::size_t i = 0; i < series.size(); i++)
                    out.push_back(series.at<std::int64_t>(static_cast<int>(i)));
    return out;
}

ReadConfig create_config() {
    return read_config(
        {binary_message("m", {binary_field("a", 0), binary_field("b", 8)})},
        std::nullopt,
        [](::synnax::bus::ReadConfig &cfg) { cfg.raw = 50; },
        {},
        {raw_channel()}
    );
}
}

TEST(Decoder, DecodesAPayloadIntoTheChannelsOfItsMessage) {
    const auto cfg = create_config();
    Decoder d(cfg);
    d.decode(0, Bytes{3, 4}, x::telem::TimeStamp(10));
    const auto frames = flush(d);
    EXPECT_EQ(values(frames, 1), std::vector<double>{3});
    EXPECT_EQ(values(frames, 2), std::vector<double>{4});
    EXPECT_EQ(times(frames, 100), std::vector<std::int64_t>{10});
}

TEST(Decoder, StampsEachSampleAfterTheLastOne) {
    const auto cfg = create_config();
    Decoder d(cfg);
    d.decode(0, Bytes{1, 1}, x::telem::TimeStamp(10));
    d.decode(0, Bytes{2, 2}, x::telem::TimeStamp(10));
    const auto frames = flush(d);
    EXPECT_EQ(times(frames, 100), (std::vector<std::int64_t>{10, 11}));
}

TEST(Decoder, MovesEachSampleIntoOneFrameOnly) {
    const auto cfg = create_config();
    Decoder d(cfg);
    d.decode(0, Bytes{1, 1}, x::telem::TimeStamp(10));
    x::telem::Frame first;
    d.flush(first);
    x::telem::Frame second;
    d.flush(second);
    EXPECT_TRUE(second.empty());
}

TEST(Decoder, WarnsAboutAPayloadThatDoesNotDecode) {
    const auto cfg = create_config();
    Decoder d(cfg);
    d.decode(0, Bytes{1}, x::telem::TimeStamp(10));
    x::telem::Frame fr;
    d.flush(fr);
    EXPECT_TRUE(fr.empty());
    EXPECT_TRUE(d.warning(x::telem::TimeStamp(0)).starts_with("m: "));
}

TEST(Decoder, HoldsAWarningUntilItsCauseStops) {
    const auto cfg = create_config();
    Decoder d(cfg);
    d.warn("a");
    d.warn("b");
    const x::telem::TimeStamp now(0);
    EXPECT_EQ(d.warning(now), "a; b");
    EXPECT_EQ(d.warning(now + WARNING_HOLD / 2), "a; b");
    EXPECT_EQ(d.warning(now + WARNING_HOLD), "");
}

TEST(Decoder, ClearsTheHeldWarningOnReset) {
    const auto cfg = create_config();
    Decoder d(cfg);
    d.warn("a");
    EXPECT_EQ(d.warning(x::telem::TimeStamp(0)), "a");
    d.reset();
    EXPECT_EQ(d.warning(x::telem::TimeStamp(0)), "");
}

TEST(Decoder, WritesEachFrameToTheRawChannel) {
    const auto cfg = create_config();
    Decoder d(cfg);
    d.raw(bytes("ab"));
    d.raw(bytes("c"));
    x::telem::Frame fr;
    d.flush(fr);
    std::vector<std::string> raw;
    for (const auto &[key, series]: fr)
        if (key == 50) raw = samples(series);
    EXPECT_EQ(raw, (std::vector<std::string>{"ab", "c"}));
}

TEST(Decoder, WritesTheIndexFieldAndRawChannels) {
    const auto cfg = create_config();
    const Decoder d(cfg);
    EXPECT_EQ(
        d.writer_config().channels,
        (std::vector<synnax::channel::Key>{100, 1, 2, 50})
    );
    EXPECT_EQ(d.channels().size(), 2);
}
}
