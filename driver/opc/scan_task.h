// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "nlohmann/json.hpp"
#include "open62541/types.h"

#include "client/cpp/synnax.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/xjson/xjson.h"

#include "driver/opc/conn/conn.h"
#include "driver/opc/types/types.h"
#include "driver/task/task.h"
#include "opc.h"

using json = nlohmann::json;

namespace opc {
struct NodeProperties {
    ::telem::DataType data_type;
    std::string node_class;
    std::string name;
    std::string node_id;
    bool is_array;

    NodeProperties(
        const ::telem::DataType &data_type,
        const std::string &name,
        const std::string &node_id,
        const std::string &node_class,
        const bool is_array
    ):
        data_type(data_type),
        node_class(node_class),
        name(name),
        node_id(node_id),
        is_array(is_array) {}

    explicit NodeProperties(xjson::Parser &p):
        data_type(::telem::DataType(p.required<std::string>("data_type"))),
        name(p.required<std::string>("name")),
        node_id(p.required<std::string>("node_id")),
        is_array(p.optional<bool>("is_array", false)) {}

    json to_json() const {
        return {
            {"data_type", data_type.name()},
            {"name", name},
            {"node_id", node_id},
            {"node_class", node_class},
            {"is_array", is_array}
        };
    }
};

struct DeviceProperties {
    opc::conn::Config connection;
    std::vector<NodeProperties> channels;

    DeviceProperties(
        const opc::conn::Config &connection,
        const std::vector<NodeProperties> &channels
    ):
        connection(connection), channels(channels) {}

    explicit DeviceProperties(const xjson::Parser &parser):
        connection(parser.child("connection")) {
        parser.iter("channels", [&](xjson::Parser &cb) { channels.emplace_back(cb); });
    }

    json to_json() const {
        json j;
        j["connection"] = connection.to_json();
        j["channels"] = json::array();
        for (const auto &ch: channels)
            j["channels"].push_back(ch.to_json());
        return j;
    }
};
///@brief The parameters for connecting to and iterating through nodes in the OPC UA
/// server.A
struct ScanCommandArgs {
    opc::conn::Config connection;
    std::string node_id;
    opc::NodeId node;

    explicit ScanCommandArgs(xjson::Parser &parser):
        connection(opc::conn::Config(parser.child("connection"))),
        node_id(parser.optional<std::string>("node_id", "")) {
        if (node_id.empty())
            node = opc::NodeId(UA_NODEID_NUMERIC(0, UA_NS0ID_OBJECTSFOLDER));
        else
            node = opc::NodeId::parse("node_id", parser);
    }
};

const std::string SCAN_CMD_TYPE = "scan";
const std::string TEST_CONNECTION_CMD_TYPE = "test_connection";

class ScanTask final : public task::Task {
public:
    explicit ScanTask(
        std::shared_ptr<task::Context> ctx,
        synnax::Task task,
        std::shared_ptr<opc::conn::Pool> conn_pool
    ):
        ctx(std::move(ctx)), task(std::move(task)), conn_pool_(std::move(conn_pool)) {}

    std::string name() const override { return task.name; }

    void exec(task::Command &cmd) override;

    void stop(bool will_reconfigure) override {}

private:
    std::shared_ptr<task::Context> ctx;
    const synnax::Task task;
    std::shared_ptr<opc::conn::Pool> conn_pool_;

    void scan(const task::Command &cmd) const;

    void test_connection(const task::Command &cmd) const;
};
}
