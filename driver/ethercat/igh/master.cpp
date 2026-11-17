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
#include <charconv>
#include <cstring>
#include <format>
#include <optional>
#include <string_view>
#include <system_error>

#include "absl/log/log.h"
#include <dirent.h>
#include <sys/stat.h>

#include "driver/ethercat/coe/coe.h"
#include "driver/ethercat/esi/known_devices.h"
#include "driver/ethercat/esi/settings.h"
#include "driver/ethercat/igh/master.h"
#include "driver/ethercat/telem/telem.h"

namespace driver::ethercat::igh {
namespace {
/// @brief where the IgH kernel module lists its masters, as EtherCAT<n>.
constexpr auto SYSFS_PATH = "/sys/class/EtherCAT";
/// @brief the device of the first IgH master.
constexpr auto DEVICE_PATH = "/dev/EtherCAT0";

/// @brief returns a read of the object dictionary of the slave at position.
coe::Read sdo_reader(const API &api, ec_master_t *master, const uint16_t position) {
    return [&api, master, position](const uint16_t index, const uint8_t sub_index)
               -> std::pair<uint32_t, x::errors::Error> {
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
    };
}

/// @brief reads the PDO layout that the slave at position assigns.
std::pair<std::vector<pdo::Sync>, x::errors::Error>
read_layout(const API &api, ec_master_t *master, const uint16_t position) {
    return coe::read_layout(
        sdo_reader(api, master, position),
        EC_MAX_SYNC_MANAGERS,
        coe::OnError::FAIL
    );
}

/// @brief returns the reason for ret, the negative error code of an IgH call.
std::string reason(const int ret) {
    return std::generic_category().message(-ret);
}

/// @brief the time between reads of the slave states while the slaves move to OP.
const auto OP_POLL_INTERVAL = x::telem::MILLISECOND * 10;

/// @brief returns the state that a slave in state moves to next on its way to OP.
std::string next_state(const slave::State state) {
    switch (state) {
        case slave::State::SAFE_OP:
            return "OP";
        case slave::State::PRE_OP:
            return "SAFE-OP";
        default:
            return "PRE-OP";
    }
}

/// @brief returns the IgH ring position, which counts from 0, of the slave at position.
uint16_t ring_position(const uint16_t position) {
    return position - 1;
}

/// @brief returns errors::SLAVE_CONFIG_ERROR when config enables slave in a mode that
/// the IgH master cannot run.
x::errors::Error
check_op_mode(const slave::Properties &slave, const slave::Config &config) {
    if (!config.enabled || !config.op_mode) return x::errors::NIL;
    const auto err = config.op_mode->assign_activate != 0
                       ? x::errors::Error(
                             errors::SLAVE_CONFIG_ERROR,
                             "the IgH master does not run DC sync"
                         )
                       : dc::check_cycles({}, config.op_mode->cycles);
    if (!err) return x::errors::NIL;
    return x::errors::Error(
        errors::SLAVE_CONFIG_ERROR,
        std::format(
            "slave {} ({}) cannot use its chosen mode: {}",
            slave.position,
            slave.name,
            err.data
        )
    );
}

/// @brief returns s as a decimal number, or nullopt when s holds anything else.
std::optional<unsigned int> parse_index(const std::string_view s) {
    unsigned int index = 0;
    const auto [end, ec] = std::from_chars(s.data(), s.data() + s.size(), index);
    if (ec != std::errc{} || end != s.data() + s.size()) return std::nullopt;
    return index;
}
}

Master::Master(std::shared_ptr<API> api, const unsigned int master_index, Config cfg):
    api(std::move(api)),
    master_index(master_index),
    cfg(std::move(cfg)),
    ec_master(nullptr),
    initialized(false),
    activated(false) {}

Master::~Master() {
    this->deactivate();
}

std::pair<std::vector<slave::DiscoveryResult>, x::errors::Error> Master::scan() {
    if (!this->initialized)
        if (auto err = this->initialize()) return {{}, err};
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
            return fail(std::format("failed to read slave {}: {}", i + 1, reason(ret)));
        slave::DiscoveryResult result{};
        result.properties.network = "igh:" + std::to_string(this->master_index);
        result.properties.position = static_cast<uint16_t>(i + 1);
        result.properties.vendor_id = slave_info.vendor_id;
        result.properties.product_code = slave_info.product_code;
        result.properties.revision = slave_info.revision_number;
        result.properties.serial = slave_info.serial_number;
        result.properties.name = slave_info.name;
        result.status.state = slave::State::INIT;
        for (uint8_t sm = 0; sm < slave_info.sync_count; ++sm)
            if (const auto err = this->append_scanned_pdos(result.properties, sm)) {
                result.properties.input_pdos.clear();
                result.properties.output_pdos.clear();
                result.status.pdo_discovery_error = "failed to read the PDO layout: " +
                                                    err.data;
                break;
            }
        esi::fill(result);
        auto [settings, settings_err] = esi::read_settings(
            slave_info.vendor_id,
            slave_info.product_code,
            slave_info.revision_number,
            result.properties.op_modes,
            sdo_reader(*this->api, this->ec_master, static_cast<uint16_t>(i))
        );
        result.status.settings_error = settings_err.data;
        result.properties.settings = std::move(settings);
        this->cached_slaves.push_back(std::move(result));
    }

