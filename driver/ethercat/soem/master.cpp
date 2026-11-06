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
/// The AssignActivate bit (0x0980 bit 0) that gives the sync unit to the PDI.
constexpr uint16_t PDI_SYNC_UNIT = 0x0001;
namespace {
/// @brief converts a span to the microseconds that SOEM timeouts take.
int micros(const x::telem::TimeSpan span) {
    return static_cast<int>(span.microseconds());
}

/// @brief returns the little-endian bytes of value.
template<typename T>
std::vector<uint8_t> little_endian(const T value) {
    std::vector<uint8_t> bytes(sizeof(T));
    for (size_t i = 0; i < sizeof(T); ++i)
        bytes[i] = static_cast<uint8_t>(static_cast<uint64_t>(value) >> (8 * i));
    return bytes;
}

/// @brief returns the value of little-endian bytes.
template<typename T, size_t N>
T from_little_endian(const std::array<uint8_t, N> &bytes) {
    uint64_t value = 0;
    for (size_t i = 0; i < N; ++i)
        value |= static_cast<uint64_t>(bytes[i]) << (8 * i);
    return static_cast<T>(value);
}
}

/// SOEM group 0 addresses every slave, so enabled and disabled slaves get their own.
constexpr uint8_t ENABLED_GROUP = 1;
constexpr uint8_t DISABLED_GROUP = 2;
/// AL status code for "Invalid DC SYNC configuration".
constexpr uint16_t AL_INVALID_DC_SYNC = 0x0030;
constexpr std::string_view
    DC_SYNC_ADVICE = "A slave that reports \"Invalid DC SYNC configuration\" "
                     "expects a synchronization mode that the Driver did not "
                     "set up. The Driver uses SM sync when a slave lists an "
                     "SM mode, and DC sync when it lists only DC modes. Set "
                     "the slave to the mode that the Driver uses (in "
                     "DewesoftX for Dewesoft modules), or disable the slave "
                     "so the rest of the bus can run.";
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

void Master::set_slave_enabled(const uint16_t position, const bool enabled) {
    std::lock_guard lock(this->mu);
    if (!enabled)
        this->disabled_slaves.insert(position);
    else
        this->disabled_slaves.erase(position);
}

x::errors::Error Master::activate(
    const x::telem::TimeSpan period,
    const std::function<void()> &start_cycle
) {
    if (!this->initialized)
        return x::errors::Error(errors::ACTIVATION_ERROR, "master not initialized");
    if (this->activated)
        return x::errors::Error(errors::ACTIVATION_ERROR, "master already activated");
    auto [dc_setups, dc_err] = this->resolve_dc(period);
    if (dc_err) return dc_err;
    this->dc_setups = std::move(dc_setups);

    {
        std::lock_guard lock(this->mu);
        for (int i = 1; i <= this->api->slave_count(); ++i)
            this->api->set_slave_group(
                i,
                this->disabled_slaves.contains(i) ? DISABLED_GROUP : ENABLED_GROUP
            );
    }

    if (const auto err = this->clear_dc_activation()) return err;

    // The master requests SAFE-OP itself, once the bus cycles and SYNC0 runs.
    this->api->set_manual_state_change(true);
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
    this->has_dc = this->api->group_hasdc(ENABLED_GROUP);

    this->input_sz = this->api->group_Ibytes(ENABLED_GROUP);
    this->output_sz = this->api->group_Obytes(ENABLED_GROUP);

    this->expected_wkc = (this->api->group_outputsWKC(ENABLED_GROUP) * 2) +
                         this->api->group_inputsWKC(ENABLED_GROUP);
    this->activated = true;
    start_cycle();
    if (auto err = this->start_dc()) return err;

    const int timeout = micros(this->cfg.state_change_timeout);
    if (auto err = this->request_state(EC_STATE_SAFE_OP, timeout)) return err;
    // A slave with outputs refuses OP with a sync manager watchdog error until a frame
    // passes it in SAFE-OP.
    if (auto err = this->wait_for_frame(timeout)) return err;
    if (auto err = this->request_state(EC_STATE_OPERATIONAL, timeout)) return err;
    this->cache_pdo_offsets();
    return x::errors::NIL;
}

