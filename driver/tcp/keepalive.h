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

namespace driver::tcp::keepalive {
/// @brief IDLE is the silence on a connection before the first probe.
inline const x::telem::TimeSpan IDLE = 10 * x::telem::SECOND;
/// @brief INTERVAL is the wait between unanswered probes.
inline const x::telem::TimeSpan INTERVAL = 5 * x::telem::SECOND;
/// @brief COUNT is the number of unanswered probes after which the connection drops.
constexpr int COUNT = 3;

/// @brief turns on keepalive probes with IDLE, INTERVAL, and COUNT, so a peer that
/// stops answering drops after 25 s. Windows always sends 10 probes, so it drops the
/// peer after 60 s.
/// @returns the error of the first option that the socket refuses.
std::error_code enable(asio::ip::tcp::socket::native_handle_type socket);
}
