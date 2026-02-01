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
    LOG(INFO) << "[ethercat] engine started on " << this->master->interface_name();
    const auto cycle_time = telem::TimeSpan(
        this->cycle_time_ns.load(std::memory_order_acquire)
    );
    xthread::RTConfig rt_cfg = this->config.rt;
    if (rt_cfg.enabled && !rt_cfg.has_timing()) {
        rt_cfg.period = cycle_time;
        rt_cfg.computation = cycle_time * 0.2;
        rt_cfg.deadline = cycle_time * 0.8;
        rt_cfg.prefer_deadline_scheduler = true;
    }
    xthread::apply_rt_config(rt_cfg);
    loop::Timer timer(cycle_time);

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

xerrors::Error Engine::reconfigure() {
    if (this->breaker.running()) {
        LOG(INFO) << "[ethercat] restarting engine " + this->master->interface_name() +
                         " for reconfiguration";
        this->restarting = true;
        this->read_cv.notify_all();
        this->breaker.stop();
        if (this->run_thread.joinable()) this->run_thread.join();
        this->master->deactivate();
    }
    std::vector<PDOEntry> all_entries;
    {
        std::scoped_lock lk(this->registration_mu, this->write_mu);
        for (const auto &reg: this->read_registrations)
            all_entries
                .insert(all_entries.end(), reg.entries.begin(), reg.entries.end());
        for (const auto &reg: this->write_registrations)
            all_entries
                .insert(all_entries.end(), reg.entries.begin(), reg.entries.end());
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

void Engine::update_cycle_time() {
    telem::Rate max_rate(0);
    {
        std::lock_guard lock(this->registration_mu);
        for (const auto &reg: this->read_registrations)
            if (reg.rate > max_rate) max_rate = reg.rate;
    }
    {
        std::lock_guard lock(this->write_mu);
        for (const auto &reg: this->write_registrations)
            if (reg.rate > max_rate) max_rate = reg.rate;
    }
    if (max_rate.hz() > 0)
        this->cycle_time_ns.store(
            max_rate.period().nanoseconds(),
            std::memory_order_release
        );
}

telem::Rate Engine::cycle_rate() const {
    const auto ns = this->cycle_time_ns.load(std::memory_order_acquire);
    return telem::Rate(1e9 / static_cast<double>(ns));
}

void Engine::publish_inputs(const std::span<const uint8_t> src) {
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
    this->update_cycle_time();
    if (!this->should_be_running()) this->stop();
}

void Engine::unregister_writer(const size_t id) {
    {
        std::lock_guard lock(this->write_mu);
        std::erase_if(this->write_registrations, [id](const Registration &r) {
            return r.id == id;
        });
    }
    this->update_cycle_time();
    if (!this->should_be_running()) this->stop();
}

Engine::Reader::Reader(
    Engine &eng,
    const size_t id,
    const size_t total_size,
    std::vector<ResolvedPDO> pdos,
    const size_t input_frame_size
):
    engine(eng),
    id(id),
    total_size(total_size),
    pdos(std::move(pdos)),
    private_buffer(input_frame_size) {}

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
    // User commanded stop - not an error
    if (!brk.running()) return xerrors::NIL;
    if (!this->engine.breaker.running())
        return xerrors::Error(CYCLIC_ERROR, "engine stopped unexpectedly");

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

    if (frame.series->size() < this->pdos.size())
        return xerrors::Error(
            CYCLIC_ERROR,
            "frame has fewer series than registered PDO entries"
        );

    for (size_t i = 0; i < this->pdos.size(); ++i) {
        const auto &pdo = this->pdos[i];
        const size_t byte_len = (pdo.bit_length + 7) / 8;

        if (pdo.offset.byte + byte_len > this->private_buffer.size())
            return xerrors::Error(
                CYCLIC_ERROR,
                "PDO offset out of bounds in input buffer"
            );

        auto &series = frame.series->at(i);
        const uint8_t *src = this->private_buffer.data() + pdo.offset.byte;

        if (pdo.bit_length < 8) {
            // Handle sub-byte values that may span byte boundaries
            uint16_t two_bytes = src[0];
            if (pdo.offset.bit + pdo.bit_length > 8)
                two_bytes |= static_cast<uint16_t>(src[1]) << 8;
            const uint8_t mask = static_cast<uint8_t>((1u << pdo.bit_length) - 1);
            const uint8_t extracted = static_cast<uint8_t>(
                (two_bytes >> pdo.offset.bit) & mask
            );
            series.write_casted(&extracted, 1, telem::UINT8_T);
        } else if (pdo.bit_length == 24) {
            // Handle 24-bit values with potential bit offset
            uint32_t raw = static_cast<uint32_t>(src[0]) |
                           (static_cast<uint32_t>(src[1]) << 8) |
                           (static_cast<uint32_t>(src[2]) << 16);
            if (pdo.offset.bit > 0)
                raw = (raw >> pdo.offset.bit) |
                      (static_cast<uint32_t>(src[3]) << (24 - pdo.offset.bit));
            uint32_t val = raw & 0x00FFFFFF;
            if (pdo.data_type == telem::INT32_T || pdo.data_type == telem::INT64_T)
                if (val & 0x800000) val |= 0xFF000000;
            series.write_casted(&val, 1, pdo.data_type);
        } else {
            telem::DataType source_type = pdo.data_type;
            if (source_type == telem::UNKNOWN_T) source_type = series.data_type();
            series.write_casted(src, 1, source_type);
        }
    }

    return xerrors::NIL;
}

xerrors::Error Engine::Reader::wait(const breaker::Breaker &brk) const {
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
    // User commanded stop - not an error
    if (!brk.running()) return xerrors::NIL;
    if (!this->engine.breaker.running())
        return xerrors::Error(CYCLIC_ERROR, "engine stopped unexpectedly");

    this->last_seen_epoch = observed_epoch;
    return xerrors::NIL;
}

Engine::Writer::Writer(Engine &eng, const size_t id, std::vector<ResolvedPDO> pdos):
    engine(eng), id(id), pdos(std::move(pdos)) {}

Engine::Writer::~Writer() {
    this->engine.unregister_writer(this->id);
}

Engine::Writer::Transaction::Transaction(
    Engine &eng,
    const std::vector<ResolvedPDO> &pdos
):
    engine(eng), pdos(pdos), lock(eng.write_mu) {}

void Engine::Writer::Transaction::write(
    const size_t pdo_index,
    const telem::SampleValue &value
) const {
    if (pdo_index >= this->pdos.size()) return;
    const auto &pdo = this->pdos[pdo_index];
    const size_t byte_len = (pdo.bit_length + 7) / 8;

    size_t required_bytes = byte_len;
    if (pdo.bit_length == 24 && pdo.offset.bit > 0)
        required_bytes = 4;
    else if (pdo.bit_length < 8 && pdo.offset.bit + pdo.bit_length > 8)
        required_bytes = 2;

    if (pdo.offset.byte + required_bytes > this->engine.write_staging.size()) return;

    const auto casted = pdo.data_type == telem::UNKNOWN_T ? value
                                                          : pdo.data_type.cast(value);
    uint8_t *dest = this->engine.write_staging.data() + pdo.offset.byte;

    if (pdo.bit_length < 8) {
        const auto src_val = telem::cast<uint8_t>(casted);
        const uint16_t mask = static_cast<uint16_t>((1u << pdo.bit_length) - 1);

        if (pdo.offset.bit + pdo.bit_length > 8) {
            uint16_t two_bytes = static_cast<uint16_t>(dest[0]) |
                                 (static_cast<uint16_t>(dest[1]) << 8);
            const uint16_t shifted_mask = static_cast<uint16_t>(mask << pdo.offset.bit);
            const uint16_t shifted_val = static_cast<uint16_t>(
                (src_val & mask) << pdo.offset.bit
            );
            two_bytes = static_cast<uint16_t>(
                (two_bytes & ~shifted_mask) | shifted_val
            );
            dest[0] = static_cast<uint8_t>(two_bytes & 0xFF);
            dest[1] = static_cast<uint8_t>((two_bytes >> 8) & 0xFF);
        } else {
            const uint8_t shifted_mask = static_cast<uint8_t>(mask << pdo.offset.bit);
            const uint8_t shifted_val = static_cast<uint8_t>(
                (src_val & mask) << pdo.offset.bit
            );
            dest[0] = static_cast<uint8_t>((dest[0] & ~shifted_mask) | shifted_val);
        }
    } else if (pdo.bit_length == 24) {
        const uint32_t src_val = telem::cast<uint32_t>(casted);
        const uint32_t masked_val = src_val & 0x00FFFFFF;

        if (pdo.offset.bit > 0) {
            uint32_t four_bytes = static_cast<uint32_t>(dest[0]) |
                                  (static_cast<uint32_t>(dest[1]) << 8) |
                                  (static_cast<uint32_t>(dest[2]) << 16) |
                                  (static_cast<uint32_t>(dest[3]) << 24);
            const uint32_t write_mask = 0x00FFFFFFu << pdo.offset.bit;
            const uint32_t shifted_val = masked_val << pdo.offset.bit;
            four_bytes = (four_bytes & ~write_mask) | shifted_val;
            dest[0] = static_cast<uint8_t>(four_bytes & 0xFF);
            dest[1] = static_cast<uint8_t>((four_bytes >> 8) & 0xFF);
            dest[2] = static_cast<uint8_t>((four_bytes >> 16) & 0xFF);
            dest[3] = static_cast<uint8_t>((four_bytes >> 24) & 0xFF);
        } else {
            dest[0] = static_cast<uint8_t>(masked_val & 0xFF);
            dest[1] = static_cast<uint8_t>((masked_val >> 8) & 0xFF);
            dest[2] = static_cast<uint8_t>((masked_val >> 16) & 0xFF);
        }
    } else {
        const void *data = telem::cast_to_void_ptr(casted);
        std::memcpy(dest, data, byte_len);
    }
}

Engine::Writer::Transaction Engine::Writer::open_tx() const {
    return Transaction(this->engine, this->pdos);
}

void Engine::Writer::write(
    const size_t pdo_index,
    const telem::SampleValue &value
) const {
    this->open_tx().write(pdo_index, value);
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

Engine::Engine(std::shared_ptr<master::Master> master):
    Engine(std::move(master), Config{}) {}

Engine::~Engine() {
    this->stop();
}

std::pair<std::unique_ptr<Engine::Reader>, xerrors::Error> Engine::open_reader(
    const std::vector<PDOEntry> &entries,
    const telem::Rate sample_rate
) {
    size_t total_size = 0;
    for (const auto &e: entries)
        total_size += e.byte_length();

    const size_t reg_id = this->next_id.fetch_add(1, std::memory_order_relaxed);
    {
        std::lock_guard lock(this->registration_mu);
        this->read_registrations.push_back({reg_id, entries, {}, sample_rate});
    }
    this->update_cycle_time();

    if (auto err = this->reconfigure(); err) {
        std::lock_guard lock(this->registration_mu);
        this->read_registrations.pop_back();
        return {nullptr, err};
    }

    std::vector<ResolvedPDO> resolved_pdos;
    size_t input_frame_size;
    {
        std::lock_guard lock(this->registration_mu);
        input_frame_size = this->shared_input_buffer.size();
        for (const auto &reg: this->read_registrations) {
            if (reg.id == reg_id) {
                resolved_pdos.reserve(reg.entries.size());
                for (size_t i = 0; i < reg.entries.size(); ++i)
                    resolved_pdos.push_back(
                        {reg.offsets[i],
                         reg.entries[i].data_type,
                         reg.entries[i].bit_length}
                    );
                break;
            }
        }
    }

    return {
        std::make_unique<Reader>(
            *this,
            reg_id,
            total_size,
            std::move(resolved_pdos),
            input_frame_size
        ),
        xerrors::NIL
    };
}

std::pair<std::unique_ptr<Engine::Writer>, xerrors::Error> Engine::open_writer(
    const std::vector<PDOEntry> &entries,
    const telem::Rate execution_rate
) {
    const size_t reg_id = this->next_id.fetch_add(1, std::memory_order_relaxed);
    {
        std::lock_guard lock(this->write_mu);
        this->write_registrations.push_back({reg_id, entries, {}, execution_rate});
    }
    this->update_cycle_time();

    if (auto err = this->reconfigure(); err) {
        std::lock_guard lock(this->write_mu);
        this->write_registrations.pop_back();
        return {nullptr, err};
    }

    std::vector<ResolvedPDO> resolved_pdos;
    {
        std::lock_guard lock(this->write_mu);
        for (const auto &reg: this->write_registrations) {
            if (reg.id == reg_id) {
                resolved_pdos.reserve(reg.entries.size());
                for (size_t i = 0; i < reg.entries.size(); ++i)
                    resolved_pdos.push_back(
                        {reg.offsets[i],
                         reg.entries[i].data_type,
                         reg.entries[i].bit_length}
                    );
                break;
            }
        }
    }

    return {
        std::make_unique<Writer>(*this, reg_id, std::move(resolved_pdos)),
        xerrors::NIL
    };
}
}
