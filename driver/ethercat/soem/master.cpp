// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <sstream>
#include <unordered_map>

#include "glog/logging.h"

#include "driver/ethercat/esi/known_devices.h"
#include "driver/ethercat/slave/slave.h"
#include "driver/ethercat/soem/master.h"
#include "driver/ethercat/telem/telem.h"

namespace driver::ethercat::soem {
/// Timeout for EtherCAT state transitions (INIT->PRE_OP->SAFE_OP->OP) in microseconds.
const int STATE_CHANGE_TIMEOUT = static_cast<int>(
    (x::telem::SECOND * 2).microseconds()
);
/// Timeout for cyclic process data exchange in microseconds.
const int PROCESSDATA_TIMEOUT = static_cast<int>(x::telem::MILLISECOND.microseconds());
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

Master::~Master() {
    if (this->activated || this->initialized) this->deactivate();
}

x::errors::Error Master::initialize() {
    if (this->initialized) return x::errors::NIL;

    if (this->api->init(this->iface_name.c_str()) <= 0)
        return x::errors::Error(
            errors::MASTER_INIT_ERROR,
            "failed to initialize EtherCAT on interface: " + this->iface_name
        );

    if (this->api->config_init() <= 0) {
        this->api->close();
        return x::errors::Error(
            errors::MASTER_INIT_ERROR,
            "no EtherCAT slaves found on network"
        );
    }

    this->populate_slaves();
    this->initialized = true;
    return x::errors::NIL;
}

x::errors::Error Master::register_pdos(const std::vector<pdo::Entry> &) {
    return x::errors::NIL;
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
        for (const auto pos: this->disabled_slaves)
            if (pos > 0 && pos <= static_cast<uint16_t>(this->api->slave_count()))
                this->api->set_slave_group(pos, 1);
    }

    const int iomap_size = this->api->config_map_group(this->iomap.data(), 0);

    if (iomap_size <= 0)
        return x::errors::Error(
            errors::ACTIVATION_ERROR,
            "failed to configure PDO mapping"
        );

    this->input_sz = this->api->group_Ibytes(0);
    this->output_sz = this->api->group_Obytes(0);

    this->expected_wkc = (this->api->group_outputsWKC(0) * 2) +
                         this->api->group_inputsWKC(0);

    auto err = this->request_state(EC_STATE_SAFE_OP, STATE_CHANGE_TIMEOUT);
    if (err) return err;

    this->api->send_processdata();
    this->api->receive_processdata(PROCESSDATA_TIMEOUT);

    err = this->request_state(EC_STATE_OPERATIONAL, STATE_CHANGE_TIMEOUT);
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
        this->input_sz = 0;
        this->output_sz = 0;
        this->pdo_offsets.clear();
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

    const int wkc = this->api->receive_processdata(PROCESSDATA_TIMEOUT);

    if (wkc < 0)
        return x::errors::Error(errors::CYCLIC_ERROR, "process data receive failed");

    if (wkc != this->expected_wkc) {
        std::string failing_slaves;
        for (int i = 1; i <= this->api->slave_count(); ++i) {
            if ((this->api->slave_state(i) & EC_STATE_OPERATIONAL) == 0) {
                if (!failing_slaves.empty()) failing_slaves += ", ";
                failing_slaves += std::to_string(i) + " (" + this->api->slave_name(i) +
                                  ")";
            }
        }
        return x::errors::Error(
            errors::WORKING_COUNTER_ERROR,
            "working counter mismatch: expected " + std::to_string(this->expected_wkc) +
                ", got " + std::to_string(wkc) +
                (failing_slaves.empty() ? "" : "; slaves not in OP: " + failing_slaves)
        );
    }

    return x::errors::NIL;
}

x::errors::Error Master::send() {
    if (!this->activated)
        return x::errors::Error(errors::CYCLIC_ERROR, "master not activated");

    const int result = this->api->send_processdata();

    if (result <= 0)
        return x::errors::Error(errors::CYCLIC_ERROR, "process data send failed");

    return x::errors::NIL;
}

std::span<const uint8_t> Master::input_data() {
    if (!this->activated) return {};
    return {this->iomap.data() + this->output_sz, this->input_sz};
}

std::span<uint8_t> Master::output_data() {
    if (!this->activated) return {};
    return {this->iomap.data(), this->output_sz};
}

pdo::Offset Master::pdo_offset(const pdo::Entry &entry) const {
    std::lock_guard lock(this->mu);
    return pdo::find_offset(this->pdo_offsets, entry);
}

std::vector<slave::DiscoveryResult> Master::slaves() const {
    std::lock_guard lock(this->mu);
    return this->slave_list;
}

slave::State Master::slave_state(const uint16_t position) const {
    std::lock_guard lock(this->mu);

    if (position == 0 || position > static_cast<uint16_t>(this->api->slave_count()))
        return slave::State::UNKNOWN;

    return slave::from_al_state(this->api->slave_state(position));
}

