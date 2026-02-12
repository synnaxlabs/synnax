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

#include "x/cpp/loop/loop.h"

#include "driver/ethercat/engine/engine.h"
#include "driver/ethercat/errors/errors.h"

namespace driver::ethercat::engine {
void Engine::run() {
    LOG(INFO) << "[ethercat] engine started on " << this->master->interface_name();
    const auto cycle_time = x::telem::TimeSpan(
        this->cycle_time_ns.load(std::memory_order_acquire)
    );
    x::thread::rt::Config rt_cfg = this->config.rt;
    if (rt_cfg.enabled && !rt_cfg.has_timing()) {
        rt_cfg.period = cycle_time;
        rt_cfg.computation = cycle_time * 0.2;
        rt_cfg.deadline = cycle_time * 0.8;
        rt_cfg.prefer_deadline_scheduler = true;
    }
    x::thread::rt::apply_config(rt_cfg);
    x::loop::Timer timer(cycle_time);

    // Track error state to avoid log spam - only log on state transitions
    bool had_receive_error = false;
    bool had_send_error = false;
    uint64_t receive_error_count = 0;
    uint64_t send_error_count = 0;

    while (this->breaker.running()) {
        if (auto err = this->master->receive(); err) {
            receive_error_count++;
            if (!had_receive_error) {
                LOG(WARNING) << "[ethercat] receive error: " << err.message();
                had_receive_error = true;
            }
        } else if (had_receive_error) {
            LOG(INFO) << "[ethercat] receive recovered after " << receive_error_count
                      << " errors";
            had_receive_error = false;
            receive_error_count = 0;
        }

        this->publish_inputs(this->master->input_data());
        size_t output_len = 0;
        const uint8_t *output_data = this->consume_outputs(output_len);
        auto outputs = this->master->output_data();
        if (!outputs.empty() && output_len > 0)
            std::memcpy(
                outputs.data(),
                output_data,
                std::min(output_len, outputs.size())
            );

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

        auto [elapsed, on_time] = timer.wait();
        if (!on_time && this->config.max_overrun.nanoseconds() > 0)
            VLOG(2) << "[ethercat] cycle overrun: " << elapsed;
    }

    if (had_receive_error)
        LOG(WARNING) << "[ethercat] engine stopped with " << receive_error_count
                     << " consecutive receive errors";
    if (had_send_error)
        LOG(WARNING) << "[ethercat] engine stopped with " << send_error_count
                     << " consecutive send errors";

    LOG(INFO) << "[ethercat] engine stopped";
}

void Engine::stop() {
    if (!this->breaker.running()) return;
    this->breaker.stop();
    this->read_cv.notify_all();
    if (this->run_thread.joinable()) this->run_thread.join();
    this->master->deactivate();
}

x::errors::Error Engine::reconfigure() {
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

    this->breaker.start();
    while (this->breaker.running()) {
        if (auto err = this->master->initialize(); err) {
            if (!this->breaker.wait(err)) {
                this->restarting = false;
                this->breaker.reset();
                return err;
            }
            continue;
        }
        if (auto err = this->master->register_pdos(all_entries); err) {
            this->master->deactivate();
            if (!this->breaker.wait(err)) {
                this->restarting = false;
                this->breaker.reset();
                return err;
            }
            continue;
        }
        if (auto err = this->master->activate(); err) {
            this->master->deactivate();
            if (!this->breaker.wait(err)) {
                this->restarting = false;
                this->breaker.reset();
                return err;
            }
            continue;
        }
        break;
    }

    this->breaker.reset();
    this->config_gen.fetch_add(1, std::memory_order_release);
    this->update_read_offsets_locked();
    this->update_write_offsets_locked(this->master->output_data().size());
    this->restarting = false;
    this->breaker.start();
    this->run_thread = std::thread(&Engine::run, this);
    return x::errors::NIL;
}

bool Engine::should_be_running() const {
    std::scoped_lock lk(this->registration_mu, this->write_mu);
    return !this->read_registrations.empty() || !this->write_registrations.empty();
}

void Engine::update_cycle_time() {
    x::telem::Rate max_rate(0);
    {
        std::lock_guard lock(this->registration_mu);
        for (const auto &reg: this->read_registrations)
            if (reg->rate > max_rate) max_rate = reg->rate;
    }
    {
        std::lock_guard lock(this->write_mu);
        for (const auto &reg: this->write_registrations)
            if (reg->rate > max_rate) max_rate = reg->rate;
    }
    if (max_rate.hz() > 0)
        this->cycle_time_ns.store(
            max_rate.period().nanoseconds(),
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
    this->read_seq.seq.fetch_add(1, std::memory_order_release);
    std::memcpy(ptr, src.data(), n);
    this->read_seq.seq.fetch_add(1, std::memory_order_release);
    this->read_epoch.epoch.fetch_add(1, std::memory_order_release);
    this->read_cv.notify_all();
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
    this->read_seq.seq.fetch_add(1, std::memory_order_release);
    this->shared_input_ptr.store(new_buf.get(), std::memory_order_release);
    this->shared_input_size.store(input_size, std::memory_order_release);
    this->read_seq.seq.fetch_add(1, std::memory_order_release);
    this->shared_input_prev = std::move(this->shared_input_current);
    this->shared_input_current = std::move(new_buf);
    for (auto &reg: this->read_registrations) {
        reg->offsets.clear();
        for (const auto &entry: reg->entries)
            reg->offsets.push_back(this->master->pdo_offset(entry));
    }
}

void Engine::update_read_offsets() {
    std::lock_guard lock(this->registration_mu);
    this->update_read_offsets_locked();
}

void Engine::update_write_offsets_locked(const size_t total_size) {
    for (auto &reg: this->write_registrations) {
        reg->offsets.clear();
        for (const auto &entry: reg->entries)
            reg->offsets.push_back(this->master->pdo_offset(entry));
    }
    const std::vector<uint8_t> old_staging = std::move(this->write_staging);
    this->write_staging.resize(total_size, 0);
    this->write_active.resize(total_size, 0);
    std::memcpy(
        this->write_staging.data(),
        old_staging.data(),
        std::min(old_staging.size(), this->write_staging.size())
    );
}

void Engine::update_write_offsets(const size_t total_size) {
    std::lock_guard lock(this->write_mu);
    this->update_write_offsets_locked(total_size);
}

void Engine::unregister_reader(const size_t id) {
    {
        std::lock_guard lock(this->registration_mu);
        std::erase_if(
            this->read_registrations,
            [id](const std::shared_ptr<Registration> &r) { return r->id == id; }
        );
    }
    this->update_cycle_time();
    if (!this->should_be_running()) this->stop();
}

void Engine::unregister_writer(const size_t id) {
    {
        std::lock_guard lock(this->write_mu);
        std::erase_if(
            this->write_registrations,
            [id](const std::shared_ptr<Registration> &r) { return r->id == id; }
        );
    }
    this->update_cycle_time();
    if (!this->should_be_running()) this->stop();
}

Engine::Engine(std::shared_ptr<master::Master> master, const Config &config):
    config(config),
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
    this->stop();
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
        Registration{reg_id, entries, {}, sample_rate}
    );
    {
        std::lock_guard lock(this->registration_mu);
        this->read_registrations.push_back(reg);
    }
    this->update_cycle_time();

    if (auto err = this->reconfigure(); err) {
        this->unregister_reader(reg_id);
        return {nullptr, err};
    }

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
    auto reg = std::make_shared<Registration>(
        Registration{reg_id, entries, {}, execution_rate}
    );
    {
        std::lock_guard lock(this->write_mu);
        this->write_registrations.push_back(reg);
    }
    this->update_cycle_time();

    if (auto err = this->reconfigure(); err) {
        this->unregister_writer(reg_id);
        return {nullptr, err};
    }

    return {std::make_unique<Writer>(*this, reg_id, std::move(reg)), x::errors::NIL};
}

x::errors::Error Engine::ensure_initialized() const {
    std::lock_guard lock(this->master_init_mu);
    return this->master->initialize();
}

std::vector<slave::DiscoveryResult> Engine::slaves() const {
    return this->master->slaves();
}

std::string Engine::interface_name() const {
    return this->master->interface_name();
}

void Engine::set_slave_enabled(const uint16_t position, const bool enabled) {
    this->master->set_slave_enabled(position, enabled);
}
}
