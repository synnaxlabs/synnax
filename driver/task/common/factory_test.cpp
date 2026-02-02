// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/test/test.h"

#include "driver/task/common/factory.h"
#include "driver/task/task.h"

namespace driver::task::common {
class MockTask final : public driver::task::Task {
public:
    explicit MockTask() = default;

    void exec(synnax::task::Command &cmd) override {}
    void stop(bool will_reconfigure) override {}
    [[nodiscard]] std::string name() const override { return "mock_task"; }
};

class MockFactory {
public:
    std::pair<std::unique_ptr<driver::task::Task>, x::errors::Error> configure_task(
        const std::shared_ptr<driver::task::Context> &ctx,
        const synnax::task::Task &task
    ) {
        configured_tasks.push_back(task);
        if (should_fail) { return {nullptr, x::errors::Error("mock", "mock error")}; }
        return {std::make_unique<MockTask>(), x::errors::NIL};
    }

    std::vector<synnax::task::Task> configured_tasks;
    bool should_fail = false;
};

/// @brief it should create a new task when type does not exist on rack.
TEST(TestFactory, TestCreateIfTypeNotExistsOnRack_NewTask) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("test_rack"));
    synnax::task::Task task{.name = "test_task", .type = "test_type"};
    auto created = ASSERT_NIL_P(create_if_type_not_exists_on_rack(rack, task));
    ASSERT_TRUE(created);
    ASSERT_NE(task.key, 0);
    ASSERT_EQ(synnax::task::rack_key_from_task_key(task.key), rack.key);
    ASSERT_EQ(task.name, "test_task");
    ASSERT_EQ(task.type, "test_type");
}

/// @brief it should not create task when type already exists on rack.
TEST(TestFactory, TestCreateIfTypeNotExistsOnRack_ExistingTask) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("test_rack"));
    synnax::task::Task existing_task{.name = "existing_task", .type = "test_type"};
    ASSERT_NIL(rack.tasks.create(existing_task));
    synnax::task::Task new_task{.name = "new_task", .type = "test_type"};
    auto created = ASSERT_NIL_P(create_if_type_not_exists_on_rack(rack, new_task));
    ASSERT_FALSE(created);
    ASSERT_EQ(synnax::task::local_task_key(new_task.key), 0);
}

/// @brief it should successfully configure initial factory tasks.
TEST(TestFactory, TestConfigureInitialFactoryTasks_Success) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    const auto rack = ASSERT_NIL_P(client->racks.create("test_rack"));
    const auto ctx = std::make_shared<driver::task::MockContext>(client);
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

/// @brief it should skip configuration when task type already exists.
TEST(TestFactory, TestConfigureInitialFactoryTasks_ExistingTask) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("test_rack"));
    auto ctx = std::make_shared<driver::task::MockContext>(client);
    auto factory = std::make_unique<MockFactory>();
    synnax::task::Task existing_task{.name = "existing_task", .type = "test_type"};
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

/// @brief it should return empty tasks list when configuration fails.
TEST(TestFactory, TestConfigureInitialFactoryTasks_ConfigurationFailure) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("test_rack"));
    auto ctx = std::make_shared<driver::task::MockContext>(client);
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

/// @brief it should configure multiple tasks of different types.
TEST(TestFactory, TestConfigureInitialFactoryTasks_MultipleConfigurations) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("test_rack"));
    auto ctx = std::make_shared<driver::task::MockContext>(client);
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

/// @brief it should delete legacy task by type successfully.
TEST(TestFactory, TestDeleteLegacyTaskByType_Success) {
    const auto client = std::make_shared<synnax::Synnax>(new_test_client());
    const auto rack = ASSERT_NIL_P(client->racks.create("test_rack"));
    synnax::task::Task legacy_task{.name = "legacy_task", .type = "legacy_type"};
    ASSERT_NIL(rack.tasks.create(legacy_task));
    ASSERT_NIL(delete_legacy_task_by_type(rack, "legacy_type", "test_integration"));
    ASSERT_OCCURRED_AS_P(
        rack.tasks.retrieve_by_type("legacy_type"),
        x::errors::NOT_FOUND
    );
}

/// @brief it should handle non-existent legacy task type gracefully.
TEST(TestFactory, TestDeleteLegacyTaskByType_NonExistent) {
    const auto client = std::make_shared<synnax::Synnax>(new_test_client());
    const auto rack = ASSERT_NIL_P(client->racks.create("test_rack"));
    ASSERT_NIL(
        delete_legacy_task_by_type(rack, "non_existent_type", "test_integration")
    );
}
}
