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

#include "glog/logging.h"

#include "driver/ethercat/cyclic_engine.h"
#include "driver/ethercat/errors/errors.h"

namespace ethercat {
CyclicEngine::CyclicEngine(std::shared_ptr<Master> master, CyclicEngineConfig config):
    master_(std::move(master)),
    config_(config),
    running_(false),
    task_count_(0),
    input_cycle_count_(0) {}

CyclicEngine::~CyclicEngine() {
    running_ = false;
    if (cycle_thread_.joinable()) cycle_thread_.join();
}

std::pair<size_t, xerrors::Error> CyclicEngine::register_input_pdo(const PDOEntry &entry
) {
    std::lock_guard lock(registration_mutex_);
    if (running_)
        return {0, xerrors::Error(PDO_MAPPING_ERROR, "cannot register PDO while running")
        };

    PDOEntry input_entry = entry;
    input_entry.is_input = true;

    // Calculate relative offset within this slave's input region
    const uint16_t slave_pos = input_entry.slave_position;
    size_t relative_offset = slave_input_offsets_[slave_pos];
    slave_input_offsets_[slave_pos] += input_entry.byte_length();

    // Store registration with index = input_pdos_.size()
    const size_t registration_index = input_pdos_.size();
    input_pdos_.push_back({input_entry, relative_offset, 0});

    return {registration_index, xerrors::NIL};
}

std::pair<size_t, xerrors::Error> CyclicEngine::register_output_pdo(const PDOEntry &entry
) {
    std::lock_guard lock(registration_mutex_);
    if (running_)
        return {0, xerrors::Error(PDO_MAPPING_ERROR, "cannot register PDO while running")
        };

    PDOEntry output_entry = entry;
    output_entry.is_input = false;

    // Calculate relative offset within this slave's output region
    const uint16_t slave_pos = output_entry.slave_position;
    size_t relative_offset = slave_output_offsets_[slave_pos];
    slave_output_offsets_[slave_pos] += output_entry.byte_length();

    // Store registration with index = output_pdos_.size()
    const size_t registration_index = output_pdos_.size();
    output_pdos_.push_back({output_entry, relative_offset, 0});

    return {registration_index, xerrors::NIL};
}

xerrors::Error CyclicEngine::add_task() {
    const int prev_count = task_count_.fetch_add(1);
    if (prev_count > 0) return xerrors::NIL;

    if (auto err = master_->initialize(); err) {
        task_count_.fetch_sub(1);
        return err;
    }

    if (auto err = master_->activate(); err) {
        master_->deactivate();
        task_count_.fetch_sub(1);
        return err;
    }

    // Resolve actual PDO offsets now that the master is activated
    resolve_pdo_offsets();

    // Get the active domain from the master - this has the actual IOmap
    auto *active = master_->active_domain();
    const size_t input_sz = active ? active->input_size() : 0;
    const size_t output_sz = active ? active->output_size() : 0;

    {
        std::lock_guard lock(input_mutex_);
        input_snapshot_.resize(input_sz, 0);
    }
    {
        std::lock_guard lock(output_mutex_);
        output_buffer_.resize(output_sz, 0);
    }

    running_ = true;
    cycle_thread_ = std::thread(&CyclicEngine::cycle_loop, this);
    return xerrors::NIL;
}

void CyclicEngine::remove_task() {
    const int prev_count = task_count_.fetch_sub(1);
    if (prev_count > 1) return;

    running_ = false;
    input_cv_.notify_all();

    if (cycle_thread_.joinable()) cycle_thread_.join();

    master_->deactivate();
}

void CyclicEngine::cycle_loop() {
    LOG(INFO) << "EtherCAT cyclic engine started on " << master_->interface_name();

    const auto cycle_ns = config_.cycle_time.chrono();
    auto next_cycle = std::chrono::steady_clock::now() + cycle_ns;

    // Get the active domain from master - this has the actual IOmap buffer
    auto *active_domain = master_->active_domain();

    while (running_) {
        if (auto err = master_->receive(); err) {
            LOG(WARNING) << "EtherCAT receive error: " << err.message();
            last_error_ = err;
        }

        if (active_domain != nullptr) {
            if (auto err = master_->process(*active_domain); err) {
                LOG(WARNING) << "EtherCAT process error: " << err.message();
                last_error_ = err;
            }

            {
                std::lock_guard lock(input_mutex_);
                if (active_domain->data() != nullptr && !input_snapshot_.empty())
                    std::memcpy(
                        input_snapshot_.data(),
                        active_domain->data(),
                        active_domain->input_size()
                    );
                input_cycle_count_++;
            }
            input_cv_.notify_all();

            {
                std::lock_guard lock(output_mutex_);
                if (active_domain->data() != nullptr && !output_buffer_.empty())
                    std::memcpy(
                        active_domain->data() + active_domain->input_size(),
                        output_buffer_.data(),
                        output_buffer_.size()
                    );
            }

            if (auto err = master_->queue(*active_domain); err) {
                LOG(WARNING) << "EtherCAT queue error: " << err.message();
                last_error_ = err;
            }
        }

        if (auto err = master_->send(); err) {
            LOG(WARNING) << "EtherCAT send error: " << err.message();
            last_error_ = err;
        }

        const auto now = std::chrono::steady_clock::now();
        if (now < next_cycle)
            std::this_thread::sleep_until(next_cycle);
        else if (config_.max_overrun.nanoseconds() > 0) {
            const auto overrun = std::chrono::duration_cast<std::chrono::nanoseconds>(
                now - next_cycle
            );
            if (overrun.count() > config_.max_overrun.nanoseconds())
                LOG(WARNING) << "EtherCAT cycle overrun: "
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
    std::unique_lock lock(input_mutex_);
    const uint64_t start_count = input_cycle_count_;

    const auto timeout = config_.cycle_time * 2;
    const bool notified = input_cv_.wait_for(lock, timeout.chrono(), [&] {
        return !running_ || breaker || input_cycle_count_ > start_count;
    });

    if (!running_ || breaker) return xerrors::Error(CYCLIC_ERROR, "engine stopped");

    if (!notified) return xerrors::Error(CYCLE_OVERRUN, "timeout waiting for inputs");

    if (buffer.size() != input_snapshot_.size()) buffer.resize(input_snapshot_.size());

    std::memcpy(buffer.data(), input_snapshot_.data(), input_snapshot_.size());
    return xerrors::NIL;
}

void CyclicEngine::write_output(
    const size_t offset,
    const void *data,
    const size_t length
) {
    std::lock_guard lock(output_mutex_);
    if (offset + length <= output_buffer_.size())
        std::memcpy(output_buffer_.data() + offset, data, length);
}

uint64_t CyclicEngine::cycle_count() const {
    std::lock_guard lock(input_mutex_);
    return input_cycle_count_;
}

xerrors::Error CyclicEngine::last_error() const { return last_error_; }

void CyclicEngine::resolve_pdo_offsets() {
    std::lock_guard lock(registration_mutex_);

    // Resolve actual offsets for input PDOs
    for (auto &reg : input_pdos_) {
        const auto offsets = master_->slave_data_offsets(reg.entry.slave_position);
        // Actual offset = slave's input base offset + relative offset within slave
        reg.actual_offset = offsets.input_offset + reg.relative_offset;
    }

    // Resolve actual offsets for output PDOs
    for (auto &reg : output_pdos_) {
        const auto offsets = master_->slave_data_offsets(reg.entry.slave_position);
        // Actual offset = slave's output base offset + relative offset within slave
        reg.actual_offset = offsets.output_offset + reg.relative_offset;
    }
}

size_t CyclicEngine::get_actual_input_offset(const size_t registration_index) const {
    std::lock_guard lock(registration_mutex_);
    if (registration_index >= input_pdos_.size()) return 0;
    return input_pdos_[registration_index].actual_offset;
}

size_t CyclicEngine::get_actual_output_offset(const size_t registration_index) const {
    std::lock_guard lock(registration_mutex_);
    if (registration_index >= output_pdos_.size()) return 0;
    return output_pdos_[registration_index].actual_offset;
}
}
