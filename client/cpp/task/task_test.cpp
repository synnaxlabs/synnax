// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <set>

#include "gtest/gtest.h"

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/json/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

// NOLINTBEGIN(cppcoreguidelines-avoid-non-const-global-variables): one seed per suite,
// printed so that a failing run can be replayed.
std::mt19937 gen_rand_task = random_generator("Task Tests");
// NOLINTEND(cppcoreguidelines-avoid-non-const-global-variables)

namespace synnax::task {
/// @brief it should correctly create a module on the rack.
TEST(TaskTests, testCreateTask) {
    auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    auto m = Task{
        .name = "test_module",
        .type = "pagerduty_alert",
        .config = x::json::json{{"routing_key", "rk"}},
        .internal = false
    };
    ASSERT_NIL(r.tasks.create(m));
    ASSERT_EQ(m.name, "test_module");
    ASSERT_EQ(m.rack, r.key);
    ASSERT_FALSE(m.key.is_nil());
    ASSERT_FALSE(m.config_hash.empty());
}

/// @brief it should correctly retrieve a module from the rack.
TEST(TaskTests, testRetrieveTask) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    auto t = Task{
        .name = "test_module",
        .type = "pagerduty_alert",
        .config = {{"routing_key", "rk"}},
        .internal = false
    };
    ASSERT_NIL(r.tasks.create(t));
    const auto t2 = ASSERT_NIL_P(r.tasks.retrieve(t.key));
    ASSERT_EQ(t2.name, "test_module");
    ASSERT_EQ(t.rack, r.key);
    ASSERT_EQ(t2.key, t.key);
}

/// @brief it should retrieve a task by its name
TEST(TaskTests, testRetrieveTaskByName) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    const auto rand_name = std::to_string(gen_rand_task());
    auto t = Task{
        .name = rand_name,
        .type = "pagerduty_alert",
        .config = {{"routing_key", "rk"}}
    };
    ASSERT_NIL(r.tasks.create(t));
    const auto t2 = ASSERT_NIL_P(r.tasks.retrieve(rand_name));
    ASSERT_EQ(t2.name, rand_name);
    ASSERT_EQ(t.rack, r.key);
}

/// @brief it should retrieve a task by its type
TEST(TaskTests, testRetrieveTaskByType) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    auto t = Task{
        .name = "test_module",
        .type = "pagerduty_alert",
        .config = {{"routing_key", "rk"}}
    };
    ASSERT_NIL(r.tasks.create(t));
    const auto t2 = ASSERT_NIL_P(
        r.tasks.retrieve({.type = "pagerduty_alert"}, x::filter::one)
    );
    ASSERT_EQ(t2.name, "test_module");
    ASSERT_EQ(t.rack, r.key);
}

/// @brief it should reject a type that more than one task on the rack has.
TEST(TaskTests, testRetrieveTaskByTypeMultipleFound) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    for (const auto *name: {"first", "second"}) {
        auto t = Task{
            .name = name,
            .type = "pagerduty_alert",
            .config = {{"routing_key", "rk"}}
        };
        ASSERT_NIL(r.tasks.create(t));
    }
    ASSERT_OCCURRED_AS_P(
        r.tasks.retrieve({.type = "pagerduty_alert"}, x::filter::one),
        x::errors::MULTIPLE_RESULTS
    );
}

/// @brief it should name the filter in the error when no task matches it.
TEST(TaskTests, testRetrieveOneWhereNotFound) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    const auto [_, err] = r.tasks.retrieve({.type = "missing_type"}, x::filter::one);
    ASSERT_MATCHES(err, x::errors::NOT_FOUND);
    ASSERT_EQ(err.data, R"(Found no task where type = "missing_type")");
}

/// @brief it should correctly list the tasks on a rack.
TEST(TaskTests, testListTasks) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    auto m = Task{
        .name = "test_module",
        .type = "pagerduty_alert",
        .config = {{"routing_key", "rk"}}
    };
    ASSERT_NIL(r.tasks.create(m));
    const auto tasks = ASSERT_NIL_P(r.tasks.retrieve(Where{}));
    ASSERT_EQ(tasks.size(), 1);
    ASSERT_EQ(tasks[0].name, "test_module");
    ASSERT_EQ(tasks[0].rack, r.key);
    ASSERT_FALSE(tasks[0].key.is_nil());
}

