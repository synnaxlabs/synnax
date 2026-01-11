// Copyright 2026 Synnax Labs, Inc.
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
#include "x/cpp/json/json.h"
#include "x/cpp/loop/loop.h"
#include "x/cpp/thread/thread.h"

#include "driver/pipeline/base.h"
#include "driver/pipeline/control.h"
#include "driver/task/common/status.h"
#include "driver/task/task.h"

namespace driver::task::common {
/// @brief the default rate to scan for devices.
const auto DEFAULT_SCAN_RATE = x::telem::Rate(x::telem::SECOND * 5);

/// @brief Base configuration for scan tasks with rate and enabled settings.
struct ScanTaskConfig {
    x::telem::Rate scan_rate = DEFAULT_SCAN_RATE;
    bool enabled = true;

    ScanTaskConfig() = default;

    explicit ScanTaskConfig(x::json::Parser &cfg):
        scan_rate(
            x::telem::Rate(cfg.field<double>(
                std::vector<std::string>{"scan_rate", "rate"},
                DEFAULT_SCAN_RATE.hz()
            ))
        ),
        enabled(cfg.field<bool>("enabled", true)) {}
};

struct ScannerContext {
    /// @brief the number of scans run before the current one.
    std::size_t count = 0;
    /// @brief Devices currently tracked by the scan task. The scanner can use this
    /// to check health or perform other device-specific operations without maintaining
    /// its own device registry.
    const std::unordered_map<std::string, synnax::device::Device> *devices = nullptr;
};

/// @brief Configuration for a scanner, defining its make and signal monitoring
/// behavior.
struct ScannerConfig {
    /// @brief The make/integration name for device filtering (e.g., "opc", "ni").
    std::string make;
    /// @brief Log prefix for this scanner (e.g., "[opc] ", "[ni] ").
    std::string log_prefix;
};

struct Scanner {
    virtual ~Scanner() = default;

    /// @brief Returns the scanner configuration.
    virtual ScannerConfig config() const = 0;

    /// @brief Lifecycle method called when the scan task starts.
    virtual x::errors::Error start() { return x::errors::NIL; }

    /// @brief Lifecycle method called when the scan task stops.
    virtual x::errors::Error stop() { return x::errors::NIL; }

    /// @brief Periodic scan method to discover/update devices.
    virtual std::pair<std::vector<synnax::device::Device>, x::errors::Error>
    scan(const ScannerContext &ctx) = 0;

    /// @brief Optional: Handle custom commands. Return true if handled.
    virtual bool exec(
        synnax::task::Command &cmd,
        const synnax::task::Task &task,
        const std::shared_ptr<driver::task::Context> &ctx
    ) {
        return false;
    }
};

struct ClusterAPI {
    virtual ~ClusterAPI() = default;

    virtual std::pair<std::vector<synnax::device::Device>, x::errors::Error>
    retrieve_devices(const synnax::rack::Key &rack, const std::string &make) = 0;

    virtual std::pair<synnax::device::Device, x::errors::Error>
    retrieve_device(const std::string &key) = 0;

    virtual x::errors::Error
    create_devices(std::vector<synnax::device::Device> &devs) = 0;

    virtual x::errors::Error
    update_statuses(std::vector<synnax::device::Status> statuses) = 0;

    virtual std::pair<std::unique_ptr<driver::pipeline::Streamer>, x::errors::Error>
    open_streamer(synnax::framer::StreamerConfig config) = 0;

    virtual std::pair<std::vector<synnax::channel::Channel>, x::errors::Error>
    retrieve_channels(const std::vector<std::string> &names) = 0;
};

struct SynnaxClusterAPI final : ClusterAPI {
    std::shared_ptr<synnax::Synnax> client;
    synnax::channel::Channel state_channel;
    std::unique_ptr<synnax::framer::Writer> state_writer;

    explicit SynnaxClusterAPI(const std::shared_ptr<synnax::Synnax> &client):
        client(client) {}

