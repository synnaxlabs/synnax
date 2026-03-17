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

namespace ethercat::virtual_esc {

/// EtherCAT ethertype (0x88A4).
constexpr uint16_t ETHERCAT_ETHERTYPE = 0x88A4;
/// Ethernet header size in bytes.
constexpr size_t ETHERNET_HEADER_SIZE = 14;
/// EtherCAT header size in bytes.
constexpr size_t ETHERCAT_HEADER_SIZE = 2;
/// Minimum EtherCAT frame size (Ethernet header + EtherCAT header).
constexpr size_t MIN_FRAME_SIZE = ETHERNET_HEADER_SIZE + ETHERCAT_HEADER_SIZE;
/// Datagram header size in bytes.
constexpr size_t DATAGRAM_HEADER_SIZE = 10;
/// Minimum datagram size (header + 2-byte working counter).
constexpr size_t MIN_DATAGRAM_SIZE = DATAGRAM_HEADER_SIZE + 2;

/// @brief Represents an Ethernet frame header.
struct EthernetHeader {
    uint8_t dest_mac[6];
    uint8_t src_mac[6];
    uint16_t ethertype;

    /// @brief Parses an Ethernet header from raw bytes.
    static EthernetHeader parse(std::span<const uint8_t> data) {
        EthernetHeader hdr{};
        if (data.size() < ETHERNET_HEADER_SIZE) return hdr;
        std::memcpy(hdr.dest_mac, data.data(), 6);
        std::memcpy(hdr.src_mac, data.data() + 6, 6);
        hdr.ethertype = static_cast<uint16_t>(data[12] << 8 | data[13]);
        return hdr;
    }

    /// @brief Writes the header to a buffer.
    void write(std::span<uint8_t> data) const {
        if (data.size() < ETHERNET_HEADER_SIZE) return;
        std::memcpy(data.data(), this->dest_mac, 6);
        std::memcpy(data.data() + 6, this->src_mac, 6);
        data[12] = static_cast<uint8_t>(this->ethertype >> 8);
        data[13] = static_cast<uint8_t>(this->ethertype & 0xFF);
    }

    /// @brief Checks if this is an EtherCAT frame.
    [[nodiscard]] bool is_ethercat() const {
        return this->ethertype == ETHERCAT_ETHERTYPE;
    }
};

/// @brief Represents an EtherCAT frame header.
struct EthercatHeader {
    /// Length of all datagrams in bytes (bits 0-10).
    uint16_t length;
    /// Reserved (bit 11).
    uint8_t reserved;
    /// Protocol type: 0x1 = EtherCAT commands (bits 12-15).
    uint8_t type;

    /// @brief Parses an EtherCAT header from raw bytes.
    static EthercatHeader parse(std::span<const uint8_t> data) {
        EthercatHeader hdr{};
        if (data.size() < ETHERCAT_HEADER_SIZE) return hdr;
        const uint16_t raw = static_cast<uint16_t>(data[0] | (data[1] << 8));
        hdr.length = raw & 0x07FF;
        hdr.reserved = (raw >> 11) & 0x01;
        hdr.type = (raw >> 12) & 0x0F;
        return hdr;
    }

    /// @brief Writes the header to a buffer.
    void write(std::span<uint8_t> data) const {
        if (data.size() < ETHERCAT_HEADER_SIZE) return;
        const uint16_t raw = (this->length & 0x07FF) | ((this->reserved & 0x01) << 11) |
                             ((this->type & 0x0F) << 12);
        data[0] = static_cast<uint8_t>(raw & 0xFF);
        data[1] = static_cast<uint8_t>(raw >> 8);
    }

    /// @brief Checks if this is an EtherCAT command frame.
    [[nodiscard]] bool is_command_frame() const { return this->type == 0x01; }
};

/// @brief Represents a complete EtherCAT frame for parsing and response generation.
class Frame {
public:
    EthernetHeader eth_header;
    EthercatHeader ec_header;
    std::vector<uint8_t> raw_data;

    Frame() = default;

    /// @brief Parses a frame from raw bytes.
    /// @return true if parsing succeeded and this is a valid EtherCAT frame.
    bool parse(std::span<const uint8_t> data) {
        if (data.size() < MIN_FRAME_SIZE) return false;
        this->eth_header = EthernetHeader::parse(data);
        if (!this->eth_header.is_ethercat()) return false;
        this->ec_header = EthercatHeader::parse(data.subspan(ETHERNET_HEADER_SIZE));
        if (!this->ec_header.is_command_frame()) return false;
        this->raw_data.assign(data.begin(), data.end());
        return true;
    }

    /// @brief Returns the datagram payload (after Ethernet and EtherCAT headers).
    [[nodiscard]] std::span<uint8_t> datagrams() {
        if (this->raw_data.size() <= MIN_FRAME_SIZE) return {};
        return std::span<uint8_t>(
            this->raw_data.data() + MIN_FRAME_SIZE,
            this->ec_header.length
        );
    }

    /// @brief Returns a const view of the datagram payload.
    [[nodiscard]] std::span<const uint8_t> datagrams() const {
        if (this->raw_data.size() <= MIN_FRAME_SIZE) return {};
        return std::span<const uint8_t>(
            this->raw_data.data() + MIN_FRAME_SIZE,
            this->ec_header.length
        );
    }

    /// @brief Swaps source and destination MAC addresses for the response.
    void swap_mac_addresses() {
        uint8_t tmp[6];
        std::memcpy(tmp, this->eth_header.dest_mac, 6);
        std::memcpy(this->eth_header.dest_mac, this->eth_header.src_mac, 6);
        std::memcpy(this->eth_header.src_mac, tmp, 6);
        this->eth_header.write(
            std::span<uint8_t>(this->raw_data.data(), ETHERNET_HEADER_SIZE)
        );
    }

    /// @brief Returns the raw frame data for transmission.
    [[nodiscard]] std::span<uint8_t> data() { return this->raw_data; }

    /// @brief Returns the frame size.
    [[nodiscard]] size_t size() const { return this->raw_data.size(); }
};

}
