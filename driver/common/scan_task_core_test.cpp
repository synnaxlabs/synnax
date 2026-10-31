// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/common/scan_task.h"
#include "driver/task/task.h"

namespace driver::common {
/// @brief reports the same devices on every scan.
class FixedScanner final : public Scanner {
public:
    std::vector<synnax::device::Device> devices;
    std::atomic<size_t> scan_count = 0;

    explicit FixedScanner(std::vector<synnax::device::Device> devices):
        devices(std::move(devices)) {}

    [[nodiscard]] ScannerConfig config() const override {
        return {.make = "test_make", .log_prefix = "[test] "};
    }

    std::pair<std::vector<synnax::device::Device>, x::errors::Error>
    scan(const ScannerContext &) override {
        this->scan_count++;
        return {this->devices, x::errors::NIL};
    }
};

/// @brief talks to a real Core and records the key of every device it writes.
class RecordingClusterAPI final : public ClusterAPI {
    SynnaxClusterAPI api;

public:
    std::vector<std::string> written;

    explicit RecordingClusterAPI(const std::shared_ptr<synnax::Synnax> &client):
        api(client) {}

    std::pair<std::vector<synnax::device::Device>, x::errors::Error>
    retrieve_devices(const synnax::rack::Key &rack, const std::string &make) override {
        return this->api.retrieve_devices(rack, make);
    }

    std::pair<synnax::device::Device, x::errors::Error>
    retrieve_device(const std::string &key) override {
        return this->api.retrieve_device(key);
    }

    x::errors::Error
    create_devices(std::vector<synnax::device::Device> &devs) override {
        for (const auto &dev: devs)
            this->written.push_back(dev.key);
        return this->api.create_devices(devs);
    }

    x::errors::Error
    update_statuses(std::vector<synnax::device::Status> statuses) override {
        return this->api.update_statuses(std::move(statuses));
    }

    std::pair<std::unique_ptr<pipeline::Streamer>, x::errors::Error>
    open_streamer(synnax::framer::StreamerConfig config) override {
        return this->api.open_streamer(std::move(config));
    }

    std::pair<std::vector<synnax::channel::Channel>, x::errors::Error>
    retrieve_channels(const std::vector<std::string> &names) override {
        return this->api.retrieve_channels(names);
    }
};

/// @brief it should write each device once, even as set signals reload it.
TEST(TestScanTask, testWritesDeviceHierarchyOnce) {
    const auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = synnax::rack::Rack{.name = "scan_task_test_rack"};
    ASSERT_NIL(client->racks.create(rack));

    synnax::device::Device chassis{
        .key = "chassis_" + std::to_string(rack.key),
        .rack = rack.key,
        .location = "chassis",
        .make = "test_make",
        .model = "test_model",
        .name = "chassis",
    };
    synnax::device::Device child = chassis;
    child.key = "child_" + std::to_string(rack.key);
    child.location = "child";
    child.parent = synnax::device::ontology_id(chassis.key);

    auto scanner = std::make_unique<FixedScanner>(
        std::vector<synnax::device::Device>{chassis, child}
    );
    const auto *scanner_ptr = scanner.get();
    auto cluster_api = std::make_unique<RecordingClusterAPI>(client);
    const auto *cluster_api_ptr = cluster_api.get();

    synnax::task::Task task;
    task.key = x::uuid::create();
    task.rack = rack.key;
    task.name = "Test Scan Task";

    ScanTask scan_task(
        std::move(scanner),
        std::make_shared<task::MockContext>(client),
        task,
        x::breaker::Config{},
        x::telem::HERTZ * 100,
        std::move(cluster_api)
    );

    scan_task.start();
    ASSERT_EVENTUALLY_GE(scanner_ptr->scan_count.load(), 50);
    scan_task.stop();

    EXPECT_EQ(
        cluster_api_ptr->written,
        (std::vector<std::string>{chassis.key, child.key})
    );
}
}
