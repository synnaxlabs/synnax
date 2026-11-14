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
/// @brief how long warning() keeps a drop after the last call that saw one.
const auto DROP_HOLD = x::telem::SECOND * 10;
}

Engine::Reader::Reader(Engine &engine, std::shared_ptr<Input> input):
    engine(engine), input(std::move(input)) {
    size_t width = 0;
    for (const auto &entry: this->input->entries)
        width = std::max(width, entry.decoder.type().density());
    this->values.resize(this->input->samples_per_read * width);
}

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
                errors::sentence(
                    "Every bus cycle failed: " + this->engine.cycle_error().data
                )
            );
        if (std::ranges::any_of(in.slaves, [](const Input::Slave &slave) {
                return slave.lost.load(std::memory_order_relaxed);
            }))
            return x::errors::NIL;
        return x::errors::Error(
            errors::CYCLE_OVERRUN,
            std::format(
                "No inputs from {} within {}.",
                this->engine.network_name,
                timeout.to_string()
            )
        );
    }
    const auto slot = [&](const size_t s) { return (tail + s) % in.capacity; };
    for (size_t i = 0; i < in.entries.size(); ++i) {
        const auto &decoder = in.entries[i].decoder;
        const size_t width = decoder.type().density();
        const uint8_t *entry = in.samples.get() + in.entries[i].sample_offset;
        for (size_t s = 0; s < count; ++s)
            decoder.decode(
                entry + slot(s) * in.sample_size,
                this->values.data() + s * width
            );
        frame.series->at(i).write_casted(this->values.data(), count, decoder.type());
    }
    for (size_t s = 0; s < count; ++s)
        times.write(in.times[slot(s)]);
    in.tail.store(tail + count, std::memory_order_release);
    return x::errors::NIL;
}

std::string Engine::Reader::warning() {
    const auto now = this->engine.config.host_clock.now();
    std::string msg;
    const auto add = [&msg](const std::string &sentence) {
        msg += (msg.empty() ? "" : " ") + sentence;
    };
    const auto hold = [&](Drop &drop, std::atomic<uint64_t> &samples, auto reason) {
        if (samples.exchange(0, std::memory_order_relaxed) > 0) drop = {reason(), now};
        if (!drop.reason.empty() && now - drop.at < DROP_HOLD) add(drop.reason);
    };
    hold(this->failed, this->input->failed_samples, [this] {
        return errors::sentence(
            "Bus cycles failed: " + this->engine.cycle_error().data
        );
    });
    for (auto &slave: this->input->slaves)
        if (slave.lost.exchange(false, std::memory_order_acquire))
            add(slave.silent.load(std::memory_order_relaxed)
                    ? std::format(
                          "Lost contact with slave {} ({}).",
                          slave.position,
                          slave.name
                      )
                    : std::format(
                          "Slave {} ({}) is not in OP.",
                          slave.position,
                          slave.name
                      ));
    hold(this->overflowed, this->input->overflowed_samples, [] {
        return std::string(
            "The task cannot keep up with the bus. Lower the stream rate for the task."
        );
    });
    return msg;
}
}
