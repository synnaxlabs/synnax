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

    UA_NodeClass nodeClass;
    UA_StatusCode retval = UA_Client_readNodeClassAttribute(
        ctx->client.get(),
        child_id,
        &nodeClass
    );
    if (retval != UA_STATUSCODE_GOOD) return retval;

    UA_QualifiedName browseName;
    retval = UA_Client_readBrowseNameAttribute(ua_client, child_id, &browseName);
    if (retval != UA_STATUSCODE_GOOD) return retval;
    auto name = std::string((char *) browseName.name.data,
                            browseName.name.length);
    LOG(INFO) << "Node id: " << nodeIdToString(child_id) << " Name: " << name <<
            std::endl;
    ctx->channels->emplace_back(
        synnax::FLOAT32,
        name,
        nodeIdToString(child_id),
        true
    );
    // if (nodeClass == UA_NODECLASS_VARIABLE && child_id.namespaceIndex != 0) {
    //     UA_Variant value;
    //     UA_Variant_init(&value);
    //     retval = UA_Client_readValueAttribute(ua_client, child_id, &value);
    //
    //     if (retval == UA_STATUSCODE_GOOD && value.type != nullptr) {
    //         auto [dt, is_array] = variant_data_type(value);
    //         LOG(INFO) << "Node id: " << node_id << " Name: " << name << " Is array: " <<
    //                 is_array << " Data type: " << dt.value << std::endl;
    //         if (dt != synnax::DATA_TYPE_UNKNOWN && !dt.is_variable())
    //
    //     }
    // }
    return UA_STATUSCODE_GOOD;
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
