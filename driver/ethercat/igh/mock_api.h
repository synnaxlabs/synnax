// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <cerrno>
#include <cstdint>
#include <cstring>
#include <map>
#include <span>
#include <string>
#include <utility>
#include <vector>

#include "driver/ethercat/igh/api.h"

namespace driver::ethercat::igh {
/// @brief a fake IgH master for tests. Each slave has its own object dictionary and PDO
/// layout, and each configured slave moves to its state when the master activates.
/// Positions count from 0, as IgH counts them.
class MockAPI final : public API {
    static constexpr uintptr_t CONFIG_BASE = 0x100;
    ec_master_t *const handle = reinterpret_cast<ec_master_t *>(0x1);
    ec_domain_t *const output_domain = reinterpret_cast<ec_domain_t *>(0x2);
    ec_domain_t *const input_domain = reinterpret_cast<ec_domain_t *>(0x3);
    mutable int domains_created = 0;
    mutable bool activated = false;

public:
    /// @brief an object that a PDO maps.
    struct Entry {
        uint16_t index;
        uint8_t sub_index;
        uint8_t bit_length;

        bool operator==(const Entry &) const = default;
    };

    /// @brief a PDO and the objects it maps.
    struct PDO {
        uint16_t index;
        std::vector<Entry> entries;

        bool operator==(const PDO &) const = default;
    };

    /// @brief a sync manager and the PDOs it carries.
    struct Sync {
        ec_direction_t dir;
        std::vector<PDO> pdos;
    };

    /// @brief one SDO write: index, sub-index, and bytes.
    struct SDO {
        uint16_t index;
        uint8_t sub_index;
        std::vector<uint8_t> data;

        bool operator==(const SDO &) const = default;
    };

    /// @brief a process data domain.
    struct Domain {
        /// @brief the size in bytes, which grows with each registered entry.
        size_t size = 0;
        std::vector<uint8_t> data;
        ec_domain_state_t state{.working_counter = 0, .wc_state = EC_WC_COMPLETE};
    };

    /// @brief a slave on the bus.
    struct Slave {
        /// @brief the identity and name that master_get_slave reads, with the state and
        /// AL error flag of the slave.
        ec_slave_info_t info{};
        /// @brief the PDO layout that IgH scanned, by sync manager.
        std::vector<Sync> syncs;
        /// @brief the object dictionary, keyed by index and sub-index. A download
        /// writes to it, and an upload of a missing object aborts.
        std::map<std::pair<uint16_t, uint8_t>, uint32_t> objects;
        /// @brief abort codes for objects that reject downloads. A code of 0 makes the
        /// download time out.
        std::map<std::pair<uint16_t, uint8_t>, uint32_t> download_aborts;
        /// @brief whether master_get_slave fails for the slave.
        bool unreadable = false;
        /// @brief whether slave_config_state fails for the slave.
        bool state_unreadable = false;
        /// @brief whether master_get_pdo fails for the slave.
        bool layout_unreadable = false;
        bool online = true;
        /// @brief the state the slave reaches after activation.
        uint8_t reaches = EC_AL_STATE_OP;
        /// @brief whether the slave sets its AL error flag when it reaches a state
        /// below OP.
        bool refuses = false;
        /// @brief how many state reads after activation the slave takes to reach its
        /// state.
        int reads_to_reach = 0;
        bool configured = false;
        bool settled = false;
        /// @brief the SDO downloads, in call order.
        std::vector<SDO> downloads;
        /// @brief the SDOs added to the config, in call order.
        std::vector<SDO> startup_sdos;
        /// @brief the PDOs that the config assigns to each sync manager.
        std::map<uint8_t, std::vector<PDO>> assigned;
        /// @brief the index and sub-index of each registered entry, in call order.
        std::vector<std::pair<uint16_t, uint8_t>> registered;

