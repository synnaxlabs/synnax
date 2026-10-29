// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <array>
#include <memory>
#include <optional>
#include <ranges>
#include <set>
#include <string>
#include <tuple>
#include <utility>
#include <vector>

#include "absl/log/log.h"
#include "open62541/client.h"
#include "open62541/client_highlevel.h"
#include "open62541/types.h"

#include "x/cpp/json/json.h"

#include "driver/opcua/device/device.h"
#include "driver/opcua/errors/errors.h"
#include "driver/opcua/scan_task.h"
#include "driver/opcua/structure/structure.h"
#include "driver/opcua/telem/telem.h"
#include "driver/opcua/types/types.h"

namespace driver::opcua {
namespace {
/// @returns the status message for a failed connection, or the fallback for a failure
/// that is not about certificates.
std::string failure_message(
    const x::errors::Error &err,
    const connection::Config &cfg,
    const std::string &fallback
) {
    if (err.matches(errors::SERVER_CERTIFICATE_UNTRUSTED))
        return "Server certificate not trusted";
    if (err.matches(errors::SERVER_CERTIFICATE_TIME_INVALID))
        return "Server certificate expired";
    if (err.matches(errors::CLIENT_CERTIFICATE_REJECTED))
        return cfg.uses_files() ? "Server rejected the client certificate"
                                : "Server rejected the Driver certificate";
    if (err.matches(errors::CLIENT_CERTIFICATE_INVALID))
        return "Failed to read client certificate";
    if (err.matches(errors::DRIVER_CERTIFICATE_FAILED))
        return "Failed to load the Driver certificate";
    return fallback;
}
}

Scanner::Scanner(
    std::shared_ptr<task::Context> ctx,
    synnax::task::Task task,
    std::shared_ptr<connection::Pool> conn_pool,
    std::shared_ptr<certificate::Store> store
):
    ctx(std::move(ctx)),
    task(std::move(task)),
    conn_pool(std::move(conn_pool)),
    store(std::move(store)) {}

common::ScannerConfig Scanner::config() const {
    return common::ScannerConfig{
        .make = INTEGRATION_NAME,
        .log_prefix = SCAN_LOG_PREFIX
    };
}

std::pair<std::vector<synnax::device::Device>, x::errors::Error>
Scanner::scan(const common::ScannerContext &scan_ctx) {
    std::vector<synnax::device::Device> devices;
    if (scan_ctx.devices == nullptr) return {devices, x::errors::NIL};
    for (const auto &dev: *scan_ctx.devices | std::views::values)
        devices.push_back(dev);
    this->conn_pool->close_idle();
    const auto renew_err = this->renew_certificate(devices);
    for (auto &dev: devices)
        if (const auto err = this->check_device_health(dev); err)
            LOG(WARNING) << SCAN_LOG_PREFIX << "health check failed for " << dev.name
                         << ": " << err;
    if (renew_err)
        return {
            devices,
            x::errors::Error(
                common::PARTIAL_SCAN_ERROR,
                "Failed to renew the Driver certificate: " + renew_err.data
            )
        };
    return {devices, x::errors::NIL};
}

bool Scanner::exec(
    synnax::task::Command &cmd,
    const synnax::task::Task &,
    const std::shared_ptr<task::Context> &
) {
    if (cmd.type == BROWSE_CMD_TYPE) {
        this->browse_nodes(cmd);
        return true;
    }
    if (cmd.type == BROWSE_VARIABLES_CMD_TYPE) {
        this->browse_variables(cmd);
        return true;
    }
    if (cmd.type == READ_NODE_CMD_TYPE) {
        this->read_node(cmd);
        return true;
    }
    if (cmd.type == TEST_CONNECTION_CMD_TYPE) {
        this->test_connection(cmd);
        return true;
    }
    if (cmd.type == GET_ENDPOINTS_CMD_TYPE) {
        this->endpoints(cmd);
        return true;
    }
    if (cmd.type == GET_CERTIFICATE_CMD_TYPE) {
        this->certificates(cmd, false);
        return true;
    }
    if (cmd.type == REGENERATE_CERTIFICATE_CMD_TYPE) {
        this->certificates(cmd, true);
        return true;
    }
    return false; // Not handled
}

x::errors::Error Scanner::check_device_health(synnax::device::Device &dev) {
    const auto rack_key = this->task.rack;
    const auto parser = x::json::Parser(dev.properties);
    const auto props = device::Properties(parser);
    synnax::device::Status status{
        .key = synnax::device::status_key(dev),
        .name = dev.name,
        .variant = synnax::status::VARIANT_SUCCESS,
        .message = "Server connected",
        .time = ::x::telem::TimeStamp::now(),
        .details = {.rack = rack_key, .device = dev.key},
    };
    if (parser.error()) {
        status.variant = synnax::status::VARIANT_WARNING;
        status.message = "Invalid device properties";
        status.description = common::config_err_description(parser.error());
        dev.status = status;
        return parser.error();
    }
    const auto &cfg = props.connection;
    const auto previous = std::exchange(
        this->checked[dev.key],
        x::telem::TimeStamp::now()
    );
    const bool pin = cfg.secured() && !cfg.certificate_validation_disabled &&
                     !cfg.server_certificate.has_value();
    if (pin || this->conn_pool->last_reply(cfg) <= previous) {
        auto [conn, conn_err] = this->conn_pool->acquire(cfg, SCAN_LOG_PREFIX);
        if (conn_err) {
            status.message = failure_message(conn_err, cfg, "Failed to reach server");
            status.variant = conn_err.matches(errors::SECURITY)
                               ? synnax::status::VARIANT_ERROR
                               : synnax::status::VARIANT_WARNING;
            status.description = conn_err.data;
            dev.status = status;
            return x::errors::NIL;
        }
        if (pin)
            if (const auto cert = connection::server_certificate(conn.get())) {
                LOG(INFO) << SCAN_LOG_PREFIX << "trusting server certificate "
                          << cert->thumbprint() << " for " << dev.name;
                dev.properties["connection"]["server_certificate"] = cert->pem();
            }
    }

    const auto accepted = this->renewal.accepted.find(dev.key);
    if (cfg.secured() && cfg.certificate_validation_disabled) {
        status.variant = synnax::status::VARIANT_WARNING;
        status.message = "Connected without certificate validation";
        status.description = "The Driver accepts any server certificate for this "
                             "device, so it cannot detect a server that impersonates "
                             "this one. Turn validation on again when you finish "
                             "troubleshooting.";
    } else if (accepted != this->renewal.accepted.end() && !accepted->second) {
        auto [current, current_err] = this->store->current();
        if (current_err) return current_err;
        status.variant = synnax::status::VARIANT_WARNING;
        status.message = "Server does not trust the next certificate";
        status.description = "Approve the next Driver certificate " +
                             this->renewal.thumbprint + " on the server before " +
                             current.certificate.not_after().iso8601() +
                             ", when the current certificate expires.";
    }
    dev.status = status;
    return x::errors::NIL;
}

x::errors::Error
Scanner::renew_certificate(const std::vector<synnax::device::Device> &devices) {
    if (const auto err = this->store->renew()) return err;
    auto [next, next_err] = this->store->next();
    if (next_err) return next_err;
    if (!next.has_value()) {
        this->renewal = Renewal{};
        return x::errors::NIL;
    }
    const auto thumbprint = next->certificate.thumbprint();
    if (this->renewal.thumbprint != thumbprint)
        this->renewal = {.thumbprint = thumbprint};
    const auto now = x::telem::TimeStamp::now();
    if (this->renewal.last_attempt != x::telem::TimeStamp() &&
        now - this->renewal.last_attempt < RENEWAL_ATTEMPT_INTERVAL)
        return x::errors::NIL;
    this->renewal.last_attempt = now;
    this->renewal.accepted.clear();
    bool all_accepted = true;
    for (const auto &dev: devices) {
        const auto parser = x::json::Parser(dev.properties);
        const auto props = device::Properties(parser);
        if (parser.error() || !props.connection.secured() ||
            props.connection.uses_files())
            continue;
        const connection::Security sec{
            .identity = next,
            .first_certificate_trusted = true,
        };
        const auto err = connection::connect(props.connection, sec, SCAN_LOG_PREFIX)
                             .second;
        if (err && !err.matches(errors::CLIENT_CERTIFICATE_REJECTED)) {
            all_accepted = false;
            continue;
        }
        this->renewal.accepted[dev.key] = !err;
        all_accepted = all_accepted && !err;
    }
    if (!all_accepted) return x::errors::NIL;
    if (const auto err = this->store->promote()) return err;
    LOG(INFO) << SCAN_LOG_PREFIX << "switched to the next certificate " << thumbprint;
    this->renewal = Renewal{};
    return x::errors::NIL;
}

void Scanner::certificates(
    const synnax::task::Command &cmd,
    const bool regenerate
) const {
    synnax::task::Status status{
        .key = synnax::task::status_key(this->task),
        .name = this->task.name,
        .variant = synnax::status::VARIANT_ERROR,
        .details = synnax::task::StatusDetails{
            .task = task.key,
            .running = true,
            .cmd = cmd.key,
            .config_hash = task.config_hash,
        }
    };
    x::errors::Error err = x::errors::NIL;
    if (regenerate) err = this->store->regenerate().second;
    auto [current, current_err] = this->store->current();
    auto [next, next_err] = this->store->next();
    if (!err) err = current_err ? current_err : next_err;
    if (err) {
        status.message = regenerate ? "Failed to regenerate the Driver certificate"
                                    : "Failed to load the Driver certificate";
        status.description = err.data;
        ctx->set_status(status);
        return;
    }
    x::json::json data = {{"current", current.certificate.info().to_json()}};
    if (next.has_value()) data["next"] = next->certificate.info().to_json();
    status.variant = synnax::status::VARIANT_SUCCESS;
    status.message = regenerate ? "Regenerated the Driver certificate"
                                : "Retrieved the Driver certificate";
    status.details.data = data;
    ctx->set_status(status);
}

namespace {
/// @brief the data type of a variable and whether it holds an array.
struct VariableType {
    x::telem::DataType data_type = x::telem::UNKNOWN_T;
    bool is_array = false;
};

/// @brief finds the types of the variables that refs name from their DataType and
/// ValueRank. When a DataType is abstract or unknown, or a ValueRank allows both a
/// scalar and an array, the current value tells the type.
/// @returns one type for each reference, unknown for a node that is not a variable.
std::pair<std::vector<VariableType>, x::errors::Error> variable_types(
    const connection::Pool::Connection &conn,
    const std::vector<types::Reference> &refs
) {
    std::vector<VariableType> out(refs.size());
    std::vector<size_t> variables;
    types::ReadRequestBuilder req;
    for (size_t i = 0; i < refs.size(); ++i) {
        if (refs[i].node_class != UA_NODECLASS_VARIABLE) continue;
        req.add_node(refs[i].node, UA_ATTRIBUTEID_DATATYPE);
        req.add_node(refs[i].node, UA_ATTRIBUTEID_VALUERANK);
        variables.push_back(i);
    }
    if (variables.empty()) return {std::move(out), x::errors::NIL};
    auto [res, err] = conn.read(req.build());
    if (err) return {{}, err};
    struct Unknown {
        size_t ref;
        bool rank_known;
    };
    std::vector<Unknown> unknown;
    types::ReadRequestBuilder values;
    for (size_t j = 0; j < variables.size(); ++j) {
        const auto &data_type = res.get().results[2 * j];
        const auto &value_rank = res.get().results[2 * j + 1];
        auto &type = out[variables[j]];
        if (data_type.hasValue &&
            UA_Variant_hasScalarType(&data_type.value, &UA_TYPES[UA_TYPES_NODEID]))
            type.data_type = telem::to_data_type(
                *static_cast<const UA_NodeId *>(data_type.value.data)
            );
        std::optional<UA_Int32> rank;
        if (value_rank.hasValue &&
            UA_Variant_hasScalarType(&value_rank.value, &UA_TYPES[UA_TYPES_INT32]))
            rank = *static_cast<const UA_Int32 *>(value_rank.value.data);
        const bool rank_known = rank.has_value() && *rank >= UA_VALUERANK_SCALAR;
        type.is_array = rank_known && *rank != UA_VALUERANK_SCALAR;
        if (type.data_type != x::telem::UNKNOWN_T && rank_known) continue;
        values.add_node(refs[variables[j]].node, UA_ATTRIBUTEID_VALUE);
        unknown.push_back({.ref = variables[j], .rank_known = rank_known});
    }
    if (unknown.empty()) return {std::move(out), x::errors::NIL};
    auto [value_res, value_err] = conn.read(values.build());
    if (value_err) return {{}, value_err};
    for (size_t k = 0; k < unknown.size(); ++k) {
        const auto &value = value_res.get().results[k];
        if (!value.hasValue) continue;
        auto &type = out[unknown[k].ref];
        if (type.data_type == x::telem::UNKNOWN_T)
            type.data_type = telem::ua_to_data_type(value.value.type);
        if (!unknown[k].rank_known) type.is_array = !UA_Variant_isScalar(&value.value);
    }
    return {std::move(out), x::errors::NIL};
}

/// @returns the nodes that refs name, with the types of the variables.
std::pair<std::vector<types::Node>, x::errors::Error> to_nodes(
    const connection::Pool::Connection &conn,
    const std::vector<types::Reference> &refs
) {
    auto [variables, err] = variable_types(conn, refs);
    if (err) return {{}, err};
    std::vector<types::Node> nodes;
    nodes.reserve(refs.size());
    for (size_t i = 0; i < refs.size(); ++i)
        nodes.emplace_back(
            variables[i].data_type,
            refs[i].browse_name,
            types::NodeId::to_string(refs[i].node, conn.server().namespaces),
            types::node_class_to_string(refs[i].node_class),
            variables[i].is_array
        );
    return {std::move(nodes), x::errors::NIL};
}

/// @brief lists the members of the structure value of a variable, at path. A node that
/// holds no structure has no members.
/// @returns x::errors::VALIDATION if path is not empty and does not name a structure
/// member of the node's value.
std::pair<std::vector<types::Node>, x::errors::Error> structure_members(
    const connection::Pool::Connection &conn,
    const UA_NodeId &id,
    const std::string &node_id,
    const structure::Path &path
) {
    const auto none =
        [&path]() -> std::pair<std::vector<types::Node>, x::errors::Error> {
        if (path.empty()) return {{}, x::errors::NIL};
        return {
            {},
            x::errors::Error(
                x::errors::VALIDATION,
                "the node does not hold a structure"
            )
        };
    };
    types::ReadRequestBuilder req;
    req.add_node(id, UA_ATTRIBUTEID_NODECLASS);
    req.add_node(id, UA_ATTRIBUTEID_DATATYPE);
    auto [res, err] = conn.read(req.build());
    if (err) return {{}, err};
    const auto &cls = res.get().results[0].value;
    const auto &dt = res.get().results[1].value;
    // The client decodes the NodeClass enumeration as its Int32 encoding.
    if (!UA_Variant_hasScalarType(&cls, &UA_TYPES[UA_TYPES_INT32]) ||
        *static_cast<const UA_Int32 *>(cls.data) != UA_NODECLASS_VARIABLE ||
        !UA_Variant_hasScalarType(&dt, &UA_TYPES[UA_TYPES_NODEID]))
        return none();
    const auto &data_type = *static_cast<const UA_NodeId *>(dt.data);
    structure::Types types;
    if (UA_findDataType(&data_type) == nullptr) {
        std::tie(types, err) = structure::Types::read(conn, {id});
        if (err) return {{}, err};
    }
    const auto *type = types.find(data_type);
    if (type == nullptr) return none();
    auto [member_type, member_err] = structure::Types::member_type(*type, path);
    if (member_err) return {{}, member_err};
    std::vector<types::Node> members;
    for (const auto &m: structure::Types::members(*member_type)) {
        auto member_path = path;
        member_path.push_back(m.name);
        members.emplace_back(
            telem::ua_to_data_type(m.type),
            m.name,
            node_id,
            types::node_class_to_string(UA_NODECLASS_VARIABLE),
            m.is_array,
            structure::to_string(member_path)
        );
    }
    return {std::move(members), x::errors::NIL};
}

/// @brief a variable that a search under a node found.
struct Found {
    types::Reference ref;
    /// @brief the browse names from below the node where the search started.
    std::vector<std::string> path;
};

/// @brief what a search for the variables under a node found.
struct Search {
    std::vector<Found> variables;
    /// @brief whether the search stopped at a limit before it visited every object.
    bool truncated = false;
};

/// @brief finds the variables under a node, through objects at any depth, breadth
/// first. Does not look under a variable. A node that fails to browse is skipped.
std::pair<Search, x::errors::Error>
find_variables(const connection::Pool::Connection &conn, const UA_NodeId &start) {
    struct Object {
        types::NodeId node;
        std::vector<std::string> path;
    };
    Search search;
    std::set<std::string> visited{types::NodeId::to_string(start)};
    std::vector<Object> objects;
    objects.push_back({.node = types::NodeId(start), .path = {}});
    size_t browsed = 0;
    while (!objects.empty()) {
        if (search.variables.size() >= MAX_VARIABLES || browsed >= MAX_OBJECTS) {
            search.truncated = true;
            break;
        }
        if (objects.size() > MAX_OBJECTS - browsed) {
            objects.resize(MAX_OBJECTS - browsed);
            search.truncated = true;
        }
        browsed += objects.size();
        std::vector<UA_NodeId> ids;
        ids.reserve(objects.size());
        for (const auto &object: objects)
            ids.push_back(object.node.get());
        auto [results, err] = conn.browse(ids);
        if (err) return {Search{}, err};
        std::vector<Object> next;
        for (size_t i = 0; i < results.size(); ++i) {
            if (results[i].err) {
                LOG(WARNING) << SCAN_LOG_PREFIX << "skipped "
                             << types::NodeId::to_string(objects[i].node)
                             << " in a search for variables: " << results[i].err;
                continue;
            }
            for (auto &ref: results[i].references) {
                if (ref.node_class != UA_NODECLASS_VARIABLE &&
                    ref.node_class != UA_NODECLASS_OBJECT)
                    continue;
                if (!visited.insert(types::NodeId::to_string(ref.node)).second)
                    continue;
                auto path = objects[i].path;
                path.push_back(ref.browse_name);
                if (ref.node_class == UA_NODECLASS_OBJECT)
                    next.push_back(
                        {.node = std::move(ref.node), .path = std::move(path)}
                    );
                else
                    search.variables.push_back(
                        {.ref = std::move(ref), .path = std::move(path)}
                    );
            }
        }
        objects = std::move(next);
    }
    if (search.variables.size() > MAX_VARIABLES) {
        search.variables.resize(MAX_VARIABLES);
        search.truncated = true;
    }
    return {std::move(search), x::errors::NIL};
}

/// @returns the DateTime as ISO 8601 text, or nothing when the server did not set it
/// or a timestamp cannot hold it.
std::optional<std::string> date_text(UA_DateTime time) {
    if (time == 0) return std::nullopt;
    UA_Variant v;
    UA_Variant_setScalar(&v, &time, &UA_TYPES[UA_TYPES_DATETIME]);
    const auto [sample, err] = telem::to_sample(v, x::telem::TIMESTAMP_T);
    if (err) return std::nullopt;
    return std::get<x::telem::TimeStamp>(sample).iso8601();
}

std::string text(const UA_String &s) {
    return std::string(reinterpret_cast<const char *>(s.data), s.length);
}

/// @returns the scalar of type T that the result holds, or nothing.
template<typename T>
std::optional<T> scalar(const UA_DataValue &result, const UA_DataType &type) {
    if (!result.hasValue || !UA_Variant_hasScalarType(&result.value, &type))
        return std::nullopt;
    return *static_cast<const T *>(result.value.data);
}

/// @returns the name of the data type, reading its BrowseName when the client does
/// not know it.
std::pair<std::string, x::errors::Error>
type_name(const connection::Pool::Connection &conn, const UA_NodeId &data_type) {
    if (const auto *type = UA_findDataType(&data_type))
        return {type->typeName, x::errors::NIL};
    types::ReadRequestBuilder req;
    req.add_node(data_type, UA_ATTRIBUTEID_BROWSENAME);
    auto [res, err] = conn.read(req.build());
    if (err) return {"", err};
    const auto name = scalar<UA_QualifiedName>(
        res.get().results[0],
        UA_TYPES[UA_TYPES_QUALIFIEDNAME]
    );
    if (!name.has_value()) return {types::NodeId::to_string(data_type), x::errors::NIL};
    return {text(name->name), x::errors::NIL};
}
}

void Scanner::browse_nodes(const synnax::task::Command &cmd) const {
    x::json::Parser parser(cmd.args);
    const ScanCommandArgs args(parser);
    synnax::task::Status status{
        .key = synnax::task::status_key(this->task),
        .name = this->task.name,
        .variant = synnax::status::VARIANT_ERROR,
        .details = synnax::task::StatusDetails{
            .task = task.key,
            .cmd = cmd.key,
            .config_hash = task.config_hash,
        }
    };
    if (!parser.ok()) {
        status.message = "Failed to parse scan command";
        status.description = common::config_err_description(parser.error());
        status.details.data = parser.error_json();
        ctx->set_status(status);
        return;
    }

    auto [connection, err] = conn_pool->acquire(args.connection, SCAN_LOG_PREFIX);
    if (err) {
        status.message = "Failed to connect to server";
        status.description = err.data;
        ctx->set_status(status);
        return;
    }
    const auto start = args.node.resolve(connection.server().namespaces);
    if (!start.has_value()) {
        status.message = "Failed to browse nodes";
        status.description = "The server has no namespace " +
                             args.node.namespace_uri() + ".";
        ctx->set_status(status);
        return;
    }

    std::vector<types::Node> nodes;
    if (args.field.empty()) {
        auto [results, browse_err] = connection.browse({start.value()});
        err = browse_err ? browse_err : results[0].err;
        if (!err) std::tie(nodes, err) = to_nodes(connection, results[0].references);
    }
    if (!err) {
        auto [members, members_err] = structure_members(
            connection,
            start.value(),
            args.node_id,
            args.field
        );
        err = members_err;
        for (auto &m: members)
            nodes.push_back(std::move(m));
    }
    if (err) {
        status.message = "Failed to browse nodes";
        status.description = err.data;
        ctx->set_status(status);
        return;
    }

    x::json::json channels = x::json::json::array();
    for (const auto &node: nodes)
        channels.push_back(node.to_json());
    status.message = "Browsed nodes";
    status.variant = synnax::status::VARIANT_SUCCESS;
    status.details.data = {{"channels", channels}};
    ctx->set_status(status);
}

void Scanner::browse_variables(const synnax::task::Command &cmd) const {
    x::json::Parser parser(cmd.args);
    const ScanCommandArgs args(parser);
    synnax::task::Status status{
        .key = synnax::task::status_key(this->task),
        .name = this->task.name,
        .variant = synnax::status::VARIANT_ERROR,
        .details = synnax::task::StatusDetails{
            .task = task.key,
            .cmd = cmd.key,
            .config_hash = task.config_hash,
        }
    };
    if (!parser.ok()) {
        status.message = "Failed to parse scan command";
        status.description = common::config_err_description(parser.error());
        status.details.data = parser.error_json();
        ctx->set_status(status);
        return;
    }
    auto [connection, err] = conn_pool->acquire(args.connection, SCAN_LOG_PREFIX);
    if (err) {
        status.message = "Failed to connect to server";
        status.description = err.data;
        ctx->set_status(status);
        return;
    }
    const auto start = args.node.resolve(connection.server().namespaces);
    if (!start.has_value()) {
        status.message = "Failed to find variables";
        status.description = "The server has no namespace " +
                             args.node.namespace_uri() + ".";
        ctx->set_status(status);
        return;
    }
    auto [search, search_err] = find_variables(connection, start.value());
    std::vector<types::Reference> refs;
    refs.reserve(search.variables.size());
    for (auto &found: search.variables)
        refs.push_back(std::move(found.ref));
    std::vector<types::Node> nodes;
    err = search_err;
    if (!err) std::tie(nodes, err) = to_nodes(connection, refs);
    if (err) {
        status.message = "Failed to find variables";
        status.description = err.data;
        ctx->set_status(status);
        return;
    }
    x::json::json channels = x::json::json::array();
    for (size_t i = 0; i < nodes.size(); ++i) {
        auto node = nodes[i].to_json();
        node["path"] = search.variables[i].path;
        channels.push_back(node);
    }
    status.message = "Found variables";
    status.variant = synnax::status::VARIANT_SUCCESS;
    status.details.data = {{"channels", channels}, {"truncated", search.truncated}};
    ctx->set_status(status);
}

void Scanner::read_node(const synnax::task::Command &cmd) const {
    x::json::Parser parser(cmd.args);
    const ScanCommandArgs args(parser);
    synnax::task::Status status{
        .key = synnax::task::status_key(this->task),
        .name = this->task.name,
        .variant = synnax::status::VARIANT_ERROR,
        .details = synnax::task::StatusDetails{
            .task = task.key,
            .cmd = cmd.key,
            .config_hash = task.config_hash,
        }
    };
    if (!parser.ok()) {
        status.message = "Failed to parse scan command";
        status.description = common::config_err_description(parser.error());
        status.details.data = parser.error_json();
        ctx->set_status(status);
        return;
    }
    auto [connection, err] = conn_pool->acquire(args.connection, SCAN_LOG_PREFIX);
    if (err) {
        status.message = "Failed to connect to server";
        status.description = err.data;
        ctx->set_status(status);
        return;
    }
    const auto &namespaces = connection.server().namespaces;
    const auto id = args.node.resolve(namespaces);
    if (!id.has_value()) {
        status.message = "Failed to read node";
        status.description = "The server has no namespace " +
                             args.node.namespace_uri() + ".";
        ctx->set_status(status);
        return;
    }
    const auto details = [&]() -> std::pair<x::json::json, x::errors::Error> {
        constexpr std::array attributes{
            UA_ATTRIBUTEID_NODECLASS,
            UA_ATTRIBUTEID_BROWSENAME,
            UA_ATTRIBUTEID_DISPLAYNAME,
            UA_ATTRIBUTEID_DESCRIPTION,
            UA_ATTRIBUTEID_DATATYPE,
            UA_ATTRIBUTEID_VALUERANK,
            UA_ATTRIBUTEID_ARRAYDIMENSIONS,
            UA_ATTRIBUTEID_USERACCESSLEVEL,
            UA_ATTRIBUTEID_HISTORIZING,
            UA_ATTRIBUTEID_MINIMUMSAMPLINGINTERVAL,
            UA_ATTRIBUTEID_VALUE,
        };
        types::ReadRequestBuilder req;
        for (const auto attribute: attributes)
            req.add_node(id.value(), attribute);
        auto raw_req = req.build();
        raw_req.timestampsToReturn = UA_TIMESTAMPSTORETURN_BOTH;
        auto [res, read_err] = connection.read(raw_req);
        if (read_err) return {{}, read_err};
        const auto *r = res.get().results;
        if (const auto node_err = errors::parse(r[0].status)) return {{}, node_err};
        // The client decodes the NodeClass enumeration as its Int32 encoding.
        const auto cls = static_cast<UA_NodeClass>(
            scalar<UA_Int32>(r[0], UA_TYPES[UA_TYPES_INT32]).value_or(0)
        );
        const auto browse_name = scalar<UA_QualifiedName>(
            r[1],
            UA_TYPES[UA_TYPES_QUALIFIEDNAME]
        );
        x::json::json out = {
            {"node_class", types::node_class_to_string(cls)},
            {"browse_name", browse_name.has_value() ? text(browse_name->name) : ""},
        };
        if (const auto v = scalar<UA_LocalizedText>(
                r[2],
                UA_TYPES[UA_TYPES_LOCALIZEDTEXT]
            ))
            out["display_name"] = text(v->text);
        if (const auto v = scalar<UA_LocalizedText>(
                r[3],
                UA_TYPES[UA_TYPES_LOCALIZEDTEXT]
            ))
            out["description"] = text(v->text);
        auto data_type = x::telem::UNKNOWN_T;
        auto is_array = false;
        if (cls == UA_NODECLASS_VARIABLE) {
            const auto type_id = scalar<UA_NodeId>(r[4], UA_TYPES[UA_TYPES_NODEID]);
            const auto rank = scalar<UA_Int32>(r[5], UA_TYPES[UA_TYPES_INT32]);
            if (type_id.has_value()) {
                auto [name, name_err] = type_name(connection, *type_id);
                if (name_err) return {{}, name_err};
                out["type_name"] = name;
                data_type = telem::to_data_type(*type_id);
            }
            if (rank.has_value()) {
                out["value_rank"] = *rank;
                is_array = *rank != UA_VALUERANK_SCALAR && *rank >= UA_VALUERANK_SCALAR;
            }
            if (r[6].hasValue &&
                UA_Variant_hasArrayType(&r[6].value, &UA_TYPES[UA_TYPES_UINT32])) {
                const auto *dims = static_cast<const UA_UInt32 *>(r[6].value.data);
                out["array_dimensions"] = std::vector<UA_UInt32>(
                    dims,
                    dims + r[6].value.arrayLength
                );
            }
            if (const auto access = scalar<UA_Byte>(r[7], UA_TYPES[UA_TYPES_BYTE])) {
                out["readable"] = (*access & UA_ACCESSLEVELMASK_READ) != 0;
                out["writable"] = (*access & UA_ACCESSLEVELMASK_WRITE) != 0;
                out["history_readable"] = (*access & UA_ACCESSLEVELMASK_HISTORYREAD) !=
                                          0;
            }
            if (const auto v = scalar<UA_Boolean>(r[8], UA_TYPES[UA_TYPES_BOOLEAN]))
                out["historizing"] = *v;
            if (const auto v = scalar<UA_Double>(r[9], UA_TYPES[UA_TYPES_DOUBLE]))
                out["minimum_sampling_interval"] = *v;
            const auto &value = r[10];
            out["status"] = UA_StatusCode_name(value.status);
            if (value.hasSourceTimestamp)
                if (const auto t = date_text(value.sourceTimestamp))
                    out["source_timestamp"] = *t;
            if (value.hasServerTimestamp)
                if (const auto t = date_text(value.serverTimestamp))
                    out["server_timestamp"] = *t;
            structure::Types types;
            if (!args.field.empty() ||
                (value.value.type != nullptr &&
                 value.value.type->typeKind == UA_DATATYPEKIND_EXTENSIONOBJECT)) {
                auto [read_types, types_err] = structure::Types::read(
                    connection,
                    {id.value()}
                );
                if (types_err) return {{}, types_err};
                types = std::move(read_types);
            }
            if (value.hasValue) {
                auto [value_text, text_err] = types.print(value.value, args.field);
                if (text_err) return {{}, text_err};
                out["value"] = value_text;
            }
            if (!args.field.empty()) {
                const auto *type = type_id.has_value() ? types.find(*type_id) : nullptr;
                if (type == nullptr)
                    return {
                        {},
                        x::errors::Error(
                            x::errors::VALIDATION,
                            "the node does not hold a structure"
                        )
                    };
                auto [member, member_err] = structure::Types::member_type(
                    *type,
                    args.field
                );
                if (member_err) return {{}, member_err};
                out["type_name"] = member->typeName;
                data_type = telem::ua_to_data_type(member);
                is_array = false;
            }
        }
        out["node"] = types::Node(
                          data_type,
                          args.field.empty() ? out["browse_name"].get<std::string>()
                                             : args.field.back(),
                          types::NodeId::to_string(id.value(), namespaces),
                          types::node_class_to_string(cls),
                          is_array,
                          structure::to_string(args.field)
        )
                          .to_json();
        return {out, x::errors::NIL};
    };
    auto [data, details_err] = details();
    if (details_err) {
        status.message = "Failed to read node";
        status.description = details_err.data;
        ctx->set_status(status);
        return;
    }
    status.message = "Read node";
    status.variant = synnax::status::VARIANT_SUCCESS;
    status.details.data = data;
    ctx->set_status(status);
}

void Scanner::test_connection(const synnax::task::Command &cmd) const {
    x::json::Parser parser(cmd.args);
    const ScanCommandArgs args(parser);
    synnax::task::Status status{
        .key = synnax::task::status_key(this->task),
        .name = this->task.name,
        .variant = synnax::status::VARIANT_ERROR,
        .details = synnax::task::StatusDetails{
            .task = task.key,
            .running = true,
            .cmd = cmd.key,
            .config_hash = task.config_hash,
        }
    };
    if (!parser.ok()) {
        status.message = "Failed to parse test command";
        status.description = common::config_err_description(parser.error());
        status.details.data = parser.error_json();
        ctx->set_status(status);
        return;
    }
    auto [sec, sec_err] = connection::resolve_security(args.connection, *this->store);
    if (sec_err) {
        status.message = failure_message(
            sec_err,
            args.connection,
            "Failed to connect to server"
        );
        status.description = sec_err.data;
        ctx->set_status(status);
        return;
    }
    auto [client, err] = connection::connect(args.connection, sec, SCAN_LOG_PREFIX);
    if (err) {
        status.message = failure_message(
            err,
            args.connection,
            "Failed to connect to server"
        );
        status.description = err.data;
        if (err.matches(errors::SERVER_CERTIFICATE_UNTRUSTED))
            if (const auto cert = connection::server_certificate(client.get()))
                status.details.data = {{"server_certificate", cert->info().to_json()}};
        ctx->set_status(status);
        return;
    }
    status.variant = synnax::status::VARIANT_SUCCESS;
    status.message = "Connected to server";
    ctx->set_status(status);
}

void Scanner::endpoints(const synnax::task::Command &cmd) const {
    x::json::Parser parser(cmd.args);
    const connection::Config cfg(parser.child("connection"));
    synnax::task::Status status{
        .key = synnax::task::status_key(this->task),
        .name = this->task.name,
        .variant = synnax::status::VARIANT_ERROR,
        .details = synnax::task::StatusDetails{
            .task = task.key,
            .running = true,
            .cmd = cmd.key,
            .config_hash = task.config_hash,
        }
    };
    if (!parser.ok()) {
        status.message = "Failed to parse endpoints command";
        status.description = common::config_err_description(parser.error());
        status.details.data = parser.error_json();
        ctx->set_status(status);
        return;
    }
    auto [endpoints, err] = connection::endpoints(cfg);
    if (err) {
        status.message = "Failed to get endpoints";
        status.description = err.data;
        ctx->set_status(status);
        return;
    }
    x::json::json data = x::json::json::array();
    for (const auto &endpoint: endpoints)
        data.push_back(endpoint.to_json());
    status.variant = synnax::status::VARIANT_SUCCESS;
    status.message = "Received endpoints";
    status.details.data = {{"endpoints", data}};
    ctx->set_status(status);
}
}
