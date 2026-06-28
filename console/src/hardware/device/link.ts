// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback } from "react";

import { CONFIGURE_MODALS, getMake } from "@/hardware/device/make";
import { type Link } from "@/layered/service/link";
import { Modals } from "@/layered/service/modals";

export const useLink = (): Link.Handler => {
  const placeModal = Modals.use();
  return useCallback(
    async ({ client, key }) => {
      const device = await client.devices.retrieve({ key });
      const make = getMake(device.make);
      if (make == null) return;
      placeModal.open(CONFIGURE_MODALS[make], {
        deviceKey: device.key,
        title: device.name,
      });
    },
    [placeModal],
  );
};
