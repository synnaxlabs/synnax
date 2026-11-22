// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <atomic>
#include <cstdint>
#include <functional>
#include <mutex>
#include <shared_mutex>
#include <unordered_map>
#include <utility>
#include <vector>

#include "absl/log/log.h"

#include "client/cpp/channel/channel.h"
#include "x/cpp/control/control.h"
#include "x/cpp/telem/frame.h"

namespace driver::control {
/// @brief receives a frame that the mirror lets onto the local bus.
using Publish = std::function<void(const x::telem::Frame &)>;

/// @brief maintains a local mirror of per-channel authority state.
class States {
    mutable std::shared_mutex mu;
    std::unordered_map<synnax::channel::Key, x::control::State<synnax::channel::Key>>
        states;

    /// @brief a frame that waits for the mirror to be read again.
    struct Held {
        x::telem::Frame frame;
        x::control::Subject subject;
        Publish publish;
    };

    /// @brief how route treats a frame.
    enum class Mode : uint8_t {
        /// @brief publishes the channels that the subject may write.
        OPEN,
        /// @brief keeps the frame until the read of the hold lands.
        HOLDING,
        /// @brief drops the frame, since the mirror was never read.
        CLOSED,
    };

    /// @brief read without hold_mu on the fast path of route, and changed only under
    /// hold_mu.
    std::atomic<Mode> mode{Mode::OPEN};
    /// @brief guards held, known, generation, and every change of mode. It is held
    /// while the held frames publish, so a frame that route gets meanwhile comes after
    /// them.
    std::mutex hold_mu;
    /// @brief the frames that route held, in the order they came.
    std::vector<Held> held;
    /// @brief true once a set has replaced the mirror.
    bool known = false;
    /// @brief counts holds, so set can ignore a read that started before a hold.
    uint64_t gen = 0;

    /// @brief publishes each held frame filtered against the mirror, in order, and
    /// clears them. Requires hold_mu, and not mu.
    void release_locked() {
        for (const auto &h: this->held) {
            x::telem::Frame filtered;
            {
                std::shared_lock lock(this->mu);
                filtered = this->filter_locked(h.frame, h.subject);
            }
            if (!filtered.empty()) h.publish(filtered);
        }
        this->held.clear();
    }

    /// @brief returns the channels of frame that subject may write. Requires mu.
    [[nodiscard]] x::telem::Frame filter_locked(
        const x::telem::Frame &frame,
        const x::control::Subject &subject
    ) const {
        x::telem::Frame out;
        for (auto [key, series]: frame) {
            auto it = this->states.find(key);
            if (it == this->states.end() || it->second.subject == subject)
                out.emplace(key, series.shallow_copy());
        }
        return out;
    }

public:
    /// @brief the most frames a hold keeps. A hold that would keep more publishes its
    /// frames filtered against the last mirror and opens the bus, and the read it waits
    /// for still replaces the mirror when it lands.
    static constexpr size_t MAX_HELD = 1024;

    States() = default;

    States(const States &) = delete;
    States &operator=(const States &) = delete;

    /// @brief applies a control update directly (for testing without a streamer).
    void apply(const x::control::Update<synnax::channel::Key> &update) {
        std::unique_lock lock(this->mu);
        for (const auto &transfer: update.transfers) {
            if (transfer.to.has_value()) {
                VLOG(1) << "[bus.authority] channel " << transfer.to->resource
                        << " now held by " << transfer.to->subject.name;
                this->states[transfer.to->resource] = *transfer.to;
            } else if (transfer.from.has_value()) {
                VLOG(1) << "[bus.authority] channel " << transfer.from->resource
                        << " released by " << transfer.from->subject.name;
                this->states.erase(transfer.from->resource);
            }
        }
    }

    /// @returns the generation to pass to set for a read that starts now.
    [[nodiscard]] uint64_t generation() {
        std::lock_guard lock(this->hold_mu);
        return this->gen;
    }

    /// @brief replaces the mirror with the cluster's complete control state, and opens
    /// the bus. Each frame of a hold is filtered against the new mirror and published
    /// in the order it came, before any frame that route gets after this call. A read
    /// that started before a later hold is ignored, so it cannot end that hold.
    /// @param generation the generation that the read started at.
    /// @returns false when the read was ignored.
    bool
    set(const std::vector<x::control::State<synnax::channel::Key>> &next,
        const uint64_t generation) {
        std::lock_guard hold_lock(this->hold_mu);
        if (generation != this->gen) return false;
        {
            std::unique_lock lock(this->mu);
            this->states.clear();
            for (const auto &state: next)
                this->states[state.resource] = state;
        }
        this->known = true;
        this->release_locked();
        this->mode = Mode::OPEN;
        return true;
    }

    /// @brief replaces the mirror and opens the bus, for a read that no hold can
    /// outdate.
    void set(const std::vector<x::control::State<synnax::channel::Key>> &next) {
        this->set(next, this->generation());
    }

    /// @brief keeps every frame that route gets until the read that starts now lands,
    /// because the mirror may be out of date. The frames still go to the Core. A bus
    /// that was never opened stays closed.
    /// @returns the generation to pass to set for that read.
    uint64_t hold() {
        std::lock_guard lock(this->hold_mu);
        ++this->gen;
        if (this->mode == Mode::OPEN) this->mode = Mode::HOLDING;
        return this->gen;
    }

    /// @brief ends a hold whose read failed: publishes the held frames filtered against
    /// the last mirror, and opens the bus with it. A bus that was never opened stays
    /// closed, and its held frames are dropped.
    void release() {
        std::lock_guard lock(this->hold_mu);
        if (!this->known) {
            this->held.clear();
            return;
        }
        this->release_locked();
        this->mode = Mode::OPEN;
    }

