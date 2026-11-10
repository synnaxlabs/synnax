// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cerrno>
#include <system_error>

#include "asio/error.hpp"
#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/tcp/errors.h"
#include "driver/transport/errors.h"

namespace driver::tcp {
TEST(Translate, ReportsAGoneOrMissingPeerAsUnreachable) {
    for (const std::error_code ec:
         {asio::error::make_error_code(asio::error::connection_refused),
          asio::error::make_error_code(asio::error::connection_reset),
          asio::error::make_error_code(asio::error::connection_aborted),
          asio::error::make_error_code(asio::error::not_connected),
          asio::error::make_error_code(asio::error::broken_pipe),
          asio::error::make_error_code(asio::error::timed_out),
          asio::error::make_error_code(asio::error::host_unreachable),
          asio::error::make_error_code(asio::error::network_unreachable),
          asio::error::make_error_code(asio::error::network_down),
          asio::error::make_error_code(asio::error::network_reset),
          asio::error::make_error_code(asio::error::bad_descriptor),
          asio::error::make_error_code(asio::error::operation_aborted),
          asio::error::make_error_code(asio::error::eof),
          asio::error::make_error_code(asio::error::host_not_found),
          asio::error::make_error_code(asio::error::host_not_found_try_again),
          asio::error::make_error_code(asio::error::no_data),
          asio::error::make_error_code(asio::error::no_recovery),
          std::make_error_code(std::errc::timed_out)}) {
        SCOPED_TRACE(ec.message());
        ASSERT_OCCURRED_AS(translate("failed", ec), transport::UNREACHABLE_ERROR);
    }
}

TEST(Translate, ReportsAConnectionTheHostForbidsAsAConfigError) {
    for (const std::error_code ec:
         {asio::error::make_error_code(asio::error::access_denied),
          asio::error::make_error_code(asio::error::no_permission),
          asio::error::make_error_code(asio::error::invalid_argument),
          asio::error::make_error_code(asio::error::address_family_not_supported),
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
    const auto ec = asio::error::make_error_code(asio::error::connection_refused);
    EXPECT_EQ(
        translate("failed to connect to 10.0.0.2:502", ec).data,
        "failed to connect to 10.0.0.2:502: " + ec.message()
    );
}
}