void Master::deactivate() {
    if (this->activated) {
        this->request_state(EC_STATE_INIT, micros(this->cfg.state_change_timeout));
        // INIT leaves the DC activation as it is.
        if (const auto err = this->clear_dc_activation())
            LOG(WARNING) << "[ethercat] " << err.data;
        this->dc_setups.clear();
        this->activated = false;
        this->expected_wkc = 0;
        this->has_dc = false;
        this->input_sz = 0;
        this->output_sz = 0;
        this->pdo_offsets.clear();
        this->layout_errors.clear();
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

    return x::errors::NIL;
}

x::errors::Error Master::send() {
    if (!this->activated)
        return x::errors::Error(errors::CYCLIC_ERROR, "master not activated");

    const int result = this->api->send_processdata(ENABLED_GROUP);

    if (result <= 0)
        return x::errors::Error(errors::CYCLIC_ERROR, "process data send failed");

    return x::errors::NIL;
}

std::optional<x::telem::TimeStamp> Master::dc_time() const {
    if (!this->has_dc) return std::nullopt;
    return master::DC_EPOCH + x::telem::TimeSpan(this->api->dc_time());
}

std::optional<dc::Registers> Master::dc_sync() const {
    for (const auto &[pos, regs]: this->dc_setups)
        if (regs.sync0_cycle > 0 && (regs.assign_activate & PDI_SYNC_UNIT) == 0)
            return regs;
    return std::nullopt;
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
    if (const auto it = this->layout_errors.find(entry.slave_position);
        it != this->layout_errors.end())
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
        if (breaker.running()) this->recover(static_cast<uint16_t>(i));
    }
    return health;
}

