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

#include "driver/bypass/bypass.h"
#include "driver/control/state.h"
#include "driver/pipeline/acquisition.h"

namespace driver::bypass::pipeline {
/// @brief a pipeline Writer that publishes frames to the local bus before forwarding to
/// the Core. When set_authority is called with an authority higher than the current
/// holder, the increase is applied directly to the control::States before forwarding to
/// the Core, eliminating the staleness window for authority increases.
class Writer final : public ::driver::pipeline::Writer {
    std::unique_ptr<::driver::pipeline::Writer> server;
    std::shared_ptr<Bus> bus;
    std::shared_ptr<control::States> states;
    synnax::framer::WriterConfig cfg;

public:
    Writer(
        std::unique_ptr<::driver::pipeline::Writer> server,
        std::shared_ptr<Bus> bus,
        std::shared_ptr<control::States> states,
        synnax::framer::WriterConfig cfg
    ):
        server(std::move(server)),
        bus(std::move(bus)),
        states(std::move(states)),
        cfg(std::move(cfg)) {}

    [[nodiscard]] x::errors::Error write(const x::telem::Frame &fr) override {
        if (this->states->all_authorized(fr, this->cfg.subject)) {
            VLOG(1) << "[bus.writer] publishing frame with " << fr.size()
                    << " channels to bus";
            this->bus->publish(fr);
        } else {
            auto filtered = this->states->filter(fr, this->cfg.subject);
            if (!filtered.empty()) {
                VLOG(1) << "[bus.writer] publishing filtered frame with "
                        << filtered.size() << " channels to bus";
                this->bus->publish(filtered);
            }
        }
        return this->server->write(fr);
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
        return this->server->set_authority(authorities);
    }

    [[nodiscard]] x::errors::Error close() override { return this->server->close(); }
};

/// @brief a WriterFactory that wraps writers with bus publish capability. Injects the
/// group identity into writer configs for Core-side deduplication and threads
/// control::States into writers for short-circuit authority updates.
class WriterFactory final : public ::driver::pipeline::WriterFactory {
    std::shared_ptr<::driver::pipeline::WriterFactory> server;
    std::shared_ptr<Bus> bus;
    std::shared_ptr<control::States> states;
    std::uint32_t group;

public:
    WriterFactory(
        std::shared_ptr<::driver::pipeline::WriterFactory> server,
        const std::shared_ptr<Bus> &bus,
        const std::shared_ptr<control::States> &states,
        const std::uint32_t group
    ):
        server(std::move(server)), bus(bus), states(states), group(group) {}

    std::pair<std::unique_ptr<::driver::pipeline::Writer>, x::errors::Error>
    open_writer(const synnax::framer::WriterConfig &config) override {
        auto cfg = config;
        if (this->group != 0 && cfg.subject.group == 0) cfg.subject.group = this->group;
        this->bus->register_channels(cfg.channels);
        auto [writer, err] = this->server->open_writer(cfg);
        if (err) return {nullptr, err};
        VLOG(1) << "[bus.writer_factory] opened writer for " << cfg.channels.size()
                << " channels, group=" << this->group;
        return {
            std::make_unique<Writer>(std::move(writer), this->bus, this->states, cfg),
            x::errors::NIL,
        };
    }
};
}
