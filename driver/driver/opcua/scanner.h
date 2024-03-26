#pragma once

#include <string>
#include <memory>
#include "nlohmann/json.hpp"
#include "synnax/synnax.h"

using json = nlohmann::json;


namespace opcua {
struct scannerConfig {
    std::shared_ptr<synnax::Synnax> client;
};

struct scannerScanCmd {
    std::string endpoint;
    std::string username;
    std::string password;

    scannerScanCmd(const json &cmd, json& err, bool &ok);
};

const std::string SCAN_CMD_TYPE = "scan";

class scanner {
public:
    scanner(const scannerConfig& config);

    void exec(std::string type, const json &cmd, json &err, bool &ok);
private:
    std::shared_ptr<synnax::Synnax> client;


    void scan(const scannerScanCmd &cmd);
};
}