    std::pair<std::vector<synnax::device::Device>, x::errors::Error>
    retrieve_devices(const synnax::rack::Key &rack, const std::string &make) override {
        synnax::device::RetrieveRequest req;
        req.makes = {make};
        req.racks = {rack};
        req.include_status = true;
        return this->client->devices.retrieve(req);
    }

    std::pair<synnax::device::Device, x::errors::Error>
    retrieve_device(const std::string &key) override {
        return this->client->devices.retrieve(
            key,
            synnax::device::RetrieveOptions{
                .include_status = true,
            }
        );
    }

    x::errors::Error
    create_devices(std::vector<synnax::device::Device> &devs) override {
        if (devs.empty()) return x::errors::NIL;
        return this->client->devices.create(devs);
    }

    x::errors::Error
    update_statuses(std::vector<synnax::device::Status> statuses) override {
        if (statuses.empty()) return x::errors::NIL;
        return this->client->statuses.set(statuses);
    }

    std::pair<std::unique_ptr<driver::pipeline::Streamer>, x::errors::Error>
    open_streamer(synnax::framer::StreamerConfig config) override {
        auto [s, err] = this->client->telem.open_streamer(config);
        if (err) return {nullptr, err};
        return {
            std::make_unique<driver::pipeline::SynnaxStreamer>(std::move(s)),
            x::errors::NIL
        };
    }

    std::pair<std::vector<synnax::channel::Channel>, x::errors::Error>
    retrieve_channels(const std::vector<std::string> &names) override {
        return this->client->channels.retrieve(names);
    }
};

class ScanTask final : public driver::task::Task, public driver::pipeline::Base {
    const synnax::task::Task task;
    x::loop::Timer timer;
    std::unique_ptr<Scanner> scanner;
    std::shared_ptr<driver::task::Context> ctx;
    synnax::task::Status status;
    ScannerContext scanner_ctx;
    std::unique_ptr<ClusterAPI> client;
    std::unordered_map<std::string, synnax::device::Device> dev_states;
    std::string log_prefix;

    // Signal monitoring infrastructure
    synnax::channel::Channel device_set_channel;
    synnax::channel::Channel device_delete_channel;
    std::unique_ptr<driver::pipeline::Streamer> signal_streamer;
    std::thread signal_thread;
    std::mutex mu;

    [[nodiscard]] bool update_threshold_exceeded(const std::string &dev_key) {
        auto last_updated = x::telem::TimeStamp(0);
        if (const auto dev_state = this->dev_states.find(dev_key);
            dev_state != this->dev_states.end() &&
            dev_state->second.status.has_value()) {
            last_updated = dev_state->second.status->time;
        }
        const auto delta = x::telem::TimeStamp::now() - last_updated;
        return delta > x::telem::SECOND * 30;
    }

    /// @brief Starts signal monitoring thread for device set/delete events.
    x::errors::Error start_signal_monitoring() {
        auto [channels, err] = this->client->retrieve_channels(
            {synnax::device::DEVICE_SET_CHANNEL, synnax::device::DEVICE_DELETE_CHANNEL}
        );
        if (err) return err;

        for (const auto &ch: channels) {
            if (ch.name == synnax::device::DEVICE_SET_CHANNEL)
                this->device_set_channel = ch;
            else if (ch.name == synnax::device::DEVICE_DELETE_CHANNEL)
                this->device_delete_channel = ch;
        }

        auto [s, open_err] = this->client->open_streamer(
            synnax::framer::StreamerConfig{
                .channels = {device_set_channel.key, device_delete_channel.key}
            }
        );
        if (open_err) return open_err;
        this->signal_streamer = std::move(s);
        this->signal_thread = std::thread([this]() { this->signal_thread_run(); });
        VLOG(1) << this->log_prefix
                << "started signal monitoring for devices with make: "
                << this->scanner->config().make;
        return x::errors::NIL;
    }

    /// @brief Stops signal monitoring thread.
    void stop_signal_monitoring() {
        {
            std::lock_guard lock(this->mu);
            if (this->signal_streamer != nullptr) this->signal_streamer->close_send();
        }
        if (this->signal_thread.joinable()) this->signal_thread.join();
    }

