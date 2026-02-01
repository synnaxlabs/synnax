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
#include <string>
#include <vector>

#include "x/cpp/telem/telem.h"

namespace ethercat {
/// EtherCAT slave application layer states as defined in ETG.1000.
enum class SlaveState : uint8_t {
    /// Slave is not responding or in an unknown state.
    UNKNOWN = 0,
    /// Initialization state - slave is being configured.
    INIT = 1,
    /// Pre-operational state - CoE/FoE communication available, no PDO exchange.
    PRE_OP = 2,
    /// Safe-operational state - inputs are valid, outputs are in safe state.
    SAFE_OP = 4,
    /// Operational state - full PDO exchange, normal operation.
    OP = 8,
    /// Bootstrap state - firmware update mode (optional).
    BOOT = 3
};

/// Converts a SlaveState enum value to its human-readable string representation.
/// @param state The slave state to convert.
/// @returns A string describing the state (e.g., "OPERATIONAL", "PRE-OPERATIONAL").
inline std::string slave_state_to_string(const SlaveState state) {
    switch (state) {
        case SlaveState::INIT:
            return "INIT";
        case SlaveState::PRE_OP:
            return "PRE-OP";
        case SlaveState::SAFE_OP:
            return "SAFE-OP";
        case SlaveState::OP:
            return "OP";
        case SlaveState::BOOT:
            return "BOOT";
        default:
            return "UNKNOWN";
    }
}

/// Information about a single PDO entry discovered during slave enumeration.
/// This represents one data point that can be exchanged cyclically.
struct PDOEntryInfo {
    /// Parent PDO index (e.g., 0x1A00 for TxPDO, 0x1600 for RxPDO).
    uint16_t pdo_index;
    /// Object dictionary index of this entry.
    uint16_t index;
    /// Object dictionary subindex of this entry.
    uint8_t subindex;
    /// Size of the data in bits.
    uint8_t bit_length;
    /// True for input (TxPDO, slave→master), false for output (RxPDO, master→slave).
    bool is_input;
    /// Human-readable name from CoE object dictionary, or generated fallback.
    std::string name;
    /// Synnax data type for seamless channel creation.
    telem::DataType data_type;

    PDOEntryInfo():
        pdo_index(0),
        index(0),
        subindex(0),
        bit_length(0),
        is_input(true),
        data_type(telem::UINT8_T) {}

    PDOEntryInfo(
        const uint16_t pdo_index,
        const uint16_t index,
        const uint8_t subindex,
        const uint8_t bit_length,
        const bool is_input,
        std::string name,
        const telem::DataType &data_type
    ):
        pdo_index(pdo_index),
        index(index),
        subindex(subindex),
        bit_length(bit_length),
        is_input(is_input),
        name(std::move(name)),
        data_type(data_type) {}

    /// Returns the size of this PDO entry in bytes (rounded up from bits).
    [[nodiscard]] size_t byte_length() const { return (bit_length + 7) / 8; }
};

/// Information about an EtherCAT slave device discovered on the network.
struct SlaveInfo {
    /// Position of the slave on the EtherCAT bus (0-based index).
    uint16_t position;
    /// EtherCAT vendor ID assigned by ETG.
    uint32_t vendor_id;
    /// Product code identifying the slave type.
    uint32_t product_code;
    /// Revision number for hardware/firmware versioning.
    uint32_t revision;
    /// Serial number of the device (if available).
    uint32_t serial;
    /// Human-readable name of the slave device.
    std::string name;
    /// Current application layer state of the slave.
    SlaveState state;
    /// Total input size in bits (from SOEM Ibits).
    uint32_t input_bits;
    /// Total output size in bits (from SOEM Obits).
    uint32_t output_bits;
    /// Discovered input PDOs (TxPDO, slave→master).
    std::vector<PDOEntryInfo> input_pdos;
    /// Discovered output PDOs (RxPDO, master→slave).
    std::vector<PDOEntryInfo> output_pdos;
    /// True if PDO discovery completed (even if partially).
    bool pdos_discovered;
    /// True if PDOs were discovered via CoE assignment objects, ensuring correct order.
    bool coe_pdo_order_reliable;
    /// Error message if PDO discovery failed (empty on success).
    std::string pdo_discovery_error;

    SlaveInfo():
        position(0),
        vendor_id(0),
        product_code(0),
        revision(0),
        serial(0),
        state(SlaveState::UNKNOWN),
        input_bits(0),
        output_bits(0),
        pdos_discovered(false),
        coe_pdo_order_reliable(false) {}

    SlaveInfo(
        const uint16_t position,
        const uint32_t vendor_id,
        const uint32_t product_code,
        const uint32_t revision,
        const uint32_t serial,
        std::string name,
        const SlaveState state,
        const uint32_t input_bits = 0,
        const uint32_t output_bits = 0
    ):
        position(position),
        vendor_id(vendor_id),
        product_code(product_code),
        revision(revision),
        serial(serial),
        name(std::move(name)),
        state(state),
        input_bits(input_bits),
        output_bits(output_bits),
        pdos_discovered(false),
        coe_pdo_order_reliable(false) {}

    /// Returns the total number of discovered PDO entries.
    [[nodiscard]] size_t pdo_count() const {
        return input_pdos.size() + output_pdos.size();
    }
};

/// Describes a single PDO entry (object) to be exchanged cyclically.
struct PDOEntry {
    /// Position of the slave on the EtherCAT bus.
    uint16_t slave_position = 0;
    /// Index of the PDO object in the CoE object dictionary (e.g., 0x6000).
    uint16_t index = 0;
    /// Subindex of the PDO object.
    uint8_t subindex = 0;
    /// Size of the data in bits.
    uint8_t bit_length = 0;
    /// True for input (TxPDO, slave→master), false for output (RxPDO, master→slave).
    bool is_input = true;
    /// Actual hardware data type from the PDO (e.g., INT16, UINT24).
    telem::DataType data_type = telem::UNKNOWN_T;

    /// Returns the size of this PDO entry in bytes (rounded up from bits).
    [[nodiscard]] size_t byte_length() const { return (bit_length + 7) / 8; }
};
}
