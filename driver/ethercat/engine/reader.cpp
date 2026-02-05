// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/ethercat/engine/engine.h"
#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/telem/telem.h"

namespace ethercat::engine {
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
        const size_t required = pdo_required_bytes(pdo.offset.bit, pdo.bit_length);
        if (pdo.offset.byte + required > this->private_buffer.size())
            return xerrors::Error(
                CYCLIC_ERROR,
                "PDO offset out of bounds in input buffer"
            );
        auto &series = frame.series->at(i);
        const uint8_t *src = this->private_buffer.data() + pdo.offset.byte;
        read_pdo_to_series(src, pdo.offset.bit, pdo.bit_length, pdo.data_type, series);
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
}
