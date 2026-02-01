// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstring>
#include <mutex>
#include <span>
#include <unordered_map>
#include <vector>

extern "C" {
#include "soem/soem.h"
}

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/master/master.h"

namespace ethercat::soem {
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

/// SOEM-based implementation of the ethercat::master::Master interface.
///
/// Master wraps the SOEM library to provide EtherCAT master functionality.
/// It manages the ecx_contextt and handles the lifecycle of the EtherCAT network.
///
/// Thread safety: The cyclic methods (receive/send) must be called from a single
/// thread. Initialization and slave queries are thread-safe.
class Master final : public master::Master {
    /// The network interface name (e.g., "eth0", "enp3s0").
    std::string iface_name;

    /// The SOEM context containing all network state.
    ecx_contextt context;

    /// IOmap buffer for PDO exchange.
    std::vector<uint8_t> iomap;

    /// Input size in bytes (TxPDO, slave→master).
    size_t input_sz;

    /// Output size in bytes (RxPDO, master→slave).
    size_t output_sz;

    /// Cached PDO offsets computed at activation time.
    std::unordered_map<PDOEntryKey, size_t, PDOEntryKeyHash> pdo_offset_cache;

    /// Cached slave information populated during initialization.
    std::vector<SlaveInfo> slave_list;

    /// Protects slave state queries.
    mutable std::mutex mu;

    /// Whether the master has been initialized.
    bool initialized;

    /// Whether the master has been activated.
    bool activated;

    /// Expected working counter value for cyclic exchange.
    int expected_wkc;

public:
    /// Constructs a SOEM master for the specified network interface.
    /// @param interface_name The network interface name (e.g., "eth0").
    explicit Master(std::string interface_name);

    ~Master() override;

    Master(const Master &) = delete;
    Master &operator=(const Master &) = delete;

    xerrors::Error initialize() override;

    xerrors::Error activate() override;

    void deactivate() override;

    xerrors::Error receive() override;

    xerrors::Error send() override;

    std::span<const uint8_t> input_data() override;

    std::span<uint8_t> output_data() override;

    size_t pdo_offset(const PDOEntry &entry) const override;

    std::vector<SlaveInfo> slaves() const override;

    SlaveState slave_state(uint16_t position) const override;

    bool all_slaves_operational() const override;

    std::string interface_name() const override;

private:
    /// Converts SOEM slave state to our SlaveState enum.
    static SlaveState convert_state(uint16_t soem_state);

    /// Populates the cached slave list from SOEM's slavelist.
    void populate_slaves();

    /// Computes and caches PDO offsets for all slaves after activation.
    void cache_pdo_offsets();

    /// Discovers PDO entries for a slave and populates its PDO lists.
    /// @param slave The SlaveInfo to populate with discovered PDOs.
    void discover_slave_pdos(SlaveInfo &slave);

    /// Discovers PDOs using CoE SDO reads (primary method).
    /// @param slave The SlaveInfo to populate.
    /// @returns true if discovery succeeded, false to try SII fallback.
    bool discover_pdos_coe(SlaveInfo &slave);

    /// Discovers PDOs from SII EEPROM (fallback method).
    /// @param slave The SlaveInfo to populate.
    void discover_pdos_sii(SlaveInfo &slave);

    /// Reads PDO assignment object to get list of assigned PDOs.
    /// @param slave_pos The 1-based slave position.
    /// @param assign_index The assignment index (0x1C12 for RxPDO, 0x1C13 for TxPDO).
    /// @param is_input True for TxPDO (inputs), false for RxPDO (outputs).
    /// @param slave The SlaveInfo to populate with entries.
    /// @returns xerrors::NIL on success.
    xerrors::Error read_pdo_assign(
        uint16_t slave_pos,
        uint16_t assign_index,
        bool is_input,
        SlaveInfo &slave
    );

    /// Reads PDO mapping entries from a specific PDO.
    /// @param slave_pos The 1-based slave position.
    /// @param pdo_index The PDO index (e.g., 0x1600, 0x1A00).
    /// @param is_input True for TxPDO (inputs), false for RxPDO (outputs).
    /// @param slave The SlaveInfo to populate with entries.
    /// @returns xerrors::NIL on success.
    xerrors::Error read_pdo_mapping(
        uint16_t slave_pos,
        uint16_t pdo_index,
        bool is_input,
        SlaveInfo &slave
    );

    /// Reads the name of a PDO entry from the CoE object dictionary.
    /// @param slave_pos The 1-based slave position.
    /// @param index The object dictionary index.
    /// @param subindex The object dictionary subindex.
    /// @returns The entry name, or empty string on failure.
    std::string
    read_pdo_entry_name(uint16_t slave_pos, uint16_t index, uint8_t subindex);

    /// Scans the CoE object dictionary to find PDO mapping indices.
    /// Used as fallback when standard PDO assignment objects (0x1C12/0x1C13) don't
    /// exist.
    /// @param slave_pos The 1-based slave position.
    /// @param slave The SlaveInfo to populate with discovered PDOs.
    /// @returns true if any PDOs were discovered.
    bool scan_object_dictionary_for_pdos(uint16_t slave_pos, SlaveInfo &slave);

    /// Transitions all slaves to the specified state.
    /// @param state Target EtherCAT state (EC_STATE_*).
    /// @param timeout Timeout in microseconds.
    /// @returns xerrors::NIL on success.
    xerrors::Error request_state(uint16_t state, int timeout);
};
}
