// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/test/xtest.h"

std::mt19937 gen_rand_task = random_generator("Task Tests");

namespace synnax {
/// @brief it should correctly create a module on the rack.
TEST(TaskTests, testCreateTask) {
    auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.racks.create(r));
    auto m = Task(r.key, "test_module", "mock", "config", false, true);
    ASSERT_NIL(r.tasks.create(m));
    ASSERT_EQ(m.name, "test_module");
    ASSERT_EQ(synnax::rack_key_from_task_key(m.key), r.key);
    ASSERT_NE(synnax::local_task_key(m.key), 0);
}

/// @brief it should correctly retrieve a module from the rack.
TEST(TaskTests, testRetrieveTask) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.racks.create(r));
    auto t = Task(r.key, "test_module", "mock", "config", false, true);
    ASSERT_NIL(r.tasks.create(t));
    const auto t2 = ASSERT_NIL_P(r.tasks.retrieve(t.key));
    ASSERT_EQ(t2.name, "test_module");
    ASSERT_EQ(synnax::rack_key_from_task_key(t.key), r.key);
    ASSERT_EQ(synnax::local_task_key(t2.key), synnax::local_task_key(t.key));
    ASSERT_TRUE(t2.snapshot);
}

/// @brief it should retrieve a task by its name
TEST(TaskTests, testRetrieveTaskByName) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.racks.create(r));
    const auto rand_name = std::to_string(gen_rand_task());
    auto t = Task(r.key, rand_name, "mock", "config");
    ASSERT_NIL(r.tasks.create(t));
    const auto t2 = ASSERT_NIL_P(r.tasks.retrieve(rand_name));
    ASSERT_EQ(t2.name, rand_name);
    ASSERT_EQ(synnax::rack_key_from_task_key(t.key), r.key);
}

/// @brief it should retrieve a task by its type
TEST(TaskTests, testRetrieveTaskByType) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.racks.create(r));
    const auto rand_type = std::to_string(gen_rand_task());
    auto t = Task(r.key, "test_module", rand_type, "config");
    ASSERT_NIL(r.tasks.create(t));
    const auto t2 = ASSERT_NIL_P(r.tasks.retrieve_by_type(rand_type));
    ASSERT_EQ(t2.name, "test_module");
    ASSERT_EQ(synnax::rack_key_from_task_key(t.key), r.key);
}

/// @brief it should correctly list the tasks on a rack.
TEST(TaskTests, testListTasks) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.racks.create(r));
    auto m = Task(r.key, "test_module", "mock", "config");
    ASSERT_NIL(r.tasks.create(m));
    const auto tasks = ASSERT_NIL_P(r.tasks.list());
    ASSERT_EQ(tasks.size(), 1);
    ASSERT_EQ(tasks[0].name, "test_module");
    ASSERT_EQ(synnax::rack_key_from_task_key(tasks[0].key), r.key);
    ASSERT_NE(synnax::local_task_key(tasks[0].key), 0);
}

/// @brief it should correctly delete a task from the rack.
TEST(TaskTests, testDeleteTask) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.racks.create(r));
    auto t = Task(r.key, "test_module", "mock", "config");
    ASSERT_NIL(r.tasks.create(t));
    ASSERT_NIL(r.tasks.del(t.key));
    ASSERT_OCCURRED_AS_P(r.tasks.retrieve(t.key), x::errors::NOT_FOUND);
}

/// @brief it should convert a task key to an ontology ID
TEST(TaskTests, testTaskOntologyId) {
    constexpr synnax::TaskKey key = 12345678901234;
    const auto id = synnax::task_ontology_id(key);
    ASSERT_EQ(id.type, "task");
    ASSERT_EQ(id.key, "12345678901234");
}

/// @brief it should convert multiple task keys to ontology IDs
TEST(TaskTests, testTaskOntologyIds) {
    const std::vector<synnax::TaskKey> keys = {100, 200, 300};
    const auto ids = synnax::task_ontology_ids(keys);
    ASSERT_EQ(ids.size(), 3);
    ASSERT_EQ(ids[0].type, "task");
    ASSERT_EQ(ids[0].key, "100");
    ASSERT_EQ(ids[1].type, "task");
    ASSERT_EQ(ids[1].key, "200");
    ASSERT_EQ(ids[2].type, "task");
    ASSERT_EQ(ids[2].key, "300");
}

