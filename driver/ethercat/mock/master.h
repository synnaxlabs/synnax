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
#include <cstring>
#include <mutex>
#include <span>
#include <unordered_map>
#include <vector>

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/master/master.h"

namespace ethercat::mock {
/// Key for PDO offset cache lookup.
struct PDOEntryKey {
    uint16_t slave_position;
    uint16_t index;
    uint8_t subindex;
    bool is_input;

    bool operator==(const PDOEntryKey &other) const {
        return slave_position == other.slave_position && index == other.index &&
               subindex == other.subindex && is_input == other.is_input;
    }
};

/// Hash function for PDOEntryKey.
struct PDOEntryKeyHash {
    size_t operator()(const PDOEntryKey &key) const {
        return std::hash<uint64_t>()(
            (static_cast<uint64_t>(key.slave_position) << 32) |
            (static_cast<uint64_t>(key.index) << 16) |
            (static_cast<uint64_t>(key.subindex) << 8) |
            static_cast<uint64_t>(key.is_input)
        );
    }
};

/// Configuration for a simulated slave device.
struct MockSlaveConfig {
    uint16_t position;
    uint32_t vendor_id;
    uint32_t product_code;
    uint32_t revision;
    uint32_t serial;
    std::string name;
    std::vector<PDOEntryInfo> input_pdos;
    std::vector<PDOEntryInfo> output_pdos;
    bool pdos_discovered;
    std::string pdo_discovery_error;

    MockSlaveConfig(
        const uint16_t position,
        const uint32_t vendor_id,
        const uint32_t product_code,
        std::string name
    ):
        position(position),
        vendor_id(vendor_id),
        product_code(product_code),
        revision(0),
        serial(0),
        name(std::move(name)),
        pdos_discovered(false) {}

    MockSlaveConfig(
        const uint16_t position,
        const uint32_t vendor_id,
        const uint32_t product_code,
        const uint32_t serial,
        std::string name
    ):
        position(position),
        vendor_id(vendor_id),
        product_code(product_code),
        revision(0),
        serial(serial),
        name(std::move(name)),
        pdos_discovered(false) {}

    MockSlaveConfig &with_input_pdos(std::vector<PDOEntryInfo> pdos) {
        this->input_pdos = std::move(pdos);
        this->pdos_discovered = true;
        return *this;
    }

    MockSlaveConfig &with_output_pdos(std::vector<PDOEntryInfo> pdos) {
        this->output_pdos = std::move(pdos);
        this->pdos_discovered = true;
        return *this;
    }

    MockSlaveConfig &with_pdo_discovery_error(std::string error) {
        this->pdo_discovery_error = std::move(error);
        return *this;
    }
};

/// Mock implementation of Master for testing without real EtherCAT hardware.
///
/// Simulates the EtherCAT master lifecycle and cyclic operations. Can be configured
/// with virtual slaves and inject errors for testing error handling paths.
class Master final : public ethercat::master::Master {
    std::string iface_name;
    std::vector<SlaveInfo> slave_list;
    std::unordered_map<uint16_t, SlaveState> slave_states;
    std::unordered_map<PDOEntryKey, size_t, PDOEntryKeyHash> pdo_offset_cache;
    bool initialized;
    bool activated;
    mutable std::mutex mu;

    xerrors::Error inject_init_err;
    xerrors::Error inject_activate_err;
    xerrors::Error inject_receive_err;
    xerrors::Error inject_send_err;

    std::unordered_map<uint16_t, SlaveState> state_transition_failures;

    std::vector<uint8_t> iomap;
    size_t input_sz;
    size_t output_sz;
    std::vector<std::string> calls;
    size_t init_calls;

public:
    explicit Master(std::string interface_name = "mock0"):
        iface_name(std::move(interface_name)),
        initialized(false),
        activated(false),
        input_sz(0),
        output_sz(0),
        init_calls(0) {}

