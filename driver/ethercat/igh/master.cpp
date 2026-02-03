// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <map>

#include "glog/logging.h"

#include "driver/ethercat/esi/known_devices.h"
#include "driver/ethercat/igh/master.h"
#include "driver/ethercat/telem/telem.h"

namespace ethercat::igh {
Master::Master(std::shared_ptr<API> api, const unsigned int master_index):
    api(std::move(api)),
    master_index(master_index),
    ec_master(nullptr),
    input_domain(nullptr),
    output_domain(nullptr),
    input_domain_data(nullptr),
    output_domain_data(nullptr),
    input_sz(0),
    output_sz(0),
    initialized(false),
    activated(false),
    input_domain_state{},
    output_domain_state{} {}

Master::~Master() {
    this->deactivate();
}

xerrors::Error Master::initialize() {
    if (this->initialized) return xerrors::NIL;

    this->ec_master = this->api->request_master(this->master_index);
    if (!this->ec_master)
        return xerrors::Error(
            MASTER_INIT_ERROR,
            "IgH master not available - is kernel module loaded?"
        );

    ec_master_info_t master_info;
    if (this->api->master(this->ec_master, &master_info) < 0) {
        this->api->release_master(this->ec_master);
        this->ec_master = nullptr;
        return xerrors::Error(MASTER_INIT_ERROR, "failed to get master info");
    }

    this->cached_slaves.clear();
    this->cached_slaves.reserve(master_info.slave_count);

    for (unsigned int i = 0; i < master_info.slave_count; ++i) {
        ec_slave_info_t slave_info;
        if (this->api->master_get_slave(this->ec_master, i, &slave_info) == 0) {
            slave::Properties info(
                static_cast<uint16_t>(i),
                slave_info.vendor_id,
                slave_info.product_code,
                slave_info.revision_number,
                slave_info.serial_number,
                slave_info.name,
                slave::State::INIT
            );
            this->discover_slave_pdos(info);
            this->cached_slaves.push_back(std::move(info));
        }
    }

    this->output_domain = this->api->master_create_domain(this->ec_master);
    if (!this->output_domain) {
        this->api->release_master(this->ec_master);
        this->ec_master = nullptr;
        return xerrors::Error(MASTER_INIT_ERROR, "failed to create output domain");
    }

    this->input_domain = this->api->master_create_domain(this->ec_master);
    if (!this->input_domain) {
        this->api->release_master(this->ec_master);
        this->ec_master = nullptr;
        return xerrors::Error(MASTER_INIT_ERROR, "failed to create input domain");
    }

    this->initialized = true;
    VLOG(1) << "[ethercat.igh] master " << this->master_index << " initialized with "
            << this->cached_slaves.size() << " slaves";
    return xerrors::NIL;
}

xerrors::Error Master::register_pdos(const std::vector<pdo::Entry> &entries) {
    for (const auto &entry: entries) {
        auto [offset, err] = this->register_pdo(entry);
        if (err) return err;
    }
    return xerrors::NIL;
}

void Master::set_slave_enabled(const uint16_t position, const bool enabled) {
    std::lock_guard lock(this->mu);
    if (!enabled)
        this->disabled_slaves.insert(position);
    else
        this->disabled_slaves.erase(position);
}

xerrors::Error Master::activate() {
    if (!this->initialized) return xerrors::Error(ACTIVATION_ERROR, "not initialized");
    if (this->activated) return xerrors::NIL;
    if (!this->output_domain || !this->input_domain)
        return xerrors::Error(ACTIVATION_ERROR, "domains not created");

    const size_t output_domain_size = this->api->domain_size(this->output_domain);
    const size_t input_domain_size = this->api->domain_size(this->input_domain);

    VLOG(1) << "[ethercat.igh] activating master " << this->master_index << " with "
            << this->slave_configs.size() << " configured slaves"
            << ", calculated: input_sz=" << this->input_sz
            << ", output_sz=" << this->output_sz
            << ", actual: input_domain_size=" << input_domain_size
            << ", output_domain_size=" << output_domain_size;

    this->output_sz = output_domain_size;
    this->input_sz = input_domain_size;

    if (this->api->master_activate(this->ec_master) < 0)
        return xerrors::Error(ACTIVATION_ERROR, "ecrt_master_activate failed");

    this->output_domain_data = this->api->domain_data(this->output_domain);
    this->input_domain_data = this->api->domain_data(this->input_domain);

    if (!this->output_domain_data && this->output_sz > 0) {
        this->api->master_deactivate(this->ec_master);
        return xerrors::Error(
            ACTIVATION_ERROR,
            "failed to get output domain data pointer"
        );
    }
    if (!this->input_domain_data && this->input_sz > 0) {
        this->api->master_deactivate(this->ec_master);
        return xerrors::Error(
            ACTIVATION_ERROR,
            "failed to get input domain data pointer"
        );
    }

    this->activated = true;
    LOG(INFO) << "[ethercat.igh] master " << this->master_index
              << " activated successfully";
    VLOG(1) << "[ethercat.igh] output_domain_data="
            << static_cast<void *>(this->output_domain_data)
            << ", input_domain_data=" << static_cast<void *>(this->input_domain_data)
            << ", output_sz=" << this->output_sz << ", input_sz=" << this->input_sz;

    for (const auto &[pos, sc]: this->slave_configs) {
        ec_slave_config_state_t state;
        this->api->slave_config_state(sc, &state);
        VLOG(1) << "[ethercat.igh] slave " << pos << " state after activation: "
                << "al_state=0x" << std::hex << static_cast<int>(state.al_state)
                << std::dec << " ("
                << slave_state_to_string(convert_state(state.al_state)) << ")"
                << ", online=" << state.online << ", operational=" << state.operational;
    }

    return xerrors::NIL;
}

void Master::deactivate() {
    if (!this->initialized) return;

    VLOG(1) << "[ethercat.igh] master " << this->master_index << " deactivating";
    if (this->activated) this->api->master_deactivate(this->ec_master);
    if (this->ec_master) {
        this->api->release_master(this->ec_master);
        this->ec_master = nullptr;
    }
    this->input_domain = nullptr;
    this->output_domain = nullptr;
    this->input_domain_data = nullptr;
    this->output_domain_data = nullptr;
    this->activated = false;
    this->initialized = false;
    this->slave_configs.clear();
    this->pdo_offsets.clear();
    this->cached_slaves.clear();
    this->input_sz = 0;
    this->output_sz = 0;
}

xerrors::Error Master::receive() {
    if (!this->activated) return xerrors::Error(CYCLIC_ERROR, "not activated");

    this->api->master_receive(this->ec_master);

    this->api->domain_process(this->output_domain);
    this->api->domain_process(this->input_domain);

    this->api->domain_state(this->output_domain, &this->output_domain_state);
    this->api->domain_state(this->input_domain, &this->input_domain_state);

    if (this->output_domain_state.wc_state == EC_WC_ZERO ||
        this->input_domain_state.wc_state == EC_WC_ZERO)
        return xerrors::Error(WORKING_COUNTER_ERROR, "no slaves responded");

    if (this->output_domain_state.wc_state == EC_WC_INCOMPLETE ||
        this->input_domain_state.wc_state == EC_WC_INCOMPLETE)
        VLOG(2) << "[ethercat.igh] incomplete WC: output="
                << this->output_domain_state.working_counter
                << ", input=" << this->input_domain_state.working_counter;

    return xerrors::NIL;
}

xerrors::Error Master::send() {
    if (!this->activated) return xerrors::Error(CYCLIC_ERROR, "not activated");

    this->api->domain_queue(this->output_domain);
    this->api->domain_queue(this->input_domain);
    this->api->master_send(this->ec_master);

    return xerrors::NIL;
}

std::span<const uint8_t> Master::input_data() {
    if (!this->activated || !this->input_domain_data) return {};
    return {this->input_domain_data, this->input_sz};
}

std::span<uint8_t> Master::output_data() {
    if (!this->activated || !this->output_domain_data) return {};
    return {this->output_domain_data, this->output_sz};
}

pdo::Offset Master::pdo_offset(const pdo::Entry &entry) const {
    std::lock_guard lock(this->mu);
    const pdo::Key key{
        entry.slave_position,
        entry.index,
        entry.sub_index,
        entry.is_input
    };
    const auto it = this->pdo_offsets.find(key);
    if (it != this->pdo_offsets.end()) return it->second;
    return {};
}

std::vector<slave::Properties> Master::slaves() const {
    std::lock_guard lock(this->mu);
    return this->cached_slaves;
}

slave::State Master::slave_state(const uint16_t position) const {
    std::lock_guard lock(this->mu);

    if (position >= this->cached_slaves.size()) return slave::State::UNKNOWN;

    ec_slave_config_state_t state;
    auto it = this->slave_configs.find(position);
    if (it == this->slave_configs.end()) return slave::State::UNKNOWN;

    this->api->slave_config_state(it->second, &state);
    return convert_state(state.al_state);
}

/// IgH EtherCAT AL (Application Layer) state value for OPERATIONAL.
/// EtherCAT state machine states: INIT=0x01, PRE_OP=0x02, BOOT=0x03, SAFE_OP=0x04,
/// OP=0x08.
constexpr uint8_t IGH_AL_STATE_OP = 0x08;

bool Master::all_slaves_operational() const {
    std::lock_guard lock(this->mu);

    if (!this->activated) return false;

    for (const auto &[pos, sc]: this->slave_configs) {
        if (this->disabled_slaves.contains(pos)) continue;
        ec_slave_config_state_t state;
        this->api->slave_config_state(sc, &state);
        if (state.al_state != IGH_AL_STATE_OP) return false;
    }
    return true;
}

std::string Master::interface_name() const {
    return "igh:" + std::to_string(this->master_index);
}

ec_slave_config_t *Master::get_or_create_slave_config(const uint16_t position) {
    std::lock_guard lock(this->mu);

    if (this->disabled_slaves.contains(position)) {
        VLOG(1) << "[ethercat.igh] skipping slave config for disabled slave "
                << position;
        return nullptr;
    }

    auto it = this->slave_configs.find(position);
    if (it != this->slave_configs.end()) return it->second;

    if (position >= this->cached_slaves.size()) return nullptr;

    const auto &slave = this->cached_slaves[position];
    ec_slave_config_t *sc = this->api->master_slave_config(
        this->ec_master,
        0,
        position,
        slave.vendor_id,
        slave.product_code
    );

    if (!sc) {
        LOG(ERROR) << "[ethercat.igh] failed to get slave config for position "
                   << position << " (vendor=0x" << std::hex << slave.vendor_id
                   << ", product=0x" << slave.product_code << std::dec << ")";
        return nullptr;
    }

    this->configure_slave_pdos(sc, slave);

    size_t registered_outputs = 0;
    size_t registered_inputs = 0;

    for (const auto &pdo: slave.output_pdos) {
        const int result = this->api->slave_config_reg_pdo_entry(
            sc,
            pdo.index,
            pdo.sub_index,
            this->output_domain,
            nullptr
        );
        if (result >= 0) {
            const size_t abs_offset = static_cast<size_t>(result);
            const size_t byte_size = (pdo.bit_length + 7) / 8;
            pdo::Key key{position, pdo.index, pdo.sub_index, false};
            this->pdo_offsets[key] = {abs_offset, 0};
            if (abs_offset + byte_size > this->output_sz)
                this->output_sz = abs_offset + byte_size;
            registered_outputs++;
        } else {
            VLOG(2) << "[ethercat.igh] skipped sub-byte output PDO 0x" << std::hex
                    << pdo.index << ":" << static_cast<int>(pdo.sub_index) << std::dec
                    << " (" << static_cast<int>(pdo.bit_length) << " bits)"
                    << " for slave " << position;
        }
    }

    for (const auto &pdo: slave.input_pdos) {
        int result = this->api->slave_config_reg_pdo_entry(
            sc,
            pdo.index,
            pdo.subindex,
            this->input_domain,
            nullptr
        );
        if (result >= 0) {
            const size_t abs_offset = static_cast<size_t>(result);
            const size_t byte_size = (pdo.bit_length + 7) / 8;
            pdo::Key key{position, pdo.index, pdo.sub_index, true};
            this->pdo_offsets[key] = {abs_offset, 0};
            if (abs_offset + byte_size > this->input_sz)
                this->input_sz = abs_offset + byte_size;
            registered_inputs++;
        } else {
            VLOG(2) << "[ethercat.igh] skipped sub-byte input PDO 0x" << std::hex
                    << pdo.index << ":" << static_cast<int>(pdo.sub_index) << std::dec
                    << " (" << static_cast<int>(pdo.bit_length) << " bits)"
                    << " for slave " << position;
        }
    }

    VLOG(1) << "[ethercat.igh] slave " << position << " (" << slave.name
            << "): registered " << registered_outputs << " output PDOs and "
            << registered_inputs << " input PDOs"
            << " (output_sz=" << this->output_sz << ", input_sz=" << this->input_sz
            << ")";

    this->slave_configs[position] = sc;
    return sc;
}

void Master::configure_slave_pdos(
    ec_slave_config_t *sc,
    const slave::Properties &slave
) {
    std::map<uint16_t, std::vector<ec_pdo_entry_info_t>> output_pdo_entries;
    std::map<uint16_t, std::vector<ec_pdo_entry_info_t>> input_pdo_entries;

    for (const auto &pdo: slave.output_pdos)
        output_pdo_entries[pdo.pdo_index].push_back(
            {pdo.index, pdo.sub_index, pdo.bit_length}
        );
    for (const auto &pdo: slave.input_pdos)
        input_pdo_entries[pdo.pdo_index].push_back(
            {pdo.index, pdo.sub_index, pdo.bit_length}
        );

    std::vector<ec_pdo_info_t> output_pdos;
    std::vector<ec_pdo_info_t> input_pdos;
    output_pdos.reserve(output_pdo_entries.size());
    input_pdos.reserve(input_pdo_entries.size());

    for (auto &[pdo_index, entries]: output_pdo_entries)
        output_pdos.push_back(
            {pdo_index, static_cast<unsigned int>(entries.size()), entries.data()}
        );
    for (auto &[pdo_index, entries]: input_pdo_entries)
        input_pdos.push_back(
            {pdo_index, static_cast<unsigned int>(entries.size()), entries.data()}
        );

    ec_sync_info_t syncs[5] = {
        {0, EC_DIR_OUTPUT, 0, nullptr, EC_WD_DISABLE},
        {1, EC_DIR_INPUT, 0, nullptr, EC_WD_DISABLE},
        {2,
         EC_DIR_OUTPUT,
         static_cast<unsigned int>(output_pdos.size()),
         output_pdos.empty() ? nullptr : output_pdos.data(),
         EC_WD_ENABLE},
        {3,
         EC_DIR_INPUT,
         static_cast<unsigned int>(input_pdos.size()),
         input_pdos.empty() ? nullptr : input_pdos.data(),
         EC_WD_DISABLE},
        {0xff, EC_DIR_INPUT, 0, nullptr, EC_WD_DISABLE}
    };

    if (this->api->slave_config_pdos(sc, 4, syncs) < 0) {
        LOG(WARNING) << "[ethercat.igh] failed to configure PDOs for slave "
                     << slave.position;
    } else {
        VLOG(2) << "[ethercat.igh] configured " << output_pdos.size()
                << " output PDOs and " << input_pdos.size() << " input PDOs for slave "
                << slave.position;
    }
}

std::pair<size_t, xerrors::Error> Master::register_pdo(const pdo::Entry &entry) {
    if (this->activated)
        return {
            0,
            xerrors::Error(PDO_MAPPING_ERROR, "cannot register PDO after activation")
        };

    // get_or_create_slave_config() now registers ALL PDOs for the slave,
    // so we just need to ensure the slave is configured and look up the offset.
    ec_slave_config_t *sc = this->get_or_create_slave_config(entry.slave_position);
    if (!sc)
        return {
            0,
            xerrors::Error(PDO_MAPPING_ERROR, "failed to get slave configuration")
        };

    // Look up the cached offset (already registered by get_or_create_slave_config)
    std::lock_guard lock(this->mu);
    const pdo::Key key{
        entry.slave_position,
        entry.index,
        entry.sub_index,
        entry.is_input
    };
    const auto it = this->pdo_offsets.find(key);
    if (it == this->pdo_offsets.end()) {
        LOG(ERROR) << "[ethercat.igh] PDO 0x" << std::hex << entry.index << ":"
                   << static_cast<int>(entry.sub_index) << std::dec
                   << " not found in cache for slave " << entry.slave_position
                   << " (is_input=" << entry.is_input << ")";
        return {
            0,
            xerrors::Error(
                PDO_MAPPING_ERROR,
                "PDO not found - may not exist in slave's PDO mapping"
            )
        };
    }

    VLOG(1) << "[ethercat.igh] PDO 0x" << std::hex << entry.index << ":"
            << static_cast<int>(entry.sub_index) << std::dec << " for slave "
            << entry.slave_position << " found at offset=" << it->second.byte;

    return {it->second.byte, xerrors::NIL};
}

slave::State Master::convert_state(const uint8_t igh_state) {
    switch (igh_state) {
        case 0x01:
            return slave::State::INIT;
        case 0x02:
            return slave::State::PRE_OP;
        case 0x03:
            return slave::State::BOOT;
        case 0x04:
            return slave::State::SAFE_OP;
        case 0x08:
            return slave::State::OP;
        default:
            return slave::State::UNKNOWN;
    }
}

std::string Master::read_pdo_entry_name(
    const uint16_t slave_pos,
    const uint16_t index,
    const uint8_t subindex
) {
    (void) slave_pos;
    (void) index;
    (void) subindex;
    return "";
}

void Master::discover_slave_pdos(slave::Properties &slave) {
    if (esi::lookup_device_pdos(
            slave.vendor_id,
            slave.product_code,
            slave.revision,
            slave
        )) {
        VLOG(1) << "[ethercat.igh] slave " << slave.position
                << " PDOs discovered via ESI: " << slave.input_pdos.size()
                << " inputs, " << slave.output_pdos.size() << " outputs";
        return;
    }

    ec_slave_info_t slave_info;
    if (this->api->master_get_slave(this->ec_master, slave.position, &slave_info) !=
        0) {
        slave.pdo_discovery_error = "failed to get slave info";
        return;
    }

    for (uint8_t sm_idx = 0; sm_idx < slave_info.sync_count; ++sm_idx) {
        ec_sync_info_t sync_info{};
        if (this->api->master_get_sync_manager(
                this->ec_master,
                slave.position,
                sm_idx,
                &sync_info
            ) != 0)
            continue;

        const bool is_input = (sync_info.dir == EC_DIR_INPUT);

        for (unsigned int pdo_pos = 0; pdo_pos < sync_info.n_pdos; ++pdo_pos) {
            ec_pdo_info_t pdo_info{};
            if (this->api->master_get_pdo(
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
                if (this->api->master_get_pdo_entry(
                        this->ec_master,
                        slave.position,
                        sm_idx,
                        static_cast<uint16_t>(pdo_pos),
                        static_cast<uint16_t>(entry_pos),
                        &entry_info
                    ) != 0)
                    continue;

                if (entry_info.index == 0 && entry_info.subindex == 0) continue;

                const telem::DataType data_type = infer_type_from_bit_length(
                    entry_info.bit_length
                );
                const std::string coe_name = this->read_pdo_entry_name(
                    slave.position,
                    entry_info.index,
                    entry_info.subindex
                );
                const std::string name = generate_pdo_entry_name(
                    coe_name,
                    entry_info.index,
                    entry_info.subindex,
                    is_input,
                    data_type
                );

                pdo::Properties entry(
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
    slave.coe_pdo_order_reliable = true;
    VLOG(1) << "[ethercat.igh] slave " << slave.position
            << " PDOs discovered via IgH: " << slave.input_pdos.size() << " inputs, "
            << slave.output_pdos.size() << " outputs";
}
}
