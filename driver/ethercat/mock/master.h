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

namespace driver::ethercat::mock {
/// @brief mock implementation of Master for testing without real hardware.
class Master final : public master::Master {
    std::string iface_name;
    std::vector<slave::DiscoveryResult> slave_list;
    std::unordered_map<uint16_t, slave::State> slave_states;
    pdo::Offsets pdo_offset_cache;
    bool initialized;
    bool activated;
    mutable std::mutex mu;

    x::errors::Error inject_init_err;
    x::errors::Error inject_activate_err;
    x::errors::Error inject_receive_err;
    x::errors::Error inject_send_err;

    std::unordered_map<uint16_t, slave::State> state_transition_failures;

    std::vector<uint8_t> input_iomap;
    std::vector<uint8_t> output_iomap;
    size_t input_sz;
    size_t output_sz;
    std::vector<std::string> calls;
    size_t init_calls;
    std::vector<pdo::Entry> registered_pdos;
    size_t output_padding_ = 0;

public:
    explicit Master(std::string interface_name = "mock0"):
        iface_name(std::move(interface_name)),
        initialized(false),
        activated(false),
        input_sz(0),
        output_sz(0),
        init_calls(0) {}

    /// @brief adds a simulated slave to the mock master.
    void add_slave(slave::Properties props) {
        const auto pos = props.position;
        slave::DiscoveryResult result{};
        result.properties = std::move(props);
        result.status.state = slave::State::INIT;
        result.status.pdos_discovered = true;
        this->slave_list.push_back(std::move(result));
        this->slave_states[pos] = slave::State::INIT;
    }

    /// @brief injects an error to be returned by initialize().
    void inject_init_error(const x::errors::Error &err) { this->inject_init_err = err; }

    /// @brief injects an error to be returned by activate().
    void inject_activate_error(const x::errors::Error &err) {
        this->inject_activate_err = err;
    }

    /// @brief injects an error to be returned by receive().
    void inject_receive_error(const x::errors::Error &err) {
        this->inject_receive_err = err;
    }

    /// @brief injects an error to be returned by send().
    void inject_send_error(const x::errors::Error &err) { this->inject_send_err = err; }

    /// @brief clears all injected errors.
    void clear_injected_errors() {
        this->inject_init_err = x::errors::NIL;
        this->inject_activate_err = x::errors::NIL;
        this->inject_receive_err = x::errors::NIL;
        this->inject_send_err = x::errors::NIL;
    }

    /// @brief sets padding bytes before output PDO offsets, simulating real masters
    /// where output offsets shift after reconfigure.
    void set_output_padding(const size_t padding) { this->output_padding_ = padding; }

    /// @brief sets a slave to fail state transition to the given target state.
    void
    set_slave_transition_failure(const uint16_t position, const slave::State target) {
        this->state_transition_failures[position] = target;
    }

    /// @brief directly sets the state of a specific slave.
    void set_slave_state(const uint16_t position, const slave::State state) {
        std::lock_guard lock(this->mu);
        this->slave_states[position] = state;
        for (auto &slave: this->slave_list)
            if (slave.properties.position == position) slave.status.state = state;
    }

    /// @brief returns the log of method calls for verification.
    const std::vector<std::string> &call_log() const { return this->calls; }

    /// @brief clears the method call log.
    void clear_call_log() { this->calls.clear(); }

    /// @brief checks if a specific method was called.
    bool was_called(const std::string &method) const {
        return std::find(this->calls.begin(), this->calls.end(), method) !=
               this->calls.end();
    }

    x::errors::Error initialize() override {
        std::lock_guard lock(this->mu);
        this->calls.push_back("initialize");
        this->init_calls++;
        if (this->inject_init_err) return this->inject_init_err;
        this->initialized = true;
        return x::errors::NIL;
    }

    x::errors::Error register_pdos(const std::vector<pdo::Entry> &entries) override {
        std::lock_guard lock(this->mu);
        this->calls.push_back("register_pdos");
        this->registered_pdos = entries;
        return x::errors::NIL;
    }

