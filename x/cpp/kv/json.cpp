// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#ifdef _WIN32
#include <fstream>
#include <sstream>

#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
#include <windows.h>
#else
#include <cerrno>
#include <cstring>

#include <fcntl.h>
#include <sys/stat.h>
#include <unistd.h>
#endif
#include <filesystem>
#include <utility>

#include "nlohmann/json.hpp"

#include "x/cpp/errors/errors.h"
#include "x/cpp/kv/kv.h"

using nlohmann::json;

namespace x::kv {
namespace {
errors::Error
check_minimum_permissions(const std::filesystem::path &path, const char *context) {
    const auto perms = std::filesystem::status(path).permissions();
    if ((perms & std::filesystem::perms::owner_write) == std::filesystem::perms::none ||
        (perms & std::filesystem::perms::owner_read) == std::filesystem::perms::none) {
        return errors::Error("insufficient permissions on " + std::string(context));
    }
    return errors::NIL;
}

#ifdef _WIN32
std::pair<std::string, errors::Error> read_file(const std::filesystem::path &path) {
    std::ifstream file(path);
    if (!file.is_open()) return {"", errors::Error("failed to open file")};
    std::stringstream ss;
    ss << file.rdbuf();
    return {ss.str(), errors::NIL};
}

errors::Error write_file(
    const std::filesystem::path &path,
    std::filesystem::perms,
    const std::string &data
) {
    std::ofstream file(path);
    if (!file.is_open()) return errors::Error("failed to open file for writing");
    file << data;
    return errors::NIL;
}
#else
/// @brief a descriptor that closes itself.
struct FD {
    int fd = -1;
    ~FD() {
        if (this->fd >= 0) close(this->fd);
    }
};

errors::Error errno_error(const std::string &action) {
    return errors::Error(action + ": " + std::strerror(errno));
}

std::filesystem::path parent_of(const std::filesystem::path &path) {
    return path.has_parent_path() ? path.parent_path() : std::filesystem::path(".");
}

/// @brief reads the file at path, and refuses a link.
std::pair<std::string, errors::Error> read_file(const std::filesystem::path &path) {
    FD file{open(path.c_str(), O_RDONLY | O_NOFOLLOW | O_CLOEXEC)};
    if (file.fd < 0) return {"", errno_error("failed to open file")};
    std::string data;
    char buf[4096];
    ssize_t n = 0;
    while ((n = read(file.fd, buf, sizeof(buf))) > 0)
        data.append(buf, static_cast<size_t>(n));
    if (n < 0) return {"", errno_error("failed to read file")};
    return {data, errors::NIL};
}

/// @brief replaces the file at path with data through a renamed temporary file, so a
/// link at path is replaced and never followed. The file keeps the owner and mode of
/// the file it replaces, or gets mode and the effective user when it is new.
errors::Error write_file(
    const std::filesystem::path &path,
    const std::filesystem::perms mode,
    const std::string &data
) {
    FD dir{
        open(parent_of(path).c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC)
    };
    if (dir.fd < 0) return errno_error("failed to open directory");
    const auto name = path.filename().string();
    const auto tmp = name + ".tmp";
    struct stat old{};
    bool replaces = false;
    {
        FD prev{openat(dir.fd, name.c_str(), O_RDONLY | O_NOFOLLOW | O_CLOEXEC)};
        if (prev.fd >= 0) {
            if (fstat(prev.fd, &old) != 0) return errno_error("failed to read file");
            replaces = true;
        } else if (errno != ENOENT && errno != ELOOP)
            return errno_error("failed to open file");
    }
    unlinkat(dir.fd, tmp.c_str(), 0);
    const auto new_mode = replaces ? old.st_mode & 07777 : static_cast<mode_t>(mode);
    FD file{openat(
        dir.fd,
        tmp.c_str(),
        O_WRONLY | O_CREAT | O_EXCL | O_NOFOLLOW | O_CLOEXEC,
        S_IRUSR | S_IWUSR
    )};
    if (file.fd < 0) return errno_error("failed to create file");
    const auto fail = [&](const std::string &action) {
        auto err = errno_error(action);
        unlinkat(dir.fd, tmp.c_str(), 0);
        return err;
    };
    if (replaces && (old.st_uid != geteuid() || old.st_gid != getegid()) &&
        fchown(file.fd, old.st_uid, old.st_gid) != 0)
        return fail("failed to keep the owner of file");
    if (fchmod(file.fd, new_mode) != 0) return fail("failed to set the mode of file");
    const char *p = data.data();
    size_t left = data.size();
    while (left > 0) {
        const auto n = write(file.fd, p, left);
        if (n < 0) return fail("failed to write file");
        p += n;
        left -= static_cast<size_t>(n);
    }
    if (fsync(file.fd) != 0) return fail("failed to write file");
    if (renameat(dir.fd, tmp.c_str(), dir.fd, name.c_str()) != 0)
        return fail("failed to replace file");
    return errors::NIL;
}
#endif
}

JSONFile::JSONFile(
    const std::string &path,
    json data,
    const std::filesystem::perms file_mode
):
    path(path), data(std::move(data)), file_mode(file_mode) {}

std::pair<std::shared_ptr<KV>, errors::Error>
JSONFile::open_file(const JSONFileConfig &config) {
    const auto empty = [&] {
        return std::make_shared<JSONFile>(
            config.path.string(),
            json::object(),
            config.file_mode
        );
    };
    try {
        const auto dir = config.path.parent_path();
        if (!std::filesystem::exists(dir)) {
            auto parent_dir = dir.parent_path();
            if (std::filesystem::exists(parent_dir))
                if (auto err = check_minimum_permissions(
                        parent_dir,
                        "parent directory"
                    ))
                    return {empty(), err};
            std::filesystem::create_directories(dir);
            std::filesystem::permissions(dir, config.dir_mode);
        }
        if (!std::filesystem::exists(std::filesystem::symlink_status(config.path))) {
            if (auto err = write_file(config.path, config.file_mode, "{}"))
                return {empty(), err};
            return {empty(), errors::NIL};
        }
        auto [raw, read_err] = read_file(config.path);
        if (read_err) return {empty(), read_err};
        json data;
        try {
            data = json::parse(raw);
        } catch (const json::exception &e) {
            return {
                empty(),
                errors::Error("failed to parse JSON: " + std::string(e.what()))
            };
        }
        return {
            std::make_shared<JSONFile>(config.path.string(), data, config.file_mode),
            errors::NIL
        };
    } catch (const std::exception &e) {
        return {
            empty(),
            errors::Error("filesystem operation failed: " + std::string(e.what()))
        };
    }
}

errors::Error JSONFile::set(const std::string &key, const std::string &value) {
    this->data[key] = value;
    return write_file(this->path, this->file_mode, this->data.dump(4));
}

errors::Error JSONFile::get(const std::string &key, std::string &value) {
    if (!this->data.contains(key)) return errors::NOT_FOUND;
    try {
        value = this->data[key].get<std::string>();
    } catch (const json::exception &e) {
        return errors::Error("failed to get value: " + std::string(e.what()));
    }
    return errors::NIL;
}

errors::Error JSONFile::del(const std::string &key) {
    if (!this->data.contains(key)) return errors::NIL;
    this->data.erase(key);
    return write_file(this->path, this->file_mode, this->data.dump(4));
}
}
