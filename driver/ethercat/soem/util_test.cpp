// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "driver/ethercat/soem/util.h"

namespace driver::ethercat::soem {

/// @brief it should accept wired Ethernet adapters by name or by description.
TEST(IsWiredInterface, AcceptsWiredAdapters) {
    ASSERT_TRUE(is_wired_interface("eth0", ""));
    ASSERT_TRUE(is_wired_interface("enp3s0", ""));
    ASSERT_TRUE(is_wired_interface("eno1", ""));
    ASSERT_TRUE(is_wired_interface("en7", "USB 10/100/1000 LAN"));
    ASSERT_TRUE(is_wired_interface(
        "\\Device\\NPF_{1A2B}",
        "Intel(R) Ethernet Connection I219-LM"
    ));
}

/// @brief it should reject loopback interfaces.
TEST(IsWiredInterface, RejectsLoopback) {
    ASSERT_FALSE(is_wired_interface("lo", ""));
    ASSERT_FALSE(is_wired_interface("lo0", ""));
    ASSERT_FALSE(is_wired_interface("localhost", ""));
    ASSERT_FALSE(is_wired_interface("\\Device\\NPF_Loopback", "Adapter for loopback"));
}

/// @brief it should reject wireless adapters by name or by description.
TEST(IsWiredInterface, RejectsWirelessAdapters) {
    ASSERT_FALSE(is_wired_interface("wlan0", ""));
    ASSERT_FALSE(is_wired_interface("wlp2s0", ""));
    ASSERT_FALSE(is_wired_interface("\\Device\\NPF_{3C4D}", "Intel(R) Wi-Fi 6 AX201"));
    ASSERT_FALSE(
        is_wired_interface("\\Device\\NPF_{5E6F}", "Realtek 8822CE Wireless LAN")
    );
}

/// @brief it should reject virtual, bridge, VPN, and container interfaces.
TEST(IsWiredInterface, RejectsVirtualInterfaces) {
    ASSERT_FALSE(is_wired_interface("veth1234", ""));
    ASSERT_FALSE(is_wired_interface("docker0", ""));
    ASSERT_FALSE(is_wired_interface("br-abcdef", ""));
    ASSERT_FALSE(is_wired_interface("virbr0", ""));
    ASSERT_FALSE(is_wired_interface("tun0", ""));
    ASSERT_FALSE(is_wired_interface("tap0", ""));
    ASSERT_FALSE(is_wired_interface("tailscale0", ""));
    ASSERT_FALSE(is_wired_interface("utun3", ""));
    ASSERT_FALSE(is_wired_interface("awdl0", ""));
    ASSERT_FALSE(is_wired_interface("bridge0", ""));
    ASSERT_FALSE(
        is_wired_interface("\\Device\\NPF_{7A8B}", "Hyper-V Virtual Ethernet Adapter")
    );
    ASSERT_FALSE(is_wired_interface("\\Device\\NPF_{9C0D}", "WAN Miniport (IP)"));
}

}
