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
#include "driver/ethercat/soem/master.h"
#include "driver/ethercat/telem/telem.h"

namespace ethercat::soem {
/// Timeout for EtherCAT state transitions (INIT→PRE_OP→SAFE_OP→OP) in microseconds.
const int STATE_CHANGE_TIMEOUT = static_cast<int>((telem::SECOND * 2).microseconds());
/// Timeout for cyclic process data exchange in microseconds.
const int PROCESSDATA_TIMEOUT = static_cast<int>(telem::MILLISECOND.microseconds());
/// IOMap buffer size for process data image. This is local memory only - actual wire
/// traffic is determined by slave PDO mappings. 128KB handles networks with hundreds
/// of slaves or slaves with large PDO mappings (e.g., multi-axis drives).
constexpr size_t DEFAULT_IOMAP_SIZE = 131072;

/// @name EtherCAT Object Dictionary Standard Addresses
/// These are defined by the EtherCAT specification (ETG.1000.6) and are the standard
/// locations for PDO mapping objects in any compliant EtherCAT slave.
/// @{

/// First TxPDO (slave→master) mapping object index. Range: 0x1A00-0x1BFF.
constexpr uint16_t TXPDO_MAPPING_START = 0x1A00;
/// Last TxPDO mapping object index.
constexpr uint16_t TXPDO_MAPPING_END = 0x1BFF;
/// First RxPDO (master→slave) mapping object index. Range: 0x1600-0x17FF.
constexpr uint16_t RXPDO_MAPPING_START = 0x1600;
/// Last RxPDO mapping object index.
constexpr uint16_t RXPDO_MAPPING_END = 0x17FF;

/// @}

Master::Master(std::string interface_name):
    iface_name(std::move(interface_name)),
    context{},
    iomap(DEFAULT_IOMAP_SIZE, 0),
    input_sz(0),
    output_sz(0),
    initialized(false),
    activated(false),
    expected_wkc(0) {
    std::memset(&this->context, 0, sizeof(this->context));
}

Master::~Master() {
    if (this->activated || this->initialized) this->deactivate();
}

xerrors::Error Master::initialize() {
    if (this->initialized) return xerrors::NIL;

    if (ecx_init(&this->context, this->iface_name.c_str()) <= 0)
        return xerrors::Error(
            MASTER_INIT_ERROR,
            "failed to initialize EtherCAT on interface: " + this->iface_name
        );

    if (ecx_config_init(&this->context) <= 0) {
        ecx_close(&this->context);
        return xerrors::Error(MASTER_INIT_ERROR, "no EtherCAT slaves found on network");
    }

    this->populate_slaves();
    this->initialized = true;
    return xerrors::NIL;
}

xerrors::Error Master::register_pdos(const std::vector<PDOEntry> &) {
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
    if (!this->initialized)
        return xerrors::Error(ACTIVATION_ERROR, "master not initialized");
    if (this->activated)
        return xerrors::Error(ACTIVATION_ERROR, "master already activated");

    {
        std::lock_guard lock(this->mu);
        for (const auto pos: this->disabled_slaves)
            if (pos > 0 && pos <= static_cast<uint16_t>(this->context.slavecount))
                this->context.slavelist[pos].group = 1;
    }

    const int iomap_size = ecx_config_map_group(&this->context, this->iomap.data(), 0);

    if (iomap_size <= 0)
        return xerrors::Error(ACTIVATION_ERROR, "failed to configure PDO mapping");

    const auto &group = this->context.grouplist[0];
    this->input_sz = group.Ibytes;
    this->output_sz = group.Obytes;

    this->expected_wkc = (this->context.grouplist[0].outputsWKC * 2) +
                         this->context.grouplist[0].inputsWKC;

    auto err = this->request_state(EC_STATE_SAFE_OP, STATE_CHANGE_TIMEOUT);
    if (err) return err;

    ecx_send_processdata(&this->context);
    ecx_receive_processdata(&this->context, PROCESSDATA_TIMEOUT);

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
    return xerrors::NIL;
}

void Master::deactivate() {
    if (this->activated) {
        this->request_state(EC_STATE_INIT, STATE_CHANGE_TIMEOUT);
        this->activated = false;
        this->expected_wkc = 0;
        this->input_sz = 0;
        this->output_sz = 0;
        this->pdo_offset_cache.clear();
    }

    if (this->initialized) {
        ecx_close(&this->context);
        this->initialized = false;
        this->slave_list.clear();
    }
}

xerrors::Error Master::receive() {
    if (!this->activated) return xerrors::Error(CYCLIC_ERROR, "master not activated");

    const int wkc = ecx_receive_processdata(&this->context, PROCESSDATA_TIMEOUT);

    if (wkc < 0) return xerrors::Error(CYCLIC_ERROR, "process data receive failed");

    if (wkc != this->expected_wkc) {
        std::string failing_slaves;
        for (int i = 1; i <= this->context.slavecount; ++i) {
            if ((this->context.slavelist[i].state & EC_STATE_OPERATIONAL) == 0) {
                if (!failing_slaves.empty()) failing_slaves += ", ";
                failing_slaves += std::to_string(i) + " (" +
                                  std::string(this->context.slavelist[i].name) + ")";
            }
        }
        return xerrors::Error(
            WORKING_COUNTER_ERROR,
            "working counter mismatch: expected " + std::to_string(this->expected_wkc) +
                ", got " + std::to_string(wkc) +
                (failing_slaves.empty() ? "" : "; slaves not in OP: " + failing_slaves)
        );
    }

    return xerrors::NIL;
}

xerrors::Error Master::send() {
    if (!this->activated) return xerrors::Error(CYCLIC_ERROR, "master not activated");

    const int result = ecx_send_processdata(&this->context);

    if (result <= 0) return xerrors::Error(CYCLIC_ERROR, "process data send failed");

    return xerrors::NIL;
}

std::span<const uint8_t> Master::input_data() {
    if (!this->activated) return {};
    return {this->iomap.data() + this->output_sz, this->input_sz};
}

std::span<uint8_t> Master::output_data() {
    if (!this->activated) return {};
    return {this->iomap.data(), this->output_sz};
}

master::PDOOffset Master::pdo_offset(const PDOEntry &entry) const {
    std::lock_guard lock(this->mu);
    PDOEntryKey key{entry.slave_position, entry.index, entry.subindex, entry.is_input};
    auto it = this->pdo_offset_cache.find(key);
    if (it != this->pdo_offset_cache.end()) return it->second;
    return {};
}

std::vector<SlaveInfo> Master::slaves() const {
    std::lock_guard lock(this->mu);
    return this->slave_list;
}

SlaveState Master::slave_state(const uint16_t position) const {
    std::lock_guard lock(this->mu);

    if (position == 0 || position > static_cast<uint16_t>(this->context.slavecount))
        return SlaveState::UNKNOWN;

    return convert_state(this->context.slavelist[position].state);
}

bool Master::all_slaves_operational() const {
    std::lock_guard lock(this->mu);

    if (!this->activated) return false;

    for (int i = 1; i <= this->context.slavecount; ++i)
        if ((this->context.slavelist[i].state & EC_STATE_OPERATIONAL) == 0)
            return false;
    return true;
}

std::string Master::interface_name() const {
    return this->iface_name;
}

SlaveState Master::convert_state(const uint16_t soem_state) {
    const uint16_t state = soem_state & 0x0F;

    switch (state) {
        case EC_STATE_INIT:
            return SlaveState::INIT;
        case EC_STATE_PRE_OP:
            return SlaveState::PRE_OP;
        case EC_STATE_BOOT:
            return SlaveState::BOOT;
        case EC_STATE_SAFE_OP:
            return SlaveState::SAFE_OP;
        case EC_STATE_OPERATIONAL:
            return SlaveState::OP;
        default:
            return SlaveState::UNKNOWN;
    }
}

void Master::populate_slaves() {
    this->slave_list.clear();
    this->slave_list.reserve(this->context.slavecount);

    for (int i = 1; i <= this->context.slavecount; ++i) {
        const auto &soem_slave = this->context.slavelist[i];
        SlaveInfo info{};
        info.position = static_cast<uint16_t>(i);
        info.vendor_id = soem_slave.eep_man;
        info.product_code = soem_slave.eep_id;
        info.revision = soem_slave.eep_rev;
        info.serial = soem_slave.eep_ser;
        info.name = soem_slave.name;
        info.state = convert_state(soem_slave.state);
        info.input_bits = soem_slave.Ibits;
        info.output_bits = soem_slave.Obits;
        this->discover_slave_pdos(info);
        this->slave_list.push_back(info);
    }
}

void Master::cache_pdo_offsets() {
    std::lock_guard lock(this->mu);
    this->pdo_offset_cache.clear();

    for (const auto &slave: this->slave_list) {
        if (slave.position == 0 ||
            slave.position > static_cast<uint16_t>(this->context.slavecount))
            continue;

        if (this->context.slavelist[slave.position].group != 0) continue;

        const auto &soem_slave = this->context.slavelist[slave.position];
        const auto *iomap_base = this->iomap.data();

        const size_t slave_output_offset = soem_slave.outputs != nullptr
                                             ? static_cast<size_t>(
                                                   soem_slave.outputs - iomap_base
                                               )
                                             : 0;

        const size_t slave_input_abs_offset = soem_slave.inputs != nullptr
                                                ? static_cast<size_t>(
                                                      soem_slave.inputs - iomap_base
                                                  )
                                                : this->output_sz;
        const size_t slave_input_offset = slave_input_abs_offset - this->output_sz;

        if (!slave.coe_pdo_order_reliable) {
            LOG(WARNING) << "Slave " << slave.position
                         << " PDO offsets may be incorrect (fallback discovery used)";
        }

        size_t input_bit_offset = 0;
        for (const auto &pdo: slave.input_pdos) {
            PDOEntryKey key{slave.position, pdo.index, pdo.subindex, true};
            this->pdo_offset_cache[key] = {
                slave_input_offset + input_bit_offset / 8,
                static_cast<uint8_t>(input_bit_offset % 8)
            };
            input_bit_offset += pdo.bit_length;
        }

        size_t output_bit_offset = 0;
        for (const auto &pdo: slave.output_pdos) {
            PDOEntryKey key{slave.position, pdo.index, pdo.subindex, false};
            this->pdo_offset_cache[key] = {
                slave_output_offset + output_bit_offset / 8,
                static_cast<uint8_t>(output_bit_offset % 8)
            };
            output_bit_offset += pdo.bit_length;
        }
    }
}

/// Timeout for SDO (Service Data Object) reads during PDO discovery in microseconds.
const int SDO_TIMEOUT = static_cast<int>((telem::MILLISECOND * 700).microseconds());
constexpr int MAX_SDO_FAILURES = 3;

void Master::discover_slave_pdos(SlaveInfo &slave) {
    if (esi::lookup_device_pdos(
            slave.vendor_id,
            slave.product_code,
            slave.revision,
            slave
        )) {
        VLOG(1) << "Slave " << slave.position
                << " PDOs discovered via ESI: " << slave.input_pdos.size()
                << " inputs, " << slave.output_pdos.size() << " outputs";
        return;
    }

    if (!this->discover_pdos_coe(slave)) this->discover_pdos_sii(slave);
}

bool Master::discover_pdos_coe(SlaveInfo &slave) {
    const auto &soem_slave = this->context.slavelist[slave.position];
    VLOG(2) << "Slave " << slave.position << " mbx_proto: 0x" << std::hex
            << static_cast<int>(soem_slave.mbx_proto) << std::dec;
    if ((soem_slave.mbx_proto & ECT_MBXPROT_COE) == 0) {
        VLOG(1) << "Slave " << slave.position << " (" << slave.name
                << ") does not support CoE, falling back to SII";
        return false;
    }

    VLOG(2) << "Slave " << slave.position << " supports CoE, reading PDO assignments";

    bool tx_success = false;
    bool rx_success = false;

    bool tx_order_reliable = false;
    bool rx_order_reliable = false;

    auto err = this->read_pdo_assign(slave.position, ECT_SDO_TXPDOASSIGN, true, slave);
    if (err) {
        VLOG(2) << "Failed to read TxPDO assignment for slave " << slave.position
                << ": " << err.message() << " - trying direct PDO read";
        err = this->read_pdo_mapping(slave.position, TXPDO_MAPPING_START, true, slave);
        if (err)
            VLOG(2) << "Failed to read TxPDO mapping 0x" << std::hex
                    << TXPDO_MAPPING_START << std::dec << " for slave "
                    << slave.position << ": " << err.message();
        else
            tx_success = true;
    } else {
        tx_success = true;
        tx_order_reliable = true;
    }

    err = this->read_pdo_assign(slave.position, ECT_SDO_RXPDOASSIGN, false, slave);
    if (err) {
        VLOG(2) << "Failed to read RxPDO assignment for slave " << slave.position
                << ": " << err.message() << " - trying direct PDO read";
        err = this->read_pdo_mapping(slave.position, RXPDO_MAPPING_START, false, slave);
        if (err)
            VLOG(2) << "Failed to read RxPDO mapping 0x" << std::hex
                    << RXPDO_MAPPING_START << std::dec << " for slave "
                    << slave.position << ": " << err.message();
        else
            rx_success = true;
    } else {
        rx_success = true;
        rx_order_reliable = true;
    }

    if (!tx_success && !rx_success) {
        VLOG(2) << "Standard PDO discovery failed for slave " << slave.position
                << ", scanning object dictionary";
        if (this->scan_object_dictionary_for_pdos(slave.position, slave)) {
            tx_success = !slave.input_pdos.empty();
            rx_success = !slave.output_pdos.empty();
        }
    }

    if (!tx_success && !rx_success) {
        slave.pdo_discovery_error = "no PDO objects found in object dictionary";
        return false;
    }

    slave.pdos_discovered = true;
    slave.coe_pdo_order_reliable = tx_order_reliable && rx_order_reliable;
    VLOG(1) << "Slave " << slave.position
            << " PDOs discovered via CoE: " << slave.input_pdos.size() << " inputs, "
            << slave.output_pdos.size() << " outputs";
    return true;
}

void Master::discover_pdos_sii(SlaveInfo &slave) {
    VLOG(1) << "Using SII fallback for slave " << slave.position;

    for (uint8_t t = 0; t <= 1; ++t) {
        const bool is_input = (t == 0);
        const uint16_t startpos = ecx_siifind(
            &this->context,
            slave.position,
            ECT_SII_PDO + t
        );
        VLOG(2) << "Slave " << slave.position << " SII category " << (ECT_SII_PDO + t)
                << " (" << (is_input ? "TxPDO" : "RxPDO") << ") startpos: " << startpos;
        if (startpos == 0) continue;

        uint16_t a = startpos;
        uint16_t length = ecx_siigetbyte(&this->context, slave.position, a++);
        length += (ecx_siigetbyte(&this->context, slave.position, a++) << 8);

        VLOG(2) << "Slave " << slave.position << " " << (is_input ? "TxPDO" : "RxPDO")
                << " length: " << length;

        uint16_t c = 1;
        while (c < length) {
            uint16_t pdo_index = ecx_siigetbyte(&this->context, slave.position, a++);
            pdo_index += (ecx_siigetbyte(&this->context, slave.position, a++) << 8);
            c++;

            const uint8_t num_entries = ecx_siigetbyte(
                &this->context,
                slave.position,
                a++
            );
            const uint8_t sync_manager = ecx_siigetbyte(
                &this->context,
                slave.position,
                a++
            );
            a++;
            const uint8_t pdo_name_idx = ecx_siigetbyte(
                &this->context,
                slave.position,
                a++
            );
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
                uint16_t obj_idx = ecx_siigetbyte(&this->context, slave.position, a++);
                obj_idx += (ecx_siigetbyte(&this->context, slave.position, a++) << 8);
                const uint8_t obj_subidx = ecx_siigetbyte(
                    &this->context,
                    slave.position,
                    a++
                );
                const uint8_t obj_name_idx = ecx_siigetbyte(
                    &this->context,
                    slave.position,
                    a++
                );
                a++;
                const uint8_t bitlen = ecx_siigetbyte(
                    &this->context,
                    slave.position,
                    a++
                );
                a += 2;

                if (obj_idx == 0 && obj_subidx == 0) continue;

                std::string entry_name;
                if (obj_name_idx > 0) {
                    char name_buf[EC_MAXNAME + 1] = {0};
                    ecx_siistring(
                        &this->context,
                        name_buf,
                        slave.position,
                        obj_name_idx
                    );
                    entry_name = name_buf;
                }

                const telem::DataType data_type = infer_type_from_bit_length(bitlen);
                const std::string name = generate_pdo_entry_name(
                    entry_name,
                    obj_idx,
                    obj_subidx,
                    is_input,
                    data_type
                );

                PDOEntryInfo entry(
                    pdo_index,
                    obj_idx,
                    obj_subidx,
                    bitlen,
                    is_input,
                    name,
                    data_type
                );

                if (is_input)
                    slave.input_pdos.push_back(entry);
                else
                    slave.output_pdos.push_back(entry);
            }
            c++;
        }
    }

