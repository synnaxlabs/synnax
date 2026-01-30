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
#include <memory>
#include <mutex>
#include <string>
#include <unordered_map>
#include <vector>

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/igh/ecrt.h"
#include "driver/ethercat/master/master.h"

namespace ethercat::igh {
class Master;

/// IgH EtherLab implementation of the Domain interface.
///
/// Domain manages PDO registration and process data exchange using the IgH
/// EtherCAT master kernel module. Unlike SOEM, IgH requires explicit PDO
/// registration via ecrt_slave_config_reg_pdo_entry() before activation.
///
/// Key differences from SOEM:
/// - Offsets are assigned during registration (not activation)
/// - Requires slave configuration before PDO registration (lazy config)
/// - Domain data pointer is only valid after master activation
class Domain final : public ethercat::Domain {
    /// IgH domain handle from ecrt_master_create_domain().
    ec_domain_t *domain;

    /// Back-pointer to master for lazy slave configuration.
    Master *master;

    /// Process data pointer (valid only after activation).
    uint8_t *domain_data;

    /// Offsets filled by IgH during PDO registration.
    std::vector<unsigned int> pdo_offsets;

    /// Tracked PDO entries for offset lookup.
    std::vector<PDOEntry> registered_entries;

    /// Total input size (TxPDO, slave to master) in bytes.
    size_t input_sz;

    /// Total output size (RxPDO, master to slave) in bytes.
    size_t output_sz;

    /// Whether the master has been activated.
    bool activated;

public:
    /// Constructs an IgH domain wrapper.
    /// @param domain IgH domain handle from ecrt_master_create_domain().
    /// @param master Back-pointer to the owning master.
    Domain(ec_domain_t *domain, Master *master);

    std::pair<size_t, xerrors::Error> register_pdo(const PDOEntry &entry) override;

    uint8_t *data() override;

    size_t size() const override;

    size_t input_size() const override;

    size_t output_size() const override;

    /// Returns the native IgH domain handle.
    ec_domain_t *native_handle() { return this->domain; }

    /// Sets the domain as activated with the given data pointer.
    /// Called by Master after ecrt_master_activate().
    /// @param data_ptr Process data pointer from ecrt_domain_data().
    void set_activated(uint8_t *data_ptr);

    /// Returns data offsets for a specific slave position.
    /// @param position The slave position (0-based).
    /// @returns SlaveDataOffsets with input/output offsets and sizes.
    SlaveDataOffsets get_slave_offsets(uint16_t position) const;
};

/// IgH EtherLab implementation of the Master interface.
///
/// Master wraps the IgH EtherCAT master kernel module (/dev/EtherCAT0) to
/// provide real-time EtherCAT master functionality on Linux. The kernel module
/// handles the actual Ethernet communication while this class manages the
/// userspace API.
///
/// Key differences from SOEM:
/// - Kernel module based (requires ec_master.ko loaded)
/// - Master index based (not interface name based)
/// - Linux only (no Windows/macOS support)
/// - Explicit PDO registration required
/// - Slave configuration required before PDO registration
///
/// Thread safety: The cyclic methods (receive/process/queue/send) must be called
/// from a single thread. Initialization and slave queries are thread-safe.
class Master final : public ethercat::Master {
    /// IgH master index (typically 0, configured in /etc/ethercat.conf).
    unsigned int master_index;

    /// IgH master handle from ecrt_request_master().
    ec_master_t *ec_master;

    /// The domain for PDO exchange.
    std::unique_ptr<Domain> domain;

    /// Cached slave information populated during initialization.
    std::vector<SlaveInfo> cached_slaves;

    /// Lazily configured slave handles (position -> slave_config).
    std::unordered_map<uint16_t, ec_slave_config_t *> slave_configs;

    /// Protects slave state queries and configuration.
    mutable std::mutex mutex;

    /// Whether the master has been initialized.
    bool initialized;

    /// Whether the master has been activated.
    bool activated;

    /// Domain state for WKC checking.
    ec_domain_state_t domain_state;

public:
    /// Constructs an IgH master for the specified master index.
    /// @param master_index The IgH master index (default 0).
    explicit Master(unsigned int master_index = 0);

    ~Master() override;

    Master(const Master &) = delete;
    Master &operator=(const Master &) = delete;

    xerrors::Error initialize() override;

    std::unique_ptr<ethercat::Domain> create_domain() override;

    xerrors::Error activate() override;

    void deactivate() override;

    xerrors::Error receive() override;

    xerrors::Error process(ethercat::Domain &domain) override;

    xerrors::Error queue(ethercat::Domain &domain) override;

    xerrors::Error send() override;

    std::vector<SlaveInfo> slaves() const override;

    SlaveState slave_state(uint16_t position) const override;

    bool all_slaves_operational() const override;

    std::string interface_name() const override;

    SlaveDataOffsets slave_data_offsets(uint16_t position) const override;

    ethercat::Domain *active_domain() const override;

    /// Returns or creates a slave configuration (lazy configuration).
    ///
    /// IgH requires ecrt_master_slave_config() to be called before PDO
    /// registration. This method lazily creates the slave configuration
    /// on first access.
    ///
    /// @param position The slave position (0-based).
    /// @returns The slave configuration handle, or nullptr on failure.
    ec_slave_config_t *get_or_create_slave_config(uint16_t position);

private:
    /// Converts IgH slave state to our SlaveState enum.
    static SlaveState convert_state(uint8_t igh_state);

    /// Discovers PDO entries for a slave and populates its PDO lists.
    /// @param slave The SlaveInfo to populate with discovered PDOs.
    void discover_slave_pdos(SlaveInfo &slave);
};

/// Checks if the IgH EtherCAT master kernel module is available.
/// @returns true if /dev/EtherCAT0 exists, false otherwise.
bool igh_available();
}
