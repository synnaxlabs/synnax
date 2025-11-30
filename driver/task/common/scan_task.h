// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <mutex>
#include <thread>

#include "glog/logging.h"

#include "x/cpp/breaker/breaker.h"
#include "x/cpp/loop/loop.h"
#include "x/cpp/xjson/xjson.h"

#include "driver/pipeline/base.h"
#include "driver/pipeline/control.h"
#include "driver/task/common/status.h"
#include "driver/task/task.h"

namespace common {
struct ScannerContext {
    std::size_t count = 0;
};

/// @brief Configuration for a scanner, defining its make and signal monitoring
/// behavior.
struct ScannerConfig {
    /// @brief The make/integration name for device filtering (e.g., "opc", "ni").
    std::string make;
};

struct Scanner {
    virtual ~Scanner() = default;

    /// @brief Returns the scanner configuration.
    virtual ScannerConfig config() const = 0;

    /// @brief Lifecycle method called when the scan task starts.
    virtual xerrors::Error start() { return xerrors::NIL; }

    /// @brief Lifecycle method called when the scan task stops.
    virtual xerrors::Error stop() { return xerrors::NIL; }

    /// @brief Periodic scan method to discover/update devices.
    virtual std::pair<std::vector<synnax::Device>, xerrors::Error>
    scan(const ScannerContext &ctx) = 0;

    /// @brief Optional: Handle custom commands. Return true if handled.
    virtual bool exec(
        task::Command &cmd,
        const synnax::Task &task,
        const std::shared_ptr<task::Context> &ctx
    ) {
        return false;
    }

    /// @brief Optional: Called when a device matching make/rack is created/updated.
    virtual void on_device_set(const synnax::Device &dev) {}

    /// @brief Optional: Called when a device is deleted.
    virtual void on_device_delete(const std::string &key) {}
};

struct ClusterAPI {
    virtual ~ClusterAPI() = default;

    virtual std::pair<std::vector<synnax::Device>, xerrors::Error>
    retrieve_devices(const synnax::RackKey &rack, const std::string &make) = 0;

    virtual std::pair<synnax::Device, xerrors::Error>
    retrieve_device(const std::string &key) = 0;

    virtual xerrors::Error create_devices(std::vector<synnax::Device> &devs) = 0;

    virtual xerrors::Error
    update_statuses(std::vector<synnax::DeviceStatus> statuses) = 0;

    virtual std::pair<std::unique_ptr<pipeline::Streamer>, xerrors::Error>
    open_streamer(synnax::StreamerConfig config) = 0;

    virtual std::pair<std::vector<synnax::Channel>, xerrors::Error>
    retrieve_channels(const std::vector<std::string> &names) = 0;
};

struct SynnaxClusterAPI final : ClusterAPI {
    std::shared_ptr<synnax::Synnax> client;
    synnax::Channel state_channel;
    std::unique_ptr<synnax::Writer> state_writer;

    explicit SynnaxClusterAPI(const std::shared_ptr<synnax::Synnax> &client):
        client(client) {}

    std::pair<std::vector<synnax::Device>, xerrors::Error>
    retrieve_devices(const synnax::RackKey &rack, const std::string &make) override {
        synnax::DeviceRetrieveRequest req;
        req.makes = {make};
        req.racks = {rack};
        req.include_status = true;
        return this->client->devices.retrieve(req);
    }

    std::pair<synnax::Device, xerrors::Error>
    retrieve_device(const std::string &key) override {
        return this->client->devices.retrieve(
            key,
            synnax::DeviceRetrieveOptions{
                .include_status = true,
            }
        );
    }

    xerrors::Error create_devices(std::vector<synnax::Device> &devs) override {
        return this->client->devices.create(devs);
    }

    xerrors::Error
    update_statuses(std::vector<synnax::DeviceStatus> statuses) override {
        return this->client->statuses.set(statuses);
    }

    std::pair<std::unique_ptr<pipeline::Streamer>, xerrors::Error>
    open_streamer(synnax::StreamerConfig config) override {
        auto [s, err] = this->client->telem.open_streamer(config);
        if (err) return {nullptr, err};
        return {std::make_unique<pipeline::SynnaxStreamer>(std::move(s)), xerrors::NIL};
    }

    std::pair<std::vector<synnax::Channel>, xerrors::Error>
    retrieve_channels(const std::vector<std::string> &names) override {
        return this->client->channels.retrieve(names);
    }
};

class ScanTask final : public task::Task, public pipeline::Base {
    const synnax::Task task;
    loop::Timer timer;
    std::unique_ptr<Scanner> scanner;
    std::shared_ptr<task::Context> ctx;
    synnax::TaskStatus status;
    ScannerContext scanner_ctx;
    std::unique_ptr<ClusterAPI> client;
    std::unordered_map<std::string, synnax::Device> dev_states;
    std::mutex dev_states_mu;

