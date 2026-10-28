// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arduinoSwitchScript } from "./common/arduino-switch";

/**
 * Docs `client/resources/build-device-driver/schematic`: bind a switch to the virtual
 * command channel for both its command and its state, take control, and switch it on.
 */
export default arduinoSwitchScript((fixture) => fixture.command);
