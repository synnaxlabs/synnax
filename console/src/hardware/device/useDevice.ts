// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Form, Synnax, useAsyncEffect } from "@synnaxlabs/pluto";
import { type UnknownRecord } from "@synnaxlabs/x";
import { useCallback, useState } from "react";

export const useDevice = <P extends UnknownRecord>(
  ctx: Form.ContextValue<any>,
): device.Device<P> | undefined => {
  const client = Synnax.use();
  const [device, setDevice] = useState<device.Device<P> | undefined>(undefined);
  useAsyncEffect(async () => {
    if (client == null) return;
    const dev = ctx.value().config.device;
    if (dev === "") return;
    const d = await client.hardware.devices.retrieve<P>(dev);
    setDevice(d);
  }, [client?.key]);
  Form.useFieldListener<string>({
    ctx,
    path: "config.device",
    onChange: useCallback(
      (fs) => {
        if (!fs.touched || fs.status.variant !== "success" || client == null) return;
        client.hardware.devices
          .retrieve<P>(fs.value)
          .then((d) => setDevice(d))
          .catch(console.error);
      },
      [client?.key, setDevice],
    ),
  });
  return device;
};