    /// Adds a simulated slave to the mock master.
    /// Must be called before initialize().
    void add_slave(const MockSlaveConfig &config) {
        SlaveInfo slave(
            config.position,
            config.vendor_id,
            config.product_code,
            config.revision,
            config.serial,
            config.name,
            SlaveState::INIT
        );
        slave.input_pdos = config.input_pdos;
        slave.output_pdos = config.output_pdos;
        slave.pdos_discovered = config.pdos_discovered;
        slave.pdo_discovery_error = config.pdo_discovery_error;
        this->slave_list.push_back(std::move(slave));
        this->slave_states[config.position] = SlaveState::INIT;
    }

    /// Injects an error to be returned by initialize().
    void inject_init_error(const xerrors::Error &err) { this->inject_init_err = err; }

    /// Injects an error to be returned by activate().
    void inject_activate_error(const xerrors::Error &err) {
        this->inject_activate_err = err;
    }

    /// Injects an error to be returned by receive().
    void inject_receive_error(const xerrors::Error &err) {
        this->inject_receive_err = err;
    }

    /// Injects an error to be returned by send().
    void inject_send_error(const xerrors::Error &err) { this->inject_send_err = err; }

    /// Clears all injected errors.
    void clear_injected_errors() {
        this->inject_init_err = xerrors::NIL;
        this->inject_activate_err = xerrors::NIL;
        this->inject_receive_err = xerrors::NIL;
        this->inject_send_err = xerrors::NIL;
    }

    /// Sets a slave to fail state transition to the given target state.
    void
    set_slave_transition_failure(const uint16_t position, const SlaveState target) {
        this->state_transition_failures[position] = target;
    }

    /// Directly sets the state of a specific slave.
    void set_slave_state(const uint16_t position, const SlaveState state) {
        std::lock_guard lock(this->mu);
        this->slave_states[position] = state;
        for (auto &slave: this->slave_list)
            if (slave.position == position) slave.state = state;
    }

    /// Returns the log of method calls for verification.
    const std::vector<std::string> &call_log() const { return this->calls; }

    /// Clears the method call log.
    void clear_call_log() { this->calls.clear(); }

    /// Checks if a specific method was called.
    bool was_called(const std::string &method) const {
        return std::find(this->calls.begin(), this->calls.end(), method) !=
               this->calls.end();
    }

    xerrors::Error initialize() override {
        std::lock_guard lock(this->mu);
        this->calls.push_back("initialize");
        this->init_calls++;
        if (this->inject_init_err) return this->inject_init_err;
        this->initialized = true;
        return xerrors::NIL;
    }

    xerrors::Error activate() override {
        std::lock_guard lock(this->mu);
        this->calls.push_back("activate");
        if (this->inject_activate_err) return this->inject_activate_err;
        if (!this->initialized)
            return xerrors::Error(ACTIVATION_ERROR, "master not initialized");
        this->activated = true;
        this->input_sz = this->slave_list.size() * 4;
        this->output_sz = this->slave_list.size() * 4;
        this->iomap.resize(this->input_sz + this->output_sz, 0);
        this->cache_pdo_offsets();
        for (auto &[pos, state]: this->slave_states) {
            auto it = this->state_transition_failures.find(pos);
            if (it != this->state_transition_failures.end() &&
                it->second == SlaveState::OP)
                state = SlaveState::SAFE_OP;
            else
                state = SlaveState::OP;
        }
        for (auto &slave: this->slave_list) {
            auto it = this->state_transition_failures.find(slave.position);
            if (it != this->state_transition_failures.end() &&
                it->second == SlaveState::OP)
                slave.state = SlaveState::SAFE_OP;
            else
                slave.state = SlaveState::OP;
        }
        return xerrors::NIL;
    }

    void deactivate() override {
        std::lock_guard lock(this->mu);
        this->calls.push_back("deactivate");
        this->activated = false;
        this->initialized = false;
        this->pdo_offset_cache.clear();
        this->input_sz = 0;
        this->output_sz = 0;
        for (auto &[pos, state]: this->slave_states)
            state = SlaveState::INIT;
        for (auto &slave: this->slave_list)
            slave.state = SlaveState::INIT;
    }

    xerrors::Error receive() override {
        std::lock_guard lock(this->mu);
        this->calls.push_back("receive");
        if (this->inject_receive_err) return this->inject_receive_err;
        return xerrors::NIL;
    }

