// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <atomic>
#include <mutex>
#include <optional>
#include <ranges>
#include <set>
#include <string>
#include <thread>
#include <unordered_map>
#include <unordered_set>
#include <utility>
#include <vector>

#include "absl/log/log.h"

#include "client/cpp/device/device.h"
#include "client/cpp/task/json.gen.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/filter/types.gen.h"
#include "x/cpp/json/json.h"
#include "x/cpp/loop/loop.h"
#include "x/cpp/thread/thread.h"

#include "driver/common/status.h"
#include "driver/errors/errors.h"
#include "driver/pipeline/base.h"
#include "driver/pipeline/control.h"
#include "driver/task/task.h"

namespace driver::common {
/// @brief the default rate to scan for devices.
const auto DEFAULT_SCAN_RATE = x::telem::Rate(x::telem::SECOND * 5);

/// @brief Merges scanner-discovered properties with existing remote properties.
/// Scanner properties take precedence on conflicts, but remote properties are preserved
/// if the scanner doesn't specify them.
/// @param remote_props JSON properties from the remote/cluster.
/// @param scanned_props JSON properties from the scanner.
/// @return Merged JSON with scanner properties overriding remote on conflicts.
inline x::json::json merge_device_properties(
    const x::json::json &remote_props,
    const x::json::json &scanned_props
) {
    x::json::json merged = remote_props.is_object() ? remote_props
                                                    : x::json::json::object();
    if (scanned_props.is_object()) {
        for (const auto &[k, v]: scanned_props.items())
            merged[k] = v;
    }
    return merged;
}

/// @brief common scan task configuration shared across device discovery tasks.
/// Wraps the schema-generated scan config (rate, disabled) so the field set has a
/// single definition in the oracle schema.
struct ScanTaskConfig : ::synnax::task::ScanConfig {
    ScanTaskConfig() = default;

    explicit ScanTaskConfig(x::json::Parser &cfg):
        ::synnax::task::ScanConfig(::synnax::task::ScanConfig::parse(cfg)) {}
};

/// @brief returned by a scanner whose scan failed in part, together with the devices it
/// found. The scan task keeps the devices and reports the error as a warning.
const x::errors::Error PARTIAL_SCAN_ERROR = driver::errors::BASE_ERROR.sub(
    "partial_scan"
);

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
    [[nodiscard]] virtual ScannerConfig config() const = 0;

    /// @brief Lifecycle method called when the scan task starts.
    virtual x::errors::Error start() { return x::errors::NIL; }

    /// @brief Lifecycle method called when the scan task stops.
    virtual x::errors::Error stop() { return x::errors::NIL; }

    /// @brief Periodic scan method to discover/update devices. Returns
    /// PARTIAL_SCAN_ERROR with the devices found when part of the scan fails.
    virtual std::pair<std::vector<synnax::device::Device>, x::errors::Error>
    scan(const ScannerContext &ctx) = 0;

    /// @brief returns the data that the status of the scan task carries, such as a
    /// problem of the last scan that the user can fix from the Console. Implementations
    /// must allow a call from one thread while another runs scan().
    [[nodiscard]] virtual std::optional<x::json::json::object_t> status_data() const {
        return std::nullopt;
    }

    /// @brief handles a command that the scan task does not know. Returns true when it
    /// handled the command and answered it with its own status.
    virtual bool exec(
        synnax::task::Command &cmd,
        const synnax::task::Task &task,
        const std::shared_ptr<task::Context> &ctx
    ) {
        return false;
    }

    /// @brief Called when a device is set/updated via the signal channel.
    virtual void on_device_set(const synnax::device::Device &dev) {}
};

struct ClusterAPI {
    virtual ~ClusterAPI() = default;

    virtual std::pair<std::vector<synnax::device::Device>, x::errors::Error>
    retrieve_devices(const synnax::rack::Key &rack, const std::string &make) = 0;

    virtual std::pair<synnax::device::Device, x::errors::Error>
    retrieve_device(const std::string &key) = 0;

    /// @brief retrieves the devices that have the given keys, on any rack.
    virtual std::pair<std::vector<synnax::device::Device>, x::errors::Error>
    retrieve_devices(const std::vector<std::string> &keys) = 0;

    virtual x::errors::Error
    create_devices(std::vector<synnax::device::Device> &devs) = 0;

    /// @brief applies the actions to the device with the given key.
    virtual x::errors::Error dispatch_device(
        const std::string &key,
        const std::vector<synnax::device::Action> &actions
    ) = 0;