    /// @brief closes the bus until the first set. Call it before the first read.
    void close() {
        std::lock_guard lock(this->hold_mu);
        this->held.clear();
        this->mode = Mode::CLOSED;
    }

    /// @brief publishes the channels of frame that subject may write. During a hold it
    /// keeps the frame instead, and before the first set it drops the frame.
    void route(
        const x::telem::Frame &frame,
        const x::control::Subject &subject,
        const Publish &publish
    ) {
        if (this->mode != Mode::OPEN) {
            std::lock_guard lock(this->hold_mu);
            if (this->mode == Mode::HOLDING && this->held.size() >= MAX_HELD) {
                LOG(WARNING) << "[bus.writer] published " << this->held.size()
                             << " held frames with the last control state: the read "
                                "of control state is taking too long";
                this->release_locked();
                this->mode = Mode::OPEN;
            }
            if (this->mode == Mode::HOLDING) {
                x::telem::Frame copy;
                for (auto [key, series]: frame)
                    copy.emplace(key, series.shallow_copy());
                this->held.push_back({std::move(copy), subject, publish});
                return;
            }
            if (this->mode == Mode::CLOSED) {
                LOG_EVERY_N(WARNING, 1000)
                    << "[bus.writer] dropped a frame from the bus for " << subject.name
                    << ": the control state was never read";
                return;
            }
        }
        std::shared_lock lock(this->mu);
        auto filtered = this->filter_locked(frame, subject);
        if (filtered.size() < frame.size())
            LOG_FIRST_N(WARNING, 10)
                << "[bus.writer] dropped " << frame.size() - filtered.size() << " of "
                << frame.size() << " channels from bus for " << subject.name
                << ": no authority. This is normal when another writer holds control.";
        if (!filtered.empty()) publish(filtered);
    }

    /// @brief applies the control updates carried by a series from the control channel.
    void apply(const x::telem::Series &series) {
        if (series.data_type() != x::telem::JSON_T) {
            LOG(ERROR) << "[bus.authority] expected a JSON control state series, got "
                       << series.data_type();
            return;
        }
        for (const auto &json_str: series.strings()) {
            x::json::Parser parser(json_str);
            if (!parser.ok()) continue;
            auto update = x::control::Update<synnax::channel::Key>::parse(parser);
            if (parser.ok()) this->apply(update);
        }
    }

    /// @brief optimistically applies an authority increase for a single channel. If the
    /// incoming authority is strictly greater than the current holder's, the mirror is
    /// updated immediately. Equal or lower authority is ignored, matching the Core's
    /// position-based tiebreak (earlier gate wins ties). This is safe because the relay
    /// will eventually overwrite with the authoritative state from the Core.
    void apply_increase(
        const x::control::Subject &subject,
        synnax::channel::Key channel,
        x::control::Authority authority
    ) {
        std::unique_lock lock(this->mu);
        auto it = this->states.find(channel);
        if (it != this->states.end() && it->second.authority >= authority) return;
        this->states[channel] = x::control::State<synnax::channel::Key>{
            .subject = subject,
            .resource = channel,
            .authority = authority,
        };
    }

    /// @brief filters a frame, keeping only channels where subject holds authority or
    /// no authority state exists (uncontrolled). Series are shallow copied (shared_ptr
    /// refcount bump), not deep-copied.
    x::telem::Frame
    filter(const x::telem::Frame &frame, const x::control::Subject &subject) const {
        std::shared_lock lock(this->mu);
        x::telem::Frame out;
        for (auto [key, series]: frame) {
            auto it = this->states.find(key);
            if (it == this->states.end() || it->second.subject == subject)
                out.emplace(key, series.shallow_copy());
        }
        return out;
    }

    /// @brief filters a frame by move, keeping only channels where subject holds
    /// authority or no authority state exists (uncontrolled). Takes ownership of the
    /// input frame and moves passing series instead of copying them.
    x::telem::Frame
    filter(x::telem::Frame &&frame, const x::control::Subject &subject) const {
        std::shared_lock lock(this->mu);
        if (frame.channels == nullptr || frame.series == nullptr) return {};
        bool all_pass = true;
        for (unsigned int i: *frame.channels) {
            auto it = this->states.find(i);
            if (it != this->states.end() && it->second.subject != subject) {
                all_pass = false;
                break;
            }
        }
        if (all_pass) return std::move(frame);
        x::telem::Frame out;
        for (size_t i = 0; i < frame.channels->size(); i++) {
            auto key = frame.channels->at(i);
            auto it = this->states.find(key);
            if (it == this->states.end() || it->second.subject == subject)
                out.emplace(key, std::move(frame.series->at(i)));
        }
        return out;
    }

    /// @brief returns true if subject holds authority (or no state exists) for every
    /// channel in the frame. Used as a fast path to avoid deep copying when the writer
    /// is fully authorized.
    bool all_authorized(
        const x::telem::Frame &frame,
        const x::control::Subject &subject
    ) const {
        std::shared_lock lock(this->mu);
        for (auto [key, _]: frame) {
            auto it = this->states.find(key);
            if (it != this->states.end() && it->second.subject != subject) return false;
        }
        return true;
    }

    /// @brief checks if subject holds authority on a specific channel.
    bool
    is_authorized(synnax::channel::Key key, const x::control::Subject &subject) const {
        std::shared_lock lock(this->mu);
        auto it = this->states.find(key);
        if (it == this->states.end()) return true;
        return it->second.subject == subject;
    }
};
}
