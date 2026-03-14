// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <cstring>
#include <span>
#include <vector>

#include "driver/ethercat/virtual/frame.h"

namespace ethercat::virtual_esc {

/// @brief EtherCAT datagram command codes.
enum class Command : uint8_t {
    NOP = 0x00,
    APRD = 0x01,
    APWR = 0x02,
    APRW = 0x03,
    FPRD = 0x04,
    FPWR = 0x05,
    FPRW = 0x06,
    BRD = 0x07,
    BWR = 0x08,
    BRW = 0x09,
    LRD = 0x0A,
    LWR = 0x0B,
    LRW = 0x0C,
    ARMW = 0x0D,
    FRMW = 0x0E,
};

/// @brief Represents an EtherCAT datagram header and provides access to its data.
class Datagram {
public:
    /// @brief Creates a datagram from raw bytes at a given offset.
    /// @param frame_data The full frame data buffer.
    /// @param offset Offset within the datagram payload section.
    Datagram(std::span<uint8_t> frame_data, size_t offset):
        frame(frame_data), dgram_offset(offset) {}

    /// @brief Returns true if this datagram is valid.
    [[nodiscard]] bool valid() const {
        return this->frame.size() >= this->dgram_offset + MIN_DATAGRAM_SIZE &&
               this->data_length() + DATAGRAM_HEADER_SIZE + 2 <=
                   this->frame.size() - this->dgram_offset;
    }

    /// @brief Returns the command byte.
    [[nodiscard]] Command command() const {
        return static_cast<Command>(this->frame[this->dgram_offset]);
    }

    /// @brief Returns the index field (used by master to match responses).
    [[nodiscard]] uint8_t index() const {
        return this->frame[this->dgram_offset + 1];
    }

    /// @brief Returns the address field (interpretation depends on command).
    [[nodiscard]] uint32_t address() const {
        return static_cast<uint32_t>(this->frame[this->dgram_offset + 2]) |
               (static_cast<uint32_t>(this->frame[this->dgram_offset + 3]) << 8) |
               (static_cast<uint32_t>(this->frame[this->dgram_offset + 4]) << 16) |
               (static_cast<uint32_t>(this->frame[this->dgram_offset + 5]) << 24);
    }

    /// @brief Returns the slave position from address (for APRD/APWR).
    [[nodiscard]] int16_t adp() const {
        return static_cast<int16_t>(
            this->frame[this->dgram_offset + 2] |
            (this->frame[this->dgram_offset + 3] << 8)
        );
    }

    /// @brief Returns the physical memory offset from address (for APRD/APWR/FPRD/FPWR).
    [[nodiscard]] uint16_t ado() const {
        return static_cast<uint16_t>(
            this->frame[this->dgram_offset + 4] |
            (this->frame[this->dgram_offset + 5] << 8)
        );
    }

    /// @brief Returns the configured station address (for FPRD/FPWR).
    [[nodiscard]] uint16_t configured_address() const {
        return static_cast<uint16_t>(
            this->frame[this->dgram_offset + 2] |
            (this->frame[this->dgram_offset + 3] << 8)
        );
    }

    /// @brief Returns the logical address (for LRD/LWR/LRW).
    [[nodiscard]] uint32_t logical_address() const { return this->address(); }

    /// @brief Returns the data length field.
    [[nodiscard]] uint16_t data_length() const {
        const uint16_t raw = static_cast<uint16_t>(
            this->frame[this->dgram_offset + 6] |
            (this->frame[this->dgram_offset + 7] << 8)
        );
        return raw & 0x07FF;
    }

    /// @brief Returns the R (reserved) field bit 11.
    [[nodiscard]] bool reserved_bit() const {
        return (this->frame[this->dgram_offset + 7] & 0x08) != 0;
    }

    /// @brief Returns the C (circulating) field bit 14.
    [[nodiscard]] bool circulating() const {
        return (this->frame[this->dgram_offset + 7] & 0x40) != 0;
    }

    /// @brief Returns the M (more) field indicating more datagrams follow.
    [[nodiscard]] bool more_follows() const {
        return (this->frame[this->dgram_offset + 7] & 0x80) != 0;
    }

    /// @brief Returns the IRQ field.
    [[nodiscard]] uint16_t irq() const {
        return static_cast<uint16_t>(
            this->frame[this->dgram_offset + 8] |
            (this->frame[this->dgram_offset + 9] << 8)
        );
    }

    /// @brief Returns a mutable view of the data payload.
    [[nodiscard]] std::span<uint8_t> data() {
        return std::span<uint8_t>(
            this->frame.data() + this->dgram_offset + DATAGRAM_HEADER_SIZE,
            this->data_length()
        );
    }

