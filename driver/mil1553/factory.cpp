// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "client/cpp/library/types.gen.h"
#include "client/cpp/mil1553/json.gen.h"
#include "x/cpp/breaker/breaker.h"

#include "driver/bus/scan.h"
#include "driver/bus/task.h"
#include "driver/common/factory.h"
#include "driver/common/scan_task.h"
#include "driver/common/status.h"
#include "driver/common/write_task.h"
#include "driver/mil1553/ballard/backend.h"
#include "driver/mil1553/ddc/backend.h"
#include "driver/mil1553/mil1553.h"
#include "driver/mil1553/read.h"
#include "driver/mil1553/simulated/backend.h"
#include "driver/mil1553/write.h"

namespace driver::mil1553 {
Backends create_backends() {
    return {
        {synnax::mil1553::BACKEND_SIMULATED, std::make_shared<simulated::Backend>()},
        {synnax::mil1553::BACKEND_DDC, std::make_shared<ddc::Backend>()},
        {synnax::mil1553::BACKEND_BALLARD, std::make_shared<ballard::Backend>()},
    };
}

std::string address(const codec::mil1553::Command &command) {
    return "terminal " + std::to_string(command.rt) + " subaddress " +
           std::to_string(command.subaddress);
}

namespace {
x::errors::Error
invalid(const synnax::library::MessageEntry &message, const std::string &why) {
    return x::errors::Error(
        x::errors::VALIDATION,
        "message " + message.name + " " + why
    );
}

bool owns(const synnax::mil1553::Properties &props, const std::uint8_t rt) {
    return std::ranges::find(props.terminals, rt) != props.terminals.end();
}

x::errors::Error check_direction(
    const synnax::library::MessageEntry &message,
    const synnax::library::Mil1553Identifier &id,
    const std::string &direction,
    const std::string &role
) {
    if (id.direction == direction) return x::errors::NIL;
    return invalid(
        message,
        "must be a " + direction + " message for a " + role + " to " +
            (direction == synnax::library::DIRECTION_TRANSMIT ? "read" : "send")
    );
}

x::errors::Error check_owned(
    const synnax::library::MessageEntry &message,
    const synnax::library::Mil1553Identifier &id,
    const synnax::mil1553::Properties &props
) {
    if (owns(props, id.rt)) return x::errors::NIL;
    return invalid(
        message,
        "is for terminal " + std::to_string(id.rt) +
            ", which the remote terminal does not own"
    );
}

/// @brief checks that a read task on a channel with props can receive a message. A
/// bus controller polls transmit messages on their period, a remote terminal receives
/// messages to its own terminals, and a monitor sees every message.
/// @returns x::errors::VALIDATION naming the problem.
x::errors::Error check_read(
    const synnax::library::MessageEntry &message,
    const synnax::mil1553::Properties &props
) {
    const auto &id = std::get<synnax::library::Mil1553Identifier>(*message.identifier);
    if (props.role == synnax::mil1553::ROLE_BUS_CONTROLLER) {
        if (const auto err = check_direction(
                message,
                id,
                synnax::library::DIRECTION_TRANSMIT,
                "bus controller"
            ))
            return err;
        if (!message.period.has_value())
            return invalid(message, "needs a period for the bus controller to poll it");
        return x::errors::NIL;
    }
    if (props.role == synnax::mil1553::ROLE_REMOTE_TERMINAL) {
        if (id.direction != synnax::library::DIRECTION_RECEIVE)
            return invalid(
                message,
                "must be a receive message for a remote terminal to read"
            );
        return check_owned(message, id, props);
    }
    return x::errors::NIL;
}

/// @brief checks that a write task on a channel with props can send a message. A bus
/// controller sends receive messages, a remote terminal answers transmit messages for
/// its own terminals, and a monitor sends nothing.
/// @returns x::errors::VALIDATION naming the problem.
x::errors::Error check_write(
    const synnax::library::MessageEntry &message,
    const synnax::mil1553::Properties &props
) {
    const auto &id = std::get<synnax::library::Mil1553Identifier>(*message.identifier);
    if (props.role == synnax::mil1553::ROLE_BUS_CONTROLLER)
        return check_direction(
            message,
            id,
            synnax::library::DIRECTION_RECEIVE,
            "bus controller"
        );
    if (props.role == synnax::mil1553::ROLE_REMOTE_TERMINAL) {
        if (id.direction != synnax::library::DIRECTION_TRANSMIT)
            return invalid(
                message,
                "must be a transmit message for a remote terminal to answer with"
            );
        return check_owned(message, id, props);
    }
    return invalid(message, "cannot be sent by a monitor");
}

/// @brief binds the error of check to each enabled message of a resolved config
/// that fails it.
/// @returns x::errors::VALIDATION when a message fails.
template<typename Messages, typename Resolved>
x::errors::Error check_role(
    const x::json::Parser &parser,
    const Messages &messages,
    const Resolved &resolved,
    const synnax::mil1553::Properties &props,
    x::errors::Error (*check)(
        const synnax::library::MessageEntry &,
        const synnax::mil1553::Properties &
    )
) {
    for (std::size_t i = 0; i < messages.size(); i++) {
        if (messages[i].disabled) continue;
        const auto it = std::ranges::find_if(resolved, [&](const auto &m) {
            return m.entry.key == messages[i].message;
        });
        if (const auto err = check(it->entry, props))
            parser.field_err("messages." + std::to_string(i) + ".message", err.data);
    }
    return parser.ok() ? x::errors::NIL : parser.error();
}

/// @brief the device properties and channel of a task config.
struct Device {
    synnax::mil1553::Properties props;
    Acquire acquire;
};

std::pair<Device, x::errors::Error> retrieve_device(
    const Backends &backends,
    const std::shared_ptr<Links> &links,
    const synnax::Synnax &client,
    const x::json::Parser &parser,
    const std::string &key
) {
    auto [props, err] = bus::retrieve_properties<synnax::mil1553::Properties>(
        client,
        parser,
        key
    );
    if (err) return {Device{}, err};
    if (const auto props_err = validate(props)) {
        parser.field_err("device", props_err.data);
        return {Device{}, parser.error()};
    }
    const auto it = backends.find(props.backend);
    if (it == backends.end()) {
        parser.field_err("device", "unknown MIL-STD-1553 backend " + props.backend);
        return {Device{}, parser.error()};
    }
    auto acquire = acquirer(links, key, it->second, props);
    return {Device{std::move(props), std::move(acquire)}, x::errors::NIL};
}

std::pair<common::ConfigureResult, x::errors::Error> configure_read(
    const Backends &backends,
    const std::shared_ptr<Links> &links,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) {
    x::json::Parser parser(task.config);
    const auto cfg = synnax::mil1553::ReadConfig::parse(parser);
    if (!parser.ok()) return {common::ConfigureResult{}, parser.error()};
    auto [dev, dev_err] = retrieve_device(
        backends,
        links,
        *ctx->client,
        parser,
        cfg.device
    );
    if (dev_err) return {common::ConfigureResult{}, dev_err};
    auto [resolved, err] = bus::ReadConfig::parse(
        *ctx->client,
        parser,
        cfg,
        {},
        std::nullopt,
        bus::Medium::MIL1553
    );
    if (err) return {common::ConfigureResult{}, err};
    if (const auto role_err = check_role(
            parser,
            cfg.messages,
            resolved.messages,
            dev.props,
            check_read
        ))
        return {common::ConfigureResult{}, role_err};
    return {
        common::ConfigureResult{
            .task = std::make_unique<common::ReadTask>(
                task,
                ctx,
                x::breaker::default_config(task.name),
                std::make_unique<Source>(
                    std::move(resolved),
                    std::move(dev.props),
                    std::move(dev.acquire)
                )
            ),
            .auto_start = cfg.auto_start,
        },
        x::errors::NIL,
    };
}

std::pair<common::ConfigureResult, x::errors::Error> configure_write(
    const Backends &backends,
    const std::shared_ptr<Links> &links,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) {
    x::json::Parser parser(task.config);
    const auto cfg = synnax::mil1553::WriteConfig::parse(parser);
    if (!parser.ok()) return {common::ConfigureResult{}, parser.error()};
    auto [dev, dev_err] = retrieve_device(
        backends,
        links,
        *ctx->client,
        parser,
        cfg.device
    );
    if (dev_err) return {common::ConfigureResult{}, dev_err};
    auto [resolved, err] = bus::WriteConfig::parse(
        *ctx->client,
        parser,
        cfg,
        std::nullopt,
        bus::Medium::MIL1553
    );
    if (err) return {common::ConfigureResult{}, err};
    if (const auto role_err = check_role(
            parser,
            cfg.messages,
            resolved.messages,
            dev.props,
            check_write
        ))
        return {common::ConfigureResult{}, role_err};
    auto transmitter = std::make_unique<Transmitter>(
        resolved,
        std::move(dev.props),
        std::move(dev.acquire)
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

std::pair<common::ConfigureResult, x::errors::Error> configure_scan(
    const Backends &backends,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) {
    x::json::Parser parser(task.config);
    const auto cfg = synnax::mil1553::ScanConfig::parse(parser);
    if (!parser.ok()) return {common::ConfigureResult{}, parser.error()};
    return {
        common::ConfigureResult{
            .task = std::make_unique<common::ScanTask>(
                std::make_unique<
                    bus::Scanner<Backend>>(task, INTEGRATION_NAME, MAKE, backends),
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
        "MIL-STD-1553 Scanner",
        SCAN_TASK_TYPE,
        INTEGRATION_NAME
    );
}
}