/// @brief it should retrieve the tasks on the client's rack that match any branch of
/// an or.
TEST(TaskTests, testRetrieveWhere) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    std::vector<Task> tasks;
    for (int i = 0; i < 3; ++i) {
        auto t = Task{
            .name = "where_task_" + std::to_string(i),
            .type = "pagerduty_alert",
            .config = {{"routing_key", "rk"}}
        };
        ASSERT_NIL(r.tasks.create(t));
        tasks.push_back(t);
    }
    auto other = rack::Rack{.name = "other_rack"};
    ASSERT_NIL(client.racks.create(other));
    auto off_rack = Task{
        .name = tasks[0].name,
        .type = "pagerduty_alert",
        .config = {{"routing_key", "rk"}}
    };
    ASSERT_NIL(other.tasks.create(off_rack));
    const auto retrieved = ASSERT_NIL_P(
        r.tasks.retrieve(Where{.name = tasks[0].name} || Where{.key = tasks[1].key})
    );
    std::set<std::string> keys;
    for (const auto &t: retrieved)
        keys.insert(t.key.to_string());
    ASSERT_EQ(
        keys,
        (std::set<std::string>{tasks[0].key.to_string(), tasks[1].key.to_string()})
    );
}

/// @brief it should correctly delete a task from the rack.
TEST(TaskTests, testDeleteTask) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    auto t = Task{
        .name = "test_module",
        .type = "pagerduty_alert",
        .config = {{"routing_key", "rk"}}
    };
    ASSERT_NIL(r.tasks.create(t));
    ASSERT_NIL(r.tasks.del(t.key));
    ASSERT_OCCURRED_AS_P(r.tasks.retrieve(t.key), x::errors::NOT_FOUND);
}

/// @brief it should convert a task key to a resource ID
TEST(TaskTests, testTaskResourceID) {
    const auto key = x::uuid::create();
    const auto id = resource_id(key);
    ASSERT_EQ(id.type, "task");
    ASSERT_EQ(id.key, key.to_string());
}

/// @brief it should correctly create and retrieve a task with a status.
TEST(TaskTests, testCreateTaskWithStatus) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    auto t = Task{
        .name = "test_task_with_status",
        .type = "pagerduty_alert",
        .config = {{"routing_key", "rk"}},
        .status = Status{
            .key = "task-status-key",
            .variant = synnax::status::VARIANT_SUCCESS,
            .message = "Task is running",
            .time = x::telem::TimeStamp::now(),
            .details = task::StatusDetails{
                .running = true,
                .cmd = "start",
                .config_hash = "abc123",
            }
        }
    };
    ASSERT_NIL(r.tasks.create(t));
    const auto t2 = ASSERT_NIL_P(r.tasks.retrieve(t.key, {.include_status = true}));
    ASSERT_EQ(t2.name, "test_task_with_status");
    ASSERT_TRUE(t2.status.has_value());
    ASSERT_EQ(t2.status.value().variant, synnax::status::VARIANT_SUCCESS);
    ASSERT_EQ(t2.status.value().message, "Task is running");
    ASSERT_EQ(t2.status.value().details.running, true);
    ASSERT_EQ(t2.status.value().details.cmd, "start");
    ASSERT_EQ(t2.status.value().details.config_hash, "abc123");
}

/// @brief it should correctly retrieve a task with status by name.
TEST(TaskTests, testRetrieveTaskWithStatusByName) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    const auto rand_name = std::to_string(gen_rand_task());
    auto t = Task{
        .name = rand_name,
        .type = "pagerduty_alert",
        .config = {{"routing_key", "rk"}},
        .status = Status{
            .key = "task-status-by-name",
            .variant = synnax::status::VARIANT_WARNING,
            .message = "Task warning",
            .time = x::telem::TimeStamp::now()
        }
    };
    ASSERT_NIL(r.tasks.create(t));
    const auto t2 = ASSERT_NIL_P(r.tasks.retrieve(rand_name, {.include_status = true}));
    ASSERT_EQ(t2.name, rand_name);
    ASSERT_TRUE(t2.status.has_value());
    ASSERT_EQ(t2.status.value().variant, synnax::status::VARIANT_WARNING);
    ASSERT_EQ(t2.status.value().message, "Task warning");
}

