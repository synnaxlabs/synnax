// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { HARDWARE } from "@/feature/labjack/device/hardware";
import { Device } from "@/platform/device";

/** Returns an opener for the configure form of a LabJack device. */
export const useConfigure = () => {
  const configure = Device.useConfigureModal();
  return ({ deviceKey }: Device.ConfigureParams) =>
    configure({ deviceKey, hardware: HARDWARE });
};
