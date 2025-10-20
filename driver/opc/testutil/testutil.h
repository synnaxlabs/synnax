// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "open62541/client.h"

#include "x/cpp/telem/telem.h"

namespace opc::testutil {
std::pair<::telem::Series, xerrors::Error>
simple_read(std::shared_ptr<UA_Client> client, const std::string &node_id);
}
