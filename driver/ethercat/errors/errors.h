// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

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
/// @brief error returned when the network interface cannot be opened.
const x::errors::Error INTERFACE_ERROR = BASE_ERROR.sub("interface");
/// @brief error returned when slave configuration fails.
const x::errors::Error SLAVE_CONFIG_ERROR = BASE_ERROR.sub("slave_config");
/// @brief error returned when PDO mapping is invalid or fails.
const x::errors::Error PDO_MAPPING_ERROR = BASE_ERROR.sub("pdo_mapping");
/// @brief error returned when domain creation or registration fails.
const x::errors::Error DOMAIN_ERROR = BASE_ERROR.sub("domain");
/// @brief error returned when master activation fails.
const x::errors::Error ACTIVATION_ERROR = BASE_ERROR.sub("activation");
/// @brief error returned when cyclic communication fails.
const x::errors::Error CYCLIC_ERROR = BASE_ERROR.sub("cyclic");
/// @brief error returned when a slave enters an unexpected state.
const x::errors::Error SLAVE_STATE_ERROR = BASE_ERROR.sub("slave_state");
/// @brief error returned when a slave is not found at the expected position.
const x::errors::Error SLAVE_NOT_FOUND = BASE_ERROR.sub("slave_not_found");
/// @brief error returned when a slave is disconnected.
const x::errors::Error SLAVE_DISCONNECTED = BASE_ERROR.sub("slave_disconnected");
/// @brief error returned when the working counter does not match expected value.
const x::errors::Error WORKING_COUNTER_ERROR = BASE_ERROR.sub("working_counter");
/// @brief error returned when the cycle time cannot be maintained.
const x::errors::Error CYCLE_OVERRUN = BASE_ERROR.sub("cycle_overrun");
/// @brief error returned when a state machine transition fails.
const x::errors::Error STATE_CHANGE_ERROR = BASE_ERROR.sub("state_change");
/// @brief error returned when the cyclic engine is restarting for reconfiguration.
const x::errors::Error ENGINE_RESTARTING = TEMPORARY_ERROR.sub("engine_restarting");
/// @brief error returned when PDO discovery fails for a slave.
const x::errors::Error PDO_DISCOVERY_ERROR = BASE_ERROR.sub("pdo_discovery");
/// @brief error returned when an SDO read operation fails.
const x::errors::Error SDO_READ_ERROR = BASE_ERROR.sub("sdo_read");
/// @brief error returned when engine rate doesn't match requested rate.
const x::errors::Error RATE_MISMATCH = BASE_ERROR.sub("rate_mismatch");
/// @brief error returned when bus topology doesn't match configured devices.
const x::errors::Error TOPOLOGY_MISMATCH = BASE_ERROR.sub("topology_mismatch");
}
