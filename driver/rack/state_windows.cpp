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

#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
// clang-format off
// aclapi.h and sddl.h must be included after windows.h.
#include <windows.h>
#include <aclapi.h>
#include <sddl.h>
// clang-format on

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

/// @brief gives the state directory of the computer, and the files in it, to
/// LocalSystem and the administrators alone. Each user can read a new folder in
/// ProgramData.
x::errors::Error secure(const fs::path &path, const bool dedicated) {
    if (!dedicated || path.parent_path() != machine_dir()) return x::errors::NIL;
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
}
