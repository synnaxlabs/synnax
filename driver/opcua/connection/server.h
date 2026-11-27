// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "client/cpp/opcua/server/types.gen.h"

#include "driver/opcua/connection/connection.h"

namespace driver::opcua::connection {
/// @brief returns the configuration that connects to a stored OPC UA server. The
/// password and certificates stay unread, as for a Config parsed from JSON.
Config from_server(const ::synnax::opcua::server::Connection &conn);
}