/// @brief it should return empty vector for empty input
TEST(TaskTests, testTaskOntologyIdsEmpty) {
    const std::vector<synnax::TaskKey> keys;
    const auto ids = synnax::task_ontology_ids(keys);
    ASSERT_TRUE(ids.empty());
}

/// @brief it should correctly create and retrieve a task with a status.
TEST(TaskTests, testCreateTaskWithStatus) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.racks.create(r));
    auto t = Task(r.key, "test_task_with_status", "mock", "config");
    t.status = TaskStatus{};
    t.status->key = "task-status-key";
    t.status->variant = synnax::status::variant_success;
    t.status->message = "Task is running";
    t.status->time = x::telem::TimeStamp::now();
    t.status->details.task = 0;
    t.status->details.running = true;
    t.status->details.cmd = "start";
    ASSERT_NIL(r.tasks.create(t));
    const auto t2 = ASSERT_NIL_P(r.tasks.retrieve(t.key, {.include_status = true}));
    ASSERT_EQ(t2.name, "test_task_with_status");
    ASSERT_TRUE(t2.status.has_value());
    ASSERT_EQ(t2.status->variant, synnax::status::variant_success);
    ASSERT_EQ(t2.status->message, "Task is running");
    ASSERT_EQ(t2.status->details.running, true);
    ASSERT_EQ(t2.status->details.cmd, "start");
}

/// @brief it should correctly retrieve a task with status by name.
TEST(TaskTests, testRetrieveTaskWithStatusByName) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.racks.create(r));
    const auto rand_name = std::to_string(gen_rand_task());
    auto t = Task(r.key, rand_name, "mock", "config");
    t.status = TaskStatus{};
    t.status->key = "task-status-by-name";
    t.status->variant = synnax::status::variant_warning;
    t.status->message = "Task warning";
    t.status->time = x::telem::TimeStamp::now();
    ASSERT_NIL(r.tasks.create(t));
    const auto t2 = ASSERT_NIL_P(r.tasks.retrieve(rand_name, {.include_status = true}));
    ASSERT_EQ(t2.name, rand_name);
    ASSERT_TRUE(t2.status.has_value());
    ASSERT_EQ(t2.status->variant, synnax::status::variant_warning);
    ASSERT_EQ(t2.status->message, "Task warning");
}

/// @brief it should correctly list tasks with statuses.
TEST(TaskTests, testListTasksWithStatus) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.racks.create(r));
    auto t = Task(r.key, "test_task_list_status", "mock", "config");
    t.status = TaskStatus{};
    t.status->key = "task-list-status";
    t.status->variant = synnax::status::variant_info;
    t.status->message = "Task info";
    t.status->time = x::telem::TimeStamp::now();
    ASSERT_NIL(r.tasks.create(t));
    const auto tasks = ASSERT_NIL_P(r.tasks.list({.include_status = true}));
    ASSERT_EQ(tasks.size(), 1);
    ASSERT_TRUE(tasks[0].status.has_value());
    ASSERT_EQ(tasks[0].status->variant, synnax::status::variant_info);
    ASSERT_EQ(tasks[0].status->message, "Task info");
}
/// @brief it should retrieve multiple tasks by their names.
TEST(TaskTests, testRetrieveTasksByNames) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.racks.create(r));
    const auto rand1 = std::to_string(gen_rand_task());
    const auto rand2 = std::to_string(gen_rand_task());
    auto t1 = Task(r.key, rand1, "mock", "config1");
    auto t2 = Task(r.key, rand2, "mock", "config2");
    ASSERT_NIL(r.tasks.create(t1));
    ASSERT_NIL(r.tasks.create(t2));
    const std::vector<std::string> names = {rand1, rand2};
    const auto tasks = ASSERT_NIL_P(r.tasks.retrieve(names));
    ASSERT_EQ(tasks.size(), 2);
    bool found1 = false, found2 = false;
    for (const auto &t: tasks) {
        if (t.name == rand1) found1 = true;
        if (t.name == rand2) found2 = true;
    }
    ASSERT_TRUE(found1);
    ASSERT_TRUE(found2);
}
/// @brief it should retrieve multiple tasks by their types.
TEST(TaskTests, testRetrieveTasksByTypes) {
    const auto client = new_test_client();
    auto r = Rack("test_rack");
    ASSERT_NIL(client.racks.create(r));
    const auto type1 = std::to_string(gen_rand_task());
    const auto type2 = std::to_string(gen_rand_task());
    auto t1 = Task(r.key, "task_by_type_1", type1, "config1");
    auto t2 = Task(r.key, "task_by_type_2", type2, "config2");
    ASSERT_NIL(r.tasks.create(t1));
    ASSERT_NIL(r.tasks.create(t2));
    const std::vector<std::string> types = {type1, type2};
    const auto tasks = ASSERT_NIL_P(r.tasks.retrieve_by_type(types));
    ASSERT_EQ(tasks.size(), 2);
    bool found1 = false, found2 = false;
    for (const auto &t: tasks) {
        if (t.type == type1) found1 = true;
        if (t.type == type2) found2 = true;
    }
    ASSERT_TRUE(found1);
    ASSERT_TRUE(found2);
}

