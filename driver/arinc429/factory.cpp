// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "client/cpp/arinc429/json.gen.h"
#include "x/cpp/breaker/breaker.h"

#include "driver/arinc429/arinc429.h"
#include "driver/arinc429/ballard/backend.h"
#include "driver/arinc429/ddc/backend.h"
#include "driver/arinc429/read.h"
#include "driver/arinc429/simulated/backend.h"
#include "driver/arinc429/write.h"
#include "driver/bus/scan.h"
#include "driver/bus/task.h"
#include "driver/common/factory.h"
#include "driver/common/scan_task.h"
#include "driver/common/status.h"
#include "driver/common/write_task.h"
#include "driver/errors/errors.h"

namespace driver::arinc429 {
Backends create_backends() {
    return {
        {synnax::arinc429::BACKEND_SIMULATED, std::make_shared<simulated::Backend>()},
        {synnax::arinc429::BACKEND_DDC, std::make_shared<ddc::Backend>()},
        {synnax::arinc429::BACKEND_BALLARD, std::make_shared<ballard::Backend>()},
    };
}

x::errors::Error check(const synnax::library::MessageEntry &message) {
    if (const auto err = codec::arinc429::validate(message)) return err;
    if (message.query.has_value())
        return x::errors::Error(
            x::errors::VALIDATION,
            "message " + message.name + " has a query, which ARINC 429 cannot send"
        );
    return x::errors::NIL;
}

std::pair<std::shared_ptr<Backend>, x::errors::Error>
resolve(const Backends &backends, const synnax::arinc429::Properties &props) {
    if (props.speed != synnax::arinc429::SPEED_LOW &&
        props.speed != synnax::arinc429::SPEED_HIGH)
        return {
            nullptr,
            x::errors::Error(
                errors::CONFIGURATION_ERROR,
                "unknown ARINC 429 speed " + props.speed
            ),
        };
    const auto it = backends.find(props.backend);
    if (it == backends.end())
        return {
            nullptr,
            x::errors::Error(
                errors::CONFIGURATION_ERROR,
                "unknown ARINC 429 backend " + props.backend
            ),
        };
    return {it->second, x::errors::NIL};
}

namespace {
/// @brief the device properties and backend of a task config.
struct Device {
    synnax::arinc429::Properties props;
    std::shared_ptr<Backend> backend;
};

std::pair<Device, x::errors::Error> retrieve_device(
    const Backends &backends,
    const synnax::Synnax &client,
    const x::json::Parser &parser,
    const std::string &key
) {
    auto [props, err] = bus::retrieve_properties<synnax::arinc429::Properties>(
        client,
        parser,
        key
    );
    if (err) return {Device{}, err};
    auto [backend, backend_err] = resolve(backends, props);
    if (backend_err) {
        parser.field_err("device", backend_err.data);
        return {Device{}, parser.error()};
    }
    return {Device{std::move(props), std::move(backend)}, x::errors::NIL};
}

std::pair<common::ConfigureResult, x::errors::Error> configure_read(
    const Backends &backends,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) {
    x::json::Parser parser(task.config);
    const auto cfg = synnax::arinc429::ReadConfig::parse(parser);
    if (!parser.ok()) return {common::ConfigureResult{}, parser.error()};
    auto [dev, dev_err] = retrieve_device(backends, *ctx->client, parser, cfg.device);
    if (dev_err) return {common::ConfigureResult{}, dev_err};
    auto [resolved, err] = bus::ReadConfig::parse(
        *ctx->client,
        parser,
        cfg,
        {},
        std::nullopt,
        check
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
                    std::move(dev.backend),
                    std::move(dev.props)
                )
            ),
            .auto_start = cfg.auto_start,
        },
        x::errors::NIL,
    };
}

std::pair<common::ConfigureResult, x::errors::Error> configure_write(
    const Backends &backends,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) {
    x::json::Parser parser(task.config);
    const auto cfg = synnax::arinc429::WriteConfig::parse(parser);
    if (!parser.ok()) return {common::ConfigureResult{}, parser.error()};
    auto [dev, dev_err] = retrieve_device(backends, *ctx->client, parser, cfg.device);
    if (dev_err) return {common::ConfigureResult{}, dev_err};
    auto [resolved, err] = bus::WriteConfig::parse(
        *ctx->client,
        parser,
        cfg,
        std::nullopt,
        check
    );
    if (err) return {common::ConfigureResult{}, err};
    auto output = std::make_unique<Output>(
        resolved,
        std::move(dev.backend),
        std::move(dev.props)
    );
    return {
        common::ConfigureResult{
            .task = std::make_unique<common::WriteTask>(
                task,
                ctx,
                x::breaker::default_config(task.name),
                std::make_unique<
                    bus::Sink>(std::move(resolved), std::move(output), ctx, task)
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
    const auto cfg = synnax::arinc429::ScanConfig::parse(parser);
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
            configure_read(this->backends, ctx, task),
            cmd_key
        );
    if (task.type == WRITE_TASK_TYPE)
        return common::handle_config_err(
            ctx,
            task,
            configure_write(this->backends, ctx, task),
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
        "ARINC 429 Scanner",
        SCAN_TASK_TYPE,
        INTEGRATION_NAME
    );
}
}
