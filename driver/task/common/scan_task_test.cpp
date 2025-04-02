// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// external
#include "gtest/gtest.h"

/// internal
#include "driver/task/common/scan_task.h"

#include "x/cpp/xtest/xtest.h"

class MockScanner final : public common::Scanner {
public:
    size_t scan_count = 0;
    std::vector<std::vector<synnax::Device>> devices;
    std::vector<xerrors::Error> scan_errors;

    size_t start_count = 0;
    std::vector<xerrors::Error> start_errors;

    size_t stop_count = 0;
    std::vector<xerrors::Error> stop_errors;

    MockScanner(
        const std::vector<std::vector<synnax::Device>> &devices_,
        const std::vector<xerrors::Error> &scan_errors_,
        const std::vector<xerrors::Error> &start_errors_,
        const std::vector<xerrors::Error> &stop_errors_
    ):
        devices(devices_),
        scan_errors(scan_errors_),
        start_errors(start_errors_),
        stop_errors(stop_errors_) {}

    xerrors::Error start() override {
        if (this->start_count >= start_errors.size()) return xerrors::NIL;
        return start_errors[this->start_count++];
    }

    xerrors::Error stop() override {
        if (this->stop_count >= stop_errors.size()) return xerrors::NIL;
        return stop_errors[this->stop_count++];
    }

    std::pair<std::vector<synnax::Device>, xerrors::Error>
    scan(const common::ScannerContext &ctx) override {
        std::vector<synnax::Device> devs = {};
        auto err = xerrors::NIL;
        if (this->scan_count < this->devices.size())
            devs = this->devices[this->scan_count];
        if (this->scan_count < this->scan_errors.size())
            err = this->scan_errors[this->scan_count];
        this->scan_count++;
        return {devs, err};
    }
};

class MockClusterAPI final : public common::ClusterAPI {
public:
    std::shared_ptr<std::vector<synnax::Device>> remote;
    std::shared_ptr<std::vector<synnax::Device>> created;

    MockClusterAPI(
        const std::shared_ptr<std::vector<synnax::Device>> &remote_,
        const std::shared_ptr<std::vector<synnax::Device>> &created_
    ):
        remote(remote_), created(created_) {}

    std::pair<std::vector<synnax::Device>, xerrors::Error>
    retrieve_devices(std::vector<std::string> &keys) override {
        return {*remote, xerrors::NIL};
    }

    xerrors::Error create_devices(std::vector<synnax::Device> &devs) override {
        created->insert(created->end(), devs.begin(), devs.end());
        return xerrors::NIL;
    }
};

