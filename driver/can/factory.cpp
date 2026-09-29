// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <optional>
#include <string>
#include <utility>

#include "client/cpp/can/json.gen.h"

#include "driver/bus/task.h"
#include "driver/can/factory.h"
#include "driver/can/read.h"
#include "driver/can/scan_task.h"
#include "driver/can/write.h"
#include "driver/common/factory.h"
#include "driver/common/status.h"

namespace driver::can {
namespace {
using Result = std::pair<common::ConfigureResult, x::errors::Error>;

Result configure_read(
    const std::shared_ptr<const Backends> &backends,
    const std::shared_ptr<Links> &links,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) {
    x::json::Parser parser(task.config);
    const auto cfg = synnax::can::ReadConfig::parse(parser);
    if (!parser.ok()) return {common::ConfigureResult{}, parser.error()};
    auto [props, props_err] = bus::retrieve_properties<synnax::can::Properties>(
        *ctx->client,
        parser,
        cfg.device
    );
    if (props_err) return {common::ConfigureResult{}, props_err};
    auto [resolved, err] = bus::ReadConfig::parse(
        *ctx->client,
        parser,
        cfg,
        ::synnax::bus::PollConfig{},
        std::nullopt,
        bus::Medium::CAN
    );
    if (err) return {common::ConfigureResult{}, err};
    return {
        common::ConfigureResult{
            .task = std::make_unique<common::ReadTask>(
                task,
                ctx,
                x::breaker::default_config(task.name),
                std::make_unique<Source>(
                    std::move(resolved),
                    acquirer(links, cfg.device, backends, std::move(props))
                )
            ),
            .auto_start = cfg.auto_start,
        },
        x::errors::NIL,
    };
}

/// @brief binds an error for each message whose frames the device's bus cannot send.
void check_frames(
    const x::json::Parser &parser,
    const bus::WriteConfig &cfg,
    const synnax::can::Properties &props
) {
    for (const auto &m: cfg.messages) {
        const auto &id = std::get<synnax::library::CanIdentifier>(*m.entry.identifier);
        if (id.fd && !props.fd) {
            parser.field_err(
                "messages",
                "message " + m.entry.name + " is CAN FD, but the device's bus is not"
            );
            continue;
        }
        const auto length = m.plan.length();
        const auto max = id.fd ? MAX_FD_LENGTH : MAX_CLASSIC_LENGTH;
        if (length > max || !length_to_dlc(static_cast<std::uint8_t>(length)))
            parser.field_err(
                "messages",
                "message " + m.entry.name + " is " + std::to_string(length) +
                    " bytes, which is not a valid " + (id.fd ? "CAN FD" : "CAN") +
                    " frame length"
            );
    }
}

Result configure_write(
    const std::shared_ptr<const Backends> &backends,
    const std::shared_ptr<Links> &links,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) {
    x::json::Parser parser(task.config);
    const auto cfg = synnax::can::WriteConfig::parse(parser);
    if (!parser.ok()) return {common::ConfigureResult{}, parser.error()};
    auto [props, props_err] = bus::retrieve_properties<synnax::can::Properties>(
        *ctx->client,
        parser,
        cfg.device
    );
    if (props_err) return {common::ConfigureResult{}, props_err};
    if (props.listen_only) {
        parser.field_err("device", "device is listen only, so it cannot send");
        return {common::ConfigureResult{}, parser.error()};
    }
    auto [resolved, err] = bus::WriteConfig::parse(
        *ctx->client,
        parser,
        cfg,
        std::nullopt,
        bus::Medium::CAN
    );
    if (err) return {common::ConfigureResult{}, err};
    check_frames(parser, resolved, props);
    if (!parser.ok()) return {common::ConfigureResult{}, parser.error()};
    auto transmitter = std::make_unique<Transmitter>(
        resolved,
        acquirer(links, cfg.device, backends, std::move(props))
    );
    return {
        common::ConfigureResult{
            .task = std::make_unique<common::WriteTask>(
                task,
                ctx,
                x::breaker::default_config(task.name),
                std::make_unique<
                    bus::Sink>(std::move(resolved), std::move(transmitter), ctx, task)
            ),
            .auto_start = cfg.auto_start,
        },
        x::errors::NIL,
    };
}

Result configure_scan(
    const std::shared_ptr<const Backends> &backends,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) {
    x::json::Parser parser(task.config);
    const auto cfg = synnax::can::ScanConfig::parse(parser);
    if (!parser.ok()) return {common::ConfigureResult{}, parser.error()};
    return {
        common::ConfigureResult{
            .task = std::make_unique<common::ScanTask>(
                std::make_unique<Scanner>(task, backends),
                ctx,
                task,
                x::breaker::default_config(task.name),
                cfg.rate
            ),
            .auto_start = !cfg.disabled,
        },
        x::errors::NIL,
    };
}
}

std::pair<std::unique_ptr<task::Task>, bool> Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    const std::string &cmd_key
) {
    if (task.type == READ_TASK_TYPE)
        return common::handle_config_err(
            ctx,
            task,
            configure_read(this->backends, this->links, ctx, task),
            cmd_key
        );
    if (task.type == WRITE_TASK_TYPE)
        return common::handle_config_err(
            ctx,
            task,
            configure_write(this->backends, this->links, ctx, task),
            cmd_key
        );
    if (task.type == SCAN_TASK_TYPE)
        return common::handle_config_err(
            ctx,
            task,
            configure_scan(this->backends, ctx, task),
            cmd_key
        );
    return {nullptr, false};
}

std::vector<std::pair<synnax::task::Task, std::unique_ptr<task::Task>>>
Factory::configure_initial_tasks(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::rack::Rack &rack
) {
    return common::configure_initial_factory_tasks(
        this,
        ctx,
        rack,
        "CAN Scanner",
        SCAN_TASK_TYPE,
        INTEGRATION_NAME
    );
}
}
