// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <filesystem>
#include <memory>
#include <string>
#include <utility>

#include "open62541/client.h"

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/opcua/certificate/store.h"
#include "driver/opcua/connection/connection.h"

namespace driver::opcua::testutil {
std::pair<::x::telem::Series, x::errors::Error>
simple_read(const std::shared_ptr<UA_Client> &client, const std::string &node_id);

/// @brief creates a Store over a new directory in the temporary directory. The Store
/// writes to the directory only when a secured connection needs a Driver certificate.
inline std::shared_ptr<certificate::Store> create_store() {
    return std::make_shared<certificate::Store>(certificate::StoreConfig{
        .dir = std::filesystem::temp_directory_path() /
               ("opcua-pki-" + x::uuid::create().to_string()),
    });
}

/// @returns the time now plus offset, rounded to the microsecond so that it survives
/// the 100 ns resolution of an OPC UA DateTime.
inline x::telem::TimeStamp now(const x::telem::TimeSpan offset = {}) {
    const auto ns = (x::telem::TimeStamp::now() + offset).nanoseconds();
    return x::telem::TimeStamp(ns / 1000 * 1000);
}

/// @brief encodes the configuration as the connection properties of a device.
inline x::json::json connection_json(const connection::Config &cfg) {
    x::json::json j = {
        {"endpoint", cfg.endpoint},
        {"username", cfg.username},
        {"password", cfg.password},
        {"security_mode", cfg.security_mode},
        {"security_policy", cfg.security_policy},
        {"client_certificate", cfg.client_cert},
        {"client_private_key", cfg.client_private_key},
        {"certificate_validation_disabled", cfg.certificate_validation_disabled},
        {"client_timeout_ms", cfg.client_timeout_ms},
    };
    if (cfg.server_certificate.has_value())
        j["server_certificate"] = cfg.server_certificate->pem();
    return j;
}
}
