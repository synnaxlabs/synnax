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
#include <string>
#include <vector>

#include "x/cpp/xerrors/errors.h"

#include "driver/ethercat/master/domain.h"
#include "driver/ethercat/master/slave_info.h"

namespace ethercat {
/// Abstract interface for an EtherCAT master.
///
/// The master manages the EtherCAT network and coordinates cyclic process data
/// exchange with slaves. The lifecycle follows EtherCAT state machine conventions:
///
/// 1. Construction: Create master for a network interface
/// 2. initialize(): Scan bus, enumerate slaves, configure network
/// 3. create_domain(): Create domain(s) for PDO exchange
/// 4. Register PDOs with domain(s)
/// 5. activate(): Transition slaves to OPERATIONAL, start cyclic operation
/// 6. Cyclic loop: receive() → process() → [read inputs, write outputs] → queue() → send()
/// 7. deactivate(): Stop cyclic operation, transition slaves to INIT
///
/// Thread safety: The cyclic methods (receive/process/queue/send) must be called
/// from a single thread. Slave queries (slaves(), slave_state()) are thread-safe.
class Master {
public:
    virtual ~Master() = default;

    /// Initializes the master and scans the EtherCAT network.
    ///
    /// This method opens the network interface, scans for slaves, and prepares
    /// the master for domain creation. After successful initialization, slaves()
    /// will return information about discovered slaves.
    ///
    /// @returns xerrors::NIL on success, or one of:
    ///          - INTERFACE_ERROR if the network interface cannot be opened
    ///          - MASTER_INIT_ERROR if master initialization fails
    ///          - SLAVE_CONFIG_ERROR if slave configuration fails
    [[nodiscard]] virtual xerrors::Error initialize() = 0;

    /// Creates a new process data domain for PDO exchange.
    ///
    /// Must be called after initialize() and before activate(). Multiple domains
    /// can be created for different update rates or isolation, though most
    /// applications use a single domain.
    ///
    /// @returns A unique pointer to the created domain, or nullptr if creation fails.
    [[nodiscard]] virtual std::unique_ptr<Domain> create_domain() = 0;

    /// Activates the master and transitions slaves to OPERATIONAL state.
    ///
    /// After activation, cyclic communication can begin. All PDO registrations
    /// must be complete before calling this method. The master will attempt to
    /// transition all configured slaves through PRE-OP → SAFE-OP → OP.
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

    /// Receives process data from the EtherCAT network.
    ///
    /// This is the first step in the cyclic exchange. It receives datagrams
    /// from the network interface. Must be followed by process() to update
    /// domain data.
    ///
    /// @returns xerrors::NIL on success, or:
    ///          - CYCLIC_ERROR if receive fails
    [[nodiscard]] virtual xerrors::Error receive() = 0;

    /// Processes received datagrams and updates domain input data.
    ///
    /// Called after receive() to decode the received process data and update
    /// the domain's input buffer. After this call, input PDO values are valid.
    ///
    /// @param domain The domain whose process data should be updated.
    /// @returns xerrors::NIL on success, or:
    ///          - WORKING_COUNTER_ERROR if the working counter is incorrect
    ///          - CYCLIC_ERROR if processing fails
    [[nodiscard]] virtual xerrors::Error process(Domain &domain) = 0;

    /// Queues output data from the domain for transmission.
    ///
    /// Called after updating output PDO values in the domain buffer. This
    /// prepares the datagrams for transmission.
    ///
    /// @param domain The domain whose output data should be queued.
    /// @returns xerrors::NIL on success, or:
    ///          - CYCLIC_ERROR if queuing fails
    [[nodiscard]] virtual xerrors::Error queue(Domain &domain) = 0;

    /// Sends queued process data to the EtherCAT network.
    ///
    /// This is the final step in the cyclic exchange. It transmits the
    /// prepared datagrams to the slaves.
    ///
    /// @returns xerrors::NIL on success, or:
    ///          - CYCLIC_ERROR if send fails
    [[nodiscard]] virtual xerrors::Error send() = 0;

    /// Returns information about all slaves discovered during initialization.
    ///
    /// The returned vector is ordered by slave position on the bus. This method
    /// can be called after initialize() succeeds.
    ///
    /// @returns Vector of SlaveInfo structures describing each slave.
    [[nodiscard]] virtual std::vector<SlaveInfo> slaves() const = 0;

    /// Returns the current state of a specific slave.
    ///
    /// @param position The bus position of the slave (0-based index).
    /// @returns The slave's current application layer state.
    [[nodiscard]] virtual SlaveState slave_state(uint16_t position) const = 0;

    /// Checks if all configured slaves are in OPERATIONAL state.
    ///
    /// @returns true if all slaves are operational, false otherwise.
    [[nodiscard]] virtual bool all_slaves_operational() const = 0;

    /// Returns the name of the network interface this master is bound to.
    [[nodiscard]] virtual std::string interface_name() const = 0;

    /// Returns the data offsets for a specific slave after activation.
    ///
    /// This method returns the actual byte offsets in the IOmap where the slave's
    /// input and output data is located. Must only be called after activate()
    /// succeeds.
    ///
    /// @param position The bus position of the slave (1-based for SOEM compatibility).
    /// @returns SlaveDataOffsets containing the actual offsets and sizes.
    [[nodiscard]] virtual SlaveDataOffsets slave_data_offsets(uint16_t position) const = 0;

    /// Returns the active domain after activation.
    ///
    /// This returns the domain that contains the actual process data IOmap.
    /// Must only be called after activate() succeeds. The returned pointer
    /// is valid until deactivate() is called.
    ///
    /// @returns Pointer to the active domain, or nullptr if not activated.
    [[nodiscard]] virtual Domain *active_domain() const = 0;
};
}
