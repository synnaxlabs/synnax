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
#include <variant>
#include <vector>

#include "absl/log/log.h"
#include "open62541/client.h"
#include "open62541/types.h"

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

    OutputChan(
        const ::synnax::opcua::BaseWriteChannel &base,
        const ::synnax::opcua::MethodWriteChannel *call,
        x::json::Parser &parser
    ):
        node(types::NodeId::parse("node_id", parser)),
        cmd_channel(base.cmd_channel),
        method(call != nullptr),
        arguments(
            call != nullptr ? call->arguments
                            : std::vector<::synnax::opcua::MethodArgument>{}
        ) {
        if (this->cmd_channel == 0)
            parser.field_err("cmd_channel", "channel must be specified");
        if (!this->method) return;
        this->object = types::NodeId::parse("object_id", parser);
        if (std::ranges::count_if(this->arguments, [](const auto &arg) {
                return std::holds_alternative<::synnax::opcua::CommandMethodArgument>(
                    arg
                );
            }) > 1)
            parser.field_err("arguments", "Only one argument can take the command");
    }

    /// @returns whether namespaces hold the namespace of node, and of object for a
    /// method.
    [[nodiscard]] bool resolves(const std::vector<std::string> &namespaces) const {
        return this->node.resolve(namespaces).has_value() &&
               (!this->method || this->object.resolve(namespaces).has_value());
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
            const auto parsed = ::synnax::opcua::parse_write_channel(cp);
            const auto &base = std::visit(
                [](const auto &c) -> const ::synnax::opcua::BaseWriteChannel & {
                    return c;
                },
                parsed
            );
            if (base.disabled) return;
            channels[base.cmd_channel] = std::make_unique<OutputChan>(
                base,
                std::get_if<::synnax::opcua::MethodWriteChannel>(&parsed),
                cp
            );
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
    device::Connector connector;
    connection::Pool::Connection connection;
    /// @brief the data type of each channel's node on the current connection, or
    /// nothing for a node whose type the server has not reported.
    std::unordered_map<synnax::channel::Key, const UA_DataType *> targets;
    /// @brief the data types of the input arguments of each method channel's method on
    /// the current connection, or nothing for a method whose arguments the server has
    /// not reported. A type is null when the client does not know it.
    std::unordered_map<synnax::channel::Key, std::vector<const UA_DataType *>> inputs;
    types::WriteRequestBuilder builder;
    std::vector<synnax::channel::Key> written_keys;

    /// @brief writes on conn from now on.
    /// @returns the error of reading the types of the nodes, after which the sink has
    /// no connection.
    x::errors::Error use(connection::Pool::Connection conn) {
        this->connection = std::move(conn);
        this->targets.clear();
        this->inputs.clear();
        const auto err = this->read_targets(this->cfg.cmd_keys());
        if (err) this->connection = connection::Pool::Connection(nullptr, nullptr, "");
        return err;
    }

    /// @brief reads the DataType of the node of each node channel in keys, and the
    /// input arguments of each method channel's method, so commands convert to them. A
    /// node of a type the client does not know converts by the type of its channel.
    x::errors::Error read_targets(const std::vector<synnax::channel::Key> &keys) {
        if (const auto err = this->read_inputs(keys)) return err;
        const auto &namespaces = this->connection.server().namespaces;
        types::ReadRequestBuilder req;
        std::vector<synnax::channel::Key> read;
        for (const auto key: keys) {
            const auto it = this->cfg.channels.find(key);
            if (it == this->cfg.channels.end() || it->second->method) continue;
            const auto id = it->second->node.resolve(namespaces);
            if (!id.has_value()) continue;
            req.add_node(id.value(), UA_ATTRIBUTEID_DATATYPE);
            read.push_back(key);
        }
        if (req.empty()) return x::errors::NIL;
        auto [res, err] = this->connection.read(req.build());
        if (err) return err;
        for (std::size_t i = 0; i < read.size(); ++i)
            if (const auto &value = res.get().results[i].value;
                UA_Variant_hasScalarType(&value, &UA_TYPES[UA_TYPES_NODEID]))
                this->targets[read[i]] = UA_findDataType(
                    static_cast<const UA_NodeId *>(value.data)
                );
        return x::errors::NIL;
    }

    /// @brief reads the input arguments of the method of each method channel in keys,
    /// so arguments convert to their declared types.
    x::errors::Error read_inputs(const std::vector<synnax::channel::Key> &keys) {
        const auto &namespaces = this->connection.server().namespaces;
        std::vector<synnax::channel::Key> read_keys;
        std::vector<UA_NodeId> methods;
        for (const auto key: keys) {
            const auto it = this->cfg.channels.find(key);
            if (it == this->cfg.channels.end() || !it->second->method) continue;
            if (const auto id = it->second->node.resolve(namespaces); id.has_value()) {
                read_keys.push_back(key);
                methods.push_back(*id);
            }
        }
        if (methods.empty()) return x::errors::NIL;
        auto [read, err] = method::read_inputs(this->connection, methods);
        if (err) return err;
        for (std::size_t i = 0; i < read_keys.size(); ++i) {
            if (!read[i].has_value()) continue;
            auto &types = this->inputs[read_keys[i]];
            for (const auto &arg: read[i].value()) {
                const auto id = arg.data_type.resolve({}).value();
                types.push_back(UA_findDataType(&id));
            }
        }
        return x::errors::NIL;
    }

    x::errors::Error connect() {
        auto [conn, err] = this->connector.connect();
        if (err) return err;
        return this->use(std::move(conn));
    }

public:
    WriteTaskSink(
        std::shared_ptr<synnax::Synnax> client,
        std::shared_ptr<connection::Pool> pool,
        WriteTaskConfig cfg
    ):
        Sink(cfg.cmd_keys()),
        cfg(std::move(cfg)),
        connector(std::move(client), std::move(pool), this->cfg.device, "[opc.write] "),
        connection(nullptr, nullptr, "") {}

    /// @brief retrieves the device again, so a changed trusted server certificate takes
    /// effect on restart. A server that is off or refuses Sessions does not stop the
    /// start, since each command connects again.
    x::errors::Error start() override {
        auto [c, err] = this->connector.start();
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
    /// methods of method channels. A connection that breaks gets one new connection and
    /// one more attempt, so a Session the server closed while idle does not drop the
    /// command.
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
            const auto *fixed = std::get_if<::synnax::opcua::ValueMethodArgument>(
                &ch.arguments[i]
            );
            const auto *type = known ? it->second[i] : nullptr;
            auto [v, err] = fixed == nullptr ? telem::to_variant(s, type)
                                             : telem::to_variant(
                                                   ::x::telem::Series(fixed->value),
                                                   type
                                               );
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

    /// @brief calls the method of each method channel in frame. A call that the server
    /// did not run because the connection broke gets one new connection and one more
    /// attempt. A call that the server might have run is never sent again.
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

    /// @brief reads the server's namespaces again when the frame names a channel whose
    /// namespace the connection lacks, since the server can add a namespace while it
    /// runs. Forgets the types of the nodes and of the method arguments when a
    /// namespace is new.
    x::errors::Error read_namespaces(const ::x::telem::Frame &frame) {
        const auto &namespaces = this->connection.server().namespaces;
        for (const auto &[key, _]: frame) {
            const auto it = this->cfg.channels.find(key);
            if (it == this->cfg.channels.end() || it->second->resolves(namespaces))
                continue;
            types::ReadRequestBuilder req;
            req.add_node(UA_NODEID_NUMERIC(0, UA_NS0ID_SERVER_NAMESPACEARRAY));
            auto [res, err] = this->connection.read(req.build());
            if (err) return err;
            auto [changed, ns_err] = this->connection.update_namespaces(
                res.get().results[0]
            );
            if (!ns_err && changed) {
                this->targets.clear();
                this->inputs.clear();
            }
            return ns_err;
        }
        return x::errors::NIL;
    }

    /// @brief reads the types of each node and method in the frame whose types the sink
    /// lacks, since the server can add a node while it runs.
    x::errors::Error read_untyped(const ::x::telem::Frame &frame) {
        std::vector<synnax::channel::Key> untyped;
        for (const auto &[key, _]: frame) {
            const auto it = this->cfg.channels.find(key);
            if (it == this->cfg.channels.end()) continue;
            const bool known = it->second->method ? this->inputs.contains(key)
                                                  : this->targets.contains(key);
            if (!known) untyped.push_back(key);
        }
        return this->read_targets(untyped);
    }

    /// @brief writes the frame's values and sets problems from build. When the server
    /// rejects the whole request after the client certificate changed, connects again
    /// with the new certificate and writes again.
    std::pair<types::WriteResponse, x::errors::Error>
    send(const ::x::telem::Frame &frame, std::vector<std::string> &problems) {
        const auto attempt =
            [&]() -> std::pair<types::WriteResponse, x::errors::Error> {
            if (auto err = this->read_namespaces(frame))
                return {types::WriteResponse(), err};
            if (auto err = this->read_untyped(frame))
                return {types::WriteResponse(), err};
            problems = this->build(frame);
            if (this->builder.empty()) return {types::WriteResponse(), x::errors::NIL};
            return this->connection.write(this->builder.build());
        };
        auto [res, err] = attempt();
        if (!err || err.matches(errors::UNREACHABLE)) return {std::move(res), err};
        auto [next, refresh_err] = this->connector.refresh(this->connection);
        if (refresh_err) return {std::move(res), refresh_err};
        if (!next) return {std::move(res), err};
        if (const auto use_err = this->use(std::move(next)))
            return {types::WriteResponse(), use_err};
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
