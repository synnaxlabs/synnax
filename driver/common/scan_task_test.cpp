// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <thread>

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/test/test.h"

#include "driver/common/scan_task.h"
#include "driver/pipeline/mock/pipeline.h"

namespace driver::common {
TEST(MergeDeviceProperties, ScannedOverridesRemote) {
    const std::string remote = R"({"key1":"remote_value","key2":"only_remote"})";
    const std::string scanned = R"({"key1":"scanned_value","key3":"only_scanned"})";
    const auto result = merge_device_properties(remote, scanned);
    const auto parsed = nlohmann::json::parse(result);
    EXPECT_EQ(parsed["key1"], "scanned_value");
    EXPECT_EQ(parsed["key2"], "only_remote");
    EXPECT_EQ(parsed["key3"], "only_scanned");
}

TEST(MergeDeviceProperties, EmptyRemote) {
    const std::string scanned = R"({"key1":"value1"})";
    const auto result = merge_device_properties("", scanned);
    const auto parsed = nlohmann::json::parse(result);
    EXPECT_EQ(parsed["key1"], "value1");
}

TEST(MergeDeviceProperties, EmptyScanned) {
    const std::string remote = R"({"key1":"value1"})";
    const auto result = merge_device_properties(remote, "");
    const auto parsed = nlohmann::json::parse(result);
    EXPECT_EQ(parsed["key1"], "value1");
}

TEST(MergeDeviceProperties, BothEmpty) {
    const auto result = merge_device_properties("", "");
    EXPECT_EQ(result, "");
}

TEST(MergeDeviceProperties, InvalidRemoteJsonContinues) {
    const std::string scanned = R"({"key1":"value1"})";
    const auto result = merge_device_properties("not valid json", scanned);
    const auto parsed = nlohmann::json::parse(result);
    EXPECT_EQ(parsed["key1"], "value1");
}

TEST(MergeDeviceProperties, InvalidScannedJsonPreservesRemote) {
    const std::string remote = R"({"key1":"value1"})";
    const auto result = merge_device_properties(remote, "not valid json");
    const auto parsed = nlohmann::json::parse(result);
    EXPECT_EQ(parsed["key1"], "value1");
}

TEST(MergeDeviceProperties, NestedObjectsReplacedNotMerged) {
    const std::string remote = R"({"nested":{"a":"1","b":"2"}})";
    const std::string scanned = R"({"nested":{"a":"new"}})";
    const auto result = merge_device_properties(remote, scanned);
    const auto parsed = nlohmann::json::parse(result);
    EXPECT_EQ(parsed["nested"]["a"], "new");
    EXPECT_FALSE(parsed["nested"].contains("b"));
}

class MockScanner final : public Scanner {
public:
    size_t scan_count = 0;
    std::vector<std::vector<synnax::device::Device>> devices;
    std::vector<x::errors::Error> scan_errors;

    size_t start_count = 0;
    std::vector<x::errors::Error> start_errors;

    size_t stop_count = 0;
    std::vector<x::errors::Error> stop_errors;

    ScannerConfig config() const override {
        return ScannerConfig{.make = "", .log_prefix = "[mock] "};
    }

    MockScanner(
        const std::vector<std::vector<synnax::device::Device>> &devices_,
        const std::vector<x::errors::Error> &scan_errors_,
        const std::vector<x::errors::Error> &start_errors_,
        const std::vector<x::errors::Error> &stop_errors_
    ):
        devices(devices_),
        scan_errors(scan_errors_),
        start_errors(start_errors_),
        stop_errors(stop_errors_) {}

    x::errors::Error start() override {
        if (this->start_count >= start_errors.size()) return x::errors::NIL;
        return start_errors[this->start_count++];
    }

    x::errors::Error stop() override {
        if (this->stop_count >= stop_errors.size()) return x::errors::NIL;
        return stop_errors[this->stop_count++];
    }

    std::pair<std::vector<synnax::device::Device>, x::errors::Error>
    scan(const ScannerContext &ctx) override {
        std::vector<synnax::device::Device> devs = {};
        auto err = x::errors::NIL;
        if (this->scan_count < this->devices.size())
            devs = this->devices[this->scan_count];
        if (this->scan_count < this->scan_errors.size())
            err = this->scan_errors[this->scan_count];
        this->scan_count++;
        return {devs, err};
    }
};

class MockClusterAPI : public ClusterAPI {
public:
    std::shared_ptr<std::vector<synnax::device::Device>> remote;
    std::shared_ptr<std::vector<synnax::device::Device>> created;
    std::vector<std::vector<synnax::device::Status>> propagated_statuses;
    std::shared_ptr<pipeline::mock::StreamerFactory> streamer_factory;
    std::vector<synnax::channel::Channel> signal_channels;

