#include "daqmx.h"
#include "synnax/synnax.h"
#include <string>
#include <vector>

namespace ni {
struct ChannelConfig {
    const synnax::ChannelKey key;
    const std::string type;
    const std::string config;
};

struct Config {
    const std::string name;
    synnax::Rate sample_rate;
    synnax::Rate transfer_rate;
    std::vector<ChannelConfig> channels;
};



class Reader {
private:
    TaskHandle task;
    Config config;

public:
    freighter::Error start();

    std::pair<synnax::Frame, freighter::Error> read();

    freighter::Error stop();
};

}