    /// @brief Signal thread run loop - processes device set/delete events.
    void signal_thread_run() {
        x::thread::set_name((this->task.name + ":sig").c_str());
        const auto rack_key = synnax::task::rack_key_from_task_key(this->key);
        const auto make = this->scanner->config().make;

        do {
            auto [frame, read_err] = this->signal_streamer->read();
            if (read_err) break; // close_send() was called or stream closed.

            for (size_t i = 0; i < frame.size(); i++) {
                const auto &ch_key = frame.channels->at(i);
                const auto &series = frame.series->at(i);

                if (ch_key == this->device_set_channel.key) {
                    for (const auto &dev_json: series.strings()) {
                        auto parser = x::json::Parser(dev_json);
                        auto parsed_dev = synnax::device::Device::parse(parser);
                        if (parser.error()) {
                            LOG(WARNING)
                                << this->log_prefix
                                << "failed to parse device JSON: " << parser.error();
                            continue;
                        }
                        auto [dev, err] = this->client->retrieve_device(parsed_dev.key);
                        if (err) {
                            LOG(WARNING) << this->log_prefix
                                         << "failed to retrieve device JSON: " << err;
                            continue;
                        }
                        if (dev.make != make || dev.rack != rack_key) continue;
                        std::lock_guard lock(this->mu);
                        if (this->dev_states.find(dev.key) == this->dev_states.end())
                            this->dev_states[dev.key] = dev;
                    }
                } else if (ch_key == this->device_delete_channel.key)
                    for (const auto &dev_key: series.strings()) {
                        std::lock_guard lock(this->mu);
                        this->dev_states.erase(dev_key);
                    }
            }
        } while (true);
        std::lock_guard lock(this->mu);
        if (auto err = this->signal_streamer->close())
            LOG(ERROR) << this->log_prefix
                       << "failed to close signal streamer: " << err;
        this->signal_streamer = nullptr;
    }

public:
    ScanTask(
        std::unique_ptr<Scanner> scanner,
        const std::shared_ptr<driver::task::Context> &ctx,
        const synnax::task::Task &task,
        const x::breaker::Config &breaker_config,
        const x::telem::Rate scan_rate,
        std::unique_ptr<ClusterAPI> client
    ):
        Base(breaker_config, task.name),
        task(task),
        timer(scan_rate),
        scanner(std::move(scanner)),
        ctx(ctx),
        client(std::move(client)),
        log_prefix(this->scanner->config().log_prefix) {
        if (this->log_prefix.empty())
            throw std::invalid_argument("log_prefix must be provided in ScannerConfig");
        this->key = task.key;
        this->status.key = synnax::task::status_key(task);
        this->status.name = task.name;
        this->status.details.task = task.key;
    }

