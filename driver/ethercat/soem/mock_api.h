// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <array>
#include <atomic>
#include <chrono>
#include <cstring>
#include <deque>
#include <functional>
#include <map>
#include <optional>
#include <set>
#include <span>
#include <string>
#include <utility>
#include <vector>

#include "x/cpp/binary/binary.h"
#include "x/cpp/loop/loop.h"
#include "x/cpp/telem/telem.h"

#include "driver/ethercat/soem/api.h"

namespace driver::ethercat::soem {

/// @brief one slave on the mock bus. config_init copies its identity into the SOEM
/// slave table.
struct MockSlaveInfo {
    /// @brief the state of the slave on the bus. readstate and statecheck copy it into
    /// the SOEM slave table.
    uint16_t state = EC_STATE_INIT;
    /// @brief the identity that config_init reads from the EEPROM, which is not in the
    /// catalog. A vendor and product of 0 is a failed read.
    uint32_t eep_man = 0xFFFF;
    uint32_t eep_id = 0xFFFF;
    uint32_t eep_rev = 0;
    uint32_t eep_ser = 0;
    std::string name;
    /// @brief the input and output bits that config_map_group maps for the slave.
    uint16_t Ibits = 0;
    uint16_t Obits = 0;
    /// @brief where config_map_group places the outputs and inputs of the slave in the
    /// IOmap.
    size_t output_offset = 0;
    size_t input_offset = 0;
    /// @brief the bit of its first byte where config_map_group starts the outputs and
    /// inputs of a slave that maps fewer than 8 bits.
    uint8_t output_start_bit = 0;
    uint8_t input_start_bit = 0;
    uint16_t mbx_proto = 0;
    uint16_t ALstatuscode = 0;
    /// @brief whether the slave has a distributed clock.
    bool hasdc = false;
    /// @brief the propagation delay from the reference clock in nanoseconds.
    int32_t pdelay = 0;
    /// @brief the state readstate() reports, or nullopt to report the current state.
    std::optional<uint16_t> bus_state;
    /// @brief the state last requested and not yet reached.
    std::optional<uint16_t> requested;
    /// @brief CoE objects keyed by index and sub-index. SDOread returns them, and a
    /// successful SDOwrite replaces them.
    std::map<std::pair<uint16_t, uint8_t>, uint32_t> sdo_objects;
    /// @brief abort codes for objects that reject SDO reads, keyed by index and
    /// sub-index. A code of 0 makes the read get no reply.
    std::map<std::pair<uint16_t, uint8_t>, uint32_t> sdo_read_aborts;
    /// @brief the SOEM packet error codes of objects whose SDO read replies SOEM
    /// rejects, keyed by index and sub-index.
    std::map<std::pair<uint16_t, uint8_t>, uint16_t> sdo_read_packet_errors;
    /// @brief objects whose SDO reads the slave answers with a mailbox error, keyed by
    /// index and sub-index.
    std::set<std::pair<uint16_t, uint8_t>> sdo_read_mailbox_errors;
    /// @brief objects whose SDO reads bring a CoE emergency from the slave, keyed by
    /// index and sub-index.
    std::set<std::pair<uint16_t, uint8_t>> sdo_read_emergencies;
    /// @brief whether the slave lists complete access in its CoE details.
    bool complete_access = false;
    /// @brief the bytes that a complete access read of each object returns, keyed by
    /// index. A complete access read of any other object fails.
    std::map<uint16_t, std::vector<uint8_t>> complete_objects;
    /// @brief abort codes for objects that reject SDO writes, keyed by index and
    /// sub-index. A code of 0 makes the write time out.
    std::map<std::pair<uint16_t, uint8_t>, uint32_t> sdo_write_aborts;
    /// @brief the SOEM packet error codes of objects whose SDO write replies SOEM
    /// rejects, keyed by index and sub-index.
    std::map<std::pair<uint16_t, uint8_t>, uint16_t> sdo_write_packet_errors;
    /// @brief entry names and CoE data types returned by readOEsingle, keyed by index
    /// and sub-index.
    std::map<std::pair<uint16_t, uint8_t>, std::pair<std::string, uint16_t>> oe_entries;
    /// @brief object names returned by readODdescription, keyed by index.
    std::map<uint16_t, std::string> od_names;
    /// @brief the SII address of each category, keyed by category.
    std::map<uint16_t, int16_t> sii_categories;
    /// @brief SII bytes returned by siigetbyte, keyed by address.
    std::map<uint16_t, uint8_t> sii_bytes;
    /// @brief SII strings returned by siistring, keyed by string index.
    std::map<uint16_t, std::string> sii_strings;
};

/// @brief a write through write_register.
struct RegisterWrite {
    uint16_t slave = 0;
    uint16_t address = 0;
    /// @brief the value of the written bytes, read as a little-endian integer.
    int64_t value = 0;
    /// @brief the number of bytes written.
    size_t size = 0;
    /// @brief true when the write came after config_map_group.
    bool mapped = false;

