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
#include <chrono>
#include <condition_variable>
#include <cstring>
#include <functional>
#include <map>
#include <mutex>
#include <ranges>
#include <set>
#include <span>
#include <stdexcept>
#include <string>
#include <tuple>
#include <unordered_map>
#include <utility>
#include <vector>

#include "x/cpp/loop/loop.h"

#include "driver/ethercat/coe/coe.h"
#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/master/master.h"

namespace driver::ethercat::testutil {
/// @brief the SDO abort code of a read of an object that the slave does not have.
constexpr uint32_t OBJECT_MISSING_ABORT = 0x06020000;
/// @brief an in-memory master. Inputs set by a test reach the process image on the next
/// receive(), and outputs become visible to get_output() on the next send(), as they
/// would on a real bus.
class Master final : public master::Master {
    std::vector<slave::DiscoveryResult> slave_list;
    /// @brief what scan() found, which it returns again until deactivate().
    std::vector<slave::DiscoveryResult> scanned;
    /// @brief the AL status text of each faulted slave.
    std::unordered_map<uint16_t, std::string> faults;
    /// @brief why the master could not restore the distributed clock of each slave.
    std::unordered_map<uint16_t, std::string> dc_restore_errors;
    /// @brief why each slave rejected a startup parameter.
    std::unordered_map<uint16_t, std::string> startup_errors;
    std::set<uint16_t> lost_slaves;
    master::Plan latest_plan;
    /// @brief why each slave that refuse() names refuses its state.
    std::map<uint16_t, slave::Refusal> refusing;
    std::optional<size_t> bus_slaves;
    std::optional<x::telem::TimeSpan> dc_step;
    std::vector<dc::Registers> syncs;
    /// @brief how many times as fast as the host clock the distributed clock runs, or
    /// nullopt when it does not follow the host clock.
    std::optional<double> host_dc_rate;
    /// @brief the host time at which the distributed clock read master::DC_EPOCH.
    std::optional<x::telem::TimeStamp> host_dc_start;
    x::telem::TimeStamp last_sent;
    x::telem::TimeStamp dc_now;
    pdo::Offsets pdo_offset_cache;
    bool initialized{false};
    bool activated{false};
    mutable std::mutex mu;

    x::errors::Error inject_scan_err;
    x::errors::Error inject_activate_err;
    x::errors::Error inject_state_change_err;
    x::errors::Error inject_receive_err;
    x::errors::Error inject_write_err;
    /// @brief the parameters of each write(), in order.
    std::vector<std::pair<uint16_t, std::vector<slave::StartupParameter>>> writes;

    std::unordered_map<uint16_t, slave::State> state_transition_failures;
    /// @brief the value of each object that set_object() gave, by position, index, and
    /// sub-index.
    std::map<std::tuple<uint16_t, uint16_t, uint8_t>, uint32_t> objects;

    std::vector<uint8_t> input_iomap;
    std::vector<uint8_t> output_iomap;
    std::vector<uint8_t> pending_inputs;
    std::vector<uint8_t> sent_outputs;
    size_t input_sz{0};
    size_t output_sz{0};
    std::vector<std::string> calls;
    size_t send_calls{0};
    std::condition_variable sent_cv;
    size_t start_frames{1};
    bool short_until_op{false};
    std::mutex stall_mu;
    std::condition_variable stall_cv;
    bool stalled{false};
    /// @brief whether a receive() waits for resume().
    bool blocked{false};
    /// @brief whether the next receive() stalls, set by step().
    bool stall_next{false};
    x::loop::HostClock clock;
    x::telem::TimeSpan activate_duration{0};
    /// @brief how many more frames record_dc() asked for.
    size_t frames_to_record{0};
    std::vector<x::telem::TimeStamp> recorded;
    std::condition_variable recorded_cv;
    /// @brief what name() returns.
    std::string network_name = "eth0";

public:
    /// @param clock stamps the host time at which a frame is sent.
    explicit Master(x::loop::HostClock clock = {}): clock(std::move(clock)) {}

    /// @brief records the DC time of each of the next frames that pass the reference
    /// clock, up to frames of them.
    void record_dc(const size_t frames) {
        std::lock_guard lock(this->mu);
        this->recorded.clear();
        this->frames_to_record = frames;
    }

    /// @brief waits up to 10 s for the frames that record_dc() asked for, then returns
    /// the DC time of each recorded frame.
    std::vector<x::telem::TimeStamp> recorded_dc() {
        std::unique_lock lock(this->mu);
        this->recorded_cv.wait_for(lock, std::chrono::seconds(10), [this] {
            return this->frames_to_record == 0;
        });
        return this->recorded;
    }