    synnax::Channel state_channel;
    std::unique_ptr<synnax::Writer> state_writer;

    // Signal monitoring infrastructure
    synnax::Channel device_set_channel;
    synnax::Channel device_delete_channel;
    std::unique_ptr<pipeline::Streamer> signal_streamer;
    std::mutex signal_streamer_mu;
    std::thread signal_thread;

    [[nodiscard]] bool update_threshold_exceeded(const std::string &dev_key) {
        auto last_updated = telem::TimeStamp(0);
        if (const auto dev_state = this->dev_states.find(dev_key);
            dev_state != this->dev_states.end()) {
            last_updated = dev_state->second.status.time;
        }
        const auto delta = telem::TimeStamp::now() - last_updated;
        return delta > telem::SECOND * 30;
    }

    /// @brief Starts signal monitoring thread for device set/delete events.
    xerrors::Error start_signal_monitoring() {
        auto [channels, err] = this->client->retrieve_channels(
            {synnax::DEVICE_SET_CHANNEL, synnax::DEVICE_DELETE_CHANNEL}
        );
        if (err) return err;

        for (const auto &ch: channels) {
            if (ch.name == synnax::DEVICE_SET_CHANNEL)
                this->device_set_channel = ch;
            else if (ch.name == synnax::DEVICE_DELETE_CHANNEL)
                this->device_delete_channel = ch;
        }

        auto [s, open_err] = this->client->open_streamer(
            synnax::StreamerConfig{
                .channels = {device_set_channel.key, device_delete_channel.key}
            }
        );
        if (open_err) return open_err;
        this->signal_streamer = std::move(s);
        this->signal_thread = std::thread([this]() { this->signal_thread_run(); });
        LOG(INFO) << "[scan_task] started signal monitoring for devices with make: "
                  << this->scanner->config().make;
        return xerrors::NIL;
    }

    /// @brief Stops signal monitoring thread.
    void stop_signal_monitoring() {
        {
            std::lock_guard lock(this->signal_streamer_mu);
            if (this->signal_streamer != nullptr) this->signal_streamer->close_send();
        }
        if (this->signal_thread.joinable()) this->signal_thread.join();
    }