    bool operator==(const RegisterWrite &) const = default;
};

/// @brief mock implementation of soem::API for unit testing.
class MockAPI final : public API {
public:
    std::vector<MockSlaveInfo> slave_infos;
    x::errors::Error check_socket_err = x::errors::NIL;
    /// @brief the interfaces whose adapters the DewesoftX filter is bound to.
    std::set<std::string> filtered_interfaces;
    /// @brief the Windows adapter names that adapter_name returns, keyed by interface.
    std::map<std::string, std::string> adapter_names;
    /// @brief the interfaces whose adapters are on USB.
    std::set<std::string> usb_interfaces;
    int init_return = 1;
    int config_init_return = 1;
    int config_map_return = 64;
    int send_return = 1;
    int receive_return = 3;
    int writestate_return = 1;
    std::map<uint16_t, uint16_t> statecheck_overrides;
    /// @brief per requested state, the number of statecheck polls that report the
    /// slave's current state before it reaches the requested one.
    std::map<uint16_t, int> statecheck_pending_polls;
    /// @brief the group sizes and working counters that config_map_group sets.
    uint32_t ibytes = 0;
    uint32_t obytes = 0;
    uint16_t outputs_wkc = 1;
    uint16_t inputs_wkc = 1;
    /// @brief the slaves whose EEPROM the master controls: siigetbyte adds the slave,
    /// eeprom2pdi removes it.
    std::set<uint16_t> eeprom_held;
    bool close_called = false;
    bool init_called = false;
    int send_count = 0;
    int receive_count = 0;
    /// @brief the lowest slave state that readstate last read.
    std::atomic<uint16_t> read_state{EC_STATE_INIT};
    /// @brief read_state at each send_processdata call.
    std::vector<uint16_t> send_states;
    int sync_dc_count = 0;
    /// @brief the working counter sync_dc returns.
    int sync_dc_return = 1;
    std::vector<std::pair<uint16_t, uint16_t>> state_change_requests;
    std::vector<uint8_t> processdata_groups;
    std::optional<uint8_t> mapped_group;
    /// @brief whether the context requested manual state changes at the latest
    /// config_map_group call.
    std::optional<bool> manual_at_map;
    /// @brief whether every process data sync manager of the mapped group had a length
    /// of zero at the latest config_map_group call.
    bool sms_cleared_at_map = false;
    int readstate_count = 0;
    /// @brief the number of next readstate calls that get no reply. As in SOEM, each
    /// one sets every slave to EC_STATE_NONE.
    int dropped_state_reads = 0;
    /// @brief the state reconfig_slave returns and leaves the slave in.
    int reconfig_return = EC_STATE_SAFE_OP;
    std::vector<uint16_t> reconfigured_slaves;
    /// @brief the value recover_slave returns.
    int recover_return = 1;
    std::vector<uint16_t> recovered_slaves;
    /// @brief slaves passed to writestate, in call order.
    std::vector<uint16_t> written_slaves;
    /// @brief one SDO write: slave, index, sub-index, and bytes.
    struct SDOWrite {
        uint16_t slave;
        uint16_t index;
        uint8_t sub_index;
        std::vector<uint8_t> data;

