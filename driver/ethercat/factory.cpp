// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/ethercat/ethercat.h"
#include "driver/ethercat/read_task.h"
#include "driver/ethercat/scan_task.h"
#include "driver/ethercat/soem/master.h"
#include "driver/ethercat/write_task.h"
#include "driver/task/common/factory.h"
#include "driver/task/common/read_task.h"
#include "driver/task/common/scan_task.h"
#include "driver/task/common/write_task.h"

#ifdef __linux__
#include "driver/ethercat/igh/master.h"
#endif

#include "x/cpp/breaker/breaker.h"

namespace ethercat {

std::unique_ptr<master::Manager> default_manager() {
#ifdef __linux__
    auto [igh_mgr, err] = igh::Manager::open();
    if (!err) return std::move(igh_mgr);
#endif
    return std::make_unique<soem::Manager>();
}

Factory::Factory(): pool(std::make_shared<engine::Pool>(default_manager())) {}

Factory::Factory(std::unique_ptr<master::Manager> manager):
    pool(std::make_shared<engine::Pool>(std::move(manager))) {}

std::pair<common::ConfigureResult, xerrors::Error> Factory::configure_read(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) const {
    common::ConfigureResult result;
    auto [cfg, cfg_err] = ReadTaskConfig::parse(ctx->client, task);
    if (cfg_err) return {std::move(result), cfg_err};
    auto [eng, eng_err] = this->pool->acquire(cfg.interface_name);
    if (eng_err) return {std::move(result), eng_err};
    result.auto_start = cfg.auto_start;
    result.task = std::make_unique<common::ReadTask>(
        task,
        ctx,
        breaker::default_config(task.name),
        std::make_unique<ReadTaskSource>(eng, std::move(cfg))
    );
    return {std::move(result), xerrors::NIL};
}

std::pair<common::ConfigureResult, xerrors::Error> Factory::configure_write(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) const {
    common::ConfigureResult result;
    auto [cfg, cfg_err] = WriteTaskConfig::parse(ctx->client, task);
    if (cfg_err) return {std::move(result), cfg_err};
    auto [eng, eng_err] = this->pool->acquire(cfg.interface_name);
    if (eng_err) return {std::move(result), eng_err};
    result.auto_start = cfg.auto_start;
    result.task = std::make_unique<common::WriteTask>(
        task,
        ctx,
        breaker::default_config(task.name),
        std::make_unique<WriteTaskSink>(eng, std::move(cfg))
    );
    return {std::move(result), xerrors::NIL};
}

std::pair<common::ConfigureResult, xerrors::Error> Factory::configure_scan(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    common::ConfigureResult result;
    xjson::Parser parser(task.config);
    ScanTaskConfig cfg(parser);
    if (parser.error()) return {std::move(result), parser.error()};
    auto scanner = std::make_unique<Scanner>(ctx, task, cfg, this->pool);
    result.task = std::make_unique<common::ScanTask>(
        std::move(scanner),
        ctx,
        task,
        breaker::default_config(task.name),
        cfg.scan_rate
    );
    result.auto_start = cfg.enabled;
    return {std::move(result), xerrors::NIL};
}

std::pair<std::unique_ptr<task::Task>, bool> Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    if (task.type.find(INTEGRATION_NAME) != 0) return {nullptr, false};
    std::pair<common::ConfigureResult, xerrors::Error> res;
    if (task.type == READ_TASK_TYPE)
        res = this->configure_read(ctx, task);
    else if (task.type == WRITE_TASK_TYPE)
        res = this->configure_write(ctx, task);
    else if (task.type == SCAN_TASK_TYPE)
        res = this->configure_scan(ctx, task);
    else
        return {nullptr, false};
    return common::handle_config_err(ctx, task, std::move(res));
}

std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task>>>
Factory::configure_initial_tasks(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Rack &rack
) {
    return common::configure_initial_factory_tasks(
        this,
        ctx,
        rack,
        "EtherCAT Scanner",
        SCAN_TASK_TYPE,
        INTEGRATION_NAME
    );
}

bool Factory::is_interface_active(const std::string &key) const {
    return this->pool->is_active(key);
}

std::vector<SlaveInfo> Factory::get_cached_slaves(const std::string &key) const {
    return this->pool->get_slaves(key);
}

}
