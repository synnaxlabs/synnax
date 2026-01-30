// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>

#include "glog/logging.h"

#include "driver/ethercat/pdo_types.h"
#include "driver/ethercat/soem/master.h"

namespace ethercat::soem {

////////////////////////////////////////////////////////////////////////////////
// Domain Implementation
////////////////////////////////////////////////////////////////////////////////

Domain::Domain(const size_t iomap_size):
    iomap(iomap_size, 0),
    input_offset(0),
    output_offset(0),
    input_sz(0),
    output_sz(0) {}

std::pair<size_t, xerrors::Error> Domain::register_pdo(const PDOEntry &entry) {
    size_t offset;
    const size_t byte_size = (entry.bit_length + 7) / 8;

    if (entry.is_input) {
        offset = this->output_sz + this->input_offset;
        this->input_offset += byte_size;
    } else {
        offset = this->output_offset;
        this->output_offset += byte_size;
    }

    if (offset + byte_size > this->iomap.size()) {
        return {0, xerrors::Error(PDO_MAPPING_ERROR, "IOmap buffer overflow")};
    }

    this->registered_pdos.emplace_back(entry, offset);
    return {offset, xerrors::NIL};
}

uint8_t *Domain::data() {
    return this->iomap.data();
}

size_t Domain::size() const {
    return this->iomap.size();
}

size_t Domain::input_size() const {
    return this->input_sz;
}

size_t Domain::output_size() const {
    return this->output_sz;
}

void Domain::set_sizes(const size_t input_size, const size_t output_size) {
    this->input_sz = input_size;
    this->output_sz = output_size;
}

////////////////////////////////////////////////////////////////////////////////
// Master Implementation
////////////////////////////////////////////////////////////////////////////////

/// Default timeout for state transitions (2 seconds in microseconds).
constexpr int STATE_CHANGE_TIMEOUT = 2000000;

/// Default timeout for process data receive (1 millisecond in microseconds).
constexpr int PROCESSDATA_TIMEOUT = 1000;

Master::Master(std::string interface_name):
    iface_name(std::move(interface_name)),
    context{},
    initialized(false),
    activated(false),
    dom(nullptr),
    expected_wkc(0) {
    std::memset(&this->context, 0, sizeof(this->context));
}

Master::~Master() {
    if (this->activated || this->initialized) this->deactivate();
}

xerrors::Error Master::initialize() {
    if (this->initialized) return xerrors::NIL;

    if (ecx_init(&this->context, this->iface_name.c_str()) <= 0) {
        return xerrors::Error(
            MASTER_INIT_ERROR,
            "failed to initialize EtherCAT on interface: " + this->iface_name
        );
    }

    if (ecx_config_init(&this->context) <= 0) {
        ecx_close(&this->context);
        return xerrors::Error(MASTER_INIT_ERROR, "no EtherCAT slaves found on network");
    }

    this->populate_slaves();
    this->initialized = true;
    return xerrors::NIL;
}

std::unique_ptr<ethercat::Domain> Master::create_domain() {
    return std::make_unique<Domain>();
}

xerrors::Error Master::activate() {
    if (!this->initialized) {
        return xerrors::Error(ACTIVATION_ERROR, "master not initialized");
    }
    if (this->activated) {
        return xerrors::Error(ACTIVATION_ERROR, "master already activated");
    }

    int excluded_count = 0;
    for (int i = 1; i <= this->context.slavecount; ++i) {
        if (this->context.slavelist[i].eep_id == 0x000000FC) {
            this->context.slavelist[i].group = 1;
            excluded_count++;
        } else {
            this->context.slavelist[i].group = 0;
        }
    }

    auto domain = std::make_unique<Domain>();
    const int iomap_size = ecx_config_map_group(&this->context, domain->iomap_ptr(), 0);

    if (iomap_size <= 0 && excluded_count < this->context.slavecount) {
        return xerrors::Error(ACTIVATION_ERROR, "failed to configure PDO mapping");
    }

    const auto &group = this->context.grouplist[0];
    domain->set_sizes(group.Ibytes, group.Obytes);
    this->dom = domain.release();
    this->expected_wkc = (this->context.grouplist[0].outputsWKC * 2) +
                         this->context.grouplist[0].inputsWKC;

    auto err = this->request_state(EC_STATE_SAFE_OP, STATE_CHANGE_TIMEOUT);
    if (err) {
        delete this->dom;
        this->dom = nullptr;
        return err;
    }

    ecx_send_processdata(&this->context);
    ecx_receive_processdata(&this->context, PROCESSDATA_TIMEOUT);

    err = this->request_state(EC_STATE_OPERATIONAL, STATE_CHANGE_TIMEOUT);
    if (err) {
        this->request_state(EC_STATE_SAFE_OP, STATE_CHANGE_TIMEOUT);
        delete this->dom;
        this->dom = nullptr;
        return err;
    }

    this->activated = true;
    return xerrors::NIL;
}

void Master::deactivate() {
    if (this->activated) {
        this->request_state(EC_STATE_INIT, STATE_CHANGE_TIMEOUT);
        delete this->dom;
        this->dom = nullptr;
        this->activated = false;
        this->expected_wkc = 0;
    }

    if (this->initialized) {
        ecx_close(&this->context);
        this->initialized = false;
        this->slave_list.clear();
    }
}

xerrors::Error Master::receive() {
    if (!this->activated) {
        return xerrors::Error(CYCLIC_ERROR, "master not activated");
    }

    const int wkc = ecx_receive_processdata(&this->context, PROCESSDATA_TIMEOUT);

    if (wkc < 0) { return xerrors::Error(CYCLIC_ERROR, "process data receive failed"); }

    if (wkc != this->expected_wkc) {
        return xerrors::Error(
            WORKING_COUNTER_ERROR,
            "working counter mismatch: expected " + std::to_string(this->expected_wkc) +
                ", got " + std::to_string(wkc)
        );
    }

    return xerrors::NIL;
}

xerrors::Error Master::process(ethercat::Domain &domain) {
    (void) domain;
    return xerrors::NIL;
}

xerrors::Error Master::queue(ethercat::Domain &domain) {
    (void) domain;
    return xerrors::NIL;
}

xerrors::Error Master::send() {
    if (!this->activated) {
        return xerrors::Error(CYCLIC_ERROR, "master not activated");
    }

    const int result = ecx_send_processdata(&this->context);

    if (result <= 0) {
        return xerrors::Error(CYCLIC_ERROR, "process data send failed");
    }

    return xerrors::NIL;
}

std::vector<SlaveInfo> Master::slaves() const {
    std::lock_guard lock(this->mu);
    return this->slave_list;
}

SlaveState Master::slave_state(const uint16_t position) const {
    std::lock_guard lock(this->mu);

    if (position == 0 || position > static_cast<uint16_t>(this->context.slavecount)) {
        return SlaveState::UNKNOWN;
    }

    return convert_state(this->context.slavelist[position].state);
}

bool Master::all_slaves_operational() const {
    std::lock_guard lock(this->mu);

    if (!this->activated) return false;

    for (int i = 1; i <= this->context.slavecount; ++i) {
        if ((this->context.slavelist[i].state & EC_STATE_OPERATIONAL) == 0) {
            return false;
        }
    }
    return true;
}

std::string Master::interface_name() const {
    return this->iface_name;
}

ethercat::Domain *Master::active_domain() const {
    return this->dom;
}

SlaveDataOffsets Master::slave_data_offsets(const uint16_t position) const {
    std::lock_guard lock(this->mu);

    if (!this->activated || position == 0 ||
        position > static_cast<uint16_t>(this->context.slavecount)) {
        return SlaveDataOffsets{};
    }

    if (this->context.slavelist[position].group != 0) { return SlaveDataOffsets{}; }

    const auto &slave = this->context.slavelist[position];

    const auto *iomap_base = this->dom->data();
    const size_t output_offset = slave.outputs != nullptr
                                   ? static_cast<size_t>(slave.outputs - iomap_base)
                                   : 0;
    const size_t input_offset = slave.inputs != nullptr
                                  ? static_cast<size_t>(slave.inputs - iomap_base)
                                  : 0;

    return SlaveDataOffsets{
        input_offset,
        static_cast<size_t>(slave.Ibytes),
        output_offset,
        static_cast<size_t>(slave.Obytes)
    };
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
        const auto &slave = this->context.slavelist[i];
        SlaveInfo info{};
        info.position = static_cast<uint16_t>(i);
        info.vendor_id = slave.eep_man;
        info.product_code = slave.eep_id;
        info.revision = slave.eep_rev;
        info.serial = slave.eep_ser;
        info.name = slave.name;
        info.state = convert_state(slave.state);
        this->discover_slave_pdos(info);
        this->slave_list.push_back(info);
    }
}

