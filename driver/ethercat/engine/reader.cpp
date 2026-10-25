// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <chrono>
#include <format>

#include "driver/ethercat/engine/engine.h"
#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/telem/telem.h"

namespace driver::ethercat::engine {
namespace {
/// @brief the longest the reader sleeps before it checks its breaker again.
const auto WAIT_SLICE = x::telem::MILLISECOND * 100;
/// @brief slack added to the read timeout for scheduling jitter.
const auto READ_TIMEOUT_SLACK = x::telem::MILLISECOND * 200;
}

Engine::Reader::Reader(Engine &engine, std::shared_ptr<Input> input):
    engine(engine), input(std::move(input)) {}

Engine::Reader::~Reader() {
    this->engine.close_input(this->input);
}

x::errors::Error Engine::Reader::read(
    const x::breaker::Breaker &brk,
    x::telem::Frame &frame,
    x::telem::Series &times
) {
    auto &in = *this->input;
    const size_t n = in.samples_per_read;
    const auto timeout = in.rate.period() * static_cast<double>(n * 2) +
                         READ_TIMEOUT_SLACK;
    const auto deadline = std::chrono::steady_clock::now() + timeout.chrono();
    const auto available = [&] {
        return in.head.load(std::memory_order_seq_cst) -
               in.tail.load(std::memory_order_relaxed);
    };
    {
        std::unique_lock lock(in.mu);
        in.waiting.store(true, std::memory_order_seq_cst);
        while (available() < n && brk.running()) {
            const auto now = std::chrono::steady_clock::now();
            if (now >= deadline) break;
            in.cv.wait_for(
                lock,
                std::min<std::chrono::nanoseconds>(deadline - now, WAIT_SLICE.chrono())
            );
        }
        in.waiting.store(false, std::memory_order_relaxed);
    }
    if (!brk.running()) return x::errors::NIL;

    const uint64_t tail = in.tail.load(std::memory_order_relaxed);
    const size_t count = std::min<uint64_t>(
        n,
        in.head.load(std::memory_order_acquire) - tail
    );
    if (count == 0) {
        if (in.failed_samples.exchange(0, std::memory_order_relaxed) > 0)
            return x::errors::Error(
                errors::CYCLE_FAILED,
                "every bus cycle failed: " + this->engine.cycle_error().data
            );
        return x::errors::Error(
            errors::CYCLE_OVERRUN,
            std::format(
                "no inputs from {} within {}",
                this->engine.interface_name(),
                timeout.to_string()
            )
        );
    }
    for (size_t s = 0; s < count; ++s) {
        const size_t slot = (tail + s) % in.capacity;
        const uint8_t *sample = in.samples.get() + slot * in.sample_size;
        for (size_t i = 0; i < in.entries.size(); ++i)
            telem::read_pdo_to_series(
                sample + in.sample_offsets[i],
                in.offsets[i].bit,
                in.entries[i].bit_length,
                in.entries[i].data_type,
                frame.series->at(i)
            );
        times.write(in.times[slot]);
    }
    in.tail.store(tail + count, std::memory_order_release);
    return x::errors::NIL;
}

std::string Engine::Reader::warning() {
    std::string msg;
    if (this->input->failed_samples.exchange(0, std::memory_order_relaxed) > 0)
        msg = "Bus cycles failed: " + this->engine.cycle_error().data + ".";
    if (this->input->overflowed_samples.exchange(0, std::memory_order_relaxed) > 0)
        msg += std::string(msg.empty() ? "" : " ") +
               "The task cannot keep up with the bus. Lower the stream rate for the "
               "task.";
    return msg;
}
}
