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
#include <chrono>
#include <format>
#include <map>
#include <numeric>
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
#include "driver/ethercat/master/dc.h"
#include "driver/ethercat/sii/sii.h"
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

/// @brief reads the DC modes from the SII of the slave at ring position.
/// @returns the modes and 0, or the negative error code of a failed read.
std::pair<std::vector<dc::OpMode>, int> read_op_modes(
    const API &api,
    const unsigned int master_index,
    const uint16_t position
) {
    std::map<uint16_t, std::vector<uint8_t>> categories;
    const auto read = [&](const uint32_t offset, std::span<uint16_t> words) {
        // IgH rejects a read of no words.
        if (words.empty()) return 0;
        // A read past the last word that an offset can address ends the SII, as a read
        // past the end of the SII does.
        if (offset + words.size() > UINT16_MAX) return -EINVAL;
        return api.sii_read(master_index, position, offset, words);
    };
    for (uint32_t offset = sii::CATEGORIES;;) {
        std::array<uint16_t, 2> header{};
        int ret = read(offset, header);
        const auto [type, size] = header;
        if (ret == 0 && type == sii::END) break;
        std::vector<uint16_t> words;
        if (ret == 0 && (type == sii::STRINGS || type == sii::DC)) {
            words.resize(size);
            ret = read(offset + 2, words);
        }
        // IgH rejects a read past the end of the SII with EINVAL, so a SII with no end
        // marker ends at its last whole category.
        if (ret == -EINVAL) break;
        if (ret != 0) return {{}, ret};
        for (const auto word: words) {
            categories[type].push_back(static_cast<uint8_t>(word & 0xFF));
            categories[type].push_back(static_cast<uint8_t>(word >> 8));
        }
        offset += 2 + size;
    }
    return {
        sii::parse_op_modes(
            categories[sii::DC],
            sii::parse_strings(categories[sii::STRINGS])
        ),
        0
    };
}

/// @brief returns errors::SLAVE_CONFIG_ERROR when IgH would start the SYNC0 events of
/// slave off the whole SYNC0 cycles after master::DC_EPOCH that the engine expects. IgH
/// starts SYNC0 on whole spans of the SYNC0 cycle plus the SYNC1 register.
x::errors::Error
check_sync1(const slave::Properties &slave, const dc::Registers &regs) {
    if (!regs.marks_data() || regs.sync1_cycle % regs.sync0_cycle == 0)
        return x::errors::NIL;
    return x::errors::Error(
        errors::SLAVE_CONFIG_ERROR,
        std::format(
            "slave {} ({}) cannot run its DC mode on the IgH master: SYNC1 comes {} "
            "after SYNC0, which is not a whole number of SYNC0 cycles of {}",
            slave.position,
            slave.name,
            x::telem::TimeSpan(regs.sync1_cycle).to_string(),
            x::telem::TimeSpan(regs.sync0_cycle).to_string()
        )
    );
}

/// @brief returns the latest application time at or before now that is a whole number
/// of SYNC0 cycles of each slave in dc_sync whose SYNC0 events mark data.
/// @returns errors::SLAVE_CONFIG_ERROR when the cycles have no common multiple at or
/// before now.
std::pair<uint64_t, x::errors::Error>
first_app_time(const uint64_t now, const std::vector<dc::Registers> &dc_sync) {
    uint64_t multiple = 1;
    for (const auto &regs: dc_sync) {
        if (!regs.marks_data()) continue;
        const uint64_t cycle = regs.sync0_cycle;
        const auto factor = multiple / std::gcd(multiple, cycle);
        if (factor > now / cycle)
            return {
                0,
                x::errors::Error(
                    errors::SLAVE_CONFIG_ERROR,
                    "the SYNC0 cycles of the slaves have no common multiple that the "
                    "IgH master can start them on"
                )
            };
        multiple = factor * cycle;
    }
    return {now / multiple * multiple, x::errors::NIL};
}

/// @brief returns t in nanoseconds since the steady clock epoch.
int64_t nanoseconds(const hs_clock::time_point t) {
    return std::chrono::duration_cast<std::chrono::nanoseconds>(t.time_since_epoch())
        .count();
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
        // IgH reads DC receive times only from slaves with a distributed clock, and it
        // makes the first of them the reference clock.
        if (!this->reference_clock && slave_info.ports[0].receive_time != 0)
            this->reference_clock = result.properties.position;
        for (uint8_t sm = 0; sm < slave_info.sync_count; ++sm)
            if (const auto err = this->append_scanned_pdos(result.properties, sm)) {
                result.properties.input_pdos.clear();
                result.properties.output_pdos.clear();
                result.status.pdo_discovery_error = "failed to read the PDO layout: " +
                                                    err.data;
                break;
            }
        esi::fill(result);
        auto [modes, ret] = read_op_modes(
            *this->api,
            this->master_index,
            static_cast<uint16_t>(i)
        );
        if (ret != 0)
            return fail(
                std::format(
                    "failed to read the SII of slave {}: {}",
                    i + 1,
                    reason(ret)
                )
            );
        result.properties.op_modes = std::move(modes);
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

    this->output_domain = this->api->master_create_domain(this->ec_master);
    if (!this->output_domain) return fail("failed to create output domain");
    this->input_domain = this->api->master_create_domain(this->ec_master);
    if (!this->input_domain) return fail("failed to create input domain");

    this->initialized = true;
    VLOG(1) << "[ethercat.igh] master " << this->master_index << " initialized with "
            << this->cached_slaves.size() << " slaves";
    return x::errors::NIL;
}