    xerrors::Error send() override {
        std::lock_guard lock(this->mu);
        this->calls.push_back("send");
        if (this->inject_send_err) return this->inject_send_err;
        return xerrors::NIL;
    }

    std::span<const uint8_t> input_data() override {
        if (!this->activated) return {};
        return {this->iomap.data() + this->output_sz, this->input_sz};
    }

    std::span<uint8_t> output_data() override {
        if (!this->activated) return {};
        return {this->iomap.data(), this->output_sz};
    }

    size_t pdo_offset(const PDOEntry &entry) const override {
        std::lock_guard lock(this->mu);
        PDOEntryKey key{
            entry.slave_position,
            entry.index,
            entry.subindex,
            entry.is_input
        };
        auto it = this->pdo_offset_cache.find(key);
        if (it != this->pdo_offset_cache.end()) return it->second;
        return 0;
    }

    std::vector<SlaveInfo> slaves() const override {
        std::lock_guard lock(this->mu);
        return this->slave_list;
    }

    SlaveState slave_state(const uint16_t position) const override {
        std::lock_guard lock(this->mu);
        const auto it = this->slave_states.find(position);
        if (it == this->slave_states.end()) return SlaveState::UNKNOWN;
        return it->second;
    }

    bool all_slaves_operational() const override {
        std::lock_guard lock(this->mu);
        for (const auto &[pos, state]: this->slave_states)
            if (state != SlaveState::OP) return false;
        return !this->slave_states.empty();
    }

    std::string interface_name() const override { return this->iface_name; }

    /// Returns whether the master has been initialized.
    bool is_initialized() const {
        std::lock_guard lock(this->mu);
        return this->initialized;
    }

    /// Returns whether the master has been activated.
    bool is_activated() const {
        std::lock_guard lock(this->mu);
        return this->activated;
    }

    /// Returns the number of slaves configured.
    size_t slave_count() const {
        std::lock_guard lock(this->mu);
        return this->slave_list.size();
    }

    /// Checks if any slaves have the given state.
    bool has_slave_in_state(const SlaveState state) const {
        std::lock_guard lock(this->mu);
        for (const auto &[pos, s]: this->slave_states)
            if (s == state) return true;
        return false;
    }

    /// Returns the count of slaves in the given state.
    size_t slaves_in_state(const SlaveState state) const {
        std::lock_guard lock(this->mu);
        size_t count = 0;
        for (const auto &[pos, s]: this->slave_states)
            if (s == state) ++count;
        return count;
    }

    /// Returns the number of times initialize() was called.
    size_t init_call_count() const {
        std::lock_guard lock(this->mu);
        return this->init_calls;
    }

    /// Resets the initialize call counter.
    void reset_init_call_count() {
        std::lock_guard lock(this->mu);
        this->init_calls = 0;
    }

    /// Sets a value in the input region of the IOmap for testing.
    template<typename T>
    void set_input(const size_t offset, const T value) {
        if (offset + sizeof(T) <= this->input_sz)
            std::memcpy(
                this->iomap.data() + this->output_sz + offset,
                &value,
                sizeof(T)
            );
    }

    /// Gets a value from the output region of the IOmap for verification.
    template<typename T>
    T get_output(const size_t offset) const {
        T value{};
        if (offset + sizeof(T) <= this->output_sz)
            std::memcpy(&value, this->iomap.data() + offset, sizeof(T));
        return value;
    }

private:
    void cache_pdo_offsets() {
        this->pdo_offset_cache.clear();
        size_t input_offset = 0;
        size_t output_offset = 0;
        for (const auto &slave: this->slave_list) {
            for (const auto &pdo: slave.input_pdos) {
                PDOEntryKey key{slave.position, pdo.index, pdo.subindex, true};
                this->pdo_offset_cache[key] = input_offset;
                input_offset += pdo.byte_length();
            }
            for (const auto &pdo: slave.output_pdos) {
                PDOEntryKey key{slave.position, pdo.index, pdo.subindex, false};
                this->pdo_offset_cache[key] = output_offset;
                output_offset += pdo.byte_length();
            }
        }
    }
};
}