TEST(TestScanTask, testSingleScan) {
    synnax::Device dev1;
    dev1.key = "device1";
    dev1.name = "Device 1";

    synnax::Device dev2;
    dev2.key = "device2";
    dev2.name = "Device 2";

    std::vector<std::vector<synnax::Device>> devices = {{dev1, dev2}};
    auto scanner = std::make_unique<MockScanner>(
        devices,
        std::vector<xerrors::Error>{},
        std::vector<xerrors::Error>{},
        std::vector<xerrors::Error>{}
    );

    auto remote_devices = std::make_shared<std::vector<synnax::Device>>();
    auto created_devices = std::make_shared<std::vector<synnax::Device>>();
    auto cluster_api = std::make_unique<MockClusterAPI>(
        remote_devices,
        created_devices
    );

    auto ctx = std::make_shared<task::MockContext>(nullptr);

    synnax::Task task;
    task.key = 12345;
    task.name = "Test Scan Task";

    breaker::Config breaker_config;
    telem::Rate scan_rate = telem::HZ * 1;

    common::ScanTask scan_task(
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

TEST(TestScanTask, TestNoRecreateOnExistingRemote) {
    synnax::Device dev1;
    dev1.key = "device1";
    dev1.name = "Device 1";

    synnax::Device dev2;
    dev2.key = "device2";
    dev2.name = "Device 2";

    std::vector<std::vector<synnax::Device>> devices = {{dev1, dev2}};
    auto scanner = std::make_unique<MockScanner>(
        devices,
        std::vector<xerrors::Error>{},
        std::vector<xerrors::Error>{},
        std::vector<xerrors::Error>{}
    );

    auto remote_devices = std::make_shared<std::vector<synnax::Device>>();
    remote_devices->push_back(dev1); // Device 1 already exists remotely

    auto created_devices = std::make_shared<std::vector<synnax::Device>>();
    auto cluster_api = std::make_unique<MockClusterAPI>(
        remote_devices,
        created_devices
    );

    auto ctx = std::make_shared<task::MockContext>(nullptr);

    synnax::Task task;
    task.key = 12345;
    task.name = "Test Scan Task";

    breaker::Config breaker_config;
    telem::Rate scan_rate = telem::HZ * 1;

    common::ScanTask scan_task(
        std::move(scanner),
        ctx,
        task,
        breaker_config,
        scan_rate,
        std::move(cluster_api)
    );

    ASSERT_NIL(scan_task.scan());

    EXPECT_EQ(created_devices->size(), 1);
    if (!created_devices->empty()) EXPECT_EQ((*created_devices)[0].key, "device2");
}

TEST(TestScanTask, TestRecreateWhenRackChanges) {
    synnax::Device dev1;
    dev1.key = "device1";
    dev1.name = "Device 1";
    dev1.rack = 1;
    dev1.properties = "test_properties";
    dev1.configured = true;

    synnax::Device dev1_moved = dev1;
    dev1_moved.rack = 2;
    dev1_moved.name = "cat";
    dev1_moved.properties = "";
    dev1_moved.configured = false;

    synnax::Device dev1_moved_2 = dev1;
    dev1_moved_2.rack = 3;
    dev1_moved_2.name = "dog";
    dev1_moved_2.properties = "test_properties";
    dev1_moved_2.configured = false;

    // Setup scanner with devices to be discovered
    std::vector<std::vector<synnax::Device>> devices = {{dev1_moved}, {dev1_moved_2}};
    auto scanner = std::make_unique<MockScanner>(
        devices,
        std::vector<xerrors::Error>{},
        std::vector<xerrors::Error>{},
        std::vector<xerrors::Error>{}
    );

    // Setup remote devices - device1 already exists on the remote with rack1
    auto remote_devices = std::make_shared<std::vector<synnax::Device>>();
    remote_devices->push_back(dev1);

    auto created_devices = std::make_shared<std::vector<synnax::Device>>();
    auto cluster_api = std::make_unique<MockClusterAPI>(
        remote_devices,
        created_devices
    );

    auto ctx = std::make_shared<task::MockContext>(nullptr);

    synnax::Task task;
    task.key = 12345;
    task.name = "Test Scan Task";

    breaker::Config breaker_config;
    telem::Rate scan_rate = telem::HZ * 1;

    common::ScanTask scan_task(
        std::move(scanner),
        ctx,
        task,
        breaker_config,
        scan_rate,
        std::move(cluster_api)
    );

    ASSERT_NIL(scan_task.scan());
    EXPECT_EQ(created_devices->size(), 1);
    EXPECT_EQ(created_devices->at(0).key, "device1");
    EXPECT_EQ(created_devices->at(0).rack, 2);
    EXPECT_EQ(created_devices->at(0).properties, "test_properties");
    EXPECT_TRUE(created_devices->at(0).configured);

    ASSERT_NIL(scan_task.scan());
    EXPECT_EQ(created_devices->size(), 1);
    EXPECT_EQ(created_devices->at(0).key, "device1");
    EXPECT_EQ(created_devices->at(0).rack, 2);
    EXPECT_EQ(created_devices->at(0).properties, "test_properties");
    EXPECT_TRUE(created_devices->at(0).configured);
}
