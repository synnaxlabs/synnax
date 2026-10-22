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

#include "asio/ip/tcp.hpp"

#include "x/cpp/telem/telem.h"

/// @brief TCP keepalive probes, which find a peer that vanished without closing the
/// connection. Each platform sets them in its own file.
namespace driver::tcp::keepalive {
/// @brief the silence on a connection before the first probe.
inline const x::telem::TimeSpan IDLE = 10 * x::telem::SECOND;
/// @brief the wait between unanswered probes.
inline const x::telem::TimeSpan INTERVAL = 5 * x::telem::SECOND;
/// @brief the number of unanswered probes after which the connection drops.
constexpr int COUNT = 3;

/// @brief turns on keepalive probes with IDLE, INTERVAL, and COUNT.
/// @returns the error from the first option that the socket rejects.
std::error_code enable(asio::ip::tcp::socket::native_handle_type socket);
}
