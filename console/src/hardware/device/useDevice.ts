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

export const useDevice = <P extends UnknownRecord>(
  ctx: Form.ContextValue<any>,
): device.Device<P> | undefined => {
  const client = Synnax.use();
  const addStatus = Status.useAggregator();
  const [device, setDevice] = useState<device.Device<P> | undefined>(undefined);
  const handleException = useCallback(
    (e: unknown) => {
      if (!(e instanceof Error)) throw e;
      if (NotFoundError.matches(e)) {
        if (device != null) setDevice(undefined);
        return;
      }
      addStatus({
        variant: "error",
        message: `Failed to retrieve ${device?.name ?? "device"}.`,
        description: e.message,
      });
    },
    [addStatus, device?.name, setDevice],
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
      handleException(e);
    }
  }, [ctx, client]);
  Form.useFieldListener<string>({
    ctx,
    path: "config.device",
    onChange: useCallback(
      (fs) => {
        if (!fs.touched || fs.status.variant !== "success" || client == null) return;
        client.hardware.devices
          .retrieve<P>(fs.value)
          .then((d) => setDevice(d))
          .catch(handleException);
      },
      [client, setDevice, handleException],
    ),
  });
  Observe.useListener({
    key: [client, setDevice],
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