    MockClusterAPI(
        const std::shared_ptr<std::vector<synnax::device::Device>> &remote_,
        const std::shared_ptr<std::vector<synnax::device::Device>> &created_
    ):
        remote(remote_), created(created_) {}

    std::pair<std::vector<synnax::device::Device>, x::errors::Error>
    retrieve_devices(const synnax::rack::Key &rack, const std::string &make) override {
        // Filter by make like the real implementation
        std::vector<synnax::device::Device> filtered;
        for (const auto &dev: *remote)
            if (dev.make == make) filtered.push_back(dev);
        return {filtered, x::errors::NIL};
    }

    std::pair<synnax::device::Device, x::errors::Error>
    retrieve_device(const std::string &key) override {
        for (const auto &dev: *remote)
            if (dev.key == key) return {dev, x::errors::NIL};
        return {synnax::device::Device{}, x::errors::Error("device not found")};
    }

    x::errors::Error
    create_devices(std::vector<synnax::device::Device> &devs) override {
        created->insert(created->end(), devs.begin(), devs.end());
        return x::errors::NIL;
    }

    x::errors::Error
    update_statuses(std::vector<synnax::device::Status> statuses) override {
        propagated_statuses.push_back(statuses);
        return x::errors::NIL;
    }

    std::pair<std::unique_ptr<pipeline::Streamer>, x::errors::Error>
    open_streamer(synnax::framer::StreamerConfig config) override {
        if (streamer_factory) return streamer_factory->open_streamer(config);
        return {nullptr, x::errors::NIL};
    }

    std::pair<std::vector<synnax::channel::Channel>, x::errors::Error>
    retrieve_channels(const std::vector<std::string> &names) override {
        return {signal_channels, x::errors::NIL};
    }
};

/// @brief Enhanced mock scanner that supports signal monitoring testing.
class MockScannerWithSignals final : public Scanner {
public:
    ScannerConfig scanner_config;
    std::vector<task::Command> exec_commands;
    bool exec_return_value = false;
    std::mutex mu;

    size_t scan_count = 0;
    std::vector<std::vector<synnax::device::Device>> devices;
    std::vector<x::errors::Error> scan_errors;

    explicit MockScannerWithSignals(
        const ScannerConfig &config,
        const std::vector<std::vector<synnax::device::Device>> &devices_ = {},
        const std::vector<x::errors::Error> &scan_errors_ = {}
    ):
        scanner_config(config), devices(devices_), scan_errors(scan_errors_) {}

    ScannerConfig config() const override { return scanner_config; }

    x::errors::Error start() override { return x::errors::NIL; }

    x::errors::Error stop() override { return x::errors::NIL; }

    std::pair<std::vector<synnax::device::Device>, x::errors::Error>
    scan(const ScannerContext &) override {
        std::vector<synnax::device::Device> devs = {};
        auto err = x::errors::NIL;
        if (this->scan_count < this->devices.size())
            devs = this->devices[this->scan_count];
        if (this->scan_count < this->scan_errors.size())
            err = this->scan_errors[this->scan_count];
        this->scan_count++;
        return {devs, err};
    }

