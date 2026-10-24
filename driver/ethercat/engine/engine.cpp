// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cmath>
#include <format>
#include <utility>

#include "absl/log/check.h"
#include "absl/log/log.h"

#include "x/cpp/loop/loop.h"

#include "driver/ethercat/engine/engine.h"
#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/telem/telem.h"

namespace driver::ethercat::engine {
namespace {
/// @brief the slowest rate the bus cycles at, well inside the 100 ms sync manager
/// watchdog most slaves default to.
const x::telem::Rate MIN_CYCLE_RATE(100);

/// @brief returns the bus rate for the given fastest task rate: the smallest
/// multiple of it that is at least MIN_CYCLE_RATE.
x::telem::Rate bus_rate(const x::telem::Rate fastest) {
    if (fastest >= MIN_CYCLE_RATE) return fastest;
    return fastest * std::ceil(MIN_CYCLE_RATE.hz() / fastest.hz());
}
}

void Engine::run() {
    LOG(INFO) << "[ethercat] engine started on " << this->master->interface_name();
    const auto cycle_time = x::telem::TimeSpan(
        this->cycle_time_ns.load(std::memory_order_acquire)
    );
    if (this->config.rt_handle) {
        this->config.rt_handle->apply();
    } else {
        x::thread::rt::Config rt_cfg = this->config.rt;
        if (rt_cfg.enabled && !rt_cfg.has_timing()) {
            rt_cfg.period = cycle_time;
            rt_cfg.computation = cycle_time * 0.2;
            rt_cfg.deadline = cycle_time * 0.8;
        }
        x::thread::rt::apply_config(rt_cfg);
    }
    x::loop::Timer timer(cycle_time);

    // Track error state to avoid log spam - only log on state transitions
    bool had_receive_error = false;
    bool had_send_error = false;
    uint64_t receive_error_count = 0;
    uint64_t send_error_count = 0;

    const auto send = [&] {
        if (auto err = this->master->send(); err) {
            send_error_count++;
            if (!had_send_error) {
                LOG(WARNING) << "[ethercat] send error: " << err.message();
                had_send_error = true;
            }
        } else if (had_send_error) {
            LOG(INFO) << "[ethercat] send recovered after " << send_error_count
                      << " errors";
            had_send_error = false;
            send_error_count = 0;
        }
    };

    // Each receive collects the frame sent at the end of the previous cycle.
    this->write_outputs();
    send();
    while (true) {
        auto [elapsed, on_time] = timer.wait();
        if (!this->breaker.running()) break;
        if (!on_time && this->config.max_overrun.nanoseconds() > 0)
            VLOG(1) << "[ethercat] cycle overrun: " << elapsed;

        if (auto err = this->master->receive(); err) {
            receive_error_count++;
            if (!had_receive_error) {
                LOG(WARNING) << "[ethercat] receive error: " << err.message();
                had_receive_error = true;
                std::lock_guard lock(this->cycle_err_mu);
                this->cycle_err = err;
            }
            this->invalidate_inputs();
        } else {
            if (had_receive_error) {
                LOG(INFO) << "[ethercat] receive recovered after "
                          << receive_error_count << " errors";
                had_receive_error = false;
                receive_error_count = 0;
            }
            this->publish_inputs(this->master->input_data());
        }

        this->write_outputs();
        send();
    }

    if (had_receive_error)
        LOG(WARNING) << "[ethercat] engine stopped with " << receive_error_count
                     << " consecutive receive errors";
    if (had_send_error)
        LOG(WARNING) << "[ethercat] engine stopped with " << send_error_count
                     << " consecutive send errors";

    LOG(INFO) << "[ethercat] engine stopped";
}

void Engine::stop_locked() {
    if (!this->breaker.running()) return;
    this->breaker.stop();
    this->read_cv.notify_all();
    if (this->run_thread.joinable()) this->run_thread.join();
    this->master->deactivate();
}

void Engine::stop_if_idle() {
    std::lock_guard lock(this->master_mu);
    if (!this->should_be_running()) this->stop_locked();
}

x::errors::Error Engine::reconfigure_locked() {
    std::scoped_lock lk(this->registration_mu, this->write_mu);

    if (this->breaker.running()) {
        LOG(INFO) << "[ethercat] restarting engine " + this->master->interface_name() +
                         " for reconfiguration";
        this->restarting = true;
        this->read_cv.notify_all();
        this->breaker.stop();
        if (this->run_thread.joinable()) this->run_thread.join();
        this->master->deactivate();
    }

    std::vector<pdo::Entry> all_entries;
    for (const auto &reg: this->read_registrations)
        all_entries.insert(all_entries.end(), reg->entries.begin(), reg->entries.end());
    for (const auto &reg: this->write_registrations)
        all_entries.insert(all_entries.end(), reg->entries.begin(), reg->entries.end());

    const auto fail = [this](const x::errors::Error &err) {
        this->breaker.stop();
        return err;
    };

    this->breaker.start();
    while (this->breaker.running()) {
        if (auto err = this->master->initialize(); err) {
            if (!this->breaker.wait(err)) return fail(err);
            continue;
        }
        if (auto err = this->master->register_pdos(all_entries); err) {
            this->master->deactivate();
            if (!this->breaker.wait(err)) return fail(err);
            continue;
        }
        if (auto err = this->master->activate(); err) {
            this->master->deactivate();
            if (!this->breaker.wait(err)) return fail(err);
            continue;
        }
        break;
    }

    this->breaker.reset();
    this->config_gen.fetch_add(1, std::memory_order_release);
    this->update_read_offsets_locked();
    if (auto err = this->update_write_offsets_locked(
            this->master->output_data().size()
        );
        err) {
        this->master->deactivate();
        return fail(err);
    }
    this->update_cycle_time_locked();
    this->restarting = false;
    this->breaker.start();
    this->run_thread = std::thread(&Engine::run, this);
    return x::errors::NIL;
}

x::errors::Error
Engine::add(const std::shared_ptr<Registration> &reg, const bool is_input) {
    std::lock_guard lock(this->master_mu);
    {
        std::scoped_lock lk(this->registration_mu, this->write_mu);
        if (is_input)
            this->read_registrations.push_back(reg);
        else
            this->write_registrations.push_back(reg);
        if (auto err = this->validate_rates_locked(); err) {
            this->remove_locked(reg->id);
            return err;
        }
    }
    const auto err = this->reconfigure_locked();
    if (!err) return x::errors::NIL;
    {
        std::scoped_lock lk(this->registration_mu, this->write_mu);
        this->remove_locked(reg->id);
    }
    // The failed start took the bus down, so bring it back for the tasks that were
    // already running.
    if (this->should_be_running())
        if (const auto restore_err = this->reconfigure_locked(); restore_err)
            LOG(ERROR) << "[ethercat] failed to restart "
                       << this->master->interface_name()
                       << " for running tasks: " << restore_err.message();
    this->restarting = false;
    this->read_cv.notify_all();
    return err;
}

void Engine::remove_locked(const size_t id) {
    const auto matches = [id](const std::shared_ptr<Registration> &r) {
        return r->id == id;
    };
    std::erase_if(this->read_registrations, matches);
    std::erase_if(this->write_registrations, matches);
}

bool Engine::should_be_running() const {
    std::scoped_lock lk(this->registration_mu, this->write_mu);
    return !this->read_registrations.empty() || !this->write_registrations.empty();
}

x::errors::Error Engine::validate_rates_locked() const {
    x::telem::Rate fastest(0);
    for (const auto &reg: this->read_registrations)
        fastest = std::max(fastest, reg->rate);
    for (const auto &reg: this->write_registrations)
        fastest = std::max(fastest, reg->rate);
    if (fastest.hz() <= 0) return x::errors::NIL;
    const auto rate = bus_rate(fastest);
    for (const auto &reg: this->read_registrations)
        if (!telem::whole_ratio(rate, reg->rate))
            return x::errors::Error(
                errors::RATE_MISMATCH,
                std::format(
                    "the bus rate of {} Hz on {} is not a multiple of a {} Hz read "
                    "task sample rate. Choose rates that divide the fastest task "
                    "rate on the network.",
                    rate.hz(),
                    this->master->interface_name(),
                    reg->rate.hz()
                )
            );
    return x::errors::NIL;
}

void Engine::update_cycle_time_locked() {
    x::telem::Rate fastest(0);
    for (const auto &reg: this->read_registrations)
        fastest = std::max(fastest, reg->rate);
    for (const auto &reg: this->write_registrations)
        fastest = std::max(fastest, reg->rate);
    if (fastest.hz() > 0)
        this->cycle_time_ns.store(
            bus_rate(fastest).period().nanoseconds(),
            std::memory_order_release
        );
}

x::telem::Rate Engine::cycle_rate() const {
    const auto ns = this->cycle_time_ns.load(std::memory_order_acquire);
    return x::telem::Rate(x::telem::TimeSpan(ns));
}

void Engine::publish_inputs(const std::span<const uint8_t> src) {
    uint8_t *ptr = this->shared_input_ptr.load(std::memory_order_acquire);
    const size_t sz = this->shared_input_size.load(std::memory_order_acquire);
    DCHECK_EQ(src.size(), sz);
    const size_t n = std::min(src.size(), sz);
    this->read_seq.seq.fetch_add(1, std::memory_order_relaxed);
    std::atomic_thread_fence(std::memory_order_release);
    std::memcpy(ptr, src.data(), n);
    this->inputs_valid.store(true, std::memory_order_relaxed);
    this->read_epoch.epoch.fetch_add(1, std::memory_order_relaxed);
    this->read_seq.seq.fetch_add(1, std::memory_order_release);
    this->read_cv.notify_all();
}

void Engine::invalidate_inputs() {
    this->read_seq.seq.fetch_add(1, std::memory_order_relaxed);
    std::atomic_thread_fence(std::memory_order_release);
    this->inputs_valid.store(false, std::memory_order_relaxed);
    this->read_epoch.epoch.fetch_add(1, std::memory_order_relaxed);
    this->read_seq.seq.fetch_add(1, std::memory_order_release);
    this->read_cv.notify_all();
}

x::errors::Error Engine::invalid_inputs_error() const {
    std::lock_guard lock(this->cycle_err_mu);
    return x::errors::Error(
        errors::CYCLE_FAILED,
        "dropped inputs from a failed bus cycle: " + this->cycle_err.message()
    );
}

void Engine::write_outputs() {
    size_t output_len = 0;
    const uint8_t *output_data = this->consume_outputs(output_len);
    auto outputs = this->master->output_data();
    if (!outputs.empty() && output_len > 0)
        std::memcpy(outputs.data(), output_data, std::min(output_len, outputs.size()));
}

const uint8_t *Engine::consume_outputs(size_t &out_len) {
    std::unique_lock lock(this->write_mu, std::try_to_lock);
    if (lock.owns_lock()) {
        if (this->write_active.size() != this->write_staging.size())
            this->write_active.resize(this->write_staging.size());
        std::memcpy(
            this->write_active.data(),
            this->write_staging.data(),
            this->write_staging.size()
        );
    }
    out_len = this->write_active.size();
    return this->write_active.data();
}

void Engine::update_read_offsets_locked() {
    const auto input = this->master->input_data();
    const size_t input_size = input.size();
    auto new_buf = std::make_unique<uint8_t[]>(input_size);
    if (input_size > 0) std::memcpy(new_buf.get(), input.data(), input_size);
    this->read_seq.seq.fetch_add(1, std::memory_order_relaxed);
    std::atomic_thread_fence(std::memory_order_release);
    this->shared_input_ptr.store(new_buf.get(), std::memory_order_relaxed);
    this->shared_input_size.store(input_size, std::memory_order_relaxed);
    this->inputs_valid.store(true, std::memory_order_relaxed);
    this->read_seq.seq.fetch_add(1, std::memory_order_release);
    this->shared_input_prev = std::move(this->shared_input_current);
    this->shared_input_current = std::move(new_buf);
    for (auto &reg: this->read_registrations) {
        reg->offsets.clear();
        for (const auto &entry: reg->entries)
            reg->offsets.push_back(this->master->pdo_offset(entry));
    }
}

x::errors::Error Engine::update_write_offsets_locked(const size_t total_size) {
    std::vector<std::vector<pdo::Offset>> offsets;
    for (const auto &reg: this->write_registrations) {
        auto &reg_offsets = offsets.emplace_back();
        for (const auto &entry: reg->entries) {
            const auto offset = this->master->pdo_offset(entry);
            if (offset.byte + telem::pdo_required_bytes(offset.bit, entry.bit_length) >
                total_size)
                return x::errors::Error(
                    errors::PDO_MAPPING_ERROR,
                    "output PDO " +
                        telem::format_index_sub_index(entry.index, entry.sub_index) +
                        " on slave " + std::to_string(entry.slave_position) +
                        " lies outside the output image"
                );
            reg_offsets.push_back(offset);
        }
    }
    for (size_t i = 0; i < offsets.size(); ++i)
        this->write_registrations[i]->offsets = std::move(offsets[i]);
    this->write_staging.assign(total_size, 0);
    this->write_active.assign(total_size, 0);
    this->rebuild_outputs_locked();
    return x::errors::NIL;
}

void Engine::rebuild_outputs_locked() {
    struct Applied {
        const pdo::Entry *entry;
        pdo::Offset offset;
        const Command *command;
    };
    std::vector<Applied> applied;
    for (const auto &reg: this->write_registrations)
        for (size_t i = 0; i < reg->entries.size(); ++i)
            if (reg->commands[i])
                applied.push_back(
                    {&reg->entries[i], reg->offsets[i], &*reg->commands[i]}
                );
    // Two writers can command the same output, so the latest command must land last.
    std::ranges::sort(applied, {}, [](const Applied &a) { return a.command->seq; });
    std::ranges::fill(this->write_staging, 0);
    for (const auto &[entry, offset, command]: applied)
        telem::write_pdo_from_value(
            this->write_staging.data() + offset.byte,
            offset.bit,
            entry->bit_length,
            entry->data_type,
            command->value
        );
}

void Engine::unregister_reader(const size_t id) {
    {
        std::lock_guard lock(this->registration_mu);
        std::erase_if(
            this->read_registrations,
            [id](const std::shared_ptr<Registration> &r) { return r->id == id; }
        );
    }
    this->stop_if_idle();
}

void Engine::unregister_writer(const size_t id) {
    {
        std::lock_guard lock(this->write_mu);
        std::erase_if(
            this->write_registrations,
            [id](const std::shared_ptr<Registration> &r) { return r->id == id; }
        );
        this->rebuild_outputs_locked();
    }
    this->stop_if_idle();
}

Engine::Engine(std::shared_ptr<master::Master> master, Config config):
    config(std::move(config)),
    breaker(
        x::breaker::Config{
            .name = "ethercat_engine",
            .base_interval = x::telem::MILLISECOND * 100,
            .max_retries = 10,
            .scale = 1.5f,
            .max_interval = x::telem::SECOND * 5
        }
    ),
    master(std::move(master)) {}

Engine::Engine(std::shared_ptr<master::Master> master):
    Engine(std::move(master), Config{}) {}

Engine::~Engine() {
    std::lock_guard lock(this->master_mu);
    this->stop_locked();
}

std::pair<std::unique_ptr<Engine::Reader>, x::errors::Error> Engine::open_reader(
    const std::vector<pdo::Entry> &entries,
    const x::telem::Rate sample_rate
) {
    size_t total_size = 0;
    for (const auto &e: entries)
        total_size += e.byte_length();

    const size_t reg_id = this->next_id.fetch_add(1, std::memory_order_relaxed);
    auto reg = std::make_shared<Registration>(
        Registration{reg_id, entries, {}, sample_rate, {}}
    );
    if (auto err = this->add(reg, true); err) return {nullptr, err};
    return {
        std::make_unique<Reader>(*this, reg_id, total_size, std::move(reg)),
        x::errors::NIL
    };
}

std::pair<std::unique_ptr<Engine::Writer>, x::errors::Error> Engine::open_writer(
    const std::vector<pdo::Entry> &entries,
    const x::telem::Rate execution_rate
) {
    const size_t reg_id = this->next_id.fetch_add(1, std::memory_order_relaxed);
    auto reg = std::make_shared<Registration>(Registration{
        reg_id,
        entries,
        {},
        execution_rate,
        std::vector<std::optional<Command>>(entries.size())
    });
    if (auto err = this->add(reg, false); err) return {nullptr, err};
    return {std::make_unique<Writer>(*this, reg_id, std::move(reg)), x::errors::NIL};
}

x::errors::Error Engine::ensure_initialized() const {
    std::lock_guard lock(this->master_mu);
    if (this->breaker.running()) return x::errors::NIL;
    return this->master->initialize();
}

std::vector<slave::DiscoveryResult> Engine::slaves() const {
    std::lock_guard lock(this->master_mu);
    return this->master->slaves();
}

std::string Engine::interface_name() const {
    return this->master->interface_name();
}

void Engine::set_slave_enabled(const uint16_t position, const bool enabled) {
    this->master->set_slave_enabled(position, enabled);
}
}
