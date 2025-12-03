// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "glog/logging.h"

#include "client/cpp/hardware/hardware.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/loop/loop.h"

#include "driver/pipeline/base.h"
#include "driver/task/common/status.h"
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

    virtual xerrors::Error propagate_state(telem::Series &states) = 0;
};

struct SynnaxClusterAPI final : ClusterAPI {
    std::shared_ptr<synnax::Synnax> client;
    synnax::Channel state_channel;
    std::unique_ptr<synnax::Writer> state_writer;

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

    xerrors::Error propagate_state(telem::Series &states) override {
        if (this->state_writer == nullptr) {
            const auto [state_channel, ch_err] = this->client->channels.retrieve(
                synnax::DEVICE_STATUS_CHANNEL_NAME
            );
            if (ch_err) return ch_err;
            this->state_channel = state_channel;
            auto [w, err] = this->client->telem.open_writer(
                synnax::WriterConfig{
                    .channels = {this->state_channel.key},
                    .start = telem::TimeStamp::now(),
                }
            );
            if (err) return err;
            this->state_writer = std::make_unique<synnax::Writer>(std::move(w));
        }
        this->state_writer->write(
            synnax::Frame(this->state_channel.key, std::move(states))
        );
        return xerrors::NIL;
    }
};

struct DeviceInfo {
    synnax::Device dev;
    telem::TimeStamp last_available = telem::TimeStamp(0);
};

class ScanTask final : public task::Task, public pipeline::Base {
    const std::string task_name;
    loop::Timer timer;
    std::unique_ptr<Scanner> scanner;
    std::shared_ptr<task::Context> ctx;
    synnax::TaskStatus state;
    ScannerContext scanner_ctx;
    std::unique_ptr<ClusterAPI> client;
    std::unordered_map<std::string, DeviceInfo> dev_state;

    synnax::Channel state_channel;
    std::unique_ptr<synnax::Writer> state_writer;

    [[nodiscard]] bool update_threshold_exceeded(const std::string &dev_key) {
        auto last_updated = telem::TimeStamp(0);
        if (const auto dev_state = this->dev_state.find(dev_key);
            dev_state != this->dev_state.end()) {
            last_updated = dev_state->second.last_available;
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
        this->key = task.key;
        this->state.details.task = task.key;
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
            this->state.variant = status::variant::ERR;
            this->state.message = err.message();
            this->ctx->set_status(this->state);
            return;
        }
        this->state.variant = status::variant::SUCCESS;
        this->state.message = "scan task started";
        this->ctx->set_status(this->state);
        while (this->breaker.running()) {
            if (const auto err = this->scan()) {
                this->state.variant = status::variant::WARNING;
                this->state.message = err.message();
                this->ctx->set_status(this->state);
                LOG(WARNING) << "[scan_task] failed to scan for devices: " << err;
            }
            this->timer.wait(this->breaker);
        }
        if (const auto err = this->scanner->stop()) {
            this->state.variant = status::variant::ERR;
            this->state.message = err.message();
        } else {
            this->state.variant = status::variant::SUCCESS;
            this->state.message = "scan task stopped";
        }
        this->ctx->set_status(this->state);
    }

    void exec(task::Command &cmd) override {
        this->state.key = cmd.key;
        if (cmd.type == common::STOP_CMD_TYPE) return this->stop(false);
        if (cmd.type == common::START_CMD_TYPE)
            this->start();
        else if (cmd.type == common::SCAN_CMD_TYPE) {
            const auto err = this->scan();
            this->state.variant = status::variant::ERR;
            this->state.message = err.message();
            this->ctx->set_status(this->state);
        }
    }

    xerrors::Error scan() {
        auto [scanned_devs, err] = this->scanner->scan(scanner_ctx);
        if (err) return err;
        this->scanner_ctx.count++;

        std::vector<std::string> devices;
        for (const auto &device: scanned_devs)
            devices.push_back(device.key);
        auto [remote_devs_vec, ret_err] = this->client->retrieve_devices(devices);
        if (ret_err && !ret_err.matches(xerrors::NOT_FOUND)) return ret_err;

        auto remote_devs = synnax::map_device_keys(remote_devs_vec);

        std::vector<synnax::Device> to_create;
        std::set<std::string> present;
        auto last_available = telem::TimeStamp::now();
        for (auto &scanned_dev: scanned_devs) {
            present.insert(scanned_dev.key);
            // Unless the device already exists on the remote, it should not
            // be configured. No exceptions.
            scanned_dev.configured = false;
            auto iter = remote_devs.find(scanned_dev.key);
            if (iter == remote_devs.end()) {
                to_create.push_back(scanned_dev);
                this->dev_state[scanned_dev.key] = DeviceInfo{
                    .dev = scanned_dev,
                    .last_available = last_available
                };
                continue;
            }
            const auto remote_dev = iter->second;
            if (scanned_dev.rack != remote_dev.rack &&
                this->update_threshold_exceeded(scanned_dev.key)) {
                LOG(INFO) << "[scan_task] taking ownership over device";
                scanned_dev.properties = remote_dev.properties;
                scanned_dev.name = remote_dev.name;
                scanned_dev.configured = remote_dev.configured;
                to_create.push_back(scanned_dev);
            }
            scanned_dev.status.time = last_available;
            this->dev_state[scanned_dev.key] = DeviceInfo{
                .dev = scanned_dev,
                .last_available = last_available
            };
        }

        std::vector<std::string> to_erase;
        for (auto &[key, dev]: this->dev_state) {
            if (present.find(key) != present.end()) continue;
            this->dev_state[key].dev.status = synnax::DeviceStatus{
                .key = dev.dev.key,
                .variant = status::variant::WARNING,
                .message = "Device disconnected",
                .time = dev.last_available,
                .details = synnax::DeviceStatusDetails{
                    .rack = dev.dev.rack,
                    .device = dev.dev.key,
                },
            };
            std::vector keys{dev.dev.key};
            auto [remote_devs, err] = this->client->retrieve_devices(keys);
            if (err && !err.matches(xerrors::NOT_FOUND)) {
                LOG(WARNING) << "[scan_task] failed to retrieve device: "
                             << err.message();
                continue;
            }
            if (!remote_devs.empty() &&
                remote_devs[0].rack != synnax::rack_key_from_task_key(this->key))
                to_erase.push_back(key);
        }
        for (const auto &key: to_erase)
            this->dev_state.erase(key);
        if (const auto state_err = this->propagate_state())
            LOG(ERROR) << "[scan_task] failed to propagate state: " << state_err;

        if (to_create.empty()) return xerrors::NIL;

        xerrors::Error last_err = xerrors::NIL;
        for (auto &device: to_create) {
            std::vector<synnax::Device> single_device = {device};
            if (const auto err = this->client->create_devices(single_device)) {
                LOG(WARNING) << "[scan_task] failed to create device " << device.key
                             << ": " << err.message();
                last_err = err;
            } else {
                LOG(INFO) << "[scan_task] successfully created device " << device.key;
            }
        }
        return last_err;
    }

    xerrors::Error propagate_state() {
        std::vector<json> states;
        states.reserve(this->dev_state.size());
        for (auto &[key, info]: this->dev_state)
            states.push_back(info.dev.status.to_json());
        telem::Series s(states);
        return this->client->propagate_state(s);
    }

    std::string name() const override { return this->task_name; }

    using pipeline::Base::stop;

    void stop(bool will_reconfigure) override { pipeline::Base::stop(); }
};
}