uint64_t Master::app_time() const {
    return static_cast<uint64_t>(
        nanoseconds(this->cfg.clock.steady()) + this->app_offset
    );
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
    auto [dc_setups, dc_err] = master::resolve_dc(this->cached_slaves, plan);
    if (dc_err) return {.err = dc_err};
    master::Image image;
    for (const auto &[pos, regs]: dc_setups) {
        if (auto err = check_sync1(this->cached_slaves[pos - 1].properties, regs))
            return {.err = err};
        if (regs.sync0_cycle > 0 && (regs.assign_activate & dc::PDI_SYNC_UNIT) == 0)
            image.dc_sync.push_back(regs);
    }
    this->app_offset = (this->cfg.clock.now() - master::DC_EPOCH).nanoseconds() -
                       nanoseconds(this->cfg.clock.steady());
    auto [first_app, app_err] = first_app_time(this->app_time(), image.dc_sync);
    if (app_err) return {.err = app_err};
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
        this->register_pdo_entries(slave.properties, sc, image);
        const auto setup = std::ranges::find(
            dc_setups,
            pos,
            &master::DCSetup::position
        );
        if (setup == dc_setups.end()) continue;
        const auto &regs = setup->registers;
        if (const int ret = this->api->slave_config_dc(
                sc,
                regs.assign_activate,
                regs.sync0_cycle,
                regs.sync0_shift,
                regs.sync1_cycle,
                0
            );
            ret != 0)
            return {
                .err = x::errors::Error(
                    errors::ACTIVATION_ERROR,
                    std::format(
                        "failed to set up the DC sync of slave {} ({}): {}",
                        pos,
                        slave.properties.name,
                        reason(ret)
                    )
                )
            };
    }
    this->dc_setups = std::move(dc_setups);
    // The first application time sets where IgH starts the SYNC0 events of each slave.
    if (!this->dc_setups.empty()) {
        if (const int ret = this->api
                                ->master_application_time(this->ec_master, first_app);
            ret != 0)
            return {
                .err = x::errors::Error(
                    errors::ACTIVATION_ERROR,
                    "failed to give the application time: " + reason(ret)
                )
            };
        image.reference_clock = this->reference_clock;
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

    this->activated = true;
    LOG(INFO) << "[ethercat.igh] master " << this->master_index
              << " activated successfully";
    VLOG(1) << "[ethercat.igh] output_domain_data="
            << static_cast<void *>(this->output_domain_data)
            << ", input_domain_data=" << static_cast<void *>(this->input_domain_data)
            << ", output_sz=" << this->output_sz << ", input_sz=" << this->input_sz;

    image.inputs = {this->input_domain_data, this->input_sz};
    image.outputs = {this->output_domain_data, this->output_sz};
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
    this->full_working_counter.reset();
    this->latest_shortfall.reset();
    this->reference_clock.reset();
    this->dc_setups.clear();
    this->clocks_aligned = false;
    this->latest_dc.reset();
}

