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
#include "opc.h"
#include "nlohmann/json.hpp"
#include "client/cpp/synnax.h"
#include "driver/task/task.h"
#include "driver/config/config.h"

using json = nlohmann::json;

namespace opc {

    struct ScannnerScanCommandArgs {
        ConnectionConfig connection;

        explicit ScannnerScanCommandArgs(config::Parser parser): connection(
            ConnectionConfig(parser.child("connection"))) {
        }
    };

    const std::string SCAN_CMD_TYPE = "scan";
    const std::string TEST_CONNECTION_CMD_TYPE = "test_connection";

    class Scanner final : public task::Task {
    public:
        explicit Scanner( 
            std::shared_ptr<task::Context> ctx, 
            synnax::Task task) : ctx(ctx), task(task) {
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
        int max_depth = 10;

        void scan(const task::Command &cmd) const;

        void testConnection(const task::Command &cmd) const;
    };
}
