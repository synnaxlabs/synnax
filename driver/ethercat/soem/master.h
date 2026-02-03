// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <mutex>
#include <set>
#include <span>
#include <unordered_map>
#include <vector>

#include "glog/logging.h"

extern "C" {
#include "soem/soem.h"
}

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/master/master.h"
#include "driver/ethercat/pdo/pdo.h"

namespace ethercat::soem {
/// @brief SOEM-based implementation of the Master interface.
class Master final : public master::Master {
    /// @brief the network interface name (e.g., "eth0", "enp3s0").
    std::string iface_name;
    /// @brief the SOEM context containing all network state.
    ecx_contextt context;
    /// @brief IOmap buffer for PDO exchange.
    std::vector<uint8_t> iomap;
    /// @brief input size in bytes (TxPDO, slave→master).
    size_t input_sz;
    /// @brief output size in bytes (RxPDO, master→slave).
    size_t output_sz;
    /// @brief cached PDO offsets computed at activation time.
    pdo::Offsets pdo_offsets;
    /// @brief cached slave information populated during initialization.
    std::vector<slave::Properties> slave_list;
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
    /// @brief constructs a SOEM master for the specified network interface.
    explicit Master(std::string interface_name);

    ~Master() override;

    Master(const Master &) = delete;
    Master &operator=(const Master &) = delete;

    xerrors::Error initialize() override;

    xerrors::Error register_pdos(const std::vector<pdo::Entry> &entries) override;

    void set_slave_enabled(uint16_t position, bool enabled) override;

    xerrors::Error activate() override;

    void deactivate() override;

    xerrors::Error receive() override;

    xerrors::Error send() override;

    std::span<const uint8_t> input_data() override;

    std::span<uint8_t> output_data() override;

    pdo::Offset pdo_offset(const pdo::Entry &entry) const override;

    std::vector<slave::Properties> slaves() const override;

    slave::State slave_state(uint16_t position) const override;

    bool all_slaves_operational() const override;

    std::string interface_name() const override;

private:
    /// @brief converts SOEM slave state to our slave::State enum.
    static slave::State convert_state(uint16_t soem_state);

    /// @brief populates the cached slave list from SOEM's slavelist.
    void populate_slaves();

    /// @brief computes and caches PDO offsets for all slaves after activation.
    void cache_pdo_offsets();

    /// @brief discovers PDO entries for a slave and populates its PDO lists.
    void discover_slave_pdos(slave::Properties &slave);

    /// @brief discovers PDOs using CoE SDO reads (primary method).
    bool discover_pdos_coe(slave::Properties &slave);

    /// @brief discovers PDOs from SII EEPROM (fallback method).
    void discover_pdos_sii(slave::Properties &slave);

    /// @brief reads PDO assignment object to get list of assigned PDOs.
    xerrors::Error read_pdo_assign(
        uint16_t slave_pos,
        uint16_t assign_index,
        bool is_input,
        slave::Properties &slave
    );

    /// @brief reads PDO mapping entries from a specific PDO.
    xerrors::Error read_pdo_mapping(
        uint16_t slave_pos,
        uint16_t pdo_index,
        bool is_input,
        slave::Properties &slave
    );

    /// @brief reads the name of a PDO entry from the CoE object dictionary.
    std::string
    read_pdo_entry_name(uint16_t slave_pos, uint16_t index, uint8_t sub_index);

    /// @brief scans the CoE object dictionary to find PDO mapping indices.
    bool scan_object_dictionary_for_pdos(uint16_t slave_pos, slave::Properties &slave);

    /// @brief transitions all slaves to the specified state.
    xerrors::Error request_state(uint16_t state, int timeout);
};

/// @brief SOEM-based implementation of master::Manager.
class Manager final : public master::Manager {
public:
    [[nodiscard]] std::vector<master::Info> enumerate() override {
        std::vector<master::Info> masters;
        ec_adaptert *adapter = ec_find_adapters();
        ec_adaptert *current = adapter;

        while (current != nullptr) {
            if (is_physical_interface(current->name)) {
                master::Info info;
                info.key = current->name;
                info.description = current->desc;
                masters.push_back(std::move(info));
            } else {
                VLOG(2) << "[ethercat] skipping virtual interface: " << current->name;
            }
            current = current->next;
        }

        ec_free_adapters(adapter);
        return masters;
    }

    [[nodiscard]] std::pair<std::shared_ptr<master::Master>, xerrors::Error>
    create(const std::string &key) override {
        if (key.empty())
            return {nullptr, xerrors::Error(MASTER_INIT_ERROR, "empty interface name")};
        if (key.size() >= 4 && key.substr(0, 4) == "igh:")
            return {
                nullptr,
                xerrors::Error(
                    MASTER_INIT_ERROR,
                    "invalid SOEM interface '" + key + "': IgH-style keys not supported"
                )
            };
        return {std::make_shared<Master>(key), xerrors::NIL};
    }

private:
    static bool is_physical_interface(const std::string &name) {
        if (name == "lo" || name == "localhost") return false;
        if (name.find("tailscale") != std::string::npos) return false;
        if (name.find("tun") == 0) return false;
        if (name.find("tap") == 0) return false;
        if (name.find("veth") == 0) return false;
        if (name.find("docker") != std::string::npos) return false;
        if (name.find("br-") == 0) return false;
        if (name.find("virbr") == 0) return false;
        return true;
    }
};

}
