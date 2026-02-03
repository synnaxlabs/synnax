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

#include "x/cpp/xjson/xjson.h"

#include "driver/ethercat/pdo/pdo.h"

namespace ethercat::slave {

/// @brief EtherCAT slave application layer states as defined in ETG.1000.
enum class State : uint8_t {
    UNKNOWN = 0,
    INIT = 1,
    PRE_OP = 2,
    SAFE_OP = 4,
    OP = 8,
    BOOT = 3
};

/// @brief converts a slave::State enum value to its string representation.
inline std::string slave_state_to_string(const State state) {
    switch (state) {
        case State::INIT:
            return "INIT";
        case State::PRE_OP:
            return "PRE-OP";
        case State::SAFE_OP:
            return "SAFE-OP";
        case State::OP:
            return "OP";
        case State::BOOT:
            return "BOOT";
        default:
            return "UNKNOWN";
    }
}

/// @brief information about an EtherCAT slave device discovered on the network.
struct Properties {
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
    State state;
    /// @brief total input size in bits.
    uint32_t input_bits;
    /// @brief total output size in bits.
    uint32_t output_bits;
    /// @brief discovered input PDOs (TxPDO, slave→master).
    std::vector<pdo::Properties> input_pdos;
    /// @brief discovered output PDOs (RxPDO, master→slave).
    std::vector<pdo::Properties> output_pdos;
    /// @brief true if PDO discovery completed.
    bool pdos_discovered;
    /// @brief true if PDOs were discovered via CoE assignment objects.
    bool coe_pdo_order_reliable;
    /// @brief error message if PDO discovery failed (empty on success).
    std::string pdo_discovery_error;

    Properties():
        position(0),
        vendor_id(0),
        product_code(0),
        revision(0),
        serial(0),
        state(State::UNKNOWN),
        input_bits(0),
        output_bits(0),
        pdos_discovered(false),
        coe_pdo_order_reliable(false) {}

    Properties(
        const uint16_t position,
        const uint32_t vendor_id,
        const uint32_t product_code,
        const uint32_t revision,
        const uint32_t serial,
        std::string name,
        const State state,
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
        props["pdo_order_reliable"] = this->coe_pdo_order_reliable;
        props["enabled"] = true;

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

}
