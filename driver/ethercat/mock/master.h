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
#include <condition_variable>
#include <cstring>
#include <mutex>
#include <set>
#include <span>
#include <unordered_map>
#include <vector>

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/master/master.h"

namespace driver::ethercat::mock {
/// @brief an in-memory master. Inputs set by a test reach the process image on the next
/// receive(), and outputs become visible to get_output() on the next send(), as they
/// would on a real bus.
class Master final : public master::Master {
    std::string iface_name;
    std::vector<slave::DiscoveryResult> slave_list;
    std::unordered_map<uint16_t, slave::State> slave_states;
    /// @brief the AL status text of each faulted slave.
    std::unordered_map<uint16_t, std::string> faults;
    std::set<uint16_t> lost_slaves;
    std::set<uint16_t> disabled_slaves;
    std::optional<size_t> bus_slaves;
    std::optional<x::telem::TimeSpan> dc_step;
    x::telem::TimeStamp dc_now{};
    pdo::Offsets pdo_offset_cache;
    bool initialized{false};
    bool activated{false};
    mutable std::mutex mu;

    x::errors::Error inject_init_err;
    x::errors::Error inject_activate_err;
    x::errors::Error inject_receive_err;
    x::errors::Error inject_send_err;

    std::unordered_map<uint16_t, slave::State> state_transition_failures;

    std::vector<uint8_t> input_iomap;
    std::vector<uint8_t> output_iomap;
    std::vector<uint8_t> pending_inputs;
    std::vector<uint8_t> sent_outputs;
    size_t input_sz{0};
    size_t output_sz{0};
    std::vector<std::string> calls;
    size_t init_calls{0};
    size_t output_padding = 0;
    std::mutex stall_mu;
    std::condition_variable stall_cv;
    bool stalled{false};

public:
    explicit Master(std::string interface_name = "mock0"):
        iface_name(std::move(interface_name)) {}

    /// @brief adds a simulated slave to the mock master.
    void add_slave(slave::Properties props) {
        std::lock_guard lock(this->mu);
        const auto pos = props.position;
        slave::DiscoveryResult result{};
        result.properties = std::move(props);
        result.status.state = slave::State::INIT;
        result.status.pdos_discovered = true;
        this->slave_list.push_back(std::move(result));
        this->slave_states[pos] = slave::State::INIT;
    }

    /// @brief injects an error to be returned by initialize().
    void inject_init_error(const x::errors::Error &err) {
        std::lock_guard lock(this->mu);
        this->inject_init_err = err;
    }

    /// @brief injects an error to be returned by activate().
    void inject_activate_error(const x::errors::Error &err) {
        std::lock_guard lock(this->mu);
        this->inject_activate_err = err;
    }

    /// @brief injects an error to be returned by receive().
    void inject_receive_error(const x::errors::Error &err) {
        std::lock_guard lock(this->mu);
        this->inject_receive_err = err;
    }

    /// @brief injects an error to be returned by send().
    void inject_send_error(const x::errors::Error &err) {
        std::lock_guard lock(this->mu);
        this->inject_send_err = err;
    }

    /// @brief makes receive() block until resume(), which stalls the bus cycle.
    void stall() {
        std::lock_guard lock(this->stall_mu);
        this->stalled = true;
    }

    /// @brief unblocks receive() after stall().
    void resume() {
        {
            std::lock_guard lock(this->stall_mu);
            this->stalled = false;
        }
        this->stall_cv.notify_all();
    }

    /// @brief clears all injected errors.
    void clear_injected_errors() {
        std::lock_guard lock(this->mu);
        this->inject_init_err = x::errors::NIL;
        this->inject_activate_err = x::errors::NIL;
        this->inject_receive_err = x::errors::NIL;
        this->inject_send_err = x::errors::NIL;
    }

    /// @brief sets padding bytes placed before the first output PDO at the next
    /// activate().
    void set_output_padding(const size_t padding) {
        std::lock_guard lock(this->mu);
        this->output_padding = padding;
    }

    /// @brief makes a slave fail to reach the target state, both at activation and
    /// during recovery.
    void
    set_slave_transition_failure(const uint16_t position, const slave::State target) {
        std::lock_guard lock(this->mu);
        this->state_transition_failures[position] = target;
    }

    /// @brief clears every transition failure, so check() recovers faulted slaves.
    void clear_transition_failures() {
        std::lock_guard lock(this->mu);
        this->state_transition_failures.clear();
    }

