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
struct ScannerScanCommand {
    ConnectionConfig connection;

    explicit ScannerScanCommand(config::Parser& parser): connection(
        ConnectionConfig(parser.child("connection"))) {
    }
};

const std::string SCAN_CMD_TYPE = "scan";

class Scanner final : public task::Task {
public:
    explicit Scanner(std::shared_ptr<task::Context> ctx, synnax::Task task);

    void exec(task::Command& cmd) override;

    void stop() override {
    }

private:
    std::shared_ptr<task::Context> ctx;
    const synnax::Task task;


    void scan(const ScannerScanCommand& cmd, json& err, bool& ok);
};
}
