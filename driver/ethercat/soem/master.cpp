// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/ethercat/soem/master.h"

#include <algorithm>

namespace ethercat::soem {

////////////////////////////////////////////////////////////////////////////////
// SOEMDomain Implementation
////////////////////////////////////////////////////////////////////////////////

SOEMDomain::SOEMDomain(const size_t iomap_size)
    : iomap_(iomap_size, 0),
      input_offset_(0),
      output_offset_(0),
      input_size_(0),
      output_size_(0) {
}

std::pair<size_t, xerrors::Error> SOEMDomain::register_pdo(const PDOEntry &entry) {
    // Calculate offset based on direction.
    // In SOEM, the actual offsets are determined during ecx_config_map_group(),
    // but we track expected offsets for user-level access.
    size_t offset;
    const size_t byte_size = (entry.bit_length + 7) / 8;

    if (entry.is_input) {
        // Input PDOs (TxPDO, slave → master) start after output area
        offset = output_size_ + input_offset_;
        input_offset_ += byte_size;
    } else {
        // Output PDOs (RxPDO, master → slave) at beginning
        offset = output_offset_;
        output_offset_ += byte_size;
    }

    // Check bounds
    if (offset + byte_size > iomap_.size()) {
        return {0, xerrors::Error(PDO_MAPPING_ERROR, "IOmap buffer overflow")};
    }

    registered_pdos_.emplace_back(entry, offset);
    return {offset, xerrors::NIL};
}

uint8_t *SOEMDomain::data() {
    return iomap_.data();
}

size_t SOEMDomain::size() const {
    return iomap_.size();
}

size_t SOEMDomain::input_size() const {
    return input_size_;
}

size_t SOEMDomain::output_size() const {
    return output_size_;
}

void SOEMDomain::set_sizes(const size_t input_size, const size_t output_size) {
    input_size_ = input_size;
    output_size_ = output_size;
}

////////////////////////////////////////////////////////////////////////////////
// SOEMMaster Implementation
////////////////////////////////////////////////////////////////////////////////

/// Default timeout for state transitions (2 seconds in microseconds).
constexpr int STATE_CHANGE_TIMEOUT = 2000000;

/// Default timeout for process data receive (1 millisecond in microseconds).
constexpr int PROCESSDATA_TIMEOUT = 1000;

SOEMMaster::SOEMMaster(std::string interface_name)
    : interface_name_(std::move(interface_name)),
      context_{},
      initialized_(false),
      activated_(false),
      domain_(nullptr),
      expected_wkc_(0) {
    // Zero-initialize the SOEM context
    std::memset(&context_, 0, sizeof(context_));
}

SOEMMaster::~SOEMMaster() {
    if (activated_) {
        deactivate();
    }
    if (initialized_) {
        ecx_close(&context_);
    }
}

xerrors::Error SOEMMaster::initialize() {
    if (initialized_) {
        // Already initialized - idempotent success
        return xerrors::NIL;
    }

    // Initialize the SOEM port on the specified interface
    if (ecx_init(&context_, interface_name_.c_str()) <= 0) {
        return xerrors::Error(
            MASTER_INIT_ERROR,
            "failed to initialize EtherCAT on interface: " + interface_name_
        );
    }

    // Scan for slaves on the network
    if (ecx_config_init(&context_) <= 0) {
        ecx_close(&context_);
        return xerrors::Error(MASTER_INIT_ERROR, "no EtherCAT slaves found on network");
    }

    // Populate the cached slave list
    populate_slaves();

    initialized_ = true;
    return xerrors::NIL;
}

std::unique_ptr<Domain> SOEMMaster::create_domain() {
    return std::make_unique<SOEMDomain>();
}

xerrors::Error SOEMMaster::activate() {
    if (!initialized_) {
        return xerrors::Error(ACTIVATION_ERROR, "master not initialized");
    }
    if (activated_) {
        return xerrors::Error(ACTIVATION_ERROR, "master already activated");
    }

    // Separate problematic slaves into group 1 before PDO mapping.
    // Some slave types (like DEWESoft 6xSTG strain gauge modules, product 0xFC)
    // have SM configuration issues that prevent them from reaching SAFE_OP.
    // By putting them in a separate group, we can still operate the working slaves.
    int excluded_count = 0;
    for (int i = 1; i <= context_.slavecount; ++i) {
        // DEWESoft 6xSTG has known SM OUT configuration issues (AL code 38)
        if (context_.slavelist[i].eep_id == 0x000000FC) {
            context_.slavelist[i].group = 1;  // Exclude from group 0
            excluded_count++;
        } else {
            context_.slavelist[i].group = 0;  // Include in group 0
        }
    }

    // Map the process data using SOEM's auto-configuration
    // This configures all slaves' PDO mappings based on their EEPROM/CoE settings
    // We use group 0 which includes only the working slaves
    auto domain = std::make_unique<SOEMDomain>();
    const int iomap_size = ecx_config_map_group(&context_, domain->iomap_ptr(), 0);

    if (iomap_size <= 0 && excluded_count < context_.slavecount) {
        return xerrors::Error(ACTIVATION_ERROR, "failed to configure PDO mapping");
    }

    // Calculate total input/output sizes from group 0
    const auto &group = context_.grouplist[0];
    domain->set_sizes(group.Ibytes, group.Obytes);

    // Store domain pointer for cyclic operations
    domain_ = domain.release();

    // Calculate expected working counter
    // WKC = (outputs + inputs) for LRW command
    expected_wkc_ = (context_.grouplist[0].outputsWKC * 2) +
                    context_.grouplist[0].inputsWKC;

    // Transition all slaves to Safe-Op
    auto err = request_state(EC_STATE_SAFE_OP, STATE_CHANGE_TIMEOUT);
    if (err) {
        delete domain_;
        domain_ = nullptr;
        return err;
    }

    // Transition all slaves to Operational
    // First, start sending process data (required before OP transition)
    ecx_send_processdata(&context_);
    ecx_receive_processdata(&context_, PROCESSDATA_TIMEOUT);

    err = request_state(EC_STATE_OPERATIONAL, STATE_CHANGE_TIMEOUT);
    if (err) {
        // Try to return to Safe-Op on failure
        request_state(EC_STATE_SAFE_OP, STATE_CHANGE_TIMEOUT);
        delete domain_;
        domain_ = nullptr;
        return err;
    }

    activated_ = true;
    return xerrors::NIL;
}

void SOEMMaster::deactivate() {
    if (!activated_) return;

    // Transition slaves back to Init state
    request_state(EC_STATE_INIT, STATE_CHANGE_TIMEOUT);

    // Clean up domain
    delete domain_;
    domain_ = nullptr;
    activated_ = false;
    expected_wkc_ = 0;
}

xerrors::Error SOEMMaster::receive() {
    if (!activated_) {
        return xerrors::Error(CYCLIC_ERROR, "master not activated");
    }

    // Receive process data from slaves
    // This updates the input portion of the IOmap
    const int wkc = ecx_receive_processdata(&context_, PROCESSDATA_TIMEOUT);

    if (wkc < 0) {
        return xerrors::Error(CYCLIC_ERROR, "process data receive failed");
    }

    // Check working counter
    if (wkc != expected_wkc_) {
        // Working counter mismatch indicates communication issues
        // This could be a slave dropping out or communication error
        return xerrors::Error(
            WORKING_COUNTER_ERROR,
            "working counter mismatch: expected " +
            std::to_string(expected_wkc_) + ", got " + std::to_string(wkc)
        );
    }

    return xerrors::NIL;
}

xerrors::Error SOEMMaster::process(Domain &domain) {
    // In SOEM, process data is directly in the IOmap buffer
    // This method is a no-op as data is already available after receive()
    (void)domain;
    return xerrors::NIL;
}

xerrors::Error SOEMMaster::queue(Domain &domain) {
    // In SOEM, output data is written directly to the IOmap buffer
    // This method is a no-op as the buffer is already prepared for send()
    (void)domain;
    return xerrors::NIL;
}

xerrors::Error SOEMMaster::send() {
    if (!activated_) {
        return xerrors::Error(CYCLIC_ERROR, "master not activated");
    }

    // Send process data to slaves
    // This transmits the output portion of the IOmap
    const int result = ecx_send_processdata(&context_);

    if (result <= 0) {
        return xerrors::Error(CYCLIC_ERROR, "process data send failed");
    }

    return xerrors::NIL;
}

std::vector<SlaveInfo> SOEMMaster::slaves() const {
    std::lock_guard lock(mutex_);
    return slaves_;
}

SlaveState SOEMMaster::slave_state(const uint16_t position) const {
    std::lock_guard lock(mutex_);

    if (position == 0 || position > static_cast<uint16_t>(context_.slavecount)) {
        return SlaveState::UNKNOWN;
    }

    // Read current state from SOEM's slave list
    // Note: SOEM uses 1-based indexing for slaves
    return convert_state(context_.slavelist[position].state);
}

bool SOEMMaster::all_slaves_operational() const {
    std::lock_guard lock(mutex_);

    if (!activated_) return false;

    for (int i = 1; i <= context_.slavecount; ++i) {
        if ((context_.slavelist[i].state & EC_STATE_OPERATIONAL) == 0) {
            return false;
        }
    }
    return true;
}

std::string SOEMMaster::interface_name() const {
    return interface_name_;
}

Domain *SOEMMaster::active_domain() const {
    return domain_;
}

SlaveDataOffsets SOEMMaster::slave_data_offsets(const uint16_t position) const {
    std::lock_guard lock(mutex_);

    if (!activated_ || position == 0 ||
        position > static_cast<uint16_t>(context_.slavecount)) {
        return SlaveDataOffsets{};
    }

    // Only return offsets for slaves in group 0 (the active group)
    if (context_.slavelist[position].group != 0) {
        return SlaveDataOffsets{};
    }

    const auto &slave = context_.slavelist[position];

    // SOEM stores pointers to the slave's data within the IOmap.
    // We calculate the offset by subtracting the IOmap base address.
    const auto *iomap_base = domain_->data();
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

SlaveState SOEMMaster::convert_state(const uint16_t soem_state) {
    // SOEM state values match ETG.1000 spec (which our enum follows)
    // but we need to handle the error/ack flag
    const uint16_t state = soem_state & 0x0F;

    switch (state) {
        case EC_STATE_INIT: return SlaveState::INIT;
        case EC_STATE_PRE_OP: return SlaveState::PRE_OP;
        case EC_STATE_BOOT: return SlaveState::BOOT;
        case EC_STATE_SAFE_OP: return SlaveState::SAFE_OP;
        case EC_STATE_OPERATIONAL: return SlaveState::OP;
        default: return SlaveState::UNKNOWN;
    }
}

void SOEMMaster::populate_slaves() {
    slaves_.clear();
    slaves_.reserve(context_.slavecount);

    // SOEM uses 1-based indexing for slaves (index 0 is the master)
    for (int i = 1; i <= context_.slavecount; ++i) {
        const auto &slave = context_.slavelist[i];
        SlaveInfo info{};
        info.position = static_cast<uint16_t>(i);
        info.vendor_id = slave.eep_man;
        info.product_code = slave.eep_id;
        info.revision = slave.eep_rev;
        info.serial = slave.eep_ser;
        info.name = slave.name;
        slaves_.push_back(info);
    }
}

xerrors::Error SOEMMaster::request_state(const uint16_t state, const int timeout) {
    // Transition only group 0 slaves (problematic slaves are in group 1)
    // We do this per-slave to avoid affecting excluded slaves
    int success_count = 0;
    int group0_count = 0;
    std::string error_msg;

    for (int i = 1; i <= context_.slavecount; ++i) {
        // Only transition slaves in group 0
        if (context_.slavelist[i].group != 0) continue;
        group0_count++;

        context_.slavelist[i].state = state;
        ecx_writestate(&context_, i);

        const uint16_t result = ecx_statecheck(&context_, i, state, timeout);

        if ((result & 0x0F) == (state & 0x0F)) {
            success_count++;
        } else {
            if (!error_msg.empty()) error_msg += "; ";
            error_msg += "slave " + std::to_string(i) + " in state " +
                         std::to_string(result);
            if (result & EC_STATE_ERROR) {
                error_msg += " (ERROR flag set, AL status code: " +
                             std::to_string(context_.slavelist[i].ALstatuscode) + ")";
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