    bool exec(
        task::Command &cmd,
        const synnax::task::Task &,
        const std::shared_ptr<task::Context> &
    ) override {
        std::lock_guard lock(mu);
        exec_commands.push_back(cmd);
        return exec_return_value;
    }
};

/// @brief it should scan and create new devices in the cluster.
TEST(TestScanTask, testSingleScan) {
    synnax::device::Device dev1;
    dev1.key = "device1";
    dev1.name = "Device 1";

    synnax::device::Device dev2;
    dev2.key = "device2";
    dev2.name = "Device 2";

    std::vector<std::vector<synnax::device::Device>> devices = {{dev1, dev2}};
    auto scanner = std::make_unique<MockScanner>(
        devices,
        std::vector<x::errors::Error>{},
        std::vector<x::errors::Error>{},
        std::vector<x::errors::Error>{}
    );

    auto remote_devices = std::make_shared<std::vector<synnax::device::Device>>();
    auto created_devices = std::make_shared<std::vector<synnax::device::Device>>();
    auto cluster_api = std::make_unique<MockClusterAPI>(
        remote_devices,
        created_devices
    );

    auto ctx = std::make_shared<MockContext>(nullptr);

    synnax::task::Task task;
    task.key = 12345;
    task.name = "Test Scan Task";

    x::breaker::Config breaker_config;
    x::telem::Rate scan_rate = x::telem::HERTZ * 1;

    ScanTask scan_task(
        std::move(scanner),
        ctx,
        task,
        breaker_config,
        scan_rate,
        std::move(cluster_api)
    );

    ASSERT_NIL(scan_task.scan());

    EXPECT_EQ(created_devices->size(), 2);
    if (created_devices->size() >= 2) {
        EXPECT_EQ((*created_devices)[0].key, "device1");
        EXPECT_EQ((*created_devices)[1].key, "device2");
    }
}

/// @brief it should not recreate devices that already exist on remote.
TEST(TestScanTask, TestNoRecreateOnExistingRemote) {
    synnax::device::Device dev1;
    dev1.key = "device1";
    dev1.name = "Device 1";

    synnax::device::Device dev2;
    dev2.key = "device2";
    dev2.name = "Device 2";

    std::vector<std::vector<synnax::device::Device>> devices = {{dev1, dev2}};
    auto scanner = std::make_unique<MockScanner>(
        devices,
        std::vector<x::errors::Error>{},
        std::vector<x::errors::Error>{},
        std::vector<x::errors::Error>{}
    );

    auto remote_devices = std::make_shared<std::vector<synnax::device::Device>>();
    remote_devices->push_back(dev1);

    auto created_devices = std::make_shared<std::vector<synnax::device::Device>>();
    auto cluster_api = std::make_unique<MockClusterAPI>(
        remote_devices,
        created_devices
    );

    auto ctx = std::make_shared<MockContext>(nullptr);

    synnax::task::Task task;
    task.key = 12345;
    task.name = "Test Scan Task";

    x::breaker::Config breaker_config;
    x::telem::Rate scan_rate = x::telem::HERTZ * 1;

    ScanTask scan_task(
        std::move(scanner),
        ctx,
        task,
        breaker_config,
        scan_rate,
        std::move(cluster_api)
    );

    ASSERT_NIL(scan_task.init());
    ASSERT_NIL(scan_task.scan());

    EXPECT_EQ(created_devices->size(), 1);
    if (!created_devices->empty()) { EXPECT_EQ((*created_devices)[0].key, "device2"); }
}

TEST(TestScanTask, TestRecreateWhenRackChanges) {
    const std::string user_props = R"({"user_key":"user_value"})";

    synnax::device::Device dev1;
    dev1.key = "device1";
    dev1.name = "Device 1";
    dev1.rack = 1;
    dev1.properties = user_props;
    dev1.configured = true;

    synnax::device::Device dev1_moved = dev1;
    dev1_moved.rack = 2;
    dev1_moved.name = "cat";
    dev1_moved.properties = json{};
    dev1_moved.configured = false;

    synnax::device::Device dev1_moved_2 = dev1;
    dev1_moved_2.rack = 3;
    dev1_moved_2.name = "dog";
    dev1_moved_2.properties = "";
    dev1_moved_2.configured = false;

    std::vector<std::vector<synnax::device::Device>> devices = {
        {dev1_moved},
        {dev1_moved_2}
    };
    auto scanner = std::make_unique<MockScanner>(
        devices,
        std::vector<x::errors::Error>{},
        std::vector<x::errors::Error>{},
        std::vector<x::errors::Error>{}
    );

    auto remote_devices = std::make_shared<std::vector<synnax::device::Device>>();
    remote_devices->push_back(dev1);

    auto created_devices = std::make_shared<std::vector<synnax::device::Device>>();
    auto cluster_api = std::make_unique<MockClusterAPI>(
        remote_devices,
        created_devices
    );

    auto ctx = std::make_shared<MockContext>(nullptr);

    synnax::task::Task task;
    task.key = 12345;
    task.name = "Test Scan Task";

    x::breaker::Config breaker_config;
    x::telem::Rate scan_rate = x::telem::HERTZ * 1;

    ScanTask scan_task(
        std::move(scanner),
        ctx,
        task,
        breaker_config,
        scan_rate,
        std::move(cluster_api)
    );

    ASSERT_NIL(scan_task.init());
    ASSERT_NIL(scan_task.scan());
    EXPECT_EQ(created_devices->size(), 1);
    EXPECT_EQ(created_devices->at(0).key, "device1");
    EXPECT_EQ(created_devices->at(0).rack, 2);
    EXPECT_EQ(created_devices->at(0).properties, user_props);
    EXPECT_TRUE(created_devices->at(0).configured);

    ASSERT_NIL(scan_task.scan());
    EXPECT_EQ(created_devices->size(), 1);
    EXPECT_EQ(created_devices->at(0).key, "device1");
    EXPECT_EQ(created_devices->at(0).rack, 2);
    EXPECT_EQ(created_devices->at(0).properties, user_props);
    EXPECT_TRUE(created_devices->at(0).configured);
}

TEST(TestScanTask, TestUpdateWhenLocationChanges) {
    const std::string user_props = R"({"user_key":"user_value"})";

    synnax::device::Device dev1;
    dev1.key = "device1";
    dev1.name = "Device 1";
    dev1.rack = 1;
    dev1.location = "old_location";
    dev1.properties = user_props;
    dev1.configured = true;

    synnax::device::Device dev1_renamed = dev1;
    dev1_renamed.location = "new_location";
    dev1_renamed.name = "scanner_name";
    dev1_renamed.properties = "";
    dev1_renamed.configured = false;

    std::vector<std::vector<synnax::device::Device>> devices = {{dev1_renamed}};
    auto scanner = std::make_unique<MockScanner>(
        devices,
        std::vector<x::errors::Error>{},
        std::vector<x::errors::Error>{},
        std::vector<x::errors::Error>{}
    );

    auto remote_devices = std::make_shared<std::vector<synnax::device::Device>>();
    remote_devices->push_back(dev1);

    auto created_devices = std::make_shared<std::vector<synnax::device::Device>>();
    auto cluster_api = std::make_unique<MockClusterAPI>(
        remote_devices,
        created_devices
    );

    auto ctx = std::make_shared<MockContext>(nullptr);

    synnax::task::Task task;
    task.key = 12345;
    task.name = "Test Scan Task";

    x::breaker::Config breaker_config;
    x::telem::Rate scan_rate = x::telem::HERTZ * 1;

    ScanTask scan_task(
        std::move(scanner),
        ctx,
        task,
        breaker_config,
        scan_rate,
        std::move(cluster_api)
    );

    ASSERT_NIL(scan_task.init());
    ASSERT_NIL(scan_task.scan());

    ASSERT_EQ(created_devices->size(), 1);
    EXPECT_EQ(created_devices->at(0).key, "device1");
    EXPECT_EQ(created_devices->at(0).location, "new_location");
    EXPECT_EQ(created_devices->at(0).name, "Device 1");
    EXPECT_EQ(created_devices->at(0).properties, user_props);
    EXPECT_TRUE(created_devices->at(0).configured);
}

TEST(TestScanTask, TestNoUpdateWhenLocationSame) {
    const std::string user_props = R"({"user_key":"user_value"})";

    synnax::device::Device dev1;
    dev1.key = "device1";
    dev1.name = "Device 1";
    dev1.rack = 1;
    dev1.location = "same_location";
    dev1.properties = user_props;
    dev1.configured = true;

    synnax::device::Device dev1_scanned = dev1;
    dev1_scanned.name = "scanner_name";
    dev1_scanned.properties = "";
    dev1_scanned.configured = false;

    std::vector<std::vector<synnax::device::Device>> devices = {{dev1_scanned}};
    auto scanner = std::make_unique<MockScanner>(
        devices,
        std::vector<x::errors::Error>{},
        std::vector<x::errors::Error>{},
        std::vector<x::errors::Error>{}
    );

    auto remote_devices = std::make_shared<std::vector<synnax::device::Device>>();
    remote_devices->push_back(dev1);

    auto created_devices = std::make_shared<std::vector<synnax::device::Device>>();
    auto cluster_api = std::make_unique<MockClusterAPI>(
        remote_devices,
        created_devices
    );

    auto ctx = std::make_shared<MockContext>(nullptr);

    synnax::task::Task task;
    task.key = 12345;
    task.name = "Test Scan Task";

    x::breaker::Config breaker_config;
    x::telem::Rate scan_rate = x::telem::HERTZ * 1;

    ScanTask scan_task(
        std::move(scanner),
        ctx,
        task,
        breaker_config,
        scan_rate,
        std::move(cluster_api)
    );

    ASSERT_NIL(scan_task.init());
    ASSERT_NIL(scan_task.scan());

    EXPECT_EQ(created_devices->size(), 0);
}

/// @brief it should deduplicate devices keeping the last occurrence (new slot last).
TEST(TestScanTask, TestDeduplicateKeepsLastNewSlot) {
    synnax::device::Device dev1_old;
    dev1_old.key = "device1";
    dev1_old.name = "Device 1";
    dev1_old.rack = 1;
    dev1_old.location = "old_slot";

    synnax::device::Device dev1_new = dev1_old;
    dev1_new.location = "new_slot";

    // Old slot first, new slot last -> new_slot should win
    std::vector<std::vector<synnax::device::Device>> devices = {{dev1_old, dev1_new}};
    auto scanner = std::make_unique<MockScanner>(
        devices,
        std::vector<x::errors::Error>{},
        std::vector<x::errors::Error>{},
        std::vector<x::errors::Error>{}
    );

    auto remote_devices = std::make_shared<std::vector<synnax::device::Device>>();
    auto created_devices = std::make_shared<std::vector<synnax::device::Device>>();
    auto cluster_api = std::make_unique<MockClusterAPI>(
        remote_devices,
        created_devices
    );

    auto ctx = std::make_shared<task::MockContext>(nullptr);

    synnax::task::Task task;
    task.key = 12345;
    task.name = "Test Scan Task";

    x::breaker::Config breaker_config;
    x::telem::Rate scan_rate = x::telem::HERTZ * 1;

    ScanTask scan_task(
        std::move(scanner),
        ctx,
        task,
        breaker_config,
        scan_rate,
        std::move(cluster_api)
    );

    ASSERT_NIL(scan_task.init());
    ASSERT_NIL(scan_task.scan());

    ASSERT_EQ(created_devices->size(), 1);
    EXPECT_EQ(created_devices->at(0).key, "device1");
    EXPECT_EQ(created_devices->at(0).location, "new_slot");
}

/// @brief it should deduplicate devices keeping the last occurrence (old slot last).
TEST(TestScanTask, TestDeduplicateKeepsLastOldSlot) {
    synnax::device::Device dev1_old;
    dev1_old.key = "device1";
    dev1_old.name = "Device 1";
    dev1_old.rack = 1;
    dev1_old.location = "old_slot";

    synnax::device::Device dev1_new = dev1_old;
    dev1_new.location = "new_slot";

    // New slot first, old slot last -> old_slot should win
    std::vector<std::vector<synnax::device::Device>> devices = {{dev1_new, dev1_old}};
    auto scanner = std::make_unique<MockScanner>(
        devices,
        std::vector<x::errors::Error>{},
        std::vector<x::errors::Error>{},
        std::vector<x::errors::Error>{}
    );

    auto remote_devices = std::make_shared<std::vector<synnax::device::Device>>();
    auto created_devices = std::make_shared<std::vector<synnax::device::Device>>();
    auto cluster_api = std::make_unique<MockClusterAPI>(
        remote_devices,
        created_devices
    );

    auto ctx = std::make_shared<task::MockContext>(nullptr);

    synnax::task::Task task;
    task.key = 12345;
    task.name = "Test Scan Task";

    x::breaker::Config breaker_config;
    x::telem::Rate scan_rate = x::telem::HERTZ * 1;

    ScanTask scan_task(
        std::move(scanner),
        ctx,
        task,
        breaker_config,
        scan_rate,
        std::move(cluster_api)
    );

    ASSERT_NIL(scan_task.init());
    ASSERT_NIL(scan_task.scan());

    ASSERT_EQ(created_devices->size(), 1);
    EXPECT_EQ(created_devices->at(0).key, "device1");
    EXPECT_EQ(created_devices->at(0).location, "old_slot");
}

TEST(TestScanTask, TestDeduplicateOnUpdate) {
    const std::string user_props = R"({"user_key":"user_value"})";

    synnax::device::Device existing_dev;
    existing_dev.key = "device1";
    existing_dev.name = "Device 1";
    existing_dev.rack = 1;
    existing_dev.location = "original_slot";
    existing_dev.properties = user_props;
    existing_dev.configured = true;

    synnax::device::Device dev1_old;
    dev1_old.key = "device1";
    dev1_old.name = "Scanner Name";
    dev1_old.rack = 1;
    dev1_old.location = "intermediate_slot";

    synnax::device::Device dev1_new = dev1_old;
    dev1_new.location = "final_slot";

    std::vector<std::vector<synnax::device::Device>> devices = {{dev1_old, dev1_new}};
    auto scanner = std::make_unique<MockScanner>(
        devices,
        std::vector<x::errors::Error>{},
        std::vector<x::errors::Error>{},
        std::vector<x::errors::Error>{}
    );

    auto remote_devices = std::make_shared<std::vector<synnax::device::Device>>();
    remote_devices->push_back(existing_dev);

    auto created_devices = std::make_shared<std::vector<synnax::device::Device>>();
    auto cluster_api = std::make_unique<MockClusterAPI>(
        remote_devices,
        created_devices
    );

    auto ctx = std::make_shared<task::MockContext>(nullptr);

    synnax::task::Task task;
    task.key = 12345;
    task.name = "Test Scan Task";

    x::breaker::Config breaker_config;
    x::telem::Rate scan_rate = x::telem::HERTZ * 1;

    ScanTask scan_task(
        std::move(scanner),
        ctx,
        task,
        breaker_config,
        scan_rate,
        std::move(cluster_api)
    );

    ASSERT_NIL(scan_task.init());
    ASSERT_NIL(scan_task.scan());

    ASSERT_EQ(created_devices->size(), 1);
    EXPECT_EQ(created_devices->at(0).key, "device1");
    EXPECT_EQ(created_devices->at(0).location, "final_slot");
    EXPECT_EQ(created_devices->at(0).name, "Device 1");
    EXPECT_EQ(created_devices->at(0).properties, user_props);
    EXPECT_TRUE(created_devices->at(0).configured);
}

/// @brief it should propagate device status to cluster.
TEST(TestScanTask, TestStatePropagation) {
    synnax::device::Device dev1;
    dev1.key = "device1";
    dev1.name = "Device 1";
    dev1.rack = 1;
    dev1.status.key = dev1.status_key();
    dev1.status.variant = x::status::VARIANT_SUCCESS;
    dev1.status.details.rack = 1;

    synnax::device::Device dev2;
    dev2.key = "device2";
    dev2.name = "Device 2";
    dev2.rack = 2;
    dev2.status.key = dev2.status_key();
    dev2.status.variant = x::status::VARIANT_WARNING;
    dev2.status.details.rack = 2;

    // First scan will find both devices, second scan only dev1
    std::vector<std::vector<synnax::device::Device>> devices = {{dev1, dev2}, {dev1}};
    auto scanner = std::make_unique<MockScanner>(
        devices,
        std::vector<x::errors::Error>{},
        std::vector<x::errors::Error>{},
        std::vector<x::errors::Error>{}
    );

    auto remote_devices = std::make_shared<std::vector<synnax::device::Device>>();
    auto created_devices = std::make_shared<std::vector<synnax::device::Device>>();
    auto cluster_api = std::make_unique<MockClusterAPI>(
        remote_devices,
        created_devices
    );
    auto cluster_api_ptr = cluster_api.get();

    auto ctx = std::make_shared<MockContext>(nullptr);

    synnax::task::Task task;
    task.key = 12345;
    task.name = "Test Scan Task";

    x::breaker::Config breaker_config;
    x::telem::Rate scan_rate = x::telem::HERTZ * 1;

    ScanTask scan_task(
        std::move(scanner),
        ctx,
        task,
        breaker_config,
        scan_rate,
        std::move(cluster_api)
    );

    // First scan - both devices should be available
    ASSERT_NIL(scan_task.scan());
    ASSERT_EQ(cluster_api_ptr->propagated_statuses.size(), 1);

    auto &first_states = cluster_api_ptr->propagated_statuses[0];
    ASSERT_EQ(first_states.size(), 2);

    for (size_t i = 0; i < first_states.size(); i++) {
        auto status = first_states.at(i);
        if (status.key == "device:device1") {
            ASSERT_EQ(status.variant, x::status::VARIANT_SUCCESS);
            ASSERT_EQ(status.details.rack, 1);
        } else if (status.key == "device:device2") {
            ASSERT_EQ(status.variant, x::status::VARIANT_WARNING);
            ASSERT_EQ(status.details.rack, 2);
        } else
            FAIL() << "Unexpected device key: " << status.key;
    }

    ASSERT_NIL(scan_task.scan());
    ASSERT_EQ(cluster_api_ptr->propagated_statuses.size(), 2);
    auto &second_states = cluster_api_ptr->propagated_statuses[1];
    ASSERT_EQ(second_states.size(), 2);

    for (size_t i = 0; i < second_states.size(); i++) {
        auto status = second_states.at(i);
        if (status.key == "device:device1") {
            ASSERT_EQ(status.variant, x::status::VARIANT_SUCCESS);
            ASSERT_EQ(status.details.rack, 1);
        } else if (status.key == "device:device2") {
            ASSERT_EQ(status.variant, x::status::VARIANT_WARNING);
            ASSERT_EQ(status.details.rack, 2);
            ASSERT_EQ(status.message, "Device disconnected");
        } else
            FAIL() << "Unexpected device key: " << status.key;
    }
}

/// @brief it should delegate unknown commands to scanner exec handler.
TEST(TestScanTask, testCustomCommandDelegation) {
    ScannerConfig cfg{.make = "test", .log_prefix = "[test] "};
    auto scanner = std::make_unique<MockScannerWithSignals>(cfg);
    auto scanner_ptr = scanner.get();
    scanner_ptr->exec_return_value = true;

    auto remote_devices = std::make_shared<std::vector<synnax::device::Device>>();
    auto created_devices = std::make_shared<std::vector<synnax::device::Device>>();
    auto cluster_api = std::make_unique<MockClusterAPI>(
        remote_devices,
        created_devices
    );

    auto ctx = std::make_shared<MockContext>(nullptr);

    synnax::task::Task task;
    task.key = 12345;
    task.name = "Test Scan Task";

    x::breaker::Config breaker_config;
    x::telem::Rate scan_rate = x::telem::HERTZ * 1;

    ScanTask scan_task(
        std::move(scanner),
        ctx,
        task,
        breaker_config,
        scan_rate,
        std::move(cluster_api)
    );

    // Execute a custom command that should be delegated to the scanner
    synnax::task::Command cmd{
        .task = task.key,
        .type = "custom_command",
        .key = "test_cmd",
        .args = x::json::json{{"arg", "value"}}
    };
    scan_task.exec(cmd);

    // Verify the scanner received the command
    ASSERT_EQ(scanner_ptr->exec_commands.size(), 1);
    EXPECT_EQ(scanner_ptr->exec_commands[0].type, "custom_command");
    EXPECT_EQ(scanner_ptr->exec_commands[0].key, "test_cmd");
}

/// @brief it should return expected config values from scanner.
TEST(TestScanTask, testScannerConfigReturnsExpectedValues) {
    ScannerConfig cfg{.make = "test_make"};
    MockScannerWithSignals scanner(cfg);

    auto returned_cfg = scanner.config();
    EXPECT_EQ(returned_cfg.make, "test_make");
}

/// @brief Mock scanner that captures ctx.devices for verification.
class DeviceCapturingScanner final : public Scanner {
public:
    ScannerConfig scanner_config;
    mutable std::mutex mu;
    std::vector<std::unordered_map<std::string, synnax::device::Device>>
        captured_devices;

