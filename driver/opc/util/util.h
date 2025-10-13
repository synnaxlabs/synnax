// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <memory>
#include <string>
#include <utility>

/// external
#include "open62541/client.h"

/// module
#include "x/cpp/telem/series.h"
#include "x/cpp/xerrors/errors.h"

/// internal
#include "driver/opc/errors/errors.h"
#include "driver/opc/telem/telem.h"
#include "driver/opc/types/types.h"

namespace util {
std::pair<telem::Series, xerrors::Error>
simple_read(std::shared_ptr<UA_Client> client, const std::string &node_id);
}