    /// @brief adds a simulated slave to the mock master.
    void add_slave(slave::Properties props) {
        std::lock_guard lock(this->mu);
        slave::DiscoveryResult result{};
        result.properties = std::move(props);
        result.status.state = slave::State::INIT;
        this->slave_list.push_back(std::move(result));
    }

    /// @brief makes the PDO discovery of the slave at position fail with error and find
    /// no PDOs.
    void fail_pdo_discovery(const uint16_t position, const std::string &error) {
        std::lock_guard lock(this->mu);
        for (auto &slave: this->slave_list)
            if (slave.properties.position == position) {
                slave.status.pdo_discovery_error = error;
                slave.properties.input_pdos.clear();
                slave.properties.output_pdos.clear();
            }
    }

    /// @brief gives the slave at position error and keeps its PDOs, as a master does
    /// when it falls back to the PDOs in the SII.
    void fall_back_pdo_discovery(const uint16_t position, const std::string &error) {
        std::lock_guard lock(this->mu);
        for (auto &slave: this->slave_list)
            if (slave.properties.position == position)
                slave.status.pdo_discovery_error = error;
    }

    /// @brief makes the settings read of the slave at position fail with error.
    void fail_settings_read(const uint16_t position, const std::string &error) {
        std::lock_guard lock(this->mu);
        for (auto &slave: this->slave_list)
            if (slave.properties.position == position)
                slave.status.settings_error = error;
    }

    /// @brief injects an error to be returned by scan().
    void inject_scan_error(const x::errors::Error &err) {
        std::lock_guard lock(this->mu);
        this->inject_scan_err = err;
    }

    /// @brief injects an error to be returned by activate().
    void inject_activate_error(const x::errors::Error &err) {
        std::lock_guard lock(this->mu);
        this->inject_activate_err = err;
    }

    /// @brief injects an error to be returned by write().
    void inject_write_error(const x::errors::Error &err) {
        std::lock_guard lock(this->mu);
        this->inject_write_err = err;
    }

    /// @brief returns the position and parameters of each write(), in order.
    [[nodiscard]] std::vector<std::pair<uint16_t, std::vector<slave::StartupParameter>>>
    written() const {
        std::lock_guard lock(this->mu);
        return this->writes;
    }

    /// @brief makes each call to activate() take span on the clock of the master.
    void set_activate_duration(const x::telem::TimeSpan span) {
        std::lock_guard lock(this->mu);
        this->activate_duration = span;
    }

    /// @brief makes activate() return err after the bus cycle starts, as when a slave
    /// refuses SAFE-OP or OP. A slave that refuse() names fails the start first.
    void inject_state_change_error(const x::errors::Error &err) {
        std::lock_guard lock(this->mu);
        this->inject_state_change_err = err;
    }

    /// @brief makes activate() fail with errors::STATE_CHANGE_REFUSED while the slave
    /// at position is on the bus, as when the slave refuses a state.
    void refuse(const uint16_t position, slave::Refusal refusal) {
        std::lock_guard lock(this->mu);
        this->refusing[position] = std::move(refusal);
    }

    /// @brief makes the slaves reach OP only after the cycle sends frames frames. Until
    /// then, receive() returns errors::WORKING_COUNTER_ERROR, as slaves below OP do.
    void set_start_frames(const size_t frames) {
        std::lock_guard lock(this->mu);
        this->start_frames = frames;
        this->short_until_op = true;
    }

    /// @brief injects an error to be returned by receive().
    void inject_receive_error(const x::errors::Error &err) {
        std::lock_guard lock(this->mu);
        this->inject_receive_err = err;
    }

    /// @brief makes receive() block until resume(), which stalls the bus cycle.
    void stall() {
        std::lock_guard lock(this->stall_mu);
        this->stalled = true;
    }

    /// @brief waits until a receive() blocks after stall().
    void wait_until_stalled() {
        std::unique_lock lock(this->stall_mu);
        this->stall_cv.wait(lock, [this] { return this->blocked; });
    }

    /// @brief unblocks receive() after stall().
    void resume() {
        {
            std::lock_guard lock(this->stall_mu);
            this->stalled = false;
            this->blocked = false;
            this->stall_next = false;
        }
        this->stall_cv.notify_all();
    }

