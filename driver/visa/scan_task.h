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

#include "driver/task/task.h"
#include "driver/visa/device/device.h"
#include "driver/visa/errors.h"
#include "driver/visa/api/types.h"
#include "driver/visa/api/api.h"

namespace visa {

using namespace visa_types;

/// @brief Scanner task for discovering VISA resources.
class ScanTask final : public task::Task {
    std::shared_ptr<task::Context> ctx;
    synnax::Task task;
    std::shared_ptr<device::Manager> devices;
    synnax::TaskStatus state;

public:
    explicit ScanTask(
        std::shared_ptr<task::Context> ctx,
        synnax::Task task,
        std::shared_ptr<device::Manager> devices
    ):
        ctx(std::move(ctx)),
        task(std::move(task)),
        devices(std::move(devices)) {
        this->key = this->task.key;
        this->state.details.task = this->task.key;
    }

    [[nodiscard]] std::string name() const override { return this->task.name; }

    void exec(task::Command &cmd) override {
        if (cmd.type == "scan") {
            this->scan();
        } else if (cmd.type == "test_connection") {
            this->test_connection(cmd);
        }
    }

    void stop(const bool will_reconfigure) override {}

private:
    /// @brief scans for VISA resources and reports discovered devices.
    void scan() {
        LOG(INFO) << "[visa.scanner] starting device scan";

        // Use Manager to find resources
        std::vector<std::string> resources;
        if (const auto err = devices->find_resources("?*::INSTR", resources); err) {
            LOG(ERROR) << "[visa.scanner] failed to find resources: " << err.message();
            this->state.variant = "error";
            this->state.message = "Failed to scan for VISA resources: " + err.message();
            ctx->set_status(this->state);
            return;
        }

        if (resources.empty()) {
            LOG(INFO) << "[visa.scanner] no devices found";
            this->state.variant = "success";
            this->state.message = "Scan complete. No devices found.";
            ctx->set_status(this->state);
            return;
        }

        LOG(INFO) << "[visa.scanner] found " << resources.size() << " devices";

        // Process each discovered device
        for (const auto &resource_name : resources) {
            process_device(resource_name);
        }

        this->state.variant = "success";
        this->state.message = "Scan complete. Found " + std::to_string(resources.size()) + " devices.";
        ctx->set_status(this->state);
    }

    /// @brief processes a discovered device.
    void process_device(const std::string &resource_name) {
        LOG(INFO) << "[visa.scanner] discovered: " << resource_name;

        // Query device identification using Manager
        std::string idn;
        if (const auto err = devices->query_idn(resource_name, idn); !err) {
            // Trim whitespace from idn
            idn.erase(idn.find_last_not_of(" \n\r\t") + 1);
            LOG(INFO) << "[visa.scanner] " << resource_name << " -> " << idn;
        } else {
            LOG(WARNING) << "[visa.scanner] failed to query " << resource_name
                         << ": " << err.message();
        }
    }

    /// @brief tests connection to a specific resource.
    void test_connection(const task::Command &cmd) {
        const auto resource_name = cmd.args.value("resource_name", "");
        if (resource_name.empty()) {
            this->state.key = cmd.key;
            this->state.variant = "error";
            this->state.message = "No resource name provided";
            ctx->set_status(this->state);
            return;
        }

        LOG(INFO) << "[visa.scanner] testing connection to " << resource_name;

        // Try to open the device
        const device::ConnectionConfig config{resource_name};
        auto [session, err] = devices->acquire(config);

        if (err) {
            this->state.key = cmd.key;
            this->state.variant = "error";
            this->state.message = "Connection failed: " + err.message();
            ctx->set_status(this->state);
            return;
        }

        // Try to query *IDN?
        char idn[256];
        if (err = session->query("*IDN?\n", idn, sizeof(idn)); err) {
            this->state.key = cmd.key;
            this->state.variant = "warning";
            this->state.message = "Connected, but *IDN? failed: " + err.message();
            ctx->set_status(this->state);
            return;
        }

        this->state.key = cmd.key;
        this->state.variant = "success";
        this->state.message = "Connection successful. Device: " + std::string(idn);
        ctx->set_status(this->state);
    }
};

}