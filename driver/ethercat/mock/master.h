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
#include <unordered_map>
#include <vector>

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/master/master.h"

namespace ethercat::mock {
/// Mock implementation of Domain for testing without real EtherCAT hardware.
///
/// Provides a simulated process data buffer that can be pre-populated with
/// test values or inspected after write operations.
class Domain final : public ethercat::Domain {
    std::vector<uint8_t> domain_data;
    size_t input_sz;
    size_t output_sz;
    std::vector<std::pair<PDOEntry, size_t>> registered_pdo_entries;

public:
    Domain(): input_sz(0), output_sz(0) {}

    std::pair<size_t, xerrors::Error> register_pdo(const PDOEntry &entry) override {
        const size_t byte_len = entry.byte_length();
        size_t offset;
        if (entry.is_input) {
            offset = this->input_sz;
            this->input_sz += byte_len;
        } else {
            offset = this->output_sz;
            this->output_sz += byte_len;
        }
        this->registered_pdo_entries.emplace_back(entry, offset);
        this->domain_data.resize(this->input_sz + this->output_sz, 0);
        return {offset, xerrors::NIL};
    }

    uint8_t *data() override { return this->domain_data.data(); }

    size_t size() const override { return this->domain_data.size(); }

    size_t input_size() const override { return this->input_sz; }

    size_t output_size() const override { return this->output_sz; }

    /// Sets a value in the input region of the domain buffer for testing.
    /// @tparam T The type of value to set.
    /// @param offset Byte offset into the input region.
    /// @param value The value to write.
    template<typename T>
    void set_input(const size_t offset, const T value) {
        if (offset + sizeof(T) <= this->input_sz)
            std::memcpy(this->domain_data.data() + offset, &value, sizeof(T));
    }

    /// Gets a value from the output region of the domain buffer for verification.
    /// @tparam T The type of value to get.
    /// @param offset Byte offset into the output region.
    /// @returns The value at the specified offset.
    template<typename T>
    T get_output(const size_t offset) const {
        T value{};
        if (offset + sizeof(T) <= this->output_sz)
            std::memcpy(
                &value,
                this->domain_data.data() + this->input_sz + offset,
                sizeof(T)
            );
        return value;
    }

    /// Returns the list of registered PDO entries with their offsets.
    const std::vector<std::pair<PDOEntry, size_t>> &registered_pdos() const {
        return this->registered_pdo_entries;
    }

    /// Sets raw data at a specific offset in the input region.
    /// @param offset Byte offset into the input region.
    /// @param src_data Pointer to the source data.
    /// @param length Number of bytes to copy.
    void
    set_input_data(const size_t offset, const void *src_data, const size_t length) {
        if (offset + length <= this->input_sz)
            std::memcpy(this->domain_data.data() + offset, src_data, length);
    }

    /// Gets raw data from a specific offset in the output region.
    /// @returns Vector containing the output data.
    std::vector<uint8_t> get_output_data() const {
        return std::vector<uint8_t>(
            this->domain_data.begin() + static_cast<std::ptrdiff_t>(this->input_sz),
            this->domain_data.end()
        );
    }

    /// Gets raw data from the entire domain buffer.
    /// @returns Vector containing all domain data.
    std::vector<uint8_t> get_all_data() const { return this->domain_data; }

    /// Clears all domain data to zero.
    void clear_data() {
        std::fill(this->domain_data.begin(), this->domain_data.end(), 0);
    }

    /// Sets the domain sizes directly for testing.
    void set_sizes(size_t input_size, size_t output_size) {
        this->input_sz = input_size;
        this->output_sz = output_size;
        this->domain_data.resize(this->input_sz + this->output_sz, 0);
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
        name(std::move(name)) {}

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
        name(std::move(name)) {}
};

/// Mock implementation of Master for testing without real EtherCAT hardware.
///
/// Simulates the EtherCAT master lifecycle and cyclic operations. Can be configured
/// with virtual slaves and inject errors for testing error handling paths.
class Master final : public ethercat::Master {
    std::string iface_name;
    std::vector<SlaveInfo> slave_list;
    std::unordered_map<uint16_t, SlaveState> slave_states;
    bool initialized;
    bool activated;
    mutable std::mutex mu;

    xerrors::Error inject_init_err;
    xerrors::Error inject_activate_err;
    xerrors::Error inject_receive_err;
    xerrors::Error inject_send_err;
    xerrors::Error inject_process_err;
    xerrors::Error inject_queue_err;

