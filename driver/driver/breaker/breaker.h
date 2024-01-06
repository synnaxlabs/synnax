#include "synnax/synnax.h"
#include <chrono>
#include <thread>

namespace breaker {


struct Config {
    synnax::TimeSpan base_interval;
    uint32_t max_retries;
    float_t scale;
};


class Breaker {
public:
    Breaker(Config config) : config(config), interval(config.base_interval), retries(0) {}

    bool wait() {
        if (retries >= config.max_retries) return false;
        std::this_thread::sleep_for(std::chrono::nanoseconds(interval.value));
        interval = interval * config.scale;
        retries++;
        return true;
    }

    void reset() {
        retries = 0;
        interval = config.base_interval;
    }
private:
    Config config;
    synnax::TimeSpan interval;
    uint32_t retries;
};
}

