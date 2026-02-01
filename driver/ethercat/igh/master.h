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
class Master final : public ethercat::master::Master {
    /// IgH master index (typically 0, configured in /etc/ethercat.conf).
    unsigned int master_index;

    // The following pointers are non-owning references to resources managed by the
    // IgH kernel module. They are cleaned up via ecrt_release_master() in deactivate(),
    // not via delete/free. Smart pointers are intentionally not used.

    /// IgH master handle from ecrt_request_master().
    ec_master_t *ec_master;

    /// IgH input domain handle (LRD datagram, TxPDO data).
    ec_domain_t *input_domain;

    /// IgH output domain handle (LWR datagram, RxPDO data).
    ec_domain_t *output_domain;

    /// Input domain process data pointer (valid only after activation).
    uint8_t *input_domain_data;

    /// Output domain process data pointer (valid only after activation).
    uint8_t *output_domain_data;

    /// Input size in bytes (TxPDO, slave→master).
    size_t input_sz;

    /// Output size in bytes (RxPDO, master→slave).
    size_t output_sz;

    /// Cached PDO offsets computed during activation.
    std::unordered_map<PDOEntryKey, master::PDOOffset, PDOEntryKeyHash>
        pdo_offset_cache;

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

    /// Input domain state for WKC checking.
    ec_domain_state_t input_domain_state;

    /// Output domain state for WKC checking.
    ec_domain_state_t output_domain_state;

public:
    /// Constructs an IgH master for the specified master index.
    /// @param master_index The IgH master index (default 0).
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

    /// Configures the PDO mapping for a slave based on discovered PDOs.
    /// @param sc The slave configuration handle.
    /// @param slave The SlaveInfo containing discovered PDOs.
    void configure_slave_pdos(ec_slave_config_t *sc, const SlaveInfo &slave);

    /// Reads the name of a PDO entry from the slave's object dictionary via SDO.
    /// @param slave_pos The slave position on the bus.
    /// @param index The object dictionary index.
    /// @param subindex The object dictionary subindex.
    /// @returns The entry name, or empty string if read fails.
    std::string
    read_pdo_entry_name(uint16_t slave_pos, uint16_t index, uint8_t subindex);
};

/// Checks if the IgH EtherCAT master kernel module is available.
/// @returns true if /dev/EtherCAT0 exists, false otherwise.
bool igh_available();

/// Sysfs path where IgH EtherCAT masters are exposed.
const std::string SYSFS_ETHERCAT_PATH = "/sys/class/EtherCAT";

/// IgH-based implementation of master::Manager.
///
/// Reads /sys/class/EtherCAT/ to enumerate configured IgH EtherCAT masters
/// and creates igh::Master instances for each.
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
