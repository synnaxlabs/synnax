// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.
import { NI_USB_6343 } from "@/fixtures/hardware";

import { configureDeviceScript } from "./common/configure-device";

/**
 * Docs `reference/hardware/ni/configure-device` (Configure a device): configure an
 * unconfigured NI USB-6343 with a name and an identifier. The device is simulated.
 */
export default configureDeviceScript({
  device: NI_USB_6343,
  name: "Test stand DAQ",
  identifier: "dev1",
});
