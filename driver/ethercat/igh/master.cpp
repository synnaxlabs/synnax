// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>

#include "absl/log/log.h"

#include "driver/ethercat/esi/known_devices.h"
#include "driver/ethercat/igh/master.h"
#include "driver/ethercat/telem/telem.h"

namespace driver::ethercat::igh {
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

x::errors::Error Master::initialize() {
    if (this->initialized) return x::errors::NIL;

    if (auto err = this->api->check_device(this->master_index)) return err;
    this->ec_master = this->api->request_master(this->master_index);
    if (!this->ec_master)
        return x::errors::Error(
            errors::MASTER_INIT_ERROR,
            "IgH master not available - is kernel module loaded?"
        );

    ec_master_info_t master_info;
    if (this->api->master(this->ec_master, &master_info) < 0) {
        this->api->release_master(this->ec_master);
        this->ec_master = nullptr;
        return x::errors::Error(errors::MASTER_INIT_ERROR, "failed to get master info");
    }

    this->cached_slaves.clear();
    this->cached_slaves.reserve(master_info.slave_count);

    for (unsigned int i = 0; i < master_info.slave_count; ++i) {
        ec_slave_info_t slave_info;
        if (this->api->master_get_slave(this->ec_master, i, &slave_info) == 0) {
            slave::DiscoveryResult result{};
            result.properties.network = this->interface_name();
            result.properties.position = static_cast<uint16_t>(i);
            result.properties.vendor_id = slave_info.vendor_id;
            result.properties.product_code = slave_info.product_code;
            result.properties.revision = slave_info.revision_number;
            result.properties.serial = slave_info.serial_number;
            result.properties.name = slave_info.name;
            result.status.state = slave::State::INIT;
            this->discover_slave_pdos(result);
            this->cached_slaves.push_back(std::move(result));
        }
    }

    this->output_domain = this->api->master_create_domain(this->ec_master);
    if (!this->output_domain) {
        this->api->release_master(this->ec_master);
        this->ec_master = nullptr;
        return x::errors::Error(
            errors::MASTER_INIT_ERROR,
            "failed to create output domain"
        );
    }

    this->input_domain = this->api->master_create_domain(this->ec_master);
    if (!this->input_domain) {
        this->api->release_master(this->ec_master);
        this->ec_master = nullptr;
        return x::errors::Error(
            errors::MASTER_INIT_ERROR,
            "failed to create input domain"
        );
    }

    this->initialized = true;
    VLOG(1) << "[ethercat.igh] master " << this->master_index << " initialized with "
            << this->cached_slaves.size() << " slaves";
    return x::errors::NIL;
}

std::pair<size_t, x::errors::Error> Master::count_slaves() {
    if (!this->initialized)
        return {0, x::errors::Error(errors::MASTER_INIT_ERROR, "not initialized")};
    ec_master_info_t info;
    if (this->api->master(this->ec_master, &info) < 0)
        return {
            0,
            x::errors::Error(errors::MASTER_INIT_ERROR, "failed to get master info")
        };
    return {info.slave_count, x::errors::NIL};
}

void Master::set_slave_enabled(const uint16_t position, const bool enabled) {
    std::lock_guard lock(this->mu);
    if (!enabled)
        this->disabled_slaves.insert(position);
    else
        this->disabled_slaves.erase(position);
}

x::errors::Error Master::activate() {
    if (!this->initialized)
        return x::errors::Error(errors::ACTIVATION_ERROR, "not initialized");
    if (this->activated) return x::errors::NIL;
    if (!this->output_domain || !this->input_domain)
        return x::errors::Error(errors::ACTIVATION_ERROR, "domains not created");

    for (const auto &slave: this->cached_slaves) {
        const auto pos = slave.properties.position;
        {
            std::lock_guard lock(this->mu);
            if (this->disabled_slaves.contains(pos)) continue;
        }
        if (this->get_or_create_slave_config(pos) == nullptr)
            return x::errors::Error(
                errors::ACTIVATION_ERROR,
                "failed to configure slave " + std::to_string(pos) + " (" +
                    slave.properties.name + ")"
            );
    }

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
        return x::errors::Error(
            errors::ACTIVATION_ERROR,
            "ecrt_master_activate failed"
        );

    this->output_domain_data = this->api->domain_data(this->output_domain);
    this->input_domain_data = this->api->domain_data(this->input_domain);

    if (!this->output_domain_data && this->output_sz > 0) {
        this->api->master_deactivate(this->ec_master);
        return x::errors::Error(
            errors::ACTIVATION_ERROR,
            "failed to get output domain data pointer"
        );
    }
    if (!this->input_domain_data && this->input_sz > 0) {
        this->api->master_deactivate(this->ec_master);
        return x::errors::Error(
            errors::ACTIVATION_ERROR,
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
                << slave_state_to_string(slave::from_al_state(state.al_state)) << ")"
                << ", online=" << state.online << ", operational=" << state.operational;
    }

    return x::errors::NIL;
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

x::errors::Error Master::receive() {
    if (!this->activated)
        return x::errors::Error(errors::CYCLIC_ERROR, "not activated");

    this->api->master_receive(this->ec_master);

    this->api->domain_process(this->output_domain);
    this->api->domain_process(this->input_domain);

    this->api->domain_state(this->output_domain, &this->output_domain_state);
    this->api->domain_state(this->input_domain, &this->input_domain_state);

    // IgH reports a zero working counter for an empty domain, which exchanges no data.
    if ((this->output_sz > 0 && this->output_domain_state.wc_state == EC_WC_ZERO) ||
        (this->input_sz > 0 && this->input_domain_state.wc_state == EC_WC_ZERO))
        return x::errors::Error(errors::WORKING_COUNTER_ERROR, "no slaves responded");

    if (this->output_domain_state.wc_state == EC_WC_INCOMPLETE ||
        this->input_domain_state.wc_state == EC_WC_INCOMPLETE)
        return x::errors::Error(
            errors::WORKING_COUNTER_ERROR,
            "incomplete working counter: output " +
                std::to_string(this->output_domain_state.working_counter) + ", input " +
                std::to_string(this->input_domain_state.working_counter)
        );

    return x::errors::NIL;
}

x::errors::Error Master::send() {
    if (!this->activated)
        return x::errors::Error(errors::CYCLIC_ERROR, "not activated");

    this->api->domain_queue(this->output_domain);
    this->api->domain_queue(this->input_domain);
    this->api->master_send(this->ec_master);

    return x::errors::NIL;
}

std::span<const uint8_t> Master::input_data() {
    if (!this->activated || !this->input_domain_data) return {};
    return {this->input_domain_data, this->input_sz};
}

std::span<uint8_t> Master::output_data() {
    if (!this->activated || !this->output_domain_data) return {};
    return {this->output_domain_data, this->output_sz};
}

std::pair<pdo::Offset, x::errors::Error>
Master::pdo_offset(const pdo::Entry &entry) const {
    std::lock_guard lock(this->mu);
    return pdo::find_offset(this->pdo_offsets, entry);
}

std::vector<slave::DiscoveryResult> Master::slaves() const {
    std::lock_guard lock(this->mu);
    return this->cached_slaves;
}

std::vector<slave::Health> Master::check(const x::breaker::Breaker &) {
    std::vector<std::pair<uint16_t, ec_slave_config_t *>> configs;
    {
        std::lock_guard lock(this->mu);
        configs.assign(this->slave_configs.begin(), this->slave_configs.end());
    }
    std::ranges::sort(configs);
    std::vector<slave::Health> health;
    health.reserve(configs.size());
    for (const auto &[pos, sc]: configs) {
        ec_slave_config_state_t state{};
        this->api->slave_config_state(sc, &state);
        ec_slave_info_t info{};
        const bool faulted = this->api->master_get_slave(this->ec_master, pos, &info) ==
                                 0 &&
                             info.error_flag != 0;
        health.push_back({
            .position = pos,
            .state = slave::from_al_state(state.al_state),
            .faulted = faulted,
            .lost = state.online == 0,
        });
    }
    return health;
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

    const auto &slave = this->cached_slaves[position].properties;
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

    size_t registered_outputs = 0;
    size_t registered_inputs = 0;

    for (const auto &pdo: slave.output_pdos) {
        if (pdo.is_padding()) continue;
        unsigned int bit_pos = 0;
        const int result = this->api->slave_config_reg_pdo_entry(
            sc,
            pdo.index,
            pdo.sub_index,
            this->output_domain,
            &bit_pos
        );
        if (result >= 0) {
            const auto abs_offset = static_cast<size_t>(result);
            const size_t byte_size = telem::pdo_required_bytes(
                static_cast<uint8_t>(bit_pos),
                pdo.bit_length
            );
            pdo::Key key{position, pdo.index, pdo.sub_index, false};
            this->pdo_offsets[key] = {
                abs_offset,
                static_cast<uint8_t>(bit_pos),
                pdo.bit_length
            };
            this->output_sz = std::max(this->output_sz, abs_offset + byte_size);
            registered_outputs++;
        } else {
            LOG(WARNING) << "[ethercat.igh] failed to register output PDO 0x"
                         << std::hex << pdo.index << ":"
                         << static_cast<int>(pdo.sub_index) << std::dec << " ("
                         << static_cast<int>(pdo.bit_length) << " bits)"
                         << " for slave " << position << " (error=" << result << ")";
        }
    }

    for (const auto &pdo: slave.input_pdos) {
        if (pdo.is_padding()) continue;
        unsigned int bit_pos = 0;
        const int result = this->api->slave_config_reg_pdo_entry(
            sc,
            pdo.index,
            pdo.sub_index,
            this->input_domain,
            &bit_pos
        );
        if (result >= 0) {
            const auto abs_offset = static_cast<size_t>(result);
            const size_t byte_size = telem::pdo_required_bytes(
                static_cast<uint8_t>(bit_pos),
                pdo.bit_length
            );
            pdo::Key key{position, pdo.index, pdo.sub_index, true};
            this->pdo_offsets[key] = {
                abs_offset,
                static_cast<uint8_t>(bit_pos),
                pdo.bit_length
            };
            this->input_sz = std::max(this->input_sz, abs_offset + byte_size);
            registered_inputs++;
        } else {
            LOG(WARNING) << "[ethercat.igh] failed to register input PDO 0x" << std::hex
                         << pdo.index << ":" << static_cast<int>(pdo.sub_index)
                         << std::dec << " (" << static_cast<int>(pdo.bit_length)
                         << " bits)"
                         << " for slave " << position << " (error=" << result << ")";
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

void Master::discover_slave_pdos(slave::DiscoveryResult &slave) {
    auto &props = slave.properties;
    this->read_slave_pdos(slave);
    if (!props.input_pdos.empty() || !props.output_pdos.empty()) {
        esi::fill_known_entries(props);
        return;
    }
    if (!esi::lookup_device_pdos(
            props.vendor_id,
            props.product_code,
            props.revision,
            props
        ))
        return;
    VLOG(1) << "[ethercat.igh] slave " << props.position
            << " PDOs discovered via ESI: " << props.input_pdos.size() << " inputs, "
            << props.output_pdos.size() << " outputs";
    slave.status.pdos_discovered = true;
    slave.status.pdo_discovery_error.clear();
    props.coe_pdo_order_reliable = true;
}

void Master::read_slave_pdos(slave::DiscoveryResult &slave) {
    auto &props = slave.properties;
    auto &status = slave.status;
    ec_slave_info_t slave_info;
    if (this->api->master_get_slave(this->ec_master, props.position, &slave_info) !=
        0) {
        status.pdo_discovery_error = "failed to get slave info";
        return;
    }

    for (uint8_t sm_idx = 0; sm_idx < slave_info.sync_count; ++sm_idx) {
        ec_sync_info_t sync_info{};
        if (this->api->master_get_sync_manager(
                this->ec_master,
                props.position,
                sm_idx,
                &sync_info
            ) != 0)
            continue;

        const bool is_input = (sync_info.dir == EC_DIR_INPUT);

        for (unsigned int pdo_pos = 0; pdo_pos < sync_info.n_pdos; ++pdo_pos) {
            ec_pdo_info_t pdo_info{};
            if (this->api->master_get_pdo(
                    this->ec_master,
                    props.position,
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
                        props.position,
                        sm_idx,
                        static_cast<uint16_t>(pdo_pos),
                        static_cast<uint16_t>(entry_pos),
                        &entry_info
                    ) != 0)
                    continue;

                if (entry_info.index == 0) {
                    pdo::Properties gap{
                        .pdo_index = pdo_info.index,
                        .bit_length = entry_info.bit_length,
                        .is_input = is_input,
                    };
                    if (is_input)
                        props.input_pdos.push_back(gap);
                    else
                        props.output_pdos.push_back(gap);
                    continue;
                }

                const x::telem::DataType data_type = telem::infer_type_from_bit_length(
                    entry_info.bit_length
                );
                const std::string coe_name = this->read_pdo_entry_name(
                    props.position,
                    entry_info.index,
                    entry_info.subindex
                );
                const std::string name = telem::generate_pdo_entry_name(
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
                    props.input_pdos.push_back(entry);
                else
                    props.output_pdos.push_back(entry);
            }
        }
    }

    status.pdos_discovered = true;
    props.coe_pdo_order_reliable = true;
    VLOG(1) << "[ethercat.igh] slave " << props.position
            << " PDOs discovered via IgH: " << props.input_pdos.size() << " inputs, "
            << props.output_pdos.size() << " outputs";
}
}