bool Master::all_slaves_operational() const {
    std::lock_guard lock(this->mu);

    if (!this->activated) return false;

    for (int i = 1; i <= this->api->slave_count(); ++i)
        if ((this->api->slave_state(i) & EC_STATE_OPERATIONAL) == 0) return false;
    return true;
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

    for (const auto &slave: this->slave_list) {
        const auto &props = slave.properties;
        if (props.position == 0 ||
            props.position > static_cast<uint16_t>(this->api->slave_count()))
            continue;

        if (this->api->slave_group(props.position) != 0) continue;

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
    if (esi::lookup_device_pdos(
            props.vendor_id,
            props.product_code,
            props.revision,
            props
        )) {
        VLOG(1) << "Slave " << props.position
                << " PDOs discovered via ESI: " << props.input_pdos.size()
                << " inputs, " << props.output_pdos.size() << " outputs";
        slave.status.pdos_discovered = true;
        props.coe_pdo_order_reliable = true;
        return;
    }

    if (!this->discover_pdos_coe(slave)) this->discover_pdos_sii(slave);
}

bool Master::discover_pdos_coe(slave::DiscoveryResult &slave) {
    auto &props = slave.properties;
    auto &status = slave.status;
    VLOG(2) << "Slave " << props.position << " mbx_proto: 0x" << std::hex
            << static_cast<int>(this->api->slave_mbx_proto(props.position)) << std::dec;
    if ((this->api->slave_mbx_proto(props.position) & ECT_MBXPROT_COE) == 0) {
        VLOG(1) << "Slave " << props.position << " (" << props.name
                << ") does not support CoE, falling back to SII";
        return false;
    }

    VLOG(2) << "Slave " << props.position << " supports CoE, reading PDO assignments";

    bool tx_success = false;
    bool rx_success = false;

    bool tx_order_reliable = false;
    bool rx_order_reliable = false;

    auto err = this->read_pdo_assign(props.position, ECT_SDO_TXPDOASSIGN, true, slave);
    if (err) {
        VLOG(2) << "Failed to read TxPDO assignment for slave " << props.position
                << ": " << err.message() << " - trying direct PDO read";
        err = this->read_pdo_mapping(props.position, TXPDO_MAPPING_START, true, slave);
        if (err)
            VLOG(2) << "Failed to read TxPDO mapping 0x" << std::hex
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
        VLOG(2) << "Failed to read RxPDO assignment for slave " << props.position
                << ": " << err.message() << " - trying direct PDO read";
        err = this->read_pdo_mapping(props.position, RXPDO_MAPPING_START, false, slave);
        if (err)
            VLOG(2) << "Failed to read RxPDO mapping 0x" << std::hex
                    << RXPDO_MAPPING_START << std::dec << " for slave "
                    << props.position << ": " << err.message();
        else
            rx_success = true;
    } else {
        rx_success = true;
        rx_order_reliable = true;
    }

    if (!tx_success && !rx_success) {
        VLOG(2) << "Standard PDO discovery failed for slave " << props.position
                << ", scanning object dictionary";
        if (this->scan_object_dictionary_for_pdos(props.position, slave)) {
            tx_success = !props.input_pdos.empty();
            rx_success = !props.output_pdos.empty();
        }
    }

    if (!tx_success && !rx_success) {
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
        VLOG(2) << "Slave " << props.position << " SII category " << (ECT_SII_PDO + t)
                << " (" << (is_input ? "TxPDO" : "RxPDO") << ") startpos: " << startpos;
        if (startpos == 0) continue;

        uint16_t a = startpos;
        uint16_t length = this->api->siigetbyte(props.position, a++);
        length += (this->api->siigetbyte(props.position, a++) << 8);

        VLOG(2) << "Slave " << props.position << " " << (is_input ? "TxPDO" : "RxPDO")
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

            (void) pdo_name_idx;

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
                a++;
                const uint8_t bitlen = this->api->siigetbyte(props.position, a++);
                a += 2;

                if (obj_idx == 0 && obj_subidx == 0) continue;

                std::string entry_name;
                if (obj_name_idx > 0) {
                    char name_buf[EC_MAXNAME + 1] = {0};
                    this->api->siistring(name_buf, props.position, obj_name_idx);
                    entry_name = name_buf;
                }

                const x::telem::DataType data_type = telem::infer_type_from_bit_length(
                    bitlen
                );
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
                    data_type
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
            VLOG(2) << "Slave " << props.position
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
        VLOG(2) << "Slave " << slave_pos << " SDO read 0x" << std::hex << assign_index
                << ":0 failed, result=" << std::dec << result
                << " ecx_err=" << this->api->slave_ALstatuscode(slave_pos);
        return x::errors::Error(
            errors::SDO_READ_ERROR,
            "failed to read PDO assignment count"
        );
    }
    VLOG(2) << "Slave " << slave_pos << " PDO assign 0x" << std::hex << assign_index
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
            VLOG(2) << "Failed to read PDO mapping 0x" << std::hex << pdo_index
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
        VLOG(2) << "Slave " << slave_pos << " SDO read 0x" << std::hex << pdo_index
                << ":0 failed, result=" << std::dec << result;
        return x::errors::Error(
            errors::SDO_READ_ERROR,
            "failed to read PDO mapping count"
        );
    }
    VLOG(2) << "Slave " << slave_pos << " PDO 0x" << std::hex << pdo_index << " has "
            << std::dec << static_cast<int>(num_entries) << " entries";

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

        const uint16_t index = static_cast<uint16_t>((mapping >> 16) & 0xFFFF);
        const uint8_t sub_index = static_cast<uint8_t>((mapping >> 8) & 0xFF);
        const uint8_t bit_length = static_cast<uint8_t>(mapping & 0xFF);

        if (index == 0 && sub_index == 0) continue;

        const std::string coe_name = this->read_pdo_entry_name(
            slave_pos,
            index,
            sub_index
        );
        const x::telem::DataType data_type = telem::infer_type_from_bit_length(
            bit_length
        );
        const std::string name = telem::generate_pdo_entry_name(
            coe_name,
            index,
            sub_index,
            is_input,
            data_type
        );

        pdo::Properties
            entry(pdo_index, index, sub_index, bit_length, is_input, name, data_type);

        if (is_input)
            props.input_pdos.push_back(entry);
        else
            props.output_pdos.push_back(entry);
    }

    return x::errors::NIL;
}

