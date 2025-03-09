// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// module
#include "x/cpp/xjson/xjson.h"

/// internal
#include "driver/opc/opc.h"
#include "driver/opc/util/util.h"
#include "driver/pipeline/control.h"
#include "driver/task/common/write_task.h"

/// external
#include "open62541/client.h"


namespace opc {
struct OutputChan {
    /// @brief whether output for the channel is enabled.
    const bool enabled;
    /// @brief the OPC UA node id.
    const UA_NodeId node;
    /// @brief the corresponding channel key to write the variable for the node from.
    const synnax::ChannelKey cmd_channel;

    explicit OutputChan(xjson::Parser &parser) :
        enabled(parser.optional<bool>("enabled", true)),
        node(util::parse_node_id("node_id", parser)),
        cmd_channel(parser.required<synnax::ChannelKey>("channel")) {
    }
};

struct WriteTaskConfig {
    /// @brief the device representing the OPC UA server to read from.
    std::string device;
    /// @brief the list of channels to read from the server.
    std::unordered_map<synnax::ChannelKey, std::unique_ptr<OutputChan>> channels;
    /// @brief the config for connecting to the OPC UA server.
    util::ConnectionConfig conn;

    explicit WriteTaskConfig(
        const std::shared_ptr<synnax::Synnax> &client,
        xjson::Parser &parser
    ): device(parser.required<std::string>("device")) {
        parser.iter("channels", [&](xjson::Parser &channel_builder) {
            auto ch = std::make_unique<OutputChan>(channel_builder);
            if (ch->enabled) channels[ch->cmd_channel] = std::move(ch);
        });
        auto [dev, err] = client->hardware.retrieve_device(this->device);
        if (err) {
            parser.field_err("device", "failed to retrieve device: " + err.message());
            return;
        }
        const auto properties = xjson::Parser(dev.properties);
        this->conn = util::ConnectionConfig(properties.child("connection"));
        if (properties.error())
            parser.field_err("device", properties.error().message());
    }

    [[nodiscard]] std::vector<ChannelKey> cmd_keys() const {
        std::vector<ChannelKey> keys;
        keys.reserve(this->channels.size());
        for (const auto &[key, _]: channels) keys.push_back(key);
        return keys;
    }

    static std::pair<WriteTaskConfig, xerrors::Error> parse(
        const std::shared_ptr<synnax::Synnax> &client,
        const synnax::Task &task
    ) {
        auto parser = xjson::Parser(task.config);
        return {WriteTaskConfig(client, parser), parser.error()};
    }
};

class WriteTaskSink final : public common::Sink {
    const WriteTaskConfig cfg;
    const std::shared_ptr<UA_Client> client;
public:
    WriteTaskSink(
        WriteTaskConfig cfg,
        const std::shared_ptr<UA_Client> &client
    ): Sink(cfg.cmd_keys()),
       cfg(std::move(cfg)),
       client(client) {
    }

    xerrors::Error write(const synnax::Frame &frame) override {
        UA_WriteRequest req;
        UA_WriteRequest_init(&req);
        req.nodesToWrite = static_cast<UA_WriteValue *>(UA_Array_new(
            frame.size(),
            &UA_TYPES[UA_TYPES_WRITEVALUE]
        ));
        req.nodesToWriteSize = 0;
        x::defer clear_req([&req] {
            UA_Array_delete(
                req.nodesToWrite,
                req.nodesToWriteSize,
                &UA_TYPES[UA_TYPES_WRITEVALUE]
            );
            UA_WriteRequest_clear(&req);
        });
        for (const auto &[key, s]: frame) {
            auto it = this->cfg.channels.find(key);
            if (it == this->cfg.channels.end()) continue;
            const auto &ch = it->second;
            const auto [val, err] = util::series_to_variant(s);
            if (err != xerrors::NIL) continue;
            UA_WriteValue &node = req.nodesToWrite[req.nodesToWriteSize];
            node.attributeId = UA_ATTRIBUTEID_VALUE;
            node.nodeId = ch->node;
            node.value.hasValue = true;
            node.value.value = val;
            req.nodesToWriteSize++;
        }
        if (req.nodesToWriteSize == 0) return xerrors::NIL;
        UA_WriteResponse res = UA_Client_Service_write(this->client.get(), req);
        auto err = util::parse_error(res.responseHeader.serviceResult);
        UA_WriteResponse_clear(&res);
        return err;
    }
};
}
