#pragma once

#include <string>
#include <memory>
#include "nlohmann/json.hpp"
#include "client/cpp/synnax/synnax.h"

using json = nlohmann::json;


namespace opcua {
struct ScannerConfig {
    std::shared_ptr<synnax::Synnax> client;
};

struct ScannerScanCommand {
    std::string endpoint;
    std::string username;
    std::string password;

    ScannerScanCommand(const json &cmd, json& err, bool &ok);
};

const std::string SCAN_CMD_TYPE = "scan";

class Scanner {
public:
    explicit Scanner(const ScannerConfig& config);

    void exec(std::string type, const json &cmd, json &err, bool &ok);
private:
    std::shared_ptr<synnax::Synnax> client;


    void scan(const ScannerScanCommand &cmd, json &err, bool &ok);
};
}