        bool operator==(const SDOWrite &) const = default;
    };
    /// @brief SDO writes in call order.
    std::vector<SDOWrite> sdo_writes;
    /// @brief the SOEM error list, oldest first.
    std::deque<ec_errort> errors;
    /// @brief PO2SOconfig hooks keyed by slave. config_map_group and reconfig_slave
    /// run them.
    std::map<uint16_t, std::function<void(uint16_t)>> po2so_configs;
    /// @brief the value count_bus_slaves returns, or nullopt for the slave count.
    std::optional<int> bus_slaves;
    /// @brief writes through write_register, in call order.
    std::vector<RegisterWrite> register_writes;
    /// @brief the working counter write_register returns.
    int write_register_return = 1;
    /// @brief the timeouts of write_register and read_register, in call order.
    std::vector<int> register_timeouts;
    /// @brief slaves whose register writes return a working counter of 0.
    std::set<uint16_t> unwritable_slaves;
    /// @brief the number of state change requests before each register write, in the
    /// order of register_writes.
    std::vector<size_t> requests_before_write;
    /// @brief the values read_register gives, keyed by slave and address. A register
    /// missing from the map reads as zero. A write_register that reaches the slave
    /// stores its value here.
    std::map<std::pair<uint16_t, uint16_t>, int64_t> register_values;
    /// @brief the working counter read_register returns.
    int read_register_return = 1;
    /// @brief registers, keyed by slave and address, whose reads get no reply.
    std::set<std::pair<uint16_t, uint16_t>> unanswered_reads;
    int configdc_count = 0;
    /// @brief the host clock that the reference clock of host_dc_offset and
    /// latch_receive_times run on.
    x::loop::HostClock clock;
    /// @brief when set, each send_processdata sets the DC time to the clock time plus
    /// this, in nanoseconds, like a reference clock that runs on.
    std::optional<int64_t> host_dc_offset;
    /// @brief the number of latch_receive_times calls.
    int latch_count = 0;
    /// @brief the clock time of the last latch_receive_times call, in nanoseconds.
    int64_t latched_at = 0;
    /// @brief the timeout of the last latch_receive_times call.
    int latch_timeout = 0;
    /// @brief the value latch_receive_times returns.
    int latch_return = 1;

private:
    /// @brief returns the mock slave at pos, or nullptr when no slave is there.
    MockSlaveInfo *info(const uint16_t pos) {
        if (pos == 0 || pos > this->slave_infos.size()) return nullptr;
        return &this->slave_infos[pos - 1];
    }

    /// @brief returns the state the slave at pos is in on the bus.
    uint16_t bus_state_of(const uint16_t pos) {
        const auto *slave = this->info(pos);
        return slave == nullptr ? 0 : slave->state;
    }

    /// @brief runs the slave's PO2SOconfig hook, like SOEM does in PRE-OP.
    void run_PO2SOconfig(const uint16_t slave) {
        if (const auto it = this->po2so_configs.find(slave);
            it != this->po2so_configs.end())
            it->second(slave);
    }

    /// @brief returns the clock time in nanoseconds.
    [[nodiscard]] int64_t now() const {
        return x::telem::TimeSpan(this->clock.steady().time_since_epoch())
            .nanoseconds();
    }

public:
    x::errors::Error check_socket(const char *) override {
        return this->check_socket_err;
    }

    bool dewesoft_filter_bound(const char *ifname) override {
        return this->filtered_interfaces.contains(ifname);
    }

    std::optional<std::string> adapter_name(const char *ifname) override {
        const auto it = this->adapter_names.find(ifname);
        if (it == this->adapter_names.end()) return std::nullopt;
        return it->second;
    }

    bool on_usb(const char *ifname) override {
        return this->usb_interfaces.contains(ifname);
    }

    int init(const char *) override {
        this->init_called = true;
        return this->init_return;
    }

