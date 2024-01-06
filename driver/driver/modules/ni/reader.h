#include "daqmx.h"
#include "synnax/synnax.h"
#include <string>
#include <vector>

namespace ni {
class Reader {
private:
    TaskHandle task;

    Reader();
public:
    std::pair<synnax::Frame, freighter::Error> read();

    freighter::Error configure(synnax::Module config);

    freighter::Error stop();
};

}
class Factory {
};