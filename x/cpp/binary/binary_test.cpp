// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/binary/binary.h"

TEST(BinaryWriter, testUint8Write) {
    std::vector<uint8_t> buffer;
    binary::Writer writer(buffer, 3);

    writer.uint8(0x12);
    writer.uint8(0x34);
    writer.uint8(0x56);

    ASSERT_EQ(buffer.size(), 3);
    ASSERT_EQ(buffer[0], 0x12);
    ASSERT_EQ(buffer[1], 0x34);
    ASSERT_EQ(buffer[2], 0x56);
}

TEST(BinaryWriter, testUint32Write) {
    std::vector<uint8_t> buffer;
    binary::Writer writer(buffer, 5);

    writer.uint32(0x12345678);

    ASSERT_EQ(buffer.size(), 5);
    ASSERT_EQ(buffer[0], 0x78);
    ASSERT_EQ(buffer[1], 0x56);
    ASSERT_EQ(buffer[2], 0x34);
    ASSERT_EQ(buffer[3], 0x12);
    ASSERT_EQ(buffer[4], 0x00);
}

TEST(BinaryWriter, testUint64Write) {
    std::vector<uint8_t> buffer;
    binary::Writer writer(buffer, 8);

    writer.uint64(0x1234567890ABCDEF);

    ASSERT_EQ(buffer.size(), 8);
    ASSERT_EQ(buffer[0], 0xEF);
    ASSERT_EQ(buffer[1], 0xCD);
    ASSERT_EQ(buffer[2], 0xAB);
    ASSERT_EQ(buffer[3], 0x90);
    ASSERT_EQ(buffer[4], 0x78);
    ASSERT_EQ(buffer[5], 0x56);
    ASSERT_EQ(buffer[6], 0x34);
    ASSERT_EQ(buffer[7], 0x12);
}

TEST(BinaryWriter, testRawWrite) {
    std::vector<uint8_t> buffer;
    binary::Writer writer(buffer, 5);

    const uint8_t data[] = {0x12, 0x34, 0x56, 0x78, 0x90};
    writer.write(data, 5);

    ASSERT_EQ(buffer.size(), 5);
    ASSERT_EQ(buffer[0], 0x12);
    ASSERT_EQ(buffer[1], 0x34);
    ASSERT_EQ(buffer[2], 0x56);
    ASSERT_EQ(buffer[3], 0x78);
    ASSERT_EQ(buffer[4], 0x90);
}

TEST(BinaryWriter, testMultipleWrites) {
    std::vector<uint8_t> buffer;
    binary::Writer writer(buffer, 16);

    writer.uint8(0x01);
    writer.uint32(0x12345678);

    constexpr uint8_t data[] = {0xAA, 0xBB, 0xCC};
    writer.write(data, 3);

    writer.uint64(0x1122334455667788);

    ASSERT_EQ(buffer.size(), 16);
    ASSERT_EQ(buffer[0], 0x01);
    ASSERT_EQ(buffer[1], 0x78);
    ASSERT_EQ(buffer[2], 0x56);
    ASSERT_EQ(buffer[3], 0x34);
    ASSERT_EQ(buffer[4], 0x12);
    ASSERT_EQ(buffer[5], 0xAA);
    ASSERT_EQ(buffer[6], 0xBB);
    ASSERT_EQ(buffer[7], 0xCC);
    ASSERT_EQ(buffer[8], 0x88);
    ASSERT_EQ(buffer[9], 0x77);
    ASSERT_EQ(buffer[10], 0x66);
    ASSERT_EQ(buffer[11], 0x55);
    ASSERT_EQ(buffer[12], 0x44);
    ASSERT_EQ(buffer[13], 0x33);
    ASSERT_EQ(buffer[14], 0x22);
    ASSERT_EQ(buffer[15], 0x11);
}

TEST(BinaryWriter, testPartialWrite) {
    std::vector<uint8_t> buffer;
    binary::Writer writer(buffer, 15); // Buffer too small for all writes

    ASSERT_EQ(writer.uint8(0x01), 1);
    ASSERT_EQ(writer.uint32(0x12345678), 4);

    constexpr uint8_t data[] = {0xAA, 0xBB, 0xCC};
    ASSERT_EQ(writer.write(data, 3), 3);

    // Trying to write 8 more bytes but only 7 remain - partial write
    const size_t written = writer.uint64(0x1122334455667788);
    ASSERT_EQ(written, 7);

    // Verify the 7 bytes that were written
    ASSERT_EQ(buffer[8], 0x88);
    ASSERT_EQ(buffer[9], 0x77);
    ASSERT_EQ(buffer[10], 0x66);
    ASSERT_EQ(buffer[11], 0x55);
    ASSERT_EQ(buffer[12], 0x44);
    ASSERT_EQ(buffer[13], 0x33);
    ASSERT_EQ(buffer[14], 0x22);
}

