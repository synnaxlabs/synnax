// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdlib>
#include <filesystem>
#include <string>
#include <system_error>

#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
// clang-format off
// aclapi.h and sddl.h must be included after windows.h.
#include <windows.h>
#include <aclapi.h>
#include <sddl.h>
// clang-format on

#include "x/cpp/defer/defer.h"

#include "driver/rack/rack.h"
#include "driver/rack/state.h"

namespace fs = std::filesystem;

namespace driver::rack::state {
fs::path machine_dir() {
    return "C:\\ProgramData\\synnax-driver";
}

fs::path user_dir() {
    if (const char *appdata = std::getenv("LOCALAPPDATA"))
        return fs::path(appdata) / "synnax-driver";
    return {};
}

namespace {
/// @brief the administrators own the directory. SYSTEM and the administrators have full
/// access to it and everything in it, and no one else has access.
constexpr auto DESCRIPTOR = L"O:BAD:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)";

x::errors::Error state_error(const std::string &action, const DWORD code) {
    return x::errors::Error(
        STATE_ERROR,
        action + ": " + std::system_category().message(static_cast<int>(code))
    );
}

/// @brief returns an error unless SYSTEM or the administrators own dir itself, and not
/// only the target of a link. Any other owner can give itself access again, and can
/// leave links or files in dir.
x::errors::Error check_owner(const std::wstring &dir) {
    const HANDLE handle = CreateFileW(
        dir.c_str(),
        READ_CONTROL,
        FILE_SHARE_READ | FILE_SHARE_WRITE | FILE_SHARE_DELETE,
        nullptr,
        OPEN_EXISTING,
        FILE_FLAG_BACKUP_SEMANTICS | FILE_FLAG_OPEN_REPARSE_POINT,
        nullptr
    );
    if (handle == INVALID_HANDLE_VALUE)
        return state_error("failed to open state directory", GetLastError());
    PSID owner = nullptr;
    PSECURITY_DESCRIPTOR descriptor = nullptr;
    const DWORD result = GetSecurityInfo(
        handle,
        SE_FILE_OBJECT,
        OWNER_SECURITY_INFORMATION,
        &owner,
        nullptr,
        nullptr,
        nullptr,
        &descriptor
    );
    CloseHandle(handle);
    if (result != ERROR_SUCCESS)
        return state_error("failed to read the owner of the state directory", result);
    const bool trusted = IsWellKnownSid(owner, WinLocalSystemSid) != 0 ||
                         IsWellKnownSid(owner, WinBuiltinAdministratorsSid) != 0;
    LocalFree(descriptor);
    if (trusted) return x::errors::NIL;
    return x::errors::Error(
        STATE_ERROR,
        "another user owns " + fs::path(dir).string() +
            ". Delete it, then install the Driver again"
    );
}
}

/// @brief creates the state directory of the computer for LocalSystem and the
/// administrators alone, or gives an existing one to them alone. Each user can create
/// and read a new folder in ProgramData.
x::errors::Error secure(const fs::path &path, const bool dedicated) {
    if (!dedicated || path.parent_path() != machine_dir()) return x::errors::NIL;
    auto dir = path.parent_path().wstring();
    PSECURITY_DESCRIPTOR descriptor = nullptr;
    if (ConvertStringSecurityDescriptorToSecurityDescriptorW(
            DESCRIPTOR,
            SDDL_REVISION_1,
            &descriptor,
            nullptr
        ) == 0)
        return state_error("failed to secure state directory", GetLastError());
    x::defer::defer free_descriptor([&] { LocalFree(descriptor); });
    // Created with its access in one step, so no user can add a file to it first.
    SECURITY_ATTRIBUTES attributes{sizeof(attributes), descriptor, FALSE};
    if (CreateDirectoryW(dir.c_str(), &attributes) == 0)
        if (const DWORD code = GetLastError(); code != ERROR_ALREADY_EXISTS)
            return state_error("failed to create state directory", code);
    if (auto err = check_owner(dir)) return err;
    BOOL present = FALSE;
    BOOL defaulted = FALSE;
    PACL dacl = nullptr;
    if (GetSecurityDescriptorDacl(descriptor, &present, &dacl, &defaulted) == 0)
        return state_error("failed to secure state directory", GetLastError());
    const DWORD result = SetNamedSecurityInfoW(
        dir.data(),
        SE_FILE_OBJECT,
        DACL_SECURITY_INFORMATION | PROTECTED_DACL_SECURITY_INFORMATION,
        nullptr,
        nullptr,
        dacl,
        nullptr
    );
    if (result != ERROR_SUCCESS)
        return state_error("failed to secure state directory", result);
    return x::errors::NIL;
}
}
