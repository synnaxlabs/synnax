#pragma once

#include "driver/driver/config/config.h"
#include "driver/driver/task/task.h"

namespace opcua {
/// @brief the configuration for an OPC UA connection.
struct ConnectionConfig {
    /// @brief the endpoint of the OPC UA server.
    std::string endpoint;
    /// @brief the username to use for authentication. Not required.
    std::string username;
    /// @brief the password to use for authentication. Not required.
    std::string password;

    ConnectionConfig() = default;

    explicit ConnectionConfig(
        config::Parser builder
    ): endpoint(builder.required<std::string>("endpoint")),
       username(builder.optional<std::string>("username", "")),
       password(builder.optional<std::string>("password", "")) {
    }
};


class Factory final : public task::Factory {
    std::pair<std::unique_ptr<task::Task>, bool> configureTask(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) override;

    std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task> > >
    configureInitialTasks(const std::shared_ptr<task::Context> &ctx,
                          const synnax::Rack &rack) override;
};
}
