// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cassert>
#include <utility>
#include <stdio.h>

#include "driver/ni/ni.h"
#include "driver/ni/reader.h"
#include "driver/ni/writer.h"
#include "driver/ni/scanner.h"
#include "driver/pipeline/middleware.h"

ni::ScannerTask::ScannerTask(
    const std::shared_ptr<SysCfg> &syscfg,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) : breaker(breaker::default_config(task.name)),
    scanner(syscfg, ctx, task),
    ctx(ctx),
    task(task) {
    auto parser = xjson::Parser(task.config);
    bool enabled = parser.optional<bool>("enabled", true);

    if (!scanner.ok() || !enabled) {
        ctx->set_state({
            .task = task.key,
            .variant = "error",
            .details = {"message", "failed to initialize scanner"}
        });
        return;
    }
    this->breaker.start();
    thread = std::make_shared<std::thread>(&ni::ScannerTask::run, this);
    this->scanner.set_scan_thread(thread);
}

std::unique_ptr<task::Task> ni::ScannerTask::configure(
    const std::shared_ptr<SysCfg> &syscfg,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    return std::make_unique<ni::ScannerTask>(syscfg, ctx, task);
}

void ni::ScannerTask::stop() {
    this->breaker.stop();
}

void ni::ScannerTask::exec(task::Command &cmd) {
    if (cmd.type == "scan") {
        scanner.scan();
        scanner.create_devices();
    } else if (cmd.type == "stop") {
        this->stop();
        this->scanner.join_scan_thread();
    } else {
        LOG(ERROR) << "unknown command type: " << cmd.type;
    }
}

void ni::ScannerTask::run() {
    auto scan_cmd = task::Command{task.key, "scan", {}};
    while (this->breaker.running()) {
        this->breaker.wait_for(this->scan_rate.period().chrono());
        this->exec(scan_cmd);
    }
    LOG(INFO) << "[ni.scanner] stopped scanning " << this->task.name;
}