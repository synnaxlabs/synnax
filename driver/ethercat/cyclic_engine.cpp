// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <cstring>

#ifdef __linux__
#include <cerrno>

#include <sched.h>
#endif

#include "glog/logging.h"

#include "driver/ethercat/cyclic_engine.h"
#include "driver/ethercat/errors/errors.h"

namespace ethercat {
CyclicEngine::CyclicEngine(std::shared_ptr<Master> master, CyclicEngineConfig config):
    master(std::move(master)),
    config(config),
    running(false),
    task_count(0),
    input_cycle_count(0),
    restarting(false),
    restart_breaker(
        breaker::Config{
            .name = "cyclic_engine_restart",
            .base_interval = telem::MILLISECOND * 100,
            .max_retries = 10,
            .scale = 1.5f,
            .max_interval = telem::SECOND * 5
        }
    ) {}

CyclicEngine::~CyclicEngine() {
    this->running = false;
    if (this->cycle_thread.joinable()) this->cycle_thread.join();
}

std::pair<PDOHandle, xerrors::Error>
CyclicEngine::register_input_pdo(const PDOEntry &entry) {
    std::lock_guard lock(this->registration_mu);

    if (this->running) {
        if (auto err = this->restart_for_reconfiguration(); err)
            return {PDOHandle::invalid(), err};
    }

    PDOEntry input_entry = entry;
    input_entry.is_input = true;

    const uint16_t slave_pos = input_entry.slave_position;
    size_t relative_offset = this->slave_input_offsets[slave_pos];
    this->slave_input_offsets[slave_pos] += input_entry.byte_length();

    const size_t registration_index = this->input_pdos.size();
    this->input_pdos.push_back({input_entry, relative_offset, 0});

    return {PDOHandle{registration_index, true}, xerrors::NIL};
}

std::pair<PDOHandle, xerrors::Error>
CyclicEngine::register_output_pdo(const PDOEntry &entry) {
    std::lock_guard lock(this->registration_mu);

    if (this->running) {
        if (auto err = this->restart_for_reconfiguration(); err)
            return {PDOHandle::invalid(), err};
    }

    PDOEntry output_entry = entry;
    output_entry.is_input = false;

    const uint16_t slave_pos = output_entry.slave_position;
    size_t relative_offset = this->slave_output_offsets[slave_pos];
    this->slave_output_offsets[slave_pos] += output_entry.byte_length();

    const size_t registration_index = this->output_pdos.size();
    this->output_pdos.push_back({output_entry, relative_offset, 0});

    return {PDOHandle{registration_index, false}, xerrors::NIL};
}

xerrors::Error CyclicEngine::add_task() {
    const int prev_count = this->task_count.fetch_add(1);
    if (prev_count > 0) return xerrors::NIL;

    if (auto err = this->master->initialize(); err) {
        this->task_count.fetch_sub(1);
        return err;
    }

    if (auto err = this->master->activate(); err) {
        this->master->deactivate();
        this->task_count.fetch_sub(1);
        return err;
    }

    {
        std::lock_guard lock(this->registration_mu);
        this->resolve_pdo_offsets();
    }

    // Get the active domain from the master - this has the actual IOmap
    auto *active = this->master->active_domain();
    const size_t input_sz = active ? active->input_size() : 0;
    const size_t output_sz = active ? active->output_size() : 0;

    {
        std::lock_guard lock(this->input_mu);
        this->input_snapshot.resize(input_sz, 0);
    }
    {
        std::lock_guard lock(this->output_mu);
        this->output_buffer.resize(output_sz, 0);
    }

    this->running = true;
    this->cycle_thread = std::thread(&CyclicEngine::cycle_loop, this);
    return xerrors::NIL;
}

void CyclicEngine::remove_task() {
    const int prev_count = this->task_count.fetch_sub(1);
    if (prev_count > 1) return;

    this->running = false;
    this->input_cv.notify_all();

    if (this->cycle_thread.joinable()) this->cycle_thread.join();

    this->master->deactivate();
}

void CyclicEngine::cycle_loop() {
    LOG(INFO) << "EtherCAT cyclic engine started on " << this->master->interface_name();

#ifdef __linux__
    if (this->config.enable_realtime) {
        struct sched_param param;
        param.sched_priority = this->config.realtime_priority;
        if (sched_setscheduler(0, SCHED_FIFO, &param) < 0)
            LOG(WARNING) << "Failed to set RT priority: " << strerror(errno)
                         << " (requires CAP_SYS_NICE or root)";
        else
            VLOG(1) << "Set RT priority to " << this->config.realtime_priority;
    }
#endif

    const auto cycle_ns = this->config.cycle_time.chrono();
    auto next_cycle = std::chrono::steady_clock::now() + cycle_ns;

    auto *active_domain = this->master->active_domain();

    while (this->running) {
        if (auto err = this->master->receive(); err) {
            VLOG(2) << "EtherCAT receive error: " << err.message();
            this->last_err = err;
        }

        if (active_domain != nullptr) {
            if (auto err = this->master->process(*active_domain); err) {
                VLOG(2) << "EtherCAT process error: " << err.message();
                this->last_err = err;
            }

            {
                std::lock_guard lock(this->input_mu);
                if (active_domain->data() != nullptr && !this->input_snapshot.empty())
                    std::memcpy(
                        this->input_snapshot.data(),
                        active_domain->data(),
                        active_domain->input_size()
                    );
                this->input_cycle_count++;
            }
            this->input_cv.notify_all();

            {
                std::lock_guard lock(this->output_mu);
                if (active_domain->data() != nullptr && !this->output_buffer.empty())
                    std::memcpy(
                        active_domain->data() + active_domain->input_size(),
                        this->output_buffer.data(),
                        this->output_buffer.size()
                    );
            }

            if (auto err = this->master->queue(*active_domain); err) {
                VLOG(2) << "EtherCAT queue error: " << err.message();
                this->last_err = err;
            }
        }

        if (auto err = this->master->send(); err) {
            VLOG(2) << "EtherCAT send error: " << err.message();
            this->last_err = err;
        }

        const auto now = std::chrono::steady_clock::now();
        if (now < next_cycle)
            std::this_thread::sleep_until(next_cycle);
        else if (this->config.max_overrun.nanoseconds() > 0) {
            const auto overrun = std::chrono::duration_cast<std::chrono::nanoseconds>(
                now - next_cycle
            );
            if (overrun.count() > this->config.max_overrun.nanoseconds())
                VLOG(2) << "EtherCAT cycle overrun: "
                        << telem::TimeSpan(overrun.count());
        }

        next_cycle += cycle_ns;
    }

    LOG(INFO) << "EtherCAT cyclic engine stopped";
}

xerrors::Error CyclicEngine::wait_for_inputs(
    std::vector<uint8_t> &buffer,
    std::atomic<bool> &breaker
) {
    std::unique_lock lock(this->input_mu);
    const uint64_t start_count = this->input_cycle_count;

    const auto timeout = this->config.cycle_time * 2;
    const bool notified = this->input_cv.wait_for(lock, timeout.chrono(), [&] {
        return !this->running || breaker || this->restarting ||
               this->input_cycle_count > start_count;
    });

    if (this->restarting) return xerrors::Error(ENGINE_RESTARTING, "engine restarting");

    if (!this->running || breaker)
        return xerrors::Error(CYCLIC_ERROR, "engine stopped");

    if (!notified) return xerrors::Error(CYCLE_OVERRUN, "timeout waiting for inputs");

    if (buffer.size() != this->input_snapshot.size())
        buffer.resize(this->input_snapshot.size());

    std::memcpy(
        buffer.data(),
        this->input_snapshot.data(),
        this->input_snapshot.size()
    );
    return xerrors::NIL;
}

void CyclicEngine::write_output(
    const size_t offset,
    const void *data,
    const size_t length
) {
    std::lock_guard lock(this->output_mu);
    if (offset + length <= this->output_buffer.size())
        std::memcpy(this->output_buffer.data() + offset, data, length);
}

uint64_t CyclicEngine::cycle_count() const {
    std::lock_guard lock(this->input_mu);
    return this->input_cycle_count;
}

xerrors::Error CyclicEngine::last_error() const {
    return this->last_err;
}

void CyclicEngine::resolve_pdo_offsets() {
    for (auto &reg: this->input_pdos) {
        const auto offsets = this->master->slave_data_offsets(reg.entry.slave_position);
        // Actual offset = slave's input base offset + relative offset within slave
        reg.actual_offset = offsets.input_offset + reg.relative_offset;
    }

    // Resolve actual offsets for output PDOs
    for (auto &reg: this->output_pdos) {
        const auto offsets = this->master->slave_data_offsets(reg.entry.slave_position);
        // Actual offset = slave's output base offset + relative offset within slave
        reg.actual_offset = offsets.output_offset + reg.relative_offset;
    }
}

size_t CyclicEngine::get_actual_input_offset(const size_t registration_index) const {
    std::lock_guard lock(this->registration_mu);
    if (registration_index >= this->input_pdos.size()) return 0;
    return this->input_pdos[registration_index].actual_offset;
}

size_t CyclicEngine::get_actual_output_offset(const size_t registration_index) const {
    std::lock_guard lock(this->registration_mu);
    if (registration_index >= this->output_pdos.size()) return 0;
    return this->output_pdos[registration_index].actual_offset;
}

xerrors::Error CyclicEngine::restart_for_reconfiguration() {
    LOG(INFO) << "EtherCAT cyclic engine restarting for reconfiguration";

    this->restarting = true;
    this->input_cv.notify_all();

    this->running = false;
    if (this->cycle_thread.joinable()) this->cycle_thread.join();

    this->master->deactivate();

    this->restart_breaker.start();
    while (this->restart_breaker.running()) {
        if (auto err = this->master->initialize(); err) {
            if (!this->restart_breaker.wait(err)) {
                this->restarting = false;
                this->restart_breaker.reset();
                return err;
            }
            continue;
        }
        if (auto err = this->master->activate(); err) {
            this->master->deactivate();
            if (!this->restart_breaker.wait(err)) {
                this->restarting = false;
                this->restart_breaker.reset();
                return err;
            }
            continue;
        }
        break;
    }
    this->restart_breaker.stop();
    this->restart_breaker.reset();

    this->resolve_pdo_offsets();

    auto *active = this->master->active_domain();
    const size_t input_sz = active ? active->input_size() : 0;
    const size_t output_sz = active ? active->output_size() : 0;

    {
        std::lock_guard lock(this->input_mu);
        this->input_snapshot.resize(input_sz, 0);
    }
    {
        std::lock_guard lock(this->output_mu);
        std::vector<uint8_t> old_output = std::move(this->output_buffer);
        this->output_buffer.resize(output_sz, 0);
        std::memcpy(
            this->output_buffer.data(),
            old_output.data(),
            std::min(old_output.size(), this->output_buffer.size())
        );
    }

    this->running = true;
    this->restarting = false;
    this->cycle_thread = std::thread(&CyclicEngine::cycle_loop, this);

    return xerrors::NIL;
}

xerrors::Error
CyclicEngine::read_input(const PDOHandle handle, void *buffer, const size_t length) {
    if (!handle.valid() || !handle.is_input)
        return xerrors::Error(PDO_MAPPING_ERROR, "invalid input handle");

    std::lock_guard reg_lock(this->registration_mu);
    if (handle.index >= this->input_pdos.size())
        return xerrors::Error(PDO_MAPPING_ERROR, "handle out of range");

    const size_t offset = this->input_pdos[handle.index].actual_offset;

    std::lock_guard lock(this->input_mu);
    if (offset + length > this->input_snapshot.size())
        return xerrors::Error(PDO_MAPPING_ERROR, "read exceeds buffer");

    std::memcpy(buffer, this->input_snapshot.data() + offset, length);
    return xerrors::NIL;
}

void CyclicEngine::write_output(
    const PDOHandle handle,
    const void *data,
    const size_t length
) {
    if (!handle.valid() || handle.is_input) return;

    std::lock_guard reg_lock(this->registration_mu);
    if (handle.index >= this->output_pdos.size()) return;

    const size_t offset = this->output_pdos[handle.index].actual_offset;

    std::lock_guard lock(this->output_mu);
    if (offset + length <= this->output_buffer.size())
        std::memcpy(this->output_buffer.data() + offset, data, length);
}
}