    void set_slave_enabled(uint16_t, bool) override {}

    x::errors::Error activate() override {
        std::lock_guard lock(this->mu);
        this->calls.push_back("activate");
        if (this->inject_activate_err) return this->inject_activate_err;
        if (!this->initialized)
            return x::errors::Error(errors::ACTIVATION_ERROR, "master not initialized");
        this->activated = true;
        this->input_sz = 0;
        this->output_sz = 0;
        if (!this->registered_pdos.empty()) {
            for (const auto &pdo: this->registered_pdos) {
                if (pdo.is_input)
                    this->input_sz += pdo.byte_length();
                else
                    this->output_sz += pdo.byte_length();
            }
        } else {
            for (const auto &slave: this->slave_list) {
                for (const auto &pdo: slave.properties.input_pdos)
                    this->input_sz += pdo.byte_length();
                for (const auto &pdo: slave.properties.output_pdos)
                    this->output_sz += pdo.byte_length();
            }
        }
        this->output_sz += this->output_padding_;
        if (this->input_sz == 0) this->input_sz = this->slave_list.size() * 4;
        if (this->output_sz == 0) this->output_sz = this->slave_list.size() * 4;
        this->input_iomap.resize(this->input_sz, 0);
        this->output_iomap.resize(this->output_sz, 0);
        this->cache_pdo_offsets();
        for (auto &[pos, state]: this->slave_states) {
            auto it = this->state_transition_failures.find(pos);
            if (it != this->state_transition_failures.end() &&
                it->second == slave::State::OP)
                state = slave::State::SAFE_OP;
            else
                state = slave::State::OP;
        }
        for (auto &slave: this->slave_list) {
            auto it = this->state_transition_failures.find(slave.properties.position);
            if (it != this->state_transition_failures.end() &&
                it->second == slave::State::OP)
                slave.status.state = slave::State::SAFE_OP;
            else
                slave.status.state = slave::State::OP;
        }
        return x::errors::NIL;
    }

    void deactivate() override {
        std::lock_guard lock(this->mu);
        this->calls.push_back("deactivate");
        this->activated = false;
        this->initialized = false;
        this->pdo_offset_cache.clear();
        this->registered_pdos.clear();
        this->input_sz = 0;
        this->output_sz = 0;
        for (auto &[pos, state]: this->slave_states)
            state = slave::State::INIT;
        for (auto &slave: this->slave_list)
            slave.status.state = slave::State::INIT;
    }

    x::errors::Error receive() override {
        std::lock_guard lock(this->mu);
        this->calls.push_back("receive");
        if (this->inject_receive_err) return this->inject_receive_err;
        return x::errors::NIL;
    }

    x::errors::Error send() override {
        std::lock_guard lock(this->mu);
        this->calls.push_back("send");
        if (this->inject_send_err) return this->inject_send_err;
        return x::errors::NIL;
    }

    std::span<const uint8_t> input_data() override {
        if (!this->activated) return {};
        return {this->input_iomap.data(), this->input_sz};
    }

    std::span<uint8_t> output_data() override {
        if (!this->activated) return {};
        return {this->output_iomap.data(), this->output_sz};
    }

    pdo::Offset pdo_offset(const pdo::Entry &entry) const override {
        std::lock_guard lock(this->mu);
        return pdo::find_offset(this->pdo_offset_cache, entry);
    }

    std::vector<slave::DiscoveryResult> slaves() const override {
        std::lock_guard lock(this->mu);
        return this->slave_list;
    }

    slave::State slave_state(const uint16_t position) const override {
        std::lock_guard lock(this->mu);
        const auto it = this->slave_states.find(position);
        if (it == this->slave_states.end()) return slave::State::UNKNOWN;
        return it->second;
    }

    bool all_slaves_operational() const override {
        std::lock_guard lock(this->mu);
        for (const auto &[pos, state]: this->slave_states)
            if (state != slave::State::OP) return false;
        return !this->slave_states.empty();
    }