    if (slave.input_pdos.empty() && slave.output_pdos.empty()) {
        const auto &soem_slave = this->context.slavelist[slave.position];
        if (soem_slave.Ibits > 0 || soem_slave.Obits > 0) {
            VLOG(2) << "Slave " << slave.position << " has Ibits=" << soem_slave.Ibits
                    << " Obits=" << soem_slave.Obits << " but no SII PDO info";
            slave.pdo_discovery_error = "PDO details not available (I/O size: " +
                                        std::to_string(soem_slave.Ibits) +
                                        " input bits, " +
                                        std::to_string(soem_slave.Obits) +
                                        " output bits)";
        } else {
            slave.pdo_discovery_error = "no PDOs found";
        }
    }

    slave.pdos_discovered = true;
    VLOG(1) << "Slave " << slave.position
            << " PDOs discovered via SII: " << slave.input_pdos.size() << " inputs, "
            << slave.output_pdos.size() << " outputs";
}

xerrors::Error Master::read_pdo_assign(
    const uint16_t slave_pos,
    const uint16_t assign_index,
    const bool is_input,
    SlaveInfo &slave
) {
    uint8_t num_pdos = 0;
    int size = sizeof(num_pdos);
    int result = ecx_SDOread(
        &this->context,
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
                << " ecx_err=" << this->context.slavelist[slave_pos].ALstatuscode;
        return xerrors::Error(SDO_READ_ERROR, "failed to read PDO assignment count");
    }
    VLOG(2) << "Slave " << slave_pos << " PDO assign 0x" << std::hex << assign_index
            << " has " << std::dec << static_cast<int>(num_pdos) << " PDOs";

    int consecutive_failures = 0;
    for (uint8_t i = 1; i <= num_pdos; ++i) {
        uint16_t pdo_index = 0;
        size = sizeof(pdo_index);
        result = ecx_SDOread(
            &this->context,
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
                return xerrors::Error(
                    SDO_READ_ERROR,
                    "too many consecutive SDO failures"
                );
            continue;
        }
        consecutive_failures = 0;

        if (pdo_index == 0) continue;

        auto err = this->read_pdo_mapping(slave_pos, pdo_index, is_input, slave);
        if (err)
            VLOG(2) << "Failed to read PDO mapping 0x" << std::hex << pdo_index
                    << " for slave " << std::dec << slave_pos << ": " << err.message();
    }

    return xerrors::NIL;
}