TEST(BinaryWriter, testStartingOffset) {
    std::vector<uint8_t> buffer;
    constexpr size_t offset = 3;
    buffer.resize(offset);
    for (size_t i = 0; i < offset; i++)
        buffer[i] = static_cast<uint8_t>(i + 1);
    binary::Writer writer(buffer, offset + 5, offset);

    ASSERT_EQ(writer.uint8(0xAA), 1);
    ASSERT_EQ(writer.uint32(0xBBCCDDEE), 4);

    ASSERT_EQ(buffer.size(), offset + 5);
    ASSERT_EQ(buffer[0], 0x01);
    ASSERT_EQ(buffer[1], 0x02);
    ASSERT_EQ(buffer[2], 0x03);
    ASSERT_EQ(buffer[3], 0xAA);
    ASSERT_EQ(buffer[4], 0xEE);
    ASSERT_EQ(buffer[5], 0xDD);
    ASSERT_EQ(buffer[6], 0xCC);
    ASSERT_EQ(buffer[7], 0xBB);
}

TEST(BinaryWriter, testBufferFull) {
    std::vector<uint8_t> buffer;
    binary::Writer writer(buffer, 5);

    // Write 5 bytes successfully
    ASSERT_EQ(writer.uint8(0x01), 1);
    ASSERT_EQ(writer.uint32(0x12345678), 4);

    // Buffer is now full - attempting to write should return 0
    ASSERT_EQ(writer.uint8(0xFF), 0);
}

TEST(BinaryWriter, testRawWritePartial) {
    std::vector<uint8_t> buffer;
    binary::Writer writer(buffer, 3);

    const uint8_t data[] = {0x01, 0x02, 0x03, 0x04, 0x05};

    // Attempting to write 5 bytes into 3-byte buffer - should write 3
    const size_t written = writer.write(data, 5);
    ASSERT_EQ(written, 3);
    ASSERT_EQ(buffer[0], 0x01);
    ASSERT_EQ(buffer[1], 0x02);
    ASSERT_EQ(buffer[2], 0x03);
}

TEST(BinaryReader, testUint8Read) {
    std::vector<uint8_t> buffer = {0x12, 0x34, 0x56};
    binary::Reader reader(buffer);

    ASSERT_EQ(reader.uint8(), 0x12);
    ASSERT_EQ(reader.uint8(), 0x34);
    ASSERT_EQ(reader.uint8(), 0x56);
}

TEST(BinaryReader, testUint32Read) {
    std::vector<uint8_t> buffer = {0x78, 0x56, 0x34, 0x12, 0x00};
    binary::Reader reader(buffer);

    ASSERT_EQ(reader.uint32(), 0x12345678);
}

TEST(BinaryReader, testUint64Read) {
    std::vector<uint8_t> buffer = {0xEF, 0xCD, 0xAB, 0x90, 0x78, 0x56, 0x34, 0x12};
    binary::Reader reader(buffer);

    ASSERT_EQ(reader.uint64(), 0x1234567890ABCDEF);
}

TEST(BinaryReader, testRawRead) {
    std::vector<uint8_t> buffer = {0x12, 0x34, 0x56, 0x78, 0x90};
    binary::Reader reader(buffer);

    uint8_t data[5] = {};
    reader.read(data, 5);

    ASSERT_EQ(data[0], 0x12);
    ASSERT_EQ(data[1], 0x34);
    ASSERT_EQ(data[2], 0x56);
    ASSERT_EQ(data[3], 0x78);
    ASSERT_EQ(data[4], 0x90);
}

TEST(BinaryReader, testMultipleReads) {
    std::vector<uint8_t> buffer = {
        0x01, // uint8
        0x78,
        0x56,
        0x34,
        0x12, // uint32
        0xAA,
        0xBB,
        0xCC, // raw bytes
        0x88,
        0x77,
        0x66,
        0x55,
        0x44,
        0x33,
        0x22,
        0x11 // uint64
    };

    binary::Reader reader(buffer);

    ASSERT_EQ(reader.uint8(), 0x01);
    ASSERT_EQ(reader.uint32(), 0x12345678);

    uint8_t data[3] = {};
    reader.read(data, 3);
    ASSERT_EQ(data[0], 0xAA);
    ASSERT_EQ(data[1], 0xBB);
    ASSERT_EQ(data[2], 0xCC);

    ASSERT_EQ(reader.uint64(), 0x1122334455667788);
}

