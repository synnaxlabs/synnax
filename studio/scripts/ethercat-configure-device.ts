// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fixtures } from "@/index";

import { configureDeviceScript } from "./common/configure-device";
import { TANK_PRESSURES } from "./common/task";

/**
 * Docs `reference/hardware/ethercat/configure-device` (Configure a device): configure
 * an unconfigured EL3102 module with a name and an identifier. The module is simulated.
 */
export default configureDeviceScript(fixtures.EL3102, TANK_PRESSURES);
