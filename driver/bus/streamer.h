// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>

#include "driver/bus/authority.h"
#include "driver/bus/bus.h"
#include "driver/pipeline/control.h"

namespace driver::bus {
/// @brief a pipeline Streamer that merges local bus frames with server frames,
/// filtering by authority.
class Streamer final : public pipeline::Streamer {
    std::unique_ptr<pipeline::Streamer> server;
    std::unique_ptr<Subscription> subscription;
    AuthorityMirror &authority;
    x::control::Subject subject;

public:
    Streamer(
        std::unique_ptr<pipeline::Streamer> server,
        std::unique_ptr<Subscription> subscription,
        AuthorityMirror &authority,
        x::control::Subject subject
    ): server(std::move(server)),
       subscription(std::move(subscription)),
       authority(authority),
       subject(std::move(subject)) {}

    std::pair<x::telem::Frame, x::errors::Error> read() override {
        x::telem::Frame local;
        if (this->subscription->try_pop(local)) {
            auto filtered = this->authority.filter(local, this->subject);
            if (!filtered.empty()) {
                VLOG(1) << "[bus.streamer] delivering local frame with "
                        << filtered.size() << " channels (bypassed server)";
                return {std::move(filtered), x::errors::NIL};
            }
            VLOG(1) << "[bus.streamer] local frame filtered out by authority";
        }
        return this->server->read();
    }

    x::errors::Error close() override { return this->server->close(); }

    void close_send() override {
        this->subscription->close();
        this->server->close_send();
    }
};

/// @brief a StreamerFactory that wraps streamers with bus subscription capability.
class StreamerFactory final : public pipeline::StreamerFactory {
    std::shared_ptr<pipeline::StreamerFactory> server;
    Bus &bus;
    AuthorityMirror &authority;
    x::control::Subject subject;

public:
    StreamerFactory(
        std::shared_ptr<pipeline::StreamerFactory> server,
        Bus &bus,
        AuthorityMirror &authority,
        x::control::Subject subject
    ): server(std::move(server)),
       bus(bus),
       authority(authority),
       subject(std::move(subject)) {}

    std::pair<std::unique_ptr<pipeline::Streamer>, x::errors::Error>
    open_streamer(synnax::framer::StreamerConfig config) override {
        auto [streamer, err] = this->server->open_streamer(config);
        if (err) return {nullptr, err};
        auto subscription = this->bus.subscribe(config.channels);
        VLOG(1) << "[bus.streamer_factory] opened streamer for "
                << config.channels.size() << " channels, subject="
                << this->subject.name;
        return {
            std::make_unique<Streamer>(
                std::move(streamer),
                std::move(subscription),
                this->authority,
                this->subject
            ),
            x::errors::NIL,
        };
    }
};
}
