// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstddef>
#include <cstdint>
#include <filesystem>
#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/telem/telem.h"

#include "driver/task/task.h"

namespace driver::file {
/// @brief the name of the integration.
const std::string INTEGRATION_NAME = "file";
/// @brief the name of the task that the integration adds to each rack.
const std::string TASK_NAME = "File Browser";
/// @brief the type of the task that the integration adds to each rack.
const std::string TASK_TYPE = "file_browser";
/// @brief the command that lists a folder.
const std::string LIST_CMD_TYPE = "list";
/// @brief the most entries that one reply holds.
constexpr std::size_t MAX_ENTRIES = 1000;
/// @brief the largest file that the browser reads as a certificate.
constexpr std::uintmax_t MAX_CERTIFICATE_SIZE = 1 << 20;

/// @brief what a Browser lists at the top level, and the folder it never reads.
struct Config {
    /// @brief the roots of the file systems, such as "/" or "C:\".
    std::vector<std::filesystem::path> roots;
    /// @brief the home folder of the user that runs the Driver. Empty when unknown.
    std::filesystem::path home;
    /// @brief the rack's secrets folder. Empty when none is set.
    std::filesystem::path secrets_dir;
};

/// @returns the roots of the file systems on this computer.
std::vector<std::filesystem::path> roots();

/// @returns the home folder of the user that runs the Driver, or an empty path.
std::filesystem::path home();

/// @returns the time as a TimeStamp.
x::telem::TimeStamp to_timestamp(std::filesystem::file_time_type time);

/// @brief lists folders on the computer that runs the Driver. Gives the details of
/// each certificate file, and never opens a file in the secrets folder.
class Browser {
public:
    explicit Browser(Config cfg);

    /// @brief lists the folder at path, or the roots, the home folder, and the secrets
    /// folder when path is empty. It leaves out an entry whose name is not valid UTF-8.
    /// @param path an absolute path as UTF-8.
    /// @returns the data of the reply to a list command.
    /// @returns x::fs::INVALID_PATH if the path is relative or not a folder,
    /// x::fs::NOT_FOUND if nothing is at the path, x::fs::PERMISSION_DENIED if the
    /// Driver cannot read the folder, or x::fs::READ_ERROR for any other failure. The
    /// message names the path.
    [[nodiscard]] std::pair<x::json::json, x::errors::Error>
    list(const std::string &path) const;

private:
    Config cfg;

    [[nodiscard]] x::json::json top() const;
    [[nodiscard]] std::string secrets_dir() const;
};

/// @brief the File Browser task. It runs nothing, and answers each list command with
/// a status.
class Task final : public task::Task {
public:
    Task(std::shared_ptr<task::Context> ctx, synnax::task::Task task, Browser browser);

    [[nodiscard]] std::string name() const override { return this->task.name; }

    void exec(synnax::task::Command &cmd, x::breaker::Breaker &cancel) override;

    void stop(bool will_reconfigure) override {}

private:
    std::shared_ptr<task::Context> ctx;
    synnax::task::Task task;
    Browser browser;
};

/// @brief configures the File Browser task, and adds it to each rack.
class Factory final : public task::Factory {
public:
    explicit Factory(Config cfg);

    std::string name() override { return INTEGRATION_NAME; }

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task,
        const std::string &cmd_key,
        x::breaker::Breaker &cancel
    ) override;

    std::vector<std::pair<synnax::task::Task, std::unique_ptr<task::Task>>>
    configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::rack::Rack &rack,
        x::breaker::Breaker &cancel
    ) override;

private:
    Config cfg;
};
}
