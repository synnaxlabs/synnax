// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <mutex>
#include <unordered_map>

#include "glog/logging.h"

#include "x/cpp/breaker/breaker.h"

#include "driver/ethercat/ethercat.h"
#include "driver/ethercat/cyclic_engine.h"
#include "driver/ethercat/read_task.h"
#include "driver/ethercat/write_task.h"
#include "driver/ethercat/soem/master.h"
#include "driver/task/common/factory.h"
#include "driver/task/common/read_task.h"
#include "driver/task/common/write_task.h"

namespace ethercat {
struct Factory::Impl {
    std::unordered_map<std::string, std::shared_ptr<CyclicEngine>> engines;
    std::mutex engines_mutex;

    std::shared_ptr<CyclicEngine> get_or_create_engine(const std::string &interface_name
    ) {
        std::lock_guard lock(engines_mutex);
        auto it = engines.find(interface_name);
        if (it != engines.end()) return it->second;

        auto master = std::make_shared<soem::SOEMMaster>(interface_name);

        auto engine = std::make_shared<CyclicEngine>(std::move(master));
        engines[interface_name] = engine;
        return engine;
    }

    std::pair<common::ConfigureResult, xerrors::Error> configure_read(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) {
        common::ConfigureResult result;
        auto [cfg, err] = ReadTaskConfig::parse(ctx->client, task);
        if (err) return {std::move(result), err};

        auto engine = get_or_create_engine(cfg.device_key);
        result.auto_start = cfg.auto_start;
        result.task = std::make_unique<common::ReadTask>(
            task,
            ctx,
            breaker::default_config(task.name),
            std::make_unique<ReadTaskSource>(engine, std::move(cfg))
        );
        return {std::move(result), xerrors::NIL};
    }

    std::pair<common::ConfigureResult, xerrors::Error> configure_write(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) {
        common::ConfigureResult result;
        auto [cfg, err] = WriteTaskConfig::parse(ctx->client, task);
        if (err) return {std::move(result), err};

        auto engine = get_or_create_engine(cfg.device_key);
        result.auto_start = cfg.auto_start;
        result.task = std::make_unique<common::WriteTask>(
            task,
            ctx,
            breaker::default_config(task.name),
            std::make_unique<WriteTaskSink>(engine, std::move(cfg))
        );
        return {std::move(result), xerrors::NIL};
    }
};

Factory::Factory(): impl_(std::make_unique<Impl>()) {}

Factory::~Factory() = default;

std::pair<std::unique_ptr<task::Task>, bool> Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    if (task.type.find(INTEGRATION_NAME) != 0) return {nullptr, false};
    std::pair<common::ConfigureResult, xerrors::Error> res;
    if (task.type == READ_TASK_TYPE)
        res = impl_->configure_read(ctx, task);
    else if (task.type == WRITE_TASK_TYPE)
        res = impl_->configure_write(ctx, task);
    else
        return {nullptr, false};
    return common::handle_config_err(ctx, task, std::move(res));
}

std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task>>>
Factory::configure_initial_tasks(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Rack &rack
) {
    return {};
}
}