TEST(BinaryReader, testStartingOffset) {
    std::vector<uint8_t> buffer = {
        0x01,
        0x02,
        0x03, // initial bytes to skip
        0xAA, // uint8
        0xEE,
        0xDD,
        0xCC,
        0xBB,
        0x00 // uint32
    };

    constexpr size_t offset = 3;
    binary::Reader reader(buffer, offset);

    ASSERT_EQ(reader.uint8(), 0xAA);
    ASSERT_EQ(reader.uint32(), 0xBBCCDDEE);
}

TEST(BinaryRoundTrip, testReadWriteRoundTrip) {
    std::vector<uint8_t> buffer;
    constexpr size_t size = 17;
    binary::Writer writer(buffer, size);

    writer.uint8(0x01);
    writer.uint32(0x12345678);
    writer.uint64(0x1122334455667788);

    constexpr uint8_t raw_data[] = {0xAA, 0xBB, 0xCC, 0xDD};
    writer.write(raw_data, 4);

    binary::Reader reader(buffer);

    ASSERT_EQ(reader.uint8(), 0x01);
    ASSERT_EQ(reader.uint32(), 0x12345678);
    ASSERT_EQ(reader.uint64(), 0x1122334455667788);

    uint8_t read_raw_data[4] = {};
    reader.read(read_raw_data, 4);

    ASSERT_EQ(read_raw_data[0], 0xAA);
    ASSERT_EQ(read_raw_data[1], 0xBB);
    ASSERT_EQ(read_raw_data[2], 0xCC);
    ASSERT_EQ(read_raw_data[3], 0xDD);
}

TEST(BitUtils, testGetBit) {
    uint8_t byte = 0b10101010;

    ASSERT_FALSE(binary::get_bit(byte, 0));
    ASSERT_TRUE(binary::get_bit(byte, 1));
    ASSERT_FALSE(binary::get_bit(byte, 2));
    ASSERT_TRUE(binary::get_bit(byte, 3));
    ASSERT_FALSE(binary::get_bit(byte, 4));
    ASSERT_TRUE(binary::get_bit(byte, 5));
    ASSERT_FALSE(binary::get_bit(byte, 6));
    ASSERT_TRUE(binary::get_bit(byte, 7));
}

TEST(BitUtils, testSetBit) {
    uint8_t byte = 0b00000000;

    byte = binary::set_bit(byte, 0, true);
    ASSERT_EQ(byte, 0b00000001);

    byte = binary::set_bit(byte, 1, true);
    ASSERT_EQ(byte, 0b00000011);

    byte = binary::set_bit(byte, 7, true);
    ASSERT_EQ(byte, 0b10000011);

    byte = binary::set_bit(byte, 0, false);
    ASSERT_EQ(byte, 0b10000010);

    byte = binary::set_bit(byte, 7, false);
    ASSERT_EQ(byte, 0b00000010);
}

TEST(BitUtils, testSetBitNoChangeWhenSameValue) {
    uint8_t byte = 0b10101010;

    uint8_t result = binary::set_bit(byte, 0, false);
    ASSERT_EQ(result, byte);

    result = binary::set_bit(byte, 1, true);
    ASSERT_EQ(result, byte);
}

TEST(BitUtils, testFlipAllBits) {
    uint8_t original = 0b10101010;
    uint8_t flipped = original;
    for (uint8_t i = 0; i < 8; i++)
        flipped = binary::set_bit(flipped, i, !binary::get_bit(flipped, i));
    ASSERT_EQ(flipped, 0b01010101);
}

// This test stresses the encoding/decoding of different byte patterns
TEST(BinaryStressTest, testVariousBytePatterns) {
    constexpr uint64_t test_values[] = {
        0,
        1,
        0xFF,
        0xFFFF,
        0xFFFFFFFF,
        0xFFFFFFFFFFFFFFFF,
        0x1234567890ABCDEF,
        0x0F0F0F0F0F0F0F0F,
        0xF0F0F0F0F0F0F0F0
    };

    for (const auto &value: test_values) {
        std::vector<uint8_t> buffer;
        binary::Writer writer(buffer, 8);
        writer.uint64(value);

        binary::Reader reader(buffer);
        uint64_t decoded = reader.uint64();

        ASSERT_EQ(decoded, value) << "Failed for value 0x" << std::hex << value;
    }
}
