// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <map>

/// module
#include "x/cpp/xjson/xjson.h"
#include "x/cpp/loop/loop.h"

/// internal
#include "driver/opc/opc.h"
#include "driver/opc/util.h"
#include "driver/pipeline/control.h"
#include "driver/task/common/write_task.h"

/// external
#include "vendor/open62541/open62541/include/open62541/client.h"


namespace opc {
struct WriterChannelConfig {
    const bool enabled;
    const UA_NodeId node;
    /// @brief the corresponding channel key to write the variable for the node from.
    ChannelKey cmd_channel;

    explicit WriterChannelConfig(xjson::Parser &parser) :
        enabled(parser.optional<bool>("enabled", true)),
        node(parse_node_id("node_id", parser)),
        cmd_channel(parser.required<ChannelKey>("channel")) {
    }
};

struct WriterConfig {
    /// @brief the device representing the OPC UA server to read from.
    std::string device;
    /// @brief the list of channels to read from the server.
    std::unordered_map<synnax::ChannelKey, WriterChannelConfig> channels;
    /// @brief frequency state of a controlled channel is published
    telem::Rate state_rate = telem::Rate(1); // default to 1 Hz
    /// @brief index key for all state channels in this task
    synnax::ChannelKey state_index_key;

    WriterConfig() = default;

    explicit WriterConfig(xjson::Parser &parser);

    [[nodiscard]] std::vector<ChannelKey> cmd_keys() const {
        std::vector<ChannelKey> keys;
        keys.reserve(this->channels.size());
        for (const auto &[key, _]: channels) keys.push_back(key);
        return keys;
    }
};

class WriterSink final : public common::Sink {
    WriterConfig cfg;
    std::shared_ptr<UA_Client> client;

public:
    WriterSink(
        WriterConfig cfg,
        const std::shared_ptr<UA_Client> &ua_client
    ): Sink(cfg.cmd_keys()),
       cfg(std::move(cfg)), client(ua_client) {
    }

    xerrors::Error write(const synnax::Frame &frame) override {
        const auto client = this->client.get();
        
        UA_WriteRequest request;
        UA_WriteRequest_init(&request);
        
        request.nodesToWrite = static_cast<UA_WriteValue *>(UA_Array_new(
            frame.size(),
            &UA_TYPES[UA_TYPES_WRITEVALUE]
        ));
        request.nodesToWriteSize = 0;

        // Prepare all write requests
        for (const auto &[key, s]: frame) {
            auto it = this->cfg.channels.find(key);
            if (it == this->cfg.channels.end()) continue;
            const auto ch = it->second;
            const auto [val, err] = series_to_variant(s);
            if (err != xerrors::NIL) continue;
            UA_WriteValue *node = &request.nodesToWrite[request.nodesToWriteSize];
            node->attributeId = UA_ATTRIBUTEID_VALUE;
            node->nodeId = ch.node;
            node->value.hasValue = true;
            node->value.value = *val;
            request.nodesToWriteSize++;
        }

        // Perform batch write if we have valid requests
        if (request.nodesToWriteSize > 0) {
            UA_WriteResponse res = UA_Client_Service_write(client, request);
                UA_Array_delete(request.nodesToWrite, request.nodesToWriteSize,
                    &UA_TYPES[UA_TYPES_WRITEVALUE]);

            UA_Array_delete(request.nodesToWrite, request.nodesToWriteSize,
                &UA_TYPES[UA_TYPES_WRITEVALUE]);
        }

        return xerrors::NIL;
    }
};
}
