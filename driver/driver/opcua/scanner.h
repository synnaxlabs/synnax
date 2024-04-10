#pragma once

#include <string>
#include <memory>
#include "opcua.h"
#include "nlohmann/json.hpp"
#include "client/cpp/synnax/synnax.h"
#include "driver/driver/task/task.h"
#include "driver/driver/config/config.h"

using json = nlohmann::json;

namespace opcua {
struct ScannnerScanCommandArgs {
    ConnectionConfig connection;

    explicit ScannnerScanCommandArgs(config::Parser parser): connection(
        ConnectionConfig(parser.child("connection"))) {
    }
};

struct ScannerTestConnectionCommandArgs {
    ConnectionConfig connection;

    explicit ScannerTestConnectionCommandArgs(config::Parser parser): connection(
        ConnectionConfig(parser.child("connection"))) {
    }
};

const std::string SCAN_CMD_TYPE = "scan";
const std::string TEST_CONNECTION_CMD_TYPE = "test_connection";

class Scanner final : public task::Task {
public:
    explicit Scanner(std::shared_ptr<task::Context> ctx, synnax::Task task);

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
