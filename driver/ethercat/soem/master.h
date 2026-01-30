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
#include <memory>
#include <mutex>
#include <vector>

extern "C" {
#include "soem/soem.h"
}

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/master/master.h"

namespace ethercat::soem {
/// SOEM-based implementation of the ethercat::Domain interface.
///
/// Domain manages the IOmap buffer that SOEM uses for PDO exchange.
/// PDO registration is implicit in SOEM - it's handled during ecx_config_map_group().
/// This class provides offset tracking for user-level PDO access.
class Domain final : public ethercat::Domain {
    /// The IOmap buffer used by SOEM for process data exchange.
    std::vector<uint8_t> iomap;

    /// Current offset for input PDOs (TxPDO, slave→master).
    size_t input_offset;

    /// Current offset for output PDOs (RxPDO, master→slave).
    size_t output_offset;

    /// Total input size after configuration.
    size_t input_sz;

    /// Total output size after configuration.
    size_t output_sz;

    /// Registered PDO entries with their offsets.
    std::vector<std::pair<PDOEntry, size_t>> registered_pdos;

public:
    /// Constructs a domain with the specified IOmap size.
    /// @param iomap_size Size of the IOmap buffer (default 4096 bytes).
    explicit Domain(size_t iomap_size = 4096);

    std::pair<size_t, xerrors::Error> register_pdo(const PDOEntry &entry) override;

    uint8_t *data() override;

    size_t size() const override;

    size_t input_size() const override;

    size_t output_size() const override;

    /// Returns a pointer to the raw IOmap buffer for SOEM configuration.
    /// Called internally by Master during activation.
    uint8_t *iomap_ptr() { return this->iomap.data(); }

    /// Sets the actual input/output sizes after SOEM configuration.
    /// Called internally by Master after ecx_config_map_group().
    /// @param input_size Total input bytes from all slaves.
    /// @param output_size Total output bytes from all slaves.
    void set_sizes(size_t input_size, size_t output_size);
};

/// SOEM-based implementation of the ethercat::Master interface.
///
/// Master wraps the SOEM library to provide EtherCAT master functionality.
/// It manages the ecx_contextt and handles the lifecycle of the EtherCAT network.
///
/// Thread safety: The cyclic methods (receive/process/queue/send) must be called
/// from a single thread. Initialization and slave queries are thread-safe.
class Master final : public ethercat::Master {
    /// The network interface name (e.g., "eth0", "enp3s0").
    std::string iface_name;

    /// The SOEM context containing all network state.
    ecx_contextt context;

    /// Cached slave information populated during initialization.
    std::vector<SlaveInfo> slave_list;

    /// Protects slave state queries.
    mutable std::mutex mu;

    /// Whether the master has been initialized.
    bool initialized;

    /// Whether the master has been activated.
    bool activated;

    /// Pointer to the domain (borrowed during activate).
    Domain *dom;

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

private:
    /// Converts SOEM slave state to our SlaveState enum.
    static SlaveState convert_state(uint16_t soem_state);

    /// Populates the cached slave list from SOEM's slavelist.
    void populate_slaves();

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

    /// Transitions all slaves to the specified state.
    /// @param state Target EtherCAT state (EC_STATE_*).
    /// @param timeout Timeout in microseconds.
    /// @returns xerrors::NIL on success.
    xerrors::Error request_state(uint16_t state, int timeout);
};
}
