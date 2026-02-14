// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <gtest/gtest.h>

#include "driver/ethercat/soem/util.h"

namespace driver::ethercat::soem {

TEST(IsPhysicalInterface, AcceptsPhysicalInterfaces) {
    ASSERT_TRUE(is_physical_interface("eth0"));
    ASSERT_TRUE(is_physical_interface("enp3s0"));
    ASSERT_TRUE(is_physical_interface("eno1"));
    ASSERT_TRUE(is_physical_interface("wlan0"));
}

TEST(IsPhysicalInterface, RejectsLoopback) {
    ASSERT_FALSE(is_physical_interface("lo"));
    ASSERT_FALSE(is_physical_interface("localhost"));
}

TEST(IsPhysicalInterface, RejectsVirtualInterfaces) {
    ASSERT_FALSE(is_physical_interface("veth1234"));
    ASSERT_FALSE(is_physical_interface("docker0"));
    ASSERT_FALSE(is_physical_interface("br-abcdef"));
    ASSERT_FALSE(is_physical_interface("virbr0"));
    ASSERT_FALSE(is_physical_interface("tun0"));
    ASSERT_FALSE(is_physical_interface("tap0"));
    ASSERT_FALSE(is_physical_interface("tailscale0"));
}

}
