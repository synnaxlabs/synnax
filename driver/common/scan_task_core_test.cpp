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
    std::atomic<size_t> reload_count = 0;

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

    void on_device_set(const synnax::device::Device &) override {
        this->reload_count++;
    }
};

/// @brief talks to a real Core and records the key of every device it writes.
class RecordingClusterAPI final : public ClusterAPI {
    SynnaxClusterAPI api;

public:
    std::vector<std::string> written;
    /// @brief the error of the latest update_statuses call.
    x::errors::Error status_err;

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

    std::pair<std::vector<synnax::device::Device>, x::errors::Error>
    retrieve_devices(const std::vector<std::string> &keys) override {
        return this->api.retrieve_devices(keys);
    }

    x::errors::Error
    create_devices(std::vector<synnax::device::Device> &devs) override {
        for (const auto &dev: devs)
            this->written.push_back(dev.key);
        return this->api.create_devices(devs);
    }

    x::errors::Error dispatch_device(
        const std::string &key,
        const std::vector<synnax::device::Action> &actions
    ) override {
        this->written.push_back(key);
        return this->api.dispatch_device(key, actions);
    }

    x::errors::Error
    update_statuses(std::vector<synnax::device::Status> statuses) override {
        this->status_err = this->api.update_statuses(std::move(statuses));
        return this->status_err;
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
    child.parent = chassis.key;

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
    ASSERT_EVENTUALLY_GE(scanner_ptr->reload_count.load(), 2);
    const auto scans = scanner_ptr->scan_count.load();
    ASSERT_EVENTUALLY_GE(scanner_ptr->scan_count.load(), scans + 10);
    scan_task.stop();

    EXPECT_EQ(
        cluster_api_ptr->written,
        (std::vector<std::string>{chassis.key, child.key})
    );
}

/// @brief it should send the status of a device that the first scan creates, and the
/// Core should take it.
TEST(TestScanTask, testSendsTheStatusOfANewDevice) {
    const auto client = std::make_shared<synnax::Synnax>(new_test_client());
    const auto rack = ASSERT_NIL_P(client->racks.create("scan_task_status_rack"));
    synnax::device::Device dev{
        .key = "new_" + std::to_string(rack.key),
        .rack = rack.key,
        .location = "usb",
        .make = "test_make",
        .model = "test_model",
        .name = "new device",
    };
    synnax::device::Status status{
        .name = dev.name,
        .owner = synnax::device::resource_id(dev.key),
    };
    status.variant = synnax::status::VARIANT_SUCCESS;
    status.message = "Device found";
    dev.status = status;
    auto cluster_api = std::make_unique<RecordingClusterAPI>(client);
    const auto *cluster_api_ptr = cluster_api.get();
    synnax::task::Task task;
    task.key = x::uuid::create();
    task.rack = rack.key;
    task.name = "Test Scan Task";
    ScanTask scan_task(
        std::make_unique<FixedScanner>(std::vector{dev}),
        std::make_shared<task::MockContext>(client),
        task,
        x::breaker::Config{},
        x::telem::HERTZ * 100,
        std::move(cluster_api)
    );

    ASSERT_NIL(scan_task.init());
    ASSERT_NIL(scan_task.scan());
    EXPECT_NIL(cluster_api_ptr->status_err);
    EXPECT_EQ(cluster_api_ptr->written, std::vector{dev.key});
}

/// @brief returns a scan task on rack that reports the devices once per scan.
std::unique_ptr<ScanTask> new_scan_task(
    const std::shared_ptr<synnax::Synnax> &client,
    const synnax::rack::Key rack,
    std::vector<synnax::device::Device> devices
) {
    synnax::task::Task task;
    task.key = x::uuid::create();
    task.rack = rack;
    task.name = "Test Scan Task";
    return std::make_unique<ScanTask>(
        std::make_unique<FixedScanner>(std::move(devices)),
        std::make_shared<task::MockContext>(client),
        task,
        x::breaker::Config{},
        x::telem::HERTZ * 100,
        std::make_unique<SynnaxClusterAPI>(client)
    );
}

/// @brief it should move a device that another rack holds to its own rack, and keep the
/// name and configured flag of the device.
TEST(TestScanTask, testTakesOverDeviceOnOtherRack) {
    const auto client = std::make_shared<synnax::Synnax>(new_test_client());
    const auto old_rack = ASSERT_NIL_P(client->racks.create("scan_task_old_rack"));
    const auto new_rack = ASSERT_NIL_P(client->racks.create("scan_task_new_rack"));
    synnax::device::Device dev{
        .key = "taken_" + std::to_string(new_rack.key),
        .rack = old_rack.key,
        .location = "usb",
        .make = "test_make",
        .model = "test_model",
        .name = "user name",
        .configured = true,
    };
    ASSERT_NIL(client->devices.create(dev));
    auto scanned = dev;
    scanned.rack = new_rack.key;
    scanned.name = "scanner name";
    scanned.configured = false;
    const auto scan_task = new_scan_task(client, new_rack.key, {scanned});

    ASSERT_NIL(scan_task->init());
    ASSERT_NIL(scan_task->scan());
    const auto got = ASSERT_NIL_P(client->devices.retrieve(dev.key));
    EXPECT_EQ(got.rack, new_rack.key);
    EXPECT_EQ(got.name, "user name");
    EXPECT_TRUE(got.configured);
}

/// @brief it should set the location of a device that a scan finds elsewhere, and keep
/// the name a user gave the device.
TEST(TestScanTask, testUpdatesDeviceLocation) {
    const auto client = std::make_shared<synnax::Synnax>(new_test_client());
    const auto rack = ASSERT_NIL_P(client->racks.create("scan_task_location_rack"));
    synnax::device::Device dev{
        .key = "moved_" + std::to_string(rack.key),
        .rack = rack.key,
        .location = "slot-1",
        .make = "test_make",
        .model = "test_model",
        .name = "user name",
        .properties = {{"user_key", "user_value"}},
    };
    ASSERT_NIL(client->devices.create(dev));
    auto scanned = dev;
    scanned.location = "slot-2";
    scanned.name = "scanner name";
    scanned.properties = {};
    const auto scan_task = new_scan_task(client, rack.key, {scanned});

    ASSERT_NIL(scan_task->init());
    ASSERT_NIL(scan_task->scan());
    const auto got = ASSERT_NIL_P(client->devices.retrieve(dev.key));
    EXPECT_EQ(got.location, "slot-2");
    EXPECT_EQ(got.name, "user name");
    EXPECT_EQ(x::json::json(got.properties), x::json::json(dev.properties));
}
}
