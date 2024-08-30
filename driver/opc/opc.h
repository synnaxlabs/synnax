// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "driver/config/config.h"
#include "driver/task/task.h"
#include "include/open62541/types.h"

namespace opc {
/// @brief the configuration for an OPC UA connection.
    struct ConnectionConfig {
        /// @brief the endpoint of the OPC UA server.
        std::string endpoint;
        /// @brief the username to use for authentication. Not required.
        std::string username;
        /// @brief the password to use for authentication. Not required.
        std::string password;
        /// @brief the security mode.
        std::string security_mode;
        /// @brief the security policy.
        std::string security_policy;
        /// @brief the client certificate used to sign and encrypt messages. Only required
        /// if the security policy is not "None".
        std::string client_cert;
        /// @brief the client private key used to sign and encrypt messages. Only required
        /// if the security policy is not "None".
        std::string client_private_key;
        /// @brief a trusted server certificate. Only req
        std::string server_cert;

        ConnectionConfig() = default;

        explicit ConnectionConfig(
                config::Parser parser
        ) : endpoint(parser.required<std::string>("endpoint")),
            username(parser.optional<std::string>("username", "")),
            password(parser.optional<std::string>("password", "")),
            security_mode(parser.optional<std::string>("security_mode", "None")),
            security_policy(parser.optional<std::string>("security_policy", "None")),
            client_cert(parser.optional<std::string>("client_certificate", "")),
            client_private_key(parser.optional<std::string>("client_private_key", "")),
            server_cert(parser.optional<std::string>("server_certificate", "")) {
        }

        json toJSON() const {
            return {
                    {"endpoint",           endpoint},
                    {"username",           username},
                    {"password",           password},
                    {"security_mode",      security_mode},
                    {"security_policy",    security_policy},
                    {"client_certificate", client_cert},
                    {"client_private_key", client_private_key}
            };
        }
    };

    struct DeviceNodeProperties {
        synnax::DataType data_type;
        std::string node_class;
        std::string name;
        std::string node_id;
        bool is_array;

        DeviceNodeProperties(
                synnax::DataType data_type,
                std::string name,
                std::string node_id,
                std::string node_class,
                bool is_array
        ) : data_type(data_type), name(name), node_id(node_id), is_array(is_array),
            node_class(node_class) {
        }

        explicit DeviceNodeProperties(config::Parser parser) : data_type(
                synnax::DataType(parser.required<std::string>("data_type"))),
                                                               name(parser.required<std::string>(
                                                                       "name")),
                                                               node_id(parser.required<std::string>(
                                                                       "node_id")),
                                                               is_array(
                                                                       parser.optional<bool>(
                                                                               "is_array",
                                                                               false)) {
        }

        json toJSON() const {
            return {
                    {"data_type",  data_type.name()},
                    {"name",       name},
                    {"node_id",    node_id},
                    {"node_class", node_class},
                    {"is_array",   is_array}
            };
        }
    };

    struct DeviceProperties {
        ConnectionConfig connection;
        std::vector<DeviceNodeProperties> channels;

        DeviceProperties(
                ConnectionConfig connection,
                std::vector<DeviceNodeProperties> channels
        ) : connection(connection), channels(channels) {
        }

        explicit DeviceProperties(
                config::Parser parser
        ) : connection(parser.child("connection")),
            channels({}) {
            parser.iter("channels", [&](const config::Parser &cb) {
                channels.emplace_back(cb);
            });
        }

        json toJSON() const {
            json j;
            j["connection"] = connection.toJSON();
            j["channels"] = json::array();
            for (const auto &ch: channels)
                j["channels"].push_back(ch.toJSON());
            return j;
        }
    };

    const std::string INTEGRATION_NAME = "opc";

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
