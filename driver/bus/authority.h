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
#include <shared_mutex>
#include <thread>
#include <unordered_map>

#include "client/cpp/synnax.h"
#include "x/cpp/control/control.h"
#include "x/cpp/telem/frame.h"

namespace driver::bus {
/// @brief maintains a local mirror of per-channel authority state by
/// subscribing to the server's control state channel.
class AuthorityMirror {
    mutable std::shared_mutex mu;
    std::unordered_map<synnax::channel::Key, x::control::State> states;
    std::unique_ptr<synnax::framer::Streamer> streamer;
    std::thread thread;
    std::atomic<bool> running{false};

public:
    AuthorityMirror() = default;

    AuthorityMirror(const AuthorityMirror &) = delete;
    AuthorityMirror &operator=(const AuthorityMirror &) = delete;

    /// @brief starts subscribing to control state updates.
    x::errors::Error start(
        const std::shared_ptr<synnax::Synnax> &client,
        synnax::channel::Key control_state_key
    ) {
        auto [streamer, err] = client->telem.open_streamer(
            synnax::framer::StreamerConfig{.channels = {control_state_key}}
        );
        if (err) return err;
        this->streamer = std::make_unique<synnax::framer::Streamer>(
            std::move(streamer)
        );
        this->running.store(true);
        this->thread = std::thread([this] { this->run(); });
        return x::errors::NIL;
    }

    /// @brief stops the update thread.
    void stop() {
        if (!this->running.exchange(false)) return;
        this->streamer->close_send();
        if (this->thread.joinable()) this->thread.join();
    }

    ~AuthorityMirror() { this->stop(); }

    /// @brief applies a control update directly (for testing without a streamer).
    void apply(const x::control::Update &update) {
        std::unique_lock lock(this->mu);
        for (const auto &transfer : update.transfers) {
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

    /// @brief filters a frame, keeping only channels where subject holds
    /// authority or no authority state exists (uncontrolled).
    x::telem::Frame filter(
        const x::telem::Frame &frame,
        const x::control::Subject &subject
    ) const {
        std::shared_lock lock(this->mu);
        x::telem::Frame out;
        for (auto [key, series] : frame) {
            auto it = this->states.find(key);
            if (it == this->states.end() || it->second.subject == subject)
                out.emplace(key, series.deep_copy());
        }
        return out;
    }

    /// @brief checks if subject holds authority on a specific channel.
    bool is_authorized(
        synnax::channel::Key key,
        const x::control::Subject &subject
    ) const {
        std::shared_lock lock(this->mu);
        auto it = this->states.find(key);
        if (it == this->states.end()) return true;
        return it->second.subject == subject;
    }

private:
    void run() {
        while (this->running.load()) {
            auto [frame, err] = this->streamer->read();
            if (err) break;
            for (auto [key, series] : frame) {
                if (series.data_type() != x::telem::STRING_T) continue;
                auto json_str = series.at<std::string>(0);
                x::json::Parser parser(json_str);
                if (!parser.ok()) continue;
                auto update = x::control::Update::parse(parser);
                if (parser.ok()) this->apply(update);
            }
        }
    }
};
}
