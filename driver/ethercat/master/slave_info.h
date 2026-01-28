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

    SlaveInfo():
        position(0),
        vendor_id(0),
        product_code(0),
        revision(0),
        serial(0),
        state(SlaveState::UNKNOWN) {}

    SlaveInfo(
        const uint16_t position,
        const uint32_t vendor_id,
        const uint32_t product_code,
        const uint32_t revision,
        const uint32_t serial,
        std::string name,
        const SlaveState state
    ):
        position(position),
        vendor_id(vendor_id),
        product_code(product_code),
        revision(revision),
        serial(serial),
        name(std::move(name)),
        state(state) {}
};

/// Data offset information for a slave's process data in the IOmap.
/// Used to calculate actual byte offsets after master activation.
struct SlaveDataOffsets {
    /// Byte offset in the IOmap where this slave's input data starts.
    size_t input_offset;

    /// Size of this slave's input data in bytes.
    size_t input_size;

    /// Byte offset in the IOmap where this slave's output data starts.
    size_t output_offset;

    /// Size of this slave's output data in bytes.
    size_t output_size;

    SlaveDataOffsets():
        input_offset(0), input_size(0), output_offset(0), output_size(0) {}

    SlaveDataOffsets(
        const size_t input_offset,
        const size_t input_size,
        const size_t output_offset,
        const size_t output_size
    ):
        input_offset(input_offset),
        input_size(input_size),
        output_offset(output_offset),
        output_size(output_size) {}
};

/// Describes a single PDO entry (object) to be exchanged cyclically.
struct PDOEntry {
    /// Position of the slave on the EtherCAT bus.
    uint16_t slave_position;

    /// Index of the PDO object in the CoE object dictionary (e.g., 0x6000).
    uint16_t index;

    /// Subindex of the PDO object.
    uint8_t subindex;

    /// Size of the data in bits.
    uint8_t bit_length;

    /// True for input (TxPDO, slave→master), false for output (RxPDO, master→slave).
    bool is_input;

    PDOEntry():
        slave_position(0),
        index(0),
        subindex(0),
        bit_length(0),
        is_input(true) {}

    PDOEntry(
        const uint16_t slave_position,
        const uint16_t index,
        const uint8_t subindex,
        const uint8_t bit_length,
        const bool is_input
    ):
        slave_position(slave_position),
        index(index),
        subindex(subindex),
        bit_length(bit_length),
        is_input(is_input) {}

    /// Returns the size of this PDO entry in bytes (rounded up from bits).
    [[nodiscard]] size_t byte_length() const { return (bit_length + 7) / 8; }
};
}
