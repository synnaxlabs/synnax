// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.
// ReSharper disable CppUseStructuredBinding

/// std
#include <sstream>
#include <vector>

/// external
#include "gtest/gtest.h"

/// internal
#include "client/cpp/framer/framer.h"
#include "x/cpp/xtest/xtest.h"

synnax::Frame create_test_frame() {
    auto frame = synnax::Frame(3);
    auto s1 = telem::Series(std::vector{1.0f, 2.0f, 3.0f});
    s1.alignment = 10;
    s1.time_range = {telem::TimeStamp(1000), telem::TimeStamp(2000)};
    frame.emplace(65537, std::move(s1));

    auto s2 = telem::Series(std::vector{4.0, 5.0, 6.0});
    s2.alignment = 20;
    s2.time_range = {telem::TimeStamp(1000), telem::TimeStamp(2000)};
    frame.emplace(65538, std::move(s2));
    
    auto s3 = telem::Series(std::vector{7, 8, 9});
    s3.alignment = 30;
    s3.time_range = {telem::TimeStamp(1500), telem::TimeStamp(2500)};
    frame.emplace(65539, std::move(s3));
    
    return frame;
}

synnax::Frame create_equal_properties_frame() {
    auto frame = synnax::Frame(3);
    
    auto tr = telem::TimeRange{telem::TimeStamp(1000), telem::TimeStamp(2000)};
    uint64_t alignment = 10;
    
    auto s1 = telem::Series(std::vector{1.0f, 2.0f, 3.0f});
    s1.alignment = alignment;
    s1.time_range = tr;
    frame.emplace(65537, std::move(s1));
    
    auto s2 = telem::Series(std::vector{4.0f, 5.0f, 6.0f});
    s2.alignment = alignment;
    s2.time_range = tr;
    frame.emplace(65538, std::move(s2));
    
    auto s3 = telem::Series(std::vector{7.0f, 8.0f, 9.0f});
    s3.alignment = alignment;
    s3.time_range = tr;
    frame.emplace(65539, std::move(s3));
    
    return frame;
}

synnax::Frame create_zero_properties_frame() {
    auto frame = synnax::Frame(3);
    
    auto tr = telem::TimeRange{telem::TimeStamp(0), telem::TimeStamp(0)};
    uint64_t alignment = 0;
    
    auto s1 = telem::Series(std::vector{1.0f, 2.0f, 3.0f});
    s1.alignment = alignment;
    s1.time_range = tr;
    frame.emplace(65537, std::move(s1));
    
    auto s2 = telem::Series(std::vector{4.0f, 5.0f, 6.0f});
    s2.alignment = alignment;
    s2.time_range = tr;
    frame.emplace(65538, std::move(s2));
    
    auto s3 = telem::Series(std::vector{7.0f, 8.0f, 9.0f});
    s3.alignment = alignment;
    s3.time_range = tr;
    frame.emplace(65539, std::move(s3));
    
    return frame;
}

synnax::Frame create_diff_lengths_frame() {
    auto frame = synnax::Frame(3);
    
    auto tr = telem::TimeRange{telem::TimeStamp(1000), telem::TimeStamp(2000)};
    uint64_t alignment = 10;
    
    auto s1 = telem::Series(std::vector{1.0f, 2.0f, 3.0f});
    s1.alignment = alignment;
    s1.time_range = tr;
    frame.emplace(65537, std::move(s1));
    
    auto s2 = telem::Series(std::vector{4.0f, 5.0f, 6.0f, 7.0f});
    s2.alignment = alignment;
    s2.time_range = tr;
    frame.emplace(65538, std::move(s2));
    
    auto s3 = telem::Series(std::vector{7.0f, 8.0f});
    s3.alignment = alignment;
    s3.time_range = tr;
    frame.emplace(65539, std::move(s3));
    
    return frame;
}

// Helper function to verify that two frames are equal
void assert_frames_equal(const synnax::Frame& expected, const synnax::Frame& actual) {
    ASSERT_EQ(expected.size(), actual.size());
    
    for (size_t i = 0; i < expected.channels->size(); i++) {
        auto expected_key = expected.channels->at(i);
        
        auto it = std::find(actual.channels->begin(), actual.channels->end(), expected_key);
        ASSERT_NE(it, actual.channels->end())
            << "Channel key not found: " << expected_key;

        const size_t idx = std::distance(actual.channels->begin(), it);
        const auto& expected_series = expected.series->at(i);
        const auto& actual_series = actual.series->at(idx);
        
        ASSERT_EQ(expected_series.data_type(), actual_series.data_type());
        ASSERT_EQ(expected_series.size(), actual_series.size());
        ASSERT_EQ(expected_series.byte_size(), actual_series.byte_size());
        ASSERT_EQ(expected_series.alignment, actual_series.alignment);
        ASSERT_EQ(expected_series.time_range.start, actual_series.time_range.start);
        ASSERT_EQ(expected_series.time_range.end, actual_series.time_range.end);
        
        ASSERT_EQ(0, std::memcmp(expected_series.data(), actual_series.data(),
                                 expected_series.byte_size()));
    }
}

