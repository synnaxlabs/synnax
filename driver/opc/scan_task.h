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

#include "driver/opc/connection/connection.h"
#include "driver/opc/types/types.h"
#include "driver/task/task.h"
#include "opc.h"

using json = nlohmann::json;

namespace opc {
///@brief The parameters for connecting to and iterating through nodes in the OPC UA
/// server.A
struct ScanCommandArgs {
    opc::connection::Config connection;
    std::string node_id;
    opc::NodeId node;

    explicit ScanCommandArgs(xjson::Parser &parser):
        connection(opc::connection::Config(parser.child("connection"))),
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
        std::shared_ptr<opc::connection::Pool> conn_pool
    ):
        ctx(std::move(ctx)), task(std::move(task)), conn_pool_(std::move(conn_pool)) {}

    std::string name() const override { return task.name; }

    void exec(task::Command &cmd) override;

    void stop(bool will_reconfigure) override {}

private:
    std::shared_ptr<task::Context> ctx;
    const synnax::Task task;
    std::shared_ptr<opc::connection::Pool> conn_pool_;

    void scan(const task::Command &cmd) const;

    void test_connection(const task::Command &cmd) const;
};
}