    void close() override { this->close_called = true; }

    /// @brief like ecx_config_init, fills the slave table from slave_infos. Each slave
    /// gets the sync manager lengths of its SII.
    int config_init() override {
        if (this->config_init_return <= 0) return this->config_init_return;
        this->context.slavecount = static_cast<int>(this->slave_infos.size());
        for (size_t i = 0; i < this->slave_infos.size(); ++i) {
            const auto &info = this->slave_infos[i];
            auto &slave = this->context.slavelist[i + 1];
            slave = {};
            slave.state = info.state;
            slave.ALstatuscode = info.ALstatuscode;
            slave.eep_man = info.eep_man;
            slave.eep_id = info.eep_id;
            slave.eep_rev = info.eep_rev;
            slave.eep_ser = info.eep_ser;
            std::strncpy(slave.name, info.name.c_str(), EC_MAXNAME);
            slave.mbx_proto = info.mbx_proto;
            slave.CoEdetails = info.complete_access ? ECT_COEDET_SDOCA : 0;
            slave.hasdc = info.hasdc ? 1 : 0;
            slave.pdelay = info.pdelay;
            for (int sm = 2; sm < EC_MAXSM; ++sm)
                slave.SM[sm].SMlength = 1;
        }
        return this->config_init_return;
    }

    /// @brief like ecx_config_map_group, maps the slaves of group into iomap, sets the
    /// group sizes and working counters, and runs the PO2SOconfig hook of each slave.
    int config_map_group(void *iomap, uint8_t group) override {
        auto *base = static_cast<uint8_t *>(iomap);
        this->mapped_group = group;
        this->manual_at_map = this->context.manualstatechange != 0;
        this->sms_cleared_at_map = true;
        for (size_t i = 0; i < this->slave_infos.size(); ++i) {
            auto &slave = this->context.slavelist[i + 1];
            if (slave.group != group) continue;
            for (int sm = 2; sm < EC_MAXSM; ++sm)
                if (slave.SM[sm].SMlength != 0) this->sms_cleared_at_map = false;
            const auto pos = static_cast<uint16_t>(i + 1);
            this->run_PO2SOconfig(pos);
            const auto &info = this->slave_infos[i];
            slave.Ibits = info.Ibits;
            slave.Obits = info.Obits;
            slave.outputs = base + info.output_offset;
            slave.inputs = base + info.input_offset;
            slave.Ostartbit = info.output_start_bit;
            slave.Istartbit = info.input_start_bit;
        }
        auto &mapped = this->context.grouplist[group];
        mapped.Ibytes = this->ibytes;
        mapped.Obytes = this->obytes;
        mapped.outputsWKC = this->outputs_wkc;
        mapped.inputsWKC = this->inputs_wkc;
        return this->config_map_return;
    }

    int send_processdata(uint8_t group) override {
        if (this->host_dc_offset) this->context.DCtime = now() + *this->host_dc_offset;
        this->send_count++;
        this->send_states.push_back(this->read_state.load());
        this->processdata_groups.push_back(group);
        return this->send_return;
    }

    int receive_processdata(uint8_t group, int) override {
        this->receive_count++;
        this->processdata_groups.push_back(group);
        return this->receive_return;
    }

    /// @brief like ecx_writestate, requests the state in the slave table entry of the
    /// slave.
    int writestate(uint16_t slave) override {
        this->written_slaves.push_back(slave);
        if (auto *info = this->info(slave); info != nullptr) {
            const uint16_t state = this->context.slavelist[slave].state;
            this->state_change_requests.emplace_back(slave, state);
            info->requested = state;
        }
        return this->writestate_return;
    }