    this->initialized = true;
    VLOG(1) << "[ethercat.igh] master " << this->master_index << " initialized with "
            << this->cached_slaves.size() << " slaves";
    return x::errors::NIL;
}

std::pair<master::Change, x::errors::Error> Master::changed() {
    if (!this->initialized)
        return {{}, x::errors::Error(errors::MASTER_INIT_ERROR, "not initialized")};
    ec_master_info_t info;
    if (this->api->master(this->ec_master, &info) < 0)
        return {
            {},
            x::errors::Error(errors::MASTER_INIT_ERROR, "failed to get master info")
        };
    if (info.slave_count != this->cached_slaves.size())
        return {master::Change{.stale = true}, x::errors::NIL};
    // IgH scans the bus again by itself, so its slave info is current.
    for (const auto &slave: this->cached_slaves) {
        const auto &props = slave.properties;
        ec_slave_info_t slave_info;
        if (const int ret = this->api->master_get_slave(
                this->ec_master,
                static_cast<uint16_t>(props.position - 1),
                &slave_info
            );
            ret != 0)
            return {
                {},
                x::errors::Error(
                    errors::MASTER_INIT_ERROR,
                    std::format(
                        "failed to read slave {}: {}",
                        props.position,
                        reason(ret)
                    )
                )
            };
        if (slave_info.vendor_id != props.vendor_id ||
            slave_info.product_code != props.product_code ||
            slave_info.revision_number != props.revision)
            return {master::Change{.stale = true}, x::errors::NIL};
    }
    return {{}, x::errors::NIL};
}

x::errors::Error
Master::download(const slave::Properties &slave, const slave::StartupParameter &param) {
    const auto data = param.bytes();
    uint32_t abort_code = 0;
    if (this->api->master_sdo_download(
            this->ec_master,
            ring_position(slave.position),
            param.index,
            param.sub_index,
            data.data(),
            data.size(),
            &abort_code
        ) >= 0)
        return x::errors::NIL;
    return x::errors::Error(
        errors::startup_parameter_error(abort_code),
        "slave " + std::to_string(slave.position) + " (" + slave.name +
            ") rejected startup parameter " + param.to_string() + ": " +
            errors::sdo_abort_reason(abort_code)
    );
}

