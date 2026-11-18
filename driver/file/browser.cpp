// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <array>
#include <cctype>
#include <fstream>
#include <iterator>
#include <optional>
#include <string>
#include <string_view>
#include <system_error>
#include <utility>
#include <vector>

#include "x/cpp/fs/fs.h"
#include "x/cpp/x509/x509.h"

#include "driver/file/file.h"

namespace driver::file {
namespace {
constexpr std::array<std::string_view, 5> CERTIFICATE_EXTENSIONS = {
    ".pem",
    ".crt",
    ".cer",
    ".der",
    ".cert",
};

std::filesystem::path to_path(const std::string &utf8) {
    return {std::u8string(utf8.begin(), utf8.end())};
}

std::string to_utf8(const std::filesystem::path &path) {
    const auto s = path.u8string();
    return {s.begin(), s.end()};
}

/// @returns the path without "." or ".." segments or a trailing separator.
std::filesystem::path normal(const std::filesystem::path &path) {
    auto out = path.lexically_normal();
    if (!out.has_filename() && out.has_relative_path()) out = out.parent_path();
    return out;
}

/// @returns true if every segment of prefix starts path.
bool starts_with(
    const std::filesystem::path &path,
    const std::filesystem::path &prefix
) {
    return std::ranges::mismatch(prefix, path).in1 == prefix.end();
}

bool certificate_extension(const std::filesystem::path &path) {
    auto ext = to_utf8(path.extension());
    std::ranges::transform(ext, ext.begin(), [](const unsigned char c) {
        return static_cast<char>(std::tolower(c));
    });
    return std::ranges::find(CERTIFICATE_EXTENSIONS, ext) !=
           CERTIFICATE_EXTENSIONS.end();
}

/// @returns the details of the certificate in the shape of the Core's
/// certificate.Details, or nothing if the file holds no certificate.
std::optional<x::json::json> read_certificate(const std::filesystem::path &path) {
    std::ifstream in(path, std::ios::binary);
    if (!in) return std::nullopt;
    const std::string data{
        std::istreambuf_iterator<char>(in),
        std::istreambuf_iterator<char>()
    };
    const auto [cert, err] = x::x509::Certificate::parse(data);
    if (err) return std::nullopt;
    return x::json::json{
        {"subject", cert.subject()},
        {"issuer", cert.issuer()},
        {"not_before", std::to_string(cert.not_before().nanoseconds())},
        {"not_after", std::to_string(cert.not_after().nanoseconds())},
        {"sha1", cert.sha1()},
        {"sha256", cert.sha256()},
    };
}

x::errors::Error read_error(const std::error_code &ec, const std::string &path) {
    if (ec == std::errc::permission_denied)
        return {x::fs::PERMISSION_DENIED, "The Driver cannot read " + path + "."};
    return {x::fs::READ_ERROR, "Failed to read " + path + ": " + ec.message() + "."};
}

/// @brief a file or folder that a reply lists.
struct Entry {
    std::string name;
    std::filesystem::path path;
    bool directory = false;
};

x::json::json reply(
    const std::string &path,
    const std::string &parent,
    const std::string &secrets_dir,
    const bool secret,
    const bool truncated,
    x::json::json entries
) {
    return {
        {"path", path},
        {"parent", parent},
        {"secrets_dir", secrets_dir},
        {"secret", secret},
        {"truncated", truncated},
        {"entries", std::move(entries)},
    };
}
}

Browser::Browser(Config cfg): cfg(std::move(cfg)) {
    if (this->cfg.secrets_dir.empty()) return;
    this->cfg.secrets_dir = normal(this->cfg.secrets_dir);
    std::error_code ec;
    this->secrets = std::filesystem::weakly_canonical(this->cfg.secrets_dir, ec);
    if (ec) this->secrets = this->cfg.secrets_dir;
}

bool Browser::secret(const std::filesystem::path &path) const {
    if (this->secrets.empty()) return false;
    std::error_code ec;
    const auto resolved = std::filesystem::weakly_canonical(path, ec);
    // A path that does not resolve can still lead into the secrets folder.
    return ec || starts_with(resolved, this->secrets);
}

x::json::json Browser::top() const {
    auto entries = x::json::json::array();
    for (const auto &root: this->cfg.roots)
        entries.push_back(
            {{"name", to_utf8(root)}, {"path", to_utf8(root)}, {"kind", "directory"}}
        );
    for (const auto &dir: {this->cfg.home, this->cfg.secrets_dir}) {
        if (dir.empty()) continue;
        const auto path = normal(dir);
        entries.push_back(
            {{"name", to_utf8(path.has_filename() ? path.filename() : path)},
             {"path", to_utf8(path)},
             {"kind", "directory"}}
        );
    }
    return reply("", "", to_utf8(this->cfg.secrets_dir), false, false, entries);
}

std::pair<x::json::json, x::errors::Error>
Browser::list(const std::string &path) const {
    if (path.empty()) return {this->top(), x::errors::NIL};
    const auto requested = to_path(path);
    if (!requested.is_absolute())
        return {{}, {x::fs::INVALID_PATH, path + " is not an absolute path."}};
    const auto dir = normal(requested);
    const auto dir_name = to_utf8(dir);
    std::error_code ec;
    const auto status = std::filesystem::status(dir, ec);
    if (status.type() == std::filesystem::file_type::not_found)
        return {{}, {x::fs::NOT_FOUND, dir_name + " does not exist."}};
    if (ec) return {{}, read_error(ec, dir_name)};
    if (!std::filesystem::is_directory(status))
        return {{}, {x::fs::INVALID_PATH, dir_name + " is not a folder."}};

    std::vector<Entry> found;
    for (auto it = std::filesystem::directory_iterator(dir, ec);
         !ec && it != std::filesystem::directory_iterator();
         it.increment(ec)) {
        std::error_code entry_ec;
        const auto entry_status = it->status(entry_ec);
        if (entry_ec) continue;
        const bool directory = std::filesystem::is_directory(entry_status);
        if (!directory && !std::filesystem::is_regular_file(entry_status)) continue;
        const auto name = it->path().filename();
        found.push_back({to_utf8(name), dir / name, directory});
    }
    if (ec) return {{}, read_error(ec, dir_name)};
    std::ranges::sort(found, [](const Entry &a, const Entry &b) {
        if (a.directory != b.directory) return a.directory;
        return a.name < b.name;
    });
    const bool truncated = found.size() > MAX_ENTRIES;
    if (truncated) found.resize(MAX_ENTRIES);

    const bool listed_secret = this->secret(dir);
    auto entries = x::json::json::array();
    for (const auto &e: found) {
        x::json::json entry = {
            {"name", e.name},
            {"path", to_utf8(e.path)},
            {"kind", e.directory ? "directory" : "file"},
        };
        if (listed_secret || this->secret(e.path)) {
            entries.push_back(std::move(entry));
            continue;
        }
        std::error_code stat_ec;
        const auto modified = std::filesystem::last_write_time(e.path, stat_ec);
        if (stat_ec) continue;
        const std::uintmax_t size = e.directory
                                      ? 0
                                      : std::filesystem::file_size(e.path, stat_ec);
        if (stat_ec) continue;
        entry["size"] = size;
        entry["modified"] = std::to_string(to_timestamp(modified).nanoseconds());
        if (!e.directory && size <= MAX_CERTIFICATE_SIZE &&
            certificate_extension(e.path))
            if (auto details = read_certificate(e.path))
                entry["certificate"] = std::move(*details);
        entries.push_back(std::move(entry));
    }
    const auto parent = dir.parent_path();
    return {
        reply(
            dir_name,
            parent == dir ? "" : to_utf8(parent),
            to_utf8(this->cfg.secrets_dir),
            listed_secret,
            truncated,
            std::move(entries)
        ),
        x::errors::NIL
    };
}
}
