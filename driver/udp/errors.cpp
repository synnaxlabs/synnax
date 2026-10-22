// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <system_error>

#include "asio/error.hpp"

#include "driver/transport/errors.h"
#include "driver/udp/errors.h"

namespace driver::udp {
x::errors::Error
translate(const std::string &context, const std::error_code &reported) {
    using transport::any_of;
    if (reported == asio::error::host_not_found ||
        reported == asio::error::host_not_found_try_again ||
        reported == asio::error::no_data || reported == asio::error::no_recovery)
        return transport::error(transport::UNREACHABLE_ERROR, context, reported);
    const auto ec = transport::from_asio(reported);
    if (any_of(
            ec,
            {std::errc::host_unreachable,
             std::errc::network_unreachable,
             std::errc::network_down,
             std::errc::address_not_available,
             std::errc::no_such_device,
             std::errc::timed_out,
             std::errc::bad_file_descriptor,
             std::errc::operation_canceled}
        ))
        return transport::error(transport::UNREACHABLE_ERROR, context, ec);
    if (any_of(
            ec,
            {std::errc::address_in_use,
             std::errc::permission_denied,
             std::errc::operation_not_permitted,
             std::errc::invalid_argument,
             std::errc::address_family_not_supported,
             std::errc::message_size,
             std::errc::no_protocol_option,
             std::errc::operation_not_supported}
        ))
        return transport::error(transport::CONFIG_ERROR, context, ec);
    return transport::error(transport::CRITICAL_ERROR, context, ec);
}
}
