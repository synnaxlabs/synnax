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
#include <cstring>
#include <mutex>
#include <span>
#include <string>
#include <unordered_map>
#include <vector>

#include <dirent.h>

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/igh/ecrt.h"
#include "driver/ethercat/master/master.h"

namespace ethercat::igh {
/// @brief key for PDO offset cache lookup.
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

/// @brief hash function for PDOEntryKey.
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

/// @brief IgH EtherLab implementation of the Master interface.
class Master final : public ethercat::master::Master {
    /// @brief IgH master index (typically 0, configured in /etc/ethercat.conf).
    unsigned int master_index;

    // Non-owning pointers managed by IgH kernel module via ecrt_release_master().

    /// @brief IgH master handle from ecrt_request_master().
    ec_master_t *ec_master;
    /// @brief IgH input domain handle (LRD datagram, TxPDO data).
    ec_domain_t *input_domain;
    /// @brief IgH output domain handle (LWR datagram, RxPDO data).
    ec_domain_t *output_domain;
    /// @brief input domain process data pointer (valid only after activation).
    uint8_t *input_domain_data;
    /// @brief output domain process data pointer (valid only after activation).
    uint8_t *output_domain_data;
    /// @brief input size in bytes (TxPDO, slave→master).
    size_t input_sz;
    /// @brief output size in bytes (RxPDO, master→slave).
    size_t output_sz;
    /// @brief cached PDO offsets computed during activation.
    std::unordered_map<PDOEntryKey, master::PDOOffset, PDOEntryKeyHash>
        pdo_offset_cache;
    /// @brief cached slave information populated during initialization.
    std::vector<SlaveInfo> cached_slaves;
    /// @brief lazily configured slave handles (position -> slave_config).
    std::unordered_map<uint16_t, ec_slave_config_t *> slave_configs;
    /// @brief protects slave state queries and configuration.
    mutable std::mutex mu;
    /// @brief whether the master has been initialized.
    bool initialized;
    /// @brief whether the master has been activated.
    bool activated;
    /// @brief input domain state for WKC checking.
    ec_domain_state_t input_domain_state;
    /// @brief output domain state for WKC checking.
    ec_domain_state_t output_domain_state;

public:
    /// @brief constructs an IgH master for the specified master index.
    explicit Master(unsigned int master_index = 0);

    ~Master() override;

    Master(const Master &) = delete;
    Master &operator=(const Master &) = delete;

    xerrors::Error initialize() override;

    xerrors::Error register_pdos(const std::vector<PDOEntry> &entries) override;

    xerrors::Error activate() override;

    void deactivate() override;

    xerrors::Error receive() override;

    xerrors::Error send() override;

    std::span<const uint8_t> input_data() override;

    std::span<uint8_t> output_data() override;

    master::PDOOffset pdo_offset(const PDOEntry &entry) const override;

    std::vector<SlaveInfo> slaves() const override;

    SlaveState slave_state(uint16_t position) const override;

    bool all_slaves_operational() const override;

    std::string interface_name() const override;

    /// @brief returns or creates a slave configuration for the given position.
    ec_slave_config_t *get_or_create_slave_config(uint16_t position);

    /// @brief registers a PDO entry for cyclic exchange.
    std::pair<size_t, xerrors::Error> register_pdo(const PDOEntry &entry);

private:
    /// @brief converts IgH slave state to our SlaveState enum.
    static SlaveState convert_state(uint8_t igh_state);

    /// @brief discovers PDO entries for a slave and populates its PDO lists.
    void discover_slave_pdos(SlaveInfo &slave);

    /// @brief configures the PDO mapping for a slave based on discovered PDOs.
    void configure_slave_pdos(ec_slave_config_t *sc, const SlaveInfo &slave);

    /// @brief reads the name of a PDO entry from the slave's object dictionary.
    std::string
    read_pdo_entry_name(uint16_t slave_pos, uint16_t index, uint8_t subindex);
};

/// @brief checks if the IgH EtherCAT master kernel module is available.
bool igh_available();

/// @brief sysfs path where IgH EtherCAT masters are exposed.
const std::string SYSFS_ETHERCAT_PATH = "/sys/class/EtherCAT";

/// @brief IgH-based implementation of master::Manager.
class Manager final : public master::Manager {
public:
    [[nodiscard]] std::vector<master::Info> enumerate() override {
        std::vector<master::Info> masters;
        DIR *dir = opendir(SYSFS_ETHERCAT_PATH.c_str());
        if (dir == nullptr) return masters;

        while (dirent *entry = readdir(dir)) {
            if (std::strncmp(entry->d_name, "EtherCAT", 8) != 0) continue;

            const char *index_str = entry->d_name + 8;
            char *end = nullptr;
            const long index = std::strtol(index_str, &end, 10);
            if (end == index_str || *end != '\0') continue;

            master::Info info;
            info.key = "igh:" + std::to_string(index);
            info.description = "IgH EtherCAT Master " + std::to_string(index);
            masters.push_back(std::move(info));
        }

        closedir(dir);
        return masters;
    }

    [[nodiscard]] std::pair<std::shared_ptr<master::Master>, xerrors::Error>
    create(const std::string &key) override {
        if (key.size() < 5 || key.substr(0, 4) != "igh:")
            return {
                nullptr,
                xerrors::Error(
                    MASTER_INIT_ERROR,
                    "invalid IgH master key '" + key + "': expected format 'igh:N'"
                )
            };
        try {
            const int index = std::stoi(key.substr(4));
            return {std::make_shared<Master>(index), xerrors::NIL};
        } catch (const std::exception &) {
            return {
                nullptr,
                xerrors::Error(
                    MASTER_INIT_ERROR,
                    "invalid IgH master key '" + key + "': could not parse index"
                )
            };
        }
    }
};

}
