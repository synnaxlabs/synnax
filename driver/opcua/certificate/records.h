// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <optional>
#include <string>
#include <utility>
#include <vector>

#include "client/cpp/synnax.h"
#include "x/cpp/errors/errors.h"

#include "driver/opcua/certificate/store.h"

namespace driver::opcua::certificate {
/// @brief keeps one file certificate in the Core for each identity in a Store, on the
/// rack of the Driver. A record names the certificate file, never the private key. Not
/// safe for concurrent use.
class Records {
public:
    Records(
        std::shared_ptr<Store> store,
        std::shared_ptr<synnax::Synnax> client,
        synnax::rack::Key rack
    );

    /// @brief saves one record for each identity, and deletes every other record of a
    /// Store certificate file, in any directory. Calls the Core only when the
    /// identities or the rack name changed since the last sync that succeeded.
    /// @param rack_name the name of the rack, which starts the name of each record.
    [[nodiscard]] x::errors::Error sync(const std::string &rack_name);

private:
    std::shared_ptr<Store> store;
    std::shared_ptr<synnax::Synnax> client;
    synnax::rack::Key rack;
    /// @brief the name and path of each record that the last sync saved.
    std::optional<std::vector<std::pair<std::string, std::string>>> synced;
};
}
