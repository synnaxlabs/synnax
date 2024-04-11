// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <utility>
#include "nlohmann/json.hpp"

#include "scanner.h"
#include "glog/logging.h"
#include "driver/driver/config/config.h"
#include "include/open62541/statuscodes.h"
#include "include/open62541/types.h"
#include "include/open62541/client_config_default.h"
#include "include/open62541/client_highlevel.h"
#include "include/open62541/client.h"
#include "driver/driver/opcua/util.h"

using namespace opcua;

Scanner::Scanner(
    std::shared_ptr<task::Context> ctx,
    synnax::Task task
): ctx(std::move(ctx)), task(std::move(task)) {
}

void Scanner::exec(task::Command &cmd) {
    if (cmd.type == SCAN_CMD_TYPE) return scan(cmd);
    if (cmd.type == TEST_CONNECTION_CMD_TYPE) return testConnection(cmd);
    LOG(ERROR) << "[OPCUA] Scanner received unknown command type: " << cmd.type;
}


// Forward declaration of the callback function for recursive calls
static UA_StatusCode nodeIter(UA_NodeId childId, UA_Boolean isInverse,
                              UA_NodeId referenceTypeId, void *handle);

const int MAX_DEPTH = 2;

struct DeviceNode {
    std::string name;
    std::uint32_t node_id;
    synnax::DataType data_type;

    json toJSON() const {
        return {
            {"name", name},
            {"node_id", node_id},
            {"data_type", data_type.name()}
        };
    }
};

struct ScanContext {
    std::shared_ptr<UA_Client> client;
    UA_UInt32 depth;
    std::shared_ptr<std::vector<DeviceNode> > channels;
};

// Function to recursively iterate through all children
void iterateChildren(ScanContext *ctx, UA_NodeId nodeId) {
    UA_Client_forEachChildNodeCall(ctx->client.get(), nodeId, nodeIter, ctx);
}

// Callback function to handle each child node
static UA_StatusCode nodeIter(
    UA_NodeId childId,
    UA_Boolean isInverse,
    UA_NodeId referenceTypeId,
    void *handle
) {
    if (isInverse) return UA_STATUSCODE_GOOD;
    auto *ctx = static_cast<ScanContext *>(handle);
    const auto ua_client = ctx->client.get();

    UA_NodeClass nodeClass;
    UA_StatusCode retval = UA_Client_readNodeClassAttribute(
        ctx->client.get(),
        childId,
        &nodeClass
    );
    if (retval != UA_STATUSCODE_GOOD) return retval;

    if (nodeClass == UA_NODECLASS_VARIABLE && childId.namespaceIndex != 0) {
        UA_QualifiedName browseName;
        retval = UA_Client_readBrowseNameAttribute(ua_client, childId, &browseName);
        if (retval != UA_STATUSCODE_GOOD) return retval;
        UA_Variant value;
        UA_Variant_init(&value);
        retval = UA_Client_readValueAttribute(ua_client, childId, &value);
        if (retval == UA_STATUSCODE_GOOD && value.type != nullptr)
            ctx->channels->push_back({
                .name = std::string((char *) browseName.name.data,
                                    browseName.name.length),
                .node_id = childId.identifier.numeric,
                .data_type = variant_data_type(value)
            });
    }

    if (ctx->depth >= MAX_DEPTH) return UA_STATUSCODE_GOOD;
    ctx->depth++;
    iterateChildren(ctx, childId);
    ctx->depth--;
    return UA_STATUSCODE_GOOD;
}

void Scanner::scan(const task::Command &cmd) const {
    config::Parser parser(cmd.args);
    ScannnerScanCommandArgs args(parser);
    if (!parser.ok())
        return ctx->setState({
            .task = task.key,
            .key = cmd.key,
            .details = parser.error_json()
        });

    auto [ua_client, err] = connect(args.connection);
    if (err) {
        parser.field_err("", "failed to connect");
        return ctx->setState({
            .task = task.key,
            .key = cmd.key,
            .variant = "error",
            .details = parser.error_json()
        });
    }

    UA_NodeId rootFolderId = UA_NODEID_NUMERIC(0, UA_NS0ID_OBJECTSFOLDER);
    auto scan_ctx = new ScanContext{
        ua_client,
        0,
        std::make_shared<std::vector<DeviceNode>>()
    };
    iterateChildren(scan_ctx, rootFolderId);

    json scan_result = json::array();
    for (const auto &channel : *scan_ctx->channels)
        scan_result.push_back(channel.toJSON());

    delete scan_ctx;

    ctx->setState({
        .task = task.key,
        .variant = "success",
        .key = cmd.key,
        .details = scan_result,
    });
    return;
}

void Scanner::testConnection(const task::Command &cmd) const {
    config::Parser parser(cmd.args);
    ScannerTestConnectionCommandArgs args(parser);
    if (!parser.ok())
        return ctx->setState({
            .task = task.key,
            .key = cmd.key,
            .details = parser.error_json()
        });
    const auto err = connect(args.connection).second;
    if (err)
        return ctx->setState({
            .task = task.key,
            .key = cmd.key,
            .variant = "error",
            .details = {
                {"message", err.data}
            }
        });
    return ctx->setState({
        .task = task.key,
        .key = cmd.key,
        .variant = "success",
        .details = {
            {"message", "Connection successful"}
        },
    });
}