    virtual x::errors::Error
    update_statuses(std::vector<synnax::device::Status> statuses) = 0;

    virtual std::pair<std::unique_ptr<pipeline::Streamer>, x::errors::Error>
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
        return this->client->devices.retrieve(
            synnax::device::Where{.make = make, .rack = rack},
            {.include_status = true}
        );
    }

    std::pair<synnax::device::Device, x::errors::Error>
    retrieve_device(const std::string &key) override {
        return this->client->devices.retrieve(
            key,
            synnax::device::RetrieveOptions{.include_status = true}
        );
    }

    std::pair<std::vector<synnax::device::Device>, x::errors::Error>
    retrieve_devices(const std::vector<std::string> &keys) override {
        return this->client->devices.retrieve(keys);
    }

    x::errors::Error
    create_devices(std::vector<synnax::device::Device> &devs) override {
        return this->client->devices.create(devs);
    }

    x::errors::Error dispatch_device(
        const std::string &key,
        const std::vector<synnax::device::Action> &actions
    ) override {
        return this->client->devices.dispatch(key, actions);
    }

    x::errors::Error
    update_statuses(std::vector<synnax::device::Status> statuses) override {
        if (statuses.empty()) return x::errors::NIL;
        return this->client->statuses.set(statuses);
    }

    std::pair<std::unique_ptr<pipeline::Streamer>, x::errors::Error>
    open_streamer(synnax::framer::StreamerConfig config) override {
        auto [s, err] = this->client->telem.open_streamer(config);
        if (err) return {nullptr, err};
        return {
            std::make_unique<pipeline::SynnaxStreamer>(std::move(s)),
            x::errors::NIL
        };
    }

    std::pair<std::vector<synnax::channel::Channel>, x::errors::Error>
    retrieve_channels(const std::vector<std::string> &names) override {
        return this->client->channels.retrieve(names);
    }
};

class ScanTask final : public task::Task, public pipeline::Base {
    const synnax::task::Task task;
    x::loop::Timer timer;
    std::unique_ptr<Scanner> scanner;
    std::shared_ptr<task::Context> ctx;
    synnax::task::Status status;
    /// @brief whether the scanner answered a command with a status of its own since the
    /// last scan.
    std::atomic<bool> status_replaced = false;
    ScannerContext scanner_ctx;
    std::unique_ptr<ClusterAPI> client;
    std::unordered_map<std::string, synnax::device::Device> dev_states;
    std::string log_prefix;

    /// @brief the changes that a scan saves to one device.
    struct Update {
        /// @brief the device as the scan saves it.
        synnax::device::Device scanned;
        /// @brief the copy of the device that dev_states held before the scan. Absent
        /// when the scan looked the device up.
        std::optional<synnax::device::Device> tracked;
        /// @brief the actions that save the changes.
        std::vector<synnax::device::Action> actions;
    };

    // Signal monitoring infrastructure
    synnax::channel::Channel device_set_channel;
    synnax::channel::Channel device_delete_channel;
    std::unique_ptr<pipeline::Streamer> signal_streamer;
    std::thread signal_thread;
    std::mutex mu;

    /// @brief sends the current status, then clears the command key so later
    /// unsolicited updates do not claim it.
    void send_status() {
        this->status.time = x::telem::TimeStamp::now();
        this->ctx->set_status(this->status);
        this->status.details.cmd = driver::task::NO_COMMAND;
    }

    /// @brief sets the variant, text, and scanner data of the status, then sends it.
    void send_status(
        const std::string &variant,
        const std::string &message,
        const std::string &description = ""
    ) {
        this->status.variant = variant;
        this->status.message = message;
        this->status.description = description;
        this->status.details.data = this->scanner->status_data();
        this->send_status();
    }

    /// @brief sends a warning when the scan error changes, and a success once scans
    /// recover.
    void report_scan(const x::errors::Error &err) {
        const bool replaced = this->status_replaced.exchange(false);
        if (err) {
            if (!replaced && this->status.variant == synnax::status::VARIANT_WARNING &&
                this->status.description == err.data &&
                this->status.details.data == this->scanner->status_data())
                return;
            LOG(WARNING) << this->log_prefix << "failed to scan for devices: " << err;
            this->send_status(
                synnax::status::VARIANT_WARNING,
                "Failed to scan for devices",
                err.data
            );
        } else if (replaced || this->status.variant != synnax::status::VARIANT_SUCCESS)
            this->send_status(synnax::status::VARIANT_SUCCESS, "Scan task running");
    }