std::string Master::read_pdo_entry_name(
    const uint16_t slave_pos,
    const uint16_t index,
    const uint8_t sub_index
) {
    ec_ODlistt od_list{};
    od_list.Slave = slave_pos;
    od_list.Index[0] = index;
    od_list.Entries = 1;

    ec_OElistt oe_list{};
    if (this->api->readOEsingle(0, sub_index, &od_list, &oe_list) > 0)
        if (oe_list.Entries > 0 && oe_list.Name[0][0] != '\0')
            return std::string(oe_list.Name[0]);

    return "";
}

bool Master::scan_object_dictionary_for_pdos(
    const uint16_t slave_pos,
    slave::DiscoveryResult &slave
) {
    ec_ODlistt od_list{};
    std::memset(&od_list, 0, sizeof(od_list));
    od_list.Slave = slave_pos;

    if (this->api->readODlist(slave_pos, &od_list) <= 0) {
        VLOG(2) << "Slave " << slave_pos << " failed to read object dictionary list";
        return false;
    }

    VLOG(2) << "Slave " << slave_pos << " object dictionary has " << od_list.Entries
            << " entries";

    std::vector<uint16_t> txpdo_indices;
    std::vector<uint16_t> rxpdo_indices;

    for (uint16_t i = 0; i < od_list.Entries; ++i) {
        const uint16_t index = od_list.Index[i];
        if (index >= TXPDO_MAPPING_START && index <= TXPDO_MAPPING_END) {
            txpdo_indices.push_back(index);
            VLOG(2) << "Slave " << slave_pos << " found TxPDO object 0x" << std::hex
                    << index << std::dec;
        } else if (index >= RXPDO_MAPPING_START && index <= RXPDO_MAPPING_END) {
            rxpdo_indices.push_back(index);
            VLOG(2) << "Slave " << slave_pos << " found RxPDO object 0x" << std::hex
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

    VLOG(2) << "Slave " << slave_pos << " OD scan found " << txpdo_indices.size()
            << " TxPDO objects, " << rxpdo_indices.size() << " RxPDO objects";

    return found_any;
}

x::errors::Error Master::request_state(const uint16_t state, const int timeout) {
    int success_count = 0;
    int group0_count = 0;
    std::string error_msg;

    for (int i = 1; i <= this->api->slave_count(); ++i) {
        if (this->api->slave_group(i) != 0) continue;
        group0_count++;

        this->api->set_slave_state(i, state);
        this->api->writestate(i);

        const uint16_t result = this->api->statecheck(i, state, timeout);

        if ((result & 0x0F) == (state & 0x0F)) {
            success_count++;
        } else {
            if (!error_msg.empty()) error_msg += "; ";
            error_msg += "slave " + std::to_string(i) + " in state " +
                         std::to_string(result);
            if (result & EC_STATE_ERROR) {
                error_msg += " (ERROR flag set, AL status code: " +
                             std::to_string(this->api->slave_ALstatuscode(i)) + ")";
            }
        }
    }

    if (success_count < group0_count)
        return x::errors::Error(
            errors::STATE_CHANGE_ERROR,
            "state transition failed: " + std::to_string(success_count) + "/" +
                std::to_string(group0_count) + " slaves reached state " +
                std::to_string(state) + "; " + error_msg
        );

    return x::errors::NIL;
}

std::vector<master::Info> Manager::enumerate() {
    std::vector<master::Info> masters;
    ec_adaptert *adapter = ec_find_adapters();
    ec_adaptert *current = adapter;

    while (current != nullptr) {
        if (is_physical_interface(current->name)) {
            master::Info info;
            info.key = current->name;
            info.description = current->desc;
            masters.push_back(std::move(info));
        } else {
            VLOG(2) << "[ethercat] skipping virtual interface: " << current->name;
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
    if (key.size() >= 4 && key.substr(0, 4) == "igh:")
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
