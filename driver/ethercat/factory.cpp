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
#include "driver/ethercat/read_task.h"
#include "driver/ethercat/scan_task.h"
#include "driver/ethercat/soem/master.h"
#include "driver/ethercat/write_task.h"
#include "driver/task/common/factory.h"
#include "driver/task/common/read_task.h"
#include "driver/task/common/scan_task.h"
#include "driver/task/common/write_task.h"
#include "loop/loop.h"

#ifdef __linux__
#include "driver/ethercat/igh/master.h"
#endif

namespace ethercat {

std::shared_ptr<Master> create_master(
    const std::string &interface_name,
    const std::string &backend,
    unsigned int master_index
) {
#ifdef __linux__
    bool use_igh = (backend == "igh") || (backend == "auto" && igh::igh_available());
    if (use_igh) {
        VLOG(1) << "Using IgH EtherCAT master (index " << master_index << ")";
        return std::make_shared<igh::Master>(master_index);
    }
#endif
    VLOG(1) << "Using SOEM EtherCAT master on " << interface_name;
    return std::make_shared<soem::Master>(interface_name);
}

struct Factory::Impl {
    std::unordered_map<std::string, std::shared_ptr<Loop>> engines;
    mutable std::mutex engines_mutex;

    std::shared_ptr<Loop> get_or_create_engine(
        const std::string &interface_name,
        const std::string &backend = "auto",
        unsigned int master_index = 0
    ) {
        std::lock_guard lock(engines_mutex);
        std::string key = backend == "igh" ? "igh:" + std::to_string(master_index)
                                           : interface_name;
        auto it = engines.find(key);
        if (it != engines.end()) return it->second;

        auto master = create_master(interface_name, backend, master_index);
        auto engine = std::make_shared<Loop>(std::move(master));
        engines[key] = engine;
        return engine;
    }

    std::pair<common::ConfigureResult, xerrors::Error> configure_read(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) {
        common::ConfigureResult result;
        auto [cfg, err] = ReadTaskConfig::parse(ctx->client, task);
        if (err) return {std::move(result), err};

        auto engine = get_or_create_engine(cfg.interface_name);
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

        auto engine = get_or_create_engine(cfg.interface_name);
        result.auto_start = cfg.auto_start;
        result.task = std::make_unique<common::WriteTask>(
            task,
            ctx,
            breaker::default_config(task.name),
            std::make_unique<WriteTaskSink>(engine, std::move(cfg))
        );
        return {std::move(result), xerrors::NIL};
    }

    std::pair<common::ConfigureResult, xerrors::Error> configure_scan(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task,
        Factory *factory
    ) {
        common::ConfigureResult result;
        xjson::Parser parser(task.config);
        ScanTaskConfig cfg(parser);
        if (parser.error()) return {std::move(result), parser.error()};

        auto scanner = std::make_unique<Scanner>(ctx, task, cfg, factory);
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

    bool is_interface_active(const std::string &interface) const {
        std::lock_guard lock(engines_mutex);
        auto it = engines.find(interface);
        return it != engines.end() && it->second->is_running();
    }

    std::vector<SlaveInfo> get_cached_slaves(const std::string &interface) const {
        std::lock_guard lock(engines_mutex);
        auto it = engines.find(interface);
        if (it != engines.end()) return it->second->master->slaves();
        return {};
    }
};

Factory::Factory(): impl(std::make_unique<Impl>()) {}

Factory::~Factory() = default;

std::pair<std::unique_ptr<task::Task>, bool> Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    if (task.type.find(INTEGRATION_NAME) != 0) return {nullptr, false};
    std::pair<common::ConfigureResult, xerrors::Error> res;
    if (task.type == READ_TASK_TYPE)
        res = this->impl->configure_read(ctx, task);
    else if (task.type == WRITE_TASK_TYPE)
        res = this->impl->configure_write(ctx, task);
    else if (task.type == SCAN_TASK_TYPE)
        res = this->impl->configure_scan(ctx, task, this);
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

bool Factory::is_interface_active(const std::string &interface) const {
    return this->impl->is_interface_active(interface);
}

std::vector<SlaveInfo> Factory::get_cached_slaves(const std::string &interface) const {
    return this->impl->get_cached_slaves(interface);
}
}
