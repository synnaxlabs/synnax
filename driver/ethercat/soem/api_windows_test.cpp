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
// clang-format off
// iphlpapi.h must be included after winsock2.h.
#include <winsock2.h>
#include <iphlpapi.h>
// clang-format on

#include <optional>
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "driver/ethercat/soem/api.h"

namespace driver::ethercat::soem {
namespace {
struct Adapter {
    std::string guid;
    std::string interface_name;
    std::string name;
};

std::string to_utf8(const wchar_t *wide) {
    const int
        len = WideCharToMultiByte(CP_UTF8, 0, wide, -1, nullptr, 0, nullptr, nullptr);
    if (len <= 1) return "";
    std::string utf8(static_cast<size_t>(len), '\0');
    WideCharToMultiByte(CP_UTF8, 0, wide, -1, utf8.data(), len, nullptr, nullptr);
    utf8.pop_back();
    return utf8;
}

/// @brief returns the Npcap interface name and the Windows name of the first Ethernet
/// adapter, or std::nullopt when the machine has none.
std::optional<Adapter> first_ethernet_adapter() {
    ULONG size = 16 * 1024;
    std::vector<IP_ADAPTER_ADDRESSES> buf;
    ULONG result = ERROR_BUFFER_OVERFLOW;
    while (result == ERROR_BUFFER_OVERFLOW) {
        buf.resize(size / sizeof(IP_ADAPTER_ADDRESSES) + 1);
        result = GetAdaptersAddresses(AF_UNSPEC, 0, nullptr, buf.data(), &size);
    }
    if (result != NO_ERROR) return std::nullopt;
    for (const auto *a = buf.data(); a != nullptr; a = a->Next)
        if (a->IfType == IF_TYPE_ETHERNET_CSMACD)
            return Adapter{
                .guid = a->AdapterName,
                .interface_name = std::string("\\Device\\NPF_") + a->AdapterName,
                .name = to_utf8(a->FriendlyName),
            };
    return std::nullopt;
}

std::optional<std::wstring> pnp_instance_id(const std::string &guid) {
    const auto path = L"SYSTEM\\CurrentControlSet\\Control\\Network\\{4D36E972-E325-"
                      L"11CE-BFC1-08002BE10318}\\" +
                      std::wstring(guid.begin(), guid.end()) + L"\\Connection";
    wchar_t id[512] = {};
    DWORD size = static_cast<DWORD>(sizeof(id));
    if (RegGetValueW(
            HKEY_LOCAL_MACHINE,
            path.c_str(),
            L"PnPInstanceID",
            RRF_RT_REG_SZ,
            nullptr,
            id,
            &size
        ) != ERROR_SUCCESS)
        return std::nullopt;
    return std::wstring(id);
}
}

/// @brief it should read the name Windows shows for an Ethernet adapter.
TEST(ProdAPIWindows, ReadsTheNameOfAnAdapter) {
    const auto adapter = first_ethernet_adapter();
    if (!adapter.has_value()) GTEST_SKIP() << "no Ethernet adapter";
    ProdAPI api;
    EXPECT_EQ(api.adapter_name(adapter->interface_name.c_str()), adapter->name);
}

/// @brief it should find no filter and no name for an adapter that does not exist.
/// @brief it should find an adapter on USB when Windows enumerates it on USB.
TEST(ProdAPIWindows, ReadsWhetherAnAdapterIsOnUSB) {
    const auto adapter = first_ethernet_adapter();
    if (!adapter.has_value()) GTEST_SKIP() << "no Ethernet adapter";
    const auto id = pnp_instance_id(adapter->guid);
    ASSERT_TRUE(id.has_value());
    ProdAPI api;
    EXPECT_EQ(
        api.on_usb(adapter->interface_name.c_str()),
        id.value().starts_with(L"USB\\")
    );
}

TEST(ProdAPIWindows, FindsNothingForAnUnknownAdapter) {
    ProdAPI api;
    const char *unknown = "\\Device\\NPF_{00000000-0000-0000-0000-000000000000}";
    EXPECT_FALSE(api.dewesoft_filter_bound(unknown));
    EXPECT_EQ(api.adapter_name(unknown), std::nullopt);
    EXPECT_FALSE(api.on_usb(unknown));
    EXPECT_FALSE(api.dewesoft_filter_bound("\\Device\\NPF_Loopback"));
    EXPECT_EQ(api.adapter_name("\\Device\\NPF_Loopback"), std::nullopt);
    EXPECT_FALSE(api.on_usb("\\Device\\NPF_Loopback"));
}
}