    /// @brief lets the receive() that a stall blocks return, and stalls the next one.
    void step() {
        {
            std::lock_guard lock(this->stall_mu);
            this->stalled = false;
            this->blocked = false;
            this->stall_next = true;
        }
        this->stall_cv.notify_all();
    }

    /// @brief clears all injected errors and the refusals that refuse() sets.
    void clear_injected_errors() {
        std::lock_guard lock(this->mu);
        this->refusing.clear();
        this->inject_scan_err = x::errors::NIL;
        this->inject_activate_err = x::errors::NIL;
        this->inject_state_change_err = x::errors::NIL;
        this->inject_receive_err = x::errors::NIL;
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
        this->slave(position).status.state = state;
    }

    /// @brief moves a slave to the state with the AL status error flag set and the
    /// given AL status text, as a slave does when its watchdog trips.
    void fault_slave(
        const uint16_t position,
        const slave::State state,
        const std::string &fault
    ) {
        std::lock_guard lock(this->mu);
        this->slave(position).status.state = state;
        this->faults[position] = fault;
    }

    /// @brief makes check() report that the master could not restore the distributed
    /// clock of a slave, and keep the slave out of OP, until the master deactivates.
    void fail_dc_restore(const uint16_t position, const std::string &error) {
        std::lock_guard lock(this->mu);
        this->dc_restore_errors[position] = error;
    }