    [[nodiscard]] bool update_threshold_exceeded(const std::string &dev_key) {
        auto last_updated = x::telem::TimeStamp(0);
        if (const auto dev_state = this->dev_states.find(dev_key);
            dev_state != this->dev_states.end()) {
            if (const auto &status = dev_state->second.status; status.has_value())
                last_updated = status->time;
        }
        const auto delta = x::telem::TimeStamp::now() - last_updated;
        return delta > x::telem::SECOND * 30;
    }

    /// @brief fills the fields of a scanned device that the scan does not own from its
    /// stored version, and returns the actions that save what the scan changed.
    std::vector<synnax::device::Action>
    changes(synnax::device::Device &dev, const synnax::device::Device &stored) {
        std::vector<synnax::device::Action> actions;
        if (dev.location != stored.location) {
            LOG(INFO) << this->log_prefix << "device location changed from "
                      << stored.location << " to " << dev.location;
            actions.push_back(
                synnax::device::set({.path = {"location"}, .value = dev.location})
            );
        }

        if (dev.model != stored.model) {
            LOG(INFO) << this->log_prefix << "device model changed from "
                      << stored.model << " to " << dev.model;
            actions.push_back(
                synnax::device::set({.path = {"model"}, .value = dev.model})
            );
        }

        if (dev.rack != stored.rack) {
            if (this->update_threshold_exceeded(dev.key))
                LOG(INFO) << this->log_prefix << "taking ownership over device";
            else
                dev.rack = stored.rack;
        }

        // A scan that finds no parent keeps the stored one.
        if (!dev.parent.has_value()) dev.parent = stored.parent;
        if (dev.parent != stored.parent)
            VLOG(1) << this->log_prefix << "device parent changed for " << dev.key
                    << " from '" << stored.parent.value_or("") << "' to '"
                    << *dev.parent << "'";

        if (dev.rack != stored.rack || dev.parent != stored.parent)
            actions.push_back(
                synnax::device::set_rack({.rack = dev.rack, .parent = dev.parent})
            );

        auto merged_props = merge_device_properties(stored.properties, dev.properties);
        if (merged_props != stored.properties) {
            VLOG(1) << this->log_prefix << "device properties changed for " << dev.key;
            actions.push_back(
                synnax::device::set({.path = {"properties"}, .value = merged_props})
            );
        }
        dev.properties = merged_props;
        dev.name = stored.name;
        dev.configured = stored.configured;
        return actions;
    }

    /// @brief creates the given new devices. A failed create drops them from
    /// dev_states, so the next scan looks them up again.
    x::errors::Error create(std::vector<synnax::device::Device> &devs) {
        if (devs.empty()) return x::errors::NIL;
        if (const auto err = this->client->create_devices(devs)) {
            LOG(WARNING) << this->log_prefix << "failed to create devices: " << err;
            std::lock_guard lock(this->mu);
            for (const auto &dev: devs)
                this->dev_states.erase(dev.key);
            return err;
        }
        for (const auto &dev: devs)
            LOG(INFO) << this->log_prefix << "successfully created device " << dev.key;
        return x::errors::NIL;
    }

    /// @brief returns whether a and b hold the same fields that a scan saves.
    static bool same_saved_fields(
        const synnax::device::Device &a,
        const synnax::device::Device &b
    ) {
        return a.location == b.location && a.model == b.model && a.rack == b.rack &&
               a.parent == b.parent && a.properties == b.properties;
    }

    /// @brief sends each update. When one fails, the device in dev_states goes back to
    /// its state before the scan, so the next scan finds the changes again. A device
    /// that the Core does not hold, or that the scan looked up, drops from dev_states,
    /// so the next scan looks it up. Returns the first error.
    x::errors::Error update(const std::vector<Update> &updates) {
        x::errors::Error first_err;
        for (const auto &u: updates) {
            const auto &key = u.scanned.key;
            const auto err = this->client->dispatch_device(key, u.actions);
            if (!err) {
                LOG(INFO) << this->log_prefix << "successfully updated device " << key;
                continue;
            }
            LOG(WARNING) << this->log_prefix << "failed to update device " << key
                         << ": " << err;
            if (!first_err) first_err = err;
            std::lock_guard lock(this->mu);
            // A delete signal can drop the device while the update runs.
            const auto it = this->dev_states.find(key);
            if (it == this->dev_states.end()) continue;
            if (err.matches(x::errors::NOT_FOUND) || !u.tracked.has_value()) {
                this->dev_states.erase(it);
                continue;
            }
            // Keep a newer device that a set signal stored.
            if (!same_saved_fields(it->second, u.scanned)) continue;
            it->second.location = u.tracked->location;
            it->second.model = u.tracked->model;
            it->second.rack = u.tracked->rack;
            it->second.parent = u.tracked->parent;
            it->second.properties = u.tracked->properties;
        }
        return first_err;
    }