    /// @brief directly sets the state of a specific slave.
    void set_slave_state(const uint16_t position, const slave::State state) {
        std::lock_guard lock(this->mu);
        this->slave_states[position] = state;
        for (auto &slave: this->slave_list)
            if (slave.properties.position == position) slave.status.state = state;
    }

    /// @brief moves a slave to the state with the AL status error flag set and the
    /// given AL status text, as a slave does when its watchdog trips.
    void fault_slave(
        const uint16_t position,
        const slave::State state,
        const std::string &al_status
    ) {
        std::lock_guard lock(this->mu);
        this->slave_states[position] = state;
        this->faults[position] = al_status;
    }

    /// @brief makes check() report a slave as not responding until lost is false.
    void set_slave_lost(const uint16_t position, const bool lost) {
        std::lock_guard lock(this->mu);
        if (lost)
            this->lost_slaves.insert(position);
        else
            this->lost_slaves.erase(position);
    }

    /// @brief returns the log of method calls for verification.
    std::vector<std::string> call_log() const {
        std::lock_guard lock(this->mu);
        return this->calls;
    }

    /// @brief clears the method call log.
    void clear_call_log() {
        std::lock_guard lock(this->mu);
        this->calls.clear();
    }

    /// @brief checks if a specific method was called.
    bool was_called(const std::string &method) const {
        std::lock_guard lock(this->mu);
        return std::ranges::find(this->calls, method) != this->calls.end();
    }

    /// @brief returns the number of times a method was called.
    size_t call_count(const std::string &method) const {
        std::lock_guard lock(this->mu);
        return std::count(this->calls.begin(), this->calls.end(), method);
    }

    x::errors::Error initialize() override {
        std::lock_guard lock(this->mu);
        this->calls.emplace_back("initialize");
        this->init_calls++;
        if (this->inject_init_err) return this->inject_init_err;
        this->initialized = true;
        return x::errors::NIL;
    }

    std::pair<size_t, x::errors::Error> count_slaves() override {
        std::lock_guard lock(this->mu);
        this->calls.emplace_back("count_slaves");
        return {this->bus_slaves.value_or(this->slave_list.size()), x::errors::NIL};
    }

    /// @brief makes count_slaves() report count instead of the number of added slaves,
    /// as when a slave is plugged in or removed after discovery.
    void set_bus_slaves(const std::optional<size_t> count) {
        std::lock_guard lock(this->mu);
        this->bus_slaves = count;
    }

    void set_slave_enabled(const uint16_t position, const bool enabled) override {
        std::lock_guard lock(this->mu);
        if (enabled)
            this->disabled_slaves.erase(position);
        else
            this->disabled_slaves.insert(position);
    }

    x::errors::Error activate() override {
        std::lock_guard lock(this->mu);
        this->calls.emplace_back("activate");
        if (this->inject_activate_err) return this->inject_activate_err;
        if (!this->initialized)
            return x::errors::Error(errors::ACTIVATION_ERROR, "master not initialized");
        this->activated = true;
        this->cache_pdo_offsets();
        if (this->input_sz == 0) this->input_sz = this->slave_list.size() * 4;
        if (this->output_sz == 0) this->output_sz = this->slave_list.size() * 4;
        this->input_iomap.assign(this->input_sz, 0);
        this->output_iomap.assign(this->output_sz, 0);
        this->sent_outputs.assign(this->output_sz, 0);
        for (auto &slave: this->slave_list) {
            const auto pos = slave.properties.position;
            if (this->disabled_slaves.contains(pos)) continue;
            const auto it = this->state_transition_failures.find(pos);
            const auto state = it != this->state_transition_failures.end() &&
                                       it->second == slave::State::OP
                                 ? slave::State::SAFE_OP
                                 : slave::State::OP;
            this->slave_states[pos] = state;
            slave.status.state = state;
        }
        return x::errors::NIL;
    }

    void deactivate() override {
        std::lock_guard lock(this->mu);
        this->calls.emplace_back("deactivate");
        this->activated = false;
        this->initialized = false;
        this->pdo_offset_cache.clear();
        this->input_sz = 0;
        this->output_sz = 0;
        this->faults.clear();
        for (auto &[pos, state]: this->slave_states)
            state = slave::State::INIT;
        for (auto &slave: this->slave_list)
            slave.status.state = slave::State::INIT;
    }

