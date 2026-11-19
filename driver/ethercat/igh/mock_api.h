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
#include <cstddef>
#include <cstdint>
#include <cstring>
#include <map>
#include <optional>
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
    static constexpr uintptr_t DOMAIN_BASE = 0x10000;
    ec_master_t *const handle = reinterpret_cast<ec_master_t *>(0x1);
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

    /// @brief the DC sync of a slave config, as ecrt_slave_config_dc takes it.
    struct DC {
        uint16_t assign_activate;
        uint32_t sync0_cycle;
        int32_t sync0_shift;
        uint32_t sync1_cycle;
        int32_t sync1_shift;

        bool operator==(const DC &) const = default;
    };

    /// @brief a process data domain.
    struct Domain {
        /// @brief the size in bytes, which grows with each registered entry.
        size_t size = 0;
        std::vector<uint8_t> data;
        ec_domain_state_t state{.working_counter = 0, .wc_state = EC_WC_COMPLETE};
    };

    /// @brief an object of the CoE dictionary that IgH read, with its entries by
    /// sub-index.
    struct DictionaryObject {
        ObjectInfo info;
        std::map<uint8_t, EntryInfo> entries;
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
        /// @brief objects of 8 bytes. An upload with less room fails with ENOBUFS, as
        /// in IgH.
        std::map<std::pair<uint16_t, uint8_t>, uint64_t> wide_objects;
        /// @brief the error code that an upload of each object fails with, with no
        /// abort code.
        std::map<std::pair<uint16_t, uint8_t>, int> upload_errors;
        /// @brief abort codes for objects that reject downloads. A code of 0 makes the
        /// download time out.
        std::map<std::pair<uint16_t, uint8_t>, uint32_t> download_aborts;
        /// @brief how many more reads of a slave in INIT find it in INIT before IgH
        /// moves it to PRE-OP, or -1 to keep it in INIT.
        int reads_to_pre_op = -1;
        /// @brief whether master_get_slave fails for the slave.
        bool unreadable = false;
        /// @brief whether slave_config_state fails for the slave.
        bool state_unreadable = false;
        /// @brief whether master_get_pdo fails for the slave.
        bool layout_unreadable = false;
        /// @brief the CoE dictionary that IgH read from the slave, in list order.
        std::vector<DictionaryObject> dictionary;
        /// @brief how many parts of dictionary IgH has read while it reads it, or
        /// nullopt once it has read all of it. Part 1 lists the objects, part 1 + k
        /// names object k and holds the entries of object k - 1, and the part after the
        /// final name holds the final entries. Each read of the slave reads one more
        /// part, and an SDO transfer fails while IgH reads.
        std::optional<size_t> dictionary_read;
        /// @brief whether IgH stopped reading the dictionary at dictionary_read.
        bool dictionary_stalled = false;
        /// @brief how many entries entry_info looked up.
        mutable size_t entry_reads = 0;
        /// @brief the SII from word 0.
        std::vector<uint16_t> sii;
        /// @brief what sii_read returns for the slave, when not 0.
        int sii_result = 0;
        /// @brief how many sii_read calls ran past the end of sii, which IgH logs as a
        /// kernel error.
        mutable size_t sii_overreads = 0;
        /// @brief the DC sync that the config sets up, or nullopt for none.
        std::optional<DC> dc;
        bool online = true;
        /// @brief the state the slave reaches after activation.
        uint8_t reaches = EC_AL_STATE_OP;
        /// @brief whether the slave sets its AL error flag when it reaches a state
        /// below OP.
        bool refuses = false;
        /// @brief how many state reads after activation the slave takes to reach its
        /// state.
        int reads_to_reach = 0;
        /// @brief whether a master reset makes the slave reach OP.
        bool recovers_on_reset = false;
        /// @brief how many state reads after a master reset the slave takes to reach
        /// its state again, as IgH configures it again.
        int reads_to_reconfigure = 0;
        /// @brief how many state reads remain before a master reset takes effect, or -1
        /// with no reset pending.
        int reads_to_reset = -1;
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
        /// @brief the index of the domain that holds the entries of the slave.
        std::optional<size_t> domain;

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
    /// @brief the domains, in creation order.
    mutable std::vector<Domain> domains;
    x::errors::Error check_device_err = x::errors::NIL;
    bool open_master_fails = false;
    /// @brief the result of master_reserve, and whether the master holds a reservation.
    /// IgH refuses to configure or activate a master with none.
    int reserve_result = 0;
    mutable bool reserved = false;
    int master_info_result = 0;
    /// @brief whether the network port of the master has a link.
    bool link_up = true;
    /// @brief how many more master info reads find IgH scanning the bus.
    mutable int scan_reads = 0;
    /// @brief whether the master scans the bus again, as after a slave loses power.
    /// During the scan, every slave config reads as offline.
    bool scanning = false;
    bool domain_fails = false;
    int activate_result = 0;
    bool slave_config_fails = false;
    bool config_sdo_fails = false;
    /// @brief the results of master_receive, master_send, domain_process,
    /// domain_queue, and domain_state.
    int receive_result = 0;
    int send_result = 0;
    int domain_process_result = 0;
    int domain_queue_result = 0;
    int domain_state_result = 0;
    /// @brief the domain queues, in call order.
    mutable std::vector<size_t> queued;
    /// @brief the result of master_reset, and how many state reads of a slave pass
    /// before a reset takes effect, as IgH resets in the background.
    int reset_result = 0;
    int reset_lag = 0;
    /// @brief how many times the master asked for a reset, failed or not.
    mutable int resets = 0;
    mutable bool released = false;
    mutable bool deactivated = false;
    /// @brief the times that master_application_time got, in call order.
    mutable std::vector<uint64_t> app_times;
    /// @brief the times that master_sync_reference_clock_to got, in call order.
    mutable std::vector<uint64_t> reference_syncs;
    /// @brief how many times master_sync_slave_clocks ran.
    mutable int slave_syncs = 0;
    /// @brief what master_reference_clock_time reads, and its result.
    uint32_t reference_clock_time = 0;
    int reference_clock_result = 0;
    /// @brief the result of master_sync_reference_clock_to and
    /// master_sync_slave_clocks.
    int clock_sync_result = 0;
    /// @brief the results of slave_config_dc and master_application_time.
    int dc_config_result = 0;
    int app_time_result = 0;

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

    /// @brief returns the domain that holds the entries of slave.
    Domain &domain_of(const Slave &slave) const {
        return this->domains.at(slave.domain.value());
    }

    [[nodiscard]] x::errors::Error check_device(unsigned int) const override {
        return this->check_device_err;
    }

    [[nodiscard]] ec_master_t *open_master(unsigned int) const override {
        return this->open_master_fails ? nullptr : this->handle;
    }

    int master_reserve(ec_master_t *) const override {
        if (this->reserve_result < 0) return this->reserve_result;
        this->reserved = true;
        return 0;
    }

    void release_master(ec_master_t *) const override {
        this->released = true;
        this->reserved = false;
        this->domains.clear();
        this->activated = false;
        for (auto &slave: this->slaves) {
            slave.configured = false;
            slave.settled = false;
            slave.dc.reset();
            slave.domain.reset();
        }
    }

    int master_activate(ec_master_t *) const override {
        if (!this->reserved) return -EPERM;
        if (this->activate_result < 0) return this->activate_result;
        this->activated = true;
        for (auto &domain: this->domains)
            domain.data.resize(domain.size);
        return 0;
    }

    int master_deactivate(ec_master_t *) const override {
        this->deactivated = true;
        return 0;
    }

    int master_reset(ec_master_t *) const override {
        this->resets++;
        if (this->reset_result < 0) return this->reset_result;
        for (auto &slave: this->slaves)
            if (slave.configured) slave.reads_to_reset = this->reset_lag;
        return 0;
    }

    ec_domain_t *master_create_domain(ec_master_t *) const override {
        if (!this->reserved || this->domain_fails) return nullptr;
        this->domains.emplace_back();
        return reinterpret_cast<ec_domain_t *>(DOMAIN_BASE + this->domains.size() - 1);
    }

    size_t domain_size(const ec_domain_t *domain) const override {
        return this->domain(domain).size;
    }

    uint8_t *domain_data(const ec_domain_t *domain) const override {
        auto &d = this->domain(domain);
        return d.data.empty() ? nullptr : d.data.data();
    }

    int domain_process(ec_domain_t *) const override {
        return this->domain_process_result;
    }

    int domain_queue(ec_domain_t *domain) const override {
        this->queued.push_back(this->index(domain));
        return this->domain_queue_result;
    }

    int
    domain_state(const ec_domain_t *domain, ec_domain_state_t *state) const override {
        *state = this->domain(domain).state;
        return this->domain_state_result;
    }

    int master_send(ec_master_t *) const override { return this->send_result; }

    int master_receive(ec_master_t *) const override { return this->receive_result; }

    int master(ec_master_t *, ec_master_info_t *info) const override {
        if (this->master_info_result < 0) return this->master_info_result;
        info->slave_count = static_cast<unsigned int>(this->slaves.size());
        info->link_up = this->link_up ? 1 : 0;
        info->scan_busy = this->scanning || this->scan_reads > 0 ? 1 : 0;
        if (this->scan_reads > 0) this->scan_reads--;
        return 0;
    }

    int master_get_slave(
        ec_master_t *,
        uint16_t position,
        ec_slave_info_t *info
    ) const override {
        if (position >= this->slaves.size() || this->slaves[position].unreadable)
            return -EIO;
        auto &slave = this->slaves[position];
        if (slave.reads_to_pre_op == 0) slave.info.al_state = EC_AL_STATE_PREOP;
        if (slave.reads_to_pre_op >= 0) slave.reads_to_pre_op--;
        *info = slave.info;
        info->sync_count = static_cast<uint8_t>(slave.syncs.size());
        info->sdo_count = static_cast<uint16_t>(read_parts(slave).listed);
        if (slave.dictionary_read && !slave.dictionary_stalled &&
            ++*slave.dictionary_read == slave.dictionary.size() + 2)
            slave.dictionary_read.reset();
        return 0;
    }

    ec_slave_config_t *master_slave_config(
        ec_master_t *,
        uint16_t,
        uint16_t position,
        uint32_t,
        uint32_t
    ) const override {
        if (!this->reserved || this->slave_config_fails ||
            position >= this->slaves.size())
            return nullptr;
        this->slaves[position].configured = true;
        return reinterpret_cast<ec_slave_config_t *>(CONFIG_BASE + position);
    }

    int slave_config_state(
        const ec_slave_config_t *sc,
        ec_slave_config_state_t *state
    ) const override {
        auto &slave = this->slave(sc);
        if (slave.state_unreadable) return -EIO;
        if (this->scanning) {
            *state = {};
            return 0;
        }
        if (slave.reads_to_reset > 0)
            slave.reads_to_reset--;
        else if (slave.reads_to_reset == 0) {
            slave.reads_to_reset = -1;
            // IgH configures a faulted slave again from INIT.
            if (slave.info.error_flag != 0) slave.info.al_state = EC_AL_STATE_INIT;
            slave.info.error_flag = 0;
            slave.settled = false;
            slave.reads_to_reach = slave.reads_to_reconfigure;
            if (slave.recovers_on_reset) {
                slave.reaches = EC_AL_STATE_OP;
                slave.refuses = false;
            }
        }
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
                slave.domain = this->index(domain);
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
        if (reading(slave)) return -EIO;
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
        const auto &slave = this->slaves.at(position);
        if (reading(slave)) return -EIO;
        const std::pair key{index, sub_index};
        if (const auto err = slave.upload_errors.find(key);
            err != slave.upload_errors.end())
            return err->second;
        if (const auto wide = slave.wide_objects.find(key);
            wide != slave.wide_objects.end()) {
            if (target_size < sizeof(wide->second)) return -ENOBUFS;
            for (size_t i = 0; i < sizeof(wide->second); i++)
                target[i] = static_cast<uint8_t>(wide->second >> (8 * i));
            *result_size = sizeof(wide->second);
            return 0;
        }
        const auto it = slave.objects.find(key);
        if (it == slave.objects.end()) {
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

    int slave_config_dc(
        ec_slave_config_t *sc,
        const uint16_t assign_activate,
        const uint32_t sync0_cycle,
        const int32_t sync0_shift,
        const uint32_t sync1_cycle,
        const int32_t sync1_shift
    ) const override {
        this->slave(sc).dc = DC{
            assign_activate,
            sync0_cycle,
            sync0_shift,
            sync1_cycle,
            sync1_shift
        };
        return this->dc_config_result;
    }

    int master_application_time(ec_master_t *, const uint64_t app_time) const override {
        this->app_times.push_back(app_time);
        return this->app_time_result;
    }

    int master_sync_reference_clock_to(
        ec_master_t *,
        const uint64_t sync_time
    ) const override {
        this->reference_syncs.push_back(sync_time);
        return this->clock_sync_result;
    }

    int master_sync_slave_clocks(ec_master_t *) const override {
        this->slave_syncs++;
        return this->clock_sync_result;
    }

    int
    master_reference_clock_time(const ec_master_t *, uint32_t *time) const override {
        *time = this->reference_clock_time;
        return this->reference_clock_result;
    }

    int sii_read(
        unsigned int,
        const uint16_t position,
        const uint16_t offset,
        const std::span<uint16_t> words
    ) const override {
        const auto &slave = this->slaves.at(position);
        if (slave.sii_result != 0) return slave.sii_result;
        if (words.empty()) return -EINVAL;
        if (offset + words.size() > slave.sii.size()) {
            slave.sii_overreads++;
            return -EINVAL;
        }
        std::copy_n(slave.sii.begin() + offset, words.size(), words.begin());
        return 0;
    }

    int object_info(
        unsigned int,
        const uint16_t position,
        const uint16_t list_position,
        ObjectInfo &info
    ) const override {
        const auto &slave = this->slaves.at(position);
        const auto parts = read_parts(slave);
        if (list_position >= parts.listed) return -EINVAL;
        info = slave.dictionary[list_position].info;
        if (list_position >= parts.named) info = {.index = info.index};
        return 0;
    }

    int entry_info(
        unsigned int,
        const uint16_t position,
        const uint16_t index,
        const uint8_t sub_index,
        EntryInfo &info
    ) const override {
        const auto &slave = this->slaves.at(position);
        slave.entry_reads++;
        const auto object = std::ranges::find_if(
            slave.dictionary,
            [index](const DictionaryObject &o) { return o.info.index == index; }
        );
        if (object == slave.dictionary.end() || std::cmp_greater_equal(
                                                    object - slave.dictionary.begin(),
                                                    read_parts(slave).described
                                                ))
            return -EINVAL;
        const auto entry = object->entries.find(sub_index);
        if (entry == object->entries.end()) return -EINVAL;
        info = entry->second;
        return 0;
    }

private:
    /// @brief how many objects of a dictionary IgH has listed, named, and read the
    /// entries of.
    struct Parts {
        size_t listed;
        size_t named;
        size_t described;
    };

    /// @brief returns how much of the dictionary of slave IgH has read.
    static Parts read_parts(const Slave &slave) {
        const size_t n = slave.dictionary.size();
        if (!slave.dictionary_read) return {n, n, n};
        const size_t read = *slave.dictionary_read;
        return {
            .listed = read >= 1 ? n : 0,
            .named = read >= 1 ? std::min(read - 1, n) : 0,
            .described = read >= 2 ? std::min(read - 2, n) : 0,
        };
    }

    /// @brief returns whether IgH reads the dictionary of slave, which makes an SDO
    /// transfer get a reply meant for IgH.
    static bool reading(const Slave &slave) {
        return slave.dictionary_read && !slave.dictionary_stalled;
    }

    static size_t index(const ec_domain_t *domain) {
        return reinterpret_cast<uintptr_t>(domain) - DOMAIN_BASE;
    }

    Domain &domain(const ec_domain_t *domain) const {
        return this->domains.at(index(domain));
    }

    Slave &slave(const ec_slave_config_t *sc) const {
        return this->slaves.at(reinterpret_cast<uintptr_t>(sc) - CONFIG_BASE);
    }
};
}
