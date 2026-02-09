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
#include <unordered_set>
#include <vector>

#include <dirent.h>
#include <sys/stat.h>

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/igh/api.h"
#include "driver/ethercat/igh/ecrt.h"
#include "driver/ethercat/master/master.h"
#include "driver/ethercat/pdo/pdo.h"

namespace driver::ethercat::igh {

/// @brief IgH EtherLab implementation of the Master interface.
class Master final : public master::Master {
    /// @brief API wrapper for dynamic library loading.
    std::shared_ptr<API> api;
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
    pdo::Offsets pdo_offsets;
    /// @brief cached slave information populated during initialization.
    std::vector<slave::DiscoveryResult> cached_slaves;
    /// @brief lazily configured slave handles (position -> slave_config).
    std::unordered_map<uint16_t, ec_slave_config_t *> slave_configs;
    /// @brief slaves that are disabled (excluded from cyclic exchange).
    std::unordered_set<uint16_t> disabled_slaves;
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
    /// @brief constructs an IgH master with the given API and master index.
    explicit Master(std::shared_ptr<API> api, unsigned int master_index = 0);

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

    /// @brief returns or creates a slave configuration for the given position.
    ec_slave_config_t *get_or_create_slave_config(uint16_t position);

    /// @brief registers a PDO entry for cyclic exchange.
    std::pair<size_t, x::errors::Error> register_pdo(const pdo::Entry &entry);

private:
    /// @brief converts IgH slave state to our slave::State enum.
    static slave::State convert_state(uint8_t igh_state);

    /// @brief discovers PDO entries for a slave and populates its PDO lists.
    void discover_slave_pdos(slave::DiscoveryResult &slave);

    /// @brief configures the PDO mapping for a slave based on discovered PDOs.
    void configure_slave_pdos(ec_slave_config_t *sc, const slave::Properties &slave);

    /// @brief reads the name of a PDO entry from the slave's object dictionary.
    std::string
    read_pdo_entry_name(uint16_t slave_pos, uint16_t index, uint8_t subindex);
};

/// @brief sysfs path where IgH EtherCAT masters are exposed.
/// The IgH EtherCAT master kernel module exposes masters as
/// /sys/class/EtherCAT/EtherCAT<n>.
const std::string SYSFS_ETHERCAT_PATH = "/sys/class/EtherCAT";

/// @brief length of "EtherCAT" prefix in sysfs device names.
constexpr size_t IGH_SYSFS_PREFIX_LEN = 8;

/// @brief device path for the first IgH EtherCAT master kernel module device.
const std::string IGH_DEVICE_PATH = "/dev/EtherCAT0";

/// @brief IgH-based implementation of master::Manager.
class Manager final : public master::Manager {
    /// @brief API instance loaded during open().
    std::shared_ptr<API> api;

    explicit Manager(std::shared_ptr<API> api): api(std::move(api)) {}

public:
    /// @brief opens the IgH manager, checking device availability and loading the API.
    /// @return pair of manager instance and error (nil on success).
    static std::pair<std::unique_ptr<Manager>, x::errors::Error> open() {
        struct stat st;
        if (stat(IGH_DEVICE_PATH.c_str(), &st) != 0)
            return {
                nullptr,
                x::errors::Error(errors::MASTER_INIT_ERROR, "IgH device not found")
            };
        auto [api, err] = API::load();
        if (err) return {nullptr, err};
        return {std::unique_ptr<Manager>(new Manager(std::move(api))), x::errors::NIL};
    }

    [[nodiscard]] std::vector<master::Info> enumerate() override {
        std::vector<master::Info> masters;
        DIR *dir = opendir(SYSFS_ETHERCAT_PATH.c_str());
        if (dir == nullptr) return masters;

        while (dirent *entry = readdir(dir)) {
            if (std::strncmp(entry->d_name, "EtherCAT", IGH_SYSFS_PREFIX_LEN) != 0)
                continue;

            const char *index_str = entry->d_name + IGH_SYSFS_PREFIX_LEN;
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

    [[nodiscard]] std::pair<std::shared_ptr<master::Master>, x::errors::Error>
    create(const std::string &key) override {
        if (key.size() < 5 || key.substr(0, 4) != "igh:")
            return {
                nullptr,
                x::errors::Error(
                    errors::MASTER_INIT_ERROR,
                    "invalid IgH master key '" + key + "': expected format 'igh:N'"
                )
            };

        try {
            const int index = std::stoi(key.substr(4));
            return {std::make_shared<Master>(this->api, index), x::errors::NIL};
        } catch (const std::exception &) {
            return {
                nullptr,
                x::errors::Error(
                    errors::MASTER_INIT_ERROR,
                    "invalid IgH master key '" + key + "': could not parse index"
                )
            };
        }
    }
};

}
