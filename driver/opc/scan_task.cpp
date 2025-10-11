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
#include "x/cpp/defer/defer.h"
#include "x/cpp/status/status.h"
#include "x/cpp/xjson/xjson.h"

/// internal
#include "driver/opc/scan_task.h"
#include "driver/opc/util/util.h"

namespace opc {
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

    opc::ReadResponse res(UA_Client_Service_read(ua_client, req));
    UA_StatusCode status = res.get().responseHeader.serviceResult;
    if (status != UA_STATUSCODE_GOOD) {
        return status;
    }
    if (!res.get().results[0].hasValue) {
        return res.get().results[0].status;
    }
    if (!res.get().results[1].hasValue) {
        return res.get().results[1].status;
    }
    UA_NodeClass cls = *static_cast<UA_NodeClass *>(res.get().results[0].value.data);
    auto [ns_index, b_name] = *static_cast<UA_QualifiedName *>(
        res.get().results[1].value.data
    );
    const auto name = std::string(reinterpret_cast<char *>(b_name.data), b_name.length);
    auto data_type = telem::UNKNOWN_T;
    bool is_array = false;
    if (cls == UA_NODECLASS_VARIABLE && res.get().results[2].hasValue) {
        const auto &value = res.get().results[2].value;
        data_type = util::ua_to_data_type(value.type);
        is_array = !UA_Variant_isScalar(&value);
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
        return ctx->set_status(
            {.key = cmd.key,
             .variant = status::variant::ERR,
             .details = synnax::TaskStatusDetails{
                 .task = task.key,
                 .data = parser.error_json()
             }}
        );

    auto [conn, err] = conn_pool_->acquire(args.connection, "[opc.scanner] ");
    if (err)
        return ctx->set_status({
            .key = cmd.key,
            .variant = status::variant::ERR,
            .message = err.message(),
            .details = synnax::TaskStatusDetails{
                .task = task.key,
            },
        });

    auto scan_ctx = std::make_unique<ScanContext>(ScanContext{
        conn.shared(),
        std::make_shared<std::vector<util::NodeProperties>>(),
    });

    UA_Client_forEachChildNodeCall(
        scan_ctx->client.get(),
        args.node.get(),
        node_iter,
        scan_ctx.get()
    );

    ctx->set_status({
        .key = cmd.key,
        .variant = status::variant::SUCCESS,
        .details = synnax::TaskStatusDetails{
            .task = task.key,
            .data = util::DeviceProperties(args.connection, *scan_ctx->channels)
                        .to_json(),
        },
    });
}

void ScanTask::test_connection(const task::Command &cmd) const {
    xjson::Parser parser(cmd.args);
    const ScanCommandArgs args(parser);
    if (!parser.ok())
        return ctx->set_status(
            {.key = cmd.key,
             .variant = status::variant::ERR,
             .details = synnax::TaskStatusDetails{
                 .task = task.key,
                 .data = parser.error_json()
             }}
        );
    auto [client, err] = connect(args.connection, "[opc.scanner] ");
    if (err)
        return ctx->set_status(
            {.key = cmd.key,
             .variant = status::variant::ERR,
             .message = err.data,
             .details = synnax::TaskStatusDetails{.task = task.key, .running = true}}
        );
    return ctx->set_status(
        {.key = cmd.key,
         .variant = status::variant::SUCCESS,
         .message = "Connection successful",
         .details = synnax::TaskStatusDetails{.task = task.key, .running = true}}
    );
}
}