synnax::Frame create_large_equal_frame() {
    constexpr size_t NUM_CHANNELS = 500;
    auto frame = synnax::Frame(NUM_CHANNELS);
    auto tr = telem::TimeRange{telem::TimeStamp(1000), telem::TimeStamp(2000)};
    for (size_t i = 0; i < NUM_CHANNELS; i++) {
        uint64_t alignment = 10;
        auto series = telem::Series(std::vector{1.0f, 2.0f, 3.0f});
        series.alignment = alignment;
        series.time_range = tr;
        frame.emplace(65537 + i, std::move(series));
    }
    return frame;
}

/// @brief Test codec flags encoding and decoding
TEST(CodecTests, FlagsEncodingDecoding) {
    synnax::CodecFlags flags;
    flags.equal_lens = true;
    flags.equal_time_ranges = false;
    flags.time_ranges_zero = false;
    flags.all_channels_present = true;
    flags.equal_alignments = true;
    flags.zero_alignments = false;

    const uint8_t encoded = flags.encode();
    synnax::CodecFlags decoded = synnax::CodecFlags::decode(encoded);
    
    ASSERT_EQ(decoded.equal_lens, flags.equal_lens);
    ASSERT_EQ(decoded.equal_time_ranges, flags.equal_time_ranges);
    ASSERT_EQ(decoded.time_ranges_zero, flags.time_ranges_zero);
    ASSERT_EQ(decoded.all_channels_present, flags.all_channels_present);
    ASSERT_EQ(decoded.equal_alignments, flags.equal_alignments);
    ASSERT_EQ(decoded.zero_alignments, flags.zero_alignments);
}

/// @brief Test encoding and decoding of a frame with various data types and properties
TEST(CodecTests, EncodeDecodeVariedFrame) {
    const auto original_frame = create_test_frame();
    const std::vector data_types = {
        telem::FLOAT32_T,
        telem::FLOAT64_T,
        telem::INT32_T
    };
    const std::vector<synnax::ChannelKey> channels = {65537, 65538, 65539};
    synnax::Codec codec(data_types, channels);
    std::vector<uint8_t> encoded;
    codec.encode(original_frame, 0, encoded);
    const synnax::Frame decoded_frame = ASSERT_NIL_P(codec.decode(encoded));
    assert_frames_equal(original_frame, decoded_frame);
}

/// @brief Test encoding and decoding of a frame with equal properties
TEST(CodecTests, EncodeDecodeEqualPropertiesFrame) {
    const auto original_frame = create_equal_properties_frame();
    const std::vector data_types = {
        telem::FLOAT32_T,
        telem::FLOAT32_T,
        telem::FLOAT32_T
    };
    const std::vector<synnax::ChannelKey> channels = {65537, 65538, 65539};
    synnax::Codec codec(data_types, channels);
    std::vector<uint8_t> encoded;
    codec.encode(original_frame, 0, encoded);
    const synnax::Frame decoded_frame = ASSERT_NIL_P(codec.decode(encoded));
    assert_frames_equal(original_frame, decoded_frame);
}

/// @brief Test encoding and decoding with zero properties (optimized encoding)
TEST(CodecTests, EncodeDecodeZeroPropertiesFrame) {
    const auto original_frame = create_zero_properties_frame();
    const std::vector data_types = {
        telem::FLOAT32_T, telem::FLOAT32_T, telem::FLOAT32_T
    };
    std::vector<synnax::ChannelKey> channels = {65537, 65538, 65539};
    synnax::Codec codec(data_types, channels);
    std::vector<uint8_t> encoded;
    codec.encode(original_frame, 0, encoded);
    const synnax::Frame decoded_frame = ASSERT_NIL_P(codec.decode(encoded));
    assert_frames_equal(original_frame, decoded_frame);
}

