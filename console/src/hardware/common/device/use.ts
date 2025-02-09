// Copyright 2025 Synnax Labs, Inc.
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
import { type z } from "zod";

interface UseContextValue
  extends z.ZodObject<{
    config: z.ZodObject<{ device: z.ZodString }>;
  }> {}

/**
 * @description A hook that retrieves and subscribes to updates for a device. Must be
 * used within a Form context that has a schema matching the following structure:
 *
 * ```typescript
 * {
 *   config: {
 *     device: z.string() // The device key
 *   }
 * }
 * ```
 *
 * @returns The device if found, undefined otherwise.
 * @template P - The type of the device properties.
 * @template MK - The device make type.
 * @template MO - The device model type.
 */
export const use = <
  Properties extends UnknownRecord = UnknownRecord,
  Make extends string = string,
  Model extends string = string,
>() => {
  const ctx = Form.useContext<UseContextValue>();
  const client = Synnax.use();
  const handleException = Status.useExceptionHandler();
  const [device, setDevice] = useState<device.Device<Properties, Make, Model>>();
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
    if (deviceKey === "") {
      setDevice(undefined);
      return;
    }
    try {
      const device = await client.hardware.devices.retrieve<Properties, Make, Model>(
        deviceKey,
      );
      setDevice(device);
    } catch (e) {
      handleExc(e);
    }
  }, [ctx, client?.key]);
  Form.useFieldListener<string, UseContextValue>({
    ctx,
    path: "config.device",
    onChange: useCallback(
      (fs) => {
        if (!fs.touched || fs.status.variant !== "success" || client == null) return;
        client.hardware.devices
          .retrieve<Properties, Make, Model>(fs.value)
          .then(setDevice)
          .catch(handleExc);
      },
      [client?.key, setDevice, handleExc],
    ),
  });
  Observe.useListener({
    key: [client?.key, setDevice, device?.key],
    open: async () => await client?.hardware.devices.openDeviceTracker(),
    onChange: (changes) => {
      for (const change of changes) {
        if (change.key !== device?.key) continue;
        if (change.variant === "set")
          setDevice(change.value as device.Device<Properties, Make, Model>);
        else setDevice(undefined);
      }
    },
  });
  return device;
};
