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
import { type record } from "@synnaxlabs/x";
import { useCallback, useState } from "react";
import { type z } from "zod/v4";

interface UseContextValue
  extends z.ZodObject<{
    config: z.ZodObject<{ device: z.ZodString }>;
  }> {}

/**
 * A hook that retrieves and subscribes to updates for a device. Must be used within a
 * Form context that has a schema matching the following structure:
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
  Properties extends record.Unknown = record.Unknown,
  Make extends string = string,
  Model extends string = string,
>(): device.Device<Properties, Make, Model> | undefined => {
  const ctx = Form.useContext<UseContextValue>();
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const [dev, setDev] = useState<device.Device<Properties, Make, Model>>();
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
  useAsyncEffect(
    async (signal) => {
      if (client == null) return;
      const deviceKey = ctx.value().config.device;
      if (deviceKey === "") {
        setDev(undefined);
        return;
      }
      try {
        const d = await client.hardware.devices.retrieve<Properties, Make, Model>(
          deviceKey,
        );
        if (signal.aborted) return;
        setDev(d);
      } catch (e) {
        handleExc(e);
      }
    },
    [ctx.value, client?.key],
  );
  const device = Form.useFieldValue<string>("config.device", { ctx });
  useAsyncEffect(
    async (signal) => {
      if (client == null) return;
      const d = await client.hardware.devices.retrieve<Properties, Make, Model>(device);
      if (signal.aborted) return;
      setDev(d);
    },
    [device, client?.key],
  );
  const handleSet = useCallback(
    (d: device.Device) => {
      if (d.key === dev?.key) setDev(d as device.Device<Properties, Make, Model>);
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
