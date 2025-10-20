#include "open62541/client.h"

#include "x/cpp/telem/telem.h"

namespace opc::testutil {
std::pair<::telem::Series, xerrors::Error>
simple_read(std::shared_ptr<UA_Client> client, const std::string &node_id);
}
