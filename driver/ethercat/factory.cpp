// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <utility>
#include <vector>

#include "absl/log/log.h"

#include "x/cpp/lib/lib.h"

#include "driver/common/factory.h"
#include "driver/common/read_task.h"
#include "driver/common/scan_task.h"
#include "driver/common/status.h"
#include "driver/common/write_task.h"
#include "driver/ethercat/ethercat.h"
#include "driver/ethercat/read_task.h"
#include "driver/ethercat/scan_task.h"
#include "driver/ethercat/soem/master.h"
#include "driver/ethercat/write_task.h"

#ifdef __linux__
#include "driver/ethercat/igh/master.h"
#endif

#include "x/cpp/breaker/breaker.h"

namespace driver::ethercat {
namespace {
/// @brief the percent of a task operation's timeout that a bus start may use. The rest
/// covers the scan and the reads before the start.
constexpr int64_t START_SHARE_PERCENT = 75;

#ifdef _WIN32
const std::string PCAP_LIB_NAME = "wpcap.dll";
#elif defined(__APPLE__)
const std::string PCAP_LIB_NAME = "libpcap.dylib";
#endif

#ifndef __linux__
bool check_pcap_available() {
#ifdef _WIN32
    // Npcap installs its DLLs outside the default search path unless it runs in
    // WinPcap-compatible mode.
    char dir[MAX_PATH];
    const UINT len = GetSystemDirectoryA(dir, MAX_PATH);
    if (len > 0 && len < MAX_PATH)
        SetDllDirectoryA((std::string(dir) + "\\Npcap").c_str());
#endif
    x::lib::Shared lib(PCAP_LIB_NAME);
    return lib.load();
}
#endif

std::unique_ptr<master::Manager> default_manager() {
#ifdef __linux__
    auto [igh_mgr, err] = igh::Manager::open();
    if (!err) {
        LOG(INFO) << "[ethercat] using IgH EtherCAT master backend";
        return std::move(igh_mgr);
    }
    LOG(INFO) << "[ethercat] IgH unavailable, falling back to SOEM backend: "
              << err.message();
#else
    LOG(INFO) << "[ethercat] using SOEM backend";
    if (!check_pcap_available()) return nullptr;
#endif
    return std::make_unique<soem::Manager>();
}

/// @brief returns the settings saved for the slaves on the network. Its errors call the
/// network name.
/// @returns errors::SLAVE_CONFIG_ERROR when a slave on the network has invalid
/// properties, since the bus starts every slave on it.
std::pair<std::vector<engine::SavedSlave>, x::errors::Error> slave_configs(
    const std::shared_ptr<synnax::Synnax> &client,
    const synnax::rack::Key rack,
    const std::string &network,
    const std::string &name
) {
    auto [devices, dev_err] = client->devices.retrieve(
        synnax::device::Where{
            .make = DEVICE_MAKE,
            .model = SLAVE_DEVICE_MODEL,
            .rack = rack,
        }
    );
    if (dev_err)
        return {
            {},
            x::errors::Error(
                dev_err,
                "The Driver could not read the slave settings for " + name +
                    " from the Core: " + dev_err.data
            )
        };
    std::vector<engine::SavedSlave> saved;
    for (const auto &dev: devices) {
        auto parser = x::json::Parser(dev.properties);
        const auto props = slave::Properties::parse(parser);
        if (props.network != network) continue;
        if (parser.error())
            return {
                {},
                x::errors::Error(
                    errors::SLAVE_CONFIG_ERROR,
                    dev.name + " has invalid properties. Every slave on " + name +
                        " starts with the bus, so fix them before you start the "
                        "task.\n" +
                        common::config_err_description(parser.error())
                )
            };
        saved.push_back({
            .identity =
                {
                    .position = props.position,
                    .vendor_id = props.vendor_id,
                    .product_code = props.product_code,
                    .serial = props.serial,
                    .name = dev.name,
                },
            .config = props.config,
        });
    }
    return {std::move(saved), x::errors::NIL};
}
}

Factory::Factory(
    std::shared_ptr<synnax::Synnax> client,
    const synnax::rack::Key rack,
    std::shared_ptr<x::thread::rt::Manager> rt_manager,
    const x::telem::TimeSpan op_timeout
):
    Factory(
        default_manager(),
        std::move(client),
        rack,
        std::move(rt_manager),
        op_timeout
    ) {}

Factory::Factory(
    std::unique_ptr<master::Manager> manager,
    std::shared_ptr<synnax::Synnax> client,
    const synnax::rack::Key rack,
    std::shared_ptr<x::thread::rt::Manager> rt_manager,
    const x::telem::TimeSpan op_timeout
):
    pool(
        std::make_shared<engine::Pool>(
            std::move(manager),
            [client = std::move(client),
             rack](const std::string &network, const std::string &name) {
                return slave_configs(client, rack, network, name);
            },
            std::move(rt_manager),
            op_timeout * START_SHARE_PERCENT / 100
        )
    ) {}

std::pair<common::ConfigureResult, x::errors::Error> Factory::configure_read(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) const {
    common::ConfigureResult result;
    auto [cfg, cfg_err] = ReadTaskConfig::parse(ctx->client, task);
    if (cfg_err) return {std::move(result), cfg_err};
    auto [eng, eng_err] = this->pool->acquire(cfg.slaves.network);
    if (eng_err) return {std::move(result), eng_err};
    result.auto_start = cfg.auto_start;
    // The bus recovers a lost device without a limit, so the task waits for it.
    result.task = std::make_unique<common::ReadTask>(
        task,
        ctx,
        common::unlimited_retry_config(task.name),
        std::make_unique<ReadTaskSource>(eng, std::move(cfg))
    );
    return {std::move(result), x::errors::NIL};
}

std::pair<common::ConfigureResult, x::errors::Error> Factory::configure_write(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) const {
    common::ConfigureResult result;
    auto [cfg, cfg_err] = WriteTaskConfig::parse(ctx->client, task);
    if (cfg_err) return {std::move(result), cfg_err};
    auto [eng, eng_err] = this->pool->acquire(cfg.slaves.network);
    if (eng_err) return {std::move(result), eng_err};
    result.auto_start = cfg.auto_start;
    result.task = std::make_unique<common::WriteTask>(
        task,
        ctx,
        x::breaker::default_config(task.name),
        std::make_unique<WriteTaskSink>(eng, std::move(cfg))
    );
    return {std::move(result), x::errors::NIL};
}

std::pair<common::ConfigureResult, x::errors::Error> Factory::configure_scan(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) {
    common::ConfigureResult result;
    x::json::Parser parser(task.config);
    ScanTaskConfig cfg(parser);
    if (parser.error()) return {std::move(result), parser.error()};
    result.auto_start = !cfg.disabled;
    if (auto err = this->pool->check_available()) return {std::move(result), err};
    auto scanner = std::make_unique<Scanner>(ctx, task, cfg, this->pool);
    result.task = std::make_unique<common::ScanTask>(
        std::move(scanner),
        ctx,
        task,
        x::breaker::default_config(task.name),
        cfg.rate
    );
    return {std::move(result), x::errors::NIL};
}

std::pair<std::unique_ptr<task::Task>, bool> Factory::configure_task(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    const std::string &cmd_key,
    x::breaker::Breaker &cancel
) {
    if (!task.type.starts_with(INTEGRATION_NAME)) return {nullptr, false};
    std::pair<common::ConfigureResult, x::errors::Error> res;
    if (task.type == READ_TASK_TYPE)
        res = this->configure_read(ctx, task);
    else if (task.type == WRITE_TASK_TYPE)
        res = this->configure_write(ctx, task);
    else if (task.type == SCAN_TASK_TYPE)
        res = this->configure_scan(ctx, task);
    else
        return {nullptr, false};
    return common::handle_config_err(ctx, task, std::move(res), cmd_key, cancel);
}

std::vector<std::pair<synnax::task::Task, std::unique_ptr<task::Task>>>
Factory::configure_initial_tasks(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::rack::Rack &rack,
    x::breaker::Breaker &cancel
) {
    return common::configure_initial_factory_tasks(
        this,
        ctx,
        rack,
        "EtherCAT Scanner",
        SCAN_TASK_TYPE,
        INTEGRATION_NAME,
        cancel
    );
}

}