    /// @brief makes check() report that the slave rejected a startup parameter, and
    /// keep the slave out of OP, until the master deactivates.
    void fail_startup(const uint16_t position, const std::string &error) {
        std::lock_guard lock(this->mu);
        this->startup_errors[position] = error;
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

    /// @brief returns the number of times a method was called.
    size_t call_count(const std::string &method) const {
        std::lock_guard lock(this->mu);
        return std::count(this->calls.begin(), this->calls.end(), method);
    }

    /// @brief makes name() return name, which is "eth0" until then.
    void set_name(std::string name) {
        std::lock_guard lock(this->mu);
        this->network_name = std::move(name);
    }

    [[nodiscard]] std::string name() const override {
        std::lock_guard lock(this->mu);
        return this->network_name;
    }

    std::pair<std::vector<slave::DiscoveryResult>, x::errors::Error> scan() override {
        std::lock_guard lock(this->mu);
        this->calls.emplace_back("scan");
        if (this->inject_scan_err) return {{}, this->inject_scan_err};
        if (!this->initialized) this->scanned = this->slave_list;
        this->initialized = true;
        return {this->scanned, x::errors::NIL};
    }

    std::pair<size_t, x::errors::Error> count_slaves() override {
        std::lock_guard lock(this->mu);
        this->calls.emplace_back("count_slaves");
        if (!this->initialized)
            return {0, x::errors::Error(errors::MASTER_INIT_ERROR, "not initialized")};
        return {this->bus_slaves.value_or(this->slave_list.size()), x::errors::NIL};
    }

    /// @brief makes count_slaves() report count instead of the number of added slaves,
    /// as when a slave is plugged in or removed after discovery.
    void set_bus_slaves(const std::optional<size_t> count) {
        std::lock_guard lock(this->mu);
        this->bus_slaves = count;
    }

    /// @brief returns the plan of the latest activate().
    [[nodiscard]] master::Plan plan() const {
        std::lock_guard lock(this->mu);
        return this->latest_plan;
    }

    /// @brief returns the settings that the latest activate() gave the slave at
    /// position.
    [[nodiscard]] slave::Config config(const uint16_t position) const {
        std::lock_guard lock(this->mu);
        return this->config_locked(position);
    }

    /// @brief maps the slaves, then moves them to OP once the cycle sends a frame, as
    /// slaves with outputs need. Logs "op" when the slaves reach OP. Fails with
    /// errors::STATE_CHANGE_REFUSED when a slave that refuse() names is on the bus.
    master::Activation activate(
        const master::Plan &plan,
        const std::function<void(const master::Image &)> &start_cycle
    ) override {
        x::telem::TimeSpan duration;
        {
            std::lock_guard lock(this->mu);
            duration = this->activate_duration;
        }
        this->clock.sleep(duration);
        size_t sent = 0;
        master::Image image;
        {
            std::lock_guard lock(this->mu);
            this->calls.emplace_back("activate");
            this->latest_plan = plan;
            if (this->inject_activate_err) return {.err = this->inject_activate_err};
            if (!this->initialized)
                return {
                    .err = x::errors::Error(
                        errors::ACTIVATION_ERROR,
                        "master not initialized"
                    )
                };
            if (this->activated)
                return {
                    .err = x::errors::Error(
                        errors::ACTIVATION_ERROR,
                        "already activated"
                    )
                };
            this->activated = true;
            this->cache_pdo_offsets();
            this->input_iomap.assign(this->input_sz, 0);
            this->output_iomap.assign(this->output_sz, 0);
            this->sent_outputs.assign(this->output_sz, 0);
            sent = this->send_calls;
            image = {
                .inputs = {this->input_iomap.data(), this->input_sz},
                .outputs = {this->output_iomap.data(), this->output_sz},
                .offsets = this->pdo_offset_cache,
                .dc_sync = this->syncs,
                .slaves = this->slave_list,
            };
        }
        start_cycle(image);
        std::unique_lock lock(this->mu);
        if (!this->sent_cv.wait_for(lock, std::chrono::seconds(10), [&] {
                return this->send_calls >= sent + this->start_frames;
            }))
            return {
                .err = x::errors::Error(
                    errors::STATE_CHANGE_ERROR,
                    "failed to reach OP: no process data"
                )
            };
        std::map<uint16_t, slave::Refusal> refused;
        for (const auto &slave: this->slave_list) {
            const auto pos = slave.properties.position;
            if (!this->on_bus(pos)) continue;
            if (const auto it = this->refusing.find(pos); it != this->refusing.end())
                refused.insert(*it);
        }
        if (!refused.empty()) {
            return {
                .err = x::errors::Error(
                    errors::STATE_CHANGE_REFUSED,
                    "failed to reach SAFE-OP: " + refused.begin()->second.reason
                ),
                .refusals = std::move(refused),
            };
        }
        if (this->inject_state_change_err)
            return {.err = this->inject_state_change_err};
        this->calls.emplace_back("op");
        this->short_until_op = false;
        for (auto &slave: this->slave_list) {
            const auto pos = slave.properties.position;
            if (!this->on_bus(pos)) continue;
            const auto it = this->state_transition_failures.find(pos);
            slave.status.state = it != this->state_transition_failures.end() &&
                                         it->second == slave::State::OP
                                   ? slave::State::SAFE_OP
                                   : slave::State::OP;
        }
        return {};
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
        this->dc_restore_errors.clear();
        this->startup_errors.clear();
        for (auto &slave: this->slave_list)
            slave.status.state = slave::State::INIT;
    }

    x::errors::Error receive() override {
        {
            std::unique_lock lock(this->stall_mu);
            if (std::exchange(this->stall_next, false)) this->stalled = true;
            if (this->stalled) {
                this->blocked = true;
                this->stall_cv.notify_all();
                this->stall_cv.wait(lock, [this] { return !this->stalled; });
            }
        }
        std::lock_guard lock(this->mu);
        this->calls.emplace_back("receive");
        if (!this->activated)
            return x::errors::Error(errors::CYCLIC_ERROR, "not activated");
        const auto err = this->short_until_op ? x::errors::Error(
                                                    errors::WORKING_COUNTER_ERROR,
                                                    "expected 3, got 1"
                                                )
                                              : this->inject_receive_err;
        if (err && !err.matches(errors::WORKING_COUNTER_ERROR)) return err;
        if (this->dc_step) this->dc_now = this->dc_now + *this->dc_step;
        if (this->host_dc_rate && this->host_dc_start) {
            const auto elapsed = this->last_sent - *this->host_dc_start;
            this->dc_now = master::DC_EPOCH + elapsed * *this->host_dc_rate;
        }
        if (this->frames_to_record > 0 && (this->dc_step || this->host_dc_rate)) {
            this->recorded.push_back(this->dc_now);
            if (--this->frames_to_record == 0) this->recorded_cv.notify_all();
        }
        if (err) return err;
        std::memcpy(
            this->input_iomap.data(),
            this->pending_inputs.data(),
            std::min(this->pending_inputs.size(), this->input_iomap.size())
        );
        return x::errors::NIL;
    }

    std::optional<x::telem::TimeStamp> dc_time() const override {
        std::lock_guard lock(this->mu);
        if (!this->dc_step && !this->host_dc_rate) return std::nullopt;
        return this->dc_now;
    }

    /// @brief gives the bus a distributed clock that runs rate times as fast as the
    /// host clock. Each frame passes the reference clock as it is sent, and the next
    /// frame sent passes it at master::DC_EPOCH. A frame with a short working counter
    /// still updates it.
    void set_host_dc(const double rate) {
        std::lock_guard lock(this->mu);
        this->host_dc_rate = rate;
        this->host_dc_start.reset();
    }

    /// @brief sets the DC registers of the image that the next activate() gives.
    void set_dc_sync(const std::vector<dc::Registers> &syncs) {
        std::lock_guard lock(this->mu);
        this->syncs = syncs;
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
        if (!this->activated)
            return x::errors::Error(errors::CYCLIC_ERROR, "not activated");
        this->last_sent = this->clock.now();
        if (this->host_dc_rate && !this->host_dc_start)
            this->host_dc_start = this->last_sent;
        this->sent_outputs = this->output_iomap;
        this->send_calls++;
        this->sent_cv.notify_all();
        return x::errors::NIL;
    }

    /// @brief reports each enabled slave, then returns each faulted or non-OP slave to
    /// OP unless set_slave_transition_failure() blocks it.
    std::vector<slave::Health> check(const x::breaker::Breaker &breaker) override {
        std::lock_guard lock(this->mu);
        this->calls.emplace_back("check");
        std::vector<slave::Health> health;
        for (auto &slave: this->slave_list) {
            const auto pos = slave.properties.position;
            if (!this->on_bus(pos)) continue;
            const auto fault = this->faults.find(pos);
            const auto restore = this->dc_restore_errors.find(pos);
            const bool restore_failed = restore != this->dc_restore_errors.end();
            const auto startup = this->startup_errors.find(pos);
            const bool startup_failed = startup != this->startup_errors.end();
            health.push_back({
                .position = pos,
                .state = slave.status.state,
                .faulted = fault != this->faults.end() || restore_failed ||
                           startup_failed,
                .lost = this->lost_slaves.contains(pos),
                .al_status = fault != this->faults.end() ? fault->second : "",
                .dc_restore_error = restore_failed ? restore->second : "",
                .startup_error = startup_failed ? startup->second : "",
            });
            if (this->lost_slaves.contains(pos) || restore_failed || startup_failed)
                continue;
            const auto blocked = this->state_transition_failures.find(pos);
            if (!breaker.running() ||
                (blocked != this->state_transition_failures.end() &&
                 blocked->second == slave::State::OP))
                continue;
            slave.status.state = slave::State::OP;
            this->faults.erase(pos);
        }
        return health;
    }

    /// @brief records the write and adds the parameters to the slave in the latest
    /// plan, or returns the error that inject_write_error() set.
    x::errors::Error write(
        const uint16_t position,
        const std::vector<slave::StartupParameter> &parameters
    ) override {
        std::lock_guard lock(this->mu);
        this->calls.emplace_back("write");
        if (this->inject_write_err) return this->inject_write_err;
        this->writes.emplace_back(position, parameters);
        this->latest_plan.slaves[position].add_startup_parameters(parameters);
        return x::errors::NIL;
    }

    /// @brief returns the value that set_object() gave the object, or
    /// coe::OBJECT_MISSING.
    std::pair<uint32_t, x::errors::Error> read(
        const uint16_t position,
        const uint16_t index,
        const uint8_t sub_index
    ) override {
        std::lock_guard lock(this->mu);
        this->calls.emplace_back("read");
        const auto it = this->objects.find({position, index, sub_index});
        if (it == this->objects.end())
            return {0, coe::read_error(index, sub_index, OBJECT_MISSING_ABORT)};
        return {it->second, x::errors::NIL};
    }

    /// @brief gives the object of the slave at position the value that read() returns.
    void set_object(
        const uint16_t position,
        const uint16_t index,
        const uint8_t sub_index,
        const uint32_t value
    ) {
        std::lock_guard lock(this->mu);
        this->objects[{position, index, sub_index}] = value;
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
    /// @brief whether the slave at position takes part in the current start.
    [[nodiscard]] bool on_bus(const uint16_t position) const {
        return this->config_locked(position).enabled;
    }

    [[nodiscard]] slave::Config config_locked(const uint16_t position) const {
        const auto it = this->latest_plan.slaves.find(position);
        return it == this->latest_plan.slaves.end() ? slave::Config{} : it->second;
    }

    slave::DiscoveryResult &slave(const uint16_t position) {
        for (auto &slave: this->slave_list)
            if (slave.properties.position == position) return slave;
        throw std::out_of_range("no module at position " + std::to_string(position));
    }

    void cache_pdo_offsets() {
        this->pdo_offset_cache.clear();
        this->input_sz = 0;
        this->output_sz = 0;
        for (const auto &slave: this->slave_list) {
            const auto &props = slave.properties;
            if (!this->on_bus(props.position)) continue;
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