xerrors::Error Master::read_pdo_mapping(
    const uint16_t slave_pos,
    const uint16_t pdo_index,
    const bool is_input,
    SlaveInfo &slave
) {
    uint8_t num_entries = 0;
    int size = sizeof(num_entries);
    int result = ecx_SDOread(
        &this->context,
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
        return xerrors::Error(SDO_READ_ERROR, "failed to read PDO mapping count");
    }
    VLOG(2) << "Slave " << slave_pos << " PDO 0x" << std::hex << pdo_index << " has "
            << std::dec << static_cast<int>(num_entries) << " entries";

    int consecutive_failures = 0;
    for (uint8_t i = 1; i <= num_entries; ++i) {
        uint32_t mapping = 0;
        size = sizeof(mapping);
        result = ecx_SDOread(
            &this->context,
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
                return xerrors::Error(
                    SDO_READ_ERROR,
                    "too many consecutive SDO failures"
                );
            continue;
        }
        consecutive_failures = 0;

        if (mapping == 0) continue;

        const uint16_t index = static_cast<uint16_t>((mapping >> 16) & 0xFFFF);
        const uint8_t subindex = static_cast<uint8_t>((mapping >> 8) & 0xFF);
        const uint8_t bit_length = static_cast<uint8_t>(mapping & 0xFF);

        if (index == 0 && subindex == 0) continue;

        const std::string coe_name = this->read_pdo_entry_name(
            slave_pos,
            index,
            subindex
        );
        const telem::DataType data_type = infer_type_from_bit_length(bit_length);
        const std::string name = generate_pdo_entry_name(
            coe_name,
            index,
            subindex,
            is_input,
            data_type
        );

        PDOEntryInfo
            entry(pdo_index, index, subindex, bit_length, is_input, name, data_type);

        if (is_input)
            slave.input_pdos.push_back(entry);
        else
            slave.output_pdos.push_back(entry);
    }

    return xerrors::NIL;
}