    int readstate() override {
        this->readstate_count++;
        if (this->dropped_state_reads > 0) {
            this->dropped_state_reads--;
            for (size_t i = 0; i < this->slave_infos.size(); ++i) {
                this->slave_infos[i].state = EC_STATE_NONE;
                this->context.slavelist[i + 1].state = EC_STATE_NONE;
            }
            this->read_state.store(EC_STATE_NONE);
            return EC_STATE_NONE;
        }
        uint16_t lowest = 0xFF;
        for (size_t i = 0; i < this->slave_infos.size(); ++i) {
            auto &info = this->slave_infos[i];
            if (info.bus_state)
                info.state = *info.bus_state;
            else if (info.requested) {
                const uint16_t target = *info.requested & 0x0F;
                info.state = this->reach(static_cast<uint16_t>(i + 1), target);
                if (info.state == target) info.requested.reset();
            }
            lowest = std::min<uint16_t>(lowest, info.state & 0x0F);
            this->context.slavelist[i + 1].state = info.state;
            this->context.slavelist[i + 1].ALstatuscode = info.ALstatuscode;
        }
        this->context.slavelist[0].state = lowest;
        this->read_state.store(lowest);
        return lowest;
    }

    int count_bus_slaves() override {
        return this->bus_slaves.value_or(static_cast<int>(this->slave_infos.size()));
    }

    /// @brief like ecx_configdc, makes the first slave with a clock the reference, and
    /// makes the frames of its group carry the DC time.
    bool configdc() override {
        this->configdc_count++;
        auto &bus = this->context.slavelist[0];
        bus.hasdc = 0;
        this->context.grouplist[0].hasdc = 0;
        for (int i = 1; i <= this->context.slavecount; ++i) {
            const auto &slave = this->context.slavelist[i];
            if (slave.hasdc == 0) continue;
            bus.hasdc = 1;
            bus.DCnext = static_cast<uint16_t>(i);
            this->context.grouplist[slave.group].hasdc = 1;
            this->context.grouplist[slave.group].DCnext = static_cast<uint16_t>(i);
            return true;
        }
        return false;
    }

    int sync_dc(uint8_t, int) override {
        this->sync_dc_count++;
        return this->sync_dc_return;
    }

    int latch_receive_times(const int timeout) override {
        this->latch_timeout = timeout;
        this->latched_at = now();
        this->latch_count++;
        return this->latch_return;
    }

    void
    set_PO2SOconfig(uint16_t slave, std::function<void(uint16_t)> config) override {
        if (config == nullptr)
            this->po2so_configs.erase(slave);
        else
            this->po2so_configs[slave] = std::move(config);
    }

    int write_register(
        uint16_t slave,
        uint16_t address,
        std::vector<uint8_t> data,
        const int timeout
    ) override {
        this->register_timeouts.push_back(timeout);
        this->requests_before_write.push_back(this->state_change_requests.size());
        const int wkc = this->unwritable_slaves.contains(slave)
                          ? 0
                          : this->write_register_return;
        std::array<uint8_t, sizeof(int64_t)> padded{};
        std::ranges::copy(data, padded.begin());
        const auto value = x::binary::from_little_endian<int64_t>(padded);
        if (wkc > 0) this->register_values[{slave, address}] = value;
        this->register_writes.push_back({
            .slave = slave,
            .address = address,
            .value = value,
            .size = data.size(),
            .mapped = this->mapped_group.has_value(),
        });
        return wkc;
    }

    int read_register(
        uint16_t slave,
        uint16_t address,
        std::span<uint8_t> data,
        const int timeout
    ) override {
        this->register_timeouts.push_back(timeout);
        const auto it = this->register_values.find({slave, address});
        const auto bytes = x::binary::to_little_endian(
            it == this->register_values.end() ? int64_t{0} : it->second
        );
        std::copy_n(bytes.begin(), data.size(), data.begin());
        if (this->unanswered_reads.contains({slave, address})) return EC_NOFRAME;
        return this->read_register_return;
    }

    /// @brief like ecx_reconfig_slave, runs the slave's PO2SOconfig hook in PRE-OP and
    /// leaves the slave in the state it returns.
    int reconfig_slave(uint16_t slave, int) override {
        this->reconfigured_slaves.push_back(slave);
        this->run_PO2SOconfig(slave);
        if (auto *info = this->info(slave); info != nullptr) {
            info->state = static_cast<uint16_t>(this->reconfig_return);
            this->context.slavelist[slave].state = info->state;
        }
        return this->reconfig_return;
    }

