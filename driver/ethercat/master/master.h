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
#include <functional>
#include <map>
#include <memory>
#include <optional>
#include <span>
#include <string>
#include <string_view>
#include <unordered_map>
#include <utility>
#include <vector>

#include "x/cpp/breaker/breaker.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "driver/ethercat/dc/dc.h"
#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/pdo/pdo.h"
#include "driver/ethercat/slave/slave.h"

namespace driver::ethercat::master {
/// @brief the distributed clock epoch, 2000-01-01, as a Unix timestamp.
inline const x::telem::TimeStamp DC_EPOCH(x::telem::SECOND * 946684800);

/// @brief information about an available EtherCAT master or network.
struct Info {
    /// @brief unique identifier for this master (e.g., "igh:0" or "eth0").
    std::string key;
    /// @brief human-readable description.
    std::string description;
};

/// @brief what activate() starts the bus with.
struct Plan {
    /// @brief the period of one bus cycle.
    x::telem::TimeSpan period;
    /// @brief the settings of each slave, keyed by position. A slave missing from it
    /// uses the defaults of slave::Config.
    std::map<uint16_t, slave::Config> slaves;
};

/// @brief the process data of a bus that activate() starts. The spans stay valid until
/// deactivate().
struct Image {
    /// @brief the inputs (TxPDO, slave to master).
    std::span<const uint8_t> inputs;
    /// @brief the outputs (RxPDO, master to slave).
    std::span<uint8_t> outputs;
    /// @brief the offset of each mapped entry.
    pdo::Offsets offsets;
    /// @brief why the master did not map an entry, keyed by entry.
    std::unordered_map<pdo::Key, x::errors::Error, pdo::KeyHash> errors;
    /// @brief the DC registers of each enabled slave whose SYNC0 events the master
    /// starts.
    std::vector<dc::Registers> dc_sync;
    /// @brief the slaves with the layouts that their startup parameters set.
    std::vector<slave::DiscoveryResult> slaves;
    /// @brief the least that each enabled slave adds to the working counter of a cycle,
    /// keyed by position. A slave with no process data adds nothing.
    std::map<uint16_t, uint32_t> working_counter_shares;

    /// @brief returns the offset of entry.
    /// @returns errors::PDO_MAPPING_ERROR when the master did not map the entry, or
    /// mapped it with another bit length.
    [[nodiscard]] std::pair<pdo::Offset, x::errors::Error>
    offset(const pdo::Entry &entry) const {
        const pdo::Key key{
            entry.slave_position,
            entry.index,
            entry.sub_index,
            entry.is_input
        };
        if (const auto it = this->errors.find(key); it != this->errors.end())
            return {{}, it->second};
        return pdo::find_offset(this->offsets, entry);
    }
};

/// @brief the outcome of activate().
struct Activation {
    /// @brief NIL when every enabled slave reached OP.
    x::errors::Error err;
    /// @brief why each slave refused a state, or why the master refused to start it,
    /// keyed by position.
    std::map<uint16_t, slave::Refusal> refusals;
};

/// @brief abstract interface for an EtherCAT master.
class Master {
public:
    virtual ~Master() = default;

    /// @brief returns the name that messages to the user give the network, such as
    /// "Ethernet 2" for a Windows adapter.
    [[nodiscard]] virtual std::string name() const = 0;

    /// @brief opens the master and discovers the slaves on the bus. Returns the slaves
    /// it found without a new discovery until deactivate().
    [[nodiscard]]
    virtual std::pair<std::vector<slave::DiscoveryResult>, x::errors::Error> scan() = 0;

    /// @brief counts the slaves that answer on the bus now, without changing their
    /// state. Valid between scan() and deactivate().
    [[nodiscard]] virtual std::pair<size_t, x::errors::Error> count_slaves() = 0;

    /// @brief maps the process data of every enabled slave, calls start_cycle with the
    /// image once send() and receive() work, starts the DC sync of each slave in a DC
    /// mode, then moves the enabled slaves to OP while the caller cycles the bus. After
    /// an error, the caller stops the cycle it started before it calls deactivate().
    /// Valid after scan(). Only the SOEM master starts DC sync.
    /// @returns errors::SLAVE_CONFIG_ERROR when the DC mode of a slave gives values its
    /// registers cannot hold at the period, or errors::STATE_CHANGE_REFUSED when a
    /// slave refuses SAFE-OP or OP. The SOEM master returns errors::ACTIVATION_ERROR
    /// with a refusal for a used slave with no identity, after start_cycle only when
    /// another slave has process data.
    [[nodiscard]] virtual Activation activate(
        const Plan &plan,
        const std::function<void(const Image &)> &start_cycle
    ) = 0;

    /// @brief stops cyclic communication and closes the master.
    virtual void deactivate() = 0;

    /// @brief receives and processes input data from the EtherCAT network.
    [[nodiscard]] virtual x::errors::Error receive() = 0;

    /// @brief queues output data and sends to the EtherCAT network.
    [[nodiscard]] virtual x::errors::Error send() = 0;

    /// @brief returns how far the working counter of the latest receive() fell short of
    /// a full cycle, or nullopt when the master cannot tell. Valid after a receive()
    /// that returns errors::WORKING_COUNTER_ERROR. A lost frame can take more than the
    /// shares of the slaves that missed the cycle.
    [[nodiscard]] virtual std::optional<uint32_t> shortfall() const = 0;

    /// @brief returns the distributed clock system time, as a Unix timestamp, at which
    /// the latest received frame passed the reference clock. Returns nullopt when no
    /// enabled slave has a distributed clock or the master does not read it. The DC
    /// clock is not aligned to the host clock. Valid after a receive() that returns
    /// NIL, or errors::WORKING_COUNTER_ERROR while the slaves move to OP. After that, a
    /// short frame can miss the reference clock and leave the previous time.
    [[nodiscard]] virtual std::optional<x::telem::TimeStamp> dc_time() const = 0;

    /// @brief reads the condition of each enabled slave and asks each slave that left
    /// OP to return to it. Takes no further recovery step once breaker stops. Safe to
    /// call from one thread while another runs receive() and send(). Valid between
    /// activate() and deactivate().
    /// @returns the condition of each enabled slave before any recovery step.
    [[nodiscard]] virtual std::vector<slave::Health>
    check(const x::breaker::Breaker &breaker) = 0;

    /// @brief writes parameters to the slave at position, in order. On a running bus, a
    /// master that can also adds them to the startup parameters of the slave, so it
    /// writes them again when the slave restarts. Safe to call from one thread while
    /// another runs receive() and send(). Valid between scan() and deactivate().
    /// @returns errors::STARTUP_PARAMETER_ERROR when the slave rejects a parameter. The
    /// parameters before it stay on the slave. errors::SETTING_CONFLICT when the master
    /// cannot change a parameter on the running bus.
    [[nodiscard]] virtual x::errors::Error write(
        uint16_t position,
        const std::vector<slave::StartupParameter> &parameters
    ) = 0;

    /// @brief reads an object of at most four bytes from the slave at position, as an
    /// unsigned value. Safe to call from one thread while another runs receive() and
    /// send(). Valid between scan() and deactivate().
    /// @returns coe::read_error() when the read fails.
    [[nodiscard]] virtual std::pair<uint32_t, x::errors::Error>
    read(uint16_t position, uint16_t index, uint8_t sub_index) = 0;
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
