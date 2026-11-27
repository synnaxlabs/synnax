// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TYPED_HARDWARE } from "@/feature/labjack/device/hardware";
import { Device } from "@/platform/device";
import type { Link } from "@/platform/link";

/** Opens the configure form of the typed LabJack device that a link names. */
export const useLink = (): Link.Handler => {
  const configure = Device.useConfigureModal();
  return async ({ client, key }) => {
    const { key: deviceKey } = await client.labjack.devices.retrieve({ key });
    configure({ deviceKey, hardware: TYPED_HARDWARE });
  };
};
