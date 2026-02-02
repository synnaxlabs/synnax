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

// Helper function to verify that two frames are equal
void assert_frames_equal(
    const x::telem::Frame &expected,
    const x::telem::Frame &actual
) {
    ASSERT_EQ(expected.size(), actual.size());

    for (size_t i = 0; i < expected.channels->size(); i++) {
        auto expected_key = expected.channels->at(i);

        auto it = std::find(
            actual.channels->begin(),
            actual.channels->end(),
            expected_key
        );
        ASSERT_NE(it, actual.channels->end())
            << "Channel key not found: " << expected_key;

        const size_t idx = std::distance(actual.channels->begin(), it);
        const auto &expected_series = expected.series->at(i);
        const auto &actual_series = actual.series->at(idx);

        ASSERT_EQ(expected_series.data_type(), actual_series.data_type());
        ASSERT_EQ(expected_series.size(), actual_series.size());
        ASSERT_EQ(expected_series.byte_size(), actual_series.byte_size());
        ASSERT_EQ(expected_series.alignment, actual_series.alignment);
        ASSERT_EQ(expected_series.time_range.start, actual_series.time_range.start);
        ASSERT_EQ(expected_series.time_range.end, actual_series.time_range.end);

        ASSERT_EQ(
            0,
            std::memcmp(
                expected_series.data(),
                actual_series.data(),
                expected_series.byte_size()
            )
        );
    }
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
    const auto frame = x::telem::Frame(1);
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

/// @brief it should allow the caller to dynamically update the keys fo the codec.
TEST(CodecTests, DynamicCodecUpdate) {
    auto client = new_test_client();

    auto [idx_ch, data_ch] = create_indexed_pair(client);
    Codec codec(client.channels);

    codec.update(std::vector{idx_ch.key});

    auto frame = x::telem::Frame(
        idx_ch.key,
        x::telem::Series(x::telem::TimeStamp(x::telem::SECOND))
    );

    std::vector<uint8_t> encoded;
    ASSERT_NIL(codec.encode(frame, encoded));
    auto decoded_frame = ASSERT_NIL_P(codec.decode(encoded));
    assert_frames_equal(frame, decoded_frame);

    codec.update(std::vector{data_ch.key});
    auto frame2 = x::telem::Frame(data_ch.key, x::telem::Series(1.0f));
    ASSERT_NIL(codec.encode(frame2, encoded));
    auto decoded_frame2 = ASSERT_NIL_P(codec.decode(encoded));
    assert_frames_equal(frame2, decoded_frame2);
}

/// @brief it should correctly encode/decode values when the codec are out of sync
TEST(CodecTests, UninitializedCodec) {
    auto client = new_test_client();
    Codec codec(client.channels);

    auto [idx_ch, _] = create_indexed_pair(client);
    auto frame = x::telem::Frame(
        idx_ch.key,
        x::telem::Series(x::telem::TimeStamp(x::telem::SECOND))
    );

    std::vector<uint8_t> encoded;
    ASSERT_THROW(codec.encode(frame, encoded), std::runtime_error);
}

/// @brief it should correctly manage the lifecycle of codecs that are temporarily
/// out of sync by using historical states.
TEST(CodecTests, OutOfSyncCodecs) {
    auto client = new_test_client();
    auto [idx_ch, data_ch] = create_indexed_pair(client);

    Codec encoder(client.channels);
    Codec decoder(client.channels);

    // Initial state - both in sync
    ASSERT_NIL(encoder.update(std::vector{idx_ch.key}));
    ASSERT_NIL(decoder.update(std::vector{idx_ch.key}));

    auto frame = x::telem::Frame(
        idx_ch.key,
        x::telem::Series(x::telem::TimeStamp(x::telem::SECOND))
    );

    std::vector<uint8_t> encoded;
    ASSERT_NIL(encoder.encode(frame, encoded));
    auto decoded_frame = ASSERT_NIL_P(decoder.decode(encoded));
    assert_frames_equal(frame, decoded_frame);

    // Decoder updates but encoder doesn't - should still work with old format
    ASSERT_NIL(decoder.update(std::vector{data_ch.key}));
    ASSERT_NIL(encoder.encode(frame, encoded));
    auto decoded_frame2 = ASSERT_NIL_P(decoder.decode(encoded));
    assert_frames_equal(frame, decoded_frame2);

    // Encoder updates - old frame should now fail
    ASSERT_NIL(encoder.update(std::vector{data_ch.key}));
    ASSERT_OCCURRED_AS(encoder.encode(frame, encoded), x::errors::VALIDATION);

    // New frame with updated channel should work
    auto frame2 = x::telem::Frame(data_ch.key, x::telem::Series(1.0f));
    ASSERT_NIL(encoder.encode(frame2, encoded));
    auto decoded_frame3 = ASSERT_NIL_P(decoder.decode(encoded));
    assert_frames_equal(frame2, decoded_frame3);
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