// These tests are disabled until parse/to_json are re-enabled on generated types.
// /// @brief it should correctly parse TaskStatusDetails from JSON.
// TEST(TaskStatusDetailsTests, testParseFromJSON) {
//     json j = {
//         {"task", 123456789},
//         {"cmd", "start"},
//         {"running", true},
//         {"data", {{"key", "value"}}}
//     };
//     x::json::Parser parser(j);
//     auto details = TaskStatusDetails::parse(parser);
//     ASSERT_NIL(parser.error());
//     ASSERT_EQ(details.task, 123456789);
//     ASSERT_EQ(details.cmd, "start");
//     ASSERT_EQ(details.running, true);
//     ASSERT_EQ(details.data["key"], "value");
// }

// /// @brief it should correctly serialize TaskStatusDetails to JSON.
// TEST(TaskStatusDetailsTests, testToJSON) {
//     TaskStatusDetails details{
//         .task = 987654321,
//         .cmd = "stop",
//         .running = false,
//         .data = {{"status", "completed"}},
//     };
//     const auto j = details.to_json();
//     ASSERT_EQ(j["task"], 987654321);
//     ASSERT_EQ(j["cmd"], "stop");
//     ASSERT_EQ(j["running"], false);
//     ASSERT_EQ(j["data"]["status"], "completed");
// }

// /// @brief it should round-trip TaskStatusDetails through JSON.
// TEST(TaskStatusDetailsTests, testRoundTrip) {
//     TaskStatusDetails original{
//         .task = 555555,
//         .cmd = "configure",
//         .running = true,
//         .data = {{"config", "test"}, {"version", 2}},
//     };
//     const auto j = original.to_json();
//     x::json::Parser parser(j);
//     auto recovered = TaskStatusDetails::parse(parser);
//     ASSERT_NIL(parser.error());
//     ASSERT_EQ(recovered.task, original.task);
//     ASSERT_EQ(recovered.cmd, original.cmd);
//     ASSERT_EQ(recovered.running, original.running);
//     ASSERT_EQ(recovered.data["config"], "test");
//     ASSERT_EQ(recovered.data["version"], 2);
// }

// /// @brief it should handle empty cmd field correctly.
// TEST(TaskStatusDetailsTests, testEmptyCmd) {
//     json j = {{"task", 111}, {"cmd", ""}, {"running", true}, {"data", json::object()}};
//     x::json::Parser parser(j);
//     auto details = TaskStatusDetails::parse(parser);
//     ASSERT_NIL(parser.error());
//     ASSERT_EQ(details.task, 111);
//     ASSERT_EQ(details.cmd, "");
//     ASSERT_EQ(details.running, true);
// }
}
