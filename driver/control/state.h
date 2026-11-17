// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <mutex>
#include <shared_mutex>
#include <string>
#include <unordered_map>
#include <unordered_set>
#include <vector>

#include "absl/log/log.h"

#include "client/cpp/channel/channel.h"
#include "x/cpp/control/control.h"
#include "x/cpp/telem/frame.h"

namespace driver::control {
/// @brief maintains a local mirror of per-channel authority state. It also keeps a
/// gate for each local writer. For a writer open on a connected Core, the mirror takes
/// authority increases at once and waits for the Core's control updates for the rest.
/// After disconnect, and for a writer not yet open on the Core, it applies the Core's
/// rule over the local gates itself: the highest authority holds, and the earlier gate
/// wins a tie.
class States {
    struct Gate {
        x::control::Subject subject;
        std::unordered_map<synnax::channel::Key, x::control::Authority> authorities;
        std::uint64_t order = 0;
    };

    mutable std::shared_mutex mu;
    std::unordered_map<synnax::channel::Key, x::control::State<synnax::channel::Key>>
        states;
    std::unordered_map<std::string, Gate> gates;
    /// @brief local writers that closed while the Core was connected. The mirror can
    /// still name one as a holder until the Core's transfer arrives.
    std::unordered_set<std::string> closed;
    /// @brief local writers open on the Core since it was last lost.
    std::unordered_set<std::string> seen;
    std::uint64_t next_order = 0;
    bool connected = true;

    [[nodiscard]] bool local(const x::control::Subject &subject) const {
        return !this->connected || !this->seen.contains(subject.key);
    }

    Gate &gate(const x::control::Subject &subject) {
        auto [it, added] = this->gates.try_emplace(subject.key);
        if (added) it->second = Gate{.subject = subject, .order = this->next_order++};
        return it->second;
    }

    /// @brief gives channel to the local gate that the Core's rule picks, or leaves it
    /// uncontrolled when no local gate writes it.
    void recompute(const synnax::channel::Key channel) {
        const Gate *best = nullptr;
        x::control::Authority best_auth = 0;
        for (const auto &[_, g]: this->gates) {
            const auto it = g.authorities.find(channel);
            if (it == g.authorities.end()) continue;
            if (best == nullptr || it->second > best_auth ||
                (it->second == best_auth && g.order < best->order)) {
                best = &g;
                best_auth = it->second;
            }
        }
        if (best == nullptr) {
            this->states.erase(channel);
            return;
        }
        this->states[channel] = x::control::State<synnax::channel::Key>{
            .subject = best->subject,
            .resource = channel,
            .authority = best_auth,
        };
    }

    void set_authority_locked(
        const x::control::Subject &subject,
        const synnax::channel::Key channel,
        const x::control::Authority authority
    ) {
        this->gate(subject).authorities[channel] = authority;
        const auto it = this->states.find(channel);
        if (this->local(subject) && (it == this->states.end() ||
                                     this->gates.contains(it->second.subject.key))) {
            this->recompute(channel);
            return;
        }
        if (it == this->states.end() || it->second.authority < authority) {
            this->states[channel] = x::control::State<synnax::channel::Key>{
                .subject = subject,
                .resource = channel,
                .authority = authority,
            };
            return;
        }
    }

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

    /// @brief marks the Core as lost, so local gates hand channels to each other until
    /// the next set. A channel that a local writer holds, or held before it closed,
    /// goes to the local gate that the Core's rule picks. A holder the Core reported
    /// stays.
    void disconnect() {
        std::unique_lock lock(this->mu);
        this->connected = false;
        std::vector<synnax::channel::Key> stale;
        for (const auto &[channel, state]: this->states)
            if (this->gates.contains(state.subject.key) ||
                this->closed.contains(state.subject.key))
                stale.push_back(channel);
        for (const auto &[_, g]: this->gates)
            for (const auto &[channel, _]: g.authorities)
                if (!this->states.contains(channel)) stale.push_back(channel);
        for (const auto channel: stale)
            this->recompute(channel);
        this->closed.clear();
        this->seen.clear();
    }

    /// @brief replaces the mirror with the cluster's complete control state and marks
    /// the Core as connected. A channel the Core reports as uncontrolled goes to its
    /// local gates, so a local writer that has not reopened on the Core yet keeps its
    /// channels. Such a writer keeps the local rule until opened_on_core.
    void set(const std::vector<x::control::State<synnax::channel::Key>> &next) {
        std::unique_lock lock(this->mu);
        this->connected = true;
        this->closed.clear();
        this->states.clear();
        for (const auto &state: next)
            this->states[state.resource] = state;
        for (const auto &[_, g]: this->gates)
            for (const auto &[channel, _]: g.authorities)
                if (!this->states.contains(channel)) this->recompute(channel);
    }

    /// @brief records that the writer of subject opened on the Core. Until the next
    /// disconnect, its decreases and close wait for the Core's transfers while the Core
    /// is connected. Safe to call before its gate opens.
    void opened_on_core(const x::control::Subject &subject) {
        std::unique_lock lock(this->mu);
        this->seen.insert(subject.key);
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

    /// @brief opens a gate for a local writer with the authorities it opens with, one
    /// for all channels or one per channel. An empty list means absolute authority.
    void open_gate(
        const x::control::Subject &subject,
        const std::vector<synnax::channel::Key> &channels,
        const std::vector<x::control::Authority> &authorities
    ) {
        std::unique_lock lock(this->mu);
        for (size_t i = 0; i < channels.size(); i++) {
            auto auth = x::control::AUTHORITY_ABSOLUTE;
            if (authorities.size() == 1)
                auth = authorities[0];
            else if (i < authorities.size())
                auth = authorities[i];
            this->set_authority_locked(subject, channels[i], auth);
        }
    }

    /// @brief sets the authority of subject on channel. A higher authority than the
    /// holder's takes the channel at once. After disconnect, a channel that a local
    /// gate or nobody holds goes to the local gate that the Core's rule picks.
    void set_authority(
        const x::control::Subject &subject,
        const synnax::channel::Key channel,
        const x::control::Authority authority
    ) {
        std::unique_lock lock(this->mu);
        this->set_authority_locked(subject, channel, authority);
    }

    /// @brief closes the gate of a local writer. After disconnect, or before the writer
    /// opened on the Core, each channel it held goes to the next local gate.
    void close_gate(const x::control::Subject &subject) {
        std::unique_lock lock(this->mu);
        const auto it = this->gates.find(subject.key);
        if (it == this->gates.end()) return;
        const auto channels = it->second.authorities;
        this->gates.erase(it);
        const bool local = this->local(subject);
        this->seen.erase(subject.key);
        if (!local) {
            this->closed.insert(subject.key);
            return;
        }
        for (const auto &[channel, _]: channels) {
            const auto held = this->states.find(channel);
            if (held != this->states.end() && held->second.subject == subject)
                this->recompute(channel);
        }
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
