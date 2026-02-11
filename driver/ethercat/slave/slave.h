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
#include <optional>
#include <string>
#include <vector>

#include "x/cpp/json/json.h"

#include "driver/ethercat/pdo/pdo.h"

namespace driver::ethercat::slave {

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

/// @brief static properties of an EtherCAT slave device stored in device.properties.
struct Properties {
    /// @brief network interface the slave is connected to.
    std::string network;
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
    /// @brief total input size in bits.
    uint32_t input_bits;
    /// @brief total output size in bits.
    uint32_t output_bits;
    /// @brief discovered input PDOs (TxPDO, slave→master).
    std::vector<pdo::Properties> input_pdos;
    /// @brief discovered output PDOs (RxPDO, master→slave).
    std::vector<pdo::Properties> output_pdos;
    /// @brief true if PDOs were discovered via CoE assignment objects.
    bool coe_pdo_order_reliable;
    /// @brief whether the device is enabled or not.
    bool enabled = true;

    /// @brief returns the total number of discovered PDO entries.
    [[nodiscard]] size_t pdo_count() const {
        return this->input_pdos.size() + this->output_pdos.size();
    }

    /// @brief finds an input PDO by name.
    [[nodiscard]] std::optional<pdo::Properties>
    find_input_pdo(const std::string &pdo_name) const {
        for (const auto &pdo: input_pdos)
            if (pdo.name == pdo_name) return pdo;
        return std::nullopt;
    }

    /// @brief finds an output PDO by name.
    [[nodiscard]] std::optional<pdo::Properties>
    find_output_pdo(const std::string &pdo_name) const {
        for (const auto &pdo: output_pdos)
            if (pdo.name == pdo_name) return pdo;
        return std::nullopt;
    }

    /// @brief parses slave properties from JSON.
    static Properties parse(x::json::Parser &parser) {
        Properties props;
        props.serial = parser.field<uint32_t>("serial");
        props.vendor_id = parser.field<uint32_t>("vendor_id");
        props.product_code = parser.field<uint32_t>("product_code");
        props.revision = parser.field<uint32_t>("revision");
        props.name = parser.field<std::string>("name");
        props.network = parser.field<std::string>("network", "");
        props.position = static_cast<uint16_t>(parser.field<int>("position"));
        props.enabled = parser.field<bool>("enabled");
        const auto pdos_parser = parser.child("pdos");
        if (!pdos_parser.error()) {
            pdos_parser.iter("inputs", [&props](x::json::Parser &pdo) {
                props.input_pdos.push_back(pdo::Properties::parse(pdo, true));
            });
            pdos_parser.iter("outputs", [&props](x::json::Parser &pdo) {
                props.output_pdos.push_back(pdo::Properties::parse(pdo, false));
            });
        }
        return props;
    }

    /// @brief serializes this slave's properties to JSON.
    [[nodiscard]] nlohmann::json to_json() const {
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
        props["enabled"] = this->enabled;

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

/// @brief dynamic status information about an EtherCAT slave from discovery.
struct Status {
    /// @brief current application layer state of the slave.
    State state;
    /// @brief true if PDO discovery completed successfully.
    bool pdos_discovered;
    /// @brief error message if PDO discovery failed (empty on success).
    std::string pdo_discovery_error;
};

/// @brief combined result from slave discovery containing both static properties
/// and dynamic status.
struct DiscoveryResult {
    Properties properties;
    Status status;
};

/// @brief extracts properties from a vector of discovery results.
inline std::vector<Properties>
discovered_properties(const std::vector<DiscoveryResult> &results) {
    std::vector<Properties> props;
    props.reserve(results.size());
    for (const auto &r: results)
        props.push_back(r.properties);
    return props;
}

}
