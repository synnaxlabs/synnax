// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <fstream>
#include <system_error>

#include "x/cpp/fs/fs.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/opcua/certificate/store.h"
#include "driver/opcua/errors/errors.h"

namespace driver::opcua::certificate {
namespace {
const std::string APPLICATION_URI_PREFIX = "urn:synnax:driver:";
const std::string COMMON_NAME_PREFIX = "Synnax Driver ";
const std::string FILE_NAME = "Synnax Driver";
constexpr auto DIR_PERMISSIONS = std::filesystem::perms::owner_all;
constexpr auto KEY_PERMISSIONS = std::filesystem::perms::owner_read |
                                 std::filesystem::perms::owner_write;

x::errors::Error failed(const std::string &action, const std::filesystem::path &path) {
    return {
        errors::DRIVER_CERTIFICATE_FAILED,
        "Failed to " + action + " " + path.string()
    };
}

/// @brief writes the file through a temporary file, so a crash never leaves a partial
/// file at the path.
x::errors::Error write_file(
    const std::filesystem::path &path,
    const std::string &data,
    const std::optional<std::filesystem::perms> perms = std::nullopt
) {
    auto tmp = path;
    tmp += ".tmp";
    {
        std::ofstream out(tmp, std::ios::binary | std::ios::trunc);
        if (!out) return failed("write", tmp);
        if (perms.has_value()) {
            std::error_code ec;
            std::filesystem::permissions(tmp, *perms, ec);
            if (ec) return failed("set permissions on", tmp);
        }
        out.write(data.data(), static_cast<std::streamsize>(data.size()));
        if (!out) return failed("write", tmp);
    }
    std::error_code ec;
    std::filesystem::rename(tmp, path, ec);
    if (ec) return failed("write", path);
    return x::errors::NIL;
}

x::errors::Error make_private_dir(const std::filesystem::path &path) {
    std::error_code ec;
    std::filesystem::create_directories(path, ec);
    if (ec) return failed("create", path);
    std::filesystem::permissions(path, DIR_PERMISSIONS, ec);
    if (ec) return failed("set permissions on", path);
    return x::errors::NIL;
}
}

Store::Store(StoreConfig cfg): cfg(std::move(cfg)) {}

std::filesystem::path Store::cert_path(const Identity &id) const {
    const auto name = FILE_NAME + " [" + id.certificate.thumbprint() + "].der";
    return this->cfg.dir / "own" / "certs" / name;
}

std::filesystem::path Store::key_path(const Identity &id) const {
    const auto name = FILE_NAME + " [" + id.certificate.thumbprint() + "].pem";
    return this->cfg.dir / "own" / "private" / name;
}

x::errors::Error Store::load() {
    if (this->loaded) return x::errors::NIL;
    const auto certs = this->cfg.dir / "own" / "certs";
    std::error_code ec;
    if (!std::filesystem::exists(certs, ec)) {
        this->loaded = true;
        return x::errors::NIL;
    }
    std::vector<Identity> found;
    for (const auto &entry: std::filesystem::directory_iterator(certs, ec)) {
        if (entry.path().extension() != ".der") continue;
        auto [cert_data, read_err] = x::fs::read_file(entry.path().string());
        if (read_err) return failed("read", entry.path());
        auto [cert, parse_err] = Certificate::parse(cert_data);
        if (parse_err) return failed("parse", entry.path());
        Identity id{.certificate = std::move(cert)};
        const auto key = this->key_path(id);
        auto [key_data, key_err] = x::fs::read_file(key.string());
        if (key_err) return failed("read", key);
        id.private_key = std::move(key_data);
        found.push_back(std::move(id));
    }
    if (ec) return failed("list", certs);
    std::ranges::sort(found, [](const Identity &a, const Identity &b) {
        if (a.certificate.not_before() != b.certificate.not_before())
            return a.certificate.not_before() < b.certificate.not_before();
        return a.certificate.not_after() < b.certificate.not_after();
    });
    this->identities = std::move(found);
    this->loaded = true;
    return x::errors::NIL;
}

x::errors::Error Store::issue(const std::string &application_uri) {
    auto [id, err] = create(
        application_uri,
        COMMON_NAME_PREFIX + x::uuid::create().to_string(),
        this->cfg.host,
        this->cfg.validity
    );
    if (err) return err;
    if (const auto dir_err = make_private_dir(this->cfg.dir)) return dir_err;
    if (const auto dir_err = make_private_dir(this->cfg.dir / "own")) return dir_err;
    if (const auto dir_err = make_private_dir(this->cfg.dir / "own" / "certs"))
        return dir_err;
    if (const auto dir_err = make_private_dir(this->cfg.dir / "own" / "private"))
        return dir_err;
    // The key goes first: load() treats a certificate without a key as corrupt.
    if (const auto
            key_err = write_file(this->key_path(id), id.private_key, KEY_PERMISSIONS))
        return key_err;
    if (const auto cert_err = write_file(this->cert_path(id), id.certificate.der()))
        return cert_err;
    this->identities.push_back(std::move(id));
    return x::errors::NIL;
}

x::errors::Error Store::remove(const Identity &id) const {
    std::error_code ec;
    std::filesystem::remove(this->cert_path(id), ec);
    if (ec) return failed("delete", this->cert_path(id));
    std::filesystem::remove(this->key_path(id), ec);
    if (ec) return failed("delete", this->key_path(id));
    return x::errors::NIL;
}

std::string Store::application_uri() const {
    if (!this->identities.empty()) {
        const auto &uri = this->identities.front().certificate.application_uri();
        if (!uri.empty()) return uri;
    }
    return APPLICATION_URI_PREFIX + x::uuid::create().to_string();
}

x::errors::Error Store::promote_locked() {
    if (this->identities.size() < 2) return x::errors::NIL;
    if (const auto err = this->remove(this->identities.front())) return err;
    this->identities.erase(this->identities.begin());
    return x::errors::NIL;
}

std::pair<Identity, x::errors::Error> Store::regenerate_locked() {
    if (const auto err = this->issue(this->application_uri())) return {Identity{}, err};
    while (this->identities.size() > 1) {
        if (const auto err = this->remove(this->identities.front()))
            return {Identity{}, err};
        this->identities.erase(this->identities.begin());
    }
    return {this->identities.front(), x::errors::NIL};
}

std::pair<Identity, x::errors::Error> Store::current() {
    std::lock_guard lock(this->mu);
    if (const auto err = this->load()) return {Identity{}, err};
    if (this->identities.empty()) {
        if (const auto err = this->issue(this->application_uri()))
            return {Identity{}, err};
        return {this->identities.front(), x::errors::NIL};
    }
    const auto now = this->cfg.now();
    if (this->identities.front().certificate.valid_at(now))
        return {this->identities.front(), x::errors::NIL};
    if (this->identities.size() > 1 && this->identities[1].certificate.valid_at(now)) {
        if (const auto err = this->promote_locked()) return {Identity{}, err};
        return {this->identities.front(), x::errors::NIL};
    }
    return this->regenerate_locked();
}

std::pair<std::optional<Identity>, x::errors::Error> Store::next() {
    std::lock_guard lock(this->mu);
    if (const auto err = this->load()) return {std::nullopt, err};
    if (this->identities.size() < 2) return {std::nullopt, x::errors::NIL};
    return {this->identities[1], x::errors::NIL};
}

x::errors::Error Store::renew() {
    std::lock_guard lock(this->mu);
    if (const auto err = this->load()) return err;
    if (this->identities.size() != 1) return x::errors::NIL;
    const auto remaining = this->identities.front().certificate.not_after() -
                           this->cfg.now();
    if (remaining > RENEWAL_WINDOW) return x::errors::NIL;
    return this->issue(this->application_uri());
}

x::errors::Error Store::promote() {
    std::lock_guard lock(this->mu);
    if (const auto err = this->load()) return err;
    return this->promote_locked();
}

std::pair<Identity, x::errors::Error> Store::regenerate() {
    std::lock_guard lock(this->mu);
    if (const auto err = this->load()) return {Identity{}, err};
    return this->regenerate_locked();
}
}
