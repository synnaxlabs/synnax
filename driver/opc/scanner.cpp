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
#include "driver/config/config.h"
#include "include/open62541/statuscodes.h"
#include "include/open62541/types.h"
#include "include/open62541/client_config_default.h"
#include "include/open62541/client_highlevel.h"
#include "include/open62541/client.h"
#include "driver/opc/util.h"

using namespace opc;

std::unique_ptr<task::Task> Scanner::configure(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    return std::make_unique<Scanner>(ctx, task);
}

void Scanner::exec(task::Command &cmd) {
    if (cmd.type == SCAN_CMD_TYPE) return scan(cmd);
    if (cmd.type == TEST_CONNECTION_CMD_TYPE) return testConnection(cmd);
    LOG(ERROR) << "[opc] Scanner received unknown command type: " << cmd.type;
}


// Forward declaration of the callback function for recursive calls
static UA_StatusCode nodeIter(UA_NodeId child_id, UA_Boolean is_inverse,
                              UA_NodeId reference_type_id, void *handle);


struct ScanContext {
    std::shared_ptr<UA_Client> client;
    std::shared_ptr<std::vector<DeviceNodeProperties> > channels;
};

// Function to recursively iterate through all children
void iterateChildren(ScanContext *ctx, UA_NodeId node_id) {
    UA_Client_forEachChildNodeCall(ctx->client.get(), node_id, nodeIter, ctx);
}


std::string nodeClassToString(UA_NodeClass nodeClass) {
    switch (nodeClass) {
        case UA_NODECLASS_OBJECT: return "Object";
        case UA_NODECLASS_VARIABLE: return "Variable";
        case UA_NODECLASS_METHOD: return "Method";
        case UA_NODECLASS_OBJECTTYPE: return "ObjectType";
        case UA_NODECLASS_VARIABLETYPE: return "VariableType";
        case UA_NODECLASS_DATATYPE: return "DataType";
        case UA_NODECLASS_REFERENCETYPE: return "ReferenceType";
        case UA_NODECLASS_VIEW: return "View";
        default: return "Unknown";
    }
}


// Callback function to handle each child node
static UA_StatusCode nodeIter(
    UA_NodeId child_id,
    UA_Boolean is_inverse,
    UA_NodeId reference_type_id,
    void *handle
) {
    if (is_inverse) return UA_STATUSCODE_GOOD;
    auto *ctx = static_cast<ScanContext *>(handle);
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

    UA_ReadRequest request;
    UA_ReadRequest_init(&request);
    request.nodesToRead = ids;
    request.nodesToReadSize = 3;

    UA_ReadResponse response = UA_Client_Service_read(ua_client, request);
    UA_StatusCode status = response.responseHeader.serviceResult;

    if (status == UA_STATUSCODE_GOOD) {
        if (!response.results[0].hasValue) return response.results[0].status;
        if (!response.results[1].hasValue) return response.results[1].status;
        UA_NodeClass nodeClass = *static_cast<UA_NodeClass *>(
            response.results[0].value.data
        );
        UA_QualifiedName browseName = *static_cast<UA_QualifiedName *>(
            response.results[1].value.data
        );
        auto name = std::string(
            reinterpret_cast<char *>(browseName.name.data),
            browseName.name.length
        );
        auto data_type = synnax::DATA_TYPE_UNKNOWN;
        bool is_array = false;
        if (nodeClass == UA_NODECLASS_VARIABLE && response.results[2].hasValue) {
            UA_Variant value;
            UA_Variant_init(&value);
            UA_Variant_copy(&response.results[2].value, &value);
            auto [dt, is_arr] = variant_data_type(value);
            data_type = dt;
            is_array = is_arr;
            UA_Variant_clear(&value);
        }
        ctx->channels->emplace_back(
            data_type,
            name,
            nodeIdToString(child_id),
            nodeClassToString(nodeClass),
            is_array
        );
    }

    UA_ReadResponse_clear(&response);
    return status;
}

void Scanner::scan(const task::Command &cmd) const {
    config::Parser parser(cmd.args);
    ScannerScanCommandArgs args(parser);
    if (!parser.ok())
        return ctx->setState({
            .task = task.key,
            .key = cmd.key,
            .details = parser.error_json()
        });

    auto [ua_client, err] = connect(args.connection, "[opc.scanner] ");
    if (err)
        return ctx->setState({
            .task = task.key,
            .key = cmd.key,
            .variant = "error",
            .details = {{"message", err.message()}}
        });

    const auto scan_ctx = new ScanContext{
        ua_client,
        std::make_shared<std::vector<DeviceNodeProperties> >(),
    };
    iterateChildren(scan_ctx, args.node);
    ctx->setState({
        .task = task.key,
        .key = cmd.key,
        .variant = "success",
        .details = DeviceProperties(args.connection, *scan_ctx->channels).toJSON(),
    });
    delete scan_ctx;
}

void Scanner::testConnection(const task::Command &cmd) const {
    config::Parser parser(cmd.args);
    ScannerScanCommandArgs args(parser);
    if (!parser.ok())
        return ctx->setState({
            .task = task.key,
            .key = cmd.key,
            .details = parser.error_json()
        });
    const auto err = connect(args.connection, "[opc.scanner] ").second;
    if (err)
        return ctx->setState({
            .task = task.key,
            .key = cmd.key,
            .variant = "error",
            .details = {{"message", err.data}}
        });
    return ctx->setState({
        .task = task.key,
        .key = cmd.key,
        .variant = "success",
        .details = {{"message", "Connection successful"}},
    });
}