    explicit DeviceCapturingScanner(const ScannerConfig &config):
        scanner_config(config) {}

    ScannerConfig config() const override { return scanner_config; }

    std::pair<std::vector<synnax::device::Device>, x::errors::Error>
    scan(const ScannerContext &ctx) override {
        std::lock_guard lock(mu);
        if (ctx.devices != nullptr)
            captured_devices.push_back(*ctx.devices);
        else
            captured_devices.push_back({});
        // Return devices from context (like OPC scanner does)
        std::vector<synnax::device::Device> result;
        if (ctx.devices != nullptr)
            for (const auto &[key, dev]: *ctx.devices)
                result.push_back(dev);
        return {result, x::errors::NIL};
    }

    size_t device_count() {
        std::lock_guard lock(mu);
        if (captured_devices.empty()) return 0;
        return captured_devices.back().size();
    }

    bool has_device(const std::string &key) {
        std::lock_guard lock(mu);
        if (captured_devices.empty()) return false;
        return captured_devices.back().find(key) != captured_devices.back().end();
    }
};

/// @brief it should add devices to context when device set signal arrives.
TEST(TestScanTask, testSignalMonitoringAddsDevicesToContext) {
    synnax::channel::Channel device_set_ch;
    device_set_ch.key = 100;
    device_set_ch.name = synnax::device::SET_CHANNEL;

    synnax::channel::Channel device_delete_ch;
    device_delete_ch.key = 101;
    device_delete_ch.name = synnax::device::DELETE_CHANNEL;

    // Create a device that will be "signaled" and retrieved
    synnax::device::Device signaled_dev;
    signaled_dev.key = "signaled-device";
    signaled_dev.name = "Signaled Device";
    signaled_dev.make = "test_make";
    signaled_dev.rack = 1;

    // Create the frame with device JSON on the device_set channel
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    x::telem::Frame signal_frame(1);
    x::json::json dev_json = {{"key", signaled_dev.key}};
    signal_frame.emplace(device_set_ch.key, x::telem::Series(dev_json.dump()));
    reads->push_back(std::move(signal_frame));

    auto streamer_factory = std::make_shared<pipeline::mock::StreamerFactory>(
        std::vector<x::errors::Error>{},
        std::make_shared<std::vector<pipeline::mock::StreamerConfig>>(
            std::vector{pipeline::mock::StreamerConfig{reads, nullptr, x::errors::NIL}}
        )
    );

    auto remote_devices = std::make_shared<std::vector<synnax::device::Device>>();
    remote_devices->push_back(signaled_dev);

    auto created_devices = std::make_shared<std::vector<synnax::device::Device>>();
    auto cluster_api = std::make_unique<MockClusterAPI>(
        remote_devices,
        created_devices
    );
    cluster_api->streamer_factory = streamer_factory;
    cluster_api->signal_channels = {device_set_ch, device_delete_ch};

    ScannerConfig cfg{.make = "test_make", .log_prefix = "[test] "};
    auto scanner = std::make_unique<DeviceCapturingScanner>(cfg);
    auto scanner_ptr = scanner.get();

    auto ctx = std::make_shared<MockContext>(nullptr);

    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 12345);
    task.name = "Test Scan Task";

