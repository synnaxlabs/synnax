// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <shared_mutex>
#include <unordered_map>

#include "client/cpp/synnax.h"
#include "x/cpp/control/control.h"
#include "x/cpp/telem/frame.h"

namespace driver::control {
/// @brief maintains a local mirror of per-channel authority state.
class States {
    mutable std::shared_mutex mu;
    std::unordered_map<synnax::channel::Key, x::control::State<synnax::channel::Key>>
        states;

public:
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

    void apply(const x::telem::Series &series) {
        if (series.data_type() != x::telem::STRING_T) return;
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
        for (size_t i = 0; i < frame.channels->size(); i++) {
            auto it = this->states.find(frame.channels->at(i));
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
