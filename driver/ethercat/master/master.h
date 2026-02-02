// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <span>
#include <string>
#include <utility>
#include <vector>

#include "x/cpp/xerrors/errors.h"

#include "driver/ethercat/master/slave_info.h"

namespace ethercat::master {

/// @brief information about an available EtherCAT master or network.
struct Info {
    /// @brief unique identifier for this master (e.g., "igh:0" or "eth0").
    std::string key;
    /// @brief human-readable description.
    std::string description;
};

/// @brief byte and bit offset for a PDO entry in the process data buffer.
struct PDOOffset {
    /// @brief byte offset into the appropriate buffer (input_data or output_data).
    size_t byte = 0;
    /// @brief bit offset within the byte for sub-byte entries (0-7).
    uint8_t bit = 0;
};

/// @brief abstract interface for an EtherCAT master.
class Master {
public:
    virtual ~Master() = default;

    /// @brief initializes the master and scans the EtherCAT network for slaves.
    [[nodiscard]] virtual xerrors::Error initialize() = 0;

    /// @brief registers PDO entries for process data exchange.
    [[nodiscard]] virtual xerrors::Error
    register_pdos(const std::vector<PDOEntry> &entries) = 0;

    /// @brief activates the master and transitions slaves to OPERATIONAL state.
    [[nodiscard]] virtual xerrors::Error activate() = 0;

    /// @brief deactivates the master and stops cyclic communication.
    virtual void deactivate() = 0;

    /// @brief receives and processes input data from the EtherCAT network.
    [[nodiscard]] virtual xerrors::Error receive() = 0;

    /// @brief queues output data and sends to the EtherCAT network.
    [[nodiscard]] virtual xerrors::Error send() = 0;

    /// @brief returns the input data buffer (TxPDO, slave→master).
    [[nodiscard]] virtual std::span<const uint8_t> input_data() = 0;

    /// @brief returns the output data buffer (RxPDO, master→slave).
    [[nodiscard]] virtual std::span<uint8_t> output_data() = 0;

    /// @brief returns the byte and bit offset for a PDO entry.
    [[nodiscard]] virtual PDOOffset pdo_offset(const PDOEntry &entry) const = 0;

    /// @brief returns information about all slaves discovered during initialization.
    [[nodiscard]] virtual std::vector<SlaveInfo> slaves() const = 0;

    /// @brief returns the current state of a specific slave.
    [[nodiscard]] virtual SlaveState slave_state(uint16_t position) const = 0;

    /// @brief checks if all configured slaves are in OPERATIONAL state.
    [[nodiscard]] virtual bool all_slaves_operational() const = 0;

    /// @brief returns the name of the network interface this master is bound to.
    [[nodiscard]] virtual std::string interface_name() const = 0;
};

/// @brief abstract interface for discovering and creating EtherCAT masters.
class Manager {
public:
    virtual ~Manager() = default;

    /// @brief returns all available EtherCAT masters.
    [[nodiscard]] virtual std::vector<Info> enumerate() = 0;

    /// @brief creates a master for the given key.
    [[nodiscard]] virtual std::pair<std::shared_ptr<Master>, xerrors::Error>
    create(const std::string &key) = 0;
};

}
