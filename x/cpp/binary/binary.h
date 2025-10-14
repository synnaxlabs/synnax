// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

// std
#include <cstdint>
#include <cstring>
#include <vector>

namespace binary {
/// @brief A simple binary writer to help with frame encoding
class Writer {
    std::vector<uint8_t> &buf;
    size_t offset;

public:
    /// @brief Creates a new binary writer that writes to an existing buffer
    /// @param buffer The buffer to write to
    /// @param size The size to resize the buffer to
    /// @param starting_offset The starting offset
    Writer(
        std::vector<uint8_t> &buffer,
        const size_t size,
        const size_t starting_offset = 0
    ):
        buf(buffer), offset(starting_offset) {
        buf.resize(size);
    }

    /// @brief Writes a byte to the buffer
    /// @param value The byte to write
    void uint8(const uint8_t value) { buf[offset++] = value; }

    /// @brief Writes a 32-bit unsigned integer to the buffer
    /// @param value The uint32 to write
    void uint32(const uint32_t value) {
        this->uint8(static_cast<uint8_t>(value));
        this->uint8(static_cast<uint8_t>(value >> 8));
        this->uint8(static_cast<uint8_t>(value >> 16));
        this->uint8(static_cast<uint8_t>(value >> 24));
    }

    /// @brief Writes a 64-bit unsigned integer to the buffer
    /// @param value The uint64 to write
    void uint64(const uint64_t value) {
        this->uint8(static_cast<uint8_t>(value));
        this->uint8(static_cast<uint8_t>(value >> 8));
        this->uint8(static_cast<uint8_t>(value >> 16));
        this->uint8(static_cast<uint8_t>(value >> 24));
        this->uint8(static_cast<uint8_t>(value >> 32));
        this->uint8(static_cast<uint8_t>(value >> 40));
        this->uint8(static_cast<uint8_t>(value >> 48));
        this->uint8(static_cast<uint8_t>(value >> 56));
    }

    /// @brief Writes a 64-bit signed integer to the buffer
    /// @param value The int64 to write
    void int64(const int64_t value) { this->uint64(static_cast<uint64_t>(value)); }

    /// @brief Writes raw bytes to the buffer
    /// @param data The bytes to write
    /// @param size The number of bytes to write
    void write(const void *data, const size_t size) {
        std::memcpy(buf.data() + offset, data, size);
        offset += size;
    }

    /// @brief Returns the buffer
    /// @return The buffer as a byte vector
    [[nodiscard]] std::vector<uint8_t> &bytes() const { return this->buf; }
};

class Reader {
    const uint8_t *buf;
    const size_t size;
    size_t offset;

public:
    explicit Reader(
        const std::vector<uint8_t> &buffer,
        const size_t starting_offset = 0
    ):
        Reader(buffer.data(), buffer.size(), starting_offset) {}

    Reader(const uint8_t *buffer, const size_t size, const size_t starting_offset = 0):
        buf(buffer), size(size), offset(starting_offset) {}

    /// @brief Reads a byte from the buffer
    /// @return The byte read
    [[nodiscard]] uint8_t uint8() { return buf[offset++]; }

    /// @brief Reads a 32-bit unsigned integer from the buffer
    /// @return The uint32 read
    [[nodiscard]] uint32_t uint32() {
        uint32_t value = 0;
        value |= static_cast<uint32_t>(this->uint8());
        value |= static_cast<uint32_t>(this->uint8()) << 8;
        value |= static_cast<uint32_t>(this->uint8()) << 16;
        value |= static_cast<uint32_t>(this->uint8()) << 24;
        return value;
    }

    /// @brief Reads a 64-bit unsigned integer from the buffer
    /// @return The uint64 read
    [[nodiscard]] uint64_t uint64() {
        uint64_t value = 0;
        value |= static_cast<uint64_t>(this->uint8());
        value |= static_cast<uint64_t>(this->uint8()) << 8;
        value |= static_cast<uint64_t>(this->uint8()) << 16;
        value |= static_cast<uint64_t>(this->uint8()) << 24;
        value |= static_cast<uint64_t>(this->uint8()) << 32;
        value |= static_cast<uint64_t>(this->uint8()) << 40;
        value |= static_cast<uint64_t>(this->uint8()) << 48;
        value |= static_cast<uint64_t>(this->uint8()) << 56;
        return value;
    }

    /// @brief Reads a 64-bit signed integer from the buffer
    /// @return The int64 read
    [[nodiscard]] int64_t int64() { return static_cast<int64_t>(this->uint64()); }

    [[nodiscard]] bool exhausted() const { return offset >= this->size; }

    /// @brief Reads raw bytes from the buffer into a provided memory location
    /// @param data Pointer to the memory location to write to
    /// @param size The number of bytes to read
    size_t read(void *data, const size_t size) {
        size_t read_size = size;
        if (offset + size > this->size) read_size = this->size - offset;
        std::memcpy(data, buf + offset, read_size);
        offset += read_size;
        return read_size;
    }
};

template<typename T>
bool get_bit(const uint8_t byte, const T pos) {
    const auto uint8_pos = static_cast<uint8_t>(pos);
    return (byte >> uint8_pos) & 1;
}

template<typename T>
uint8_t set_bit(const uint8_t byte, const T pos, const bool value) {
    const auto uint8_pos = static_cast<uint8_t>(pos);
    if (value) return byte | 1 << uint8_pos;
    return byte & ~(1 << uint8_pos);
}
}