/// @brief Test encoding and decoding with different length series
TEST(CodecTests, EncodeDecodeDifferentLengthsFrame) {
    const auto original_frame = create_diff_lengths_frame();
    const std::vector data_types = {
        telem::FLOAT32_T,
        telem::FLOAT32_T,
        telem::FLOAT32_T
    };
    const std::vector<synnax::ChannelKey> channels = {65537, 65538, 65539};
    synnax::Codec codec(data_types, channels);
    std::vector<uint8_t> encoded;
    codec.encode(original_frame, 0, encoded);
    const synnax::Frame decoded_frame = ASSERT_NIL_P(codec.decode(encoded));
    assert_frames_equal(original_frame, decoded_frame);
}

/// @brief Test encoding and decoding with subset of channels
TEST(CodecTests, EncodeDecodeChannelSubset) {
    const auto original_frame = create_test_frame();
    const std::vector data_types =
        {telem::FLOAT32_T, telem::FLOAT64_T, telem::INT32_T, telem::FLOAT32_T};
    const std::vector<synnax::ChannelKey> channels = {65537, 65538, 65539, 65540};
    synnax::Codec codec(data_types, channels);
    std::vector<uint8_t> encoded;
    codec.encode(original_frame, 0, encoded);
    const synnax::Frame decoded_frame = ASSERT_NIL_P(codec.decode(encoded));
    assert_frames_equal(original_frame, decoded_frame);
}

/// @brief Test encoding with a start offset
TEST(CodecTests, EncodeWithOffset) {
    const auto original_frame = create_test_frame();
    const std::vector data_types = {telem::FLOAT32_T, telem::FLOAT64_T, telem::INT32_T};
    const std::vector<synnax::ChannelKey> channels = {65537, 65538, 65539};
    synnax::Codec codec(data_types, channels);
    constexpr size_t offset = 10;
    std::vector<uint8_t> encoded;
    codec.encode(original_frame, offset, encoded);
    ASSERT_GT(encoded.size(), offset);
    const std::vector without_offset(encoded.begin() + offset, encoded.end());
    const synnax::Frame decoded_frame = ASSERT_NIL_P(codec.decode(without_offset));
    assert_frames_equal(original_frame, decoded_frame);
}

/// @brief Test with a large frame to ensure robustness
TEST(CodecTests, LargeFrame) {
    const auto frame = synnax::Frame(1);
    std::vector large_data(100000, 3.14159f);
    auto large_series = telem::Series(large_data);
    large_series.time_range = {telem::TimeStamp(1000), telem::TimeStamp(2000)};
    large_series.alignment = 42;
    frame.emplace(65537, std::move(large_series));
    const std::vector data_types = {telem::FLOAT32_T};
    std::vector<synnax::ChannelKey> channels = {65537};
    synnax::Codec codec(data_types, channels);
    std::vector<uint8_t> encoded;
    codec.encode(frame, 0, encoded);
    const synnax::Frame decoded_frame = ASSERT_NIL_P(codec.decode(encoded));
    assert_frames_equal(frame, decoded_frame);
}

// Replace the performance test with this updated version
// TEST(CodecTests, TestPerformance) {
//     auto start = telem::TimeStamp::now();
//     const auto original_frame = create_large_equal_frame();
//
//     std::vector data_types(500, telem::FLOAT32_T);
//     std::vector<synnax::ChannelKey> channels;
//     channels.reserve(500);
//     for (size_t i = 0; i < 500; i++)
//         channels.push_back(65537 + i);
//
//     synnax::Codec codec(data_types, channels);
//     constexpr size_t count = 1e3;
//     std::vector<uint8_t> encoded;
//     for (size_t i = 0; i < count; ++i) {
//         codec.encode(original_frame, 0, encoded);
//         synnax::Frame decoded_frame = codec.decode(encoded);
//     }
//     auto end = telem::TimeStamp::now();
//     auto duration = end - start;
//     std::cout << encoded.size() << " bytes encoded" << std::endl;
//     std::cout << "Performance test duration: " << duration / count << " ms" << std::endl;
//
//     start = telem::TimeStamp::now();
//     std::vector<uint8_t> encoded2;
//     for (size_t i = 0; i < count; ++i) {
//         const auto p = new api::v1::Frame();
//         original_frame.to_proto(p);
//         const size_t size = p->ByteSizeLong();
//         encoded2.resize(size);
//         p->SerializeToArray(encoded2.data(), size);
//         const auto l = synnax::Frame(*p);
//         delete p;
//     }
//     end = telem::TimeStamp::now();
//     duration = end - start;
//     std::cout << encoded2.size() << " bytes encoded" << std::endl;
//     std::cout << "Performance test duration: " << duration / count << " ms" << std::endl;
// }