    /// @brief Starts signal monitoring thread for device set/delete events.
    x::errors::Error start_signal_monitoring() {
        auto [channels, err] = this->client->retrieve_channels(
            {synnax::device::SET_CHANNEL, synnax::device::DELETE_CHANNEL}
        );
        if (err) return err;

        for (const auto &ch: channels) {
            if (ch.name == synnax::device::SET_CHANNEL)
                this->device_set_channel = ch;
            else if (ch.name == synnax::device::DELETE_CHANNEL)
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
        const auto rack_key = this->task.rack;
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
                        this->dev_states[dev.key] = dev;
                        this->scanner->on_device_set(dev);
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
        const std::shared_ptr<task::Context> &ctx,
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
        this->status.name = task.name;
        this->status.owner = synnax::task::resource_id(task.key);
        this->status.details.config_hash = task.config_hash;
        this->status.variant = synnax::status::VARIANT_SUCCESS;
    }

    ScanTask(
        std::unique_ptr<Scanner> scanner,
        const std::shared_ptr<task::Context> &ctx,
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
            this->task.rack,
            this->scanner->config().make
        );
        if (ret_err) return ret_err;
        for (const auto &dev: remote_devs_vec)
            this->dev_states[dev.key] = dev;
        return x::errors::NIL;
    }

    void run() override {
        this->status.details.running = false;
        if (const auto err = this->init()) {
            this->send_status(
                synnax::status::VARIANT_ERROR,
                "Failed to start scan task",
                err.data
            );
            return;
        }

        if (const auto err = this->scanner->start()) {
            this->send_status(
                synnax::status::VARIANT_ERROR,
                "Failed to start scan task",
                err.data
            );
            return;
        }

        if (const auto err = this->start_signal_monitoring())
            LOG(WARNING) << this->log_prefix
                         << "failed to start signal monitoring: " << err;

        this->status.details.running = true;
        this->send_status(synnax::status::VARIANT_SUCCESS, "Started scan task");
        while (this->breaker.running()) {
            this->report_scan(this->scan());
            this->timer.wait(this->breaker);
        }

        this->stop_signal_monitoring();
        this->status.details.running = false;
        if (const auto err = this->scanner->stop())
            this->send_status(
                synnax::status::VARIANT_ERROR,
                "Failed to stop scan task",
                err.data
            );
        else
            this->send_status(synnax::status::VARIANT_SUCCESS, "Stopped scan task");
    }

    void stopped_with_err(const x::errors::Error &err) override {
        this->status.details.running = false;
        this->send_status(
            synnax::status::VARIANT_ERROR,
            "Failed to scan for devices",
            err.data
        );
    }

    void exec(synnax::task::Command &cmd) override {
        this->status.details.cmd = cmd.key;
        // run() writes the status that answers a start or a stop. A command that
        // changes nothing never reaches it, so the current status answers instead.
        if (cmd.type == STOP_CMD_TYPE) {
            if (!this->stop()) this->send_status();
            return;
        }
        if (cmd.type == synnax::task::START_CMD_TYPE) {
            if (!this->start()) this->send_status();
            return;
        }
        if (cmd.type == common::SCAN_CMD_TYPE) {
            if (const auto err = this->scan())
                this->send_status(
                    err.matches(PARTIAL_SCAN_ERROR) ? synnax::status::VARIANT_WARNING
                                                    : synnax::status::VARIANT_ERROR,
                    "Failed to scan for devices",
                    err.data
                );
            else
                this->send_status(synnax::status::VARIANT_SUCCESS, "Scan complete");
            return;
        }
        // The scanner answers the commands it handles with its own status.
        if (this->scanner->exec(cmd, this->task, this->ctx)) {
            this->status.details.cmd = driver::task::NO_COMMAND;
            this->status_replaced = true;
            return;
        }
        LOG(ERROR) << this->log_prefix << "unknown command type: " << cmd.type;
        this->send_status(
            synnax::status::VARIANT_ERROR,
            "Failed to execute command",
            "Unknown command type '" + cmd.type + "'."
        );
    }

