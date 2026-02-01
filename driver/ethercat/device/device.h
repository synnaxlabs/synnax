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

#include "x/cpp/telem/telem.h"
#include "x/cpp/xjson/xjson.h"

namespace ethercat::device {
/// Information about a single PDO entry from device discovery.
struct PDOInfo {
    /// Human-readable name of the PDO.
    std::string name;
    /// CoE object dictionary index (e.g., 0x6000).
    uint16_t index;
    /// CoE object dictionary subindex.
    uint8_t subindex;
    /// Size of the data in bits.
    uint8_t bit_length;
    /// Data type string (e.g., "uint16", "float32").
    std::string data_type;

    explicit PDOInfo(xjson::Parser &parser):
        name(parser.field<std::string>("name")),
        index(static_cast<uint16_t>(parser.field<int>("index"))),
        subindex(static_cast<uint8_t>(parser.field<int>("subindex"))),
        bit_length(static_cast<uint8_t>(parser.field<int>("bit_length"))),
        data_type(parser.field<std::string>("data_type")) {}
};

/// Properties of an EtherCAT slave device parsed from Synnax device properties.
struct SlaveProperties {
    /// Unique serial number from device EEPROM.
    uint32_t serial;
    /// EtherCAT vendor ID.
    uint32_t vendor_id;
    /// Product code identifying device model.
    uint32_t product_code;
    /// Hardware/firmware revision.
    uint32_t revision;
    /// Human-readable device name.
    std::string name;
    /// Network interface name this slave is connected to.
    std::string network;
    /// Current position on bus (may change).
    uint16_t position;
    /// Input PDOs (TxPDO, slave->master).
    std::vector<PDOInfo> input_pdos;
    /// Output PDOs (RxPDO, master->slave).
    std::vector<PDOInfo> output_pdos;

    explicit SlaveProperties(xjson::Parser &parser):
        serial(parser.field<uint32_t>("serial")),
        vendor_id(parser.field<uint32_t>("vendor_id")),
        product_code(parser.field<uint32_t>("product_code")),
        revision(parser.field<uint32_t>("revision")),
        name(parser.field<std::string>("name")),
        network(parser.field<std::string>("network", "")),
        position(static_cast<uint16_t>(parser.field<int>("position"))) {
        auto pdos_parser = parser.child("pdos");
        if (!pdos_parser.error()) {
            pdos_parser.iter("inputs", [this](xjson::Parser &pdo) {
                this->input_pdos.emplace_back(pdo);
            });
            pdos_parser.iter("outputs", [this](xjson::Parser &pdo) {
                this->output_pdos.emplace_back(pdo);
            });
        }
    }

    /// Finds an input PDO by name.
    /// @param pdo_name The name of the PDO to find.
    /// @returns The PDO info if found, nullopt otherwise.
    [[nodiscard]] std::optional<PDOInfo>
    find_input_pdo(const std::string &pdo_name) const {
        for (const auto &pdo: input_pdos)
            if (pdo.name == pdo_name) return pdo;
        return std::nullopt;
    }

    /// Finds an output PDO by name.
    /// @param pdo_name The name of the PDO to find.
    /// @returns The PDO info if found, nullopt otherwise.
    [[nodiscard]] std::optional<PDOInfo>
    find_output_pdo(const std::string &pdo_name) const {
        for (const auto &pdo: output_pdos)
            if (pdo.name == pdo_name) return pdo;
        return std::nullopt;
    }
};

}
