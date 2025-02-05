// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <vector>
#include <map>
#include <queue>
#include <utility>
#include <memory>
#include <atomic>
#include <thread>
#include <set>
#include <condition_variable>

#include "nidaqmx/nidaqmx_api.h"
#include "nidaqmx/nidaqmx.h"
#include "nisyscfg/nisyscfg.h"
#include "nisyscfg/nisyscfg_api.h"

#include "nlohmann/json.hpp"

#include "client/cpp/synnax.h"

#include "driver/ni/channels.h"
#include "driver/ni/error.h"
#include "driver/queue/ts_queue.h"
#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/control.h"
#include "driver/pipeline/middleware.h"
#include "driver/task/task.h"
#include "driver/breaker/breaker.h"
#include "driver/config/config.h"
#include "driver/errors/errors.h"
#include "driver/loop/loop.h"

namespace ni {
///////////////////////////////////////////////////////////////////////////////////
//                                    Scanner                                    //
///////////////////////////////////////////////////////////////////////////////////
class Scanner final {
public:
    explicit Scanner() = default;

    explicit Scanner(
        const std::shared_ptr<SysCfg> &syscfg,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    );

    ~Scanner();

    void scan();

    bool ok();

    json get_devices();

    void create_devices();

    void set_scan_thread(std::shared_ptr<std::thread> scan_thread);

    void join_scan_thread();

    void log_err(std::string err_msg);

private:
    std::shared_ptr<SysCfg> syscfg;

    json get_device_properties(NISysCfgResourceHandle resource);

    json devices;
    std::set<std::string> device_keys;
    bool ok_state = true;
    NISysCfgSessionHandle session;
    NISysCfgFilterHandle filter;
    NISysCfgEnumResourceHandle resources_handle;
    synnax::Task task;
    std::shared_ptr<task::Context> ctx;
    std::shared_ptr<std::thread> scan_thread = nullptr;
    //optional scan thread a task could be running
}; // class Scanner

///////////////////////////////////////////////////////////////////////////////////
//                                    ScannerTask                                //
///////////////////////////////////////////////////////////////////////////////////
class ScannerTask final : public task::Task {
public:
    explicit ScannerTask(
        const std::shared_ptr<SysCfg> &syscfg,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    );

    void exec(task::Command &cmd) override;

    static std::unique_ptr<task::Task> configure(
        const std::shared_ptr<SysCfg> &syscfg,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    );

    void run();

    void stop() override;

    std::string name() override { return task.name; }

    bool ok();

private:
    std::shared_ptr<SysCfg> syscfg;
    breaker::Breaker breaker;
    ni::Scanner scanner;
    std::shared_ptr<task::Context> ctx;
    synnax::Task task;
    std::shared_ptr<std::thread> thread;
    bool ok_state = true;
    synnax::Rate scan_rate = synnax::Rate(1);
}; // class ScannerTask
} // namespace ni