    std::string interface_name() const override { return this->iface_name; }

    /// @brief returns whether the master has been initialized.
    bool is_initialized() const {
        std::lock_guard lock(this->mu);
        return this->initialized;
    }

    /// @brief returns whether the master has been activated.
    bool is_activated() const {
        std::lock_guard lock(this->mu);
        return this->activated;
    }

    /// @brief returns the number of slaves configured.
    size_t slave_count() const {
        std::lock_guard lock(this->mu);
        return this->slave_list.size();
    }

    /// @brief checks if any slaves have the given state.
    bool has_slave_in_state(const slave::State state) const {
        std::lock_guard lock(this->mu);
        for (const auto &[pos, s]: this->slave_states)
            if (s == state) return true;
        return false;
    }

    /// @brief returns the count of slaves in the given state.
    size_t slaves_in_state(const slave::State state) const {
        std::lock_guard lock(this->mu);
        size_t count = 0;
        for (const auto &[pos, s]: this->slave_states)
            if (s == state) ++count;
        return count;
    }

    /// @brief returns the number of times initialize() was called.
    size_t init_call_count() const {
        std::lock_guard lock(this->mu);
        return this->init_calls;
    }

    /// @brief resets the initialize call counter.
    void reset_init_call_count() {
        std::lock_guard lock(this->mu);
        this->init_calls = 0;
    }

    /// @brief sets a value in the input region for testing.
    template<typename T>
    void set_input(const size_t offset, const T value) {
        if (offset + sizeof(T) <= this->input_sz)
            std::memcpy(this->input_iomap.data() + offset, &value, sizeof(T));
    }

    /// @brief gets a value from the output region for verification.
    template<typename T>
    T get_output(const size_t offset) const {
        T value{};
        if (offset + sizeof(T) <= this->output_sz)
            std::memcpy(&value, this->output_iomap.data() + offset, sizeof(T));
        return value;
    }

private:
    void cache_pdo_offsets() {
        this->pdo_offset_cache.clear();
        if (!this->registered_pdos.empty()) {
            pdo::compute_offsets(
                this->pdo_offset_cache,
                this->registered_pdos,
                0,
                this->output_padding_
            );
        } else {
            size_t input_byte_offset = 0;
            size_t output_byte_offset = this->output_padding_;
            for (const auto &slave: this->slave_list) {
                const auto &props = slave.properties;
                pdo::compute_offsets(
                    this->pdo_offset_cache,
                    props.position,
                    props.input_pdos,
                    true,
                    input_byte_offset
                );
                for (const auto &pdo: props.input_pdos)
                    input_byte_offset += pdo.byte_length();
                pdo::compute_offsets(
                    this->pdo_offset_cache,
                    props.position,
                    props.output_pdos,
                    false,
                    output_byte_offset
                );
                for (const auto &pdo: props.output_pdos)
                    output_byte_offset += pdo.byte_length();
            }
        }
    }
};

/// @brief mock implementation of master::Manager for testing.
class Manager final : public master::Manager {
    std::vector<master::Info> infos;
    std::unordered_map<std::string, std::shared_ptr<Master>> masters;

public:
    /// @brief configures a master to be returned by enumerate() and create().
    void configure(const std::string &key, std::shared_ptr<Master> m) {
        this->infos.push_back({key, "Mock " + key});
        this->masters[key] = std::move(m);
    }

    [[nodiscard]] std::vector<master::Info> enumerate() override { return this->infos; }

    [[nodiscard]] std::pair<std::shared_ptr<master::Master>, x::errors::Error>
    create(const std::string &key) override {
        auto it = this->masters.find(key);
        if (it != this->masters.end()) return {it->second, x::errors::NIL};
        return {
            nullptr,
            x::errors::Error(
                errors::MASTER_INIT_ERROR,
                "no mock master configured for key: " + key
            )
        };
    }
};

}
