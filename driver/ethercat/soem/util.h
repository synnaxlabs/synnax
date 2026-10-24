// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <array>
#include <cctype>
#include <string>
#include <string_view>

namespace driver::ethercat::soem {

/// @brief returns true if the interface is a wired physical adapter that can carry
/// EtherCAT. Loopback, wireless, VPN, bridge, and container interfaces return false.
/// @param name the interface name, e.g. "eth0" or a Windows device path.
/// @param description the adapter description, e.g. "Intel(R) Wi-Fi 6 AX201".
inline bool
is_wired_interface(const std::string &name, const std::string &description) {
    constexpr std::array<std::string_view, 14> name_prefixes = {
        "lo",
        "tun",
        "tap",
        "veth",
        "br-",
        "virbr",
        "wl",
        "utun",
        "awdl",
        "llw",
        "bridge",
        "anpi",
        "gif",
        "zt",
    };
    constexpr std::array<std::string_view, 4> name_parts = {
        "localhost",
        "loopback",
        "tailscale",
        "docker",
    };
    constexpr std::array<std::string_view, 9> description_parts = {
        "wi-fi",
        "wireless",
        "802.11",
        "wan miniport",
        "bluetooth",
        "loopback",
        "virtual",
        "vpn",
        "tap-windows",
    };
    const auto lower = [](std::string s) {
        std::ranges::transform(s, s.begin(), [](const unsigned char c) {
            return std::tolower(c);
        });
        return s;
    };
    const auto n = lower(name);
    const auto d = lower(description);
    for (const auto prefix: name_prefixes)
        if (n.starts_with(prefix)) return false;
    for (const auto part: name_parts)
        if (n.find(part) != std::string::npos) return false;
    for (const auto part: description_parts)
        if (d.find(part) != std::string::npos) return false;
    return true;
}

}
