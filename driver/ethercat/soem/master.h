// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <map>
#include <memory>
#include <mutex>
#include <optional>
#include <set>
#include <span>
#include <string>
#include <unordered_map>
#include <vector>

#include "driver/ethercat/master/master.h"
#include "driver/ethercat/pdo/pdo.h"
#include "driver/ethercat/soem/api.h"
#include "driver/ethercat/telem/telem.h"

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
    /// @brief why each slave whose discovered layout disagrees with what SOEM mapped
    /// has no offsets, keyed by position.
    std::unordered_map<uint16_t, x::errors::Error> layout_errors;
    /// @brief cached slave information populated during initialization.
    std::vector<slave::DiscoveryResult> slave_list;
    /// @brief slave positions that are disabled (excluded from cyclic exchange).
    std::set<uint16_t> disabled_slaves;
    /// @brief CoE objects written to each slave before SAFE-OP, keyed by position.
    std::map<uint16_t, std::vector<slave::StartupParameter>> startup_parameters;
    /// @brief why each slave rejected a startup parameter at its latest move to
    /// SAFE-OP, keyed by position.
    std::map<uint16_t, std::string> startup_failures;
    /// @brief protects slave state queries.
    mutable std::mutex mu;
    /// @brief whether the master has been initialized.
    bool initialized;
    /// @brief whether the master has been activated.
    bool activated;
    /// @brief expected working counter value for cyclic exchange.
    int expected_wkc;
    /// @brief whether the enabled slaves' frames carry the reference clock time.
    bool has_dc = false;

public:
    /// @brief constructs a SOEM master with injected API and interface name.
    Master(std::unique_ptr<API> api, std::string interface_name);

    ~Master() override;

    Master(const Master &) = delete;
    Master &operator=(const Master &) = delete;

    x::errors::Error initialize() override;

    std::pair<size_t, x::errors::Error> count_slaves() override;

    void set_slave_config(uint16_t position, slave::Config config) override;

    x::errors::Error activate() override;

    void deactivate() override;

    x::errors::Error receive() override;

    x::errors::Error send() override;

    std::span<const uint8_t> input_data() override;

    std::span<uint8_t> output_data() override;

    std::pair<pdo::Offset, x::errors::Error>
    pdo_offset(const pdo::Entry &entry) const override;

    std::vector<slave::DiscoveryResult> slaves() const override;

    std::vector<slave::Health> check(const x::breaker::Breaker &breaker) override;

    std::optional<x::telem::TimeStamp> dc_time() const override;

    std::string interface_name() const override;

private:
    /// @brief writes a slave's startup parameters in order, recording why the slave
    /// rejected the first one that fails.
    void write_startup_parameters(
        uint16_t pos,
        const std::vector<slave::StartupParameter> &parameters
    );

    /// @brief populates the cached slave list from SOEM's slavelist.
    void populate_slaves();

    /// @brief reads the identity and PDO layout of the slave at a position.
    slave::DiscoveryResult discover_slave(uint16_t pos);

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

    /// @brief reads the name of a CoE object, or nullopt when the slave has none.
    std::optional<std::string> read_object_name(uint16_t slave_pos, uint16_t index);

    /// @brief reads PDO mapping entries from a specific PDO.
    x::errors::Error read_pdo_mapping(
        uint16_t slave_pos,
        uint16_t pdo_index,
        bool is_input,
        slave::DiscoveryResult &slave
    );

    /// @brief reads the name and CoE data type of a PDO entry from the object
    /// dictionary.
    std::pair<std::string, telem::DataType>
    read_pdo_entry(uint16_t slave_pos, uint16_t index, uint8_t sub_index);

    /// @brief scans the CoE object dictionary to find PDO mapping indices.
    bool
    scan_object_dictionary_for_pdos(uint16_t slave_pos, slave::DiscoveryResult &slave);

    /// @brief transitions all enabled slaves to the specified state.
    x::errors::Error request_state(uint16_t state, int timeout);

    /// @brief transitions all enabled slaves to OP while exchanging process data, so
    /// slaves with a sync manager watchdog see valid outputs.
    x::errors::Error request_operational();

    /// @brief takes one recovery step for a slave that is not in OP, following the SOEM
    /// ecatcheck sample: acknowledge an error, request OP, reconfigure, or restore a
    /// slave that lost power. Takes no step for a slave that rejected a startup
    /// parameter, so it stays in PRE-OP until the next activation.
    void recover(uint16_t pos);

    /// @brief builds the error for enabled slaves that did not reach a state.
    x::errors::Error state_error(const std::vector<uint16_t> &failed, uint16_t state);
};

/// @brief SOEM-based implementation of master::Manager.
class Manager final : public master::Manager {
public:
    [[nodiscard]] std::vector<master::Info> enumerate() override;

    [[nodiscard]] std::pair<std::shared_ptr<master::Master>, x::errors::Error>
    create(const std::string &key) override;
};

}
