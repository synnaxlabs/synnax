// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// module
#include "glog/logging.h"

#include "state.h"

#include "x/cpp/breaker/breaker.h"
#include "x/cpp/loop/loop.h"

/// internal
#include "client/cpp/hardware/hardware.h"
#include "driver/pipeline/base.h"
#include "driver/task/task.h"

namespace common {
struct ScannerContext {
    std::size_t count = 0;
};


struct Scanner {
    virtual ~Scanner() = default;

    virtual xerrors::Error start() { return xerrors::NIL; }

    virtual xerrors::Error stop() { return xerrors::NIL; }

    virtual std::pair<std::vector<synnax::Device>, xerrors::Error> scan(
        const ScannerContext &ctx) = 0;
};

class ScanTask final : public task::Task, public pipeline::Base {
    const std::string task_name;
    loop::Timer timer;
    std::unique_ptr<Scanner> scanner;
    std::shared_ptr<task::Context> ctx;
    task::State state;
    ScannerContext scanner_ctx;

public:
    ScanTask(
        std::unique_ptr<Scanner> scanner,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task,
        const breaker::Config &breaker_config,
        const telem::Rate scan_rate
    ): pipeline::Base(breaker_config),
       task_name(task.name),
       timer(scan_rate),
       scanner(std::move(scanner)),
       ctx(ctx),
       scanner_ctx() {
        this->state.task = task.key;
        this->state.variant = "pending";
        this->state.details["message"] = "scan task pending";
        this->ctx->set_state(this->state);
    }


    void run() override {
        if (const auto err = this->scanner->start()) {
            this->state.variant = "error";
            this->state.details["message"] = err.message();
            this->ctx->set_state(this->state);
            return;
        }
        this->state.variant = "success";
        this->state.details["message"] = "scan task started";
        this->ctx->set_state(this->state);
        while (this->breaker.running()) {
            if (const auto err = this->scan()) {
                this->state.variant = "warning";
                this->state.details["message"] = err.message();
                this->ctx->set_state(this->state);
                LOG(WARNING) << "[ni.scan_task] failed to scan for devices: " << err;
            }
            this->timer.wait();
        }
        if (const auto err = this->scanner->stop()) {
            this->state.variant = "error";
            this->state.details["message"] = err.message();
        } else {
            this->state.variant = "success";
            this->state.details["message"] = "scan task stopped";
        }
        this->ctx->set_state(this->state);
    }

    void exec(task::Command &cmd) override {
        this->state.key = cmd.key;
        if (cmd.type == common::STOP_CMD_TYPE) return this->stop(false);
        if (cmd.type == common::START_CMD_TYPE) this->start();
        else if (cmd.type == common::SCAN_CMD_TYPE) {
            const auto err = this->scan();
            this->state.variant = "error";
            this->state.details["message"] = err.message();
            this->ctx->set_state(this->state);
        }
    }

    xerrors::Error scan() {
        auto [scanned_devs, err] = this->scanner->scan(scanner_ctx);
        this->scanner_ctx.count++;
        if (err) return err;
        const auto client = this->ctx->client;

        std::vector<std::string> devices;
        for (const auto &device: scanned_devs) devices.push_back(device.key);
        auto [remote_devs_vec, ret_err] = client->hardware.retrieve_devices(devices);
        if (ret_err) return ret_err;

        auto remote_devs = synnax::device_keys_map(remote_devs_vec);

        for (auto &scanned_dev: scanned_devs) {
            auto remote_dev = remote_devs.find(scanned_dev.key);
            if (remote_dev != remote_devs.end()) continue;
            if (const auto c_err = client->hardware.create_device(remote_dev->second))
                return c_err;
        }
        return xerrors::NIL;
    }

    std::string name() override { return this->task_name; }

    void stop(bool will_reconfigure) override {
        pipeline::Base::stop();
    }
};
}
