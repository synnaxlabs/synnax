// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <system_error>

#include "asio/ip/udp.hpp"

namespace driver::udp::icmp {
/// @brief stops socket from reporting an ICMP reply to an earlier send as the error of
/// a later send or receive, as Windows does. Linux and macOS report these replies only
/// to connected sockets.
/// @returns the error of the option that the socket refuses.
std::error_code ignore(asio::ip::udp::socket::native_handle_type socket);
}
