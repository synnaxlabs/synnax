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

#include "absl/log/log.h"

#include "driver/bypass/bypass.h"
#include "driver/bypass/pipeline/forwarder.h"
#include "driver/control/state.h"
#include "driver/pipeline/acquisition.h"

namespace driver::bypass::pipeline {
/// @brief a pipeline Writer that publishes frames to the local bus, then hands them to
/// a Forwarder that delivers them to the Core in the background. When set_authority is
/// called with an authority higher than the current holder, the increase is applied
/// directly to the control::States before it goes to the Core, eliminating the
/// staleness window for authority increases.
class Writer final : public ::driver::pipeline::Writer {
    std::unique_ptr<Forwarder> forwarder;
    std::shared_ptr<Bus> bus;
    std::shared_ptr<control::States> states;
    synnax::framer::WriterConfig cfg;
    std::shared_ptr<std::atomic<size_t>> carried_drops;

public:
    /// @param carried_drops receives the drop count on close, for a writer that the
    /// acquisition pipeline reopens in the same run. May be null.
    Writer(
        std::unique_ptr<Forwarder> forwarder,
        std::shared_ptr<Bus> bus,
        std::shared_ptr<control::States> states,
        synnax::framer::WriterConfig cfg,
        std::shared_ptr<std::atomic<size_t>> carried_drops = nullptr
    ):
        forwarder(std::move(forwarder)),
        bus(std::move(bus)),
        states(std::move(states)),
        cfg(std::move(cfg)),
        carried_drops(std::move(carried_drops)) {}

    /// @brief publishes fr to the bus and queues it for the Core.
    /// @returns the Core error that stopped delivery, if any.
    [[nodiscard]] x::errors::Error write(const x::telem::Frame &fr) override {
        if (this->states->all_authorized(fr, this->cfg.subject)) {
            VLOG(1) << "[bus.writer] publishing frame with " << fr.size()
                    << " channels to bus";
            this->bus->publish(fr);
        } else {
            auto filtered = this->states->filter(fr, this->cfg.subject);
            LOG_FIRST_N(WARNING, 10)
                << "[bus.writer] dropped " << fr.size() - filtered.size() << " of "
                << fr.size() << " channels from bus for " << this->cfg.subject.name
                << ": no authority. This is normal when another writer holds control.";
            if (!filtered.empty()) {
                VLOG(1) << "[bus.writer] publishing filtered frame with "
                        << filtered.size() << " channels to bus";
                this->bus->publish(filtered);
            }
        }
        return this->forwarder->write(fr);
    }

    /// @brief the bus already has the frame, so it goes to the Core only.
    [[nodiscard]] x::errors::Error rewrite(const x::telem::Frame &fr) override {
        return this->forwarder->write(fr);
    }

    [[nodiscard]] x::errors::Error
    set_authority(const ::driver::pipeline::Authorities &authorities) override {
        if (auto err = authorities.validate()) return err;
        auto keys = authorities.keys;
        if (keys.empty()) keys = this->cfg.channels;
        for (size_t i = 0; i < keys.size(); i++) {
            auto auth = authorities.authorities.size() == 1
                          ? authorities.authorities[0]
                          : authorities.authorities[i];
            this->states->apply_increase(this->cfg.subject, keys[i], auth);
        }
        return this->forwarder->set_authority(authorities);
    }

    [[nodiscard]] x::errors::Error close() override {
        auto err = this->forwarder->close();
        if (this->carried_drops != nullptr)
            this->carried_drops->store(this->forwarder->dropped());
        return err;
    }
};

/// @brief a WriterFactory that wraps writers with bus publish capability. It waits for
/// the first Core open, and returns once it succeeds, fails for good, or finds the
/// Core unreachable; later opens run in the background. Injects the group identity
/// into writer configs for Core-side deduplication and threads control::States into
/// writers for short-circuit authority updates.
class WriterFactory final : public ::driver::pipeline::WriterFactory {
    std::shared_ptr<::driver::pipeline::WriterFactory> server;
    std::shared_ptr<Bus> bus;
    std::shared_ptr<control::States> states;
    std::uint32_t group;
    ForwarderConfig forwarder;
    /// @brief the drop count that a reopened writer of the same run starts from.
    std::shared_ptr<std::atomic<size_t>>
        carried_drops = std::make_shared<std::atomic<size_t>>(0);

public:
    WriterFactory(
        std::shared_ptr<::driver::pipeline::WriterFactory> server,
        const std::shared_ptr<Bus> &bus,
        const std::shared_ptr<control::States> &states,
        const std::uint32_t group,
        ForwarderConfig forwarder = {}
    ):
        server(std::move(server)),
        bus(bus),
        states(states),
        group(group),
        forwarder(std::move(forwarder)) {}

    /// @brief starts the drop count of the run at 0.
    void begin_run() override { this->carried_drops->store(0); }

    std::pair<std::unique_ptr<::driver::pipeline::Writer>, x::errors::Error>
    open_writer(const synnax::framer::WriterConfig &config) override {
        auto cfg = config;
        if (this->group != 0 && cfg.subject.group == 0) cfg.subject.group = this->group;
        this->bus->register_channels(cfg.channels);
        auto forwarder_cfg = this->forwarder;
        forwarder_cfg.dropped = this->carried_drops->load();
        auto forwarder = std::make_unique<Forwarder>(this->server, cfg, forwarder_cfg);
        if (auto err = forwarder->opened()) return {nullptr, err};
        VLOG(1) << "[bus.writer_factory] opened writer for " << cfg.channels.size()
                << " channels, group=" << this->group;
        return {
            std::make_unique<Writer>(
                std::move(forwarder),
                this->bus,
                this->states,
                cfg,
                this->carried_drops
            ),
            x::errors::NIL,
        };
    }
};
}
