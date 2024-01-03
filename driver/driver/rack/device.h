#include <string>
#include <unordered_map>

#pragma once

namespace rack {


class Module {
public:
    const std::string key;
    const std::string type;
    virtual void start() = 0;
    virtual void stop() = 0;
};

struct RackConfig {
    std::string data_path;
};

class RackKey {
public:
    std::uint16_t node() {

    }


private:
    std::uint32_t value;


};

class Rack {
private:
    std::string key;
    std::unordered_map<std::string, std::unique_ptr<Module>> modules;
};

}