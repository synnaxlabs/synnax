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

namespace ethercat {
/// Base error type for all EtherCAT-related errors.
const xerrors::Error BASE_ERROR = driver::BASE_ERROR.sub("ethercat");

/// Error returned when the EtherCAT master fails to initialize.
const xerrors::Error MASTER_INIT_ERROR = BASE_ERROR.sub("master_init");

/// Error returned when the network interface cannot be opened.
const xerrors::Error INTERFACE_ERROR = BASE_ERROR.sub("interface");

/// Error returned when slave configuration fails.
const xerrors::Error SLAVE_CONFIG_ERROR = BASE_ERROR.sub("slave_config");

/// Error returned when PDO mapping is invalid or fails.
const xerrors::Error PDO_MAPPING_ERROR = BASE_ERROR.sub("pdo_mapping");

/// Error returned when domain creation or registration fails.
const xerrors::Error DOMAIN_ERROR = BASE_ERROR.sub("domain");

/// Error returned when master activation fails.
const xerrors::Error ACTIVATION_ERROR = BASE_ERROR.sub("activation");

/// Error returned when cyclic communication fails.
const xerrors::Error CYCLIC_ERROR = BASE_ERROR.sub("cyclic");

/// Error returned when a slave enters an unexpected state.
const xerrors::Error SLAVE_STATE_ERROR = BASE_ERROR.sub("slave_state");

/// Error returned when a slave is not found at the expected position.
const xerrors::Error SLAVE_NOT_FOUND = BASE_ERROR.sub("slave_not_found");

/// Error returned when a slave is disconnected.
const xerrors::Error SLAVE_DISCONNECTED = BASE_ERROR.sub("slave_disconnected");

/// Error returned when the working counter does not match expected value.
const xerrors::Error WORKING_COUNTER_ERROR = BASE_ERROR.sub("working_counter");

/// Error returned when the cycle time cannot be maintained.
const xerrors::Error CYCLE_OVERRUN = BASE_ERROR.sub("cycle_overrun");

/// Error returned when a state machine transition fails.
const xerrors::Error STATE_CHANGE_ERROR = BASE_ERROR.sub("state_change");

/// Error returned when the cyclic engine is restarting for reconfiguration.
const xerrors::Error ENGINE_RESTARTING = BASE_ERROR.sub("engine_restarting");

/// Error returned when PDO discovery fails for a slave.
const xerrors::Error PDO_DISCOVERY_ERROR = BASE_ERROR.sub("pdo_discovery");

/// Error returned when an SDO read operation fails.
const xerrors::Error SDO_READ_ERROR = BASE_ERROR.sub("sdo_read");
}