std::string Master::read_pdo_entry_name(
    const uint16_t slave_pos,
    const uint16_t index,
    const uint8_t subindex
) {
    ec_ODlistt od_list{};
    od_list.Slave = slave_pos;
    od_list.Index[0] = index;
    od_list.Entries = 1;

    ec_OElistt oe_list{};
    if (ecx_readOEsingle(&this->context, 0, subindex, &od_list, &oe_list) > 0)
        if (oe_list.Entries > 0 && oe_list.Name[0][0] != '\0')
            return std::string(oe_list.Name[0]);

    return "";
}

bool Master::scan_object_dictionary_for_pdos(
    const uint16_t slave_pos,
    SlaveInfo &slave
) {
    ec_ODlistt od_list{};
    std::memset(&od_list, 0, sizeof(od_list));
    od_list.Slave = slave_pos;

    if (ecx_readODlist(&this->context, slave_pos, &od_list) <= 0) {
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

xerrors::Error Master::request_state(const uint16_t state, const int timeout) {
    int success_count = 0;
    int group0_count = 0;
    std::string error_msg;

    for (int i = 1; i <= this->context.slavecount; ++i) {
        if (this->context.slavelist[i].group != 0) continue;
        group0_count++;

        this->context.slavelist[i].state = state;
        ecx_writestate(&this->context, i);

        const uint16_t result = ecx_statecheck(&this->context, i, state, timeout);

        if ((result & 0x0F) == (state & 0x0F)) {
            success_count++;
        } else {
            if (!error_msg.empty()) error_msg += "; ";
            error_msg += "slave " + std::to_string(i) + " in state " +
                         std::to_string(result);
            if (result & EC_STATE_ERROR) {
                error_msg += " (ERROR flag set, AL status code: " +
                             std::to_string(this->context.slavelist[i].ALstatuscode) +
                             ")";
            }
        }
    }

    if (success_count < group0_count)
        return xerrors::Error(
            STATE_CHANGE_ERROR,
            "state transition failed: " + std::to_string(success_count) + "/" +
                std::to_string(group0_count) + " slaves reached state " +
                std::to_string(state) + "; " + error_msg
        );

    return xerrors::NIL;
}
}
