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
#include <filesystem>
#include <optional>
#include <string>
#include <system_error>
#include <utility>
#include <vector>

// clang-format off
// devguid.h and setupapi.h must be included after windows.h, and initguid.h makes
// devguid.h define its GUIDs.
#include <windows.h>
#include <initguid.h>
#include <devguid.h>
#include <setupapi.h>
// clang-format on

#include "driver/serial/errors.h"
#include "driver/serial/list.h"

namespace driver::serial {
namespace {
/// @returns s as UTF-8.
std::string utf8(const std::wstring &s) {
    const auto u8 = std::filesystem::path(s).u8string();
    return {u8.begin(), u8.end()};
}

/// @returns the string registry value name of key, or nothing when it has none.
std::optional<std::wstring> value(HKEY key, const wchar_t *name) {
    std::array<wchar_t, 256> buf{};
    DWORD size = sizeof(buf) - sizeof(wchar_t);
    DWORD type = 0;
    if (RegQueryValueExW(
            key,
            name,
            nullptr,
            &type,
            reinterpret_cast<BYTE *>(buf.data()),
            &size
        ) != ERROR_SUCCESS ||
        type != REG_SZ)
        return std::nullopt;
    return std::wstring(buf.data());
}

/// @returns the COM name of the port data names in set, or nothing when it has none.
std::optional<std::wstring> port_name(HDEVINFO set, SP_DEVINFO_DATA &data) {
    HKEY key = SetupDiOpenDevRegKey(
        set,
        &data,
        DICS_FLAG_GLOBAL,
        0,
        DIREG_DEV,
        KEY_READ
    );
    if (key == INVALID_HANDLE_VALUE) return std::nullopt;
    auto name = value(key, L"PortName");
    RegCloseKey(key);
    if (!name || name->starts_with(L"LPT")) return std::nullopt;
    return name;
}

/// @returns the friendly name of the device data names in set, or nothing when it has
/// none.
std::optional<std::string> friendly_name(HDEVINFO set, SP_DEVINFO_DATA &data) {
    std::array<wchar_t, 256> buf{};
    if (!SetupDiGetDeviceRegistryPropertyW(
            set,
            &data,
            SPDRP_FRIENDLYNAME,
            nullptr,
            reinterpret_cast<BYTE *>(buf.data()),
            sizeof(buf) - sizeof(wchar_t),
            nullptr
        ) ||
        buf[0] == L'\0')
        return std::nullopt;
    return utf8(buf.data());
}
}

std::pair<std::vector<Entry>, x::errors::Error> list() {
    std::vector<Entry> entries;
    for (const auto &guid: {GUID_DEVCLASS_PORTS, GUID_DEVCLASS_MODEM}) {
        HDEVINFO set = SetupDiGetClassDevsW(&guid, nullptr, nullptr, DIGCF_PRESENT);
        if (set == INVALID_HANDLE_VALUE)
            return {
                {},
                translate(
                    "failed to list serial ports",
                    std::error_code(
                        static_cast<int>(GetLastError()),
                        std::system_category()
                    )
                )
            };
        SP_DEVINFO_DATA data{.cbSize = sizeof(SP_DEVINFO_DATA)};
        for (DWORD i = 0; SetupDiEnumDeviceInfo(set, i, &data); i++)
            if (const auto name = port_name(set, data))
                entries.push_back({
                    .path = utf8(*name),
                    .description = friendly_name(set, data),
                });
        SetupDiDestroyDeviceInfoList(set);
    }
    std::ranges::sort(entries, {}, &Entry::path);
    return {std::move(entries), x::errors::NIL};
}
}
