// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <ranges>
#include <string>
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
    std::shared_ptr<UA_Client> client;
    /// @brief the namespace URIs of the server, by index.
    const std::vector<std::string> &namespaces;
    std::shared_ptr<std::vector<types::Node>> channels;
    /// @brief the error of a request that failed during the browse.
    x::errors::Error err;
};

// open62541 combines the returned statuses with a bitwise OR, so a failure is kept in
// the context instead.
static UA_StatusCode
node_iter(UA_NodeId child_id, UA_Boolean is_inverse, UA_NodeId _, void *raw_ctx) {
    auto *ctx = static_cast<ScanContext *>(raw_ctx);
    if (is_inverse || ctx->err) return UA_STATUSCODE_GOOD;
    auto *const ua_client = ctx->client.get();
    // A read on a lost connection would open a new one outside the pool.
    if (ctx->err = connection::lost(ua_client); ctx->err) return UA_STATUSCODE_GOOD;

    UA_ReadValueId ids[3];
    UA_ReadValueId_init(&ids[0]);
    UA_ReadValueId_init(&ids[1]);
    UA_ReadValueId_init(&ids[2]);
    ids[0].nodeId = child_id;
    ids[1].nodeId = child_id;
    ids[2].nodeId = child_id;
    ids[0].attributeId = UA_ATTRIBUTEID_NODECLASS;
    ids[1].attributeId = UA_ATTRIBUTEID_BROWSENAME;
    ids[2].attributeId = UA_ATTRIBUTEID_VALUE;

    UA_ReadRequest req;
    UA_ReadRequest_init(&req);
    req.nodesToRead = ids;
    req.nodesToReadSize = 3;

    types::ReadResponse res(UA_Client_Service_read(ua_client, req));
    ctx->err = errors::parse(res.get().responseHeader.serviceResult);
    if (ctx->err || !res.get().results[0].hasValue || !res.get().results[1].hasValue)
        return UA_STATUSCODE_GOOD;
    UA_NodeClass cls = *static_cast<UA_NodeClass *>(res.get().results[0].value.data);
    auto [ns_index, b_name] = *static_cast<UA_QualifiedName *>(
        res.get().results[1].value.data
    );
    const auto name = std::string(reinterpret_cast<char *>(b_name.data), b_name.length);
    auto data_type = ::x::telem::UNKNOWN_T;
    bool is_array = false;
    if (cls == UA_NODECLASS_VARIABLE && res.get().results[2].hasValue) {
        const auto &value = res.get().results[2].value;
        data_type = telem::ua_to_data_type(value.type);
        is_array = !UA_Variant_isScalar(&value);
    } else if (cls == UA_NODECLASS_VARIABLE)
        LOG(ERROR) << "[opc.scan_task] no value for " << name;
    ctx->channels->emplace_back(
        data_type,
        name,
        types::NodeId::to_string(child_id, ctx->namespaces),
        types::node_class_to_string(cls),
        is_array
    );
    return UA_STATUSCODE_GOOD;
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
    const auto &namespaces = connection.server().namespaces;
    const auto start = args.node.resolve(namespaces);
    if (!start.has_value()) {
        status.message = "Failed to browse nodes";
        status.description = "The server has no namespace " +
                             args.node.namespace_uri() + ".";
        ctx->set_status(status);
        return;
    }
    const auto scan_ctx = std::make_unique<ScanContext>(ScanContext{
        .client = connection.shared(),
        .namespaces = namespaces,
        .channels = std::make_shared<std::vector<types::Node>>(),
    });

    auto browse_err = errors::parse(UA_Client_forEachChildNodeCall(
        scan_ctx->client.get(),
        start.value(),
        node_iter,
        scan_ctx.get()
    ));
    if (!browse_err) browse_err = scan_ctx->err;
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