        /// @brief returns the PDOs of every sync manager after configuration.
        [[nodiscard]] std::vector<PDO> layout() const {
            std::vector<PDO> pdos;
            for (size_t sm = 0; sm < this->syncs.size(); sm++)
                if (!this->assigned.contains(static_cast<uint8_t>(sm)))
                    pdos.insert(
                        pdos.end(),
                        this->syncs[sm].pdos.begin(),
                        this->syncs[sm].pdos.end()
                    );
            for (const auto &[sm, assigned]: this->assigned)
                pdos.insert(pdos.end(), assigned.begin(), assigned.end());
            return pdos;
        }

        /// @brief returns the scanned mapping of pdo_index.
        [[nodiscard]] std::vector<Entry>
        default_mapping(const uint16_t pdo_index) const {
            for (const auto &sync: this->syncs)
                for (const auto &pdo: sync.pdos)
                    if (pdo.index == pdo_index) return pdo.entries;
            return {};
        }
    };

    mutable std::vector<Slave> slaves;
    mutable Domain outputs;
    mutable Domain inputs;
    x::errors::Error check_device_err = x::errors::NIL;
    bool request_master_fails = false;
    int master_info_result = 0;
    bool output_domain_fails = false;
    bool input_domain_fails = false;
    int activate_result = 0;
    bool slave_config_fails = false;
    bool config_sdo_fails = false;
    /// @brief the result of master_receive and master_send.
    int receive_result = 0;
    int send_result = 0;
    mutable bool released = false;
    mutable bool deactivated = false;

    MockAPI(): API(MockTag{}) {}

    /// @brief adds a slave after the last one, in PRE-OP, and returns it.
    Slave &add_slave(
        const std::string &name = "TestSlave",
        const uint32_t vendor_id = 0xBEEF,
        const uint32_t product_code = 0xCAFE
    ) {
        auto &slave = this->slaves.emplace_back();
        slave.info.position = static_cast<uint16_t>(this->slaves.size() - 1);
        slave.info.vendor_id = vendor_id;
        slave.info.product_code = product_code;
        slave.info.al_state = EC_AL_STATE_PREOP;
        std::strncpy(slave.info.name, name.c_str(), EC_MAX_STRING_LENGTH - 1);
        return slave;
    }

    [[nodiscard]] x::errors::Error check_device(unsigned int) const override {
        return this->check_device_err;
    }

    [[nodiscard]] ec_master_t *request_master(unsigned int) const override {
        return this->request_master_fails ? nullptr : this->handle;
    }

    void release_master(ec_master_t *) const override {
        this->released = true;
        this->domains_created = 0;
        this->activated = false;
        for (auto &slave: this->slaves) {
            slave.configured = false;
            slave.settled = false;
        }
    }

    int master_activate(ec_master_t *) const override {
        if (this->activate_result < 0) return this->activate_result;
        this->activated = true;
        this->outputs.data.resize(this->outputs.size);
        this->inputs.data.resize(this->inputs.size);
        return 0;
    }

    int master_deactivate(ec_master_t *) const override {
        this->deactivated = true;
        return 0;
    }

    ec_domain_t *master_create_domain(ec_master_t *) const override {
        if (++this->domains_created == 1)
            return this->output_domain_fails ? nullptr : this->output_domain;
        return this->input_domain_fails ? nullptr : this->input_domain;
    }

    size_t domain_size(const ec_domain_t *domain) const override {
        return this->domain(domain).size;
    }

    uint8_t *domain_data(const ec_domain_t *domain) const override {
        auto &d = this->domain(domain);
        return d.data.empty() ? nullptr : d.data.data();
    }

    int domain_process(ec_domain_t *) const override { return 0; }

    int domain_queue(ec_domain_t *) const override { return 0; }

    int
    domain_state(const ec_domain_t *domain, ec_domain_state_t *state) const override {
        *state = this->domain(domain).state;
        return 0;
    }

    int master_send(ec_master_t *) const override { return this->send_result; }

    int master_receive(ec_master_t *) const override { return this->receive_result; }

