#include "synnax/framer/framer.h"
#include "synnax/ranger/ranger.h"
#include "synnax/channel/channel.h"

using namespace Synnax;

struct Config {
    std::string host;
    std::uint16_t port;
    bool secure;
    std::string username;
    std::string password;
};

namespace Synnax {
    class Synnax {
    public:
        Synnax(Config cfg) {

        }
        Ranger::Client ranges;
        Channel::Client channels;
        Framer::Client telem;
    };
}