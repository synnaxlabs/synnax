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
#include <mutex>
#include <optional>
#include <string>
#include <utility>
#include <vector>

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/mono_clock.h"
#include "x/cpp/telem/telem.h"

#include "driver/opcua/certificate/certificate.h"

namespace driver::opcua::certificate {
/// @brief how long before expiry the Store issues the next certificate.
inline const x::telem::TimeSpan RENEWAL_WINDOW = 60 * x::telem::DAY;
/// @brief the validity period of a certificate that the Store issues.
inline const x::telem::TimeSpan DEFAULT_VALIDITY = x::telem::DAY * 3 * 365;

/// @brief the configuration of a Store.
struct StoreConfig {
    /// @brief the root of the OPC UA Part 12 Annex F directory layout.
    std::filesystem::path dir;
    /// @brief the DNS name for the subject alternative name of new certificates.
    std::string host;
    /// @brief the validity period of new certificates.
    x::telem::TimeSpan validity = DEFAULT_VALIDITY;
    /// @brief returns the time that expiry and renewal are checked against.
    x::telem::NowFunc now = x::telem::TimeStamp::now;
};

/// @brief keeps the identities that the Driver presents to OPC UA servers. The
/// private keys stay in the directory, which only the owner can read. The Store
/// issues the next identity before the current one expires and switches to it on
/// promote() or at expiry. Safe for concurrent use.
class Store {
public:
    explicit Store(StoreConfig cfg);

    /// @returns the current identity. Makes one on first use, and switches to the
    /// next one when the current one has expired.
    /// @returns errors::DRIVER_CERTIFICATE_FAILED if the files cannot be read or
    /// written.
    std::pair<Identity, x::errors::Error> current();

    /// @returns the next identity, or nothing if renewal has not started.
    std::pair<std::optional<Identity>, x::errors::Error> next();

    /// @brief issues the next identity if the current one expires inside the
    /// renewal window and no next identity exists.
    x::errors::Error renew();

    /// @brief makes the next identity current and deletes the old one. Does nothing
    /// if there is no next identity.
    x::errors::Error promote();

    /// @brief replaces the current and next identities with a new one.
    /// @returns the new identity.
    std::pair<Identity, x::errors::Error> regenerate();

private:
    StoreConfig cfg;
    std::mutex mu;
    /// @brief the identities in order of issue: the current one, then the next.
    std::vector<Identity> identities;
    bool loaded = false;

    x::errors::Error load();
    x::errors::Error issue(const std::string &application_uri);
    [[nodiscard]] x::errors::Error remove(const Identity &id) const;
    x::errors::Error promote_locked();
    std::pair<Identity, x::errors::Error> regenerate_locked();
    /// @returns the URI of the current identity, which stays the same across
    /// renewals, or a new URI if there is none.
    [[nodiscard]] std::string application_uri() const;
    [[nodiscard]] std::filesystem::path cert_path(const Identity &id) const;
    [[nodiscard]] std::filesystem::path key_path(const Identity &id) const;
};
}