    std::unordered_map<uint16_t, SlaveState> state_transition_failures;

    std::unique_ptr<Domain> active_dom;
    std::vector<std::string> calls;

    /// Counter for initialize() calls to verify restart behavior.
    size_t init_calls;

public:
    explicit Master(std::string interface_name = "mock0"):
        iface_name(std::move(interface_name)),
        initialized(false),
        activated(false),
        init_calls(0) {}

    /// Adds a simulated slave to the mock master.
    /// Must be called before initialize().
    void add_slave(const MockSlaveConfig &config) {
        this->slave_list.emplace_back(
            config.position,
            config.vendor_id,
            config.product_code,
            config.revision,
            config.serial,
            config.name,
            SlaveState::INIT
        );
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

    /// Injects an error to be returned by process().
    void inject_process_error(const xerrors::Error &err) {
        this->inject_process_err = err;
    }

    /// Injects an error to be returned by queue().
    void inject_queue_error(const xerrors::Error &err) { this->inject_queue_err = err; }

    /// Clears all injected errors.
    void clear_injected_errors() {
        this->inject_init_err = xerrors::NIL;
        this->inject_activate_err = xerrors::NIL;
        this->inject_receive_err = xerrors::NIL;
        this->inject_send_err = xerrors::NIL;
        this->inject_process_err = xerrors::NIL;
        this->inject_queue_err = xerrors::NIL;
    }

    /// Sets a slave to fail state transition to the given target state.
    /// @param position The slave position.
    /// @param target The target state that should fail to reach.
    void
    set_slave_transition_failure(const uint16_t position, const SlaveState target) {
        this->state_transition_failures[position] = target;
    }

    /// Directly sets the state of a specific slave.
    /// @param position The slave position.
    /// @param state The new state to set.
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

    std::unique_ptr<ethercat::Domain> create_domain() override {
        std::lock_guard lock(this->mu);
        this->calls.push_back("create_domain");
        return std::make_unique<Domain>();
    }

    xerrors::Error activate() override {
        std::lock_guard lock(this->mu);
        this->calls.push_back("activate");
        if (this->inject_activate_err) return this->inject_activate_err;
        if (!this->initialized)
            return xerrors::Error(ACTIVATION_ERROR, "master not initialized");
        this->activated = true;
        this->active_dom = std::make_unique<Domain>();
        this->active_dom->set_sizes(
            this->slave_list.size() * 4,
            this->slave_list.size() * 4
        );
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

    xerrors::Error process(ethercat::Domain &) override {
        std::lock_guard lock(this->mu);
        this->calls.push_back("process");
        if (this->inject_process_err) return this->inject_process_err;
        return xerrors::NIL;
    }

    xerrors::Error queue(ethercat::Domain &) override {
        std::lock_guard lock(this->mu);
        this->calls.push_back("queue");
        if (this->inject_queue_err) return this->inject_queue_err;
        return xerrors::NIL;
    }

    xerrors::Error send() override {
        std::lock_guard lock(this->mu);
        this->calls.push_back("send");
        if (this->inject_send_err) return this->inject_send_err;
        return xerrors::NIL;
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

    SlaveDataOffsets slave_data_offsets(const uint16_t position) const override {
        std::lock_guard lock(this->mu);
        if (!this->activated) return SlaveDataOffsets{};

        // For mock, calculate offsets based on slave position order.
        // Inputs come first in our mock layout, then outputs.
        size_t input_offset = 0;
        size_t output_offset = 0;

        // Find cumulative offsets for slaves before this position
        for (const auto &slave: this->slave_list) {
            if (slave.position == position) {
                // For mock, each slave has a default 4 bytes input + 4 bytes output
                const size_t slave_input_size = 4;
                const size_t slave_output_size = 4;
                return SlaveDataOffsets{
                    input_offset,
                    slave_input_size,
                    output_offset,
                    slave_output_size
                };
            }
            input_offset += 4; // Default 4 bytes per slave input
            output_offset += 4; // Default 4 bytes per slave output
        }
        return SlaveDataOffsets{};
    }

    ethercat::Domain *active_domain() const override { return this->active_dom.get(); }

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

    /// Returns the active mock domain for direct manipulation in tests.
    Domain *mock_domain() const { return this->active_dom.get(); }

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
};
}