    int recover_slave(uint16_t slave, int) override {
        this->recovered_slaves.push_back(slave);
        return this->recover_return;
    }

    /// @brief like ecx_statecheck, a group check (slave 0) that reaches reqstate
    /// leaves each slave's state for the next readstate to update.
    uint16_t statecheck(uint16_t slave, uint16_t reqstate, int) override {
        const uint16_t state = this->reach(slave, reqstate);
        if (slave == 0 && state == reqstate)
            for (auto &info: this->slave_infos) {
                info.bus_state.reset();
                info.requested = reqstate;
            }
        if (auto *info = this->info(slave); info != nullptr) {
            info->state = state;
            if (state == reqstate) info->requested.reset();
            this->context.slavelist[slave].ALstatuscode = info->ALstatuscode;
        }
        this->context.slavelist[slave].state = state;
        return state;
    }

    /// @brief returns the state a slave reports on one poll while it moves to reqstate.
    uint16_t reach(const uint16_t slave, const uint16_t reqstate) {
        const auto it = this->statecheck_overrides.find(reqstate);
        if (auto &pending = this->statecheck_pending_polls[reqstate]; pending > 0) {
            pending--;
            return this->bus_state_of(slave);
        }
        return it != this->statecheck_overrides.end() ? it->second : reqstate;
    }

    int SDOread(
        uint16_t slave,
        uint16_t index,
        uint8_t sub_index,
        int complete_access,
        int *psize,
        void *p,
        int
    ) override {
        // SOEM sends no mailbox request to a slave whose last read state is below
        // PRE-OP.
        if ((this->context.slavelist[slave].state & 0x0F) < EC_STATE_PRE_OP) return 0;
        const auto &info = this->slave_infos.at(slave - 1);
        if (complete_access != 0) {
            const auto it = info.complete_objects.find(index);
            if (it == info.complete_objects.end()) {
                this->push_sdo_abort(slave, index, sub_index, 0x06010000);
                return 0;
            }
            *psize = std::min(*psize, static_cast<int>(it->second.size()));
            std::memcpy(p, it->second.data(), *psize);
            return 1;
        }
        for (const auto &[objects, type]:
             {std::pair{&info.sdo_read_emergencies, EC_ERR_TYPE_EMERGENCY},
              std::pair{&info.sdo_read_mailbox_errors, EC_ERR_TYPE_MBX_ERROR}})
            if (objects->contains({index, sub_index})) {
                ec_errort err{};
                err.Etype = type;
                err.Slave = slave;
                this->errors.push_back(err);
            }
        if (info.sdo_read_mailbox_errors.contains({index, sub_index})) return 0;
        if (const auto abort = info.sdo_read_aborts.find({index, sub_index});
            abort != info.sdo_read_aborts.end()) {
            if (abort->second != 0)
                this->push_sdo_abort(slave, index, sub_index, abort->second);
            return 0;
        }
        if (const auto packet = info.sdo_read_packet_errors.find({index, sub_index});
            packet != info.sdo_read_packet_errors.end()) {
            this->push_packet_error(slave, index, sub_index, packet->second);
            return 0;
        }
        const auto &objects = info.sdo_objects;
        const auto it = objects.find({index, sub_index});
        if (it == objects.end()) {
            this->push_sdo_abort(slave, index, sub_index, 0x06020000);
            return 0;
        }
        std::memcpy(p, &it->second, *psize);
        return 1;
    }