/// @brief it should correctly list tasks with statuses.
TEST(TaskTests, testListTasksWithStatus) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    auto t = Task{
        .name = "test_task_list_status",
        .type = "pagerduty_alert",
        .config = {{"routing_key", "rk"}},
        .status = Status{
            .key = "task-list-status",
            .variant = synnax::status::VARIANT_INFO,
            .message = "Task info",
            .time = x::telem::TimeStamp::now(),
        }
    };
    ASSERT_NIL(r.tasks.create(t));
    const auto tasks = ASSERT_NIL_P(
        r.tasks.retrieve(Where{}, {.include_status = true})
    );
    ASSERT_EQ(tasks.size(), 1);
    ASSERT_TRUE(tasks[0].status.has_value());
    ASSERT_EQ(tasks[0].status.value().variant, synnax::status::VARIANT_INFO);
    ASSERT_EQ(tasks[0].status.value().message, "Task info");
}
/// @brief it should retrieve multiple tasks by their names.
TEST(TaskTests, testRetrieveTasksByNames) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    const auto rand1 = std::to_string(gen_rand_task());
    const auto rand2 = std::to_string(gen_rand_task());
    auto t1 = Task{
        .name = rand1,
        .type = "pagerduty_alert",
        .config = {{"routing_key", "rk-1"}}
    };
    auto t2 = Task{
        .name = rand2,
        .type = "pagerduty_alert",
        .config = {{"routing_key", "rk-2"}}
    };
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
/// @brief it should apply the limit of the options to a retrieve by names.
TEST(TaskTests, testRetrieveTasksByNamesWithLimit) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    std::vector<std::string> names;
    for (int i = 0; i < 2; i++) {
        auto t = Task{
            .name = std::to_string(gen_rand_task()),
            .type = "pagerduty_alert",
            .config = {{"routing_key", "rk"}}
        };
        ASSERT_NIL(r.tasks.create(t));
        names.push_back(t.name);
    }
    const auto tasks = ASSERT_NIL_P(r.tasks.retrieve(names, {.limit = 1}));
    ASSERT_EQ(tasks.size(), 1);
}

/// @brief it should reject a limit or a cursor on a retrieve of one task.
TEST(TaskTests, testRetrieveTaskRejectsPage) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    for (const auto &options:
         {RetrieveOptions{.limit = 1}, RetrieveOptions{.after = Cursor{}}}) {
        const auto [_, err] = r.tasks.retrieve("test_module", options);
        ASSERT_MATCHES(err, x::errors::VALIDATION);
        ASSERT_EQ(
            err.data,
            "a query for one record takes no limit or cursor, since a page could "
            "hide a second match"
        );
    }
}

/// @brief it should page a retrieve of the tasks on a rack that a filter matches.
TEST(TaskTests, testRetrieveWherePage) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    for (int i = 0; i < 3; i++) {
        auto t = Task{
            .name = std::to_string(gen_rand_task()),
            .type = "pagerduty_alert",
            .config = {{"routing_key", "rk"}}
        };
        ASSERT_NIL(r.tasks.create(t));
    }
    const auto where = Where{.type = "pagerduty_alert"};
    const auto first = ASSERT_NIL_P(r.tasks.retrieve(where, {.limit = 2}));
    ASSERT_EQ(first.size(), 2);
    const auto rest = ASSERT_NIL_P(
        r.tasks.retrieve(where, {.after = Cursor{.key = first[1].key}})
    );
    ASSERT_EQ(rest.size(), 1);
}

