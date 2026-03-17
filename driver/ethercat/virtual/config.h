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

#include "driver/ethercat/virtual/object_dictionary.h"

namespace ethercat::virtual_esc {

/// Test vendor ID.
constexpr uint32_t TEST_VENDOR_ID = 0xDEAD;
/// Test product code.
constexpr uint32_t TEST_PRODUCT_CODE = 0xBEEF;
/// Test revision number.
constexpr uint32_t TEST_REVISION = 0x0001;
/// Test serial number.
constexpr uint32_t TEST_SERIAL = 0x12345678;

/// @brief Configuration for a Virtual ESC instance.
struct Config {
    /// Vendor ID (ETG assigned).
    uint32_t vendor_id = TEST_VENDOR_ID;
    /// Product code identifying the slave type.
    uint32_t product_code = TEST_PRODUCT_CODE;
    /// Revision number.
    uint32_t revision = TEST_REVISION;
    /// Serial number.
    uint32_t serial = TEST_SERIAL;
    /// Device name.
    std::string name = "Virtual Test Slave";
    /// Hardware version string.
    std::string hw_version = "1.0";
    /// Software version string.
    std::string sw_version = "1.0";
    /// Configured station address (assigned by master, but can be preset).
    uint16_t station_address = 0x1001;
    /// TxPDO configurations (slave to master).
    std::vector<PDOConfig> tx_pdos;
    /// RxPDO configurations (master to slave).
    std::vector<PDOConfig> rx_pdos;
    /// Mailbox output start address (master to slave).
    uint16_t mbx_out_addr = 0x1000;
    /// Mailbox output size.
    uint16_t mbx_out_size = 128;
    /// Mailbox input start address (slave to master).
    uint16_t mbx_in_addr = 0x1800;
    /// Mailbox input size.
    uint16_t mbx_in_size = 128;
    /// Supported mailbox protocols (bit field).
    uint8_t mbx_protocols = MBX_PROTO_COE;

    /// @brief Returns the total TxPDO size in bytes.
    [[nodiscard]] size_t tx_pdo_bytes() const {
        size_t total = 0;
        for (const auto &pdo: this->tx_pdos)
            total += pdo.total_bytes();
        return total;
    }

    /// @brief Returns the total RxPDO size in bytes.
    [[nodiscard]] size_t rx_pdo_bytes() const {
        size_t total = 0;
        for (const auto &pdo: this->rx_pdos)
            total += pdo.total_bytes();
        return total;
    }
};

/// @brief Creates a default test configuration with standard PDO mappings.
///
/// TxPDO 0x1A00 (slave → master, 10 bytes):
///   0x6000:01 uint16_t status_word
///   0x6000:02 int32_t  actual_position
///   0x6000:03 int16_t  actual_velocity
///   0x6000:04 uint8_t  digital_inputs
///   0x6000:05 uint8_t  error_code
///
/// RxPDO 0x1600 (master → slave, 10 bytes):
///   0x7000:01 uint16_t control_word
///   0x7000:02 int32_t  target_position
///   0x7000:03 int16_t  target_velocity
///   0x7000:04 uint8_t  digital_outputs
///   0x7000:05 uint8_t  mode
inline Config default_test_config() {
    Config cfg;
    PDOConfig tx_pdo;
    tx_pdo.index = 0x1A00;
    tx_pdo.entries = {
        {0x6000, 0x01, 16},
        {0x6000, 0x02, 32},
        {0x6000, 0x03, 16},
        {0x6000, 0x04, 8},
        {0x6000, 0x05, 8},
    };
    cfg.tx_pdos.push_back(tx_pdo);
    PDOConfig rx_pdo;
    rx_pdo.index = 0x1600;
    rx_pdo.entries = {
        {0x7000, 0x01, 16},
        {0x7000, 0x02, 32},
        {0x7000, 0x03, 16},
        {0x7000, 0x04, 8},
        {0x7000, 0x05, 8},
    };
    cfg.rx_pdos.push_back(rx_pdo);
    return cfg;
}

/// @brief Creates a minimal test configuration with a single byte input/output.
inline Config minimal_test_config() {
    Config cfg;
    PDOConfig tx_pdo;
    tx_pdo.index = 0x1A00;
    tx_pdo.entries = {{0x6000, 0x01, 8}};
    cfg.tx_pdos.push_back(tx_pdo);
    PDOConfig rx_pdo;
    rx_pdo.index = 0x1600;
    rx_pdo.entries = {{0x7000, 0x01, 8}};
    cfg.rx_pdos.push_back(rx_pdo);
    return cfg;
}

}
