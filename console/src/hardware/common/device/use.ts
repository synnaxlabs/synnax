// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, NotFoundError } from "@synnaxlabs/client";
import { Form, Observe, Status, Synnax, useAsyncEffect } from "@synnaxlabs/pluto";
import { type UnknownRecord } from "@synnaxlabs/x";
import { useCallback, useState } from "react";

export const use = <P extends UnknownRecord>(
  ctx: Form.ContextValue<any>,
): device.Device<P> | undefined => {
  const client = Synnax.use();
  const handleException = Status.useExceptionHandler();
  const [device, setDevice] = useState<device.Device<P> | undefined>(undefined);
  const handleExc = useCallback(
    (e: unknown) => {
      if (NotFoundError.matches(e)) {
        setDevice(undefined);
        return;
      }
      handleException(e, `Failed to retrieve ${device?.name ?? "device"}`);
    },
    [handleException, device?.name, setDevice],
  );
  useAsyncEffect(async () => {
    if (client == null) return;
    const deviceKey = ctx.value().config.device;
    if (typeof deviceKey !== "string") return;
    if (deviceKey === "") return;
    try {
      const d = await client.hardware.devices.retrieve<P>(deviceKey);
      setDevice(d);
    } catch (e) {
      handleExc(e);
    }
  }, [ctx, client]);
  Form.useFieldListener<string>({
    ctx,
    path: "config.device",
    onChange: useCallback(
      async (fs) => {
        if (!fs.touched || fs.status.variant !== "success" || client == null) return;
        try {
          const device = await client.hardware.devices.retrieve<P>(fs.value);
          setDevice(device);
        } catch (e) {
          handleExc(e);
        }
      },
      [client, setDevice, handleExc],
    ),
  });
  Observe.useListener({
    key: [client, setDevice, device?.key],
    open: async () => await client?.hardware.devices.openDeviceTracker(),
    onChange: (changes) => {
      for (const change of changes) {
        if (change.key !== device?.key) continue;
        if (change.variant === "set") setDevice(change.value as device.Device<P>);
        else if (change.variant === "delete") setDevice(undefined);
      }
    },
  });
  return device;
};