    x::errors::Error receive() override {
        {
            std::unique_lock lock(this->stall_mu);
            this->stall_cv.wait(lock, [this] { return !this->stalled; });
        }
        std::lock_guard lock(this->mu);
        this->calls.emplace_back("receive");
        if (this->inject_receive_err) return this->inject_receive_err;
        if (this->dc_step) this->dc_now = this->dc_now + *this->dc_step;
        std::memcpy(
            this->input_iomap.data(),
            this->pending_inputs.data(),
            std::min(this->pending_inputs.size(), this->input_iomap.size())
        );
        return x::errors::NIL;
    }

    std::optional<x::telem::TimeStamp> dc_time() const override {
        std::lock_guard lock(this->mu);
        if (!this->dc_step) return std::nullopt;
        return this->dc_now;
    }

    /// @brief gives the bus a distributed clock that advances by step at each
    /// receive(), or removes it when step is nullopt.
    void set_dc_step(const std::optional<x::telem::TimeSpan> step) {
        std::lock_guard lock(this->mu);
        this->dc_step = step;
    }

    x::errors::Error send() override {
        std::lock_guard lock(this->mu);
        this->calls.emplace_back("send");
        if (this->inject_send_err) return this->inject_send_err;
        this->sent_outputs = this->output_iomap;
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

    std::pair<pdo::Offset, x::errors::Error>
    pdo_offset(const pdo::Entry &entry) const override {
        std::lock_guard lock(this->mu);
        return pdo::find_offset(this->pdo_offset_cache, entry);
    }

    std::vector<slave::DiscoveryResult> slaves() const override {
        std::lock_guard lock(this->mu);
        return this->slave_list;
    }

    /// @brief reports each enabled slave, then returns each faulted or non-OP slave to
    /// OP unless set_slave_transition_failure() blocks it.
    std::vector<slave::Health> check(const x::breaker::Breaker &breaker) override {
        std::lock_guard lock(this->mu);
        this->calls.emplace_back("check");
        std::vector<slave::Health> health;
        for (const auto &slave: this->slave_list) {
            const auto pos = slave.properties.position;
            if (this->disabled_slaves.contains(pos)) continue;
            const auto fault = this->faults.find(pos);
            health.push_back({
                .position = pos,
                .state = this->slave_states[pos],
                .faulted = fault != this->faults.end(),
                .lost = this->lost_slaves.contains(pos),
                .al_status = fault != this->faults.end() ? fault->second : "",
            });
            if (this->lost_slaves.contains(pos)) continue;
            const auto blocked = this->state_transition_failures.find(pos);
            if (!breaker.running() ||
                (blocked != this->state_transition_failures.end() &&
                 blocked->second == slave::State::OP))
                continue;
            this->slave_states[pos] = slave::State::OP;
            this->faults.erase(pos);
        }
        return health;
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

    /// @brief returns the state of a slave, or UNKNOWN for an unknown position.
    slave::State slave_state(const uint16_t position) const {
        std::lock_guard lock(this->mu);
        const auto it = this->slave_states.find(position);
        if (it == this->slave_states.end()) return slave::State::UNKNOWN;
        return it->second;
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

    /// @brief sets a value in the input image. The bus delivers it at the next
    /// receive().
    template<typename T>
    void set_input(const size_t offset, const T value) {
        std::lock_guard lock(this->mu);
        if (this->pending_inputs.size() < offset + sizeof(T))
            this->pending_inputs.resize(offset + sizeof(T), 0);
        std::memcpy(this->pending_inputs.data() + offset, &value, sizeof(T));
    }

    /// @brief gets a value from the outputs sent at the latest send().
    template<typename T>
    T get_output(const size_t offset) const {
        std::lock_guard lock(this->mu);
        T value{};
        if (offset + sizeof(T) <= this->sent_outputs.size())
            std::memcpy(&value, this->sent_outputs.data() + offset, sizeof(T));
        return value;
    }

private:
    void cache_pdo_offsets() {
        this->pdo_offset_cache.clear();
        this->input_sz = 0;
        this->output_sz = this->output_padding;
        for (const auto &slave: this->slave_list) {
            const auto &props = slave.properties;
            if (this->disabled_slaves.contains(props.position)) continue;
            pdo::compute_offsets(
                this->pdo_offset_cache,
                props.position,
                props.input_pdos,
                true,
                this->input_sz
            );
            for (const auto &pdo: props.input_pdos)
                this->input_sz += pdo.byte_length();
            pdo::compute_offsets(
                this->pdo_offset_cache,
                props.position,
                props.output_pdos,
                false,
                this->output_sz
            );
            for (const auto &pdo: props.output_pdos)
                this->output_sz += pdo.byte_length();
        }
    }
};

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
