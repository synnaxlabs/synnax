// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>

#include "absl/log/check.h"

#include "driver/ethercat/engine/engine.h"
#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/telem/telem.h"

namespace driver::ethercat::engine {
Engine::Reader::Reader(
    Engine &eng,
    const size_t id,
    const size_t total_size,
    std::shared_ptr<Registration> registration
):
    engine(eng),
    id(id),
    total_size(total_size),
    registration(std::move(registration)) {}

void Engine::Reader::refresh_pdos() const {
    std::lock_guard lock(this->engine.registration_mu);
    DCHECK_EQ(this->registration->offsets.size(), this->registration->entries.size());
    this->pdos.clear();
    this->pdos.reserve(this->registration->entries.size());
    for (size_t i = 0; i < this->registration->entries.size(); ++i) {
        const auto &entry = this->registration->entries[i];
        const auto &offset = this->registration->offsets[i];
        auto [plan, err] = telem::plan(entry, offset.bit);
        CHECK(!err) << "open_reader validates every entry: " << err;
        this->pdos.push_back({offset.byte, entry.data_type, std::move(plan)});
    }
    if (!this->pdos.empty()) this->value = this->pdos.front().plan.values();
    this->private_buffer.resize(
        this->engine.shared_input_size.load(std::memory_order_acquire)
    );
    this->my_config_gen = this->engine.config_gen.load(std::memory_order_acquire);
}

Engine::Reader::~Reader() {
    this->engine.unregister_reader(this->id);
}

x::errors::Error Engine::Reader::read(
    const x::breaker::Breaker &brk,
    const x::telem::Frame &frame
) const {
    uint64_t observed_epoch = 0;
    {
        std::unique_lock lock(this->engine.notify_mu);
        const auto timeout = x::telem::MILLISECOND * 200;
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
            return x::errors::Error(
                errors::CYCLE_OVERRUN,
                "timeout waiting for inputs"
            );
    }

    if (this->engine.restarting.load(std::memory_order_acquire))
        return x::errors::Error(errors::ENGINE_RESTARTING, "engine restarting");
    // User commanded stop - not an error
    if (!brk.running()) return x::errors::NIL;
    if (!this->engine.breaker.running())
        return x::errors::Error(errors::CYCLIC_ERROR, "engine stopped unexpectedly");

    this->last_seen_epoch = observed_epoch;

    if (this->engine.config_gen.load(std::memory_order_acquire) != this->my_config_gen)
        this->refresh_pdos();

    uint64_t s0 = 0, s1 = 0;
    do {
        s0 = this->engine.read_seq.seq.load(std::memory_order_acquire);
        if (s0 & 1) continue;
        const uint8_t *ptr = this->engine.shared_input_ptr.load(
            std::memory_order_acquire
        );
        const size_t sz = this->engine.shared_input_size.load(
            std::memory_order_acquire
        );
        const size_t copy_len = std::min(this->private_buffer.size(), sz);
        std::memcpy(this->private_buffer.data(), ptr, copy_len);
        std::atomic_thread_fence(std::memory_order_acquire);
        s1 = this->engine.read_seq.seq.load(std::memory_order_acquire);
    } while (s0 != s1);

    if (this->engine.config_gen.load(std::memory_order_acquire) != this->my_config_gen)
        return x::errors::Error(errors::ENGINE_RESTARTING, "engine restarting");

    if (frame.series->size() < this->pdos.size())
        return x::errors::Error(
            errors::CYCLIC_ERROR,
            "frame has fewer series than registered PDO entries"
        );

    for (size_t i = 0; i < this->pdos.size(); ++i) {
        const auto &pdo = this->pdos[i];
        if (pdo.byte + pdo.plan.length() > this->private_buffer.size())
            return x::errors::Error(
                errors::CYCLIC_ERROR,
                "PDO offset out of bounds in input buffer"
            );
        const auto payload = std::span(this->private_buffer).subspan(pdo.byte);
        if (const auto err = pdo.plan.decode(payload, this->value)) return err;
        this->value.write(0, frame.series->at(i));
    }

    return x::errors::NIL;
}

x::errors::Error Engine::Reader::wait(const x::breaker::Breaker &brk) const {
    uint64_t observed_epoch = 0;
    {
        std::unique_lock lock(this->engine.notify_mu);
        const auto timeout = x::telem::MILLISECOND * 200;
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
            return x::errors::Error(
                errors::CYCLE_OVERRUN,
                "timeout waiting for inputs"
            );
    }

    if (this->engine.restarting.load(std::memory_order_acquire))
        return x::errors::Error(errors::ENGINE_RESTARTING, "engine restarting");
    // User commanded stop - not an error
    if (!brk.running()) return x::errors::NIL;
    if (!this->engine.breaker.running())
        return x::errors::Error(errors::CYCLIC_ERROR, "engine stopped unexpectedly");

    this->last_seen_epoch = observed_epoch;
    return x::errors::NIL;
}
}
