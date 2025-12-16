// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <utility>

#include "glog/logging.h"
#include "nlohmann/json.hpp"
#include "open62541/client.h"
#include "open62541/client_highlevel.h"
#include "open62541/types.h"

#include "x/cpp/defer/defer.h"
#include "x/cpp/status/status.h"
#include "x/cpp/xjson/xjson.h"

#include "driver/opc/device/device.h"
#include "driver/opc/scan_task.h"
#include "driver/opc/telem/telem.h"
#include "driver/opc/types/types.h"
#include "driver/task/common/status.h"

namespace opc {
Scanner::Scanner(
    std::shared_ptr<task::Context> ctx,
    synnax::Task task,
    std::shared_ptr<connection::Pool> conn_pool
):
    ctx(std::move(ctx)), task(std::move(task)), conn_pool(std::move(conn_pool)) {}

common::ScannerConfig Scanner::config() const {
    return common::ScannerConfig{
        .make = INTEGRATION_NAME,
        .log_prefix = SCAN_LOG_PREFIX
    };
}

std::pair<std::vector<synnax::Device>, xerrors::Error>
Scanner::scan(const common::ScannerContext &scan_ctx) {
    std::vector<synnax::Device> devices;
    if (scan_ctx.devices == nullptr) return {devices, xerrors::NIL};
    for (auto [key, dev]: *scan_ctx.devices) {
        if (const auto err = this->check_device_health(dev); err)
            LOG(WARNING) << SCAN_LOG_PREFIX << "health check failed for " << dev.name
                         << ": " << err;
        devices.push_back(dev);
    }
    return {devices, xerrors::NIL};
}

bool Scanner::exec(
    task::Command &cmd,
    const synnax::Task &,
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
    return false; // Not handled
}

xerrors::Error Scanner::check_device_health(synnax::Device &dev) {
    const auto rack_key = synnax::rack_key_from_task_key(this->task.key);
    const auto parser = xjson::Parser(dev.properties);
    const auto props = device::Properties(parser);
    if (parser.error()) {
        dev.status = synnax::DeviceStatus{
            .key = dev.status_key(),
            .name = dev.name,
            .variant = status::variant::WARNING,
            .message = "Invalid device properties",
            .description = parser.error().message(),
            .time = ::telem::TimeStamp::now(),
            .details = {.rack = rack_key, .device = dev.key},
        };
        return parser.error();
    }

    auto [conn, conn_err] = this->conn_pool->acquire(props.connection, SCAN_LOG_PREFIX);
    if (conn_err)
        dev.status = synnax::DeviceStatus{
            .key = dev.status_key(),
            .name = dev.name,
            .variant = status::variant::WARNING,
            .message = "Failed to reach server",
            .description = conn_err.message(),
            .time = ::telem::TimeStamp::now(),
            .details = {.rack = rack_key, .device = dev.key},
        };
    else
        dev.status = synnax::DeviceStatus{
            .key = dev.status_key(),
            .name = dev.name,
            .variant = status::variant::SUCCESS,
            .message = "Server connected",
            .time = ::telem::TimeStamp::now(),
            .details = {.rack = rack_key, .device = dev.key},
        };
    return xerrors::NIL;
}

struct ScanContext {
    std::shared_ptr<UA_Client> client;
    std::shared_ptr<std::vector<Node>> channels;
};

static UA_StatusCode node_iter(
    UA_NodeId child_id,
    UA_Boolean is_inverse,
    [[maybe_unused]] UA_NodeId ref_type_id,
    void *raw_ctx
) {
    if (is_inverse) return UA_STATUSCODE_GOOD;
    auto ctx = static_cast<ScanContext *>(raw_ctx);
    const auto ua_client = ctx->client.get();

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

    ReadResponse res(UA_Client_Service_read(ua_client, req));
    UA_StatusCode status = res.get().responseHeader.serviceResult;
    if (status != UA_STATUSCODE_GOOD) { return status; }
    if (!res.get().results[0].hasValue) { return res.get().results[0].status; }
    if (!res.get().results[1].hasValue) { return res.get().results[1].status; }
    UA_NodeClass cls = *static_cast<UA_NodeClass *>(res.get().results[0].value.data);
    auto [ns_index, b_name] = *static_cast<UA_QualifiedName *>(
        res.get().results[1].value.data
    );
    const auto name = std::string(reinterpret_cast<char *>(b_name.data), b_name.length);
    auto data_type = ::telem::UNKNOWN_T;
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
        NodeId::to_string(child_id),
        node_class_to_string(cls),
        is_array
    );
    return status;
}

void Scanner::browse_nodes(const task::Command &cmd) const {
    xjson::Parser parser(cmd.args);
    const ScanCommandArgs args(parser);
    synnax::TaskStatus status{
        .key = this->task.status_key(),
        .name = this->task.name,
        .variant = status::variant::ERR,
        .details = synnax::TaskStatusDetails{.task = task.key, .cmd = cmd.key}
    };
    if (!parser.ok()) {
        status.message = "Failed to parse scan command";
        status.details.data = parser.error_json();
        return ctx->set_status(status);
    }

    auto [connection, err] = conn_pool->acquire(args.connection, SCAN_LOG_PREFIX);
    if (err) {
        status.variant = status::variant::ERR;
        status.message = err.message();
        return ctx->set_status(status);
    }
    const auto scan_ctx = std::make_unique<ScanContext>(ScanContext{
        connection.shared(),
        std::make_shared<std::vector<Node>>(),
    });

    UA_Client_forEachChildNodeCall(
        scan_ctx->client.get(),
        args.node.get(),
        node_iter,
        scan_ctx.get()
    );

    status.message = "Scan successful";
    status.variant = status::variant::SUCCESS;
    status.details.data = device::Properties(args.connection, *scan_ctx->channels)
                              .to_json();
    ctx->set_status(status);
}

void Scanner::test_connection(const task::Command &cmd) const {
    xjson::Parser parser(cmd.args);
    const ScanCommandArgs args(parser);
    synnax::TaskStatus status{
        .key = this->task.status_key(),
        .name = this->task.name,
        .variant = status::variant::ERR,
        .details = synnax::TaskStatusDetails{
            .task = task.key,
            .cmd = cmd.key,
            .running = true,
        }
    };
    if (!parser.ok()) {
        status.message = "Failed to parse test command";
        status.details.data = parser.error_json();
        return ctx->set_status(status);
    }
    auto [client, err] = connect(args.connection, SCAN_LOG_PREFIX);
    if (err) {
        status.message = err.data;
        return ctx->set_status(status);
    }
    status.variant = status::variant::SUCCESS;
    status.message = "Connection successful";
    return ctx->set_status(status);
}
}
