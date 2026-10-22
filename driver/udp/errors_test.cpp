// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <system_error>

#include "asio/error.hpp"
#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/transport/errors.h"
#include "driver/udp/errors.h"

namespace driver::udp {
TEST(Translate, ReportsAMissingNameNetworkOrInterfaceAsUnreachable) {
    for (const std::error_code ec:
         {asio::error::make_error_code(asio::error::host_not_found),
          asio::error::make_error_code(asio::error::host_not_found_try_again),
          asio::error::make_error_code(asio::error::no_data),
          asio::error::make_error_code(asio::error::no_recovery),
          asio::error::make_error_code(asio::error::host_unreachable),
          asio::error::make_error_code(asio::error::network_unreachable),
          asio::error::make_error_code(asio::error::network_down),
          asio::error::make_error_code(asio::error::no_such_device),
          asio::error::make_error_code(asio::error::timed_out),
          asio::error::make_error_code(asio::error::bad_descriptor),
          asio::error::make_error_code(asio::error::operation_aborted),
          std::make_error_code(std::errc::address_not_available)}) {
        SCOPED_TRACE(ec.message());
        ASSERT_OCCURRED_AS(translate("failed", ec), transport::UNREACHABLE_ERROR);
    }
}

TEST(Translate, ReportsAPortInUseOrAForbiddenSocketAsAConfigError) {
    for (const std::error_code ec:
         {asio::error::make_error_code(asio::error::address_in_use),
          asio::error::make_error_code(asio::error::access_denied),
          asio::error::make_error_code(asio::error::no_permission),
          asio::error::make_error_code(asio::error::invalid_argument),
          asio::error::make_error_code(asio::error::address_family_not_supported),
          asio::error::make_error_code(asio::error::message_size),
          asio::error::make_error_code(asio::error::no_protocol_option),
          asio::error::make_error_code(asio::error::operation_not_supported)}) {
        SCOPED_TRACE(ec.message());
        ASSERT_OCCURRED_AS(translate("failed", ec), transport::CONFIG_ERROR);
    }
}

TEST(Translate, ReportsAnExhaustedHostOrAnUnknownErrorAsCritical) {
    for (const std::error_code ec:
         {asio::error::make_error_code(asio::error::no_descriptors),
          asio::error::make_error_code(asio::error::no_memory),
          asio::error::make_error_code(asio::error::no_buffer_space),
          std::make_error_code(std::errc::too_many_files_open_in_system),
          std::make_error_code(std::errc::state_not_recoverable)}) {
        SCOPED_TRACE(ec.message());
        ASSERT_OCCURRED_AS(translate("failed", ec), transport::CRITICAL_ERROR);
    }
}

TEST(Translate, PutsTheContextBeforeTheSystemMessage) {
    const auto ec = asio::error::make_error_code(asio::error::address_in_use);
    EXPECT_EQ(
        translate("failed to bind UDP port 5000", ec).data,
        "failed to bind UDP port 5000: " + ec.message()
    );
}
}