    int master(ec_master_t *, ec_master_info_t *info) const override {
        if (this->master_info_result < 0) return this->master_info_result;
        info->slave_count = static_cast<unsigned int>(this->slaves.size());
        return 0;
    }

    int master_get_slave(
        ec_master_t *,
        uint16_t position,
        ec_slave_info_t *info
    ) const override {
        if (position >= this->slaves.size() || this->slaves[position].unreadable)
            return -EIO;
        const auto &slave = this->slaves[position];
        *info = slave.info;
        info->sync_count = static_cast<uint8_t>(slave.syncs.size());
        return 0;
    }

    ec_slave_config_t *master_slave_config(
        ec_master_t *,
        uint16_t,
        uint16_t position,
        uint32_t,
        uint32_t
    ) const override {
        if (this->slave_config_fails || position >= this->slaves.size()) return nullptr;
        this->slaves[position].configured = true;
        return reinterpret_cast<ec_slave_config_t *>(CONFIG_BASE + position);
    }

    int slave_config_state(
        const ec_slave_config_t *sc,
        ec_slave_config_state_t *state
    ) const override {
        auto &slave = this->slave(sc);
        if (slave.state_unreadable) return -EIO;
        if (this->activated && !slave.settled) {
            if (slave.reads_to_reach > 0)
                slave.reads_to_reach--;
            else {
                slave.settled = true;
                slave.info.al_state = slave.reaches;
                slave.info.error_flag = slave.reaches != EC_AL_STATE_OP &&
                                        slave.refuses;
            }
        }
        *state = {};
        state->online = slave.online ? 1 : 0;
        state->operational = slave.online && slave.info.al_state == EC_AL_STATE_OP;
        state->al_state = slave.info.al_state;
        return 0;
    }

    int slave_config_reg_pdo_entry(
        ec_slave_config_t *sc,
        uint16_t index,
        uint8_t sub_index,
        ec_domain_t *domain,
        unsigned int *bit_position
    ) const override {
        auto &slave = this->slave(sc);
        for (const auto &pdo: slave.layout())
            for (const auto &entry: pdo.entries) {
                if (entry.index != index || entry.sub_index != sub_index) continue;
                slave.registered.emplace_back(index, sub_index);
                auto &d = this->domain(domain);
                const auto offset = d.size;
                d.size += (entry.bit_length + 7) / 8;
                *bit_position = 0;
                return static_cast<int>(offset);
            }
        return -ENOENT;
    }

    int slave_config_sdo(
        ec_slave_config_t *sc,
        uint16_t index,
        uint8_t sub_index,
        const uint8_t *data,
        size_t size
    ) const override {
        if (this->config_sdo_fails) return -ENOMEM;
        this->slave(sc).startup_sdos.push_back(
            {index, sub_index, std::vector(data, data + size)}
        );
        return 0;
    }

    /// @brief configures the PDOs as ecrt_slave_config_pdos does: a sync manager with
    /// no PDOs keeps its assignment, and a PDO with no entries keeps its mapping.
    int slave_config_pdos(
        ec_slave_config_t *sc,
        unsigned int n_syncs,
        const ec_sync_info_t syncs[]
    ) const override {
        auto &slave = this->slave(sc);
        for (const auto &sync: std::span(syncs, n_syncs)) {
            if (sync.n_pdos == 0) continue;
            auto &pdos = slave.assigned[sync.index];
            pdos.clear();
            for (const auto &info: std::span(sync.pdos, sync.n_pdos)) {
                auto &pdo = pdos.emplace_back(
                    PDO{info.index, slave.default_mapping(info.index)}
                );
                if (info.n_entries == 0) continue;
                pdo.entries.clear();
                for (const auto &e: std::span(info.entries, info.n_entries))
                    pdo.entries.push_back({e.index, e.subindex, e.bit_length});
            }
        }
        return 0;
    }

    int slave_config_pdo_assign_clear(
        ec_slave_config_t *sc,
        uint8_t sync_index
    ) const override {
        this->slave(sc).assigned[sync_index].clear();
        return 0;
    }

