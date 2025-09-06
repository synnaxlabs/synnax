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
#include "client/cpp/testutil/testutil.h"
#include "driver/task/common/factory.h"
#include "driver/task/task.h"

/// module
#include "x/cpp/xtest/xtest.h"

namespace common {
class MockTask final : public task::Task {
public:
    explicit MockTask() = default;

    void exec(task::Command &cmd) override {}
    void stop(bool will_reconfigure) override {}
    [[nodiscard]] std::string name() const override { return "mock_task"; }
};

class MockFactory {
public:
    std::pair<std::unique_ptr<task::Task>, xerrors::Error> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) {
        configured_tasks.push_back(task);
        if (should_fail) { return {nullptr, xerrors::Error("mock", "mock error")}; }
        return {std::make_unique<MockTask>(), xerrors::NIL};
    }

    std::vector<synnax::Task> configured_tasks;
    bool should_fail = false;
};

TEST(TestFactory, TestCreateIfTypeNotExistsOnRack_NewTask) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->hardware.create_rack("test_rack"));
    synnax::Task task(rack.key, "test_task", "test_type", "");
    auto created = ASSERT_NIL_P(create_if_type_not_exists_on_rack(rack, task));
    ASSERT_TRUE(created);
    ASSERT_NE(task.key, 0);
    ASSERT_EQ(synnax::rack_key_from_task_key(task.key), rack.key);
    ASSERT_EQ(task.name, "test_task");
    ASSERT_EQ(task.type, "test_type");
}

TEST(TestFactory, TestCreateIfTypeNotExistsOnRack_ExistingTask) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->hardware.create_rack("test_rack"));
    synnax::Task existing_task(rack.key, "existing_task", "test_type", "");
    ASSERT_NIL(rack.tasks.create(existing_task));
    synnax::Task new_task(rack.key, "new_task", "test_type", "");
    auto created = ASSERT_NIL_P(create_if_type_not_exists_on_rack(rack, new_task));
    ASSERT_FALSE(created);
    ASSERT_EQ(synnax::local_task_key(new_task.key), 0);
}

TEST(TestFactory, TestConfigureInitialFactoryTasks_Success) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    const auto rack = ASSERT_NIL_P(client->hardware.create_rack("test_rack"));
    const auto ctx = std::make_shared<task::MockContext>(client);
    const auto factory = std::make_unique<MockFactory>();
    auto tasks = configure_initial_factory_tasks(
        factory.get(),
        ctx,
        rack,
        "test_task",
        "test_type",
        "test_integration"
    );
    ASSERT_EQ(tasks.size(), 1);
    ASSERT_EQ(tasks[0].first.name, "test_task");
    ASSERT_EQ(tasks[0].first.type, "test_type");
    ASSERT_NE(tasks[0].second, nullptr);
    ASSERT_EQ(factory->configured_tasks.size(), 1);
}

TEST(TestFactory, TestConfigureInitialFactoryTasks_ExistingTask) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->hardware.create_rack("test_rack"));
    auto ctx = std::make_shared<task::MockContext>(client);
    auto factory = std::make_unique<MockFactory>();
    synnax::Task existing_task(rack.key, "existing_task", "test_type", "");
    ASSERT_NIL(rack.tasks.create(existing_task));
    auto tasks = configure_initial_factory_tasks(
        factory.get(),
        ctx,
        rack,
        "new_task",
        "test_type",
        "test_integration"
    );
    ASSERT_TRUE(tasks.empty());
    ASSERT_TRUE(factory->configured_tasks.empty());
}

TEST(TestFactory, TestConfigureInitialFactoryTasks_ConfigurationFailure) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->hardware.create_rack("test_rack"));
    auto ctx = std::make_shared<task::MockContext>(client);
    auto factory = std::make_unique<MockFactory>();
    factory->should_fail = true;
    auto tasks = configure_initial_factory_tasks(
        factory.get(),
        ctx,
        rack,
        "test_task",
        "test_type",
        "test_integration"
    );
    ASSERT_TRUE(tasks.empty());
    ASSERT_EQ(factory->configured_tasks.size(), 1);
}

TEST(TestFactory, TestConfigureInitialFactoryTasks_MultipleConfigurations) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->hardware.create_rack("test_rack"));
    auto ctx = std::make_shared<task::MockContext>(client);
    auto factory = std::make_unique<MockFactory>();
    auto tasks1 = configure_initial_factory_tasks(
        factory.get(),
        ctx,
        rack,
        "task1",
        "type1",
        "test_integration"
    );
    ASSERT_EQ(tasks1.size(), 1);
    ASSERT_EQ(tasks1[0].first.name, "task1");
    ASSERT_EQ(tasks1[0].first.type, "type1");
    auto tasks2 = configure_initial_factory_tasks(
        factory.get(),
        ctx,
        rack,
        "task2",
        "type2",
        "test_integration"
    );
    ASSERT_EQ(tasks2.size(), 1);
    ASSERT_EQ(tasks2[0].first.name, "task2");
    ASSERT_EQ(tasks2[0].first.type, "type2");
    ASSERT_EQ(factory->configured_tasks.size(), 2);
}

TEST(TestFactory, TestDeleteLegacyTaskByType_Success) {
    const auto client = std::make_shared<synnax::Synnax>(new_test_client());
    const auto rack = ASSERT_NIL_P(client->hardware.create_rack("test_rack"));
    synnax::Task legacy_task(rack.key, "legacy_task", "legacy_type", "");
    ASSERT_NIL(rack.tasks.create(legacy_task));
    ASSERT_NIL(delete_legacy_task_by_type(rack, "legacy_type", "test_integration"));
    ASSERT_OCCURRED_AS_P(
        rack.tasks.retrieve_by_type("legacy_type"),
        xerrors::NOT_FOUND
    );
}

TEST(TestFactory, TestDeleteLegacyTaskByType_NonExistent) {
    const auto client = std::make_shared<synnax::Synnax>(new_test_client());
    const auto rack = ASSERT_NIL_P(client->hardware.create_rack("test_rack"));
    ASSERT_NIL(
        delete_legacy_task_by_type(rack, "non_existent_type", "test_integration")
    );
}
}
