// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <format>
#include <thread>

#include "absl/log/log.h"

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/esi/known_devices.h"
#include "driver/ethercat/slave/slave.h"
#include "driver/ethercat/soem/master.h"
#include "driver/ethercat/soem/util.h"
#include "driver/ethercat/telem/telem.h"

namespace driver::ethercat::soem {
/// Timeout for EtherCAT state transitions (INIT->PRE_OP->SAFE_OP->OP) in microseconds.
const int STATE_CHANGE_TIMEOUT = static_cast<int>(
    (x::telem::SECOND * 2).microseconds()
);
/// Timeout for cyclic process data exchange in microseconds.
const int PROCESSDATA_TIMEOUT = static_cast<int>(x::telem::MILLISECOND.microseconds());
/// Time between state reads while slaves move to OP.
const auto OP_POLL_INTERVAL = x::telem::MILLISECOND;
/// Timeout for one recovery step on a slave in microseconds.
const int RECOVERY_TIMEOUT = static_cast<int>(
    (x::telem::MILLISECOND * 500).microseconds()
);
/// The distributed clock epoch, 2000-01-01, as a Unix timestamp.
const x::telem::TimeStamp DC_EPOCH(x::telem::SECOND * 946684800);

/// SOEM group 0 addresses every slave, so enabled and disabled slaves get their own.
constexpr uint8_t ENABLED_GROUP = 1;
constexpr uint8_t DISABLED_GROUP = 2;
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

Master::Master(std::unique_ptr<API> api, std::string interface_name):
    iface_name(std::move(interface_name)),
    api(std::move(api)),
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
            "no EtherCAT slaves found on " + this->iface_name
        );
    }

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

x::errors::Error Master::activate() {
    if (!this->initialized)
        return x::errors::Error(errors::ACTIVATION_ERROR, "master not initialized");
    if (this->activated)
        return x::errors::Error(errors::ACTIVATION_ERROR, "master already activated");

    {
        std::lock_guard lock(this->mu);
        for (int i = 1; i <= this->api->slave_count(); ++i)
            this->api->set_slave_group(
                i,
                this->disabled_slaves.contains(i) ? DISABLED_GROUP : ENABLED_GROUP
            );
    }

    const int iomap_size = this->api->config_map_group(
        this->iomap.data(),
        ENABLED_GROUP
    );

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

    auto err = this->request_state(EC_STATE_SAFE_OP, STATE_CHANGE_TIMEOUT);
    if (err) return err;

    err = this->request_operational();
    if (err) {
        LOG(WARNING) << "OP transition failed: " << err.message()
                     << "; attempting recovery to SAFE_OP";
        auto recovery_err = this->request_state(EC_STATE_SAFE_OP, STATE_CHANGE_TIMEOUT);
        if (recovery_err) {
            LOG(ERROR) << "Recovery to SAFE_OP failed: " << recovery_err.message()
                       << "; forcing INIT state";
            this->request_state(EC_STATE_INIT, STATE_CHANGE_TIMEOUT);
        }
        return err;
    }

    this->cache_pdo_offsets();
    this->activated = true;
    return x::errors::NIL;
}

void Master::deactivate() {
    if (this->activated) {
        this->request_state(EC_STATE_INIT, STATE_CHANGE_TIMEOUT);
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
    return DC_EPOCH + x::telem::TimeSpan(this->api->dc_time());
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
        if (const auto code = this->api->slave_ALstatuscode(i); h.faulted && code != 0)
            h.al_status = ec_ALstatuscode2string(code);
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

x::errors::Error Master::request_state(const uint16_t state, const int timeout) {
    std::vector<uint16_t> failed;
    for (int i = 1; i <= this->api->slave_count(); ++i) {
        if (this->api->slave_group(i) != ENABLED_GROUP) continue;
        this->api->set_slave_state(i, state);
        this->api->writestate(i);
        if (this->api->statecheck(i, state, timeout) != state) failed.push_back(i);
    }
    if (failed.empty()) return x::errors::NIL;
    return this->state_error(failed, state);
}

x::errors::Error Master::request_operational() {
    // Slaves with outputs need one valid process data frame before they enter OP.
    this->api->send_processdata(ENABLED_GROUP);
    this->api->receive_processdata(ENABLED_GROUP, PROCESSDATA_TIMEOUT);
    for (int i = 1; i <= this->api->slave_count(); ++i) {
        if (this->api->slave_group(i) != ENABLED_GROUP) continue;
        this->api->set_slave_state(i, EC_STATE_OPERATIONAL);
        this->api->writestate(i);
    }
    const auto deadline = std::chrono::steady_clock::now() +
                          std::chrono::microseconds(STATE_CHANGE_TIMEOUT);
    while (true) {
        this->api->send_processdata(ENABLED_GROUP);
        this->api->receive_processdata(ENABLED_GROUP, PROCESSDATA_TIMEOUT);
        this->api->readstate();
        std::vector<uint16_t> pending;
        for (int i = 1; i <= this->api->slave_count(); ++i)
            if (this->api->slave_group(i) == ENABLED_GROUP &&
                this->api->slave_state(i) != EC_STATE_OPERATIONAL)
                pending.push_back(i);
        if (pending.empty()) return x::errors::NIL;
        if (std::chrono::steady_clock::now() >= deadline)
            return this->state_error(pending, EC_STATE_OPERATIONAL);
        std::this_thread::sleep_for(OP_POLL_INTERVAL.chrono());
    }
}

x::errors::Error
Master::state_error(const std::vector<uint16_t> &failed, const uint16_t state) {
    const auto target = slave::slave_state_to_string(slave::from_al_state(state));
    std::string msg;
    for (const auto pos: failed) {
        if (!msg.empty()) msg += "; ";
        msg += "slave " + std::to_string(pos) + " (" + this->api->slave_name(pos) +
               ") is in " +
               slave::slave_state_to_string(
                   slave::from_al_state(this->api->slave_state(pos))
               );
        if (const auto code = this->api->slave_ALstatuscode(pos); code != 0)
            msg += ": " + std::string(ec_ALstatuscode2string(code));
    }
    return x::errors::Error(
        errors::STATE_CHANGE_ERROR,
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
