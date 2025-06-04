// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, NotFoundError } from "@synnaxlabs/client";
import {
  Device,
  Form,
  Status,
  Synnax,
  useAsyncEffect,
  useSyncedRef,
} from "@synnaxlabs/pluto";
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
  StateDetails extends {} = UnknownRecord,
>(): device.Device<Properties, Make, Model, StateDetails> | undefined => {
  const ctx = Form.useContext<UseContextValue>();
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const [dev, setDev] =
    useState<device.Device<Properties, Make, Model, StateDetails>>();
  const deviceNameRef = useSyncedRef(dev?.name);
  const handleExc = useCallback(
    (e: unknown) => {
      if (NotFoundError.matches(e)) {
        setDev(undefined);
        return;
      }
      handleError(e, `Failed to retrieve ${deviceNameRef.current ?? "device"}`);
    },
    [handleError],
  );
  useAsyncEffect(async () => {
    if (client == null) return;
    const deviceKey = ctx.value().config.device;
    if (deviceKey === "") {
      setDev(undefined);
      return;
    }
    try {
      const d = await client.hardware.devices.retrieve<
        Properties,
        Make,
        Model,
        StateDetails
      >(deviceKey);
      setDev(d);
    } catch (e) {
      handleExc(e);
    }
  }, [ctx.value, client?.key]);
  Form.useFieldListener<string, UseContextValue>({
    ctx,
    path: "config.device",
    onChange: useCallback(
      (fs) => {
        if (!fs.touched || fs.status.variant !== "success" || client == null) return;
        client.hardware.devices
          .retrieve<Properties, Make, Model, StateDetails>(fs.value)
          .then(setDev)
          .catch(handleExc);
      },
      [client?.key, setDev, handleExc],
    ),
  });
  const handleSet = useCallback(
    (d: device.Device) => {
      if (d.key === dev?.key)
        setDev(d as device.Device<Properties, Make, Model, StateDetails>);
    },
    [dev?.key],
  );
  const handleDelete = useCallback(
    (key: device.Key) => {
      if (key === dev?.key) setDev(undefined);
    },
    [dev?.key],
  );
  Device.useSetSynchronizer(handleSet);
  Device.useDeleteSynchronizer(handleDelete);
  return dev;
};
