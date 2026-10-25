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
 * Docs `client/resources/build-device-driver/console-setup`: bind a switch to the
 * command channel and to the state channel that the device echoes, take control, and
 * switch it on.
 */
export default arduinoSwitchScript((fixture) => fixture.state);
