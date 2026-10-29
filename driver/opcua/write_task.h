// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <cstddef>
#include <memory>
#include <string>
#include <unordered_map>
#include <utility>
#include <vector>

#include "absl/log/log.h"
#include "open62541/client.h"
#include "open62541/types.h"

#include "client/cpp/errors/errors.h"
#include "client/cpp/opcua/json.gen.h"
#include "client/cpp/synnax.h"
#include "x/cpp/json/json.h"
#include "x/cpp/strings/strings.h"

#include "driver/common/write_task.h"
#include "driver/errors/errors.h"
#include "driver/opcua/connection/connection.h"
#include "driver/opcua/device/device.h"
#include "driver/opcua/errors/errors.h"
#include "driver/opcua/method/method.h"
#include "driver/opcua/telem/telem.h"
#include "driver/opcua/types/types.h"

namespace driver::opcua {
struct OutputChan {
    /// @brief the OPC UA node id: the variable to write, or the method to call.
    types::NodeId node;
    /// @brief the corresponding channel key to write the variable for the node
    /// from.
    const synnax::channel::Key cmd_channel;
    /// @brief the channel fetched from the Synnax server. This does not need to
    /// be provided via the JSON configuration.
    synnax::channel::Channel ch;
    /// @brief whether each command calls node as a method, instead of writing its
    /// value.
    const bool method;
    /// @brief the object that holds the method.
    types::NodeId object;
    /// @brief the input arguments of the method, in order.
    const std::vector<::synnax::opcua::MethodArgument> arguments;

    OutputChan(const ::synnax::opcua::WriteChannel &parsed, x::json::Parser &parser):
        node(types::NodeId::parse("node_id", parser)),
        cmd_channel(parsed.cmd_channel),
        method(parsed.type == ::synnax::opcua::WRITE_TYPE_METHOD),
        arguments(parsed.arguments) {
        if (this->cmd_channel == 0)
            parser.field_err("cmd_channel", "channel must be specified");
        if (parsed.type != ::synnax::opcua::WRITE_TYPE_NODE && !this->method)
            parser.field_err("type", "Type must be node or method");
        if (!this->method) return;
        this->object = types::NodeId::parse("object_id", parser);
        std::size_t commands = 0;
        for (std::size_t i = 0; i < this->arguments.size(); ++i) {
            const auto &source = this->arguments[i].source;
            if (source == ::synnax::opcua::ARGUMENT_SOURCE_COMMAND)
                commands++;
            else if (source != ::synnax::opcua::ARGUMENT_SOURCE_VALUE)
                parser.field_err(
                    "arguments." + std::to_string(i) + ".source",
                    "Source must be value or command"
                );
        }
        if (commands > 1)
            parser.field_err("arguments", "Only one argument can take the command");
    }
};

struct WriteTaskConfig : common::BaseWriteTaskConfig {
    /// @brief the enabled channels to write to the server.
    std::unordered_map<synnax::channel::Key, std::unique_ptr<OutputChan>> channels;

    explicit WriteTaskConfig(
        const std::shared_ptr<synnax::Synnax> &client,
        x::json::Parser &parser
    ):
        common::BaseWriteTaskConfig(parser) {
        parser.iter("channels", [&](x::json::Parser &cp) {
            const auto parsed = ::synnax::opcua::WriteChannel::parse(cp);
            if (parsed.disabled) return;
            channels[parsed.cmd_channel] = std::make_unique<OutputChan>(parsed, cp);
        });
        if (this->channels.empty()) {
            parser.field_err("channels", "task must have at least one enabled channel");
            return;
        }

        std::vector<synnax::channel::Key> keys;
        keys.reserve(this->channels.size());
        for (const auto &[key, _]: this->channels)
            keys.push_back(key);
        auto [sy_channels, ch_err] = client->channels.retrieve(keys);
        if (ch_err) {
            parser.field_err("channels", "failed to retrieve channels: " + ch_err.data);
            return;
        }
        for (const auto &sy_ch: sy_channels) {
            auto it = this->channels.find(sy_ch.key);
            if (it != this->channels.end()) it->second->ch = sy_ch;
        }
        if (const auto err = device::retrieve_connection(*client, this->device).second)
            parser.field_err("device", common::config_err_description(err));
    }

