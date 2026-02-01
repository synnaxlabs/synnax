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

/// Information about an available EtherCAT master or network.
struct Info {
    /// Unique identifier for this master.
    /// For IgH: "igh:0", "igh:1", etc.
    /// For SOEM: the network interface name (e.g., "eth0", "enp3s0").
    std::string key;
    /// Human-readable description.
    std::string description;
};

/// Byte and bit offset for a PDO entry in the process data buffer.
struct PDOOffset {
    /// Byte offset into the appropriate buffer (input_data or output_data).
    size_t byte = 0;
    /// Bit offset within the byte for sub-byte entries (0-7).
    uint8_t bit = 0;
};

/// Abstract interface for an EtherCAT master.
///
/// The master manages the EtherCAT network and coordinates cyclic process data
/// exchange with slaves. The lifecycle follows EtherCAT state machine conventions:
///
/// 1. Construction: Create master for a network interface
/// 2. initialize(): Scan bus, enumerate slaves, configure network
/// 3. activate(): Transition slaves to OPERATIONAL, start cyclic operation
/// 4. Cyclic loop: receive() → [read inputs] → [write outputs] → send()
/// 5. deactivate(): Stop cyclic operation, transition slaves to INIT
///
/// Thread safety: The cyclic methods (receive/send) must be called from a single
/// thread. Slave queries (slaves(), slave_state()) are thread-safe.
class Master {
public:
    virtual ~Master() = default;

    /// Initializes the master and scans the EtherCAT network.
    ///
    /// This method opens the network interface, scans for slaves, and prepares
    /// the master for activation. After successful initialization, slaves()
    /// will return information about discovered slaves.
    ///
    /// @returns xerrors::NIL on success, or one of:
    ///          - INTERFACE_ERROR if the network interface cannot be opened
    ///          - MASTER_INIT_ERROR if master initialization fails
    ///          - SLAVE_CONFIG_ERROR if slave configuration fails
    [[nodiscard]] virtual xerrors::Error initialize() = 0;

    /// Registers PDO entries for process data exchange.
    ///
    /// This method must be called after initialize() and before activate().
    /// For IgH, this registers each entry with the domain via
    /// ecrt_slave_config_reg_pdo_entry(). For SOEM, this is a no-op since
    /// PDOs are automatically mapped during activation.
    ///
    /// @param entries The PDO entries to register for cyclic exchange.
    /// @returns xerrors::NIL on success, or:
    ///          - PDO_MAPPING_ERROR if registration fails
    [[nodiscard]] virtual xerrors::Error
    register_pdos(const std::vector<PDOEntry> &entries) = 0;

    /// Activates the master and transitions slaves to OPERATIONAL state.
    ///
    /// After activation, cyclic communication can begin. The master will attempt
    /// to transition all configured slaves through PRE-OP → SAFE-OP → OP.
    ///
    /// @returns xerrors::NIL on success, or one of:
    ///          - ACTIVATION_ERROR if master activation fails
    ///          - SLAVE_STATE_ERROR if slaves fail to reach OPERATIONAL
    [[nodiscard]] virtual xerrors::Error activate() = 0;

    /// Deactivates the master and stops cyclic communication.
    ///
    /// Transitions slaves back to INIT state and releases resources. After
    /// deactivation, the master can be re-initialized or destroyed.
    virtual void deactivate() = 0;

    /// Receives and processes input data from the EtherCAT network.
    ///
    /// This method receives datagrams from the network and processes them to
    /// update the input buffer. After this call, input PDO values accessible
    /// via input_data() are valid for the current cycle.
    ///
    /// @returns xerrors::NIL on success, or:
    ///          - CYCLIC_ERROR if receive fails
    ///          - WORKING_COUNTER_ERROR if the working counter is incorrect
    [[nodiscard]] virtual xerrors::Error receive() = 0;

    /// Queues output data and sends to the EtherCAT network.
    ///
    /// This method takes the current output buffer contents and transmits them
    /// to the slaves. Call this after writing output PDO values to output_data().
    ///
    /// @returns xerrors::NIL on success, or:
    ///          - CYCLIC_ERROR if send fails
    [[nodiscard]] virtual xerrors::Error send() = 0;

    /// Returns the input data buffer.
    ///
    /// The buffer contains input PDO data (TxPDO, slave→master) and is valid
    /// after receive() completes. Use pdo_offset() to find specific PDO locations.
    ///
    /// @returns Read-only span of input buffer, or empty span if not activated.
    [[nodiscard]] virtual std::span<const uint8_t> input_data() = 0;

    /// Returns the output data buffer.
    ///
    /// Write output PDO data (RxPDO, master→slave) to this buffer before calling
    /// send(). Use pdo_offset() to find specific PDO locations.
    ///
    /// @returns Mutable span of output buffer, or empty span if not activated.
    [[nodiscard]] virtual std::span<uint8_t> output_data() = 0;

    /// Returns the byte and bit offset for a PDO entry in the appropriate buffer.
    ///
    /// For input PDOs (is_input=true), returns offset into input_data().
    /// For output PDOs (is_input=false), returns offset into output_data().
    ///
    /// @param entry The PDO entry to look up.
    /// @returns PDOOffset with byte and bit offsets, or {0, 0} if entry not found.
    [[nodiscard]] virtual PDOOffset pdo_offset(const PDOEntry &entry) const = 0;

    /// Returns information about all slaves discovered during initialization.
    ///
    /// The returned vector is ordered by slave position on the bus. This method
    /// can be called after initialize() succeeds.
    ///
    /// @returns Vector of SlaveInfo structures describing each slave.
    [[nodiscard]] virtual std::vector<SlaveInfo> slaves() const = 0;

    /// Returns the current state of a specific slave.
    ///
    /// @param position The bus position of the slave (0-based for IgH, 1-based for
    /// SOEM).
    /// @returns The slave's current application layer state.
    [[nodiscard]] virtual SlaveState slave_state(uint16_t position) const = 0;

    /// Checks if all configured slaves are in OPERATIONAL state.
    ///
    /// @returns true if all slaves are operational, false otherwise.
    [[nodiscard]] virtual bool all_slaves_operational() const = 0;

    /// Returns the name of the network interface this master is bound to.
    [[nodiscard]] virtual std::string interface_name() const = 0;
};

/// Abstract interface for discovering and creating EtherCAT masters.
///
/// Different backends (IgH, SOEM) have different mechanisms for discovering
/// available masters and creating them. This interface abstracts both so the
/// Pool and scan task don't need backend-specific code.
class Manager {
public:
    virtual ~Manager() = default;

    /// Returns all available EtherCAT masters.
    ///
    /// For IgH, returns configured kernel masters from /sys/class/EtherCAT/.
    /// For SOEM, returns network interfaces that could have slaves.
    [[nodiscard]] virtual std::vector<Info> enumerate() = 0;

    /// Creates a master for the given key.
    ///
    /// @param key The master key from Info (e.g., "igh:0" or "eth0").
    /// @returns Pair of shared pointer to the created Master and error.
    [[nodiscard]] virtual std::pair<std::shared_ptr<Master>, xerrors::Error>
    create(const std::string &key) = 0;
};

}