    /// @brief Signal thread run loop - processes device set/delete events.
    void signal_thread_run() {
        const auto rack_key = synnax::rack_key_from_task_key(this->key);
        const auto make = this->scanner->config().make;

        do {
            auto [frame, read_err] = this->signal_streamer->read();
            if (read_err) break; // close_send() was called

            for (size_t i = 0; i < frame.size(); i++) {
                const auto &ch_key = frame.channels->at(i);
                const auto &series = frame.series->at(i);

                if (ch_key == this->device_set_channel.key) {
                    for (const auto &dev_json: series.strings()) {
                        auto parser = xjson::Parser(dev_json);
                        auto parsed_dev = synnax::Device::parse(parser);
                        if (parser.error()) {
                            LOG(WARNING) << "[scan_task] failed to parse device JSON: "
                                         << parser.error();
                            continue;
                        }
                        auto [dev, err] = this->client->retrieve_device(parsed_dev.key);
                        if (err) {
                            LOG(WARNING)
                                << "[scan_task] failed to retrieve device JSON: "
                                << err;
                            continue;
                        }
                        if (dev.make != make || dev.rack != rack_key) continue;
                        {
                            std::lock_guard lock(this->dev_states_mu);
                            if (this->dev_states.find(dev.key) ==
                                this->dev_states.end())
                                this->dev_states[dev.key] = dev;
                        }
                        this->scanner->on_device_set(dev);
                    }
                } else if (ch_key == this->device_delete_channel.key)
                    for (const auto &dev_key: series.strings()) {
                        this->scanner->on_device_delete(dev_key);
                        std::lock_guard lock(this->dev_states_mu);
                        this->dev_states.erase(dev_key);
                    }
            }
        } while (true);
        std::lock_guard lock(this->signal_streamer_mu);
        if (auto err = this->signal_streamer->close())
            LOG(ERROR) << "[scan_task] failed to close signal streamer: " << err;
        this->signal_streamer = nullptr;
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
        Base(breaker_config),
        task(task),
        timer(scan_rate),
        scanner(std::move(scanner)),
        ctx(ctx),
        client(std::move(client)) {
        this->key = task.key;
        this->status.key = task.status_key();
        this->status.name = task.name;
        this->status.details.task = task.key;
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

    /// @brief Initializes the scan task by loading remote devices into dev_states.
    /// This is called automatically by run(), but can be called separately for testing.
    xerrors::Error init() {
        auto [remote_devs_vec, ret_err] = this->client->retrieve_devices(
            this->task.rack(),
            this->scanner->config().make
        );
        if (ret_err) return ret_err;
        for (const auto &dev: remote_devs_vec)
            this->dev_states[dev.key] = dev;
        return xerrors::NIL;
    }

    void run() override {
        if (const auto err = this->init()) {
            this->status.variant = status::variant::ERR;
            this->status.message = err.message();
            this->ctx->set_status(this->status);
            return;
        }

        if (const auto err = this->scanner->start()) {
            this->status.variant = status::variant::ERR;
            this->status.message = err.message();
            this->ctx->set_status(this->status);
            return;
        }

        if (const auto err = this->start_signal_monitoring())
            LOG(WARNING) << "[scan_task] failed to start signal monitoring: " << err;

        this->status.variant = status::variant::SUCCESS;
        this->status.message = "Scan task started";
        this->ctx->set_status(this->status);
        while (this->breaker.running()) {
            if (const auto err = this->scan()) {
                this->status.variant = status::variant::WARNING;
                this->status.message = err.message();
                this->ctx->set_status(this->status);
                LOG(WARNING) << "[scan_task] failed to scan for devices: " << err;
            }
            this->timer.wait(this->breaker);
        }

        this->stop_signal_monitoring();
        if (const auto err = this->scanner->stop()) {
            this->status.variant = status::variant::ERR;
            this->status.message = err.message();
        } else {
            this->status.variant = status::variant::SUCCESS;
            this->status.message = "scan task stopped";
        }
        this->ctx->set_status(this->status);
    }

    void exec(task::Command &cmd) override {
        this->status.details.cmd = cmd.key;
        if (cmd.type == STOP_CMD_TYPE) return this->stop(false);
        if (cmd.type == START_CMD_TYPE) {
            this->start();
            return;
        }
        if (cmd.type == common::SCAN_CMD_TYPE) {
            const auto err = this->scan();
            this->status.variant = err ? status::variant::ERR
                                       : status::variant::SUCCESS;
            this->status.message = err ? err.message() : "Scan complete";
            this->ctx->set_status(this->status);
            return;
        }
        // Delegate unknown commands to scanner
        if (this->scanner->exec(cmd, this->task, this->ctx)) return;
        LOG(ERROR) << "[scan_task] unknown command type: " << cmd.type;
    }

    xerrors::Error scan() {
        // Step 1: Scanner produces list of devices.
        auto [scanned_devs, err] = this->scanner->scan(scanner_ctx);
        if (err) return err;
        this->scanner_ctx.count++;

        // Step 2: Track which devices are present that need to be created, and
        // track currently present devices.
        std::vector<synnax::Device> to_create;
        std::vector<synnax::DeviceStatus> statuses;
        {
            std::lock_guard lock(this->dev_states_mu);
            std::set<std::string> present;
            auto last_available = telem::TimeStamp::now();
            for (auto &scanned_dev: scanned_devs) {
                present.insert(scanned_dev.key);
                // Unless the device already exists on the remote, it should not
                // be configured. No exceptions.
                scanned_dev.configured = false;
                auto iter = this->dev_states.find(scanned_dev.key);
                if (iter == this->dev_states.end()) {
                    to_create.push_back(scanned_dev);
                    this->dev_states[scanned_dev.key] = scanned_dev;
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
                this->dev_states[scanned_dev.key] = scanned_dev;
            }

            for (auto &[key, dev]: this->dev_states) {
                if (present.find(key) != present.end()) continue;
                dev.status.variant = status::variant::WARNING;
                dev.status.message = "Device disconnected";
            }

            statuses.reserve(this->dev_states.size());
            for (auto &[key, info]: this->dev_states)
                statuses.push_back(info.status);
        }

        if (const auto state_err = this->client->update_statuses(statuses))
            LOG(ERROR) << "[scan_task] failed to propagate state: " << state_err;

        if (to_create.empty()) return xerrors::NIL;

        xerrors::Error last_err = xerrors::NIL;
        for (auto &device: to_create) {
            std::vector single_device = {device};
            if (const auto create_err = this->client->create_devices(single_device)) {
                LOG(WARNING) << "[scan_task] failed to create device " << device.key
                             << ": " << create_err;
                last_err = create_err;
            } else
                LOG(INFO) << "[scan_task] successfully created device " << device.key;
        }
        return last_err;
    }

    std::string name() const override { return this->task.name; }

    using pipeline::Base::stop;

    void stop(bool will_reconfigure) override { pipeline::Base::stop(); }
};
}