    [[nodiscard]] std::vector<synnax::channel::Key> cmd_keys() const {
        std::vector<synnax::channel::Key> keys;
        keys.reserve(this->channels.size());
        for (const auto &[key, _]: channels)
            keys.push_back(key);
        return keys;
    }

    static std::pair<WriteTaskConfig, x::errors::Error> parse(
        const std::shared_ptr<synnax::Synnax> &client,
        const synnax::task::Task &task
    ) {
        auto parser = x::json::Parser(task.config);
        return {WriteTaskConfig(client, parser), parser.error()};
    }
};

class WriteTaskSink final : public common::Sink {
    const WriteTaskConfig cfg;
    std::shared_ptr<synnax::Synnax> client;
    std::shared_ptr<connection::Pool> pool;
    /// @brief the device's connection settings as of the last retrieve.
    connection::Config conn_cfg;
    connection::Pool::Connection connection;
    /// @brief the data type of each channel's node on the current connection, or
    /// nothing for a node whose type the server did not report.
    std::unordered_map<synnax::channel::Key, const UA_DataType *> targets;
    /// @brief the data types of the input arguments of each method channel's method on
    /// the current connection, or nothing for a method whose arguments the server did
    /// not report. A type is null when the client does not know it.
    std::unordered_map<synnax::channel::Key, std::vector<const UA_DataType *>> inputs;
    types::WriteRequestBuilder builder;
    std::vector<synnax::channel::Key> written_keys;

    /// @brief writes on conn from now on.
    /// @returns the error of reading the types of the nodes, after which the sink has
    /// no connection.
    x::errors::Error use(connection::Pool::Connection conn) {
        this->connection = std::move(conn);
        const auto err = this->read_targets();
        if (err) this->connection = connection::Pool::Connection(nullptr, nullptr, "");
        return err;
    }

    /// @brief reads the DataType of each channel's node, so commands convert to it. A
    /// node of a type the client does not know converts by the type of its channel.
    /// @returns only an error that loses the connection. When the server refuses the
    /// read for another reason, every node converts by the type of its channel.
    x::errors::Error read_targets() {
        this->targets.clear();
        if (const auto err = this->read_inputs()) return err;
        const auto &namespaces = this->connection.server().namespaces;
        types::ReadRequestBuilder req;
        std::vector<synnax::channel::Key> keys;
        for (const auto &[key, ch]: this->cfg.channels) {
            if (ch->method) continue;
            const auto id = ch->node.resolve(namespaces);
            if (!id.has_value()) continue;
            req.add_node(id.value(), UA_ATTRIBUTEID_DATATYPE);
            keys.push_back(key);
        }
        if (req.empty()) return x::errors::NIL;
        auto [res, err] = this->connection.read(req.build());
        if (err.matches(errors::UNREACHABLE)) return err;
        if (err) {
            LOG(WARNING) << "[opc.write] failed to read the types of the nodes, so "
                            "commands convert by the types of their channels: "
                         << err;
            return x::errors::NIL;
        }
        for (std::size_t i = 0; i < keys.size(); ++i)
            if (const auto &value = res.get().results[i].value;
                UA_Variant_hasScalarType(&value, &UA_TYPES[UA_TYPES_NODEID]))
                this->targets[keys[i]] = UA_findDataType(
                    static_cast<const UA_NodeId *>(value.data)
                );
        return x::errors::NIL;
    }

