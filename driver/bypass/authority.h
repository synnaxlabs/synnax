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
#include <thread>
#include <unordered_map>

#include "client/cpp/synnax.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/control/control.h"
#include "x/cpp/telem/frame.h"

namespace driver::bypass {
/// @brief maintains a local mirror of per-channel authority state by
/// subscribing to the server's control state channel.
class AuthorityMirror {
    mutable std::shared_mutex mu;
    std::unordered_map<synnax::channel::Key, x::control::State> states;
    std::mutex streamer_mu;
    std::unique_ptr<synnax::framer::Streamer> streamer;
    std::shared_ptr<synnax::Synnax> client;
    synnax::channel::Key control_state_key = 0;
    std::thread thread;
    x::breaker::Breaker breaker{x::breaker::Config{
        .name = "authority_mirror",
        .base_interval = x::telem::SECOND,
        .max_retries = 50,
        .scale = 1.2f,
        .max_interval = 30 * x::telem::SECOND,
    }};

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
        this->client = client;
        this->control_state_key = control_state_key;
        this->breaker.start();
        this->thread = std::thread([this] { this->run(); });
        return x::errors::NIL;
    }

    /// @brief stops the update thread.
    void stop() {
        if (!this->breaker.stop()) return;
        {
            std::lock_guard lock(this->streamer_mu);
            if (this->streamer) this->streamer->close_send();
        }
        if (this->thread.joinable()) this->thread.join();
    }

    ~AuthorityMirror() { this->stop(); }

    /// @brief applies a control update directly (for testing without a streamer).
    void apply(const x::control::Update &update) {
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

    /// @brief optimistically applies an authority increase for a single channel.
    /// If the incoming authority is strictly greater than the current holder's,
    /// the mirror is updated immediately. Equal or lower authority is ignored,
    /// matching the server's position-based tiebreak (earlier gate wins ties).
    /// This is safe because the relay will eventually overwrite with the
    /// authoritative state from the server.
    void apply_increase(
        const x::control::Subject &subject,
        synnax::channel::Key channel,
        x::control::Authority authority
    ) {
        std::unique_lock lock(this->mu);
        auto it = this->states.find(channel);
        if (it != this->states.end() && it->second.authority >= authority) return;
        this->states[channel] = x::control::State{
            .resource = channel,
            .subject = subject,
            .authority = authority,
        };
    }

    /// @brief filters a frame by copy, keeping only channels where subject holds
    /// authority or no authority state exists (uncontrolled).
    x::telem::Frame
    filter(const x::telem::Frame &frame, const x::control::Subject &subject) const {
        std::shared_lock lock(this->mu);
        x::telem::Frame out;
        for (auto [key, series]: frame) {
            auto it = this->states.find(key);
            if (it == this->states.end() || it->second.subject == subject)
                out.emplace(key, series.deep_copy());
        }
        return out;
    }

    /// @brief filters a frame by move, keeping only channels where subject holds
    /// authority or no authority state exists (uncontrolled). Takes ownership of
    /// the input frame and moves passing series instead of copying them.
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

    /// @brief checks if subject holds authority on a specific channel.
    bool
    is_authorized(synnax::channel::Key key, const x::control::Subject &subject) const {
        std::shared_lock lock(this->mu);
        auto it = this->states.find(key);
        if (it == this->states.end()) return true;
        return it->second.subject == subject;
    }

private:
    void run() {
        while (this->breaker.running()) {
            auto [frame, err] = this->streamer->read();
            if (err) {
                {
                    std::lock_guard lock(this->streamer_mu);
                    auto close_err = this->streamer->close();
                    if (close_err)
                        LOG(WARNING) << "[authority_mirror] close error: "
                                     << close_err.message();
                }
                if (!this->breaker.running()) break;
                if (!err.matches(freighter::UNREACHABLE) ||
                    !this->breaker.wait(err.message()))
                    break;
                auto [s, reopen_err] = this->client->telem.open_streamer(
                    synnax::framer::StreamerConfig{
                        .channels = {this->control_state_key},
                    }
                );
                if (reopen_err) {
                    if (!this->breaker.wait(reopen_err.message())) break;
                    continue;
                }
                if (!this->breaker.running()) break;
                {
                    std::lock_guard lock(this->streamer_mu);
                    this->streamer = std::make_unique<synnax::framer::Streamer>(
                        std::move(s)
                    );
                }
                LOG(INFO) << "[authority_mirror] reconnected";
                continue;
            }
            this->breaker.reset();
            for (auto [key, series]: frame) {
                if (series.data_type() != x::telem::STRING_T) continue;
                for (const auto &json_str: series.strings()) {
                    x::json::Parser parser(json_str);
                    if (!parser.ok()) continue;
                    auto update = x::control::Update::parse(parser);
                    if (parser.ok()) this->apply(update);
                }
            }
        }
    }
};
}
