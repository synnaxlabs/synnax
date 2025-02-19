// Copyright 2025 Synnax Labs, Inc.
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
#include <thread>
#include <set>
#include <condition_variable>

#include "nisyscfg/nisyscfg.h"
#include "nisyscfg/nisyscfg_api.h"

#include "nlohmann/json.hpp"

#include "client/cpp/synnax.h"

#include "driver/task/task.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/loop/loop.h"

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

    bool ok() const;

    json get_devices();

    void create_devices();

    void set_scan_thread(const std::shared_ptr<std::thread> &scan_thread);

    void join_scan_thread() const;

    void log_err(std::string err_msg);

private:
    const std::vector<std::string> IGNORED_MODEL_PREFIXES = {"O", "cRIO", "nown"};  // Add more prefixes as needed

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
    telem::Rate scan_rate = telem::Rate(1);
}; // class ScannerTask
} // namespace ni