    /// @brief reads the input arguments of each method channel's method, so arguments
    /// convert to their declared types.
    /// @returns only an error that loses the connection. When the server refuses the
    /// browse or read for another reason, arguments convert by the types of their
    /// values.
    x::errors::Error read_inputs() {
        this->inputs.clear();
        const auto &namespaces = this->connection.server().namespaces;
        std::vector<synnax::channel::Key> keys;
        std::vector<UA_NodeId> methods;
        for (const auto &[key, ch]: this->cfg.channels)
            if (ch->method)
                if (const auto id = ch->node.resolve(namespaces); id.has_value()) {
                    keys.push_back(key);
                    methods.push_back(*id);
                }
        if (methods.empty()) return x::errors::NIL;
        auto [read, err] = method::read_inputs(this->connection, methods);
        if (err.matches(errors::UNREACHABLE)) return err;
        if (err) {
            LOG(WARNING) << "[opc.write] failed to read the arguments of the methods, "
                            "so arguments convert by the types of their values: "
                         << err;
            return x::errors::NIL;
        }
        for (std::size_t i = 0; i < keys.size(); ++i) {
            if (!read[i].has_value()) continue;
            auto &types = this->inputs[keys[i]];
            for (const auto &arg: read[i].value())
                types.push_back(UA_findDataType(&arg.data_type.get()));
        }
        return x::errors::NIL;
    }

    /// @brief connects with the device's current settings, so a corrected endpoint or
    /// login takes effect without a restart. Keeps the last settings while the Core is
    /// unavailable.
    x::errors::Error connect() {
        auto [conn_cfg, retrieve_err] = device::retrieve_connection(
            *this->client,
            this->cfg.device
        );
        if (!retrieve_err)
            this->conn_cfg = std::move(conn_cfg);
        else if (!synnax::errors::is_temporarily_unavailable(retrieve_err))
            return retrieve_err;
        auto [c, err] = this->pool->acquire(this->conn_cfg, "[opc.write] ");
        if (err) return err;
        return this->use(std::move(c));
    }

public:
    WriteTaskSink(
        std::shared_ptr<synnax::Synnax> client,
        std::shared_ptr<connection::Pool> pool,
        WriteTaskConfig cfg
    ):
        Sink(cfg.cmd_keys()),
        cfg(std::move(cfg)),
        client(std::move(client)),
        pool(std::move(pool)),
        connection(nullptr, nullptr, "") {}

    /// @brief retrieves the device again, so a changed trusted server certificate takes
    /// effect on restart. A server that is off or refuses Sessions does not stop the
    /// start, since each command connects again.
    x::errors::Error start() override {
        auto [conn_cfg, conn_err] = device::retrieve_connection(
            *this->client,
            this->cfg.device
        );
        if (conn_err) return conn_err;
        this->conn_cfg = std::move(conn_cfg);
        auto [c, err] = this->pool->acquire(this->conn_cfg, "[opc.write] ");
        if (!err) err = this->use(std::move(c));
        if (err.matches(driver::errors::TEMPORARY_HARDWARE_ERROR))
            return x::errors::NIL;
        return err;
    }

    x::errors::Error stop() override {
        this->connection = connection::Pool::Connection(nullptr, nullptr, "");
        return x::errors::NIL;
    }

    /// @brief writes the commands of node channels to the server, then calls the
    /// methods of method channels. A connection that breaks gets one new connection
    /// and one more attempt, so a Session the server closed while idle does not drop
    /// the command.
    x::errors::Error write(::x::telem::Frame &frame) override {
        if (!this->connection)
            if (const auto err = this->connect()) return err;
        auto err = this->perform_write(frame);
        if (err.matches(errors::UNREACHABLE)) {
            if (const auto conn_err = this->reconnect(err)) return conn_err;
            err = this->perform_write(frame);
        }
        if (err && !err.matches(errors::COMMAND_REJECTED)) return err;
        auto call_err = this->call_methods(frame);
        if (call_err && !call_err.matches(errors::COMMAND_REJECTED)) return call_err;
        std::vector<std::string> problems;
        for (const auto &e: {err, call_err})
            if (e) problems.push_back(e.data);
        if (problems.empty()) return x::errors::NIL;
        return {errors::COMMAND_REJECTED, x::strings::join(problems, " ")};
    }

private:
    /// @brief drops the connection that err broke, and connects again.
    x::errors::Error reconnect(const x::errors::Error &err) {
        LOG(WARNING) << "[opc.write_task] lost the connection, connecting again: "
                     << err;
        this->connection = connection::Pool::Connection(nullptr, nullptr, "");
        return this->connect();
    }

