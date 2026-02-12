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
#include <mutex>
#include <set>
#include <span>
#include <unordered_map>
#include <vector>

#include "glog/logging.h"

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/master/master.h"
#include "driver/ethercat/pdo/pdo.h"
#include "driver/ethercat/soem/api.h"
#include "driver/ethercat/soem/util.h"

namespace driver::ethercat::soem {
/// @brief SOEM-based implementation of the Master interface.
class Master final : public master::Master {
    /// @brief the network interface name (e.g., "eth0", "enp3s0").
    std::string iface_name;
    /// @brief API abstraction for SOEM operations.
    std::unique_ptr<API> api;
    /// @brief IOmap buffer for PDO exchange.
    std::vector<uint8_t> iomap;
    /// @brief input size in bytes (TxPDO, slave->master).
    size_t input_sz;
    /// @brief output size in bytes (RxPDO, master->slave).
    size_t output_sz;
    /// @brief cached PDO offsets computed at activation time.
    pdo::Offsets pdo_offsets;
    /// @brief cached slave information populated during initialization.
    std::vector<slave::DiscoveryResult> slave_list;
    /// @brief slave positions that are disabled (excluded from cyclic exchange).
    std::set<uint16_t> disabled_slaves;
    /// @brief protects slave state queries.
    mutable std::mutex mu;
    /// @brief whether the master has been initialized.
    bool initialized;
    /// @brief whether the master has been activated.
    bool activated;
    /// @brief expected working counter value for cyclic exchange.
    int expected_wkc;

public:
    /// @brief constructs a SOEM master with injected API and interface name.
    Master(std::unique_ptr<API> api, std::string interface_name);

    ~Master() override;

    Master(const Master &) = delete;
    Master &operator=(const Master &) = delete;

    x::errors::Error initialize() override;

    x::errors::Error register_pdos(const std::vector<pdo::Entry> &entries) override;

    void set_slave_enabled(uint16_t position, bool enabled) override;

    x::errors::Error activate() override;

    void deactivate() override;

    x::errors::Error receive() override;

    x::errors::Error send() override;

    std::span<const uint8_t> input_data() override;

    std::span<uint8_t> output_data() override;

    pdo::Offset pdo_offset(const pdo::Entry &entry) const override;

    std::vector<slave::DiscoveryResult> slaves() const override;

    slave::State slave_state(uint16_t position) const override;

    bool all_slaves_operational() const override;

    std::string interface_name() const override;

private:
    /// @brief populates the cached slave list from SOEM's slavelist.
    void populate_slaves();

    /// @brief computes and caches PDO offsets for all slaves after activation.
    void cache_pdo_offsets();

    /// @brief discovers PDO entries for a slave and populates its PDO lists.
    void discover_slave_pdos(slave::DiscoveryResult &slave);

    /// @brief discovers PDOs using CoE SDO reads (primary method).
    bool discover_pdos_coe(slave::DiscoveryResult &slave);

    /// @brief discovers PDOs from SII EEPROM (fallback method).
    void discover_pdos_sii(slave::DiscoveryResult &slave);

    /// @brief reads PDO assignment object to get list of assigned PDOs.
    x::errors::Error read_pdo_assign(
        uint16_t slave_pos,
        uint16_t assign_index,
        bool is_input,
        slave::DiscoveryResult &slave
    );

    /// @brief reads PDO mapping entries from a specific PDO.
    x::errors::Error read_pdo_mapping(
        uint16_t slave_pos,
        uint16_t pdo_index,
        bool is_input,
        slave::DiscoveryResult &slave
    );

    /// @brief reads the name of a PDO entry from the CoE object dictionary.
    std::string
    read_pdo_entry_name(uint16_t slave_pos, uint16_t index, uint8_t sub_index);

    /// @brief scans the CoE object dictionary to find PDO mapping indices.
    bool
    scan_object_dictionary_for_pdos(uint16_t slave_pos, slave::DiscoveryResult &slave);

    /// @brief transitions all slaves to the specified state.
    x::errors::Error request_state(uint16_t state, int timeout);
};

/// @brief SOEM-based implementation of master::Manager.
class Manager final : public master::Manager {
public:
    [[nodiscard]] std::vector<master::Info> enumerate() override;

    [[nodiscard]] std::pair<std::shared_ptr<master::Master>, x::errors::Error>
    create(const std::string &key) override;
};

}
