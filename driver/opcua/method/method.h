// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <optional>
#include <string>
#include <utility>
#include <vector>

#include "open62541/types.h"

#include "x/cpp/errors/errors.h"

#include "driver/opcua/connection/connection.h"
#include "driver/opcua/types/types.h"

namespace driver::opcua::method {
/// @brief an input argument of a method, as the server declares it.
struct Argument {
    /// @brief the name of the argument.
    std::string name;
    /// @brief the node of the data type that the argument takes.
    types::NodeId data_type;
};

/// @brief reads the InputArguments property of each method.
/// @returns the input arguments of each method, in order. A method without the property
/// takes none. Nothing for a method that the server did not browse, whose browse
/// stopped before it found the property, or whose property holds no arguments. Returns
/// the error of the browse or the read.
std::pair<std::vector<std::optional<std::vector<Argument>>>, x::errors::Error>
read_inputs(
    const connection::Pool::Connection &conn,
    const std::vector<UA_NodeId> &methods
);
}