constexpr int SDO_TIMEOUT = 700000;
constexpr int MAX_SDO_FAILURES = 3;

void Master::discover_slave_pdos(SlaveInfo &slave) {
    if (!this->discover_pdos_coe(slave)) this->discover_pdos_sii(slave);
}

bool Master::discover_pdos_coe(SlaveInfo &slave) {
    const auto &soem_slave = this->context.slavelist[slave.position];
    if ((soem_slave.mbx_proto & ECT_MBXPROT_COE) == 0) {
        VLOG(2) << "Slave " << slave.position << " does not support CoE";
        return false;
    }

    auto err = this->read_pdo_assign(slave.position, ECT_SDO_TXPDOASSIGN, true, slave);
    if (err) {
        VLOG(2) << "Failed to read TxPDO assignment for slave " << slave.position
                << ": " << err.message();
        slave.pdo_discovery_error = err.message();
        return false;
    }

    err = this->read_pdo_assign(slave.position, ECT_SDO_RXPDOASSIGN, false, slave);
    if (err) {
        VLOG(2) << "Failed to read RxPDO assignment for slave " << slave.position
                << ": " << err.message();
        if (slave.pdo_discovery_error.empty())
            slave.pdo_discovery_error = err.message();
    }

    slave.pdos_discovered = true;
    VLOG(1) << "Slave " << slave.position
            << " PDOs discovered via CoE: " << slave.input_pdos.size() << " inputs, "
            << slave.output_pdos.size() << " outputs";
    return true;
}