/// @brief it should retrieve multiple tasks by their types.
TEST(TaskTests, testRetrieveTasksByTypes) {
    const auto client = new_test_client();
    auto r = rack::Rack{.name = "test_rack"};
    ASSERT_NIL(client.racks.create(r));
    const std::string type1 = "pagerduty_alert";
    const std::string type2 = "labjack_scan";
    auto t1 = Task{
        .name = "task_by_type_1",
        .type = type1,
        .config = {{"routing_key", "rk-1"}}
    };
    auto t2 = Task{
        .name = "task_by_type_2",
        .type = type2,
        .config = {{"routing_key", "rk-2"}}
    };
    ASSERT_NIL(r.tasks.create(t1));
    ASSERT_NIL(r.tasks.create(t2));
    const std::vector<std::string> types = {type1, type2};
    const auto tasks = ASSERT_NIL_P(r.tasks.retrieve({.type = types}));
    ASSERT_EQ(tasks.size(), 2);
    bool found1 = false, found2 = false;
    for (const auto &t: tasks) {
        if (t.type == type1) found1 = true;
        if (t.type == type2) found2 = true;
    }
    ASSERT_TRUE(found1);
    ASSERT_TRUE(found2);
}

/// @brief it should correctly parse StatusDetails from JSON.
TEST(StatusDetailsTests, testParseFromJSON) {
    x::json::json j = {
        {"cmd", "start"},
        {"running", true},
        {"data", {{"routing_key", "rk"}}}
    };
    x::json::Parser parser(j);
    auto details = StatusDetails::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(details.cmd, "start");
    ASSERT_EQ(details.running, true);
    ASSERT_TRUE(details.data.has_value());
    ASSERT_EQ((details.data.value())["routing_key"], "rk");
}

/// @brief it should correctly serialize StatusDetails to JSON.
TEST(StatusDetailsTests, testToJSON) {
    StatusDetails details{
        .running = false,
        .cmd = "stop",
        .data = x::json::json{{"status", "completed"}},
    };
    const auto j = details.to_json();
    ASSERT_EQ(j["cmd"], "stop");
    ASSERT_EQ(j["running"], false);
    ASSERT_EQ(j["data"]["status"], "completed");
}

/// @brief it should round-trip StatusDetails through JSON.
TEST(StatusDetailsTests, testRoundTrip) {
    StatusDetails original{
        .running = true,
        .cmd = "configure",
        .data = x::json::json{{"config", "test"}, {"version", 2}},
    };
    const auto j = original.to_json();
    x::json::Parser parser(j);
    auto recovered = StatusDetails::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(recovered.cmd, original.cmd);
    ASSERT_EQ(recovered.running, original.running);
    ASSERT_TRUE(recovered.data.has_value());
    ASSERT_EQ((recovered.data.value())["config"], "test");
    ASSERT_EQ((recovered.data.value())["version"], 2);
}

/// @brief it should report a config that does not convert instead of sending an empty
/// one.
TEST(TaskTests, testToProtoReportsBadConfig) {
    const Task t{
        .name = "test_task",
        .type = "mock",
        .config = x::json::deeply_nested_object()
    };
    ASSERT_OCCURRED_AS_P(t.to_proto(), x::errors::VALIDATION);
}

/// @brief it should report command arguments that do not convert.
TEST(CommandTests, testToProtoReportsBadArgs) {
    const Command c{
        .task = x::uuid::create(),
        .type = "start",
        .args = x::json::deeply_nested_object()
    };
    ASSERT_OCCURRED_AS_P(c.to_proto(), x::errors::VALIDATION);
}

/// @brief it should report optional status data that does not convert.
TEST(StatusDetailsTests, testToProtoReportsBadData) {
    StatusDetails d;
    d.data = x::json::deeply_nested_object();
    ASSERT_OCCURRED_AS_P(d.to_proto(), x::errors::VALIDATION);
}

/// @brief it should handle empty cmd field correctly.
TEST(StatusDetailsTests, testEmptyCmd) {
    x::json::json j = {
        {"cmd", ""},
        {"running", true},
        {"data", x::json::json::object()}
    };
    x::json::Parser parser(j);
    auto details = StatusDetails::parse(parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(details.cmd, "");
    ASSERT_EQ(details.running, true);
}
}