x::errors::Error Master::receive() {
    this->latest_shortfall.reset();
    if (!this->activated)
        return x::errors::Error(errors::CYCLIC_ERROR, "not activated");

    if (const int ret = this->api->master_receive(this->ec_master); ret != 0)
        return x::errors::Error(
            errors::CYCLIC_ERROR,
            "failed to receive frames: " + reason(ret)
        );
    for (const auto &[domain, state]:
         {std::pair{this->output_domain, &this->output_domain_state},
          std::pair{this->input_domain, &this->input_domain_state}}) {
        if (const int ret = this->api->domain_process(domain); ret != 0)
            return x::errors::Error(
                errors::CYCLIC_ERROR,
                "failed to process a domain: " + reason(ret)
            );
        if (const int ret = this->api->domain_state(domain, state); ret != 0)
            return x::errors::Error(
                errors::CYCLIC_ERROR,
                "failed to read a domain state: " + reason(ret)
            );
    }

    // IgH reports a zero working counter for an empty domain, which exchanges no data.
    const bool zero = (this->output_sz > 0 &&
                       this->output_domain_state.wc_state == EC_WC_ZERO) ||
                      (this->input_sz > 0 &&
                       this->input_domain_state.wc_state == EC_WC_ZERO);
    const bool incomplete = this->output_domain_state.wc_state == EC_WC_INCOMPLETE ||
                            this->input_domain_state.wc_state == EC_WC_INCOMPLETE;
    const uint32_t wkc = this->output_domain_state.working_counter +
                         this->input_domain_state.working_counter;
    // IgH aligns the slave clocks to the application time before any slave reaches
    // SAFE-OP and gives a working counter.
    if (wkc > 0) this->clocks_aligned = true;
    if (!this->dc_setups.empty() && this->clocks_aligned) {
        uint32_t dc = 0;
        const int ret = this->api->master_reference_clock_time(this->ec_master, &dc);
        if (ret == 0)
            this->latest_dc = master::DC_EPOCH +
                              x::telem::TimeSpan(
                                  static_cast<int64_t>(this->sent_at) +
                                  static_cast<int32_t>(
                                      dc - static_cast<uint32_t>(this->sent_at)
                                  )
                              );
        // A frame that did not reach the reference clock carries no time (EIO), and IgH
        // has no reference clock while it scans the bus (ENXIO).
        else if (ret != -EIO && ret != -ENXIO)
            return x::errors::Error(
                errors::CYCLIC_ERROR,
                "failed to read the reference clock: " + reason(ret)
            );
    }
    if (!zero && !incomplete) {
        this->full_working_counter = wkc;
        return x::errors::NIL;
    }
    if (this->full_working_counter && wkc < *this->full_working_counter)
        this->latest_shortfall = *this->full_working_counter - wkc;
    if (zero)
        return x::errors::Error(errors::WORKING_COUNTER_ERROR, "no slaves responded");
    return x::errors::Error(
        errors::WORKING_COUNTER_ERROR,
        "incomplete working counter: output " +
            std::to_string(this->output_domain_state.working_counter) + ", input " +
            std::to_string(this->input_domain_state.working_counter)
    );
}

x::errors::Error Master::send() {
    if (!this->activated)
        return x::errors::Error(errors::CYCLIC_ERROR, "not activated");
    if (!this->dc_setups.empty()) {
        this->sent_at = this->app_time();
        if (const int ret = this->api->master_application_time(
                this->ec_master,
                this->sent_at
            );
            ret != 0)
            return x::errors::Error(
                errors::CYCLIC_ERROR,
                "failed to give the application time: " + reason(ret)
            );
        // IgH has no reference clock while it scans the bus.
        for (const int ret:
             {this->api->master_sync_reference_clock_to(this->ec_master, this->sent_at),
              this->api->master_sync_slave_clocks(this->ec_master)})
            if (ret != 0 && ret != -ENXIO)
                return x::errors::Error(
                    errors::CYCLIC_ERROR,
                    "failed to sync the distributed clocks: " + reason(ret)
                );
    }

    for (auto *domain: {this->output_domain, this->input_domain})
        if (const int ret = this->api->domain_queue(domain); ret != 0)
            return x::errors::Error(
                errors::CYCLIC_ERROR,
                "failed to queue a domain: " + reason(ret)
            );
    if (const int ret = this->api->master_send(this->ec_master); ret != 0)
        return x::errors::Error(
            errors::CYCLIC_ERROR,
            "failed to send frames: " + reason(ret)
        );
    return x::errors::NIL;
}

std::vector<slave::Health> Master::check(const x::breaker::Breaker &) {
    std::vector<slave::Health> health;
    health.reserve(this->slave_configs.size());
    for (const auto &[pos, sc]: this->slave_configs) {
        ec_slave_config_state_t state{};
        ec_slave_info_t info{};
        // A slave the master cannot read is not responding.
        const bool read = this->api->slave_config_state(sc, &state) == 0 &&
                          this->api->master_get_slave(
                              this->ec_master,
                              ring_position(pos),
                              &info
                          ) == 0;
        health.push_back({
            .position = pos,
            .state = slave::from_al_state(state.al_state),
            .faulted = read && info.error_flag != 0,
            .lost = !read || state.online == 0,
        });
    }
    return health;
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

void Master::register_pdo_entries(
    const slave::Properties &slave,
    ec_slave_config_t *sc,
    master::Image &image
) {
    // A slave adds 1 to the datagram of each domain that holds one of its entries.
    master::WorkingCounterShare share;
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
                        reason(offset)
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
        if (domain == this->output_domain)
            share.outputs = 1;
        else
            share.inputs = 1;
    };
    for (const auto &pdo: slave.output_pdos)
        add(pdo, this->output_domain);
    for (const auto &pdo: slave.input_pdos)
        add(pdo, this->input_domain);
    if (share != master::WorkingCounterShare{})
        image.working_counter_shares[slave.position] = share;
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
