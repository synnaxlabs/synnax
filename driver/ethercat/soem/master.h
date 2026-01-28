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
/// SOEM-based implementation of the Domain interface.
///
/// SOEMDomain manages the IOmap buffer that SOEM uses for PDO exchange.
/// PDO registration is implicit in SOEM - it's handled during ecx_config_map_group().
/// This class provides offset tracking for user-level PDO access.
class SOEMDomain final : public Domain {
    /// The IOmap buffer used by SOEM for process data exchange.
    std::vector<uint8_t> iomap_;

    /// Current offset for input PDOs (TxPDO, slave→master).
    size_t input_offset_;

    /// Current offset for output PDOs (RxPDO, master→slave).
    size_t output_offset_;

    /// Total input size after configuration.
    size_t input_size_;

    /// Total output size after configuration.
    size_t output_size_;

    /// Registered PDO entries with their offsets.
    std::vector<std::pair<PDOEntry, size_t>> registered_pdos_;

public:
    /// Constructs a domain with the specified IOmap size.
    /// @param iomap_size Size of the IOmap buffer (default 4096 bytes).
    explicit SOEMDomain(size_t iomap_size = 4096);

    std::pair<size_t, xerrors::Error> register_pdo(const PDOEntry &entry) override;

    uint8_t *data() override;

    size_t size() const override;

    size_t input_size() const override;

    size_t output_size() const override;

    /// Returns a pointer to the raw IOmap buffer for SOEM configuration.
    /// Called internally by SOEMMaster during activation.
    uint8_t *iomap_ptr() { return iomap_.data(); }

    /// Sets the actual input/output sizes after SOEM configuration.
    /// Called internally by SOEMMaster after ecx_config_map_group().
    /// @param input_size Total input bytes from all slaves.
    /// @param output_size Total output bytes from all slaves.
    void set_sizes(size_t input_size, size_t output_size);
};

/// SOEM-based implementation of the Master interface.
///
/// SOEMMaster wraps the SOEM library to provide EtherCAT master functionality.
/// It manages the ecx_contextt and handles the lifecycle of the EtherCAT network.
///
/// Thread safety: The cyclic methods (receive/process/queue/send) must be called
/// from a single thread. Initialization and slave queries are thread-safe.
class SOEMMaster final : public Master {
    /// The network interface name (e.g., "eth0", "enp3s0").
    std::string interface_name_;

    /// The SOEM context containing all network state.
    ecx_contextt context_;

    /// Cached slave information populated during initialization.
    std::vector<SlaveInfo> slaves_;

    /// Protects slave state queries.
    mutable std::mutex mutex_;

    /// Whether the master has been initialized.
    bool initialized_;

    /// Whether the master has been activated.
    bool activated_;

    /// Pointer to the domain (borrowed during activate).
    SOEMDomain *domain_;

    /// Expected working counter value for cyclic exchange.
    int expected_wkc_;

public:
    /// Constructs a SOEM master for the specified network interface.
    /// @param interface_name The network interface name (e.g., "eth0").
    explicit SOEMMaster(std::string interface_name);

    ~SOEMMaster() override;

    SOEMMaster(const SOEMMaster &) = delete;
    SOEMMaster &operator=(const SOEMMaster &) = delete;

    xerrors::Error initialize() override;

    std::unique_ptr<Domain> create_domain() override;

    xerrors::Error activate() override;

    void deactivate() override;

    xerrors::Error receive() override;

    xerrors::Error process(Domain &domain) override;

    xerrors::Error queue(Domain &domain) override;

    xerrors::Error send() override;

    std::vector<SlaveInfo> slaves() const override;

    SlaveState slave_state(uint16_t position) const override;

    bool all_slaves_operational() const override;

    std::string interface_name() const override;

    SlaveDataOffsets slave_data_offsets(uint16_t position) const override;

    Domain *active_domain() const override;

private:
    /// Converts SOEM slave state to our SlaveState enum.
    static SlaveState convert_state(uint16_t soem_state);

    /// Populates the cached slave list from SOEM's slavelist.
    void populate_slaves();

    /// Transitions all slaves to the specified state.
    /// @param state Target EtherCAT state (EC_STATE_*).
    /// @param timeout Timeout in microseconds.
    /// @returns xerrors::NIL on success.
    xerrors::Error request_state(uint16_t state, int timeout);
};
}
