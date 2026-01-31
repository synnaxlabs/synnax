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
#include <mutex>
#include <string>
#include <unordered_map>
#include <vector>

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/igh/ecrt.h"
#include "driver/ethercat/master/master.h"

namespace ethercat::igh {
/// Key for PDO offset cache lookup.
struct PDOEntryKey {
    uint16_t slave_position;
    uint16_t index;
    uint8_t subindex;
    bool is_input;

    bool operator==(const PDOEntryKey &other) const {
        return slave_position == other.slave_position && index == other.index &&
               subindex == other.subindex && is_input == other.is_input;
    }
};

/// Hash function for PDOEntryKey.
struct PDOEntryKeyHash {
    size_t operator()(const PDOEntryKey &key) const {
        return std::hash<uint64_t>()(
            (static_cast<uint64_t>(key.slave_position) << 32) |
            (static_cast<uint64_t>(key.index) << 16) |
            (static_cast<uint64_t>(key.subindex) << 8) |
            static_cast<uint64_t>(key.is_input)
        );
    }
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
///
/// Thread safety: The cyclic methods (receive/send) must be called from a single
/// thread. Initialization and slave queries are thread-safe.
class Master final : public ethercat::Master {
    /// IgH master index (typically 0, configured in /etc/ethercat.conf).
    unsigned int master_index;

    /// IgH master handle from ecrt_request_master().
    ec_master_t *ec_master;

    /// IgH domain handle from ecrt_master_create_domain().
    ec_domain_t *domain;

    /// Process data pointer (valid only after activation).
    uint8_t *domain_data;

    /// Input size in bytes (TxPDO, slave→master).
    size_t input_sz;

    /// Output size in bytes (RxPDO, master→slave).
    size_t output_sz;

    /// Cached PDO offsets computed during activation.
    std::unordered_map<PDOEntryKey, size_t, PDOEntryKeyHash> pdo_offset_cache;

    /// Cached slave information populated during initialization.
    std::vector<SlaveInfo> cached_slaves;

    /// Lazily configured slave handles (position -> slave_config).
    std::unordered_map<uint16_t, ec_slave_config_t *> slave_configs;

    /// Protects slave state queries and configuration.
    mutable std::mutex mu;

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

    xerrors::Error activate() override;

    void deactivate() override;

    xerrors::Error receive() override;

    xerrors::Error send() override;

    uint8_t *input_data() override;

    size_t input_size() const override;

    uint8_t *output_data() override;

    size_t output_size() const override;

    size_t pdo_offset(const PDOEntry &entry) const override;

    std::vector<SlaveInfo> slaves() const override;

    SlaveState slave_state(uint16_t position) const override;

    bool all_slaves_operational() const override;

    std::string interface_name() const override;

    /// Returns or creates a slave configuration (lazy configuration).
    ///
    /// IgH requires ecrt_master_slave_config() to be called before PDO
    /// registration. This method lazily creates the slave configuration
    /// on first access.
    ///
    /// @param position The slave position (0-based).
    /// @returns The slave configuration handle, or nullptr on failure.
    ec_slave_config_t *get_or_create_slave_config(uint16_t position);

    /// Registers a PDO entry for cyclic exchange.
    ///
    /// Must be called before activate(). Returns the offset where this
    /// entry's data will be located after activation.
    ///
    /// @param entry PDO entry describing the slave position, index, subindex, etc.
    /// @returns A pair containing offset and error (NIL on success).
    std::pair<size_t, xerrors::Error> register_pdo(const PDOEntry &entry);

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
