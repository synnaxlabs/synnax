// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
#include <cwchar>
#include <optional>
#include <string>
#include <string_view>

#include <windows.h>

#include "driver/ethercat/soem/api.h"
#include "driver/ethercat/soem/util.h"

namespace driver::ethercat::soem {
namespace {
/// @brief Windows keeps one key under this path for each adapter that the DewesoftX
/// filter is bound to.
constexpr std::wstring_view DEWESOFT_FILTER_ADAPTERS =
    L"SYSTEM\\CurrentControlSet\\Services\\EthcatDAQ\\Parameters\\Adapters\\";
/// @brief the key of the network adapter class, which holds the name of each adapter.
constexpr std::wstring_view NETWORK_ADAPTERS =
    L"SYSTEM\\CurrentControlSet\\Control\\Network\\{4D36E972-E325-11CE-BFC1-"
    L"08002BE10318}\\";

std::optional<std::wstring> wide_adapter_guid(const char *ifname) {
    const auto guid = adapter_guid(ifname);
    if (!guid) return std::nullopt;
    return std::wstring(guid->begin(), guid->end());
}

std::optional<std::string> to_utf8(const std::wstring &wide) {
    const int wide_len = static_cast<int>(wide.size());
    const int len = WideCharToMultiByte(
        CP_UTF8,
        0,
        wide.data(),
        wide_len,
        nullptr,
        0,
        nullptr,
        nullptr
    );
    if (len <= 0) return std::nullopt;
    std::string utf8(static_cast<size_t>(len), '\0');
    if (WideCharToMultiByte(
            CP_UTF8,
            0,
            wide.data(),
            wide_len,
            utf8.data(),
            len,
            nullptr,
            nullptr
        ) != len)
        return std::nullopt;
    return utf8;
}

/// @brief reads a string value of the Connection key of the adapter of ifname, or
/// returns std::nullopt when it is missing or empty.
std::optional<std::wstring> connection_value(const char *ifname, const wchar_t *value) {
    const auto guid = wide_adapter_guid(ifname);
    if (!guid) return std::nullopt;
    const auto path = std::wstring(NETWORK_ADAPTERS) + *guid + L"\\Connection";
    DWORD size = 0;
    if (RegGetValueW(
            HKEY_LOCAL_MACHINE,
            path.c_str(),
            value,
            RRF_RT_REG_SZ,
            nullptr,
            nullptr,
            &size
        ) != ERROR_SUCCESS)
        return std::nullopt;
    std::wstring data(size / sizeof(wchar_t), L'\0');
    if (RegGetValueW(
            HKEY_LOCAL_MACHINE,
            path.c_str(),
            value,
            RRF_RT_REG_SZ,
            nullptr,
            data.data(),
            &size
        ) != ERROR_SUCCESS)
        return std::nullopt;
    data.resize(std::wcslen(data.c_str()));
    if (data.empty()) return std::nullopt;
    return data;
}
}

x::errors::Error ProdAPI::check_socket(const char *) {
    return x::errors::NIL;
}

bool ProdAPI::dewesoft_filter_bound(const char *ifname) {
    const auto guid = wide_adapter_guid(ifname);
    if (!guid) return false;
    const auto path = std::wstring(DEWESOFT_FILTER_ADAPTERS) + *guid;
    HKEY key = nullptr;
    if (RegOpenKeyExW(HKEY_LOCAL_MACHINE, path.c_str(), 0, KEY_READ, &key) !=
        ERROR_SUCCESS)
        return false;
    RegCloseKey(key);
    return true;
}

std::optional<std::string> ProdAPI::adapter_name(const char *ifname) {
    const auto name = connection_value(ifname, L"Name");
    if (!name) return std::nullopt;
    return to_utf8(*name);
}

bool ProdAPI::on_usb(const char *ifname) {
    const auto id = connection_value(ifname, L"PnPInstanceID");
    return id && id->starts_with(L"USB\\");
}
}
