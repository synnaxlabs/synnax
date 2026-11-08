// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <filesystem>
#include <string>

#include "absl/log/log.h"

#ifdef _WIN32
// clang-format off
// aclapi.h and sddl.h must be included after windows.h.
#include <windows.h>
#include <aclapi.h>
#include <sddl.h>
// clang-format on
#else
#include <cerrno>
#include <cstring>

#include <fcntl.h>
#include <pwd.h>
#include <sys/stat.h>
#include <unistd.h>
#endif

#include "x/cpp/kv/kv.h"

#include "driver/rack/rack.h"

namespace fs = std::filesystem;

namespace driver::rack {
constexpr auto PERSISTED_STATE_FILE_PERMISSIONS = fs::perms::owner_read |
                                                  fs::perms::owner_write;

constexpr auto PERSISTED_STATE_DIR_PERMISSIONS = fs::perms::owner_all;

namespace {
const std::string STATE_FILE = "persisted-state.json";

/// @brief true when the process runs with admin rights, as the service does.
bool elevated() {
#ifdef _WIN32
    SID_IDENTIFIER_AUTHORITY authority = SECURITY_NT_AUTHORITY;
    PSID admins = nullptr;
    if (AllocateAndInitializeSid(
            &authority,
            2,
            SECURITY_BUILTIN_DOMAIN_RID,
            DOMAIN_ALIAS_RID_ADMINS,
            0,
            0,
            0,
            0,
            0,
            0,
            &admins
        ) == 0)
        return false;
    BOOL member = FALSE;
    if (CheckTokenMembership(nullptr, admins, &member) == 0) member = FALSE;
    FreeSid(admins);
    return member != FALSE;
#else
    return geteuid() == 0;
#endif
}

/// @brief the state directory of the computer, which the service reads.
fs::path machine_state_dir() {
#ifdef _WIN32
    return "C:\\ProgramData\\synnax-driver";
#elif defined(__APPLE__)
    return "/Library/Application Support/synnax-driver";
#else
    return "/var/lib/synnax-driver";
#endif
}

/// @brief the state directory of the user, or of the user that ran sudo. Empty on
/// Linux, where each process uses the directory of the computer.
fs::path user_state_dir() {
#ifdef _WIN32
    if (const char *appdata = std::getenv("LOCALAPPDATA"))
        return fs::path(appdata) / "synnax-driver";
#elif defined(__APPLE__)
    const char *home = std::getenv("HOME");
    if (const char *sudo_user = std::getenv("SUDO_USER"))
        if (const passwd *user = getpwnam(sudo_user)) home = user->pw_dir;
    if (home != nullptr)
        return fs::path(home) / "Library" / "Application Support" / "synnax-driver";
#endif
    return {};
}
}

std::string get_persisted_state_path(x::args::Parser &parser) {
    auto p_path = parser.field<std::string>("--state-file", "");
    if (!p_path.empty()) return p_path;
    auto dir = user_state_dir();
    if (dir.empty() || elevated()) dir = machine_state_dir();
    return (dir / STATE_FILE).string();
}

#ifndef _WIN32
/// @brief a descriptor that closes itself.
struct FD {
    int fd = -1;
    ~FD() {
        if (this->fd >= 0) close(this->fd);
    }
};

x::errors::Error errno_error(const std::string &action) {
    return x::errors::Error(STATE_ERROR, action + ": " + std::strerror(errno));
}

/// @brief writes all of data to fd, retrying short writes.
bool write_all(const int fd, const char *data, size_t size) {
    while (size > 0) {
        const auto n = write(fd, data, size);
        if (n < 0) return false;
        data += n;
        size -= static_cast<size_t>(n);
    }
    return true;
}

/// @brief replaces a state file that another user owns with a copy this process owns,
/// so its mode can be made owner-only.
x::errors::Error take_state_file(const int dir_fd, const std::string &name) {
    FD src{openat(dir_fd, name.c_str(), O_RDONLY | O_NOFOLLOW | O_CLOEXEC)};
    if (src.fd < 0) {
        if (errno == ENOENT) return x::errors::NIL;
        return errno_error("failed to open state file");
    }
    struct stat st{};
    if (fstat(src.fd, &st) != 0) return errno_error("failed to read state file");
    if (st.st_uid == geteuid()) return x::errors::NIL;
    const auto tmp = name + ".tmp";
    unlinkat(dir_fd, tmp.c_str(), 0);
    FD dst{openat(
        dir_fd,
        tmp.c_str(),
        O_WRONLY | O_CREAT | O_EXCL | O_NOFOLLOW | O_CLOEXEC,
        S_IRUSR | S_IWUSR
    )};
    if (dst.fd < 0) return errno_error("failed to take over state file");
    char buf[4096];
    ssize_t n = 0;
    while ((n = read(src.fd, buf, sizeof(buf))) > 0)
        if (!write_all(dst.fd, buf, static_cast<size_t>(n))) break;
    if (n != 0 || fsync(dst.fd) != 0 ||
        renameat(dir_fd, tmp.c_str(), dir_fd, name.c_str()) != 0) {
        auto err = errno_error("failed to take over state file");
        unlinkat(dir_fd, tmp.c_str(), 0);
        return err;
    }
    return x::errors::NIL;
}

/// @brief creates the OPC UA directory in the state directory if it is missing and
/// gives it to the service user. Root never writes in it.
x::errors::Error give_opcua_dir(const int dir_fd, const passwd &user) {
    if (mkdirat(dir_fd, OPCUA_STATE_DIR.c_str(), S_IRWXU) != 0 && errno != EEXIST)
        return errno_error("failed to create OPC UA directory");
    FD sub{openat(
        dir_fd,
        OPCUA_STATE_DIR.c_str(),
        O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC
    )};
    if (sub.fd < 0) return errno_error("failed to open OPC UA directory");
    if (fchown(sub.fd, user.pw_uid, user.pw_gid) != 0 || fchmod(sub.fd, S_IRWXU) != 0)
        return errno_error("failed to give OPC UA directory to the service user");
    return x::errors::NIL;
}

/// @brief opens the state directory and file without following links, so a link put
/// in their place is refused. A file that does not exist yet is skipped. Only the
/// default directory is dedicated to the Driver, so only it changes mode or owner,
/// through the open descriptors. When root writes it, as `sudo synnax-driver login`
/// does, root keeps the directory and the Linux service user gets read access to it
/// and owns the file and the OPC UA directory. The service can then read and write its
/// key but cannot put a link in the directory for root to follow. A process that is not
/// root and does not own the file takes it over first.
x::errors::Error secure_state(const fs::path &path, const bool dedicated) {
    FD dir{open(
        path.parent_path().c_str(),
        O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC
    )};
    if (dir.fd < 0) {
        if (errno == ENOENT) return x::errors::NIL;
        return errno_error("failed to open state directory");
    }
    const passwd *service_user = nullptr;
#ifdef __linux__
    if (geteuid() == 0) service_user = getpwnam("synnax");
#endif
    if (dedicated) {
        struct stat st{};
        if (fstat(dir.fd, &st) != 0)
            return errno_error("failed to read state directory");
        if (service_user != nullptr) {
            // Other users may pass through to run the service binary in bin/.
            if (fchown(dir.fd, 0, service_user->pw_gid) != 0 ||
                fchmod(dir.fd, S_IRWXU | S_IRGRP | S_IXGRP | S_IXOTH) != 0)
                return errno_error("failed to secure state directory");
            if (auto err = give_opcua_dir(dir.fd, *service_user)) return err;
        } else if (st.st_uid == geteuid() && fchmod(dir.fd, S_IRWXU) != 0)
            return errno_error("failed to secure state directory");
        if (service_user == nullptr)
            if (auto err = take_state_file(dir.fd, path.filename().string()))
                return err;
    }
    FD file{openat(dir.fd, path.filename().c_str(), O_RDONLY | O_NOFOLLOW | O_CLOEXEC)};
    if (file.fd < 0) {
        if (errno == ENOENT) return x::errors::NIL;
        return errno_error("failed to open state file");
    }
    if (dedicated && service_user != nullptr &&
        (fchown(file.fd, service_user->pw_uid, service_user->pw_gid) != 0 ||
         fchmod(file.fd, S_IRUSR | S_IWUSR) != 0))
        return errno_error("failed to give state file to the service user");
    return x::errors::NIL;
}
#else
/// @brief gives the state directory of the computer, and the files in it, to
/// LocalSystem and the administrators alone. Each user can read a new folder in
/// ProgramData.
x::errors::Error secure_state(const fs::path &path, const bool dedicated) {
    if (!dedicated || !elevated()) return x::errors::NIL;
    auto dir = path.parent_path().wstring();
    std::error_code ec;
    if (!fs::exists(dir, ec)) {
        if (ec)
            return x::errors::Error(
                STATE_ERROR,
                "failed to open state directory: " + ec.message()
            );
        return x::errors::NIL;
    }
    PSECURITY_DESCRIPTOR descriptor = nullptr;
    if (ConvertStringSecurityDescriptorToSecurityDescriptorW(
            L"D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)",
            SDDL_REVISION_1,
            &descriptor,
            nullptr
        ) == 0)
        return x::errors::Error(STATE_ERROR, "failed to secure state directory");
    BOOL present = FALSE;
    BOOL defaulted = FALSE;
    PACL dacl = nullptr;
    DWORD result = ERROR_INVALID_SECURITY_DESCR;
    if (GetSecurityDescriptorDacl(descriptor, &present, &dacl, &defaulted) != 0)
        result = SetNamedSecurityInfoW(
            dir.data(),
            SE_FILE_OBJECT,
            DACL_SECURITY_INFORMATION | PROTECTED_DACL_SECURITY_INFORMATION,
            nullptr,
            nullptr,
            dacl,
            nullptr
        );
    LocalFree(descriptor);
    if (result != ERROR_SUCCESS)
        return x::errors::Error(
            STATE_ERROR,
            "failed to secure state directory: error " + std::to_string(result)
        );
    return x::errors::NIL;
}
#endif

x::errors::Error rack::Config::secure_persisted_state(x::args::Parser &args) {
    return secure_state(
        get_persisted_state_path(args),
        args.field<std::string>("--state-file", "").empty()
    );
}

x::errors::Error rack::Config::move_user_state(x::args::Parser &args) {
    const auto user_dir = user_state_dir();
    if (user_dir.empty() || !args.field<std::string>("--state-file", "").empty())
        return x::errors::NIL;
    const auto from = user_dir / STATE_FILE;
    const auto to = machine_state_dir() / STATE_FILE;
    std::error_code from_ec;
    std::error_code to_ec;
    const bool move = fs::exists(from, from_ec) && !fs::exists(to, to_ec);
    if (from_ec || to_ec)
        return x::errors::Error(
            STATE_ERROR,
            "failed to find the saved login: " + (from_ec ? from_ec : to_ec).message()
        );
    if (!move) return x::errors::NIL;
    LOG(INFO) << "moving the saved login from " << from << " to " << to;
    std::error_code ec;
    fs::create_directories(to.parent_path(), ec);
    if (ec)
        return x::errors::Error(
            STATE_ERROR,
            "failed to create state directory: " + ec.message()
        );
    // Secured before the copy, so the key is never readable by other users.
    if (auto err = secure_state(to, true)) return err;
    fs::copy_file(from, to, ec);
    if (ec)
        return x::errors::Error(
            STATE_ERROR,
            "failed to copy the saved login: " + ec.message()
        );
    if (auto err = secure_state(to, true)) return err;
    fs::remove(from, ec);
    if (ec)
        return x::errors::Error(
            STATE_ERROR,
            "failed to remove the saved login: " + ec.message()
        );
    return x::errors::NIL;
}

std::pair<std::shared_ptr<x::kv::KV>, x::errors::Error>
open_kv(x::args::Parser &parser) {
    if (auto err = rack::Config::secure_persisted_state(parser)) return {nullptr, err};
    auto [kv, err] = x::kv::JSONFile::open_file(
        x::kv::JSONFileConfig{
            .path = get_persisted_state_path(parser),
            .dir_mode = PERSISTED_STATE_DIR_PERMISSIONS,
            .file_mode = PERSISTED_STATE_FILE_PERMISSIONS,
        }
    );
    if (!err) err = rack::Config::secure_persisted_state(parser);
    return {kv, err};
}

x::errors::Error rack::Config::load_persisted_state(x::args::Parser &args) {
    this->state_dir = fs::path(get_persisted_state_path(args)).parent_path();
    auto [kv, open_err] = open_kv(args);
    if (open_err) return open_err;

    // Load the connection config.
    std::string conn = "{}";
    const auto c_err = kv->get("conn_params", conn);
    if (c_err && !x::errors::NOT_FOUND.matches(c_err)) return c_err;
    auto conn_parser = x::json::Parser(conn);
    this->connection.override(conn_parser);

    // Load the cached remote info
    std::string remote_info = "{}";
    const auto r_err = kv->get("remote_info", remote_info);
    if (r_err && !x::errors::NOT_FOUND.matches(r_err)) return r_err;
    auto remote_parser = x::json::Parser(remote_info);
    this->remote_info.override(remote_parser);

    return x::errors::NIL;
}

x::errors::Error rack::Config::save_conn_params(
    x::args::Parser &args,
    const synnax::Config &conn_params
) {
    auto [kv, err] = open_kv(args);
    if (err) return err;
    return kv->set("conn_params", conn_params.to_json().dump());
}

x::errors::Error
rack::Config::save_api_key(x::args::Parser &args, const synnax::Config &conn) {
    auto [kv, err] = open_kv(args);
    if (err) return err;
    auto params = conn.to_json();
    params.erase("username");
    params.erase("password");
    return kv->set("conn_params", params.dump());
}

x::errors::Error
rack::Config::save_remote_info(x::args::Parser &args, const RemoteInfo &remote_info) {
    auto [kv, err] = open_kv(args);
    if (err) return err;
    return kv->set("remote_info", remote_info.to_json().dump());
}

x::errors::Error rack::Config::clear_persisted_state(x::args::Parser &args) {
    auto [kv, err] = open_kv(args);
    if (err) return err;
    if (const auto d1_err = kv->del("conn_params")) return d1_err;
    if (const auto d2_err = kv->del("remote_info")) return d2_err;
    return x::errors::NIL;
}
}