    int SDOwrite(
        uint16_t slave,
        uint16_t index,
        uint8_t sub_index,
        bool,
        int psize,
        const void *p,
        int
    ) override {
        const auto *bytes = static_cast<const uint8_t *>(p);
        this->sdo_writes.push_back(
            {slave, index, sub_index, std::vector(bytes, bytes + psize)}
        );
        auto &info = this->slave_infos.at(slave - 1);
        if (const auto packet = info.sdo_write_packet_errors.find({index, sub_index});
            packet != info.sdo_write_packet_errors.end()) {
            this->push_packet_error(slave, index, sub_index, packet->second);
            return 0;
        }
        const auto it = info.sdo_write_aborts.find({index, sub_index});
        if (it == info.sdo_write_aborts.end()) {
            uint32_t value = 0;
            std::memcpy(&value, p, std::min(psize, 4));
            info.sdo_objects[{index, sub_index}] = value;
            return 1;
        }
        if (it->second != 0) this->push_sdo_abort(slave, index, sub_index, it->second);
        return 0;
    }

    /// @brief appends an SDO abort for the object to the SOEM error list.
    void
    push_sdo_abort(uint16_t slave, uint16_t index, uint8_t sub_index, uint32_t code) {
        ec_errort err{};
        err.Etype = EC_ERR_TYPE_SDO_ERROR;
        err.Slave = slave;
        err.Index = index;
        err.SubIdx = sub_index;
        err.AbortCode = static_cast<int32>(code);
        this->errors.push_back(err);
    }

    /// @brief appends a SOEM packet error for the object to the SOEM error list.
    void push_packet_error(
        uint16_t slave,
        uint16_t index,
        uint8_t sub_index,
        uint16_t code
    ) {
        ec_errort err{};
        err.Etype = EC_ERR_TYPE_PACKET_ERROR;
        err.Slave = slave;
        err.Index = index;
        err.SubIdx = sub_index;
        err.ErrorCode = code;
        this->errors.push_back(err);
    }

    bool poperror(ec_errort &err) override {
        if (this->errors.empty()) return false;
        err = this->errors.front();
        this->errors.pop_front();
        return true;
    }

    int16_t siifind(uint16_t slave, uint16_t category) override {
        const auto &categories = this->slave_infos.at(slave - 1).sii_categories;
        const auto it = categories.find(category);
        return it == categories.end() ? 0 : it->second;
    }

    uint8_t siigetbyte(uint16_t slave, uint16_t address) override {
        this->eeprom_held.insert(slave);
        const auto &bytes = this->slave_infos.at(slave - 1).sii_bytes;
        const auto it = bytes.find(address);
        return it == bytes.end() ? 0 : it->second;
    }

    void siistring(char *str, uint16_t slave, uint16_t sn) override {
        const auto &strings = this->slave_infos.at(slave - 1).sii_strings;
        const auto it = strings.find(sn);
        const std::string value = it == strings.end() ? "" : it->second;
        std::strncpy(str, value.c_str(), EC_MAXNAME);
        str[EC_MAXNAME] = '\0';
    }

    void eeprom2pdi(uint16_t slave) override { this->eeprom_held.erase(slave); }

    int readODdescription(uint16_t item, ec_ODlistt *od_list) override {
        const auto &names = this->slave_infos.at(od_list->Slave - 1).od_names;
        const auto it = names.find(od_list->Index[item]);
        if (it == names.end()) return 0;
        std::strncpy(od_list->Name[item], it->second.c_str(), EC_MAXNAME);
        od_list->Name[item][EC_MAXNAME] = '\0';
        return 1;
    }

    int readOEsingle(
        uint16_t item,
        uint8_t sub_index,
        ec_ODlistt *od_list,
        ec_OElistt *oe_list
    ) override {
        const auto &entries = this->slave_infos.at(od_list->Slave - 1).oe_entries;
        const auto it = entries.find({od_list->Index[item], sub_index});
        if (it == entries.end()) {
            ec_errort err{};
            err.Etype = EC_ERR_TYPE_SDOINFO_ERROR;
            err.Slave = od_list->Slave;
            this->errors.push_back(err);
            return 0;
        }
        oe_list->Entries++;
        std::strncpy(oe_list->Name[sub_index], it->second.first.c_str(), EC_MAXNAME);
        oe_list->DataType[sub_index] = it->second.second;
        return 1;
    }
};

}