    int slave_config_pdo_mapping_clear(
        ec_slave_config_t *sc,
        uint16_t pdo_index
    ) const override {
        for (auto &[sm, pdos]: this->slave(sc).assigned)
            for (auto &pdo: pdos)
                if (pdo.index == pdo_index) pdo.entries.clear();
        return 0;
    }

    int master_sdo_download(
        ec_master_t *,
        uint16_t position,
        uint16_t index,
        uint8_t sub_index,
        const uint8_t *data,
        size_t size,
        uint32_t *abort_code
    ) const override {
        auto &slave = this->slaves.at(position);
        slave.downloads.push_back({index, sub_index, std::vector(data, data + size)});
        if (const auto it = slave.download_aborts.find({index, sub_index});
            it != slave.download_aborts.end()) {
            *abort_code = it->second;
            return -EIO;
        }
        uint32_t value = 0;
        for (size_t i = 0; i < size && i < sizeof(value); i++)
            value |= static_cast<uint32_t>(data[i]) << (8 * i);
        slave.objects[{index, sub_index}] = value;
        return 0;
    }

    int master_sdo_upload(
        ec_master_t *,
        uint16_t position,
        uint16_t index,
        uint8_t sub_index,
        uint8_t *target,
        size_t target_size,
        size_t *result_size,
        uint32_t *abort_code
    ) const override {
        const auto &objects = this->slaves.at(position).objects;
        const auto it = objects.find({index, sub_index});
        if (it == objects.end()) {
            *abort_code = 0x06020000;
            return -EIO;
        }
        const size_t size = std::min(target_size, sizeof(it->second));
        for (size_t i = 0; i < size; i++)
            target[i] = static_cast<uint8_t>(it->second >> (8 * i));
        *result_size = size;
        return 0;
    }

    int master_get_sync_manager(
        ec_master_t *,
        uint16_t position,
        uint8_t sync_index,
        ec_sync_info_t *sync
    ) const override {
        const auto &syncs = this->slaves.at(position).syncs;
        if (sync_index >= syncs.size()) return -ENOENT;
        sync->index = sync_index;
        sync->dir = syncs[sync_index].dir;
        sync->n_pdos = static_cast<unsigned int>(syncs[sync_index].pdos.size());
        return 0;
    }

    int master_get_pdo(
        ec_master_t *,
        uint16_t position,
        uint8_t sync_index,
        uint16_t pos,
        ec_pdo_info_t *pdo
    ) const override {
        const auto &slave = this->slaves.at(position);
        if (slave.layout_unreadable) return -EIO;
        const auto &syncs = slave.syncs;
        if (sync_index >= syncs.size() || pos >= syncs[sync_index].pdos.size())
            return -ENOENT;
        const auto &fake = syncs[sync_index].pdos[pos];
        pdo->index = fake.index;
        pdo->n_entries = static_cast<unsigned int>(fake.entries.size());
        return 0;
    }

    int master_get_pdo_entry(
        ec_master_t *,
        uint16_t position,
        uint8_t sync_index,
        uint16_t pdo_pos,
        uint16_t entry_pos,
        ec_pdo_entry_info_t *entry
    ) const override {
        const auto &syncs = this->slaves.at(position).syncs;
        if (sync_index >= syncs.size() || pdo_pos >= syncs[sync_index].pdos.size() ||
            entry_pos >= syncs[sync_index].pdos[pdo_pos].entries.size())
            return -ENOENT;
        const auto &fake = syncs[sync_index].pdos[pdo_pos].entries[entry_pos];
        entry->index = fake.index;
        entry->subindex = fake.sub_index;
        entry->bit_length = fake.bit_length;
        return 0;
    }

private:
    Domain &domain(const ec_domain_t *domain) const {
        return domain == this->output_domain ? this->outputs : this->inputs;
    }

    Slave &slave(const ec_slave_config_t *sc) const {
        return this->slaves.at(reinterpret_cast<uintptr_t>(sc) - CONFIG_BASE);
    }
};
}