x::errors::Error Master::write(
    const uint16_t position,
    const std::vector<slave::StartupParameter> &parameters
) {
    if (position == 0 || position > this->cached_slaves.size())
        return x::errors::Error(
            errors::STARTUP_PARAMETER_ERROR,
            "no slave at position " + std::to_string(position)
        );
    const auto &slave = this->cached_slaves[position - 1].properties;
    if (!this->activated) {
        for (const auto &param: parameters)
            if (auto err = this->download(slave, param)) return err;
        return x::errors::NIL;
    }
    for (const auto &param: parameters) {
        const auto want = param.bytes();
        std::vector<uint8_t> held(want.size());
        size_t size = 0;
        uint32_t abort_code = 0;
        if (this->api->master_sdo_upload(
                this->ec_master,
                ring_position(position),
                param.index,
                param.sub_index,
                held.data(),
                held.size(),
                &size,
                &abort_code
            ) < 0)
            return x::errors::Error(
                errors::startup_parameter_error(abort_code),
                "slave " + std::to_string(position) + " (" + slave.name +
                    ") failed to read startup parameter " + param.to_string() + ": " +
                    errors::sdo_abort_reason(abort_code)
            );
        if (size != want.size() || held != want)
            return x::errors::Error(
                errors::SETTING_CONFLICT,
                "the IgH master cannot change " + param.to_string() + " of slave " +
                    std::to_string(position) + " (" + slave.name +
                    ") while other tasks run the bus. Stop them, then start this "
                    "task first"
            );
    }
    return x::errors::NIL;
}

std::pair<uint32_t, x::errors::Error>
Master::read(const uint16_t position, const uint16_t index, const uint8_t sub_index) {
    return sdo_reader(*this->api, this->ec_master, ring_position(position))(
        index,
        sub_index
    );
}

