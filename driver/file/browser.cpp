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
#include "x/cpp/strings/strings.h"
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

/// @returns the path as UTF-8, or nothing when it is not valid Unicode, such as a
/// Linux file name in Latin-1.
std::optional<std::string> to_utf8(const std::filesystem::path &path) {
    std::u8string s;
    try {
        s = path.u8string();
    } catch (const std::system_error &) { return std::nullopt; }
    std::string out(s.begin(), s.end());
    if (!x::strings::valid_utf8(out)) return std::nullopt;
    return out;
}

/// @returns the path without "." or ".." segments or a trailing separator.
std::filesystem::path normal(const std::filesystem::path &path) {
    auto out = path.lexically_normal();
    if (!out.has_filename() && out.has_relative_path()) out = out.parent_path();
    return out;
}

bool certificate_extension(const std::filesystem::path &path) {
    auto ext = to_utf8(path.extension()).value_or("");
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
    std::string path_utf8;
    bool directory = false;
    bool link = false;
};

/// @brief the secrets folder, which it compares by file identity, so that another
/// spelling of the folder, such as one in another case or through a link, matches.
class Secrets {
    std::filesystem::path dir;

public:
    /// @param dir the secrets folder. A folder that does not exist holds nothing, but
    /// another error when it checks the folder fails closed.
    explicit Secrets(std::filesystem::path dir): dir(std::move(dir)) {
        std::error_code ec;
        if (!this->dir.empty() && !std::filesystem::exists(this->dir, ec) && !ec)
            this->dir.clear();
    }

    /// @returns true if path is the folder, or when the comparison fails.
    [[nodiscard]] bool is(const std::filesystem::path &path) const {
        if (this->dir.empty()) return false;
        std::error_code ec;
        return std::filesystem::equivalent(path, this->dir, ec) || ec;
    }

    /// @returns true if path or one of its parents is the folder, or when a comparison
    /// fails.
    [[nodiscard]] bool contains(const std::filesystem::path &path) const {
        if (this->dir.empty()) return false;
        for (auto p = path;; p = p.parent_path()) {
            if (this->is(p)) return true;
            if (p == p.parent_path()) return false;
        }
    }
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
    if (!this->cfg.secrets_dir.empty())
        this->cfg.secrets_dir = normal(this->cfg.secrets_dir);
}

std::string Browser::secrets_dir() const {
    return to_utf8(this->cfg.secrets_dir).value_or("");
}

x::json::json Browser::top() const {
    auto entries = x::json::json::array();
    const auto add = [&](const std::filesystem::path &path,
                         const std::filesystem::path &name) {
        const auto name_utf8 = to_utf8(name);
        const auto path_utf8 = to_utf8(path);
        if (!name_utf8 || !path_utf8) return;
        // A home or secrets folder that is a root shows once, as the root.
        if (std::ranges::any_of(entries, [&](const auto &e) {
                return e["path"] == *path_utf8;
            }))
            return;
        entries.push_back(
            {{"name", *name_utf8}, {"path", *path_utf8}, {"kind", "directory"}}
        );
    };
    for (const auto &root: this->cfg.roots)
        add(root, root);
    for (const auto &dir: {this->cfg.home, this->cfg.secrets_dir}) {
        if (dir.empty()) continue;
        const auto path = normal(dir);
        add(path, path.has_filename() ? path.filename() : path);
    }
    return reply("", "", this->secrets_dir(), false, false, entries);
}

std::pair<x::json::json, x::errors::Error>
Browser::list(const std::string &path) const {
    if (path.empty()) return {this->top(), x::errors::NIL};
    const auto requested = to_path(path);
    if (!requested.is_absolute())
        return {{}, {x::fs::INVALID_PATH, path + " is not an absolute path."}};
    const auto dir = normal(requested);
    // The path came from UTF-8, so it and its parent convert back.
    const auto dir_name = to_utf8(dir).value();
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
        const bool link = it->is_symlink(entry_ec);
        if (entry_ec) continue;
        const auto path = dir / it->path().filename();
        auto name = to_utf8(path.filename());
        auto path_utf8 = to_utf8(path);
        if (!name || !path_utf8) continue;
        found.push_back(
            {std::move(*name), path, std::move(*path_utf8), directory, link}
        );
    }
    if (ec) return {{}, read_error(ec, dir_name)};
    std::ranges::sort(found, [](const Entry &a, const Entry &b) {
        if (a.directory != b.directory) return a.directory;
        return a.name < b.name;
    });
    const bool truncated = found.size() > MAX_ENTRIES;
    if (truncated) found.resize(MAX_ENTRIES);

    const Secrets secrets(this->cfg.secrets_dir);
    std::error_code real_ec;
    const auto real_dir = std::filesystem::canonical(dir, real_ec);
    const bool listed_secret = real_ec || secrets.contains(real_dir);
    auto entries = x::json::json::array();
    for (const auto &e: found) {
        x::json::json entry = {
            {"name", e.name},
            {"path", e.path_utf8},
            {"kind", e.directory ? "directory" : "file"},
        };
        // The check and the reads use one resolved path, so a link that changes
        // between them cannot lead the reads into the secrets folder. The parents of
        // an entry that is not a link are the listed folder's.
        std::error_code resolve_ec;
        const auto target = std::filesystem::canonical(e.path, resolve_ec);
        if (listed_secret || resolve_ec ||
            (e.link ? secrets.contains(target) : secrets.is(target))) {
            entries.push_back(std::move(entry));
            continue;
        }
        std::error_code stat_ec;
        const auto modified = std::filesystem::last_write_time(target, stat_ec);
        if (stat_ec) continue;
        const std::uintmax_t size = e.directory
                                      ? 0
                                      : std::filesystem::file_size(target, stat_ec);
        if (stat_ec) continue;
        entry["size"] = size;
        entry["modified"] = std::to_string(to_timestamp(modified).nanoseconds());
        if (!e.directory && size <= MAX_CERTIFICATE_SIZE &&
            certificate_extension(e.path))
            if (auto details = read_certificate(target))
                entry["certificate"] = std::move(*details);
        entries.push_back(std::move(entry));
    }
    const auto parent = dir.parent_path();
    return {
        reply(
            dir_name,
            parent == dir ? "" : to_utf8(parent).value(),
            this->secrets_dir(),
            listed_secret,
            truncated,
            std::move(entries)
        ),
        x::errors::NIL
    };
}
}