    /// @returns whether err shows that the server did not run a method call, so the
    /// call can be sent again.
    [[nodiscard]] static bool unsent(const x::errors::Error &err) {
        if (err.matches(errors::NO_CONNECTION)) return true;
        return std::ranges::any_of(
            std::vector<std::string>{
                "BadSessionIdInvalid",
                "BadSessionClosed",
                "BadSessionNotActivated",
                "BadSecureChannelIdInvalid",
                "BadSecureChannelTokenUnknown",
            },
            [&err](const std::string &code) {
                return err.matches(errors::UNREACHABLE.sub(code));
            }
        );
    }

    /// @brief calls the method of ch with the last command in s.
    /// @returns errors::COMMAND_REJECTED when an argument does not convert or the
    /// method fails, or the error of the call.
    x::errors::Error call(
        const synnax::channel::Key key,
        const OutputChan &ch,
        const ::x::telem::Series &s
    ) {
        const auto rejected = [&ch](const std::string &problem) {
            return x::errors::Error(
                errors::COMMAND_REJECTED,
                ch.ch.name + ": " + problem
            );
        };
        const auto &namespaces = this->connection.server().namespaces;
        const auto object = ch.object.resolve(namespaces);
        const auto method = ch.node.resolve(namespaces);
        if (!object.has_value() || !method.has_value())
            return {
                errors::COMMAND_REJECTED,
                errors::channel_status(ch.ch.name, UA_STATUSCODE_BADNODEIDUNKNOWN)
            };
        const auto it = this->inputs.find(key);
        const bool known = it != this->inputs.end();
        const auto count = [](const std::size_t n) {
            return std::to_string(n) + (n == 1 ? " argument" : " arguments");
        };
        if (known && it->second.size() != ch.arguments.size())
            return rejected(
                "the method takes " + count(it->second.size()) +
                ", but the channel gives " + count(ch.arguments.size()) + "."
            );
        std::vector<types::Variant> owned(ch.arguments.size());
        std::vector<UA_Variant> args;
        args.reserve(ch.arguments.size());
        for (std::size_t i = 0; i < ch.arguments.size(); ++i) {
            const auto &arg = ch.arguments[i];
            const auto *type = known ? it->second[i] : nullptr;
            auto [v, err] = arg.source == ::synnax::opcua::ARGUMENT_SOURCE_COMMAND
                              ? telem::to_variant(s, type)
                              : telem::to_variant(::x::telem::Series(arg.value), type);
            if (err)
                return rejected(
                    "argument " + std::to_string(i + 1) + ": " + err.data + "."
                );
            *owned[i].ptr() = v;
            args.push_back(owned[i].get());
        }
        UA_CallMethodRequest req;
        UA_CallMethodRequest_init(&req);
        req.objectId = *object;
        req.methodId = *method;
        req.inputArguments = args.data();
        req.inputArgumentsSize = args.size();
        auto [res, err] = this->connection.call(req);
        if (err) return err;
        if (const auto status = res.get().results[0].statusCode;
            UA_StatusCode_isBad(status))
            return {
                errors::COMMAND_REJECTED,
                errors::channel_status(ch.ch.name, status)
            };
        return x::errors::NIL;
    }

