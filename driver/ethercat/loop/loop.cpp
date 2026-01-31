// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cstring>

#include "glog/logging.h"

#include "x/cpp/loop/loop.h"

#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/loop/loop.h"

namespace ethercat {
Loop::Reader::Reader(ReadBuffer *buf, const size_t id, const size_t total_size):
    buffer(buf), id(id), total_size(total_size) {}

Loop::Reader::~Reader() {
    if (this->buffer != nullptr) this->buffer->unregister(this->id);
}

Loop::Reader::Reader(Reader &&other) noexcept:
    buffer(other.buffer), id(other.id), total_size(other.total_size) {
    other.buffer = nullptr;
}

Loop::Reader &Loop::Reader::operator=(Reader &&other) noexcept {
    if (this != &other) {
        if (this->buffer) this->buffer->unregister(this->id);
        this->buffer = other.buffer;
        this->id = other.id;
        this->total_size = other.total_size;
        other.buffer = nullptr;
    }
    return *this;
}

xerrors::Error
Loop::Reader::read(std::vector<uint8_t> &dest, const std::atomic<bool> &stopped) const {
    if (!this->buffer) return xerrors::Error(CYCLIC_ERROR, "reader not initialized");

    std::unique_lock lock(this->buffer->mu);
    const uint64_t last_seen = this->buffer->epoch.load(std::memory_order_acquire);
    const auto timeout = telem::MILLISECOND * 100;

    const bool notified = this->buffer->cv.wait_for(lock, timeout.chrono(), [&] {
        return !this->buffer->running || stopped || this->buffer->restarting ||
               this->buffer->epoch.load(std::memory_order_acquire) > last_seen;
    });

    if (this->buffer->restarting)
        return xerrors::Error(ENGINE_RESTARTING, "engine restarting");
    if (!this->buffer->running || stopped)
        return xerrors::Error(CYCLIC_ERROR, "engine stopped");
    if (!notified) return xerrors::Error(CYCLE_OVERRUN, "timeout waiting for inputs");

    if (dest.size() != this->buffer->data.size())
        dest.resize(this->buffer->data.size());
    std::memcpy(dest.data(), this->buffer->data.data(), this->buffer->data.size());

    return xerrors::NIL;
}

Loop::Writer::Writer(
    WriteBuffer *buf,
    const size_t id,
    std::vector<size_t> offsets,
    std::vector<size_t> lengths
):
    buffer(buf), id(id), offsets(std::move(offsets)), lengths(std::move(lengths)) {}

Loop::Writer::~Writer() {
    if (this->buffer) this->buffer->unregister(this->id);
}

Loop::Writer::Writer(Writer &&other) noexcept:
    buffer(other.buffer),
    id(other.id),
    offsets(std::move(other.offsets)),
    lengths(std::move(other.lengths)) {
    other.buffer = nullptr;
}

Loop::Writer &Loop::Writer::operator=(Writer &&other) noexcept {
    if (this != &other) {
        if (this->buffer != nullptr) this->buffer->unregister(this->id);
        this->buffer = other.buffer;
        this->id = other.id;
        this->offsets = std::move(other.offsets);
        this->lengths = std::move(other.lengths);
        other.buffer = nullptr;
    }
    return *this;
}

void Loop::Writer::write(
    const size_t pdo_index,
    const void *data,
    const size_t length
) {
    if (!this->buffer || pdo_index >= this->offsets.size()) return;
    const size_t offset = this->offsets[pdo_index];
    std::lock_guard lock(this->buffer->mu);
    if (offset + length <= this->buffer->staging.size())
        std::memcpy(this->buffer->staging.data() + offset, data, length);
}

Loop::ReadBuffer::ReadBuffer(Loop &lp): loop(lp) {}

void Loop::ReadBuffer::publish(const uint8_t *src, const size_t len) {
    {
        std::lock_guard lock(this->mu);
        if (this->data.size() != len) this->data.resize(len);
        std::memcpy(this->data.data(), src, len);
    }
    this->epoch.fetch_add(1, std::memory_order_release);
    this->cv.notify_all();
}

void Loop::ReadBuffer::set_running(const bool r) {
    this->running = r;
    if (!r) this->cv.notify_all();
}

void Loop::ReadBuffer::set_restarting(const bool r) {
    this->restarting = r;
    if (r) this->cv.notify_all();
}

void Loop::ReadBuffer::update_offsets() {
    std::lock_guard lock(this->mu);
    for (auto &reg: this->registrations) {
        reg.offsets.clear();
        for (const auto &entry: reg.entries)
            reg.offsets.push_back(this->loop.master->pdo_offset(entry));
    }
}

void Loop::ReadBuffer::unregister(const size_t id) {
    {
        std::lock_guard lock(this->mu);
        std::erase_if(this->registrations, [id](const Registration &r) {
            return r.id == id;
        });
    }
    if (!this->loop.should_be_running()) this->loop.stop();
}

xerrors::Error Loop::ReadBuffer::request_reconfiguration() {
    if (this->loop.is_running()) return this->loop.reconfigure();
    return this->loop.start();
}

std::pair<Loop::Reader, xerrors::Error>
Loop::ReadBuffer::open_reader(std::vector<PDOEntry> entries) {
    size_t total_size = 0;
    for (const auto &e: entries)
        total_size += e.byte_length();

    size_t reg_id;
    {
        std::lock_guard lock(this->mu);
        reg_id = this->next_id++;
        this->registrations.push_back({reg_id, entries, {}});
    }

    if (auto err = this->request_reconfiguration(); err) {
        std::lock_guard lock(this->mu);
        this->registrations.pop_back();
        return {Reader(nullptr, 0, 0), err};
    }

    return {Reader(this, reg_id, total_size), xerrors::NIL};
}

size_t Loop::ReadBuffer::reader_count() const {
    std::lock_guard lock(this->mu);
    return this->registrations.size();
}

std::vector<PDOEntry> Loop::ReadBuffer::all_entries() const {
    std::lock_guard lock(this->mu);
    std::vector<PDOEntry> result;
    for (const auto &reg: this->registrations)
        for (const auto &entry: reg.entries)
            result.push_back(entry);
    return result;
}

Loop::WriteBuffer::WriteBuffer(Loop &lp): loop(lp) {}

const uint8_t *Loop::WriteBuffer::consume(size_t &out_len) {
    std::lock_guard lock(this->mu);
    if (this->active.size() != this->staging.size())
        this->active.resize(this->staging.size());
    std::memcpy(this->active.data(), this->staging.data(), this->staging.size());
    out_len = this->active.size();
    return this->active.data();
}

void Loop::WriteBuffer::update_offsets(const size_t total_size) {
    std::lock_guard lock(this->mu);
    for (auto &reg: this->registrations) {
        reg.offsets.clear();
        for (const auto &entry: reg.entries)
            reg.offsets.push_back(this->loop.master->pdo_offset(entry));
    }
    std::vector<uint8_t> old_staging = std::move(this->staging);
    this->staging.resize(total_size, 0);
    this->active.resize(total_size, 0);
    std::memcpy(
        this->staging.data(),
        old_staging.data(),
        std::min(old_staging.size(), this->staging.size())
    );
}

void Loop::WriteBuffer::unregister(const size_t id) {
    {
        std::lock_guard lock(this->mu);
        std::erase_if(this->registrations, [id](const Registration &r) {
            return r.id == id;
        });
    }
    if (!this->loop.should_be_running()) this->loop.stop();
}

xerrors::Error Loop::WriteBuffer::request_reconfiguration() {
    if (this->loop.is_running()) return this->loop.reconfigure();
    return this->loop.start();
}

std::pair<Loop::Writer, xerrors::Error>
Loop::WriteBuffer::open_writer(std::vector<PDOEntry> entries) {
    std::vector<size_t> entry_lengths;
    for (const auto &e: entries)
        entry_lengths.push_back(e.byte_length());

    size_t reg_id;
    {
        std::lock_guard lock(this->mu);
        reg_id = this->next_id++;
        this->registrations.push_back({reg_id, entries, {}});
    }

    if (auto err = this->request_reconfiguration(); err) {
        std::lock_guard lock(this->mu);
        this->registrations.pop_back();
        return {Writer(nullptr, 0, {}, {}), err};
    }

    std::vector<size_t> resolved_offsets;
    {
        std::lock_guard lock(this->mu);
        for (const auto &reg: this->registrations) {
            if (reg.id == reg_id) {
                resolved_offsets = reg.offsets;
                break;
            }
        }
    }

    return {
        Writer(this, reg_id, std::move(resolved_offsets), std::move(entry_lengths)),
        xerrors::NIL
    };
}

size_t Loop::WriteBuffer::writer_count() const {
    std::lock_guard lock(this->mu);
    return this->registrations.size();
}

Loop::Loop(std::shared_ptr<Master> master, const LoopConfig &config):
    master(std::move(master)),
    config(config),
    read_buf(*this),
    write_buf(*this),
    breaker(
        breaker::Config{
            .name = "ethercat_loop",
            .base_interval = telem::MILLISECOND * 100,
            .max_retries = 10,
            .scale = 1.5f,
            .max_interval = telem::SECOND * 5
        }
    ) {}

Loop::~Loop() {
    this->stop();
}

void Loop::run() {
    LOG(INFO) << "EtherCAT loop started on " << this->master->interface_name();
    xthread::apply_rt_config(this->config.rt);

    loop::Timer timer(this->config.cycle_time);

    while (this->breaker.running()) {
        if (auto err = this->master->receive(); err)
            VLOG(2) << "EtherCAT receive error: " << err.message();

        this->read_buf.publish(this->master->input_data(), this->master->input_size());

        size_t output_len = 0;
        const uint8_t *output_data = this->write_buf.consume(output_len);
        if (this->master->output_data() != nullptr && output_len > 0)
            std::memcpy(this->master->output_data(), output_data, output_len);

        if (auto err = this->master->send(); err)
            VLOG(2) << "EtherCAT send error: " << err.message();

        auto [elapsed, on_time] = timer.wait();
        if (!on_time && this->config.max_overrun.nanoseconds() > 0)
            VLOG(2) << "EtherCAT cycle overrun: " << elapsed;
    }

    LOG(INFO) << "EtherCAT loop stopped";
}

xerrors::Error Loop::start() {
    if (this->breaker.running()) return xerrors::NIL;

    if (auto err = this->master->initialize(); err) return err;
    if (auto err = this->master->activate(); err) {
        this->master->deactivate();
        return err;
    }

    this->read_buf.update_offsets();
    this->write_buf.update_offsets(this->master->output_size());

    this->read_buf.set_running(true);
    this->breaker.start();
    this->cycle_thread = std::thread(&Loop::run, this);

    return xerrors::NIL;
}

void Loop::stop() {
    if (!this->breaker.running()) return;

    this->breaker.stop();
    this->read_buf.set_running(false);

    if (this->cycle_thread.joinable()) this->cycle_thread.join();

    this->master->deactivate();
}

xerrors::Error Loop::reconfigure() {
    LOG(INFO) << "EtherCAT loop restarting for reconfiguration";

    this->read_buf.set_restarting(true);
    this->breaker.stop();

    if (this->cycle_thread.joinable()) this->cycle_thread.join();

    this->master->deactivate();

    this->breaker.start();
    while (this->breaker.running()) {
        if (auto err = this->master->initialize(); err) {
            if (!this->breaker.wait(err)) {
                this->read_buf.set_restarting(false);
                this->breaker.reset();
                return err;
            }
            continue;
        }
        if (auto err = this->master->activate(); err) {
            this->master->deactivate();
            if (!this->breaker.wait(err)) {
                this->read_buf.set_restarting(false);
                this->breaker.reset();
                return err;
            }
            continue;
        }
        break;
    }
    this->breaker.reset();

    this->read_buf.update_offsets();
    this->write_buf.update_offsets(this->master->output_size());

    this->read_buf.set_restarting(false);
    this->read_buf.set_running(true);
    this->breaker.start();
    this->cycle_thread = std::thread(&Loop::run, this);

    return xerrors::NIL;
}

bool Loop::should_be_running() const {
    return this->read_buf.reader_count() > 0 || this->write_buf.writer_count() > 0;
}

std::pair<Loop::Reader, xerrors::Error>
Loop::open_reader(std::vector<PDOEntry> entries) {
    return this->read_buf.open_reader(std::move(entries));
}

std::pair<Loop::Writer, xerrors::Error>
Loop::open_writer(std::vector<PDOEntry> entries) {
    return this->write_buf.open_writer(std::move(entries));
}
}