    ScanTask(
        std::unique_ptr<Scanner> scanner,
        const std::shared_ptr<driver::task::Context> &ctx,
        const synnax::task::Task &task,
        const x::breaker::Config &breaker_config,
        const x::telem::Rate scan_rate
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
    x::errors::Error init() {
        auto [remote_devs_vec, ret_err] = this->client->retrieve_devices(
            synnax::task::rack(this->task),
            this->scanner->config().make
        );
        if (ret_err) return ret_err;
        for (const auto &dev: remote_devs_vec)
            this->dev_states[dev.key] = dev;
        return x::errors::NIL;
    }

    void run() override {
        if (const auto err = this->init()) {
            this->status.variant = x::status::VARIANT_ERROR;
            this->status.message = err.message();
            this->ctx->set_status(this->status);
            return;
        }

        if (const auto err = this->scanner->start()) {
            this->status.variant = x::status::VARIANT_ERROR;
            this->status.message = err.message();
            this->ctx->set_status(this->status);
            return;
        }

        if (const auto err = this->start_signal_monitoring())
            LOG(WARNING) << this->log_prefix
                         << "failed to start signal monitoring: " << err;

        this->status.variant = x::status::VARIANT_SUCCESS;
        this->status.message = "Scan task started";
        this->ctx->set_status(this->status);
        while (this->breaker.running()) {
            if (const auto err = this->scan()) {
                this->status.variant = x::status::VARIANT_WARNING;
                this->status.message = err.message();
                this->ctx->set_status(this->status);
                LOG(WARNING) << this->log_prefix
                             << "failed to scan for devices: " << err;
            }
            this->timer.wait(this->breaker);
        }

        this->stop_signal_monitoring();
        if (const auto err = this->scanner->stop()) {
            this->status.variant = x::status::VARIANT_ERROR;
            this->status.message = err.message();
        } else {
            this->status.variant = x::status::VARIANT_SUCCESS;
            this->status.message = "scan task stopped";
        }
        this->ctx->set_status(this->status);
    }

    void exec(synnax::task::Command &cmd) override {
        this->status.details.cmd = cmd.key;
        if (cmd.type == STOP_CMD_TYPE) return this->stop(false);
        if (cmd.type == START_CMD_TYPE) {
            this->start();
            return;
        }
        if (cmd.type == driver::task::common::SCAN_CMD_TYPE) {
            const auto err = this->scan();
            this->status.variant = err ? x::status::VARIANT_ERROR
                                       : x::status::VARIANT_SUCCESS;
            this->status.message = err ? err.message() : "Scan complete";
            this->ctx->set_status(this->status);
            return;
        }
        // Delegate unknown commands to scanner
        if (this->scanner->exec(cmd, this->task, this->ctx)) return;
        LOG(ERROR) << this->log_prefix << "unknown command type: " << cmd.type;
    }

    x::errors::Error scan() {
        std::vector<synnax::device::Device> to_create;
        std::vector<synnax::device::Status> statuses;
        {
            std::lock_guard lock(this->mu);

            // Step 1: Scanner produces list of devices.
            this->scanner_ctx.devices = &this->dev_states;
            auto [scanned_devs, err] = this->scanner->scan(scanner_ctx);
            if (err) return err;
            this->scanner_ctx.count++;

            // Step 2: Track which devices are present that need to be created.
            std::set<std::string> present;
            auto last_available = x::telem::TimeStamp::now();
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
                    LOG(INFO) << this->log_prefix << "taking ownership over device";
                    scanned_dev.properties = remote_dev.properties;
                    scanned_dev.name = remote_dev.name;
                    scanned_dev.configured = remote_dev.configured;
                    to_create.push_back(scanned_dev);
                }
                if (!scanned_dev.status.has_value())
                    scanned_dev.status = synnax::device::Status{};
                scanned_dev.status->time = last_available;
                this->dev_states[scanned_dev.key] = scanned_dev;
            }

            for (auto &[key, dev]: this->dev_states) {
                if (present.find(key) != present.end()) continue;
                if (!dev.status.has_value()) dev.status = synnax::device::Status{};
                dev.status->variant = x::status::VARIANT_WARNING;
                dev.status->message = "Device disconnected";
            }

            statuses.reserve(this->dev_states.size());
            for (auto &[key, info]: this->dev_states) {
                if (info.status.has_value()) statuses.push_back(*info.status);
            }
        }

        if (const auto state_err = this->client->update_statuses(statuses))
            LOG(ERROR) << this->log_prefix
                       << "failed to propagate statuses: " << state_err;

        if (to_create.empty()) return x::errors::NIL;

        x::errors::Error last_err = x::errors::NIL;
        for (auto &device: to_create) {
            std::vector single_device = {device};
            if (const auto create_err = this->client->create_devices(single_device)) {
                LOG(WARNING) << this->log_prefix << "failed to create device "
                             << device.key << ": " << create_err;
                last_err = create_err;
            } else
                LOG(INFO) << this->log_prefix << "successfully created device "
                          << device.key;
        }
        return last_err;
    }

    std::string name() const override { return this->task.name; }

    using driver::pipeline::Base::stop;

    void stop(bool will_reconfigure) override { driver::pipeline::Base::stop(); }
};
}