    /// @brief calls the method of each method channel in frame. A call that the
    /// server did not run because the connection broke gets one new connection and
    /// one more attempt. A call that the server might have run is never sent again.
    x::errors::Error call_methods(const ::x::telem::Frame &frame) {
        std::vector<std::string> problems;
        for (const auto &[key, s]: frame) {
            const auto it = this->cfg.channels.find(key);
            if (it == this->cfg.channels.end() || !it->second->method) continue;
            const auto &ch = *it->second;
            if (!this->connection)
                if (const auto err = this->connect()) return err;
            auto err = this->call(key, ch, s);
            if (unsent(err)) {
                if (const auto conn_err = this->reconnect(err)) return conn_err;
                err = this->call(key, ch, s);
            }
            if (!err) continue;
            if (err.matches(errors::COMMAND_REJECTED)) {
                problems.push_back(err.data);
                continue;
            }
            if (!err.matches(errors::UNREACHABLE)) return err;
            this->connection = connection::Pool::Connection(nullptr, nullptr, "");
            problems.push_back(
                ch.ch.name +
                ": lost the connection during the call, so the method might not "
                "have run."
            );
        }
        if (problems.empty()) return x::errors::NIL;
        return {errors::COMMAND_REJECTED, x::strings::join(problems, " ")};
    }

    /// @brief fills builder with the frame's values, converted to the types of their
    /// nodes, with the namespace indexes of the current connection.
    /// @returns a problem for each value that the Driver cannot convert or whose
    /// namespace the server lacks.
    std::vector<std::string> build(const ::x::telem::Frame &frame) {
        this->builder.clear();
        this->written_keys.clear();
        std::vector<std::string> problems;
        const auto &namespaces = this->connection.server().namespaces;
        for (const auto &[key, s]: frame) {
            auto it = this->cfg.channels.find(key);
            if (it == this->cfg.channels.end()) continue;
            const auto &ch = *it->second;
            if (ch.method) continue;
            const auto id = ch.node.resolve(namespaces);
            if (!id.has_value()) {
                problems.push_back(
                    ch.ch.name + ": the server has no namespace " +
                    ch.node.namespace_uri() + "."
                );
                continue;
            }
            const auto target = this->targets.find(key);
            if (auto err = this->builder.add_value(
                    id.value(),
                    s,
                    target == this->targets.end() ? nullptr : target->second
                )) {
                problems.push_back(ch.ch.name + ": " + err.data + ".");
                continue;
            }
            this->written_keys.push_back(key);
        }
        return problems;
    }

    /// @brief writes the frame's values and sets problems from build. When the server
    /// rejects the whole request after the client certificate changed, connects again
    /// with the new certificate and writes again.
    std::pair<types::WriteResponse, x::errors::Error>
    send(const ::x::telem::Frame &frame, std::vector<std::string> &problems) {
        const auto attempt =
            [&]() -> std::pair<types::WriteResponse, x::errors::Error> {
            problems = this->build(frame);
            if (this->builder.empty()) return {types::WriteResponse(), x::errors::NIL};
            return this->connection.write(this->builder.build());
        };
        auto [res, err] = attempt();
        if (!err || err.matches(errors::UNREACHABLE)) return {std::move(res), err};
        auto [refreshed, refresh_err] = this->pool->refresh(
            this->connection,
            this->conn_cfg,
            "[opc.write] "
        );
        if (refresh_err) return {std::move(res), refresh_err};
        if (!refreshed) return {std::move(res), err};
        if (const auto targets_err = this->read_targets())
            return {types::WriteResponse(), targets_err};
        return attempt();
    }

    /// @brief writes the frame's values. A value that the Driver cannot convert, or
    /// that the server rejects, is not written, and the other values still are.
    x::errors::Error perform_write(const ::x::telem::Frame &frame) {
        std::vector<std::string> problems;
        auto [res, send_err] = this->send(frame, problems);
        if (send_err) return send_err;
        for (std::size_t i = 0; i < res.get().resultsSize; ++i) {
            const auto status = res.get().results[i];
            if (!UA_StatusCode_isBad(status)) continue;
            const auto &ch = this->cfg.channels.at(this->written_keys[i]);
            problems.push_back(errors::channel_status(ch->ch.name, status));
        }
        if (problems.empty()) return x::errors::NIL;
        return {errors::COMMAND_REJECTED, x::strings::join(problems, " ")};
    }
};
}