    ScanTask scan_task(
        std::move(scanner),
        ctx,
        task,
        x::breaker::Config{},
        x::telem::HERTZ * 1,
        std::move(cluster_api)
    );

    scan_task.start();

    // Wait for signal thread to process and device to appear in ctx.devices
    ASSERT_EVENTUALLY_GE(scanner_ptr->device_count(), 1);
    EXPECT_TRUE(scanner_ptr->has_device("signaled-device"));

    scan_task.stop();
}

/// @brief it should remove devices from context when device delete signal arrives.
TEST(TestScanTask, testSignalMonitoringRemovesDevicesFromContext) {
    synnax::channel::Channel device_set_ch;
    device_set_ch.key = 100;
    device_set_ch.name = synnax::device::SET_CHANNEL;

    synnax::channel::Channel device_delete_ch;
    device_delete_ch.key = 101;
    device_delete_ch.name = synnax::device::DELETE_CHANNEL;

    // Create the frame with device key on the device_delete channel
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    x::telem::Frame signal_frame(1);
    signal_frame.emplace(
        device_delete_ch.key,
        x::telem::Series(std::string("device-to-delete"))
    );
    reads->push_back(std::move(signal_frame));

    auto streamer_factory = std::make_shared<pipeline::mock::StreamerFactory>(
        std::vector<x::errors::Error>{},
        std::make_shared<std::vector<pipeline::mock::StreamerConfig>>(
            std::vector{pipeline::mock::StreamerConfig{reads, nullptr, x::errors::NIL}}
        )
    );

    // Pre-populate remote devices so init() loads them into dev_states
    synnax::device::Device existing_dev;
    existing_dev.key = "device-to-delete";
    existing_dev.name = "Device to Delete";
    existing_dev.make = "test_make";
    existing_dev.rack = 1;

    auto remote_devices = std::make_shared<std::vector<synnax::device::Device>>();
    remote_devices->push_back(existing_dev);

    auto created_devices = std::make_shared<std::vector<synnax::device::Device>>();
    auto cluster_api = std::make_unique<MockClusterAPI>(
        remote_devices,
        created_devices
    );
    cluster_api->streamer_factory = streamer_factory;
    cluster_api->signal_channels = {device_set_ch, device_delete_ch};

    ScannerConfig cfg{.make = "test_make", .log_prefix = "[test] "};
    auto scanner = std::make_unique<DeviceCapturingScanner>(cfg);
    auto scanner_ptr = scanner.get();

    auto ctx = std::make_shared<MockContext>(nullptr);

    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 12345);
    task.name = "Test Scan Task";

    ScanTask scan_task(
        std::move(scanner),
        ctx,
        task,
        x::breaker::Config{},
        x::telem::HERTZ * 1,
        std::move(cluster_api)
    );

    scan_task.start();

    // Wait for signal thread to process and device to be removed
    ASSERT_EVENTUALLY_FALSE(scanner_ptr->has_device("device-to-delete"));

    scan_task.stop();
}

