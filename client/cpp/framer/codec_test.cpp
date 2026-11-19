// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <fstream>
#include <functional>
#include <sstream>
#include <string>
#include <vector>

#include "gtest/gtest.h"
#include "nlohmann/json.hpp"

#include "client/cpp/framer/framer.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"

namespace synnax::framer {
x::telem::Frame create_test_frame() {
    auto frame = x::telem::Frame(3);
    auto s1 = x::telem::Series(std::vector{1.0f, 2.0f, 3.0f});
    s1.alignment = x::telem::Alignment(10);
    s1.time_range = {x::telem::TimeStamp(1000), x::telem::TimeStamp(2000)};
    frame.emplace(65537, std::move(s1));

    auto s2 = x::telem::Series(std::vector{4.0, 5.0, 6.0});
    s2.alignment = x::telem::Alignment(20);
    s2.time_range = {x::telem::TimeStamp(1000), x::telem::TimeStamp(2000)};
    frame.emplace(65538, std::move(s2));

    auto s3 = x::telem::Series(std::vector{7, 8, 9});
    s3.alignment = x::telem::Alignment(30);
    s3.time_range = {x::telem::TimeStamp(1500), x::telem::TimeStamp(2500)};
    frame.emplace(65539, std::move(s3));

    return frame;
}

x::telem::Frame create_equal_properties_frame() {
    auto frame = x::telem::Frame(3);

    auto tr = x::telem::TimeRange{x::telem::TimeStamp(1000), x::telem::TimeStamp(2000)};
    x::telem::Alignment alignment(10);

    auto s1 = x::telem::Series(std::vector{1.0f, 2.0f, 3.0f});
    s1.alignment = alignment;
    s1.time_range = tr;
    frame.emplace(65537, std::move(s1));

    auto s2 = x::telem::Series(std::vector{4.0f, 5.0f, 6.0f});
    s2.alignment = alignment;
    s2.time_range = tr;
    frame.emplace(65538, std::move(s2));

    auto s3 = x::telem::Series(std::vector{7.0f, 8.0f, 9.0f});
    s3.alignment = alignment;
    s3.time_range = tr;
    frame.emplace(65539, std::move(s3));

    return frame;
}

x::telem::Frame create_zero_properties_frame() {
    auto frame = x::telem::Frame(3);

    auto tr = x::telem::TimeRange{x::telem::TimeStamp(0), x::telem::TimeStamp(0)};
    x::telem::Alignment alignment(0);

    auto s1 = x::telem::Series(std::vector{1.0f, 2.0f, 3.0f});
    s1.alignment = alignment;
    s1.time_range = tr;
    frame.emplace(65537, std::move(s1));

    auto s2 = x::telem::Series(std::vector{4.0f, 5.0f, 6.0f});
    s2.alignment = alignment;
    s2.time_range = tr;
    frame.emplace(65538, std::move(s2));

    auto s3 = x::telem::Series(std::vector{7.0f, 8.0f, 9.0f});
    s3.alignment = alignment;
    s3.time_range = tr;
    frame.emplace(65539, std::move(s3));

    return frame;
}

x::telem::Frame create_diff_lengths_frame() {
    auto frame = x::telem::Frame(3);

    auto tr = x::telem::TimeRange{x::telem::TimeStamp(1000), x::telem::TimeStamp(2000)};
    x::telem::Alignment alignment(10);

    auto s1 = x::telem::Series(std::vector{1.0f, 2.0f, 3.0f});
    s1.alignment = alignment;
    s1.time_range = tr;
    frame.emplace(65537, std::move(s1));

    auto s2 = x::telem::Series(std::vector{4.0f, 5.0f, 6.0f, 7.0f});
    s2.alignment = alignment;
    s2.time_range = tr;
    frame.emplace(65538, std::move(s2));

    auto s3 = x::telem::Series(std::vector{7.0f, 8.0f});
    s3.alignment = alignment;
    s3.time_range = tr;
    frame.emplace(65539, std::move(s3));

    return frame;
}

x::telem::Frame create_large_equal_frame() {
    constexpr size_t NUM_CHANNELS = 500;
    auto frame = x::telem::Frame(NUM_CHANNELS);
    const auto tr = x::telem::TimeRange{
        x::telem::TimeStamp(1000),
        x::telem::TimeStamp(2000)
    };
    for (size_t i = 0; i < NUM_CHANNELS; i++) {
        auto series = x::telem::Series(std::vector{1.0f, 2.0f, 3.0f});
        series.alignment = x::telem::Alignment(10);
        series.time_range = tr;
        frame.emplace(65537 + i, std::move(series));
    }
    return frame;
}

/// @brief it should correctly encode and decode codec flags.
TEST(CodecTests, FlagsEncodingDecoding) {
    CodecFlags flags;
    flags.equal_lens = true;
    flags.equal_time_ranges = false;
    flags.time_ranges_zero = false;
    flags.all_channels_present = true;
    flags.equal_alignments = true;
    flags.zero_alignments = false;
    flags.multiples_present = true;
    flags.extended = true;

    const uint8_t encoded = flags.encode();
    const CodecFlags decoded = CodecFlags::decode(encoded);

    ASSERT_EQ(decoded.equal_lens, flags.equal_lens);
    ASSERT_EQ(decoded.equal_time_ranges, flags.equal_time_ranges);
    ASSERT_EQ(decoded.time_ranges_zero, flags.time_ranges_zero);
    ASSERT_EQ(decoded.all_channels_present, flags.all_channels_present);
    ASSERT_EQ(decoded.equal_alignments, flags.equal_alignments);
    ASSERT_EQ(decoded.zero_alignments, flags.zero_alignments);
    ASSERT_EQ(decoded.multiples_present, flags.multiples_present);
    ASSERT_EQ(decoded.extended, flags.extended);
}

/// @brief it should reject a frame that carries alignment multiples.
TEST(CodecTests, RejectsAlignmentMultiples) {
    const std::vector<channel::Key> channels = {1};
    const std::vector data_types = {x::telem::UINT8_T};
    std::vector<uint8_t> encoded;
    Codec codec(channels, data_types);
    codec.encode(
        x::telem::Frame(1, x::telem::Series(std::vector<uint8_t>{1, 2})),
        encoded
    );
    encoded[0] = x::binary::set_bit(encoded[0], FlagPosition::MultiplesPresent, true);
    const auto [_, err] = codec.decode(encoded);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    ASSERT_EQ(
        err.data,
        "remote sent alignment multiples, which this codec does not support"
    );
}

/// @brief it should set the extended bit, which shipped decoders reject, in the first
/// frame of a state.
TEST(CodecTests, FirstFrameSetsExtendedFlag) {
    const std::vector<channel::Key> channels = {1};
    const std::vector data_types = {x::telem::UINT8_T};
    std::vector<uint8_t> encoded;
    Codec codec(channels, data_types);
    ASSERT_NIL(codec.encode(
        x::telem::Frame(1, x::telem::Series(std::vector<uint8_t>{1, 2})),
        encoded
    ));
    ASSERT_TRUE(x::binary::get_bit(encoded[0], FlagPosition::Extended));
}

/// @brief it should encode and decode a frame with various data types and properties.
TEST(CodecTests, EncodeDecodeVariedFrame) {
    const auto original_frame = create_test_frame();
    const std::vector data_types = {
        x::telem::FLOAT32_T,
        x::telem::FLOAT64_T,
        x::telem::INT32_T
    };
    const std::vector<channel::Key> channels = {65537, 65538, 65539};
    Codec codec(channels, data_types);
    std::vector<uint8_t> encoded;
    codec.encode(original_frame, encoded);
    const x::telem::Frame decoded_frame = ASSERT_NIL_P(codec.decode(encoded));
    assert_frames_equal(original_frame, decoded_frame);
}

/// @brief it should correctly decode and encode a frame with only one channel present.
TEST(CodecTests, OnlyOneChannelPresent) {
    const std::vector<channel::Key> channels = {1, 2, 3, 4, 5};
    const std::vector data_types = {
        x::telem::UINT8_T,
        x::telem::UINT8_T,
        x::telem::UINT8_T,
        x::telem::UINT8_T,
        x::telem::UINT8_T
    };
    auto frame = x::telem::Frame(
        3,
        x::telem::Series(std::vector<uint8_t>{1, 2, 3, 4, 5})
    );
    std::vector<uint8_t> encoded;
    Codec codec(channels, data_types);
    codec.encode(frame, encoded);
    const x::telem::Frame decoded_frame = ASSERT_NIL_P(codec.decode(encoded));
    assert_frames_equal(frame, decoded_frame);
}

/// @brief it should correctly round-trip a BoolT series of one sample.
TEST(CodecTests, BoolSingleSample) {
    const std::vector<channel::Key> channels = {1};
    const std::vector data_types = {x::telem::BOOLEAN_T};
    uint8_t one = 1;
    auto frame = x::telem::Frame(1, x::telem::Series(&one, 1, x::telem::BOOLEAN_T));
    std::vector<uint8_t> encoded;
    Codec codec(channels, data_types);
    ASSERT_NIL(codec.encode(frame, encoded));
    const x::telem::Frame decoded = ASSERT_NIL_P(codec.decode(encoded));
    assert_frames_equal(frame, decoded);
}

/// @brief it should correctly round-trip a BoolT series at an exact byte boundary.
TEST(CodecTests, BoolExactByteBoundary) {
    const std::vector<channel::Key> channels = {1};
    const std::vector data_types = {x::telem::BOOLEAN_T};
    const std::vector<uint8_t> samples = {1, 0, 1, 0, 1, 0, 1, 0};
    auto frame = x::telem::Frame(
        1,
        x::telem::Series(samples.data(), samples.size(), x::telem::BOOLEAN_T)
    );
    std::vector<uint8_t> encoded;
    Codec codec(channels, data_types);
    ASSERT_NIL(codec.encode(frame, encoded));
    const x::telem::Frame decoded = ASSERT_NIL_P(codec.decode(encoded));
    assert_frames_equal(frame, decoded);
}

/// @brief it should correctly round-trip a BoolT series one sample past a byte
/// boundary, exercising partial-last-byte handling.
TEST(CodecTests, BoolOnePastByteBoundary) {
    const std::vector<channel::Key> channels = {1};
    const std::vector data_types = {x::telem::BOOLEAN_T};
    const std::vector<uint8_t> samples = {1, 0, 1, 0, 1, 0, 1, 0, 1};
    auto frame = x::telem::Frame(
        1,
        x::telem::Series(samples.data(), samples.size(), x::telem::BOOLEAN_T)
    );
    std::vector<uint8_t> encoded;
    Codec codec(channels, data_types);
    ASSERT_NIL(codec.encode(frame, encoded));
    const x::telem::Frame decoded = ASSERT_NIL_P(codec.decode(encoded));
    assert_frames_equal(frame, decoded);
}

/// @brief it should correctly round-trip a BoolT series mixed with other types.
TEST(CodecTests, BoolMixedWithOtherTypes) {
    const std::vector<channel::Key> channels = {1, 2, 3};
    const std::vector data_types = {
        x::telem::BOOLEAN_T,
        x::telem::FLOAT32_T,
        x::telem::UINT8_T
    };
    const std::vector<uint8_t> bool_samples = {1, 0, 1};
    x::telem::Frame frame;
    frame.reserve(3);
    frame.emplace(
        1,
        x::telem::Series(bool_samples.data(), bool_samples.size(), x::telem::BOOLEAN_T)
    );
    frame.emplace(2, x::telem::Series(std::vector<float>{1.5f, 2.5f, 3.5f}));
    frame.emplace(3, x::telem::Series(std::vector<uint8_t>{7, 8, 9}));
    std::vector<uint8_t> encoded;
    Codec codec(channels, data_types);
    ASSERT_NIL(codec.encode(frame, encoded));
    const x::telem::Frame decoded = ASSERT_NIL_P(codec.decode(encoded));
    assert_frames_equal(frame, decoded);
}

/// @brief it should encode and decode a frame with equal properties.
TEST(CodecTests, EncodeDecodeEqualPropertiesFrame) {
    const auto original_frame = create_equal_properties_frame();
    const std::vector data_types = {
        x::telem::FLOAT32_T,
        x::telem::FLOAT32_T,
        x::telem::FLOAT32_T
    };
    const std::vector<channel::Key> channels = {65537, 65538, 65539};
    Codec codec(channels, data_types);
    std::vector<uint8_t> encoded;
    codec.encode(original_frame, encoded);
    const x::telem::Frame decoded_frame = ASSERT_NIL_P(codec.decode(encoded));
    assert_frames_equal(original_frame, decoded_frame);
}

/// @brief it should encode and decode a frame with zero properties using optimized
/// encoding.
TEST(CodecTests, EncodeDecodeZeroPropertiesFrame) {
    const auto original_frame = create_zero_properties_frame();
    const std::vector data_types = {
        x::telem::FLOAT32_T,
        x::telem::FLOAT32_T,
        x::telem::FLOAT32_T
    };
    const std::vector<channel::Key> channels = {65537, 65538, 65539};
    Codec codec(channels, data_types);
    std::vector<uint8_t> encoded;
    codec.encode(original_frame, encoded);
    const x::telem::Frame decoded_frame = ASSERT_NIL_P(codec.decode(encoded));
    assert_frames_equal(original_frame, decoded_frame);
}

/// @brief it should encode and decode a frame with different length series.
TEST(CodecTests, EncodeDecodeDifferentLengthsFrame) {
    const auto original_frame = create_diff_lengths_frame();
    const std::vector data_types = {
        x::telem::FLOAT32_T,
        x::telem::FLOAT32_T,
        x::telem::FLOAT32_T
    };
    const std::vector<channel::Key> channels = {65537, 65538, 65539};
    Codec codec(channels, data_types);
    std::vector<uint8_t> encoded;
    codec.encode(original_frame, encoded);
    const x::telem::Frame decoded_frame = ASSERT_NIL_P(codec.decode(encoded));
    assert_frames_equal(original_frame, decoded_frame);
}

/// @brief it should encode and decode a frame with a subset of channels.
TEST(CodecTests, EncodeDecodeChannelSubset) {
    const auto original_frame = create_test_frame();
    const std::vector data_types = {
        x::telem::FLOAT32_T,
        x::telem::FLOAT64_T,
        x::telem::INT32_T,
        x::telem::FLOAT32_T
    };
    const std::vector<channel::Key> channels = {65537, 65538, 65539, 65540};
    Codec codec(channels, data_types);
    std::vector<uint8_t> encoded;
    codec.encode(original_frame, encoded);
    const x::telem::Frame decoded_frame = ASSERT_NIL_P(codec.decode(encoded));
    assert_frames_equal(original_frame, decoded_frame);
}

/// @brief it should handle a large frame to ensure robustness.
TEST(CodecTests, LargeFrame) {
    auto frame = x::telem::Frame(1);
    std::vector large_data(100000, 3.14159f);
    auto large_series = x::telem::Series(large_data);
    large_series.time_range = {x::telem::TimeStamp(1000), x::telem::TimeStamp(2000)};
    large_series.alignment = x::telem::Alignment(42);
    frame.emplace(65537, std::move(large_series));
    const std::vector data_types = {x::telem::FLOAT32_T};
    std::vector<channel::Key> channels = {65537};
    Codec codec(channels, data_types);
    std::vector<uint8_t> encoded;
    codec.encode(frame, encoded);
    const x::telem::Frame decoded_frame = ASSERT_NIL_P(codec.decode(encoded));
    assert_frames_equal(frame, decoded_frame);
}

/// @brief it should carry a series of another data type with a type override.
TEST(CodecTests, EncodeMismatchedDataType) {
    const std::vector data_types = {x::telem::FLOAT32_T};
    const std::vector<channel::Key> channels = {65537};
    Codec codec(channels, data_types);
    auto frame = x::telem::Frame(1);
    frame.emplace(65537, x::telem::Series(std::vector{1, 2, 3}));
    std::vector<uint8_t> encoded;
    ASSERT_NIL(codec.encode(frame, encoded));
    const auto decoded = ASSERT_NIL_P(codec.decode(encoded));
    ASSERT_EQ(decoded.series->at(0).data_type(), x::telem::INT32_T);
    ASSERT_EQ(decoded.series->at(0).values<int32_t>(), std::vector<int32_t>({1, 2, 3}));
}

/// @brief it should accept an int64 series for a timestamp channel, mirroring the
/// Int64T <-> TimestampT equivalence applied by the server-side writer validator
/// and frame codec.
TEST(CodecTests, EncodeInt64SeriesForTimestampChannel) {
    const std::vector data_types = {x::telem::TIMESTAMP_T};
    const std::vector<channel::Key> channels = {65537};
    Codec codec(channels, data_types);

    auto frame = x::telem::Frame(1);
    auto series = x::telem::Series(std::vector<int64_t>{1778020940471336961LL});
    series.time_range = {x::telem::TimeStamp(1000), x::telem::TimeStamp(2000)};
    series.alignment = x::telem::Alignment(10);
    frame.emplace(65537, std::move(series));

    std::vector<uint8_t> encoded;
    ASSERT_NIL(codec.encode(frame, encoded));
    ASSERT_FALSE(encoded.empty());
}

/// @brief it should accept a timestamp series for an int64 channel, mirroring the
/// Int64T <-> TimestampT equivalence applied by the server-side writer validator
/// and frame codec.
TEST(CodecTests, EncodeTimestampSeriesForInt64Channel) {
    const std::vector data_types = {x::telem::INT64_T};
    const std::vector<channel::Key> channels = {65537};
    Codec codec(channels, data_types);

    auto frame = x::telem::Frame(1);
    auto series = x::telem::Series(x::telem::TimeStamp(x::telem::SECOND));
    series.time_range = {x::telem::TimeStamp(1000), x::telem::TimeStamp(2000)};
    series.alignment = x::telem::Alignment(10);
    frame.emplace(65537, std::move(series));

    std::vector<uint8_t> encoded;
    ASSERT_NIL(codec.encode(frame, encoded));
    ASSERT_FALSE(encoded.empty());
}

/// @brief it should return a validation erorr when the frame has a key that was not
/// provided to the codec.
TEST(CodecTests, EncodeFrameUnknownKey) {
    const std::vector data_types = {x::telem::FLOAT32_T, x::telem::FLOAT64_T};
    const std::vector<channel::Key> channels = {65537, 65538};
    Codec codec(channels, data_types);

    // Create a frame with an unknown key
    auto frame = x::telem::Frame(1);
    auto series = x::telem::Series(std::vector{7, 8, 9});
    series.time_range = {x::telem::TimeStamp(1500), x::telem::TimeStamp(2500)};
    series.alignment = x::telem::Alignment(30);
    // Using key 65539 which wasn't provided to the codec
    frame.emplace(65539, std::move(series));

    std::vector<uint8_t> encoded;
    auto err = codec.encode(frame, encoded);

    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    ASSERT_TRUE(err.message().contains("extra key"));
}

namespace {
std::vector<uint8_t> from_hex(std::string h) {
    std::erase(h, ' ');
    std::vector<uint8_t> out;
    for (size_t i = 0; i + 1 < h.size(); i += 2)
        out.push_back(static_cast<uint8_t>(std::stoul(h.substr(i, 2), nullptr, 16)));
    return out;
}

std::string to_hex(const uint8_t *data, const size_t size) {
    static constexpr char digits[] = "0123456789abcdef";
    std::string out;
    for (size_t i = 0; i < size; i++) {
        out += digits[data[i] >> 4];
        out += digits[data[i] & 0xF];
    }
    return out;
}

x::telem::Series series_of(const nlohmann::json &s) {
    const x::telem::DataType dt(s["data_type"].get<std::string>());
    const auto data = from_hex(s["data"].get<std::string>());
    const size_t cap = dt.is_variable() ? data.size() : data.size() / dt.density();
    auto ser = x::telem::Series(dt, cap);
    ser.write(data.data(), cap);
    ser.clock = s.value("clock", 0U);
    return ser;
}

/// @brief the first frame of a float32 key 1 whose one sample, 1.0, has clock 5. Byte
/// 5 is the extension flags, byte 6 the type code, byte 15 the override kinds, and
/// bytes 20 to 23 the next override ordinal.
const std::string
    CLOCK_FRAME = "BF 01000000 03 09 01000000 00000000 02 05000000 FFFFFFFF 0000803F";

const std::string
    WORKED_EXAMPLE = "BF 01000000 0B 0901 01000000 05000000 02000000 01000000"
                     " E803000000000000 D007000000000000"
                     " 03 0A 05000000 FFFFFFFF 000000000000F03F 0000000000000040 0100";

void put_u32(std::vector<uint8_t> &b, const size_t at, const uint32_t v) {
    for (int i = 0; i < 4; i++)
        b[at + i] = static_cast<uint8_t>(v >> (8 * i));
}
}

/// @brief it should encode and decode each golden frame byte for byte. The frames are
/// shared with the Go, TypeScript, and Python codecs.
TEST(CodecTests, GoldenFrames) {
    std::ifstream file("x/testdata/framecodec/vectors.json");
    ASSERT_TRUE(file.good());
    const auto vectors = nlohmann::json::parse(file);
    ASSERT_FALSE(vectors["cases"].empty());
    for (const auto &c: vectors["cases"]) {
        const auto keys = c["keys"].get<std::vector<channel::Key>>();
        std::vector<x::telem::DataType> data_types;
        for (const auto &dt: c["data_types"])
            data_types.emplace_back(dt.get<std::string>());
        Codec enc(keys, data_types);
        Codec dec(keys, data_types);
        for (const auto &f: c["frames"]) {
            const auto name = c["name"].get<std::string>() + ": " +
                              f["name"].get<std::string>();
            x::telem::Frame fr(f["series"].size());
            for (const auto &s: f["series"])
                fr.emplace(s["key"].get<channel::Key>(), series_of(s));
            std::vector<uint8_t> encoded;
            ASSERT_NIL(enc.encode(fr, encoded));
            ASSERT_EQ(
                to_hex(encoded.data(), encoded.size()),
                f["encoded"].get<std::string>()
            ) << name;
            const auto decoded = ASSERT_NIL_P(
                dec.decode(from_hex(f["encoded"].get<std::string>()))
            );
            ASSERT_EQ(decoded.size(), f["series"].size()) << name;
            for (size_t i = 0; i < decoded.size(); i++) {
                const auto &want = f["series"][i];
                const auto &got = decoded.series->at(i);
                ASSERT_EQ(decoded.channels->at(i), want["key"].get<channel::Key>())
                    << name;
                ASSERT_EQ(got.data_type().name(), want["data_type"].get<std::string>())
                    << name;
                ASSERT_EQ(got.clock, want.value("clock", 0U)) << name;
                ASSERT_EQ(
                    to_hex(
                        reinterpret_cast<const uint8_t *>(got.data()),
                        got.byte_size()
                    ),
                    want["data"].get<std::string>()
                ) << name;
            }
        }
    }
}

/// @brief it should label an int64 series on a timestamp key with no override.
TEST(CodecTests, Int64OnTimestampKeyNeedsNoOverride) {
    Codec enc({1}, {x::telem::TIMESTAMP_T});
    std::vector<uint8_t> encoded;
    ASSERT_NIL(enc.encode(
        x::telem::Frame(1, x::telem::Series(std::vector<int64_t>{7})),
        encoded
    ));
    ASSERT_EQ(encoded, from_hex("BF 01000000 01 0B 01000000 0700000000000000"));
    Codec dec({1}, {x::telem::TIMESTAMP_T});
    const auto decoded = ASSERT_NIL_P(dec.decode(encoded));
    ASSERT_EQ(decoded.series->at(0).data_type(), x::telem::TIMESTAMP_T);
}

/// @brief it should commit nothing when a frame fails to encode.
TEST(CodecTests, EncodeErrorCommitsNothing) {
    Codec c({1}, {x::telem::FLOAT32_T});
    auto bad = x::telem::Series(x::telem::DataType("int128"), 0);
    bad.clock = 5;
    std::vector<uint8_t> encoded;
    const auto err = c.encode(x::telem::Frame(1, std::move(bad)), encoded);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    ASSERT_EQ(err.data, "data type \"int128\" of channel 1 has no type code");
    auto good = x::telem::Series(std::vector<float>{1});
    good.clock = 5;
    ASSERT_NIL(c.encode(x::telem::Frame(1, std::move(good)), encoded));
    ASSERT_EQ(encoded, from_hex(CLOCK_FRAME));
}

/// @brief it should reject a malformed frame with a pinned message, and drop its
/// state.
TEST(CodecTests, Rejects) {
    using Corrupt = std::function<void(std::vector<uint8_t> &)>;
    const std::vector<std::pair<Corrupt, std::string>> cases = {
        {[](auto &b) { b[5] |= 0b100; },
         "remote sent extension flags 00000111, which hold a reserved bit"},
        {[](auto &b) { b[5] |= 0b1000'0000; },
         "remote sent extension flags 10000011, which hold a reserved bit"},
        {[](auto &b) { b[5] = 0; }, "remote sent extension flags of 0"},
        {[](auto &b) { b[6] = 0; },
         "remote sent data type code 0, which this codec does not know"},
        {[](auto &b) { b[6] = 17; },
         "remote sent data type code 17, which this codec does not know"},
        {[](auto &b) { b[6] = 13; },
         "remote sent variable data type string under a shared length"},
        {[](auto &b) { b[15] = 0; }, "remote sent override kinds of 0 for series 0"},
        {[](auto &b) { b[15] |= 0b100; },
         "remote sent override kinds 00000110 for series 0, which hold a reserved bit"},
        {[](auto &b) { put_u32(b, 20, 0); },
         "remote sent next override 0, which is not after series 0"},
        {[](auto &b) { put_u32(b, 20, 1); },
         "remote ended the frame with the override for series 1 pending"},
        {[](auto &b) { b.push_back(0); }, "remote sent bytes after the last series"},
    };
    for (const auto &[corrupt, msg]: cases) {
        Codec dec({1}, {x::telem::FLOAT32_T});
        auto b = from_hex(CLOCK_FRAME);
        corrupt(b);
        const auto [_, err] = dec.decode(b);
        ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
        ASSERT_EQ(err.data, msg);
        const auto [__, dropped] = dec.decode(from_hex(CLOCK_FRAME));
        ASSERT_EQ(
            dropped.data,
            "remote sent a frame for state 1 after a frame of it failed to decode"
        );
    }
}

/// @brief it should reject a second type declaration, and name the older codec of a
/// remote that declares none.
TEST(CodecTests, RejectsDeclarationErrors) {
    Codec dec({1}, {x::telem::FLOAT32_T});
    ASSERT_NIL_P(dec.decode(from_hex(CLOCK_FRAME)));
    ASSERT_EQ(
        dec.decode(from_hex(CLOCK_FRAME)).second.data,
        "remote declared the types of state 1 twice"
    );
    Codec fresh({1}, {x::telem::FLOAT32_T});
    ASSERT_EQ(
        fresh.decode(from_hex("3F 01000000 01000000 0000803F")).second.data,
        "remote sent no type declaration for state 1, so it runs an older codec, "
        "which this codec does not support"
    );
}

/// @brief it should reject a malformed list of declared clocks.
TEST(CodecTests, RejectsDeclaredClockErrors) {
    const auto worked = [] {
        return Codec({6, 8}, {x::telem::FLOAT32_T, x::telem::UINT8_T});
    };
    auto requested = from_hex(WORKED_EXAMPLE);
    put_u32(requested, 12, 6);
    ASSERT_EQ(
        worked().decode(requested).second.data,
        "remote declared clock 6, which the request names"
    );
    auto overridden = from_hex(WORKED_EXAMPLE);
    put_u32(overridden, 20, 0);
    overridden.insert(overridden.begin() + 24, 0b10);
    ASSERT_EQ(
        worked().decode(overridden).second.data,
        "remote sent an override for declared clock 5"
    );
    auto huge = from_hex(WORKED_EXAMPLE);
    put_u32(huge, 8, 0xFFFFFFFF);
    ASSERT_EQ(worked().decode(huge).second.data, "remote sent a frame that ends early");
    Codec c({6}, {x::telem::FLOAT32_T});
    auto t5 = x::telem::Series(std::vector<int64_t>{1});
    t5.clock = 5;
    auto t7 = x::telem::Series(std::vector<int64_t>{1});
    t7.clock = 7;
    x::telem::Frame fr(3);
    fr.emplace(5, std::move(t5));
    fr.emplace(6, x::telem::Series(std::vector<float>{1}));
    fr.emplace(7, std::move(t7));
    std::vector<uint8_t> encoded;
    ASSERT_NIL(c.encode(fr, encoded));
    put_u32(encoded, 11, 7);
    put_u32(encoded, 15, 5);
    ASSERT_EQ(
        c.decode(encoded).second.data,
        "remote declared clock 5 after clock 7, so the list is not ascending"
    );
}
}