    /// @brief Returns a const view of the data payload.
    [[nodiscard]] std::span<const uint8_t> data() const {
        return std::span<const uint8_t>(
            this->frame.data() + this->dgram_offset + DATAGRAM_HEADER_SIZE,
            this->data_length()
        );
    }

    /// @brief Returns the working counter.
    [[nodiscard]] uint16_t working_counter() const {
        const size_t wkc_offset = this->dgram_offset + DATAGRAM_HEADER_SIZE +
                                  this->data_length();
        return static_cast<uint16_t>(
            this->frame[wkc_offset] | (this->frame[wkc_offset + 1] << 8)
        );
    }

    /// @brief Sets the working counter.
    void set_working_counter(uint16_t wkc) {
        const size_t wkc_offset = this->dgram_offset + DATAGRAM_HEADER_SIZE +
                                  this->data_length();
        this->frame[wkc_offset] = static_cast<uint8_t>(wkc & 0xFF);
        this->frame[wkc_offset + 1] = static_cast<uint8_t>(wkc >> 8);
    }

    /// @brief Increments the working counter.
    void increment_wkc() { this->set_working_counter(this->working_counter() + 1); }

    /// @brief Increments working counter by 3 (for read-write operations).
    void increment_wkc_rw() { this->set_working_counter(this->working_counter() + 3); }

    /// @brief Decrements ADP (auto-increment position) for position counting.
    void decrement_adp() {
        int16_t new_adp = this->adp() - 1;
        this->frame[this->dgram_offset + 2] = static_cast<uint8_t>(new_adp & 0xFF);
        this->frame[this->dgram_offset + 3] = static_cast<uint8_t>((new_adp >> 8) & 0xFF);
    }

    /// @brief Returns the total size of this datagram in bytes.
    [[nodiscard]] size_t total_size() const {
        return DATAGRAM_HEADER_SIZE + this->data_length() + 2;
    }

    /// @brief Returns the offset to the next datagram.
    [[nodiscard]] size_t next_offset() const {
        return this->dgram_offset + this->total_size();
    }

    /// @brief Returns true if this is a read command.
    [[nodiscard]] bool is_read() const {
        switch (this->command()) {
            case Command::APRD:
            case Command::FPRD:
            case Command::BRD:
            case Command::LRD:
                return true;
            default:
                return false;
        }
    }

    /// @brief Returns true if this is a write command.
    [[nodiscard]] bool is_write() const {
        switch (this->command()) {
            case Command::APWR:
            case Command::FPWR:
            case Command::BWR:
            case Command::LWR:
                return true;
            default:
                return false;
        }
    }

    /// @brief Returns true if this is a read-write command.
    [[nodiscard]] bool is_read_write() const {
        switch (this->command()) {
            case Command::APRW:
            case Command::FPRW:
            case Command::BRW:
            case Command::LRW:
                return true;
            default:
                return false;
        }
    }

    /// @brief Returns true if this is a broadcast command.
    [[nodiscard]] bool is_broadcast() const {
        switch (this->command()) {
            case Command::BRD:
            case Command::BWR:
            case Command::BRW:
                return true;
            default:
                return false;
        }
    }

    /// @brief Returns true if this is an auto-increment command.
    [[nodiscard]] bool is_auto_increment() const {
        switch (this->command()) {
            case Command::APRD:
            case Command::APWR:
            case Command::APRW:
                return true;
            default:
                return false;
        }
    }

    /// @brief Returns true if this is a configured address command.
    [[nodiscard]] bool is_configured_address() const {
        switch (this->command()) {
            case Command::FPRD:
            case Command::FPWR:
            case Command::FPRW:
            case Command::FRMW:
                return true;
            default:
                return false;
        }
    }

    /// @brief Returns true if this is a logical address command.
    [[nodiscard]] bool is_logical() const {
        switch (this->command()) {
            case Command::LRD:
            case Command::LWR:
            case Command::LRW:
                return true;
            default:
                return false;
        }
    }

private:
    std::span<uint8_t> frame;
    size_t dgram_offset;
};

/// @brief Iterates over datagrams in a frame.
class DatagramIterator {
public:
    explicit DatagramIterator(std::span<uint8_t> datagram_payload):
        payload(datagram_payload), current_offset(0) {}

    /// @brief Returns true if there are more datagrams to process.
    [[nodiscard]] bool has_next() const {
        return this->current_offset + MIN_DATAGRAM_SIZE <= this->payload.size();
    }

    /// @brief Returns the next datagram.
    Datagram next() {
        Datagram dgram(this->payload, this->current_offset);
        if (dgram.valid() && dgram.more_follows())
            this->current_offset = dgram.next_offset();
        else
            this->current_offset = this->payload.size();
        return dgram;
    }

private:
    std::span<uint8_t> payload;
    size_t current_offset;
};

}
