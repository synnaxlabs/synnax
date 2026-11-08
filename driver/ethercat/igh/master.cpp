// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <array>
#include <format>
#include <system_error>

#include "absl/log/log.h"

#include "driver/ethercat/coe/coe.h"
#include "driver/ethercat/esi/known_devices.h"
#include "driver/ethercat/igh/master.h"
#include "driver/ethercat/telem/telem.h"

namespace driver::ethercat::igh {
namespace {
/// @brief reads the PDO layout that the slave at position assigns.
std::pair<std::vector<pdo::Sync>, x::errors::Error>
read_layout(const API &api, ec_master_t *master, const uint16_t position) {
    return coe::read_layout(
        [&](const uint16_t index,
            const uint8_t sub_index) -> std::pair<uint32_t, x::errors::Error> {
            std::array<uint8_t, 4> data{};
            size_t size = 0;
            uint32_t abort_code = 0;
            if (api.master_sdo_upload(
                    master,
                    position,
                    index,
                    sub_index,
                    data.data(),
                    data.size(),
                    &size,
                    &abort_code
                ) < 0)
                return {0, coe::read_error(index, sub_index, abort_code)};
            uint32_t value = 0;
            for (size_t i = 0; i < size; i++)
                value |= static_cast<uint32_t>(data[i]) << (8 * i);
            return {value, x::errors::NIL};
        },
        EC_MAX_SYNC_MANAGERS
    );
}
}

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

std::pair<std::vector<slave::DiscoveryResult>, x::errors::Error> Master::scan() {
    if (!this->initialized)
        if (auto err = this->initialize()) return {{}, err};
    std::lock_guard lock(this->mu);
    return {this->cached_slaves, x::errors::NIL};
}

x::errors::Error Master::initialize() {
    if (auto err = this->api->check_device(this->master_index)) return err;
    this->ec_master = this->api->request_master(this->master_index);
    if (!this->ec_master)
        return x::errors::Error(
            errors::MASTER_INIT_ERROR,
            "IgH master not available - is kernel module loaded?"
        );

    const auto fail = [this](const std::string &msg) {
        this->api->release_master(this->ec_master);
        this->ec_master = nullptr;
        return x::errors::Error(errors::MASTER_INIT_ERROR, msg);
    };

    ec_master_info_t master_info;
    if (this->api->master(this->ec_master, &master_info) < 0)
        return fail("failed to get master info");

    this->cached_slaves.clear();
    this->cached_slaves.reserve(master_info.slave_count);

    for (unsigned int i = 0; i < master_info.slave_count; ++i) {
        ec_slave_info_t slave_info;
        if (const int ret = this->api
                                ->master_get_slave(this->ec_master, i, &slave_info);
            ret != 0)
            return fail(
                std::format(
                    "failed to read slave {}: {}",
                    i,
                    std::generic_category().message(-ret)
                )
            );
        slave::DiscoveryResult result{};
        result.properties.network = "igh:" + std::to_string(this->master_index);
        result.properties.position = static_cast<uint16_t>(i);
        result.properties.vendor_id = slave_info.vendor_id;
        result.properties.product_code = slave_info.product_code;
        result.properties.revision = slave_info.revision_number;
        result.properties.serial = slave_info.serial_number;
        result.properties.name = slave_info.name;
        result.status.state = slave::State::INIT;
        this->discover_slave_pdos(result, slave_info.sync_count);
        this->cached_slaves.push_back(std::move(result));
    }

    this->output_domain = this->api->master_create_domain(this->ec_master);
    if (!this->output_domain) return fail("failed to create output domain");
    this->input_domain = this->api->master_create_domain(this->ec_master);
    if (!this->input_domain) return fail("failed to create input domain");

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

x::errors::Error Master::add_startup_parameters(
    const slave::Properties &slave,
    ec_slave_config_t *sc,
    const std::vector<slave::StartupParameter> &parameters
) {
    const auto slave_name = "slave " + std::to_string(slave.position) + " (" +
                            slave.name + ")";
    for (const auto &param: parameters) {
        const auto data = param.bytes();
        // IgH reports a rejected config SDO only as a slave stuck in PRE-OP, so a
        // direct write first gets the abort code.
        uint32_t abort_code = 0;
        if (this->api->master_sdo_download(
                this->ec_master,
                slave.position,
                param.index,
                param.sub_index,
                data.data(),
                data.size(),
                &abort_code
            ) < 0)
            return x::errors::Error(
                errors::STARTUP_PARAMETER_ERROR,
                slave_name + " rejected startup parameter " + param.to_string() + ": " +
                    errors::sdo_abort_reason(abort_code)
            );
        if (this->api->slave_config_sdo(
                sc,
                param.index,
                param.sub_index,
                data.data(),
                data.size()
            ) < 0)
            return x::errors::Error(
                errors::ACTIVATION_ERROR,
                "failed to add startup parameter " + param.to_string() + " to " +
                    slave_name
            );
    }
    return x::errors::NIL;
}

master::Activation Master::activate(
    const master::Plan &plan,
    const std::function<void(const master::Image &)> &start_cycle
) {
    if (!this->initialized)
        return {.err = x::errors::Error(errors::ACTIVATION_ERROR, "not initialized")};
    if (this->activated)
        return {.err = x::errors::Error(errors::ACTIVATION_ERROR, "already activated")};
    if (!this->output_domain || !this->input_domain)
        return {
            .err = x::errors::Error(errors::ACTIVATION_ERROR, "domains not created")
        };
    {
        std::lock_guard lock(this->mu);
        this->disabled_slaves.clear();
        this->chosen_modes.clear();
        this->startup_parameters.clear();
        for (const auto &[position, config]: plan.slaves) {
            if (!config.enabled) this->disabled_slaves.insert(position);
            if (config.op_mode) this->chosen_modes[position] = *config.op_mode;
            this->startup_parameters[position] = config.startup_parameters;
        }
    }

    master::Image image;
    for (const auto &slave: this->cached_slaves) {
        const auto pos = slave.properties.position;
        std::vector<slave::StartupParameter> params;
        {
            std::lock_guard lock(this->mu);
            if (this->disabled_slaves.contains(pos)) continue;
            const auto choice = this->chosen_modes.find(pos);
            if (choice != this->chosen_modes.end()) {
                const auto err = choice->second.assign_activate != 0
                                   ? x::errors::Error(
                                         errors::SLAVE_CONFIG_ERROR,
                                         "the IgH master does not run DC sync"
                                     )
                                   : dc::check_cycles({}, choice->second.cycles);
                if (err)
                    return {
                        .err = x::errors::Error(
                            errors::SLAVE_CONFIG_ERROR,
                            std::format(
                                "slave {} ({}) cannot use its chosen mode: {}",
                                pos,
                                slave.properties.name,
                                err.data
                            )
                        )
                    };
            }
            if (const auto it = this->startup_parameters.find(pos);
                it != this->startup_parameters.end())
                params = it->second;
        }
        auto *sc = this->get_or_create_slave_config(pos);
        if (sc == nullptr)
            return {
                .err = x::errors::Error(
                    errors::ACTIVATION_ERROR,
                    "failed to configure slave " + std::to_string(pos) + " (" +
                        slave.properties.name + ")"
                )
            };
        if (auto err = this->add_startup_parameters(slave.properties, sc, params))
            return {.err = err};
        if (!params.empty())
            if (auto err = this->configure_assigned_pdos(pos, sc)) return {.err = err};
        this->register_pdo_entries(slave.properties, sc, image);
    }

    this->output_sz = this->api->domain_size(this->output_domain);
    this->input_sz = this->api->domain_size(this->input_domain);
    VLOG(1) << "[ethercat.igh] activating master " << this->master_index << " with "
            << this->slave_configs.size()
            << " configured slaves, input_sz=" << this->input_sz
            << ", output_sz=" << this->output_sz;

    if (this->api->master_activate(this->ec_master) < 0)
        return {
            .err = x::errors::Error(
                errors::ACTIVATION_ERROR,
                "ecrt_master_activate failed"
            )
        };

    this->output_domain_data = this->api->domain_data(this->output_domain);
    this->input_domain_data = this->api->domain_data(this->input_domain);

    if (!this->output_domain_data && this->output_sz > 0) {
        this->api->master_deactivate(this->ec_master);
        return {
            .err = x::errors::Error(
                errors::ACTIVATION_ERROR,
                "failed to get output domain data pointer"
            )
        };
    }
    if (!this->input_domain_data && this->input_sz > 0) {
        this->api->master_deactivate(this->ec_master);
        return {
            .err = x::errors::Error(
                errors::ACTIVATION_ERROR,
                "failed to get input domain data pointer"
            )
        };
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
    image.inputs = {this->input_domain_data, this->input_sz};
    image.outputs = {this->output_domain_data, this->output_sz};
    {
        std::lock_guard lock(this->mu);
        image.slaves = this->cached_slaves;
    }
    // The IgH master moves the slaves to OP on its own while the caller cycles.
    start_cycle(image);
    return {};
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

    this->slave_configs[position] = sc;
    return sc;
}

x::errors::Error
Master::configure_assigned_pdos(const uint16_t position, ec_slave_config_t *sc) {
    auto props = this->cached_slaves[position].properties;
    const auto slave_name = "slave " + std::to_string(position) + " (" + props.name +
                            ")";
    const auto [syncs, err] = read_layout(*this->api, this->ec_master, position);
    if (err)
        return x::errors::Error(
            errors::PDO_DISCOVERY_ERROR,
            "failed to read the PDO layout of " + slave_name +
                " after its startup parameters: " + err.data
        );
    const auto assignable = [&](const bool is_input) {
        return std::ranges::any_of(syncs, [is_input](const pdo::Sync &s) {
            return !s.fixed && s.is_input == is_input;
        });
    };
    if (!assignable(true) && !assignable(false)) return x::errors::NIL;
    std::vector<std::vector<ec_pdo_entry_info_t>> entry_infos;
    std::vector<std::vector<ec_pdo_info_t>> pdo_infos;
    pdo_infos.reserve(syncs.size());
    std::vector<ec_sync_info_t> sync_infos;
    std::vector<uint8_t> empty_syncs;
    std::vector<uint16_t> empty_pdos;
    // A direction with no assignable sync manager keeps the PDOs found at discovery.
    if (assignable(true)) props.input_pdos.clear();
    if (assignable(false)) props.output_pdos.clear();
    for (const auto &sync: syncs) {
        if (sync.fixed) {
            if (assignable(sync.is_input)) this->append_scanned_pdos(props, sync.index);
            continue;
        }
        auto &infos = pdo_infos.emplace_back();
        if (sync.pdos.empty()) empty_syncs.push_back(sync.index);
        for (const auto &pdo: sync.pdos) {
            // Moving a vector keeps its buffer, so IgH can hold the pointer.
            auto &entries = entry_infos.emplace_back();
            for (const auto &entry: pdo.entries)
                entries.push_back({
                    .index = entry.index,
                    .subindex = entry.sub_index,
                    .bit_length = entry.bit_length,
                });
            infos.push_back({
                .index = pdo.index,
                .n_entries = static_cast<unsigned int>(entries.size()),
                .entries = entries.data(),
            });
            if (pdo.entries.empty()) empty_pdos.push_back(pdo.index);
            pdo::append(
                sync.is_input ? props.input_pdos : props.output_pdos,
                sync.is_input,
                pdo
            );
        }
        sync_infos.push_back({
            .index = sync.index,
            .dir = sync.is_input ? EC_DIR_INPUT : EC_DIR_OUTPUT,
            .n_pdos = static_cast<unsigned int>(infos.size()),
            .pdos = infos.data(),
            .watchdog_mode = EC_WD_DEFAULT,
        });
    }
    if (this->api->slave_config_pdos(
            sc,
            static_cast<unsigned int>(sync_infos.size()),
            sync_infos.data()
        ) < 0)
        return x::errors::Error(
            errors::PDO_MAPPING_ERROR,
            "failed to configure the PDOs of " + slave_name
        );
    // ecrt_slave_config_pdos keeps the default assignment of a sync manager with no
    // PDOs and the default mapping of a PDO with no entries.
    for (const auto sync: empty_syncs)
        if (this->api->slave_config_pdo_assign_clear(sc, sync) < 0)
            return x::errors::Error(
                errors::PDO_MAPPING_ERROR,
                "failed to clear the PDO assignment of " + slave_name
            );
    for (const auto pdo: empty_pdos)
        if (this->api->slave_config_pdo_mapping_clear(sc, pdo) < 0)
            return x::errors::Error(
                errors::PDO_MAPPING_ERROR,
                "failed to clear the PDO mapping of " + slave_name
            );
    esi::fill_known_entries(props);
    std::lock_guard lock(this->mu);
    this->cached_slaves[position].properties = std::move(props);
    return x::errors::NIL;
}

void Master::register_pdo_entries(
    const slave::Properties &slave,
    ec_slave_config_t *sc,
    master::Image &image
) {
    const auto add = [&](const pdo::Properties &pdo, ec_domain_t *domain) {
        if (pdo.is_padding()) return;
        const pdo::Key key{slave.position, pdo.index, pdo.sub_index, pdo.is_input};
        unsigned int bit = 0;
        const int offset = this->api->slave_config_reg_pdo_entry(
            sc,
            pdo.index,
            pdo.sub_index,
            domain,
            &bit
        );
        if (offset < 0) {
            image.errors.emplace(
                key,
                x::errors::Error(
                    errors::PDO_MAPPING_ERROR,
                    std::format(
                        "the IgH master rejected PDO {} of slave {} ({}): {}",
                        telem::format_index_sub_index(pdo.index, pdo.sub_index),
                        slave.position,
                        slave.name,
                        std::generic_category().message(-offset)
                    )
                )
            );
            return;
        }
        image.offsets[key] = {
            static_cast<size_t>(offset),
            static_cast<uint8_t>(bit),
            pdo.bit_length
        };
    };
    for (const auto &pdo: slave.output_pdos)
        add(pdo, this->output_domain);
    for (const auto &pdo: slave.input_pdos)
        add(pdo, this->input_domain);
}

void Master::append_scanned_pdos(slave::Properties &props, const uint8_t sync_index) {
    ec_sync_info_t sync_info{};
    if (this->api->master_get_sync_manager(
            this->ec_master,
            props.position,
            sync_index,
            &sync_info
        ) != 0)
        return;
    const bool is_input = sync_info.dir == EC_DIR_INPUT;
    for (unsigned int pdo_pos = 0; pdo_pos < sync_info.n_pdos; ++pdo_pos) {
        ec_pdo_info_t pdo_info{};
        if (this->api->master_get_pdo(
                this->ec_master,
                props.position,
                sync_index,
                static_cast<uint16_t>(pdo_pos),
                &pdo_info
            ) != 0)
            continue;
        pdo::PDO pdo{.index = pdo_info.index, .entries = {}};
        for (unsigned int entry_pos = 0; entry_pos < pdo_info.n_entries; ++entry_pos) {
            ec_pdo_entry_info_t entry{};
            if (this->api->master_get_pdo_entry(
                    this->ec_master,
                    props.position,
                    sync_index,
                    static_cast<uint16_t>(pdo_pos),
                    static_cast<uint16_t>(entry_pos),
                    &entry
                ) != 0)
                continue;
            pdo.entries.push_back({
                .index = entry.index,
                .sub_index = entry.subindex,
                .bit_length = entry.bit_length,
            });
        }
        pdo::append(is_input ? props.input_pdos : props.output_pdos, is_input, pdo);
    }
}

void Master::discover_slave_pdos(
    slave::DiscoveryResult &slave,
    const uint8_t sync_count
) {
    for (uint8_t sm = 0; sm < sync_count; ++sm)
        this->append_scanned_pdos(slave.properties, sm);
    slave.status.pdos_discovered = true;
    esi::fill(slave);
}
}
