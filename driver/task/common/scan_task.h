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


/// module
#include "client/cpp/hardware/hardware.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/loop/loop.h"

/// internal
#include "driver/pipeline/base.h"
#include "driver/task/common/state.h"
#include "driver/task/task.h"

namespace common {
struct ScannerContext {
    std::size_t count = 0;
};

struct Scanner {
    virtual ~Scanner() = default;

    virtual xerrors::Error start() { return xerrors::NIL; }

    virtual xerrors::Error stop() { return xerrors::NIL; }

    virtual std::pair<std::vector<synnax::Device>, xerrors::Error>
    scan(const ScannerContext &ctx) = 0;
};

struct ClusterAPI {
    virtual ~ClusterAPI() = default;

    virtual std::pair<std::vector<synnax::Device>, xerrors::Error>
    retrieve_devices(std::vector<std::string> &keys) = 0;

    virtual xerrors::Error create_devices(std::vector<synnax::Device> &devs) = 0;
};

struct SynnaxClusterAPI final : ClusterAPI {
    std::shared_ptr<synnax::Synnax> client;

    explicit SynnaxClusterAPI(const std::shared_ptr<synnax::Synnax> &client):
        client(client) {}

    std::pair<std::vector<synnax::Device>, xerrors::Error>
    retrieve_devices(std::vector<std::string> &keys) override {
        // Ignore devices that are not found, as we can still work with partial
        // results.
        return this->client->hardware.retrieve_devices(keys, true);
    }

    xerrors::Error create_devices(std::vector<synnax::Device> &devs) override {
        return this->client->hardware.create_devices(devs);
    }
};

class ScanTask final : public task::Task, public pipeline::Base {
    const std::string task_name;
    loop::Timer timer;
    std::unique_ptr<Scanner> scanner;
    std::shared_ptr<task::Context> ctx;
    task::State state;
    ScannerContext scanner_ctx;
    std::unique_ptr<ClusterAPI> client;
    std::unordered_map<std::string, telem::TimeStamp> last_updated;

    [[nodiscard]] bool update_threshold_exceeded(const std::string &dev_key) {
        auto last_updated = telem::TimeStamp(0);
        if (const auto existing_last_updated = this->last_updated.find(dev_key);
            existing_last_updated != this->last_updated.end()) {
            last_updated = existing_last_updated->second;
        }
        const auto delta = telem::TimeStamp::now() - last_updated;
        return delta > telem::SECOND * 30;
    }

public:
    ScanTask(
        std::unique_ptr<Scanner> scanner,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task,
        const breaker::Config &breaker_config,
        const telem::Rate scan_rate,
        std::unique_ptr<ClusterAPI> client
    ):
        pipeline::Base(breaker_config),
        task_name(task.name),
        timer(scan_rate),
        scanner(std::move(scanner)),
        ctx(ctx),
        client(std::move(client)) {
        this->state.task = task.key;
        this->ctx->set_state(this->state);
    }

    ScanTask(
        std::unique_ptr<Scanner> scanner,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task,
        const breaker::Config &breaker_config,
        const telem::Rate scan_rate
    ):
        ScanTask(
            std::move(scanner),
            ctx,
            task,
            breaker_config,
            scan_rate,
            std::make_unique<SynnaxClusterAPI>(ctx->client)
        ) {}


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
                LOG(WARNING) << "[scan_task] failed to scan for devices: " << err;
            }
            this->timer.wait(this->breaker);
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
        if (cmd.type == common::START_CMD_TYPE)
            this->start();
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
        if (err || scanned_devs.empty()) return err;

        std::vector<std::string> devices;
        for (const auto &device: scanned_devs)
            devices.push_back(device.key);
        auto [remote_devs_vec, ret_err] = this->client->retrieve_devices(devices);
        if (ret_err && !ret_err.matches(xerrors::NOT_FOUND)) return ret_err;

        auto remote_devs = synnax::map_device_keys(remote_devs_vec);

        std::vector<synnax::Device> to_create;
        for (auto &scanned_dev: scanned_devs) {
            // Unless the device already exists on the remote, it should not
            // be configured. No exceptions.
            scanned_dev.configured = false;
            auto iter = remote_devs.find(scanned_dev.key);
            if (iter == remote_devs.end()) {
                to_create.push_back(scanned_dev);
                continue;
            }
            auto remote_dev = iter->second;
            if (scanned_dev.rack != remote_dev.rack &&
                this->update_threshold_exceeded(scanned_dev.key)) {
                scanned_dev.properties = remote_dev.properties;
                scanned_dev.name = remote_dev.name;
                scanned_dev.configured = remote_dev.configured;
                to_create.push_back(scanned_dev);
                this->last_updated[scanned_dev.key] = telem::TimeStamp::now();
            }
        }
        if (to_create.empty()) return xerrors::NIL;
        return this->client->create_devices(to_create);
    }

    std::string name() override { return this->task_name; }

    void stop(bool will_reconfigure) override { pipeline::Base::stop(); }
};
}
