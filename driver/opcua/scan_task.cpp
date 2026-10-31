// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <optional>
#include <ranges>
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

/// @returns an error status for the reply to cmd, which each handler fills in.
synnax::task::Status
command_status(const synnax::task::Task &task, const synnax::task::Command &cmd) {
    return {
        .key = synnax::task::status_key(task),
        .name = task.name,
        .variant = synnax::status::VARIANT_ERROR,
        .details = synnax::task::StatusDetails{
            .task = task.key,
            .running = true,
            .cmd = cmd.key,
            .config_hash = task.config_hash,
        }
    };
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
    const auto previous = this->checked[dev.key];
    const bool pin = cfg.secured() && !cfg.certificate_validation_disabled &&
                     !cfg.server_certificate.has_value();
    if (pin || this->conn_pool->last_reply(cfg) <= previous) {
        auto [conn, conn_err] = this->conn_pool->acquire(cfg, SCAN_LOG_PREFIX);
        // Marked after the acquire, so its own reply does not skip the next probe.
        this->checked[dev.key] = x::telem::TimeStamp::now();
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
    } else
        this->checked[dev.key] = x::telem::TimeStamp::now();

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
    auto status = command_status(this->task, cmd);
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

struct ScanContext {
    const connection::Pool::Connection &conn;
    std::shared_ptr<std::vector<types::Node>> channels;
    /// @brief the error of a request that failed during the browse.
    x::errors::Error err;
};

/// @brief reads attributes of one node.
std::pair<types::ReadResponse, x::errors::Error> read_attributes(
    const connection::Pool::Connection &conn,
    const UA_NodeId &id,
    const std::vector<UA_AttributeId> &attributes
) {
    types::ReadRequestBuilder req;
    for (const auto attribute: attributes)
        req.add_node(id, attribute);
    return conn.read(req.build());
}

/// @brief the data type of a variable and whether it holds an array.
struct VariableType {
    x::telem::DataType data_type = x::telem::UNKNOWN_T;
    bool is_array = false;
};

/// @brief finds the type of a variable from its DataType and ValueRank. When the
/// DataType is abstract or unknown, or the ValueRank allows both a scalar and an array,
/// the current value tells the type.
std::pair<VariableType, x::errors::Error> variable_type(
    const connection::Pool::Connection &conn,
    const UA_NodeId &id,
    const UA_DataValue &data_type,
    const UA_DataValue &value_rank
) {
    VariableType type;
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
    if (type.data_type != x::telem::UNKNOWN_T && rank_known)
        return {type, x::errors::NIL};
    auto [res, err] = read_attributes(conn, id, {UA_ATTRIBUTEID_VALUE});
    if (err) return {type, err};
    const auto &value = res.get().results[0];
    if (!value.hasValue) return {type, x::errors::NIL};
    if (type.data_type == x::telem::UNKNOWN_T)
        type.data_type = telem::to_data_type(value.value.type);
    if (!rank_known) type.is_array = !UA_Variant_isScalar(&value.value);
    return {type, x::errors::NIL};
}

// open62541 combines the returned statuses with a bitwise OR, so a failure is kept in
// the context instead.
static UA_StatusCode
node_iter(UA_NodeId child_id, UA_Boolean is_inverse, UA_NodeId _, void *raw_ctx) {
    auto *ctx = static_cast<ScanContext *>(raw_ctx);
    if (is_inverse || ctx->err) return UA_STATUSCODE_GOOD;
    auto [res, err] = read_attributes(
        ctx->conn,
        child_id,
        {UA_ATTRIBUTEID_NODECLASS,
         UA_ATTRIBUTEID_BROWSENAME,
         UA_ATTRIBUTEID_DATATYPE,
         UA_ATTRIBUTEID_VALUERANK}
    );
    ctx->err = err;
    const auto &results = res.get().results;
    if (ctx->err || !results[0].hasValue || !results[1].hasValue)
        return UA_STATUSCODE_GOOD;
    UA_NodeClass cls = *static_cast<UA_NodeClass *>(results[0].value.data);
    auto [ns_index, b_name] = *static_cast<UA_QualifiedName *>(results[1].value.data);
    const auto name = std::string(reinterpret_cast<char *>(b_name.data), b_name.length);
    VariableType type;
    if (cls == UA_NODECLASS_VARIABLE) {
        std::tie(
            type,
            ctx->err
        ) = variable_type(ctx->conn, child_id, results[2], results[3]);
        if (ctx->err) return UA_STATUSCODE_GOOD;
    }
    ctx->channels->emplace_back(
        type.data_type,
        name,
        types::NodeId::to_string(child_id, ctx->conn.server().namespaces),
        types::node_class_to_string(cls),
        type.is_array
    );
    return UA_STATUSCODE_GOOD;
}

/// @brief lists the members of the structure value of a variable, at path. A node that
/// holds no structure has no members.
/// @returns x::errors::VALIDATION if the structure type of the node does not build, or
/// if path is not empty and does not name a structure member of the node's value.
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
    auto [res, err] = read_attributes(
        conn,
        id,
        {UA_ATTRIBUTEID_NODECLASS, UA_ATTRIBUTEID_DATATYPE}
    );
    if (err) return {{}, err};
    const auto &cls = res.get().results[0].value;
    const auto &dt = res.get().results[1].value;
    // The client decodes the NodeClass enumeration as its Int32 encoding.
    if (!UA_Variant_hasScalarType(&cls, &UA_TYPES[UA_TYPES_INT32]) ||
        *static_cast<const UA_Int32 *>(cls.data) != UA_NODECLASS_VARIABLE ||
        !UA_Variant_hasScalarType(&dt, &UA_TYPES[UA_TYPES_NODEID]))
        return none();
    const auto &data_type = *static_cast<const UA_NodeId *>(dt.data);
    auto [types, types_err] = structure::Types::read(conn, {data_type});
    if (types_err) return {{}, types_err};
    auto [field, field_err] = types.field(data_type, path);
    if (field_err.matches(structure::UNDEFINED) && path.empty())
        return {{}, x::errors::NIL};
    if (field_err) return {{}, field_err};
    std::vector<types::Node> members;
    for (const auto &m: structure::Types::members(field.type())) {
        auto member_path = path;
        member_path.push_back(m.name);
        members.emplace_back(
            telem::to_data_type(m.type),
            m.name,
            node_id,
            types::node_class_to_string(UA_NODECLASS_VARIABLE),
            m.is_array,
            structure::to_string(member_path)
        );
    }
    return {std::move(members), x::errors::NIL};
}

