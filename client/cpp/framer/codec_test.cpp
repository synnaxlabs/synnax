// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <sstream>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/framer/framer.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/test/test.h"

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

    const uint8_t encoded = flags.encode();
    const CodecFlags decoded = CodecFlags::decode(encoded);

    ASSERT_EQ(decoded.equal_lens, flags.equal_lens);
    ASSERT_EQ(decoded.equal_time_ranges, flags.equal_time_ranges);
    ASSERT_EQ(decoded.time_ranges_zero, flags.time_ranges_zero);
    ASSERT_EQ(decoded.all_channels_present, flags.all_channels_present);
    ASSERT_EQ(decoded.equal_alignments, flags.equal_alignments);
    ASSERT_EQ(decoded.zero_alignments, flags.zero_alignments);
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

/// @brief it should return a validation error when the data type of a series does not
/// match that of the channel.
TEST(CodecTests, EncodeMismatchedDataType) {
    const std::vector data_types = {
        x::telem::FLOAT32_T,
        x::telem::FLOAT64_T,
        x::telem::INT32_T
    };
    const std::vector<channel::Key> channels = {65537, 65538, 65539};
    Codec codec(channels, data_types);

    // Create a frame with mismatched data types
    auto frame = x::telem::Frame(1);
    // Using INT32_T instead of FLOAT32_T for channel 65537
    auto series = x::telem::Series(std::vector{1, 2, 3});
    series.time_range = {x::telem::TimeStamp(1000), x::telem::TimeStamp(2000)};
    series.alignment = x::telem::Alignment(10);
    frame.emplace(65537, std::move(series));

    std::vector<uint8_t> encoded;
    auto err = codec.encode(frame, encoded);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    ASSERT_TRUE(err.message().find("data type") != std::string::npos);
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
    ASSERT_TRUE(err.message().find("extra key") != std::string::npos);
}
}
