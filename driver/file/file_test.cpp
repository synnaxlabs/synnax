// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <string>

#include "gtest/gtest.h"

#include "x/cpp/uuid/uuid.h"

#include "driver/file/file.h"

namespace driver::file {
namespace {
struct Fixture {
    std::shared_ptr<task::MockContext> ctx = std::make_shared<task::MockContext>(
        nullptr
    );
    synnax::task::Task task{
        .key = x::uuid::create(),
        .name = TASK_NAME,
        .type = TASK_TYPE,
        .config_hash = "hash",
    };
    Task browser_task{this->ctx, this->task, Browser(Config{.roots = {"/"}})};
    x::breaker::Breaker breaker;

    synnax::task::Status exec(const std::string &type, x::json::json::object_t args) {
        synnax::task::Command cmd{
            .task = this->task.key,
            .type = type,
            .key = "cmd",
            .args = std::move(args),
        };
        this->browser_task.exec(cmd, this->breaker);
        return this->ctx->statuses.back();
    }
};
}

/// @brief it should answer a list command with the listed folder.
TEST(FileTask, List) {
    Fixture f;
    const auto status = f.exec(LIST_CMD_TYPE, {});
    ASSERT_EQ(status.variant, synnax::status::VARIANT_SUCCESS);
    ASSERT_EQ(status.message, "Listed files");
    ASSERT_EQ(status.name, TASK_NAME);
    ASSERT_EQ(status.owner, synnax::task::resource_id(f.task.key));
    ASSERT_TRUE(status.details.running);
    ASSERT_EQ(status.details.cmd, "cmd");
    ASSERT_EQ(status.details.config_hash, "hash");
    const x::json::json data = status.details.data.value();
    ASSERT_EQ(data["path"], "");
    ASSERT_EQ(data["entries"][0]["path"], "/");
}

/// @brief it should answer a failed list with an error status that names the path.
TEST(FileTask, ListFailed) {
    Fixture f;
    const auto status = f.exec(LIST_CMD_TYPE, {{"path", "relative"}});
    ASSERT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    ASSERT_EQ(status.message, "Failed to list files");
    ASSERT_EQ(status.description, "relative is not an absolute path.");
    ASSERT_TRUE(status.details.running);
    ASSERT_EQ(status.details.cmd, "cmd");
}

/// @brief it should reject a path that is not a string.
TEST(FileTask, PathNotString) {
    Fixture f;
    const auto status = f.exec(LIST_CMD_TYPE, {{"path", 12}});
    ASSERT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    ASSERT_EQ(status.message, "Failed to list files");
    ASSERT_EQ(status.description, "The path must be a string.");
}

/// @brief it should reject a command that it does not accept.
TEST(FileTask, UnknownCommand) {
    Fixture f;
    const auto status = f.exec("start", {});
    ASSERT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    ASSERT_EQ(status.message, "Failed to execute command");
    ASSERT_EQ(status.description, "The task does not accept the 'start' command.");
    ASSERT_EQ(status.details.cmd, "cmd");
}

/// @brief it should configure only File Browser tasks.
TEST(FileFactory, ConfigureTask) {
    Factory factory(Config{.roots = {"/"}});
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    x::breaker::Breaker breaker;
    auto [browser, handled] = factory.configure_task(
        ctx,
        synnax::task::Task{.key = x::uuid::create(), .name = "a", .type = TASK_TYPE},
        "",
        breaker
    );
    ASSERT_TRUE(handled);
    ASSERT_NE(browser, nullptr);
    ASSERT_EQ(browser->name(), "a");
    auto [other, other_handled] = factory.configure_task(
        ctx,
        synnax::task::Task{.key = x::uuid::create(), .type = "opc_scan"},
        "",
        breaker
    );
    ASSERT_FALSE(other_handled);
    ASSERT_EQ(other, nullptr);
}
}