void Scanner::browse_nodes(const synnax::task::Command &cmd) const {
    x::json::Parser parser(cmd.args);
    const ScanCommandArgs args(parser);
    auto status = command_status(this->task, cmd);
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
    // A namespace that the server added since the connection opened would print as an
    // index, which a restart can move.
    types::ReadRequestBuilder ns_req;
    ns_req.add_node(UA_NODEID_NUMERIC(0, UA_NS0ID_SERVER_NAMESPACEARRAY));
    auto [ns_res, ns_err] = connection.read(ns_req.build());
    if (!ns_err) ns_err = connection.update_namespaces(ns_res.get().results[0]).second;
    if (ns_err) {
        status.message = "Failed to browse nodes";
        status.description = ns_err.data;
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
    const auto scan_ctx = std::make_unique<ScanContext>(ScanContext{
        .conn = connection,
        .channels = std::make_shared<std::vector<types::Node>>(),
    });

    x::errors::Error browse_err;
    if (args.field.empty()) {
        browse_err = errors::parse(UA_Client_forEachChildNodeCall(
            connection.get(),
            start.value(),
            node_iter,
            scan_ctx.get()
        ));
        if (!browse_err) browse_err = scan_ctx->err;
    }
    x::errors::Error members_err;
    if (!browse_err) {
        auto [members, list_err] = structure_members(
            connection,
            start.value(),
            args.node_id,
            args.field
        );
        for (auto &m: members)
            scan_ctx->channels->push_back(std::move(m));
        // A member has no children besides its own members.
        if (args.field.empty())
            members_err = list_err;
        else
            browse_err = list_err;
    }
    if (browse_err) {
        status.message = "Failed to browse nodes";
        status.description = browse_err.data;
        ctx->set_status(status);
        return;
    }

    x::json::json channels = x::json::json::array();
    for (const auto &ch: *scan_ctx->channels)
        channels.push_back(ch.to_json());
    status.message = "Browsed nodes";
    status.variant = synnax::status::VARIANT_SUCCESS;
    if (members_err) {
        status.message = "Failed to list structure members";
        status.variant = synnax::status::VARIANT_WARNING;
        status.description = members_err.data;
    }
    status.details.data = {{"channels", channels}};
    ctx->set_status(status);
}

void Scanner::test_connection(const synnax::task::Command &cmd) const {
    x::json::Parser parser(cmd.args);
    const ScanCommandArgs args(parser);
    auto status = command_status(this->task, cmd);
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
    auto status = command_status(this->task, cmd);
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