x::errors::Error Master::add_startup_parameters(
    const slave::Properties &slave,
    ec_slave_config_t *sc,
    const std::vector<slave::StartupParameter> &parameters
) {
    for (const auto &param: parameters) {
        const auto data = param.bytes();
        // IgH reports a rejected config SDO only as a slave stuck in PRE-OP, so a
        // direct write first gets the abort code.
        if (auto err = this->download(slave, param)) return err;
        if (this->api->slave_config_sdo(
                sc,
                param.index,
                param.sub_index,
                data.data(),
                data.size()
            ) < 0)
            return x::errors::Error(
                errors::ACTIVATION_ERROR,
                "failed to add startup parameter " + param.to_string() + " to slave " +
                    std::to_string(slave.position) + " (" + slave.name + ")"
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
    const auto config_of = [&plan](const uint16_t pos) {
        const auto it = plan.slaves.find(pos);
        return it == plan.slaves.end() ? slave::Config{} : it->second;
    };
    for (const auto &slave: this->cached_slaves)
        if (auto err = check_op_mode(
                slave.properties,
                config_of(slave.properties.position)
            ))
            return {.err = err};
    master::Image image;
    for (const auto &slave: this->cached_slaves) {
        const auto pos = slave.properties.position;
        const auto config = config_of(pos);
        if (!config.enabled) continue;
        auto *sc = this->api->master_slave_config(
            this->ec_master,
            0,
            ring_position(pos),
            slave.properties.vendor_id,
            slave.properties.product_code
        );
        if (sc == nullptr)
            return {
                .err = x::errors::Error(
                    errors::ACTIVATION_ERROR,
                    "failed to configure slave " + std::to_string(pos) + " (" +
                        slave.properties.name + ")"
                )
            };
        this->slave_configs.emplace_back(pos, sc);
        const auto &params = config.startup_parameters;
        if (auto err = this->add_startup_parameters(slave.properties, sc, params))
            return {.err = err};
        if (!params.empty())
            if (auto err = this->configure_assigned_pdos(pos, sc)) return {.err = err};
        if (auto err = this->register_pdo_entries(slave.properties, sc, image))
            return {.err = err};
    }

    const size_t size = this->domains.empty() ? 0 : this->domains.back().end();
    VLOG(1) << "[ethercat.igh] activating master " << this->master_index << " with "
            << this->slave_configs.size() << " configured slaves and "
            << this->domains.size() << " domains of " << size << " bytes";

    if (this->api->master_activate(this->ec_master) < 0)
        return {
            .err = x::errors::Error(
                errors::ACTIVATION_ERROR,
                "ecrt_master_activate failed"
            )
        };

    for (auto &domain: this->domains)
        domain.data = this->api->domain_data(domain.handle);
    this->inputs.assign(size, 0);
    this->outputs.assign(size, 0);
    this->latest_inputless.reserve(this->domains.size());

    this->activated = true;
    LOG(INFO) << "[ethercat.igh] master " << this->master_index
              << " activated successfully";

    image.inputs = this->inputs;
    image.outputs = this->outputs;
    image.slaves = this->cached_slaves;
    // The IgH master moves the slaves to OP on its own while the caller cycles.
    start_cycle(image);
    return this->wait_for_op();
}

master::Activation Master::wait_for_op() {
    const auto deadline = this->cfg.clock.steady() + this->cfg.op_timeout.chrono();
    while (true) {
        std::vector<uint16_t> pending;
        for (const auto &[pos, sc]: this->slave_configs) {
            ec_slave_config_state_t state{};
            if (this->api->slave_config_state(sc, &state) != 0 ||
                state.operational == 0)
                pending.push_back(pos);
        }
        if (pending.empty()) return {};
        if (this->cfg.clock.steady() >= deadline) return this->op_error(pending);
        this->cfg.clock.sleep(OP_POLL_INTERVAL);
    }
}

master::Activation Master::op_error(const std::vector<uint16_t> &pending) {
    master::Activation activation;
    std::string msg;
    for (const auto pos: pending) {
        const auto &name = this->cached_slaves[pos - 1].properties.name;
        ec_slave_info_t info{};
        if (const int ret = this->api->master_get_slave(
                this->ec_master,
                ring_position(pos),
                &info
            );
            ret != 0) {
            if (!msg.empty()) msg += "; ";
            msg += std::format(
                "slave {} ({}): failed to read its state: {}",
                pos,
                name,
                reason(ret)
            );
            continue;
        }
        const auto state = slave::from_al_state(info.al_state);
        // The slave can reach OP after the last poll and before this read.
        if (state == slave::State::OP) continue;
        if (!msg.empty()) msg += "; ";
        msg += std::format(
            "slave {} ({}) is in {}",
            pos,
            name,
            slave::to_string(state)
        );
        if (info.error_flag == 0) continue;
        msg += " with an AL error";
        activation.refusals[pos] = {.reason = "refused " + next_state(state)};
    }
    if (msg.empty()) return activation;
    activation.err = x::errors::Error(
        activation.refusals.empty() ? errors::STATE_CHANGE_ERROR
                                    : errors::STATE_CHANGE_REFUSED,
        "failed to reach OP within " + this->cfg.op_timeout.to_string() + ": " + msg
    );
    return activation;
}

void Master::deactivate() {
    if (!this->initialized) return;

    VLOG(1) << "[ethercat.igh] master " << this->master_index << " deactivating";
    if (this->activated)
        if (const int ret = this->api->master_deactivate(this->ec_master); ret != 0)
            LOG(WARNING) << "[ethercat.igh] master " << this->master_index
                         << " failed to deactivate: " << reason(ret);
    if (this->ec_master) {
        this->api->release_master(this->ec_master);
        this->ec_master = nullptr;
    }
    this->activated = false;
    this->initialized = false;
    this->slave_configs.clear();
    this->cached_slaves.clear();
    this->domains.clear();
    this->inputs.clear();
    this->outputs.clear();
    this->latest_health.clear();
    this->latest_shortfall.reset();
    this->latest_inputless.clear();
}

x::errors::Error Master::receive() {
    this->latest_shortfall.reset();
    this->latest_inputless.clear();
    if (!this->activated)
        return x::errors::Error(errors::CYCLIC_ERROR, "not activated");

    if (const int ret = this->api->master_receive(this->ec_master); ret != 0)
        return x::errors::Error(
            errors::CYCLIC_ERROR,
            "failed to receive frames: " + reason(ret)
        );
    uint32_t working_counter = 0;
    uint32_t expected = 0;
    size_t silent = 0;
    for (const auto &domain: this->domains) {
        if (const int ret = this->api->domain_process(domain.handle); ret != 0)
            return x::errors::Error(
                errors::CYCLIC_ERROR,
                "failed to process a domain: " + reason(ret)
            );
        ec_domain_state_t state{};
        if (const int ret = this->api->domain_state(domain.handle, &state); ret != 0)
            return x::errors::Error(
                errors::CYCLIC_ERROR,
                "failed to read a domain state: " + reason(ret)
            );
        std::memcpy(this->inputs.data() + domain.offset, domain.data, domain.size);
        const uint32_t full = domain.share.inputs + domain.share.outputs;
        expected += full;
        // IgH splits a domain too large for one datagram, which can change the count it
        // expects, so its own state decides whether the slave took full part.
        if (state.wc_state == EC_WC_COMPLETE) {
            working_counter += full;
            continue;
        }
        working_counter += std::min<uint32_t>(state.working_counter, full);
        if (state.wc_state != EC_WC_ZERO) continue;
        silent++;
        if (domain.share.inputs > 0) this->latest_inputless.push_back(domain.position);
    }
    if (working_counter == expected) return x::errors::NIL;
    this->latest_shortfall = expected - working_counter;
    if (silent == this->domains.size()) {
        this->latest_inputless.clear();
        return x::errors::Error(errors::WORKING_COUNTER_ERROR, "no slaves responded");
    }
    return x::errors::Error(
        errors::WORKING_COUNTER_ERROR,
        std::format("incomplete working counter: {} of {}", working_counter, expected)
    );
}

x::errors::Error Master::send() {
    if (!this->activated)
        return x::errors::Error(errors::CYCLIC_ERROR, "not activated");

    for (const auto &domain: this->domains) {
        std::memcpy(domain.data, this->outputs.data() + domain.offset, domain.size);
        if (const int ret = this->api->domain_queue(domain.handle); ret != 0)
            return x::errors::Error(
                errors::CYCLIC_ERROR,
                "failed to queue a domain: " + reason(ret)
            );
    }
    if (const int ret = this->api->master_send(this->ec_master); ret != 0)
        return x::errors::Error(
            errors::CYCLIC_ERROR,
            "failed to send frames: " + reason(ret)
        );
    return x::errors::NIL;
}

std::vector<slave::Health> Master::check(const x::breaker::Breaker &) {
    ec_master_info_t info{};
    const bool scanning = this->api->master(this->ec_master, &info) == 0 &&
                          info.scan_busy != 0;
    std::vector<slave::Health> health;
    health.reserve(this->slave_configs.size());
    for (const auto &[pos, sc]: this->slave_configs) {
        if (scanning && !this->silent(pos))
            if (const auto prev = std::ranges::find(
                    this->latest_health,
                    pos,
                    &slave::Health::position
                );
                prev != this->latest_health.end()) {
                health.push_back(*prev);
                continue;
            }
        ec_slave_config_state_t state{};
        ec_slave_info_t slave_info{};
        // A slave the master cannot read is not responding.
        const bool read = this->api->slave_config_state(sc, &state) == 0 &&
                          this->api->master_get_slave(
                              this->ec_master,
                              ring_position(pos),
                              &slave_info
                          ) == 0;
        health.push_back({
            .position = pos,
            .state = slave::from_al_state(state.al_state),
            .faulted = read && slave_info.error_flag != 0,
            .lost = !read || state.online == 0,
        });
    }
    this->latest_health = health;
    return health;
}

bool Master::silent(const uint16_t position) const {
    const auto domain = std::ranges::find(this->domains, position, &Domain::position);
    if (domain == this->domains.end()) return false;
    ec_domain_state_t state{};
    return this->api->domain_state(domain->handle, &state) != 0 ||
           state.wc_state == EC_WC_ZERO;
}

x::errors::Error
Master::configure_assigned_pdos(const uint16_t position, ec_slave_config_t *sc) {
    auto props = this->cached_slaves[position - 1].properties;
    const auto slave_name = "slave " + std::to_string(position) + " (" + props.name +
                            ")";
    const auto [syncs, err] = read_layout(
        *this->api,
        this->ec_master,
        ring_position(position)
    );
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
    std::vector<ec_sync_info_t> sync_infos;
    std::vector<uint8_t> empty_syncs;
    std::vector<uint16_t> empty_pdos;
    // A direction with no assignable sync manager keeps the PDOs found at discovery.
    if (assignable(true)) props.input_pdos.clear();
    if (assignable(false)) props.output_pdos.clear();
    for (const auto &sync: syncs) {
        if (sync.fixed) {
            if (!assignable(sync.is_input)) continue;
            if (const auto scan_err = this->append_scanned_pdos(props, sync.index))
                return x::errors::Error(
                    errors::PDO_DISCOVERY_ERROR,
                    "failed to read the PDO layout of " + slave_name + ": " +
                        scan_err.data
                );
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
    this->cached_slaves[position - 1].properties = std::move(props);
    return x::errors::NIL;
}

x::errors::Error Master::register_pdo_entries(
    const slave::Properties &slave,
    ec_slave_config_t *sc,
    master::Image &image
) {
    const auto mapped = [](const auto &pdos) {
        return std::ranges::any_of(pdos, [](const pdo::Properties &pdo) {
            return !pdo.is_padding();
        });
    };
    if (!mapped(slave.output_pdos) && !mapped(slave.input_pdos)) return x::errors::NIL;
    auto *handle = this->api->master_create_domain(this->ec_master);
    if (handle == nullptr)
        return x::errors::Error(
            errors::ACTIVATION_ERROR,
            std::format(
                "failed to create the domain of slave {} ({})",
                slave.position,
                slave.name
            )
        );
    const size_t base = this->domains.empty() ? 0 : this->domains.back().end();
    bool has_outputs = false;
    bool has_inputs = false;
    const auto add = [&](const pdo::Properties &pdo) {
        if (pdo.is_padding()) return;
        const pdo::Key key{slave.position, pdo.index, pdo.sub_index, pdo.is_input};
        unsigned int bit = 0;
        const int offset = this->api->slave_config_reg_pdo_entry(
            sc,
            pdo.index,
            pdo.sub_index,
            handle,
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
                        reason(offset)
                    )
                )
            );
            return;
        }
        (pdo.is_input ? has_inputs : has_outputs) = true;
        image.offsets[key] = {
            base + static_cast<size_t>(offset),
            static_cast<uint8_t>(bit),
            pdo.bit_length
        };
    };
    for (const auto &pdo: slave.output_pdos)
        add(pdo);
    for (const auto &pdo: slave.input_pdos)
        add(pdo);
    if (!has_outputs && !has_inputs) return x::errors::NIL;
    // IgH reads and writes a domain with both directions in one LRW datagram, which a
    // slave adds 2 to when it takes its outputs. A domain with only outputs is an LWR
    // datagram, which the slave adds 1 to.
    master::WorkingCounterShare share{.inputs = has_inputs ? 1U : 0U};
    if (has_outputs) share.outputs = has_inputs ? 2U : 1U;
    this->domains.push_back({
        .position = slave.position,
        .handle = handle,
        .offset = base,
        .size = this->api->domain_size(handle),
        .share = share,
    });
    image.working_counter_shares[slave.position] = share;
    return x::errors::NIL;
}

x::errors::Error
Master::append_scanned_pdos(slave::Properties &props, const uint8_t sync_index) {
    const auto pos = ring_position(props.position);
    ec_sync_info_t sync_info{};
    if (const int ret = this->api->master_get_sync_manager(
            this->ec_master,
            pos,
            sync_index,
            &sync_info
        );
        ret != 0)
        return x::errors::Error(
            errors::PDO_DISCOVERY_ERROR,
            std::format("failed to read sync manager {}: {}", sync_index, reason(ret))
        );
    const bool is_input = sync_info.dir == EC_DIR_INPUT;
    for (unsigned int pdo_pos = 0; pdo_pos < sync_info.n_pdos; ++pdo_pos) {
        ec_pdo_info_t pdo_info{};
        if (const int ret = this->api->master_get_pdo(
                this->ec_master,
                pos,
                sync_index,
                static_cast<uint16_t>(pdo_pos),
                &pdo_info
            );
            ret != 0)
            return x::errors::Error(
                errors::PDO_DISCOVERY_ERROR,
                std::format(
                    "failed to read PDO {} of sync manager {}: {}",
                    pdo_pos,
                    sync_index,
                    reason(ret)
                )
            );
        pdo::PDO pdo{.index = pdo_info.index, .entries = {}};
        for (unsigned int entry_pos = 0; entry_pos < pdo_info.n_entries; ++entry_pos) {
            ec_pdo_entry_info_t entry{};
            if (const int ret = this->api->master_get_pdo_entry(
                    this->ec_master,
                    pos,
                    sync_index,
                    static_cast<uint16_t>(pdo_pos),
                    static_cast<uint16_t>(entry_pos),
                    &entry
                );
                ret != 0)
                return x::errors::Error(
                    errors::PDO_DISCOVERY_ERROR,
                    std::format(
                        "failed to read entry {} of PDO {:#06x}: {}",
                        entry_pos,
                        pdo_info.index,
                        reason(ret)
                    )
                );
            pdo.entries.push_back({
                .index = entry.index,
                .sub_index = entry.subindex,
                .bit_length = entry.bit_length,
            });
        }
        pdo::append(is_input ? props.input_pdos : props.output_pdos, is_input, pdo);
    }
    return x::errors::NIL;
}

std::pair<std::unique_ptr<Manager>, x::errors::Error> Manager::open() {
    struct stat st{};
    if (stat(DEVICE_PATH, &st) != 0)
        return {
            nullptr,
            x::errors::Error(errors::MASTER_INIT_ERROR, "IgH device not found")
        };
    auto [api, err] = API::load();
    if (err) return {nullptr, err};
    return {std::make_unique<Manager>(std::move(api)), x::errors::NIL};
}

std::vector<master::Info> Manager::enumerate() {
    constexpr std::string_view prefix = "EtherCAT";
    std::vector<master::Info> masters;
    DIR *dir = opendir(SYSFS_PATH);
    if (dir == nullptr) return masters;
    while (const dirent *entry = readdir(dir)) {
        const std::string_view name = entry->d_name;
        if (!name.starts_with(prefix)) continue;
        const auto index = parse_index(name.substr(prefix.size()));
        if (!index) continue;
        masters.push_back({
            .key = "igh:" + std::to_string(*index),
            .description = "IgH EtherCAT Master " + std::to_string(*index),
        });
    }
    closedir(dir);
    return masters;
}

std::pair<std::shared_ptr<master::Master>, x::errors::Error>
Manager::create(const std::string &key) {
    constexpr std::string_view prefix = "igh:";
    const auto index = key.starts_with(prefix)
                         ? parse_index(std::string_view(key).substr(prefix.size()))
                         : std::nullopt;
    if (!index)
        return {
            nullptr,
            x::errors::Error(
                errors::MASTER_INIT_ERROR,
                "invalid IgH master key '" + key + "': expected format 'igh:N'"
            )
        };
    return {std::make_shared<Master>(this->api, *index), x::errors::NIL};
}
}
