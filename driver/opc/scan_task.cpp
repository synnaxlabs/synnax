// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std
#include <memory>
#include <utility>

/// external
#include "glog/logging.h"
#include "nlohmann/json.hpp"
#include "open62541/client.h"
#include "open62541/client_highlevel.h"
#include "open62541/types.h"

/// module
#include "x/cpp/xjson/xjson.h"

/// internal
#include "driver/opc/scan_task.h"
#include "driver/opc/util/util.h"
#include "x/cpp/defer/defer.h"

namespace opc {
std::unique_ptr<task::Task> ScanTask::configure(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    return std::make_unique<ScanTask>(ctx, task);
}

void ScanTask::exec(task::Command &cmd) {
    if (cmd.type == SCAN_CMD_TYPE) return scan(cmd);
    if (cmd.type == TEST_CONNECTION_CMD_TYPE) return test_connection(cmd);
    LOG(ERROR) << "[opc] Scanner received unknown command type: " << cmd.type;
}

struct ScanContext {
    std::shared_ptr<UA_Client> client;
    std::shared_ptr<std::vector<util::NodeProperties>> channels;
};

static UA_StatusCode
node_iter(UA_NodeId child_id, UA_Boolean is_inverse, UA_NodeId _, void *raw_ctx) {
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

    UA_ReadResponse res = UA_Client_Service_read(ua_client, req);
    x::defer clear([&res, &req] {
        // UA_ReadRequest_clear(&req);
        // UA_ReadResponse_clear(&res);
    });
    UA_StatusCode status = res.responseHeader.serviceResult;
    if (status != UA_STATUSCODE_GOOD) return status;
    if (!res.results[0].hasValue) return res.results[0].status;
    if (!res.results[1].hasValue) return res.results[1].status;
    UA_NodeClass cls = *static_cast<UA_NodeClass *>(res.results[0].value.data);
    auto [ns_index, b_name] = *static_cast<UA_QualifiedName *>(res.results[1].value.data
    );
    const auto name = std::string(reinterpret_cast<char *>(b_name.data), b_name.length);
    auto data_type = telem::UNKNOWN_T;
    bool is_array = false;
    if (cls == UA_NODECLASS_VARIABLE && res.results[2].hasValue) {
        auto value = res.results[2].value;
        data_type = util::ua_to_data_type(value.type);
        is_array = !UA_Variant_isScalar(&value);
        UA_Variant_clear(&value);
    } else if (cls == UA_NODECLASS_VARIABLE)
        LOG(ERROR) << "[opc.scanner] No value for " << name;
    ctx->channels->emplace_back(
        data_type,
        name,
        util::node_id_to_string(child_id),
        util::node_class_to_string(cls),
        is_array
    );
    return status;
}

void ScanTask::scan(const task::Command &cmd) const {
    xjson::Parser parser(cmd.args);
    const ScanCommandArgs args(parser);
    if (!parser.ok())
        return ctx->set_state(
            {.task = task.key, .key = cmd.key, .details = parser.error_json()}
        );

    auto [ua_client, err] = connect(args.connection, "[opc.scanner] ");
    if (err)
        return ctx->set_state(
            {.task = task.key,
             .key = cmd.key,
             .variant = "error",
             .details = {{"message", err.message()}}}
        );

    const auto scan_ctx = new ScanContext{
        ua_client,
        std::make_shared<std::vector<util::NodeProperties>>(),
    };
    UA_Client_forEachChildNodeCall(
        scan_ctx->client.get(),
        args.node,
        node_iter,
        scan_ctx
    );
    ctx->set_state({
        .task = task.key,
        .key = cmd.key,
        .variant = "success",
        .details = util::DeviceProperties(args.connection, *scan_ctx->channels)
                       .to_json(),
    });
    delete scan_ctx;
}

void ScanTask::test_connection(const task::Command &cmd) const {
    xjson::Parser parser(cmd.args);
    const ScanCommandArgs args(parser);
    if (!parser.ok())
        return ctx->set_state(
            {.task = task.key, .key = cmd.key, .details = parser.error_json()}
        );
    if (const auto err = connect(args.connection, "[opc.scanner] ").second)
        return ctx->set_state(
            {.task = task.key,
             .key = cmd.key,
             .variant = "error",
             .details = {{"message", err.data}}}
        );
    return ctx->set_state({
        .task = task.key,
        .key = cmd.key,
        .variant = "success",
        .details = {{"message", "Connection successful"}},
    });
}
}
