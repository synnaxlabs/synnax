// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "glog/logging.h"
#include <sys/stat.h>

#include "driver/ethercat/igh/master.h"
#include "driver/ethercat/pdo_types.h"

namespace ethercat::igh {

bool igh_available() {
    struct stat st;
    return stat("/dev/EtherCAT0", &st) == 0;
}

Domain::Domain(ec_domain_t *domain, Master *master):
    domain(domain),
    master(master),
    domain_data(nullptr),
    input_sz(0),
    output_sz(0),
    activated(false) {}

std::pair<size_t, xerrors::Error> Domain::register_pdo(const PDOEntry &entry) {
    if (this->activated)
        return {
            0,
            xerrors::Error(PDO_MAPPING_ERROR, "cannot register PDO after activation")
        };

    ec_slave_config_t *sc = this->master->get_or_create_slave_config(
        entry.slave_position
    );
    if (!sc)
        return {
            0,
            xerrors::Error(PDO_MAPPING_ERROR, "failed to get slave configuration")
        };

    unsigned int offset = 0;
    int result = ecrt_slave_config_reg_pdo_entry(
        sc,
        entry.index,
        entry.subindex,
        this->domain,
        nullptr
    );

    if (result < 0)
        return {
            0,
            xerrors::Error(PDO_MAPPING_ERROR, "ecrt_slave_config_reg_pdo_entry failed")
        };

    offset = static_cast<unsigned int>(result);
    this->pdo_offsets.push_back(offset);
    this->registered_entries.push_back(entry);

    const size_t byte_size = entry.byte_length();
    if (entry.is_input)
        this->input_sz += byte_size;
    else
        this->output_sz += byte_size;

    return {offset, xerrors::NIL};
}

uint8_t *Domain::data() {
    return this->domain_data;
}

size_t Domain::size() const {
    return this->input_sz + this->output_sz;
}

size_t Domain::input_size() const {
    return this->input_sz;
}

size_t Domain::output_size() const {
    return this->output_sz;
}

void Domain::set_activated(uint8_t *data_ptr) {
    this->domain_data = data_ptr;
    this->activated = true;
}

SlaveDataOffsets Domain::get_slave_offsets(const uint16_t position) const {
    SlaveDataOffsets offsets;
    bool found_input = false;
    bool found_output = false;

    for (size_t i = 0; i < this->registered_entries.size(); ++i) {
        const auto &entry = this->registered_entries[i];
        if (entry.slave_position != position) continue;

        if (entry.is_input && !found_input) {
            offsets.input_offset = this->pdo_offsets[i];
            offsets.input_size = entry.byte_length();
            found_input = true;
        } else if (entry.is_input) {
            offsets.input_size += entry.byte_length();
        }

        if (!entry.is_input && !found_output) {
            offsets.output_offset = this->pdo_offsets[i];
            offsets.output_size = entry.byte_length();
            found_output = true;
        } else if (!entry.is_input) {
            offsets.output_size += entry.byte_length();
        }
    }

    return offsets;
}

Master::Master(const unsigned int master_index):
    master_index(master_index),
    ec_master(nullptr),
    initialized(false),
    activated(false),
    domain_state{} {}

Master::~Master() {
    if (this->activated) this->deactivate();
    if (this->ec_master) ecrt_release_master(this->ec_master);
}

xerrors::Error Master::initialize() {
    if (this->initialized) return xerrors::NIL;

    this->ec_master = ecrt_request_master(this->master_index);
    if (!this->ec_master)
        return xerrors::Error(
            MASTER_INIT_ERROR,
            "IgH master not available - is kernel module loaded?"
        );

    ec_master_info_t master_info;
    if (ecrt_master(this->ec_master, &master_info) < 0) {
        ecrt_release_master(this->ec_master);
        this->ec_master = nullptr;
        return xerrors::Error(MASTER_INIT_ERROR, "failed to get master info");
    }

    this->cached_slaves.clear();
    this->cached_slaves.reserve(master_info.slave_count);

    for (unsigned int i = 0; i < master_info.slave_count; ++i) {
        ec_slave_info_t slave_info;
        if (ecrt_master_get_slave(this->ec_master, i, &slave_info) == 0) {
            SlaveInfo info(
                static_cast<uint16_t>(i),
                slave_info.vendor_id,
                slave_info.product_code,
                slave_info.revision_number,
                slave_info.serial_number,
                slave_info.name,
                SlaveState::INIT
            );
            this->discover_slave_pdos(info);
            this->cached_slaves.push_back(std::move(info));
        }
    }

    this->initialized = true;
    return xerrors::NIL;
}

std::unique_ptr<ethercat::Domain> Master::create_domain() {
    if (!this->initialized || this->activated) return nullptr;

    ec_domain_t *dom = ecrt_master_create_domain(this->ec_master);
    if (!dom) return nullptr;

    this->domain = std::make_unique<Domain>(dom, this);
    return nullptr;
}

xerrors::Error Master::activate() {
    if (!this->initialized) return xerrors::Error(ACTIVATION_ERROR, "not initialized");
    if (this->activated) return xerrors::NIL;
    if (!this->domain) return xerrors::Error(ACTIVATION_ERROR, "no domain created");

    if (ecrt_master_activate(this->ec_master) < 0)
        return xerrors::Error(ACTIVATION_ERROR, "ecrt_master_activate failed");

    uint8_t *data_ptr = ecrt_domain_data(this->domain->native_handle());
    if (!data_ptr) {
        ecrt_master_deactivate(this->ec_master);
        return xerrors::Error(ACTIVATION_ERROR, "failed to get domain data pointer");
    }

    this->domain->set_activated(data_ptr);
    this->activated = true;
    return xerrors::NIL;
}

void Master::deactivate() {
    if (!this->activated) return;

    ecrt_master_deactivate(this->ec_master);
    this->activated = false;
    this->slave_configs.clear();
}

xerrors::Error Master::receive() {
    if (!this->activated) return xerrors::Error(CYCLIC_ERROR, "not activated");
    ecrt_master_receive(this->ec_master);
    return xerrors::NIL;
}

xerrors::Error Master::process(ethercat::Domain &d) {
    auto *igh_domain = dynamic_cast<Domain *>(&d);
    if (!igh_domain) return xerrors::Error(CYCLIC_ERROR, "invalid domain type");

    ecrt_domain_process(igh_domain->native_handle());

    ecrt_domain_state(igh_domain->native_handle(), &this->domain_state);

    return xerrors::NIL;
}

xerrors::Error Master::queue(ethercat::Domain &d) {
    auto *igh_domain = dynamic_cast<Domain *>(&d);
    if (!igh_domain) return xerrors::Error(CYCLIC_ERROR, "invalid domain type");

    ecrt_domain_queue(igh_domain->native_handle());
    return xerrors::NIL;
}

xerrors::Error Master::send() {
    if (!this->activated) return xerrors::Error(CYCLIC_ERROR, "not activated");
    ecrt_master_send(this->ec_master);
    return xerrors::NIL;
}

std::vector<SlaveInfo> Master::slaves() const {
    std::lock_guard lock(this->mutex);
    return this->cached_slaves;
}

SlaveState Master::slave_state(const uint16_t position) const {
    std::lock_guard lock(this->mutex);

    if (position >= this->cached_slaves.size()) return SlaveState::UNKNOWN;

    ec_slave_config_state_t state;
    auto it = this->slave_configs.find(position);
    if (it == this->slave_configs.end()) return SlaveState::UNKNOWN;

    ecrt_slave_config_state(it->second, &state);
    return convert_state(state.al_state);
}

bool Master::all_slaves_operational() const {
    std::lock_guard lock(this->mutex);

    if (!this->activated) return false;

    for (const auto &[pos, sc]: this->slave_configs) {
        ec_slave_config_state_t state;
        ecrt_slave_config_state(sc, &state);
        if (state.al_state != 0x08) return false;
    }
    return true;
}

std::string Master::interface_name() const {
    return "igh:" + std::to_string(this->master_index);
}

SlaveDataOffsets Master::slave_data_offsets(const uint16_t position) const {
    if (!this->activated || !this->domain) return SlaveDataOffsets{};
    return this->domain->get_slave_offsets(position);
}

ethercat::Domain *Master::active_domain() const {
    return this->activated ? this->domain.get() : nullptr;
}

ec_slave_config_t *Master::get_or_create_slave_config(const uint16_t position) {
    std::lock_guard lock(this->mutex);

    auto it = this->slave_configs.find(position);
    if (it != this->slave_configs.end()) return it->second;

    if (position >= this->cached_slaves.size()) return nullptr;

    const auto &slave = this->cached_slaves[position];
    ec_slave_config_t *sc = ecrt_master_slave_config(
        this->ec_master,
        0,
        position,
        slave.vendor_id,
        slave.product_code
    );

    if (sc) this->slave_configs[position] = sc;
    return sc;
}

SlaveState Master::convert_state(const uint8_t igh_state) {
    switch (igh_state) {
        case 0x01:
            return SlaveState::INIT;
        case 0x02:
            return SlaveState::PRE_OP;
        case 0x03:
            return SlaveState::BOOT;
        case 0x04:
            return SlaveState::SAFE_OP;
        case 0x08:
            return SlaveState::OP;
        default:
            return SlaveState::UNKNOWN;
    }
}

void Master::discover_slave_pdos(SlaveInfo &slave) {
    ec_slave_info_t slave_info;
    if (ecrt_master_get_slave(this->ec_master, slave.position, &slave_info) != 0) {
        slave.pdo_discovery_error = "failed to get slave info";
        return;
    }

    for (uint8_t sm_idx = 0; sm_idx < slave_info.sync_count; ++sm_idx) {
        ec_sync_info_t sync_info{};
        if (ecrt_master_get_sync_manager(
                this->ec_master,
                slave.position,
                sm_idx,
                &sync_info
            ) != 0)
            continue;

        const bool is_input = (sync_info.dir == EC_DIR_INPUT);

        for (unsigned int pdo_pos = 0; pdo_pos < sync_info.n_pdos; ++pdo_pos) {
            ec_pdo_info_t pdo_info{};
            if (ecrt_master_get_pdo(
                    this->ec_master,
                    slave.position,
                    sm_idx,
                    static_cast<uint16_t>(pdo_pos),
                    &pdo_info
                ) != 0)
                continue;

            for (unsigned int entry_pos = 0; entry_pos < pdo_info.n_entries;
                 ++entry_pos) {
                ec_pdo_entry_info_t entry_info{};
                if (ecrt_master_get_pdo_entry(
                        this->ec_master,
                        slave.position,
                        sm_idx,
                        static_cast<uint16_t>(pdo_pos),
                        static_cast<uint16_t>(entry_pos),
                        &entry_info
                    ) != 0)
                    continue;

                if (entry_info.index == 0 && entry_info.subindex == 0) continue;

                const telem::DataType data_type =
                    infer_type_from_bit_length(entry_info.bit_length);
                const std::string name = generate_pdo_entry_name(
                    "",
                    entry_info.index,
                    entry_info.subindex,
                    is_input,
                    data_type
                );

                PDOEntryInfo entry(
                    pdo_info.index,
                    entry_info.index,
                    entry_info.subindex,
                    entry_info.bit_length,
                    is_input,
                    name,
                    data_type
                );

                if (is_input)
                    slave.input_pdos.push_back(entry);
                else
                    slave.output_pdos.push_back(entry);
            }
        }
    }

    slave.pdos_discovered = true;
    VLOG(1) << "Slave " << slave.position
            << " PDOs discovered via IgH: " << slave.input_pdos.size() << " inputs, "
            << slave.output_pdos.size() << " outputs";
}

}
