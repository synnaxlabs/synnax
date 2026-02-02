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

#include "nlohmann/json.hpp"

#include "x/cpp/telem/telem.h"

namespace ethercat {
/// @brief EtherCAT slave application layer states as defined in ETG.1000.
enum class SlaveState : uint8_t {
    UNKNOWN = 0,
    INIT = 1,
    PRE_OP = 2,
    SAFE_OP = 4,
    OP = 8,
    BOOT = 3
};

/// @brief converts a SlaveState enum value to its string representation.
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

/// @brief information about a single PDO entry discovered during slave enumeration.
struct PDOEntryInfo {
    /// @brief parent PDO index (e.g., 0x1A00 for TxPDO, 0x1600 for RxPDO).
    uint16_t pdo_index;
    /// @brief object dictionary index of this entry.
    uint16_t index;
    /// @brief object dictionary subindex of this entry.
    uint8_t subindex;
    /// @brief size of the data in bits.
    uint8_t bit_length;
    /// @brief true for input (TxPDO), false for output (RxPDO).
    bool is_input;
    /// @brief human-readable name from CoE object dictionary.
    std::string name;
    /// @brief Synnax data type for channel creation.
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

    /// @brief returns the size of this PDO entry in bytes (rounded up from bits).
    [[nodiscard]] size_t byte_length() const { return (this->bit_length + 7) / 8; }

    /// @brief serializes this PDO entry to JSON.
    [[nodiscard]] nlohmann::json to_json() const {
        return {
            {"name", this->name},
            {"pdo_index", this->pdo_index},
            {"index", this->index},
            {"subindex", this->subindex},
            {"bit_length", this->bit_length},
            {"data_type", this->data_type.name()}
        };
    }
};

/// @brief information about an EtherCAT slave device discovered on the network.
struct SlaveInfo {
    /// @brief position of the slave on the EtherCAT bus (0-based index).
    uint16_t position;
    /// @brief EtherCAT vendor ID assigned by ETG.
    uint32_t vendor_id;
    /// @brief product code identifying the slave type.
    uint32_t product_code;
    /// @brief revision number for hardware/firmware versioning.
    uint32_t revision;
    /// @brief serial number of the device (if available).
    uint32_t serial;
    /// @brief human-readable name of the slave device.
    std::string name;
    /// @brief current application layer state of the slave.
    SlaveState state;
    /// @brief total input size in bits.
    uint32_t input_bits;
    /// @brief total output size in bits.
    uint32_t output_bits;
    /// @brief discovered input PDOs (TxPDO, slave→master).
    std::vector<PDOEntryInfo> input_pdos;
    /// @brief discovered output PDOs (RxPDO, master→slave).
    std::vector<PDOEntryInfo> output_pdos;
    /// @brief true if PDO discovery completed.
    bool pdos_discovered;
    /// @brief true if PDOs were discovered via CoE assignment objects.
    bool coe_pdo_order_reliable;
    /// @brief error message if PDO discovery failed (empty on success).
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

    /// @brief returns the total number of discovered PDO entries.
    [[nodiscard]] size_t pdo_count() const {
        return this->input_pdos.size() + this->output_pdos.size();
    }

    /// @brief serializes this slave's properties to JSON.
    [[nodiscard]] nlohmann::json
    to_device_properties(const std::string &network) const {
        nlohmann::json props;
        props["vendor_id"] = this->vendor_id;
        props["product_code"] = this->product_code;
        props["revision"] = this->revision;
        props["serial"] = this->serial;
        props["name"] = this->name;
        props["network"] = network;
        props["position"] = this->position;
        props["input_bits"] = this->input_bits;
        props["output_bits"] = this->output_bits;

        nlohmann::json inputs = nlohmann::json::array();
        for (const auto &pdo: this->input_pdos)
            inputs.push_back(pdo.to_json());

        nlohmann::json outputs = nlohmann::json::array();
        for (const auto &pdo: this->output_pdos)
            outputs.push_back(pdo.to_json());

        props["pdos"] = {{"inputs", inputs}, {"outputs", outputs}};
        return props;
    }
};

/// @brief describes a single PDO entry (object) to be exchanged cyclically.
struct PDOEntry {
    /// @brief position of the slave on the EtherCAT bus.
    uint16_t slave_position = 0;
    /// @brief index of the PDO object in the CoE object dictionary.
    uint16_t index = 0;
    /// @brief subindex of the PDO object.
    uint8_t subindex = 0;
    /// @brief size of the data in bits.
    uint8_t bit_length = 0;
    /// @brief true for input (TxPDO), false for output (RxPDO).
    bool is_input = true;
    /// @brief actual hardware data type from the PDO.
    telem::DataType data_type = telem::UNKNOWN_T;

    /// @brief returns the size of this PDO entry in bytes (rounded up from bits).
    [[nodiscard]] size_t byte_length() const { return (bit_length + 7) / 8; }
};
}
