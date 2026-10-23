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
#include <optional>
#include <span>
#include <string>
#include <utility>
#include <vector>

#include "x/cpp/errors/errors.h"

#include "driver/ethercat/pdo/pdo.h"
#include "driver/ethercat/slave/slave.h"

namespace driver::ethercat::master {

/// @brief information about an available EtherCAT master or network.
struct Info {
    /// @brief unique identifier for this master (e.g., "igh:0" or "eth0").
    std::string key;
    /// @brief human-readable description.
    std::string description;
};

/// @brief abstract interface for an EtherCAT master.
class Master {
public:
    virtual ~Master() = default;

    /// @brief initializes the master and scans the EtherCAT network for slaves.
    [[nodiscard]] virtual x::errors::Error initialize() = 0;

    /// @brief sets whether a slave joins cyclic exchange. Takes effect at the next
    /// activate().
    virtual void set_slave_enabled(uint16_t position, bool enabled) = 0;

    /// @brief maps the process data of every enabled slave and transitions the enabled
    /// slaves to OP.
    [[nodiscard]] virtual x::errors::Error activate() = 0;

    /// @brief deactivates the master and stops cyclic communication.
    virtual void deactivate() = 0;

    /// @brief receives and processes input data from the EtherCAT network.
    [[nodiscard]] virtual x::errors::Error receive() = 0;

    /// @brief queues output data and sends to the EtherCAT network.
    [[nodiscard]] virtual x::errors::Error send() = 0;

    /// @brief returns the input data buffer (TxPDO, slave→master).
    [[nodiscard]] virtual std::span<const uint8_t> input_data() = 0;

    /// @brief returns the output data buffer (RxPDO, master→slave).
    [[nodiscard]] virtual std::span<uint8_t> output_data() = 0;

    /// @brief returns the byte and bit offset of a PDO entry in its process image, or
    /// nullopt when the entry is not mapped. Valid between activate() and
    /// deactivate().
    [[nodiscard]] virtual std::optional<pdo::Offset>
    pdo_offset(const pdo::Entry &entry) const = 0;

    /// @brief returns information about all slaves discovered during initialization.
    [[nodiscard]] virtual std::vector<slave::DiscoveryResult> slaves() const = 0;

    /// @brief reads the condition of each enabled slave and asks each slave that left
    /// OP to return to it. Safe to call from one thread while another runs receive()
    /// and send(). Valid between activate() and deactivate().
    /// @returns the condition of each enabled slave before any recovery step.
    [[nodiscard]] virtual std::vector<slave::Health> check() = 0;

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
    [[nodiscard]] virtual std::pair<std::shared_ptr<Master>, x::errors::Error>
    create(const std::string &key) = 0;
};

}