void Master::recover(const uint16_t pos) {
    const uint16_t state = this->api->slave_state(pos);
    if (state == EC_STATE_SAFE_OP + EC_STATE_ERROR) {
        this->api->set_slave_state(pos, EC_STATE_SAFE_OP + EC_STATE_ACK);
        this->api->writestate(pos);
    } else if (state == EC_STATE_SAFE_OP) {
        this->api->set_slave_state(pos, EC_STATE_OPERATIONAL);
        this->api->writestate(pos);
    } else if (state > EC_STATE_NONE && state != EC_STATE_OPERATIONAL) {
        if (this->api->reconfig_slave(pos, RECOVERY_TIMEOUT) >= EC_STATE_PRE_OP)
            this->api->set_slave_lost(pos, false);
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
    this->layout_errors.clear();

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
            this->layout_errors.emplace(props.position, std::move(err));
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

x::errors::Error Master::write_dc_register(
    const uint16_t pos,
    const uint16_t address,
    std::vector<uint8_t> data
) {
    if (this->api->write_register(pos, address, std::move(data), EC_TIMEOUTRET) == 1)
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

x::errors::Error Master::read_dc_register(
    const uint16_t pos,
    const uint16_t address,
    const std::span<uint8_t> data
) {
    if (this->api->read_register(pos, address, data, EC_TIMEOUTRET) == 1)
        return x::errors::NIL;
    return x::errors::Error(
        errors::ACTIVATION_ERROR,
        std::format(
            "slave {} ({}) did not answer a read of DC register 0x{:04X}",
            pos,
            this->api->slave_name(pos),
            address
        )
    );
}

x::errors::Error Master::clear_dc_activation() {
    std::string failed;
    for (int i = 1; i <= this->api->slave_count(); ++i) {
        if (this->api->slave_group(i) != ENABLED_GROUP || !this->api->slave_hasdc(i))
            continue;
        const auto pos = static_cast<uint16_t>(i);
        // 0x0980 goes first: while it gives the sync unit to the PDI, the slave refuses
        // writes to 0x0981.
        for (const uint16_t address: {ECT_REG_DCCUC, ECT_REG_DCSYNCACT})
            if (const auto err = this->write_dc_register(pos, address, {0})) {
                if (!failed.empty()) failed += "; ";
                failed += err.data;
                break;
            }
    }
    if (failed.empty()) return x::errors::NIL;
    return x::errors::Error(errors::ACTIVATION_ERROR, failed);
}

std::pair<std::vector<Master::DCSetup>, x::errors::Error>
Master::resolve_dc(const x::telem::TimeSpan period) const {
    std::vector<DCSetup> setups;
    std::lock_guard lock(this->mu);
    for (const auto &slave: this->slave_list) {
        const auto &props = slave.properties;
        if (this->disabled_slaves.contains(props.position)) continue;
        const auto mode = dc::select(props.op_modes);
        if (!mode) continue;
        auto [regs, err] = dc::resolve(*mode, period);
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
    }
    return {setups, x::errors::NIL};
}

x::errors::Error Master::wait_for_dc_settle() {
    const auto deadline = std::chrono::steady_clock::now() +
                          this->cfg.dc_settle_timeout.chrono();
    while (true) {
        uint16_t worst_pos = 0;
        x::telem::TimeSpan worst(0);
        for (const auto &setup: this->dc_setups) {
            const auto pos = setup.position;
            std::array<uint8_t, 4> raw{};
            if (auto err = this->read_dc_register(pos, ECT_REG_DCSYSDIFF, raw))
                return err;
            // Bit 31 is the sign. The rest is the difference in nanoseconds.
            const x::telem::TimeSpan diff(
                from_little_endian<uint32_t>(raw) & 0x7FFFFFFF
            );
            if (diff > worst) {
                worst = diff;
                worst_pos = pos;
            }
        }
        if (worst <= DC_SETTLED_DIFF) return x::errors::NIL;
        if (std::chrono::steady_clock::now() >= deadline)
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
        std::this_thread::sleep_for(DC_SETTLE_POLL_INTERVAL.chrono());
    }
}

x::errors::Error Master::start_dc() {
    if (this->dc_setups.empty()) return x::errors::NIL;
    if (auto err = this->wait_for_dc_settle()) return err;
    for (const auto &[pos, regs]: this->dc_setups) {
        const auto cuc = static_cast<uint8_t>(regs.assign_activate & 0xFF);
        const auto act = static_cast<uint8_t>(regs.assign_activate >> 8);
        if (auto err = this->write_dc_register(pos, ECT_REG_DCCUC, {cuc})) return err;
        // While 0x0980 gives the sync unit to the PDI, the PDI sets it up.
        if ((regs.assign_activate & PDI_SYNC_UNIT) != 0) continue;
        std::array<uint8_t, 8> raw{};
        if (auto err = this->read_dc_register(pos, ECT_REG_DCSYSTIME, raw)) return err;
        const int64_t cycle = regs.sync0_cycle;
        const int64_t shift = regs.sync0_shift;
        const int64_t ahead = from_little_endian<int64_t>(raw) +
                              DC_START_DELAY.nanoseconds();
        // SYNC0 events fall on whole cycles of DC time plus the shift, the grid the
        // cycle lock uses. The first one is at least DC_START_DELAY away.
        int64_t start = ahead;
        if (cycle > 0) start = ((ahead - shift) / cycle + 1) * cycle + shift;
        if (auto err = this->write_dc_register(
                pos,
                ECT_REG_DCCYCLE0,
                little_endian(regs.sync0_cycle)
            ))
            return err;
        if (auto err = this->write_dc_register(
                pos,
                ECT_REG_DCCYCLE1,
                little_endian(regs.sync1_cycle)
            ))
            return err;
        if (auto err = this->write_dc_register(
                pos,
                ECT_REG_DCSTART0,
                little_endian(start)
            ))
            return err;
        if (auto err = this->write_dc_register(pos, ECT_REG_DCSYNCACT, {act}))
            return err;
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
    }
    return x::errors::NIL;
}

x::errors::Error Master::request_state(const uint16_t state, const int timeout) {
    for (int i = 1; i <= this->api->slave_count(); ++i) {
        if (this->api->slave_group(i) != ENABLED_GROUP) continue;
        this->api->set_slave_state(i, state);
        this->api->writestate(i);
    }
    const auto deadline = std::chrono::steady_clock::now() +
                          std::chrono::microseconds(timeout);
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
        if (std::chrono::steady_clock::now() >= deadline)
            return this->state_error(pending, state);
        std::this_thread::sleep_for(STATE_POLL_INTERVAL.chrono());
    }
}

bool Master::al_error(const uint16_t pos) const {
    return (this->api->slave_state(pos) & EC_STATE_ERROR) != 0 &&
           this->api->slave_ALstatuscode(pos) != 0;
}

x::errors::Error Master::wait_for_frame(const int timeout) {
    // The next frame to come back can have left before the call.
    const auto target = this->frames.load(std::memory_order_acquire) + 2;
    const auto deadline = std::chrono::steady_clock::now() +
                          std::chrono::microseconds(timeout);
    while (this->frames.load(std::memory_order_acquire) < target) {
        if (std::chrono::steady_clock::now() >= deadline)
            return x::errors::Error(
                errors::STATE_CHANGE_ERROR,
                "failed to reach OP: no process data passed the slaves in SAFE-OP"
            );
        std::this_thread::sleep_for(STATE_POLL_INTERVAL.chrono());
    }
    return x::errors::NIL;
}

bool Master::refuses(const uint16_t pos, const uint16_t state) const {
    return this->al_error(pos) &&
           (this->api->slave_state(pos) & ~EC_STATE_ERROR) < state;
}

x::errors::Error
Master::state_error(const std::vector<uint16_t> &failed, const uint16_t state) {
    const auto target = slave::slave_state_to_string(slave::from_al_state(state));
    std::string msg;
    bool refused = false;
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
        refused |= this->refuses(pos, state) &&
                   std::ranges::find(TRANSIENT_FAULTS, code) == TRANSIENT_FAULTS.end();
        dc_sync_refused |= code == AL_INVALID_DC_SYNC;
        msg += ": " + std::string(ec_ALstatuscode2string(code));
    }
    if (dc_sync_refused) msg.append(". ").append(DC_SYNC_ADVICE);
    return x::errors::Error(
        refused ? errors::STATE_CHANGE_REFUSED : errors::STATE_CHANGE_ERROR,
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
