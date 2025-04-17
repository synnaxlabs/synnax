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

namespace {

// Helper function to create a test frame with different data types and configurations
synnax::Frame createTestFrame() {
    auto frame = synnax::Frame(3);
    
    // Add float series with alignment and time range
    auto s1 = telem::Series(std::vector{1.0f, 2.0f, 3.0f});
    s1.alignment = 10;
    s1.time_range = {telem::TimeStamp(1000), telem::TimeStamp(2000)};
    frame.emplace(65537, std::move(s1));
    
    // Add double series with different alignment
    auto s2 = telem::Series(std::vector{4.0, 5.0, 6.0});
    s2.alignment = 20;
    s2.time_range = {telem::TimeStamp(1000), telem::TimeStamp(2000)};
    frame.emplace(65538, std::move(s2));
    
    // Add int series with different time range
    auto s3 = telem::Series(std::vector{7, 8, 9});
    s3.alignment = 30;
    s3.time_range = {telem::TimeStamp(1500), telem::TimeStamp(2500)};
    frame.emplace(65539, std::move(s3));
    
    return frame;
}

// Helper function to create a test frame with all equal properties
synnax::Frame createEqualPropertiesFrame() {
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

// Helper function to create a frame with zero time ranges and alignments
synnax::Frame createZeroPropertiesFrame() {
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

// Helper function to create a frame with different length series
synnax::Frame createDifferentLengthsFrame() {
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
void assertFramesEqual(const synnax::Frame& expected, const synnax::Frame& actual) {
    ASSERT_EQ(expected.size(), actual.size());
    
    for (size_t i = 0; i < expected.channels->size(); i++) {
        auto expected_key = expected.channels->at(i);
        
        // Find matching key in actual frame
        auto it = std::find(actual.channels->begin(), actual.channels->end(), expected_key);
        ASSERT_NE(it, actual.channels->end())
            << "Channel key not found: " << expected_key;

        const size_t idx = std::distance(actual.channels->begin(), it);
        const auto& expected_series = expected.series->at(i);
        const auto& actual_series = actual.series->at(idx);
        
        // Check series properties
        ASSERT_EQ(expected_series.data_type(), actual_series.data_type());
        ASSERT_EQ(expected_series.size(), actual_series.size());
        ASSERT_EQ(expected_series.byte_size(), actual_series.byte_size());
        ASSERT_EQ(expected_series.alignment, actual_series.alignment);
        ASSERT_EQ(expected_series.time_range.start, actual_series.time_range.start);
        ASSERT_EQ(expected_series.time_range.end, actual_series.time_range.end);
        
        // Check series data (binary comparison)
        ASSERT_EQ(0, std::memcmp(expected_series.data(), actual_series.data(), 
                                 expected_series.byte_size()));
    }
}

// Add this new helper function before the test cases
synnax::Frame createLargeEqualFrame() {
    constexpr size_t NUM_CHANNELS = 500;
    constexpr size_t VALUES_PER_SERIES = 3;
    auto frame = synnax::Frame(NUM_CHANNELS);
    
    // Set common properties for all series
    auto tr = telem::TimeRange{telem::TimeStamp(1000), telem::TimeStamp(2000)};
    uint64_t alignment = 10;
    
    // Create series with float values
    for (size_t i = 0; i < NUM_CHANNELS; i++) {
        auto series = telem::Series(std::vector<float>{1.0f, 2.0f, 3.0f});
        series.alignment = alignment;
        series.time_range = tr;
        frame.emplace(65537 + i, std::move(series));
    }
    
    return frame;
}

}  // anonymous namespace

/// @brief Test codec construction with data types and channels
TEST(CodecTests, ConstructionFromDataTypesAndChannels) {
    const std::vector data_types = {telem::FLOAT32_T, telem::FLOAT64_T, telem::INT32_T};
    const std::vector<synnax::ChannelKey> channels = {65539, 65537, 65538};
    
    synnax::Codec codec(data_types, channels);
    
    const std::vector<synnax::ChannelKey> expected_keys = {65537, 65538, 65539};
    ASSERT_EQ(codec.keys, expected_keys);
    
    ASSERT_EQ(codec.key_data_types[65537], telem::FLOAT64_T);
    ASSERT_EQ(codec.key_data_types[65538], telem::INT32_T);
    ASSERT_EQ(codec.key_data_types[65539], telem::FLOAT32_T);
}

/// @brief Test codec construction from channels
TEST(CodecTests, ConstructionFromChannels) {
    std::vector channels = {
        synnax::Channel("ch1", telem::FLOAT32_T, false),
        synnax::Channel("ch2", telem::FLOAT64_T, false),
        synnax::Channel("ch3", telem::INT32_T, false)
    };
    
    // Set keys manually for testing
    channels[0].key = 65539;
    channels[1].key = 65537;
    channels[2].key = 65538;
    
    synnax::Codec codec(channels);
    
    // The keys should be sorted
    const std::vector<synnax::ChannelKey> expected_keys = {65537, 65538, 65539};
    ASSERT_EQ(codec.keys, expected_keys);
    
    // Check that data types are correctly mapped
    ASSERT_EQ(codec.key_data_types[65537], telem::FLOAT64_T);
    ASSERT_EQ(codec.key_data_types[65538], telem::INT32_T);
    ASSERT_EQ(codec.key_data_types[65539], telem::FLOAT32_T);
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
    // Create a frame with mixed properties
    const auto original_frame = createTestFrame();
    
    // Create a codec for all channels in the frame
    const std::vector data_types = {
        telem::FLOAT32_T,
        telem::FLOAT64_T,
        telem::INT32_T
    };
    const std::vector<synnax::ChannelKey> channels = {65537, 65538, 65539};
    synnax::Codec codec(data_types, channels);

    std::vector<uint8_t> encoded;
    codec.encode(original_frame, 0, encoded);
    const synnax::Frame decoded_frame = codec.decode(encoded);
    
    assertFramesEqual(original_frame, decoded_frame);
}

/// @brief Test encoding and decoding of a frame with equal properties
TEST(CodecTests, EncodeDecodeEqualPropertiesFrame) {
    // Create a frame where all series have the same properties
    const auto original_frame = createEqualPropertiesFrame();
    
    // Create a codec for all channels in the frame
    const std::vector data_types = {
        telem::FLOAT32_T,
        telem::FLOAT32_T,
        telem::FLOAT32_T
    };
    const std::vector<synnax::ChannelKey> channels = {65537, 65538, 65539};
    synnax::Codec codec(data_types, channels);
    
    // Encode the frame
    std::vector<uint8_t> encoded;
    codec.encode(original_frame, 0, encoded);
    
    // Decode the frame
    const synnax::Frame decoded_frame = codec.decode(encoded);
    
    // Verify that the decoded frame matches the original
    assertFramesEqual(original_frame, decoded_frame);
}

/// @brief Test encoding and decoding with zero properties (optimized encoding)
TEST(CodecTests, EncodeDecodeZeroPropertiesFrame) {
    const auto original_frame = createZeroPropertiesFrame();
    const std::vector data_types = {
        telem::FLOAT32_T, telem::FLOAT32_T, telem::FLOAT32_T
    };
    std::vector<synnax::ChannelKey> channels = {65537, 65538, 65539};
    synnax::Codec codec(data_types, channels);
    std::vector<uint8_t> encoded;
    codec.encode(original_frame, 0, encoded);
    const synnax::Frame decoded_frame = codec.decode(encoded);
    assertFramesEqual(original_frame, decoded_frame);
}

/// @brief Test encoding and decoding with different length series
TEST(CodecTests, EncodeDecodeDifferentLengthsFrame) {
    const auto original_frame = createDifferentLengthsFrame();
    const std::vector data_types = {
        telem::FLOAT32_T,
        telem::FLOAT32_T,
        telem::FLOAT32_T
    };
    const std::vector<synnax::ChannelKey> channels = {65537, 65538, 65539};
    synnax::Codec codec(data_types, channels);
    std::vector<uint8_t> encoded;
    codec.encode(original_frame, 0, encoded);
    const synnax::Frame decoded_frame = codec.decode(encoded);
    assertFramesEqual(original_frame, decoded_frame);
}

/// @brief Test encoding and decoding with subset of channels
TEST(CodecTests, EncodeDecodeChannelSubset) {
    const auto original_frame = createTestFrame();
    const std::vector data_types =
        {telem::FLOAT32_T, telem::FLOAT64_T, telem::INT32_T, telem::FLOAT32_T};
    const std::vector<synnax::ChannelKey> channels = {65537, 65538, 65539, 65540};
    synnax::Codec codec(data_types, channels);
    std::vector<uint8_t> encoded;
    codec.encode(original_frame, 0, encoded);
    const synnax::Frame decoded_frame = codec.decode(encoded);
    assertFramesEqual(original_frame, decoded_frame);
}

/// @brief Test direct stream decoding
TEST(CodecTests, StreamDecoding) {
    auto original_frame = createTestFrame();
    std::vector data_types = {
        telem::FLOAT32_T, telem::FLOAT64_T, telem::INT32_T
    };
    std::vector<synnax::ChannelKey> channels = {65537, 65538, 65539};
    synnax::Codec codec(data_types, channels);
    std::vector<uint8_t> encoded;
    codec.encode(original_frame, 0, encoded);
    ASSERT_FALSE(encoded.empty());
    ASSERT_GT(encoded.size(), 1);
    std::string buffer(reinterpret_cast<const char*>(encoded.data()), encoded.size());
    std::istringstream stream(buffer, std::ios::binary);
    ASSERT_TRUE(stream.good());
    synnax::Frame decoded_frame = codec.decode_stream(stream);
    assertFramesEqual(original_frame, decoded_frame);
}

/// @brief Test encoding with a start offset
TEST(CodecTests, EncodeWithOffset) {
    const auto original_frame = createTestFrame();
    const std::vector data_types = {telem::FLOAT32_T, telem::FLOAT64_T, telem::INT32_T};
    const std::vector<synnax::ChannelKey> channels = {65537, 65538, 65539};
    synnax::Codec codec(data_types, channels);
    constexpr size_t offset = 10;
    std::vector<uint8_t> encoded;
    codec.encode(original_frame, offset, encoded);
    ASSERT_GT(encoded.size(), offset);
    const std::vector without_offset(encoded.begin() + offset, encoded.end());
    const synnax::Frame decoded_frame = codec.decode(without_offset);
    assertFramesEqual(original_frame, decoded_frame);
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
    const synnax::Frame decoded_frame = codec.decode(encoded);
    assertFramesEqual(frame, decoded_frame);
}

/// @brief Test bit manipulation functions for flags
TEST(CodecTests, BitManipulation) {
    uint8_t byte = 0;
    
    byte = synnax::set_bit(byte, synnax::FlagPosition::EqualLengths, true);
    byte = synnax::set_bit(byte, synnax::FlagPosition::TimeRangesZero, true);
    
    ASSERT_EQ(byte, 1 << 3 | 1 << 1);
    
    ASSERT_TRUE(synnax::get_bit(byte, synnax::FlagPosition::EqualLengths));
    ASSERT_TRUE(synnax::get_bit(byte, synnax::FlagPosition::TimeRangesZero));
    ASSERT_FALSE(synnax::get_bit(byte, synnax::FlagPosition::EqualTimeRanges));
    
    byte = synnax::set_bit(byte, synnax::FlagPosition::EqualLengths, false);
    ASSERT_FALSE(synnax::get_bit(byte, synnax::FlagPosition::EqualLengths));
    ASSERT_TRUE(synnax::get_bit(byte, synnax::FlagPosition::TimeRangesZero));
}

/// @brief Test error handling for invalid data
TEST(CodecTests, ErrorHandlingInvalidData) {
    std::vector data_types = {telem::FLOAT32_T};
    std::vector<synnax::ChannelKey> channels = {65537};
    synnax::Codec codec(data_types, channels);
    std::vector<uint8_t> invalid_data = {0x01, 0x02, 0x03};
    ASSERT_THROW(codec.decode(invalid_data), std::runtime_error);
    std::stringstream bad_stream(std::ios::in | std::ios::binary);
    bad_stream.write(reinterpret_cast<const char*>(invalid_data.data()), invalid_data.size());
    ASSERT_THROW(codec.decode_stream(bad_stream), std::runtime_error);
}

// Replace the performance test with this updated version
TEST(CodecTests, TestPerformance) {
    auto start = telem::TimeStamp::now();
    const auto original_frame = createLargeEqualFrame();
    
    // Create data types and channels vectors for the codec
    std::vector<telem::DataType> data_types(500, telem::FLOAT32_T);
    std::vector<synnax::ChannelKey> channels;
    channels.reserve(500);
    for (size_t i = 0; i < 500; i++)
        channels.push_back(65537 + i);

    synnax::Codec codec(data_types, channels);
    constexpr size_t count = 1e3;
    std::vector<uint8_t> encoded;
    for (size_t i = 0; i < count; ++i) {
        codec.encode(original_frame, 0, encoded);
        synnax::Frame decoded_frame = codec.decode(encoded);
    }
    auto end = telem::TimeStamp::now();
    auto duration = end - start;
    std::cout << encoded.size() << " bytes encoded" << std::endl;
    std::cout << "Performance test duration: " << duration / count << " ms" << std::endl;

    start = telem::TimeStamp::now();
    std::vector<uint8_t> encoded2;
    for (size_t i = 0; i < count; ++i) {
        const auto p = new api::v1::Frame();
        original_frame.to_proto(p);
        const size_t size = p->ByteSizeLong();
        encoded2.resize(size);
        p->SerializeToArray(encoded2.data(), size);
        const auto l = synnax::Frame(*p);
        delete p;
    }
    end = telem::TimeStamp::now();
    duration = end - start;
    std::cout << encoded2.size() << " bytes encoded" << std::endl;
    std::cout << "Performance test duration: " << duration / count << " ms" << std::endl;
}