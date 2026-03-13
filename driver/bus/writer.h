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
#include <memory>

#include "driver/bus/bus.h"
#include "driver/pipeline/acquisition.h"

namespace driver::bus {
/// @brief a pipeline Writer that publishes frames to the local bus before
/// forwarding to the server.
class Writer final : public pipeline::Writer {
    std::unique_ptr<pipeline::Writer> server;
    Bus &bus;
    std::atomic<bool> has_routes;

public:
    Writer(std::unique_ptr<pipeline::Writer> server, Bus &bus, const bool has_routes):
        server(std::move(server)), bus(bus), has_routes(has_routes) {}

    [[nodiscard]] x::errors::Error write(const x::telem::Frame &fr) override {
        if (this->has_routes.load(std::memory_order_relaxed)) {
            VLOG(1) << "[bus.writer] publishing frame with " << fr.size()
                    << " channels to bus";
            this->bus.publish(fr);
        }
        return this->server->write(fr);
    }

    [[nodiscard]] x::errors::Error
    set_authority(const pipeline::Authorities &authorities) override {
        return this->server->set_authority(authorities);
    }

    [[nodiscard]] x::errors::Error close() override { return this->server->close(); }
};

/// @brief a WriterFactory that wraps writers with bus publish capability.
/// Injects the group identity into writer configs for server-side deduplication.
class WriterFactory final : public pipeline::WriterFactory {
    std::shared_ptr<pipeline::WriterFactory> server;
    Bus &bus;
    std::uint32_t group;

public:
    WriterFactory(
        std::shared_ptr<pipeline::WriterFactory> server,
        Bus &bus,
        std::uint32_t group
    ):
        server(std::move(server)), bus(bus), group(group) {}

    std::pair<std::unique_ptr<pipeline::Writer>, x::errors::Error>
    open_writer(const synnax::framer::WriterConfig &config) override {
        auto cfg = config;
        if (this->group != 0 && cfg.subject.group == 0) cfg.subject.group = this->group;
        auto [writer, err] = this->server->open_writer(cfg);
        if (err) return {nullptr, err};
        auto has_routes = this->bus.has_subscribers(cfg.channels);
        VLOG(1) << "[bus.writer_factory] opened writer for " << cfg.channels.size()
                << " channels, has_routes=" << has_routes << ", group=" << this->group;
        return {
            std::make_unique<Writer>(std::move(writer), this->bus, has_routes),
            x::errors::NIL,
        };
    }
};
}