/// @brief it should filter devices by make and not add mismatched devices.
TEST(TestScanTask, testSignalMonitoringFiltersByMake) {
    synnax::channel::Channel device_set_ch;
    device_set_ch.key = 100;
    device_set_ch.name = synnax::device::SET_CHANNEL;

    synnax::channel::Channel device_delete_ch;
    device_delete_ch.key = 101;
    device_delete_ch.name = synnax::device::DELETE_CHANNEL;

    // Create a device with DIFFERENT make than the scanner
    synnax::device::Device wrong_make_dev;
    wrong_make_dev.key = "wrong-make-device";
    wrong_make_dev.name = "Wrong Make Device";
    wrong_make_dev.make = "other_make";
    wrong_make_dev.rack = 1;

    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    x::telem::Frame signal_frame(1);
    x::json::json dev_json = {{"key", wrong_make_dev.key}};
    signal_frame.emplace(device_set_ch.key, x::telem::Series(dev_json.dump()));
    reads->push_back(std::move(signal_frame));

    auto streamer_factory = std::make_shared<pipeline::mock::StreamerFactory>(
        std::vector<x::errors::Error>{},
        std::make_shared<std::vector<pipeline::mock::StreamerConfig>>(
            std::vector{pipeline::mock::StreamerConfig{reads, nullptr, x::errors::NIL}}
        )
    );

    auto remote_devices = std::make_shared<std::vector<synnax::device::Device>>();
    remote_devices->push_back(wrong_make_dev);

    auto created_devices = std::make_shared<std::vector<synnax::device::Device>>();
    auto cluster_api = std::make_unique<MockClusterAPI>(
        remote_devices,
        created_devices
    );
    cluster_api->streamer_factory = streamer_factory;
    cluster_api->signal_channels = {device_set_ch, device_delete_ch};

    // Scanner expects "test_make" but device has "other_make"
    ScannerConfig cfg{.make = "test_make", .log_prefix = "[test] "};
    auto scanner = std::make_unique<DeviceCapturingScanner>(cfg);
    auto scanner_ptr = scanner.get();

    auto ctx = std::make_shared<MockContext>(nullptr);

    synnax::task::Task task;
    task.key = synnax::task::create_key(1, 12345);
    task.name = "Test Scan Task";

    ScanTask scan_task(
        std::move(scanner),
        ctx,
        task,
        x::breaker::Config{},
        x::telem::HERTZ * 1,
        std::move(cluster_api)
    );

    scan_task.start();

    // Give time for signal to be processed
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    // Device should NOT have been added due to make mismatch
    EXPECT_FALSE(scanner_ptr->has_device("wrong-make-device"));

    scan_task.stop();
}
}
