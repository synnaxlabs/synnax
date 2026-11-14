// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cctype>
#include <cstdint>
#include <string>

#include "driver/errors/errors.h"

namespace driver::ethercat::errors {
/// @brief base error type for all EtherCAT-related errors.
const x::errors::Error BASE_ERROR = driver::errors::BASE_ERROR.sub("ethercat");
/// @brief temporary EtherCAT error that is recoverable.
const x::errors::Error TEMPORARY_ERROR = driver::errors::TEMPORARY_HARDWARE_ERROR.sub(
    "ethercat"
);
/// @brief error returned when the EtherCAT master fails to initialize.
const x::errors::Error MASTER_INIT_ERROR = BASE_ERROR.sub("master_init");
/// @brief error returned when a network interface has no EtherCAT slaves.
const x::errors::Error NO_SLAVES_FOUND = MASTER_INIT_ERROR.sub("no_slaves_found");
/// @brief error returned when no slave answers because a filter on the adapter blocks
/// the replies. The user can remove the filter.
const x::errors::Error REPLIES_BLOCKED = NO_SLAVES_FOUND.sub("replies_blocked");
/// @brief error returned when the network interface cannot be opened.
const x::errors::Error INTERFACE_ERROR = BASE_ERROR.sub("interface");
/// @brief error returned when the Driver lacks permission to open the network
/// interface or the IgH master device.
const x::errors::Error PERMISSION_DENIED = INTERFACE_ERROR.sub("permission_denied");
/// @brief error returned when slave configuration fails.
const x::errors::Error SLAVE_CONFIG_ERROR = BASE_ERROR.sub("slave_config");
/// @brief error returned when PDO mapping is invalid or fails.
const x::errors::Error PDO_MAPPING_ERROR = BASE_ERROR.sub("pdo_mapping");
/// @brief error returned when the process image does not hold a PDO entry.
const x::errors::Error PDO_NOT_MAPPED = PDO_MAPPING_ERROR.sub("not_mapped");
/// @brief error returned when master activation fails.
const x::errors::Error ACTIVATION_ERROR = BASE_ERROR.sub("activation");
/// @brief error returned when a slave rejects a startup parameter.
const x::errors::Error STARTUP_PARAMETER_ERROR = BASE_ERROR.sub("startup_parameter");
/// @brief error returned when a slave aborts a startup parameter write with an abort
/// code. The same value gets the same abort, so retrying does not help. A write with no
/// response is a plain STARTUP_PARAMETER_ERROR.
const x::errors::Error STARTUP_PARAMETER_REJECTED = STARTUP_PARAMETER_ERROR.sub(
    "rejected"
);
/// @brief error returned when two open tasks set one object of a slave to different
/// values.
const x::errors::Error SETTING_CONFLICT = BASE_ERROR.sub("setting_conflict");
/// @brief error returned when cyclic communication fails.
const x::errors::Error CYCLIC_ERROR = BASE_ERROR.sub("cyclic");
/// @brief error returned when the working counter does not match expected value.
const x::errors::Error WORKING_COUNTER_ERROR = BASE_ERROR.sub("working_counter");
/// @brief error returned when the cycle time cannot be maintained.
const x::errors::Error CYCLE_OVERRUN = BASE_ERROR.sub("cycle_overrun");
/// @brief error returned when a state machine transition fails.
const x::errors::Error STATE_CHANGE_ERROR = BASE_ERROR.sub("state_change");
/// @brief error returned when a slave refuses a state change with an AL status code
/// that a new start cannot clear. The same configuration gets the same refusal, so
/// retrying does not help.
const x::errors::Error STATE_CHANGE_REFUSED = STATE_CHANGE_ERROR.sub("refused");
/// @brief error returned when the latest bus cycle failed and its inputs were dropped.
const x::errors::Error CYCLE_FAILED = TEMPORARY_ERROR.sub("cycle_failed");
/// @brief error returned when PDO discovery fails for a slave.
const x::errors::Error PDO_DISCOVERY_ERROR = BASE_ERROR.sub("pdo_discovery");
/// @brief error returned when an SDO read operation fails.
const x::errors::Error SDO_READ_ERROR = BASE_ERROR.sub("sdo_read");
/// @brief error returned when engine rate doesn't match requested rate.
const x::errors::Error RATE_MISMATCH = BASE_ERROR.sub("rate_mismatch");
/// @brief error returned when bus topology doesn't match configured devices.
const x::errors::Error TOPOLOGY_MISMATCH = BASE_ERROR.sub("topology_mismatch");

/// @brief returns the CoE meaning of an SDO abort code, or "no response" for 0.
std::string sdo_abort_reason(uint32_t code);

/// @brief why a slave did not complete an SDO transfer.
struct SDOFailure {
    /// @brief how the transfer failed.
    enum class Kind : uint8_t {
        /// @brief no reply came back.
        NO_REPLY,
        /// @brief the slave aborted the transfer with abort_code.
        ABORTED,
        /// @brief the reply was not the SDO response to the request.
        UNEXPECTED_REPLY,
        /// @brief the reply held more data than the request had room for.
        REPLY_TOO_LARGE,
    };

    /// @brief how the transfer failed.
    Kind kind = Kind::NO_REPLY;
    /// @brief the abort code of an ABORTED transfer.
    uint32_t abort_code = 0;

    /// @brief returns why the transfer failed, such as "no response".
    [[nodiscard]] std::string reason() const;

    bool operator==(const SDOFailure &) const = default;
};

/// @brief returns error text as a sentence: the first letter in upper case and a period
/// at the end.
inline std::string sentence(std::string text) {
    if (text.empty()) return text;
    text[0] = static_cast<char>(std::toupper(static_cast<unsigned char>(text[0])));
    if (!text.ends_with('.')) text += '.';
    return text;
}
}