    x::errors::Error scan() {
        x::errors::Error scan_err;
        std::vector<synnax::device::Device> to_create;
        std::vector<Update> to_update;
        std::vector<synnax::device::Status> statuses;
        {
            std::lock_guard lock(this->mu);

            // Step 1: Scanner produces list of devices.
            this->scanner_ctx.devices = &this->dev_states;
            auto [scanned_devs, err] = this->scanner->scan(scanner_ctx);
            if (err && !err.matches(PARTIAL_SCAN_ERROR)) return err;
            scan_err = err;
            this->scanner_ctx.count++;

            // Step 1.5: Deduplicate by key (keep last). During physical device
            // transitions NI may briefly report the same device at multiple locations.
            {
                std::unordered_set<std::string> seen;
                std::vector<synnax::device::Device> deduped;
                for (auto &dev: std::views::reverse(scanned_devs)) {
                    if (seen.contains(dev.key)) continue;
                    seen.insert(dev.key);
                    deduped.push_back(std::move(dev));
                }
                std::ranges::reverse(deduped);
                scanned_devs = std::move(deduped);
            }

            // Step 2: Retrieve the scanned devices that this task does not track. One
            // can already exist on another rack.
            std::vector<std::string> untracked;
            for (const auto &dev: scanned_devs)
                if (!this->dev_states.contains(dev.key)) untracked.push_back(dev.key);
            std::unordered_map<std::string, synnax::device::Device> stored;
            if (!untracked.empty()) {
                auto [devs, retrieve_err] = this->client->retrieve_devices(untracked);
                if (retrieve_err) return retrieve_err;
                stored = synnax::device::map_device_keys(devs);
            }

            // Step 3: Find the devices to create and the changes to existing devices.
            std::set<std::string> present;
            auto last_available = x::telem::TimeStamp::now();
            for (auto &dev: scanned_devs) {
                present.insert(dev.key);
                // Unless the device already exists on the remote, it should not
                // be configured. No exceptions.
                dev.configured = false;
                std::vector<synnax::device::Action> actions;
                std::optional<synnax::device::Device> tracked;
                if (const auto iter = this->dev_states.find(dev.key);
                    iter != this->dev_states.end()) {
                    actions = this->changes(dev, iter->second);
                    if (!actions.empty()) tracked = iter->second;
                } else if (
                    const auto found = stored.find(dev.key); found != stored.end()
                ) {
                    // The task holds no status for the device, so the update threshold
                    // lets it take the device over from another rack at once.
                    actions = this->changes(dev, found->second);
                } else {
                    to_create.push_back(dev);
                    this->dev_states[dev.key] = dev;
                    continue;
                }
                if (!dev.status.has_value()) dev.status = synnax::device::Status{};
                dev.status->time = last_available;
                if (!actions.empty())
                    to_update.push_back({
                        .scanned = dev,
                        .tracked = std::move(tracked),
                        .actions = std::move(actions),
                    });
                this->dev_states[dev.key] = dev;
            }

            for (auto &[key, dev]: this->dev_states) {
                if (present.contains(key)) continue;
                if (!dev.status.has_value())
                    dev.status = synnax::device::Status{
                        .name = dev.name,
                        .owner = synnax::device::resource_id(dev.key),
                    };
                dev.status->variant = synnax::status::VARIANT_WARNING;
                dev.status->message = "Device disconnected";
                dev.status->description = "";
                dev.status->details.connected = false;
            }

            statuses.reserve(this->dev_states.size());
            for (auto &info: this->dev_states | std::views::values)
                if (info.status.has_value()) statuses.push_back(*info.status);
        }

        if (const auto state_err = this->client->update_statuses(statuses))
            LOG(ERROR) << this->log_prefix
                       << "failed to propagate statuses: " << state_err;

        const auto create_err = this->create(to_create);
        const auto update_err = this->update(to_update);
        if (create_err) return create_err;
        if (update_err) return update_err;
        return scan_err;
    }

    std::string name() const override { return this->task.name; }

    using driver::pipeline::Base::stop;

    void stop(bool will_reconfigure) override { driver::pipeline::Base::stop(); }
};
}
