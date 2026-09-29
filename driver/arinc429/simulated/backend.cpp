// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <vector>

#include "driver/arinc429/simulated/backend.h"
#include "driver/errors/errors.h"

namespace driver::arinc429::simulated {
namespace {
using Receiver = bus::Queue<Received>;
}

struct Backend::Line {
    /// @brief mu guards receivers and transmitting.
    std::mutex mu;
    /// @brief receivers holds the queue of each open receiver.
    std::vector<std::shared_ptr<Receiver>> receivers;
    /// @brief transmitting is true while a transmitter is open.
    bool transmitting = false;
};

namespace {
/// @brief Channel is one open end of a simulated wire.
class Channel final : public arinc429::Channel {
    std::shared_ptr<Backend::Line> line;
    Direction direction;
    std::shared_ptr<Receiver> queue;

public:
    /// @brief opens an end of the line. The caller marks the line as transmitting
    /// before it opens a transmitter.
    Channel(std::shared_ptr<Backend::Line> line, const Direction direction):
        line(std::move(line)), direction(direction) {
        if (direction == Direction::TRANSMIT) return;
        std::lock_guard lock(this->line->mu);
        this->queue = std::make_shared<Receiver>(RECEIVE_CAPACITY);
        this->line->receivers.push_back(this->queue);
    }

    ~Channel() override {
        std::lock_guard lock(this->line->mu);
        if (this->direction == Direction::TRANSMIT) {
            this->line->transmitting = false;
            return;
        }
        auto &r = this->line->receivers;
        r.erase(std::remove(r.begin(), r.end(), this->queue), r.end());
    }

    std::pair<bus::Batch, x::errors::Error>
    read(const std::span<Received> out, const x::telem::TimeSpan timeout) override {
        if (this->queue == nullptr)
            return {
                {},
                x::errors::Error(
                    errors::CRITICAL_HARDWARE_ERROR,
                    "cannot read from a transmit channel"
                ),
            };
        return {this->queue->pop(out, timeout), x::errors::NIL};
    }

    x::errors::Error
    write(const std::span<const codec::arinc429::Word> words) override {
        if (this->direction != Direction::TRANSMIT)
            return x::errors::Error(
                errors::CRITICAL_HARDWARE_ERROR,
                "cannot write to a receive channel"
            );
        std::lock_guard lock(this->line->mu);
        for (const auto &w: words) {
            const Received r{.word = w, .time = x::telem::TimeStamp::now()};
            for (const auto &q: this->line->receivers)
                q->push(r);
        }
        return x::errors::NIL;
    }
};
}

std::pair<std::unique_ptr<arinc429::Channel>, x::errors::Error>
Backend::open(const synnax::arinc429::Properties &props, const Direction direction) {
    std::shared_ptr<Line> line;
    {
        std::lock_guard lock(this->mu);
        auto &l = this->lines[{props.card, props.channel}];
        if (l == nullptr) l = std::make_shared<Line>();
        line = l;
    }
    if (direction == Direction::TRANSMIT) {
        std::lock_guard lock(line->mu);
        if (line->transmitting)
            return {
                nullptr,
                x::errors::Error(
                    errors::CONFIGURATION_ERROR,
                    "simulated card " + std::to_string(props.card) + " channel " +
                        std::to_string(props.channel) + " already has a transmitter"
                ),
            };
        line->transmitting = true;
    }
    return {std::make_unique<Channel>(std::move(line), direction), x::errors::NIL};
}

std::pair<std::vector<Info>, x::errors::Error> Backend::list() {
    return {
        {Info{.card = 0, .channel = 0, .name = "Simulated ARINC 429"}},
        x::errors::NIL,
    };
}
}
