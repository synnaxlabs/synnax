// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <memory>
#include <utility>
#include "opc.h"
#include "nlohmann/json.hpp"
#include "client/cpp/synnax.h"
#include "driver/task/task.h"
#include "driver/config/config.h"
#include "include/open62541/types.h"
#include "driver/opc/util.h"

using json = nlohmann::json;

namespace opc {
    struct ScannerScanCommandArgs {
        ConnectionConfig connection;
        std::string node_id;
        UA_NodeId node{};

        explicit ScannerScanCommandArgs(config::Parser parser) : connection(
                ConnectionConfig(parser.child("connection"))),
                                                                 node_id(parser.optional<std::string>("node_id", "")) {
            if (node_id.empty()) node = UA_NODEID_NUMERIC(0, UA_NS0ID_OBJECTSFOLDER);
            else node = parseNodeId("node_id", parser);
        }
    };

    const std::string SCAN_CMD_TYPE = "scan";
    const std::string TEST_CONNECTION_CMD_TYPE = "test_connection";

    class Scanner final : public task::Task {
    public:
        explicit Scanner(
                std::shared_ptr<task::Context> ctx,
                synnax::Task task) : ctx(std::move(ctx)), task(std::move(task)) {
        }

        static std::unique_ptr<task::Task> configure(
                const std::shared_ptr<task::Context> &ctx,
                const synnax::Task &task
        );

        std::string name() override { return task.name; }

        void exec(task::Command &cmd) override;

        void stop() override {
        }

    private:
        std::shared_ptr<task::Context> ctx;
        const synnax::Task task;

        void scan(const task::Command &cmd) const;

        void testConnection(const task::Command &cmd) const;
    };
}
