// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <map>
#include <set>
#include <utility>
#include <vector>

#include "client/cpp/channel/types.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "client/cpp/synnax.h"
#include "x/cpp/errors/errors.h"

namespace driver::icd {
/// @brief Resources holds the ICDs and channels that the messages of a task use.
struct Resources {
    /// @brief icds holds each ICD that exists, in no set order.
    std::vector<synnax::icd::ICD> icds;
    /// @brief channels holds each channel that exists, by key.
    std::map<synnax::channel::Key, synnax::channel::Channel> channels;
};

/// @brief retrieves the ICDs and channels with the given keys. It skips a key with no
/// ICD or channel, so that the caller can name the one it lacks.
/// @param client the client to retrieve with.
/// @param icds the keys of the ICDs to retrieve.
/// @param channels the keys of the channels to retrieve.
/// @returns the error of a failed retrieve.
[[nodiscard]] std::pair<Resources, x::errors::Error> retrieve(
    const synnax::Synnax &client,
    const std::set<synnax::icd::Key> &icds,
    const std::set<synnax::channel::Key> &channels
);
}
