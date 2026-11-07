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
#include <chrono>
#include <format>
#include <functional>
#include <iterator>
#include <string_view>
#include <thread>

#include "absl/log/log.h"

#include "x/cpp/binary/binary.h"

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/esi/known_devices.h"
#include "driver/ethercat/slave/slave.h"
#include "driver/ethercat/soem/master.h"
#include "driver/ethercat/soem/util.h"
#include "driver/ethercat/telem/telem.h"

namespace driver::ethercat::soem {
/// Timeout for cyclic process data exchange in microseconds.
const int PROCESSDATA_TIMEOUT = static_cast<int>(x::telem::MILLISECOND.microseconds());
/// Time between state reads while slaves move to a requested state.
const auto STATE_POLL_INTERVAL = x::telem::MILLISECOND;
/// AL status codes of faults that a new start can clear. The start takes every slave
/// back through INIT with an acknowledgment.
constexpr std::array<uint16_t, 16> TRANSIENT_FAULTS = {
    0x0018, // No valid inputs available
    0x0019, // No valid outputs
    0x001A, // Synchronization error
    0x001B, // Sync manager watchdog
    0x0021, // Slave needs INIT
    0x0022, // Slave needs PREOP
    0x0023, // Slave needs SAFEOP
    0x002A, // Background watchdog
    0x002B, // No valid inputs and outputs
    0x002C, // Fatal sync error
    0x002D, // No sync error
    0x0032, // PLL error
    0x0033, // DC sync IO error
    0x0034, // DC sync timeout error
    0x0052, // External hardware not ready
    0x0060, // Slave restarted locally
};
/// Timeout for one recovery step on a slave in microseconds.
const int RECOVERY_TIMEOUT = static_cast<int>(
    (x::telem::MILLISECOND * 500).microseconds()
);
/// Time from the read of a slave's DC system time to its first SYNC0 event, so the
/// slave takes the start time before the event passes.
const auto DC_START_DELAY = x::telem::MILLISECOND * 100;
/// The largest system time difference (0x092C) of a settled slave clock.
const auto DC_SETTLED_DIFF = x::telem::MICROSECOND * 10;
/// Time between reads of the system time differences while the clocks settle.
const auto DC_SETTLE_POLL_INTERVAL = x::telem::MILLISECOND * 10;
namespace {
/// @brief converts a span to the microseconds that SOEM timeouts take.
int micros(const x::telem::TimeSpan span) {
    return static_cast<int>(span.microseconds());
}

}

/// SOEM group 0 addresses every slave, so enabled and disabled slaves get their own.
constexpr uint8_t ENABLED_GROUP = 1;
constexpr uint8_t DISABLED_GROUP = 2;
/// AL status code for "Invalid DC SYNC configuration".
constexpr uint16_t AL_INVALID_DC_SYNC = 0x0030;
/// IOMap buffer size for process data image. This is local memory only - actual wire
/// traffic is determined by slave PDO mappings. 128KB handles networks with hundreds
/// of slaves or slaves with large PDO mappings (e.g., multi-axis drives).
constexpr size_t DEFAULT_IOMAP_SIZE = 131072;

/// @name EtherCAT Object Dictionary Standard Addresses
/// These are defined by the EtherCAT specification (ETG.1000.6) and are the standard
/// locations for PDO mapping objects in any compliant EtherCAT slave.
/// @{

/// First TxPDO (slave->master) mapping object index. Range: 0x1A00-0x1BFF.
constexpr uint16_t TXPDO_MAPPING_START = 0x1A00;
/// Last TxPDO mapping object index.
constexpr uint16_t TXPDO_MAPPING_END = 0x1BFF;
/// First RxPDO (master->slave) mapping object index. Range: 0x1600-0x17FF.
constexpr uint16_t RXPDO_MAPPING_START = 0x1600;
/// Last RxPDO mapping object index.
constexpr uint16_t RXPDO_MAPPING_END = 0x17FF;

/// @}

namespace {
std::string no_slaves_message(API &api, const std::string &iface) {
    const auto found = "no EtherCAT slaves found on " + iface + ". ";
    if (!api.dewesoft_filter_bound(iface.c_str()))
        return found + "Check the cable and the power of the first device.";
    return found + std::format(
                       "The DewesoftX \"EthercatDAQ Filter\" is on adapter \"{0}\". "
                       "After DewesoftX starts, the filter blocks EtherCAT replies, "
                       "also after DewesoftX closes. Clear it in the properties of "
                       "\"{0}\" (Settings > Network & internet > Advanced network "
                       "settings > More network adapter options, then right-click the "
                       "adapter > Properties).",
                       api.adapter_name(iface.c_str()).value_or(iface)
                   );
}
}

Master::Master(std::unique_ptr<API> api, std::string interface_name, const Config cfg):
    iface_name(std::move(interface_name)),
    api(std::move(api)),
    cfg(cfg),
    iomap(DEFAULT_IOMAP_SIZE, 0),
    input_sz(0),
    output_sz(0),
    initialized(false),
    activated(false),
    expected_wkc(0) {}

// NOLINTNEXTLINE(bugprone-exception-escape): its strings throw only past max_size.
Master::~Master() {
    if (this->activated || this->initialized) this->deactivate();
}

x::errors::Error Master::initialize() {
    if (this->initialized) return x::errors::NIL;

    if (auto err = this->api->check_socket(this->iface_name.c_str())) return err;

    if (this->api->init(this->iface_name.c_str()) <= 0) {
        this->api->close();
        return x::errors::Error(
            errors::MASTER_INIT_ERROR,
            "cannot open " + this->iface_name +
                " for raw Ethernet access. Run the Driver as root (or with "
                "CAP_NET_RAW on Linux), and install Npcap on Windows."
        );
    }

    if (this->api->config_init() <= 0) {
        this->api->close();
        return x::errors::Error(
            errors::NO_SLAVES_FOUND,
            no_slaves_message(*this->api, this->iface_name)
        );
    }

    // config_init requests PRE-OP without waiting for it, and SOEM sends no mailbox
    // request to a slave whose last read state is below PRE-OP.
    this->api->statecheck(0, EC_STATE_PRE_OP, micros(this->cfg.pre_op_timeout));
    this->api->readstate();
    this->populate_slaves();
    this->initialized = true;
    return x::errors::NIL;
}

std::pair<size_t, x::errors::Error> Master::count_slaves() {
    if (!this->initialized)
        return {
            0,
            x::errors::Error(errors::MASTER_INIT_ERROR, "master not initialized")
        };
    const int count = this->api->count_bus_slaves();
    return {count > 0 ? static_cast<size_t>(count) : 0, x::errors::NIL};
}

void Master::replace_slave_configs(std::map<uint16_t, slave::Config> configs) {
    std::lock_guard lock(this->mu);
    this->disabled_slaves.clear();
    this->chosen_modes.clear();
    for (auto &[position, config]: configs) {
        if (!config.enabled) this->disabled_slaves.insert(position);
        if (config.op_mode) this->chosen_modes[position] = *config.op_mode;
    }
}

x::errors::Error Master::activate(
    const x::telem::TimeSpan period,
    const std::function<void()> &start_cycle
) {
    if (!this->initialized)
        return x::errors::Error(errors::ACTIVATION_ERROR, "master not initialized");
    if (this->activated)
        return x::errors::Error(errors::ACTIVATION_ERROR, "master already activated");
    {
        std::lock_guard lock(this->mu);
        this->start_refusals.clear();
        this->start_dc_sync_refused = false;
    }
    auto [dc_setups, dc_err] = this->resolve_dc(period);
    if (dc_err) return dc_err;
    this->dc_setups = std::move(dc_setups);

    {
        std::lock_guard lock(this->mu);
        for (int i = 1; i <= this->api->slave_count(); ++i)
            this->api->set_slave_group(
                i,
                this->disabled_slaves.contains(i) || this->left_out.contains(i)
                    ? DISABLED_GROUP
                    : ENABLED_GROUP
            );
    }

    if (const auto err = this->clear_dc_activation()) return err;

    // The master requests SAFE-OP itself, once the bus cycles and SYNC0 runs.
    this->api->set_manual_state_change(true);
    // SOEM keeps the SII length of a sync manager whose PDO assignment is empty or that
    // 0x1C00 does not list, and turns it on. A slave can refuse SAFE-OP for that.
    this->api->clear_process_data_sms(ENABLED_GROUP);
    const int iomap_size = this->api->config_map_group(
        this->iomap.data(),
        ENABLED_GROUP
    );
    this->api->set_manual_state_change(false);

    if (iomap_size <= 0)
        return x::errors::Error(
            errors::ACTIVATION_ERROR,
            "failed to configure PDO mapping"
        );

    this->api->configdc();
    this->api->set_group_dc(ENABLED_GROUP);
    this->has_dc = this->api->group_hasdc(ENABLED_GROUP);

    this->input_sz = this->api->group_Ibytes(ENABLED_GROUP);
    this->output_sz = this->api->group_Obytes(ENABLED_GROUP);

    this->expected_wkc = (this->api->group_outputsWKC(ENABLED_GROUP) * 2) +
                         this->api->group_inputsWKC(ENABLED_GROUP);
    this->activated = true;
    // A slave can stop its process data task for good when the master reads its inputs
    // while it moves to SAFE-OP.
    this->process_data_held.store(true, std::memory_order_release);
    start_cycle();
    if (auto err = this->start_dc()) return err;

    const int timeout = micros(this->cfg.state_change_timeout);
    if (auto err = this->request_state(EC_STATE_SAFE_OP, timeout)) return err;
    this->process_data_held.store(false, std::memory_order_release);
    // A slave with outputs refuses OP with a sync manager watchdog error until a frame
    // passes it in SAFE-OP.
    if (!this->wait_for_frame(
            this->cfg.clock.steady() + std::chrono::microseconds(timeout),
            [] { return false; }
        ))
        return x::errors::Error(
            errors::STATE_CHANGE_ERROR,
            "failed to reach OP: no process data passed the slaves in SAFE-OP"
        );
    if (auto err = this->request_state(EC_STATE_OPERATIONAL, timeout)) return err;
    this->cache_pdo_offsets();
    return x::errors::NIL;
}

std::map<uint16_t, slave::Refusal> Master::refusals() const {
    std::lock_guard lock(this->mu);
    return this->start_refusals;
}

bool Master::dc_sync_refused() const {
    std::lock_guard lock(this->mu);
    return this->start_dc_sync_refused;
}

void Master::leave_out(std::map<uint16_t, slave::Refusal> slaves) {
    std::lock_guard lock(this->mu);
    this->left_out = std::move(slaves);
}

void Master::deactivate() {
    if (this->activated) {
        this->request_state(EC_STATE_INIT, micros(this->cfg.state_change_timeout));
        // INIT leaves the DC activation as it is.
        if (const auto err = this->clear_dc_activation())
            LOG(WARNING) << "[ethercat] " << err.data;
        this->dc_setups.clear();
        this->last_dc_time = NO_DC_FRAME;
        this->host_dc_offset.store(NO_DC_FRAME, std::memory_order_relaxed);
        {
            std::lock_guard lock(this->mu);
            this->dc_restore_errors.clear();
        }
        this->activated = false;
        this->expected_wkc = 0;
        this->has_dc = false;
        this->input_sz = 0;
        this->output_sz = 0;
        this->pdo_offsets.clear();
        this->slave_errors.clear();
    }

    if (this->initialized) {
        this->api->close();
        this->initialized = false;
        this->slave_list.clear();
    }
}

x::errors::Error Master::receive() {
    if (!this->activated)
        return x::errors::Error(errors::CYCLIC_ERROR, "master not activated");

    if (this->held_err) return this->held_err;
    const int wkc = this->api->receive_processdata(ENABLED_GROUP, PROCESSDATA_TIMEOUT);

    if (wkc < 0)
        return x::errors::Error(errors::CYCLIC_ERROR, "process data receive failed");
    this->frames.fetch_add(1, std::memory_order_release);

    if (wkc != this->expected_wkc)
        return x::errors::Error(
            errors::WORKING_COUNTER_ERROR,
            "working counter mismatch: expected " + std::to_string(this->expected_wkc) +
                ", got " + std::to_string(wkc)
        );

    // SOEM takes the DC time without the working counter of its datagram. A lost
    // reference clock gives a frozen time, and a reset one goes back. The frame read
    // the time as it went out, which can be a bus period before this receive.
    if (const int64_t dc = this->api->dc_time();
        this->has_dc && dc > this->last_dc_time) {
        this->last_dc_time = dc;
        const x::telem::TimeSpan sent_at(this->sent_at.time_since_epoch());
        this->host_dc_offset.store(
            dc - sent_at.nanoseconds(),
            std::memory_order_relaxed
        );
    }
    return x::errors::NIL;
}

x::errors::Error Master::send() {
    if (!this->activated)
        return x::errors::Error(errors::CYCLIC_ERROR, "master not activated");

    this->held_err = x::errors::NIL;
    if (this->process_data_held.load(std::memory_order_acquire)) {
        if (this->has_dc &&
            this->api->sync_dc(ENABLED_GROUP, PROCESSDATA_TIMEOUT) <= 0) {
            this->held_err = x::errors::Error(
                errors::CYCLIC_ERROR,
                "reference clock frame failed"
            );
            return this->held_err;
        }
        if (this->has_dc) this->frames.fetch_add(1, std::memory_order_release);
        this->held_err = x::errors::Error(
            errors::WORKING_COUNTER_ERROR,
            "no process data passes the slaves until they reach SAFE-OP"
        );
        return x::errors::NIL;
    }
    const auto sent_at = this->cfg.clock.steady();
    const int result = this->api->send_processdata(ENABLED_GROUP);

    if (result <= 0)
        return x::errors::Error(errors::CYCLIC_ERROR, "process data send failed");
    this->sent_at = sent_at;
    return x::errors::NIL;
}

std::optional<x::telem::TimeStamp> Master::dc_time() const {
    if (!this->has_dc) return std::nullopt;
    return master::DC_EPOCH + x::telem::TimeSpan(this->api->dc_time());
}

std::vector<dc::Registers> Master::dc_sync() const {
    std::vector<dc::Registers> syncs;
    for (const auto &[pos, regs]: this->dc_setups)
        if (regs.sync0_cycle > 0 && (regs.assign_activate & dc::PDI_SYNC_UNIT) == 0)
            syncs.push_back(regs);
    return syncs;
}

std::span<const uint8_t> Master::input_data() {
    if (!this->activated) return {};
    return {this->iomap.data() + this->output_sz, this->input_sz};
}

std::span<uint8_t> Master::output_data() {
    if (!this->activated) return {};
    return {this->iomap.data(), this->output_sz};
}

std::pair<pdo::Offset, x::errors::Error>
Master::pdo_offset(const pdo::Entry &entry) const {
    std::lock_guard lock(this->mu);
    if (const auto it = this->slave_errors.find(entry.slave_position);
        it != this->slave_errors.end())
        return {{}, it->second};
    return pdo::find_offset(this->pdo_offsets, entry);
}

std::vector<slave::DiscoveryResult> Master::slaves() const {
    std::lock_guard lock(this->mu);
    return this->slave_list;
}

std::vector<slave::Health> Master::check(const x::breaker::Breaker &breaker) {
    this->api->readstate();
    std::vector<slave::Health> health;
    for (int i = 1; i <= this->api->slave_count(); ++i) {
        if (this->api->slave_group(i) != ENABLED_GROUP) continue;
        const uint16_t state = this->api->slave_state(i);
        auto &h = health.emplace_back(
            slave::Health{
                .position = static_cast<uint16_t>(i),
                .state = slave::from_al_state(state),
                .faulted = (state & EC_STATE_ERROR) != 0,
                .lost = this->api->slave_lost(i),
            }
        );
        if (this->al_error(i))
            h.al_status = ec_ALstatuscode2string(this->api->slave_ALstatuscode(i));
        {
            std::lock_guard lock(this->mu);
            if (const auto it = this->dc_restore_errors.find(i);
                it != this->dc_restore_errors.end()) {
                h.faulted = true;
                h.al_status = it->second;
            }
        }
        if (breaker.running()) this->recover(static_cast<uint16_t>(i), breaker);
    }
    std::lock_guard lock(this->mu);
    for (const auto &[pos, refusal]: this->left_out)
        health.push_back({.position = pos, .refusal = refusal});
    return health;
}

void Master::recover(const uint16_t pos, const x::breaker::Breaker &breaker) {
    const uint16_t state = this->api->slave_state(pos);
    if (state == EC_STATE_SAFE_OP + EC_STATE_ERROR) {
        this->api->set_slave_state(pos, EC_STATE_SAFE_OP + EC_STATE_ACK);
        this->api->writestate(pos);
    } else if (state == EC_STATE_SAFE_OP) {
        const bool restored = [this, pos] {
            std::lock_guard lock(this->mu);
            return !this->dc_restore_errors.contains(pos);
        }();
        // From PRE-OP, the next check restores the clock again.
        this->api->set_slave_state(
            pos,
            restored ? EC_STATE_OPERATIONAL : EC_STATE_PRE_OP
        );
        this->api->writestate(pos);
    } else if (state > EC_STATE_NONE && state != EC_STATE_OPERATIONAL) {
        if (this->api->slave_hasdc(pos))
            this->api->set_PO2SOconfig(pos, [this, pos, &breaker](uint16_t) {
                const auto err = this->restore_dc(pos, breaker);
                std::lock_guard lock(this->mu);
                if (err)
                    this->dc_restore_errors[pos] = std::format(
                        "Failed to restore the distributed clock: {}",
                        err.data
                    );
                else
                    this->dc_restore_errors.erase(pos);
            });
        if (this->api->reconfig_slave(pos, RECOVERY_TIMEOUT) >= EC_STATE_PRE_OP)
            this->api->set_slave_lost(pos, false);
        this->api->set_PO2SOconfig(pos, nullptr);
    } else if (state == EC_STATE_NONE && !this->api->slave_lost(pos)) {
        this->api->statecheck(pos, EC_STATE_OPERATIONAL, EC_TIMEOUTRET);
        if (this->api->slave_state(pos) == EC_STATE_NONE)
            this->api->set_slave_lost(pos, true);
    }
    if (!this->api->slave_lost(pos)) return;
    if (this->api->slave_state(pos) != EC_STATE_NONE ||
        this->api->recover_slave(pos, RECOVERY_TIMEOUT) > 0)
        this->api->set_slave_lost(pos, false);
}

std::string Master::interface_name() const {
    return this->iface_name;
}

void Master::populate_slaves() {
    this->slave_list.clear();
    this->slave_list.reserve(this->api->slave_count());

    for (int i = 1; i <= this->api->slave_count(); ++i) {
        slave::DiscoveryResult result{};
        result.properties.network = this->interface_name();
        result.properties.position = static_cast<uint16_t>(i);
        result.properties.vendor_id = this->api->slave_eep_man(i);
        result.properties.product_code = this->api->slave_eep_id(i);
        result.properties.revision = this->api->slave_eep_rev(i);
        result.properties.serial = this->api->slave_eep_ser(i);
        result.properties.name = this->api->slave_name(i);
        result.properties.input_bits = this->api->slave_Ibits(i);
        result.properties.output_bits = this->api->slave_Obits(i);
        result.status.state = slave::from_al_state(this->api->slave_state(i));
        this->discover_slave_pdos(result);
        result.properties.op_modes = this->read_op_modes(i);
        this->slave_list.push_back(result);
    }
}

void Master::cache_pdo_offsets() {
    std::lock_guard lock(this->mu);
    this->pdo_offsets.clear();
    this->slave_errors.clear();

    for (const auto &slave: this->slave_list) {
        const auto &props = slave.properties;
        if (props.position == 0 ||
            props.position > static_cast<uint16_t>(this->api->slave_count()))
            continue;

        if (this->api->slave_group(props.position) != ENABLED_GROUP) continue;

        const auto *iomap_base = this->iomap.data();

        const auto *slave_outputs = this->api->slave_outputs(props.position);
        const size_t slave_output_offset = slave_outputs != nullptr
                                             ? static_cast<size_t>(
                                                   slave_outputs - iomap_base
                                               )
                                             : 0;

        const auto *slave_inputs = this->api->slave_inputs(props.position);
        const size_t slave_input_abs_offset = slave_inputs != nullptr
                                                ? static_cast<size_t>(
                                                      slave_inputs - iomap_base
                                                  )
                                                : this->output_sz;
        const size_t slave_input_offset = slave_input_abs_offset - this->output_sz;

        const auto input_bits = pdo::total_bits(props.input_pdos);
        const auto output_bits = pdo::total_bits(props.output_pdos);
        const auto mapped_input_bits = this->api->slave_Ibits(props.position);
        const auto mapped_output_bits = this->api->slave_Obits(props.position);
        if (input_bits != mapped_input_bits || output_bits != mapped_output_bits) {
            auto err = x::errors::Error(
                errors::PDO_MAPPING_ERROR,
                std::format(
                    "slave {} ({}) maps {} input and {} output bits, but its discovered "
                    "PDO entries hold {} and {}",
                    props.position,
                    props.name,
                    mapped_input_bits,
                    mapped_output_bits,
                    input_bits,
                    output_bits
                )
            );
            LOG(WARNING) << err.message();
            this->slave_errors.emplace(props.position, std::move(err));
            continue;
        }

        if (!props.coe_pdo_order_reliable) {
            LOG(WARNING) << "Slave " << props.position
                         << " PDO offsets may be incorrect (fallback discovery used)";
        }

        pdo::compute_offsets(
            this->pdo_offsets,
            props.position,
            props.input_pdos,
            true,
            slave_input_offset
        );
        pdo::compute_offsets(
            this->pdo_offsets,
            props.position,
            props.output_pdos,
            false,
            slave_output_offset
        );
    }
    for (const auto &[pos, refusal]: this->left_out)
        this->slave_errors.insert_or_assign(
            pos,
            master::left_out_error(pos, this->api->slave_name(pos), refusal)
        );
}

/// Timeout for SDO (Service Data Object) reads during PDO discovery in microseconds.
const int SDO_TIMEOUT = static_cast<int>((x::telem::MILLISECOND * 700).microseconds());
constexpr int MAX_SDO_FAILURES = 3;

void Master::discover_slave_pdos(slave::DiscoveryResult &slave) {
    auto &props = slave.properties;
    if (this->discover_pdos_coe(slave)) {
        esi::fill_known_entries(props);
        return;
    }
    slave.status.pdo_discovery_error.clear();
    this->discover_pdos_sii(slave);
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
    VLOG(1) << "Slave " << props.position
            << " PDOs discovered via ESI: " << props.input_pdos.size() << " inputs, "
            << props.output_pdos.size() << " outputs";
    slave.status.pdo_discovery_error.clear();
    props.coe_pdo_order_reliable = true;
}

bool Master::discover_pdos_coe(slave::DiscoveryResult &slave) {
    auto &props = slave.properties;
    auto &status = slave.status;
    VLOG(1) << "Slave " << props.position << " mbx_proto: 0x" << std::hex
            << static_cast<int>(this->api->slave_mbx_proto(props.position)) << std::dec;
    if ((this->api->slave_mbx_proto(props.position) & ECT_MBXPROT_COE) == 0) {
        VLOG(1) << "Slave " << props.position << " (" << props.name
                << ") does not support CoE, falling back to SII";
        return false;
    }

    VLOG(1) << "Slave " << props.position << " supports CoE, reading PDO assignments";

    bool tx_success = false;
    bool rx_success = false;

    bool tx_order_reliable = false;
    bool rx_order_reliable = false;

    auto err = this->read_pdo_assign(props.position, ECT_SDO_TXPDOASSIGN, true, slave);
    if (err) {
        VLOG(1) << "Failed to read TxPDO assignment for slave " << props.position
                << ": " << err.message() << " - trying direct PDO read";
        err = this->read_pdo_mapping(props.position, TXPDO_MAPPING_START, true, slave);
        if (err)
            VLOG(1) << "Failed to read TxPDO mapping 0x" << std::hex
                    << TXPDO_MAPPING_START << std::dec << " for slave "
                    << props.position << ": " << err.message();
        else
            tx_success = true;
    } else {
        tx_success = true;
        tx_order_reliable = true;
    }

    err = this->read_pdo_assign(props.position, ECT_SDO_RXPDOASSIGN, false, slave);
    if (err) {
        VLOG(1) << "Failed to read RxPDO assignment for slave " << props.position
                << ": " << err.message() << " - trying direct PDO read";
        err = this->read_pdo_mapping(props.position, RXPDO_MAPPING_START, false, slave);
        if (err)
            VLOG(1) << "Failed to read RxPDO mapping 0x" << std::hex
                    << RXPDO_MAPPING_START << std::dec << " for slave "
                    << props.position << ": " << err.message();
        else
            rx_success = true;
    } else {
        rx_success = true;
        rx_order_reliable = true;
    }

    if (!tx_success && !rx_success) {
        VLOG(1) << "Standard PDO discovery failed for slave " << props.position
                << ", scanning object dictionary";
        if (this->scan_object_dictionary_for_pdos(props.position, slave)) {
            tx_success = !props.input_pdos.empty();
            rx_success = !props.output_pdos.empty();
        }
    }

    if ((!tx_success && !rx_success) ||
        (props.input_pdos.empty() && props.output_pdos.empty())) {
        status.pdo_discovery_error = "no PDO objects found in object dictionary";
        return false;
    }

    status.pdos_discovered = true;
    props.coe_pdo_order_reliable = tx_order_reliable && rx_order_reliable;
    VLOG(1) << "Slave " << props.position
            << " PDOs discovered via CoE: " << props.input_pdos.size() << " inputs, "
            << props.output_pdos.size() << " outputs";
    return true;
}

void Master::discover_pdos_sii(slave::DiscoveryResult &slave) {
    auto &props = slave.properties;
    auto &status = slave.status;
    VLOG(1) << "Using SII fallback for slave " << props.position;

    for (uint8_t t = 0; t <= 1; ++t) {
        const bool is_input = (t == 0);
        const uint16_t startpos = this->api->siifind(props.position, ECT_SII_PDO + t);
        VLOG(1) << "Slave " << props.position << " SII category " << (ECT_SII_PDO + t)
                << " (" << (is_input ? "TxPDO" : "RxPDO") << ") startpos: " << startpos;
        if (startpos == 0) continue;

        uint16_t a = startpos;
        uint16_t length = this->api->siigetbyte(props.position, a++);
        length += (this->api->siigetbyte(props.position, a++) << 8);

        VLOG(1) << "Slave " << props.position << " " << (is_input ? "TxPDO" : "RxPDO")
                << " length: " << length;

        uint16_t c = 1;
        while (c < length) {
            uint16_t pdo_index = this->api->siigetbyte(props.position, a++);
            pdo_index += (this->api->siigetbyte(props.position, a++) << 8);
            c++;

            const uint8_t num_entries = this->api->siigetbyte(props.position, a++);
            const uint8_t sync_manager = this->api->siigetbyte(props.position, a++);
            a++;
            const uint8_t pdo_name_idx = this->api->siigetbyte(props.position, a++);
            a += 2;
            c += 2;

            std::optional<std::string> pdo_name;
            if (pdo_name_idx > 0) {
                char name_buf[EC_MAXNAME + 1] = {0};
                this->api->siistring(name_buf, props.position, pdo_name_idx);
                if (name_buf[0] != '\0') pdo_name = name_buf;
            }

            if (sync_manager >= EC_MAXSM) {
                c += 4 * num_entries;
                a += 8 * num_entries;
                c++;
                continue;
            }

            for (uint8_t er = 0; er < num_entries; ++er) {
                c += 4;
                uint16_t obj_idx = this->api->siigetbyte(props.position, a++);
                obj_idx += (this->api->siigetbyte(props.position, a++) << 8);
                const uint8_t obj_subidx = this->api->siigetbyte(props.position, a++);
                const uint8_t obj_name_idx = this->api->siigetbyte(props.position, a++);
                const auto coe_type = static_cast<telem::DataType>(
                    this->api->siigetbyte(props.position, a++)
                );
                const uint8_t bitlen = this->api->siigetbyte(props.position, a++);
                a += 2;

                if (obj_idx == 0) {
                    pdo::Properties gap{
                        .pdo_index = pdo_index,
                        .bit_length = bitlen,
                        .is_input = is_input,
                    };
                    if (is_input)
                        props.input_pdos.push_back(gap);
                    else
                        props.output_pdos.push_back(gap);
                    continue;
                }

                std::string entry_name;
                if (obj_name_idx > 0) {
                    char name_buf[EC_MAXNAME + 1] = {0};
                    this->api->siistring(name_buf, props.position, obj_name_idx);
                    entry_name = name_buf;
                }

                const auto data_type = telem::map_ethercat_to_synnax(coe_type, bitlen);
                const std::string name = telem::generate_pdo_entry_name(
                    entry_name,
                    obj_idx,
                    obj_subidx,
                    is_input,
                    data_type
                );

                pdo::Properties entry(
                    pdo_index,
                    obj_idx,
                    obj_subidx,
                    bitlen,
                    is_input,
                    name,
                    data_type,
                    pdo_name
                );

                if (is_input)
                    props.input_pdos.push_back(entry);
                else
                    props.output_pdos.push_back(entry);
            }
            c++;
        }
    }
    this->api->eeprom2pdi(props.position);

    if (props.input_pdos.empty() && props.output_pdos.empty()) {
        if (this->api->slave_Ibits(props.position) > 0 ||
            this->api->slave_Obits(props.position) > 0) {
            VLOG(1) << "Slave " << props.position
                    << " has Ibits=" << this->api->slave_Ibits(props.position)
                    << " Obits=" << this->api->slave_Obits(props.position)
                    << " but no SII PDO info";
            status.pdo_discovery_error = "PDO details not available (I/O size: " +
                                         std::to_string(
                                             this->api->slave_Ibits(props.position)
                                         ) +
                                         " input bits, " +
                                         std::to_string(
                                             this->api->slave_Obits(props.position)
                                         ) +
                                         " output bits)";
        } else {
            status.pdo_discovery_error = "no PDOs found";
        }
    }

    status.pdos_discovered = true;
    VLOG(1) << "Slave " << props.position
            << " PDOs discovered via SII: " << props.input_pdos.size() << " inputs, "
            << props.output_pdos.size() << " outputs";
}

/// SII category that lists the synchronization modes of a slave.
constexpr uint16_t SII_DC_CATEGORY = 60;
/// Size in bytes of one mode in the SII DC category.
constexpr uint32_t SII_OP_MODE_SIZE = 24;

std::vector<dc::OpMode> Master::read_op_modes(const uint16_t position) {
    const uint16_t start = this->api->siifind(position, SII_DC_CATEGORY);
    if (start == 0) return {};
    uint16_t addr = start;
    const auto read = [&](const int bytes) {
        uint32_t value = 0;
        for (int i = 0; i < bytes; ++i)
            value |= static_cast<uint32_t>(this->api->siigetbyte(position, addr++))
                  << (8 * i);
        return value;
    };
    const uint32_t length = read(2) * 2;
    const uint16_t data = addr;
    std::vector<dc::OpMode> modes;
    for (uint32_t off = 0; off + SII_OP_MODE_SIZE <= length; off += SII_OP_MODE_SIZE) {
        addr = static_cast<uint16_t>(data + off);
        dc::OpMode mode;
        mode.sync0_cycle = read(4);
        mode.sync0_shift = static_cast<int32_t>(read(4));
        mode.sync1_shift = static_cast<int32_t>(read(4));
        mode.sync1_factor = static_cast<int16_t>(read(2));
        mode.assign_activate = static_cast<uint16_t>(read(2));
        mode.sync0_factor = static_cast<int16_t>(read(2));
        const auto name_index = static_cast<uint16_t>(read(1));
        if (name_index > 0) {
            char name[EC_MAXNAME + 1] = {0};
            this->api->siistring(name, position, name_index);
            mode.name = name;
        }
        modes.push_back(std::move(mode));
    }
    this->api->eeprom2pdi(position);
    VLOG(1) << "Slave " << position << " lists " << modes.size() << " OpModes in SII";
    return modes;
}

x::errors::Error Master::read_pdo_assign(
    const uint16_t slave_pos,
    const uint16_t assign_index,
    const bool is_input,
    slave::DiscoveryResult &slave
) {
    uint8_t num_pdos = 0;
    int size = sizeof(num_pdos);
    int result = this->api->SDOread(
        slave_pos,
        assign_index,
        0,
        FALSE,
        &size,
        &num_pdos,
        SDO_TIMEOUT
    );
    if (result <= 0) {
        VLOG(1) << "Slave " << slave_pos << " SDO read 0x" << std::hex << assign_index
                << ":0 failed, result=" << std::dec << result
                << " ecx_err=" << this->api->slave_ALstatuscode(slave_pos);
        return x::errors::Error(
            errors::SDO_READ_ERROR,
            "failed to read PDO assignment count"
        );
    }
    VLOG(1) << "Slave " << slave_pos << " PDO assign 0x" << std::hex << assign_index
            << " has " << std::dec << static_cast<int>(num_pdos) << " PDOs";

    int consecutive_failures = 0;
    for (uint8_t i = 1; i <= num_pdos; ++i) {
        uint16_t pdo_index = 0;
        size = sizeof(pdo_index);
        result = this->api->SDOread(
            slave_pos,
            assign_index,
            i,
            FALSE,
            &size,
            &pdo_index,
            SDO_TIMEOUT
        );
        if (result <= 0) {
            consecutive_failures++;
            if (consecutive_failures >= MAX_SDO_FAILURES)
                return x::errors::Error(
                    errors::SDO_READ_ERROR,
                    "too many consecutive SDO failures"
                );
            continue;
        }
        consecutive_failures = 0;

        if (pdo_index == 0) continue;

        if (auto err = this->read_pdo_mapping(slave_pos, pdo_index, is_input, slave))
            VLOG(1) << "Failed to read PDO mapping 0x" << std::hex << pdo_index
                    << " for slave " << std::dec << slave_pos << ": " << err.message();
    }

    return x::errors::NIL;
}

x::errors::Error Master::read_pdo_mapping(
    const uint16_t slave_pos,
    const uint16_t pdo_index,
    const bool is_input,
    slave::DiscoveryResult &slave
) {
    auto &props = slave.properties;
    uint8_t num_entries = 0;
    int size = sizeof(num_entries);
    int result = this->api->SDOread(
        slave_pos,
        pdo_index,
        0,
        FALSE,
        &size,
        &num_entries,
        SDO_TIMEOUT
    );
    if (result <= 0) {
        VLOG(1) << "Slave " << slave_pos << " SDO read 0x" << std::hex << pdo_index
                << ":0 failed, result=" << std::dec << result;
        return x::errors::Error(
            errors::SDO_READ_ERROR,
            "failed to read PDO mapping count"
        );
    }
    VLOG(1) << "Slave " << slave_pos << " PDO 0x" << std::hex << pdo_index << " has "
            << std::dec << static_cast<int>(num_entries) << " entries";

    const auto pdo_name = this->read_object_name(slave_pos, pdo_index);
    int consecutive_failures = 0;
    for (uint8_t i = 1; i <= num_entries; ++i) {
        uint32_t mapping = 0;
        size = sizeof(mapping);
        result = this->api->SDOread(
            slave_pos,
            pdo_index,
            i,
            FALSE,
            &size,
            &mapping,
            SDO_TIMEOUT
        );
        if (result <= 0) {
            consecutive_failures++;
            if (consecutive_failures >= MAX_SDO_FAILURES)
                return x::errors::Error(
                    errors::SDO_READ_ERROR,
                    "too many consecutive SDO failures"
                );
            continue;
        }
        consecutive_failures = 0;

        if (mapping == 0) continue;

        const auto index = static_cast<uint16_t>((mapping >> 16) & 0xFFFF);
        const auto sub_index = static_cast<uint8_t>((mapping >> 8) & 0xFF);
        const auto bit_length = static_cast<uint8_t>(mapping & 0xFF);

        if (index == 0) {
            pdo::Properties gap{
                .pdo_index = pdo_index,
                .bit_length = bit_length,
                .is_input = is_input,
            };
            if (is_input)
                props.input_pdos.push_back(gap);
            else
                props.output_pdos.push_back(gap);
            continue;
        }

        const auto [coe_name, coe_type] = this->read_pdo_entry(
            slave_pos,
            index,
            sub_index
        );
        const auto data_type = telem::map_ethercat_to_synnax(coe_type, bit_length);
        const std::string name = telem::generate_pdo_entry_name(
            coe_name,
            index,
            sub_index,
            is_input,
            data_type
        );

        pdo::Properties entry(
            pdo_index,
            index,
            sub_index,
            bit_length,
            is_input,
            name,
            data_type,
            pdo_name
        );

        if (is_input)
            props.input_pdos.push_back(entry);
        else
            props.output_pdos.push_back(entry);
    }

    return x::errors::NIL;
}

std::optional<std::string>
Master::read_object_name(const uint16_t slave_pos, const uint16_t index) {
    ec_ODlistt od_list{};
    od_list.Slave = slave_pos;
    od_list.Index[0] = index;
    od_list.Entries = 1;
    if (this->api->readODdescription(0, &od_list) > 0 && od_list.Name[0][0] != '\0')
        return std::string(od_list.Name[0]);
    return std::nullopt;
}

std::pair<std::string, telem::DataType> Master::read_pdo_entry(
    const uint16_t slave_pos,
    const uint16_t index,
    const uint8_t sub_index
) {
    ec_ODlistt od_list{};
    od_list.Slave = slave_pos;
    od_list.Index[0] = index;
    od_list.Entries = 1;

    ec_OElistt oe_list{};
    if (this->api->readOEsingle(0, sub_index, &od_list, &oe_list) <= 0 ||
        oe_list.Entries == 0)
        return {"", telem::DataType::EC_UNKNOWN};
    return {
        std::string(oe_list.Name[sub_index]),
        static_cast<telem::DataType>(oe_list.DataType[sub_index])
    };
}

bool Master::scan_object_dictionary_for_pdos(
    const uint16_t slave_pos,
    slave::DiscoveryResult &slave
) {
    ec_ODlistt od_list{};
    std::memset(&od_list, 0, sizeof(od_list));
    od_list.Slave = slave_pos;

    if (this->api->readODlist(slave_pos, &od_list) <= 0) {
        VLOG(1) << "Slave " << slave_pos << " failed to read object dictionary list";
        return false;
    }

    VLOG(1) << "Slave " << slave_pos << " object dictionary has " << od_list.Entries
            << " entries";

    std::vector<uint16_t> txpdo_indices;
    std::vector<uint16_t> rxpdo_indices;

    for (uint16_t i = 0; i < od_list.Entries; ++i) {
        const uint16_t index = od_list.Index[i];
        if (index >= TXPDO_MAPPING_START && index <= TXPDO_MAPPING_END) {
            txpdo_indices.push_back(index);
            VLOG(1) << "Slave " << slave_pos << " found TxPDO object 0x" << std::hex
                    << index << std::dec;
        } else if (index >= RXPDO_MAPPING_START && index <= RXPDO_MAPPING_END) {
            rxpdo_indices.push_back(index);
            VLOG(1) << "Slave " << slave_pos << " found RxPDO object 0x" << std::hex
                    << index << std::dec;
        }
    }

    bool found_any = false;

    for (const auto pdo_index: txpdo_indices) {
        auto err = this->read_pdo_mapping(slave_pos, pdo_index, true, slave);
        if (!err) found_any = true;
    }

    for (const auto pdo_index: rxpdo_indices) {
        auto err = this->read_pdo_mapping(slave_pos, pdo_index, false, slave);
        if (!err) found_any = true;
    }

    VLOG(1) << "Slave " << slave_pos << " OD scan found " << txpdo_indices.size()
            << " TxPDO objects, " << rxpdo_indices.size() << " RxPDO objects";

    return found_any;
}

template<std::integral T>
x::errors::Error
Master::write_dc_register(const uint16_t pos, const uint16_t address, const T value) {
    const auto bytes = x::binary::to_little_endian(value);
    if (this->api->write_register(
            pos,
            address,
            {bytes.begin(), bytes.end()},
            EC_TIMEOUTRET
        ) == 1)
        return x::errors::NIL;
    return x::errors::Error(
        errors::ACTIVATION_ERROR,
        std::format(
            "slave {} ({}) did not take a write to DC register 0x{:04X}",
            pos,
            this->api->slave_name(pos),
            address
        )
    );
}

template<std::integral T>
std::pair<T, x::errors::Error>
Master::read_dc_register(const uint16_t pos, const uint16_t address) {
    std::array<uint8_t, sizeof(T)> bytes{};
    if (this->api->read_register(pos, address, bytes, EC_TIMEOUTRET) == 1)
        return {x::binary::from_little_endian<T>(bytes), x::errors::NIL};
    return {
        0,
        x::errors::Error(
            errors::ACTIVATION_ERROR,
            std::format(
                "slave {} ({}) did not answer a read of DC register 0x{:04X}",
                pos,
                this->api->slave_name(pos),
                address
            )
        )
    };
}

x::errors::Error Master::clear_sync(const uint16_t pos) {
    // 0x0980 goes first: while it gives the sync unit to the PDI, the slave refuses
    // writes to 0x0981.
    for (const uint16_t address: {ECT_REG_DCCUC, ECT_REG_DCSYNCACT})
        if (auto err = this->write_dc_register<uint8_t>(pos, address, 0)) return err;
    return x::errors::NIL;
}

x::errors::Error Master::clear_dc_activation() {
    std::string failed;
    for (int i = 1; i <= this->api->slave_count(); ++i) {
        if (this->api->slave_group(i) != ENABLED_GROUP || !this->api->slave_hasdc(i))
            continue;
        if (const auto err = this->clear_sync(static_cast<uint16_t>(i))) {
            if (!failed.empty()) failed += "; ";
            failed += err.data;
        }
    }
    if (failed.empty()) return x::errors::NIL;
    return x::errors::Error(errors::ACTIVATION_ERROR, failed);
}

std::pair<std::vector<Master::DCSetup>, x::errors::Error>
Master::resolve_dc(const x::telem::TimeSpan period) const {
    std::vector<DCSetup> setups;
    std::vector<x::telem::TimeSpan> cycles;
    std::string events;
    std::lock_guard lock(this->mu);
    for (const auto &slave: this->slave_list) {
        const auto &props = slave.properties;
        if (this->disabled_slaves.contains(props.position) ||
            this->left_out.contains(props.position))
            continue;
        std::optional<dc::Choice> choice;
        if (const auto it = this->chosen_modes.find(props.position);
            it != this->chosen_modes.end())
            choice = it->second;
        const auto [mode, select_err] = dc::select(props.op_modes, choice);
        if (select_err)
            return {
                {},
                x::errors::Error(
                    errors::SLAVE_CONFIG_ERROR,
                    std::format(
                        "slave {} ({}) cannot use its chosen mode: {}",
                        props.position,
                        props.name,
                        select_err.data
                    )
                )
            };
        if (!mode) continue;
        auto [regs, err] = dc::resolve(
            *mode,
            period,
            choice ? choice->cycles : dc::Cycles{}
        );
        if (err)
            return {
                {},
                x::errors::Error(
                    errors::SLAVE_CONFIG_ERROR,
                    std::format(
                        "slave {} ({}) cannot run its DC mode at a bus cycle of {}: {}",
                        props.position,
                        props.name,
                        period.to_string(),
                        err.data
                    )
                )
            };
        setups.push_back({.position = props.position, .registers = regs});
        if (!regs.marks_data() || (regs.assign_activate & dc::PDI_SYNC_UNIT) != 0)
            continue;
        cycles.emplace_back(regs.sync0_cycle);
        if (!events.empty()) events += ", ";
        events += std::format(
            "slave {} ({}) every {}",
            props.position,
            props.name,
            cycles.back().to_string()
        );
    }
    if (!cycles.empty() && dc::grid(period, cycles) <
                               static_cast<double>(dc::MIN_EVENT_CYCLE.nanoseconds()))
        return {
            {},
            x::errors::Error(
                errors::SLAVE_CONFIG_ERROR,
                std::format(
                    "SYNC0 events do not line up with the bus cycle of {}: {}. Change "
                    "the bus rate or the SYNC0 cycles so that a span of {} or more "
                    "divides the bus cycle and each SYNC0 cycle.",
                    period.to_string(),
                    events,
                    dc::MIN_EVENT_CYCLE.to_string()
                )
            )
        };
    return {setups, x::errors::NIL};
}

x::errors::Error Master::wait_for_dc_settle(
    const std::vector<uint16_t> &positions,
    const std::function<bool()> &stopped
) {
    const auto deadline = this->cfg.clock.steady() +
                          this->cfg.dc_settle_timeout.chrono();
    const x::errors::Error stop_err(
        errors::ACTIVATION_ERROR,
        "the bus stopped before the clocks settled"
    );
    // 0x092C keeps its old value until a frame carries the reference clock to the
    // slave.
    if (!this->wait_for_frame(deadline, stopped)) {
        if (stopped()) return stop_err;
        return x::errors::Error(
            errors::ACTIVATION_ERROR,
            "no frame carried the reference clock to the slaves within " +
                this->cfg.dc_settle_timeout.to_string()
        );
    }
    while (true) {
        uint16_t worst_pos = 0;
        x::telem::TimeSpan worst(0);
        for (const auto pos: positions) {
            const auto [raw, err] = this->read_dc_register<uint32_t>(
                pos,
                ECT_REG_DCSYSDIFF
            );
            if (err) return err;
            // Bit 31 is the sign. The rest is the difference in nanoseconds.
            const x::telem::TimeSpan diff(raw & 0x7FFFFFFF);
            if (diff > worst) {
                worst = diff;
                worst_pos = pos;
            }
        }
        if (worst <= DC_SETTLED_DIFF) return x::errors::NIL;
        if (stopped()) return stop_err;
        if (this->cfg.clock.steady() >= deadline)
            return x::errors::Error(
                errors::ACTIVATION_ERROR,
                std::format(
                    "the clock of slave {} ({}) is {} off the reference clock after "
                    "{}. It must be within {} before SYNC0 starts.",
                    worst_pos,
                    this->api->slave_name(worst_pos),
                    worst.to_string(),
                    this->cfg.dc_settle_timeout.to_string(),
                    DC_SETTLED_DIFF.to_string()
                )
            );
        this->cfg.clock.sleep(DC_SETTLE_POLL_INTERVAL);
    }
}

x::errors::Error Master::start_dc() {
    if (this->dc_setups.empty()) return x::errors::NIL;
    std::vector<uint16_t> positions;
    positions.reserve(this->dc_setups.size());
    for (const auto &setup: this->dc_setups)
        positions.push_back(setup.position);
    if (auto err = this->wait_for_dc_settle(positions, [] { return false; }))
        return err;
    for (const auto &setup: this->dc_setups)
        if (auto err = this->start_sync(setup)) return err;
    return x::errors::NIL;
}

x::errors::Error Master::start_sync(const DCSetup &setup) {
    const auto &[pos, regs] = setup;
    const auto cuc = static_cast<uint8_t>(regs.assign_activate & 0xFF);
    const auto act = static_cast<uint8_t>(regs.assign_activate >> 8);
    if (auto err = this->write_dc_register(pos, ECT_REG_DCCUC, cuc)) return err;
    // While 0x0980 gives the sync unit to the PDI, the PDI sets it up.
    if ((regs.assign_activate & dc::PDI_SYNC_UNIT) != 0) return x::errors::NIL;
    const auto [time, time_err] = this->read_dc_register<int64_t>(
        pos,
        ECT_REG_DCSYSTIME
    );
    if (time_err) return time_err;
    const int64_t cycle = regs.sync0_cycle;
    const int64_t shift = regs.sync0_shift;
    const int64_t ahead = time + DC_START_DELAY.nanoseconds();
    // SYNC0 events fall on whole cycles of DC time plus the shift, the grid the cycle
    // lock uses. The first one is at least DC_START_DELAY away.
    int64_t start = ahead;
    if (cycle > 0) start = ((ahead - shift) / cycle + 1) * cycle + shift;
    if (auto err = this->write_dc_register(pos, ECT_REG_DCCYCLE0, regs.sync0_cycle))
        return err;
    if (auto err = this->write_dc_register(pos, ECT_REG_DCCYCLE1, regs.sync1_cycle))
        return err;
    if (auto err = this->write_dc_register(pos, ECT_REG_DCSTART0, start)) return err;
    if (auto err = this->write_dc_register(pos, ECT_REG_DCSYNCACT, act)) return err;
    LOG(INFO) << std::format(
        "[ethercat] slave {} ({}) runs DC sync: AssignActivate 0x{:04X}, SYNC0 "
        "cycle {} ns, SYNC1 register {} ns, first SYNC0 at {} ns",
        pos,
        this->api->slave_name(pos),
        regs.assign_activate,
        regs.sync0_cycle,
        regs.sync1_cycle,
        start
    );
    return x::errors::NIL;
}

x::errors::Error Master::align_clock(const uint16_t pos) {
    const x::telem::TimeSpan latched_at(this->cfg.clock.steady().time_since_epoch());
    if (this->api->latch_receive_times(EC_TIMEOUTRET) < 1)
        return x::errors::Error(
            errors::ACTIVATION_ERROR,
            "no slave latched its receive time"
        );
    const auto [local, local_err] = this->read_dc_register<int64_t>(pos, ECT_REG_DCSOF);
    if (local_err) return local_err;
    for (int i = 1; i <= this->api->slave_count(); ++i) {
        const auto other = static_cast<uint16_t>(i);
        // A slave in SAFE-OP or OP kept its clock since activation or its restore.
        if (other == pos || this->api->slave_group(other) != ENABLED_GROUP ||
            !this->api->slave_hasdc(other) || this->api->slave_lost(other) ||
            (this->api->slave_state(other) & 0x0F) < EC_STATE_SAFE_OP)
            continue;
        const auto [other_local, other_local_err] = this->read_dc_register<int64_t>(
            other,
            ECT_REG_DCSOF
        );
        if (other_local_err) return other_local_err;
        const auto [other_offset, other_offset_err] = this->read_dc_register<int64_t>(
            other,
            ECT_REG_DCSYSOFFSET
        );
        if (other_offset_err) return other_offset_err;
        // Each slave latched its propagation delay after the reference clock.
        const int64_t system = other_local + other_offset +
                               this->api->slave_pdelay(pos) -
                               this->api->slave_pdelay(other);
        return this->write_dc_register(pos, ECT_REG_DCSYSOFFSET, system - local);
    }
    const int64_t offset = this->host_dc_offset.load(std::memory_order_relaxed);
    if (offset == NO_DC_FRAME)
        return x::errors::Error(
            errors::ACTIVATION_ERROR,
            "no frame has carried the DC time to set the clock from"
        );
    return this->write_dc_register(
        pos,
        ECT_REG_DCSYSOFFSET,
        latched_at.nanoseconds() + offset - local
    );
}

x::errors::Error
Master::restore_dc(const uint16_t pos, const x::breaker::Breaker &breaker) {
    if (auto err = this->clear_sync(pos)) return err;
    if (auto err = this->write_dc_register(
            pos,
            ECT_REG_DCSYSDELAY,
            this->api->slave_pdelay(pos)
        ))
        return err;
    if (auto err = this->align_clock(pos)) return err;
    if (auto err = this->wait_for_dc_settle({pos}, [&breaker] {
            return !breaker.running();
        }))
        return err;
    const auto setup = std::ranges::find(this->dc_setups, pos, &DCSetup::position);
    if (setup == this->dc_setups.end()) return x::errors::NIL;
    return this->start_sync(*setup);
}

x::errors::Error Master::request_state(const uint16_t state, const int timeout) {
    for (int i = 1; i <= this->api->slave_count(); ++i) {
        if (this->api->slave_group(i) != ENABLED_GROUP) continue;
        this->api->set_slave_state(i, state);
        this->api->writestate(i);
    }
    const auto deadline = this->cfg.clock.steady() + std::chrono::microseconds(timeout);
    while (true) {
        this->api->readstate();
        std::vector<uint16_t> pending;
        // Like ecx_statecheck, a slave in the state counts even with its error flag
        // set: a request without the ACK bit leaves the flag as it is.
        for (int i = 1; i <= this->api->slave_count(); ++i)
            if (this->api->slave_group(i) == ENABLED_GROUP &&
                (this->api->slave_state(i) & ~EC_STATE_ERROR) != state)
                pending.push_back(i);
        if (pending.empty()) return x::errors::NIL;
        std::vector<uint16_t> refused;
        std::ranges::copy_if(pending, std::back_inserter(refused), [&](auto pos) {
            return this->refuses(pos, state);
        });
        if (!refused.empty()) return this->state_error(refused, state);
        if (this->cfg.clock.steady() >= deadline)
            return this->state_error(pending, state);
        this->cfg.clock.sleep(STATE_POLL_INTERVAL);
    }
}

bool Master::al_error(const uint16_t pos) const {
    return (this->api->slave_state(pos) & EC_STATE_ERROR) != 0 &&
           this->api->slave_ALstatuscode(pos) != 0;
}

bool Master::wait_for_frame(
    const std::chrono::steady_clock::time_point deadline,
    const std::function<bool()> &stopped
) const {
    // The next frame to come back can have left before the call.
    const auto target = this->frames.load(std::memory_order_acquire) + 2;
    while (this->frames.load(std::memory_order_acquire) < target) {
        if (this->cfg.clock.steady() >= deadline || stopped()) return false;
        this->cfg.clock.sleep(STATE_POLL_INTERVAL);
    }
    return true;
}

bool Master::refuses(const uint16_t pos, const uint16_t state) const {
    return this->al_error(pos) &&
           (this->api->slave_state(pos) & ~EC_STATE_ERROR) < state;
}

x::errors::Error
Master::state_error(const std::vector<uint16_t> &failed, const uint16_t state) {
    const auto target = slave::slave_state_to_string(slave::from_al_state(state));
    std::string msg;
    std::map<uint16_t, slave::Refusal> refusing;
    bool dc_sync_refused = false;
    for (const auto pos: failed) {
        if (!msg.empty()) msg += "; ";
        msg += "slave " + std::to_string(pos) + " (" + this->api->slave_name(pos) +
               ") is in " +
               slave::slave_state_to_string(
                   slave::from_al_state(this->api->slave_state(pos))
               );
        if (!this->al_error(pos)) continue;
        const auto code = this->api->slave_ALstatuscode(pos);
        const std::string text = ec_ALstatuscode2string(code);
        if (this->refuses(pos, state) &&
            std::ranges::find(TRANSIENT_FAULTS, code) == TRANSIENT_FAULTS.end())
            refusing[pos] = {
                .reason = "refused " + target + ": " + text,
                .dc_sync_invalid = code == AL_INVALID_DC_SYNC,
            };
        dc_sync_refused |= code == AL_INVALID_DC_SYNC;
        msg += ": " + text;
    }
    // A slave with no process data, such as a coupler, gives a task nothing to run.
    bool io_left = false;
    for (int i = 1; i <= this->api->slave_count(); ++i) {
        const auto pos = static_cast<uint16_t>(i);
        io_left |= this->api->slave_group(pos) == ENABLED_GROUP &&
                   !refusing.contains(pos) &&
                   (this->api->slave_Ibits(pos) > 0 || this->api->slave_Obits(pos) > 0);
    }
    {
        std::lock_guard lock(this->mu);
        if (io_left) this->start_refusals = refusing;
        this->start_dc_sync_refused = dc_sync_refused;
    }
    return x::errors::Error(
        refusing.empty() ? errors::STATE_CHANGE_ERROR : errors::STATE_CHANGE_REFUSED,
        "failed to reach " + target + ": " + msg
    );
}

std::vector<master::Info> Manager::enumerate() {
    std::vector<master::Info> masters;
    ec_adaptert *adapter = ec_find_adapters();
    ec_adaptert *current = adapter;

    while (current != nullptr) {
        if (is_wired_interface(current->name, current->desc)) {
            master::Info info;
            info.key = current->name;
            info.description = current->desc;
            masters.push_back(std::move(info));
        } else {
            VLOG(1) << "[ethercat] skipping interface that cannot carry EtherCAT: "
                    << current->name;
        }
        current = current->next;
    }

    ec_free_adapters(adapter);
    return masters;
}

std::pair<std::shared_ptr<master::Master>, x::errors::Error>
Manager::create(const std::string &key) {
    if (key.empty())
        return {
            nullptr,
            x::errors::Error(errors::MASTER_INIT_ERROR, "empty interface name")
        };
    if (key.size() >= 4 && key.starts_with("igh:"))
        return {
            nullptr,
            x::errors::Error(
                errors::MASTER_INIT_ERROR,
                "invalid SOEM interface '" + key + "': IgH-style keys not supported"
            )
        };
    return {std::make_shared<Master>(std::make_unique<ProdAPI>(), key), x::errors::NIL};
}

}
