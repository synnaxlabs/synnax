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
#include "x/cpp/telem/frame.h"

#include "driver/ethercat/engine/engine.h"
#include "driver/ethercat/errors/errors.h"

namespace ethercat::engine {
void Engine::run() {
    LOG(INFO) << "EtherCAT engine started on " << this->master->interface_name();
    xthread::apply_rt_config(this->config.rt);
    loop::Timer timer(this->config.cycle_time);
    while (this->breaker.running()) {
        if (auto err = this->master->receive(); err)
            VLOG(2) << "EtherCAT receive error: " << err.message();
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
        if (auto err = this->master->send(); err)
            VLOG(2) << "EtherCAT send error: " << err.message();
        auto [elapsed, on_time] = timer.wait();
        if (!on_time && this->config.max_overrun.nanoseconds() > 0)
            VLOG(2) << "EtherCAT cycle overrun: " << elapsed;
    }
    LOG(INFO) << "EtherCAT engine stopped";
}

void Engine::stop() {
    if (!this->breaker.running()) return;
    this->breaker.stop();
    this->read_cv.notify_all();
    if (this->run_thread.joinable()) this->run_thread.join();
    this->master->deactivate();
}

xerrors::Error Engine::reconfigure() {
    if (this->breaker.running()) {
        LOG(INFO) << "EtherCAT engine restarting for reconfiguration";
        this->restarting = true;
        this->read_cv.notify_all();
        this->breaker.stop();
        if (this->run_thread.joinable()) this->run_thread.join();
        this->master->deactivate();
    }
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
    this->update_read_offsets();
    this->update_write_offsets(this->master->output_data().size());
    this->restarting = false;
    this->breaker.start();
    this->run_thread = std::thread(&Engine::run, this);
    return xerrors::NIL;
}

bool Engine::should_be_running() const {
    std::scoped_lock lk(this->registration_mu, this->write_mu);
    return !this->read_registrations.empty() || !this->write_registrations.empty();
}

void Engine::publish_inputs(std::span<const uint8_t> src) {
    DCHECK_EQ(src.size(), this->shared_input_buffer.size());
    const size_t n = std::min(src.size(), this->shared_input_buffer.size());
    this->read_seq.seq.fetch_add(1, std::memory_order_release);
    std::memcpy(this->shared_input_buffer.data(), src.data(), n);
    this->read_seq.seq.fetch_add(1, std::memory_order_release);
    this->read_epoch.epoch.fetch_add(1, std::memory_order_release);
    this->read_cv.notify_all();
}

const uint8_t *Engine::consume_outputs(size_t &out_len) {
    std::lock_guard lock(this->write_mu);
    if (this->write_active.size() != this->write_staging.size())
        this->write_active.resize(this->write_staging.size());
    std::memcpy(
        this->write_active.data(),
        this->write_staging.data(),
        this->write_staging.size()
    );
    out_len = this->write_active.size();
    return this->write_active.data();
}

void Engine::update_read_offsets() {
    std::lock_guard lock(this->registration_mu);
    const size_t input_size = this->master->input_data().size();
    if (this->shared_input_buffer.size() != input_size)
        this->shared_input_buffer.resize(input_size);
    for (auto &reg: this->read_registrations) {
        reg.offsets.clear();
        for (const auto &entry: reg.entries)
            reg.offsets.push_back(this->master->pdo_offset(entry));
    }
}

void Engine::update_write_offsets(const size_t total_size) {
    std::lock_guard lock(this->write_mu);
    for (auto &reg: this->write_registrations) {
        reg.offsets.clear();
        for (const auto &entry: reg.entries)
            reg.offsets.push_back(this->master->pdo_offset(entry));
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

void Engine::unregister_reader(const size_t id) {
    {
        std::lock_guard lock(this->registration_mu);
        std::erase_if(this->read_registrations, [id](const Registration &r) {
            return r.id == id;
        });
    }
    if (!this->should_be_running()) this->stop();
}

void Engine::unregister_writer(const size_t id) {
    {
        std::lock_guard lock(this->write_mu);
        std::erase_if(this->write_registrations, [id](const Registration &r) {
            return r.id == id;
        });
    }
    if (!this->should_be_running()) this->stop();
}

Engine::Reader::Reader(
    Engine &eng,
    const size_t id,
    const size_t total_size,
    std::vector<size_t> offsets,
    std::vector<size_t> lengths,
    const size_t input_frame_size
):
    engine(eng),
    id(id),
    total_size(total_size),
    offsets(std::move(offsets)),
    lengths(std::move(lengths)),
    private_buffer(input_frame_size),
    last_seen_epoch(0) {}

Engine::Reader::~Reader() {
    this->engine.unregister_reader(this->id);
}

xerrors::Error
Engine::Reader::read(const breaker::Breaker &brk, const telem::Frame &frame) const {
    uint64_t observed_epoch = 0;
    {
        std::unique_lock lock(this->engine.notify_mu);
        const auto timeout = telem::MILLISECOND * 200;
        const bool notified = this->engine.read_cv.wait_for(
            lock,
            timeout.chrono(),
            [&] {
                observed_epoch = this->engine.read_epoch.epoch.load(
                    std::memory_order_acquire
                );
                return !this->engine.breaker.running() || !brk.running() ||
                       this->engine.restarting.load(std::memory_order_acquire) ||
                       observed_epoch > this->last_seen_epoch;
            }
        );
        if (!notified)
            return xerrors::Error(CYCLE_OVERRUN, "timeout waiting for inputs");
    }

    if (this->engine.restarting.load(std::memory_order_acquire))
        return xerrors::Error(ENGINE_RESTARTING, "engine restarting");
    if (!this->engine.breaker.running() || !brk.running())
        return xerrors::Error(CYCLIC_ERROR, "engine stopped");

    this->last_seen_epoch = observed_epoch;

    uint64_t s0 = 0, s1 = 0;
    do {
        s0 = this->engine.read_seq.seq.load(std::memory_order_acquire);
        if (s0 & 1) continue;
        std::memcpy(
            this->private_buffer.data(),
            this->engine.shared_input_buffer.data(),
            this->engine.shared_input_buffer.size()
        );
        std::atomic_thread_fence(std::memory_order_acquire);
        s1 = this->engine.read_seq.seq.load(std::memory_order_acquire);
    } while (s0 != s1);

    for (size_t i = 0; i < this->offsets.size(); ++i) {
        const size_t src_offset = this->offsets[i];
        const size_t len = this->lengths[i];
        if (src_offset + len <= this->private_buffer.size()) {
            auto &series = frame.series->at(i);
            series.write_casted(
                this->private_buffer.data() + src_offset,
                1,
                series.data_type()
            );
        }
    }

    return xerrors::NIL;
}

Engine::Writer::Writer(
    Engine &eng,
    const size_t id,
    std::vector<size_t> offsets,
    std::vector<size_t> lengths
):
    engine(eng), id(id), offsets(std::move(offsets)), lengths(std::move(lengths)) {}

Engine::Writer::~Writer() {
    this->engine.unregister_writer(this->id);
}

Engine::Writer::Transaction::Transaction(
    Engine &eng,
    const std::vector<size_t> &offsets
):
    engine(eng), offsets(offsets), lock(eng.write_mu) {}

void Engine::Writer::Transaction::write(
    const size_t pdo_index,
    const void *data,
    const size_t length
) const {
    if (pdo_index >= this->offsets.size()) return;
    const size_t offset = this->offsets[pdo_index];
    if (offset + length <= this->engine.write_staging.size())
        std::memcpy(this->engine.write_staging.data() + offset, data, length);
}

Engine::Writer::Transaction Engine::Writer::open_tx() const {
    return Transaction(this->engine, this->offsets);
}

void Engine::Writer::write(
    const size_t pdo_index,
    const void *data,
    const size_t length
) const {
    this->open_tx().write(pdo_index, data, length);
}

Engine::Engine(std::shared_ptr<master::Master> master, const Config &config):
    config(config),
    breaker(
        breaker::Config{
            .name = "ethercat_engine",
            .base_interval = telem::MILLISECOND * 100,
            .max_retries = 10,
            .scale = 1.5f,
            .max_interval = telem::SECOND * 5
        }
    ),
    master(std::move(master)) {}

Engine::~Engine() {
    this->stop();
}

std::pair<std::unique_ptr<Engine::Reader>, xerrors::Error>
Engine::open_reader(const std::vector<PDOEntry> &entries) {
    std::vector<size_t> entry_lengths;
    entry_lengths.reserve(entries.size());
    size_t total_size = 0;
    for (const auto &e: entries) {
        entry_lengths.push_back(e.byte_length());
        total_size += e.byte_length();
    }

    const size_t reg_id = this->next_id.fetch_add(1, std::memory_order_relaxed);
    {
        std::lock_guard lock(this->registration_mu);
        this->read_registrations.push_back({reg_id, entries, {}});
    }

    if (auto err = this->reconfigure(); err) {
        std::lock_guard lock(this->registration_mu);
        this->read_registrations.pop_back();
        return {nullptr, err};
    }

    std::vector<size_t> resolved_offsets;
    size_t input_frame_size;
    {
        std::lock_guard lock(this->registration_mu);
        input_frame_size = this->shared_input_buffer.size();
        for (const auto &reg: this->read_registrations) {
            if (reg.id == reg_id) {
                resolved_offsets = reg.offsets;
                break;
            }
        }
    }

    return {
        std::make_unique<Reader>(
            *this,
            reg_id,
            total_size,
            std::move(resolved_offsets),
            std::move(entry_lengths),
            input_frame_size
        ),
        xerrors::NIL
    };
}

std::pair<std::unique_ptr<Engine::Writer>, xerrors::Error>
Engine::open_writer(const std::vector<PDOEntry> &entries) {
    std::vector<size_t> entry_lengths;
    for (const auto &e: entries)
        entry_lengths.push_back(e.byte_length());

    const size_t reg_id = this->next_id.fetch_add(1, std::memory_order_relaxed);
    {
        std::lock_guard lock(this->write_mu);
        this->write_registrations.push_back({reg_id, entries, {}});
    }

    if (auto err = this->reconfigure(); err) {
        std::lock_guard lock(this->write_mu);
        this->write_registrations.pop_back();
        return {nullptr, err};
    }

    std::vector<size_t> resolved_offsets;
    {
        std::lock_guard lock(this->write_mu);
        for (const auto &reg: this->write_registrations) {
            if (reg.id == reg_id) {
                resolved_offsets = reg.offsets;
                break;
            }
        }
    }

    return {
        std::make_unique<Writer>(
            *this,
            reg_id,
            std::move(resolved_offsets),
            std::move(entry_lengths)
        ),
        xerrors::NIL
    };
}
}
