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

#include "absl/log/log.h"

#include "driver/ethercat/esi/known_devices.h"
#include "driver/ethercat/igh/master.h"
#include "driver/ethercat/telem/telem.h"

namespace driver::ethercat::igh {
namespace {
constexpr uint16_t SM_TYPE_INDEX = 0x1C00;
constexpr uint16_t PDO_ASSIGN_INDEX = 0x1C10;
constexpr uint32_t SM_TYPE_OUTPUTS = 3;
constexpr uint32_t SM_TYPE_INPUTS = 4;
/// @brief the most sync managers a slave can type, one per PDO assign object from
/// 0x1C10 to 0x1C2F.
constexpr uint32_t MAX_SYNC_MANAGERS = 32;
/// @brief the most PDOs an assign object, or entries a mapping object, can count.
constexpr uint32_t MAX_SUB_INDEX = 254;
constexpr uint32_t ABORT_OBJECT_MISSING = 0x06020000;
const x::errors::Error OBJECT_MISSING = errors::SDO_READ_ERROR.sub("object_missing");

struct AssignedPDO {
    uint16_t index;
    std::vector<ec_pdo_entry_info_t> entries;
};

struct AssignedSync {
    uint8_t index;
    ec_direction_t dir;
    std::vector<AssignedPDO> pdos;
};

/// @brief reads an unsigned value of at most four bytes from a slave's object
/// dictionary.
std::pair<uint32_t, x::errors::Error> read_sdo(
    const API &api,
    ec_master_t *master,
    const uint16_t position,
    const uint16_t index,
    const uint8_t sub_index
) {
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
        return {
            0,
            x::errors::Error(
                abort_code == ABORT_OBJECT_MISSING ? OBJECT_MISSING
                                                   : errors::SDO_READ_ERROR,
                "failed to read " + telem::format_index_sub_index(index, sub_index) +
                    ": " + errors::sdo_abort_reason(abort_code)
            )
        };
    uint32_t value = 0;
    for (size_t i = 0; i < size; i++)
        value |= static_cast<uint32_t>(data[i]) << (8 * i);
    return {value, x::errors::NIL};
}

/// @brief reads the PDOs a slave assigns to each process data sync manager and the
/// entries each PDO maps. Returns no sync managers for a slave without the sync
/// manager type object, whose layout is the one its EEPROM describes.
std::pair<std::vector<AssignedSync>, x::errors::Error>
read_assigned_pdos(const API &api, ec_master_t *master, const uint16_t position) {
    const auto read = [&](const uint16_t index, const uint32_t sub_index) {
        return read_sdo(api, master, position, index, static_cast<uint8_t>(sub_index));
    };
    const auto read_count =
        [&](const uint16_t index,
            const uint32_t max) -> std::pair<uint32_t, x::errors::Error> {
        const auto [count, err] = read(index, 0);
        if (err || count <= max) return {count, err};
        return {
            0,
            x::errors::Error(
                errors::PDO_DISCOVERY_ERROR,
                telem::format_index_sub_index(index, 0) + " counts " +
                    std::to_string(count) + " entries, above the maximum of " +
                    std::to_string(max)
            )
        };
    };
    const auto [sm_count, count_err] = read_count(SM_TYPE_INDEX, MAX_SYNC_MANAGERS);
    if (count_err.matches(OBJECT_MISSING)) return {{}, x::errors::NIL};
    if (count_err) return {{}, count_err};
    std::vector<AssignedSync> syncs;
    for (uint32_t sm = 0; sm < sm_count; sm++) {
        const auto [type, type_err] = read(SM_TYPE_INDEX, sm + 1);
        if (type_err) return {{}, type_err};
        if (type != SM_TYPE_OUTPUTS && type != SM_TYPE_INPUTS) continue;
        auto &sync = syncs.emplace_back(
            AssignedSync{
                .index = static_cast<uint8_t>(sm),
                .dir = type == SM_TYPE_INPUTS ? EC_DIR_INPUT : EC_DIR_OUTPUT,
                .pdos = {},
            }
        );
        const auto assign_index = static_cast<uint16_t>(PDO_ASSIGN_INDEX + sm);
        const auto [pdo_count, pdo_count_err] = read_count(assign_index, MAX_SUB_INDEX);
        if (pdo_count_err) return {{}, pdo_count_err};
        for (uint32_t i = 1; i <= pdo_count; i++) {
            const auto [pdo_index, pdo_err] = read(assign_index, i);
            if (pdo_err) return {{}, pdo_err};
            auto &pdo = sync.pdos.emplace_back(
                AssignedPDO{.index = static_cast<uint16_t>(pdo_index), .entries = {}}
            );
            const auto [entry_count, entry_count_err] = read_count(
                pdo.index,
                MAX_SUB_INDEX
            );
            if (entry_count_err) return {{}, entry_count_err};
            for (uint32_t j = 1; j <= entry_count; j++) {
                const auto [mapping, mapping_err] = read(pdo.index, j);
                if (mapping_err) return {{}, mapping_err};
                pdo.entries.push_back({
                    .index = static_cast<uint16_t>(mapping >> 16),
                    .subindex = static_cast<uint8_t>(mapping >> 8),
                    .bit_length = static_cast<uint8_t>(mapping),
                });
            }
        }
    }
    return {syncs, x::errors::NIL};
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

void Master::set_slave_config(const uint16_t position, slave::Config config) {
    std::lock_guard lock(this->mu);
    if (config.enabled)
        this->disabled_slaves.erase(position);
    else
        this->disabled_slaves.insert(position);
    this->startup_parameters[position] = std::move(config.startup_parameters);
}

x::errors::Error Master::activate() {
    if (!this->initialized)
        return x::errors::Error(errors::ACTIVATION_ERROR, "not initialized");
    if (this->activated) return x::errors::NIL;
    if (!this->output_domain || !this->input_domain)
        return x::errors::Error(errors::ACTIVATION_ERROR, "domains not created");

    for (const auto &slave: this->cached_slaves) {
        const auto pos = slave.properties.position;
        std::vector<slave::StartupParameter> params;
        {
            std::lock_guard lock(this->mu);
            if (this->disabled_slaves.contains(pos)) continue;
            if (const auto it = this->startup_parameters.find(pos);
                it != this->startup_parameters.end())
                params = it->second;
        }
        auto *sc = this->get_or_create_slave_config(pos);
        if (sc == nullptr)
            return x::errors::Error(
                errors::ACTIVATION_ERROR,
                "failed to configure slave " + std::to_string(pos) + " (" +
                    slave.properties.name + ")"
            );
        if (auto err = this->add_startup_parameters(slave.properties, sc, params))
            return err;
        if (!params.empty())
            if (auto err = this->configure_assigned_pdos(pos, sc)) return err;
        this->register_pdo_entries(pos, sc);
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

    this->slave_configs[position] = sc;
    return sc;
}

x::errors::Error
Master::configure_assigned_pdos(const uint16_t position, ec_slave_config_t *sc) {
    auto props = this->cached_slaves[position].properties;
    const auto slave_name = "slave " + std::to_string(position) + " (" + props.name +
                            ")";
    const auto [syncs, err] = read_assigned_pdos(*this->api, this->ec_master, position);
    if (err)
        return x::errors::Error(
            errors::PDO_DISCOVERY_ERROR,
            "failed to read the PDO layout of " + slave_name +
                " after its startup parameters: " + err.data
        );
    if (syncs.empty()) return x::errors::NIL;
    std::vector<std::vector<ec_pdo_info_t>> pdo_infos;
    pdo_infos.reserve(syncs.size());
    std::vector<ec_sync_info_t> sync_infos;
    props.input_pdos.clear();
    props.output_pdos.clear();
    for (const auto &sync: syncs) {
        auto &infos = pdo_infos.emplace_back();
        for (const auto &pdo: sync.pdos) {
            infos.push_back({
                .index = pdo.index,
                .n_entries = static_cast<unsigned int>(pdo.entries.size()),
                .entries = pdo.entries.data(),
            });
            for (const auto &entry: pdo.entries)
                this->append_pdo_entry(
                    props,
                    sync.dir == EC_DIR_INPUT,
                    pdo.index,
                    entry
                );
        }
        sync_infos.push_back({
            .index = sync.index,
            .dir = sync.dir,
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
    esi::fill_known_entries(props);
    std::lock_guard lock(this->mu);
    this->cached_slaves[position].properties = std::move(props);
    return x::errors::NIL;
}

void Master::register_pdo_entries(const uint16_t position, ec_slave_config_t *sc) {
    std::lock_guard lock(this->mu);
    const auto &slave = this->cached_slaves[position].properties;
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

void Master::append_pdo_entry(
    slave::Properties &props,
    const bool is_input,
    const uint16_t pdo_index,
    const ec_pdo_entry_info_t &entry_info
) {
    auto &pdos = is_input ? props.input_pdos : props.output_pdos;
    if (entry_info.index == 0) {
        pdos.push_back({
            .pdo_index = pdo_index,
            .bit_length = entry_info.bit_length,
            .is_input = is_input,
        });
        return;
    }
    const x::telem::DataType data_type = telem::infer_type_from_bit_length(
        entry_info.bit_length
    );
    const std::string coe_name = this->read_pdo_entry_name(
        props.position,
        entry_info.index,
        entry_info.subindex
    );
    pdos.emplace_back(
        pdo_index,
        entry_info.index,
        entry_info.subindex,
        entry_info.bit_length,
        is_input,
        telem::generate_pdo_entry_name(
            coe_name,
            entry_info.index,
            entry_info.subindex,
            is_input,
            data_type
        ),
        data_type
    );
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

                this->append_pdo_entry(props, is_input, pdo_info.index, entry_info);
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
