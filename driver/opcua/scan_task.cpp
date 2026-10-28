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
    this->renew_certificate(devices);
    for (auto &dev: devices)
        if (const auto err = this->check_device_health(dev); err)
            LOG(WARNING) << SCAN_LOG_PREFIX << "health check failed for " << dev.name
                         << ": " << err;
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

    if (cfg.secured() && !cfg.certificate_validation_disabled &&
        !cfg.server_certificate.has_value())
        if (const auto cert = connection::server_certificate(conn.get())) {
            LOG(INFO) << SCAN_LOG_PREFIX << "trusting server certificate "
                      << cert->thumbprint() << " for " << dev.name;
            dev.properties["connection"]["server_certificate"] = cert->pem();
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

void Scanner::renew_certificate(const std::vector<synnax::device::Device> &devices) {
    if (const auto err = this->store->renew())
        LOG(WARNING) << SCAN_LOG_PREFIX
                     << "failed to issue the next certificate: " << err;
    auto [next, next_err] = this->store->next();
    if (next_err) {
        LOG(WARNING) << SCAN_LOG_PREFIX
                     << "failed to load the next certificate: " << next_err;
        return;
    }
    if (!next.has_value()) {
        this->renewal = Renewal{};
        return;
    }
    const auto thumbprint = next->certificate.thumbprint();
    if (this->renewal.thumbprint != thumbprint)
        this->renewal = {.thumbprint = thumbprint};
    const auto now = x::telem::TimeStamp::now();
    if (this->renewal.last_attempt != x::telem::TimeStamp() &&
        now - this->renewal.last_attempt < RENEWAL_ATTEMPT_INTERVAL)
        return;
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
    if (!all_accepted) return;
    if (const auto err = this->store->promote()) {
        LOG(WARNING) << SCAN_LOG_PREFIX << "failed to switch certificates: " << err;
        return;
    }
    LOG(INFO) << SCAN_LOG_PREFIX << "switched to the next certificate " << thumbprint;
    this->renewal = Renewal{};
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

struct ScanContext {
    std::shared_ptr<UA_Client> client;
    std::shared_ptr<std::vector<types::Node>> channels;
};

static UA_StatusCode
node_iter(UA_NodeId child_id, UA_Boolean is_inverse, UA_NodeId _, void *raw_ctx) {
    if (is_inverse) return UA_STATUSCODE_GOOD;
    auto *ctx = static_cast<ScanContext *>(raw_ctx);
    auto *const ua_client = ctx->client.get();

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
    UA_StatusCode status = res.get().responseHeader.serviceResult;
    if (status != UA_STATUSCODE_GOOD) { return status; }
    if (!res.get().results[0].hasValue) { return res.get().results[0].status; }
    if (!res.get().results[1].hasValue) { return res.get().results[1].status; }
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
        types::NodeId::to_string(child_id),
        types::node_class_to_string(cls),
        is_array
    );
    return status;
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
    const auto scan_ctx = std::make_unique<ScanContext>(ScanContext{
        connection.shared(),
        std::make_shared<std::vector<types::Node>>(),
    });

    UA_Client_forEachChildNodeCall(
        scan_ctx->client.get(),
        args.node.get(),
        node_iter,
        scan_ctx.get()
    );

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
}
