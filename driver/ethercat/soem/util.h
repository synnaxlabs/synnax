// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>

namespace driver::ethercat::soem {

/// @brief returns true if the network interface name represents a physical adapter
/// (not loopback, VPN, virtual bridge, or container interface).
inline bool is_physical_interface(const std::string &name) {
    if (name == "lo" || name == "localhost") return false;
    if (name.contains("tailscale")) return false;
    if (name.starts_with("tun")) return false;
    if (name.starts_with("tap")) return false;
    if (name.starts_with("veth")) return false;
    if (name.contains("docker")) return false;
    if (name.starts_with("br-")) return false;
    if (name.starts_with("virbr")) return false;
    return true;
}

}