void Master::discover_pdos_sii(SlaveInfo &slave) {
    VLOG(2) << "Using SII fallback for slave " << slave.position;
    slave.pdos_discovered = true;
    slave.pdo_discovery_error = "CoE not supported, limited PDO info from SII";
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
    if (result <= 0)
        return xerrors::Error(SDO_READ_ERROR, "failed to read PDO assignment count");

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
            if (consecutive_failures >= MAX_SDO_FAILURES) {
                return xerrors::Error(
                    SDO_READ_ERROR,
                    "too many consecutive SDO failures"
                );
            }
            continue;
        }
        consecutive_failures = 0;

        if (pdo_index == 0) continue;

        auto err = this->read_pdo_mapping(slave_pos, pdo_index, is_input, slave);
        if (err) {
            VLOG(2) << "Failed to read PDO mapping 0x" << std::hex << pdo_index
                    << " for slave " << std::dec << slave_pos << ": " << err.message();
        }
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
    if (result <= 0)
        return xerrors::Error(SDO_READ_ERROR, "failed to read PDO mapping count");

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
    if (ecx_readOEsingle(&this->context, 0, subindex, &od_list, &oe_list) > 0) {
        if (oe_list.Entries > 0 && oe_list.Name[0][0] != '\0')
            return std::string(oe_list.Name[0]);
    }

    return "";
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

    if (success_count < group0_count) {
        return xerrors::Error(
            STATE_CHANGE_ERROR,
            "state transition failed: " + std::to_string(success_count) + "/" +
                std::to_string(group0_count) + " slaves reached state " +
                std::to_string(state) + "; " + error_msg
        );
    }

    return xerrors::NIL;
}

} // namespace ethercat::soem
