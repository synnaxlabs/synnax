// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, query } from "@synnaxlabs/client";
import { Form, Status, Synnax } from "@synnaxlabs/pluto";
import { primitive, type record } from "@synnaxlabs/x";
import { useEffect, useState } from "react";
import { type z } from "zod";

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
  Properties extends z.ZodType<record.Unknown> = z.ZodType<record.Unknown>,
  Make extends z.ZodType<string> = z.ZodString,
  Model extends z.ZodType<string> = z.ZodString,
>(
  schemas?: device.DeviceSchemas<Properties, Make, Model>,
): device.Device<Properties, Make, Model> | null => {
  const devKey = Form.useFieldValue<string>("config.device");
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  type Device = device.Device<Properties, Make, Model>;
  const [dev, setDev] = useState<Device | null>(null);
  const [frozenSchemas] = useState(() => schemas);
  useEffect(() => {
    setDev(null);
    if (primitive.isZero(devKey) || client == null) return;
    // The client answers with generically parsed properties, so the caller's schemas
    // are applied here, on the retrieve and on every update alike. deviceZ widens make
    // and model to a union with its own defaults, hence the narrowing.
    const schema = device.deviceZ(frozenSchemas) as unknown as z.ZodType<Device>;
    handleError(async () => {
      const res = await client.devices.retrieve({ key: devKey, includeStatus: true });
      setDev(schema.parse(res));
    }, "Failed to retrieve device");
    return client.devices.onChange({ key: devKey, includeStatus: true }, (res) => {
      if (query.isLive(res)) setDev(schema.parse(res));
    });
  }, [client, devKey, frozenSchemas, handleError]);
  return dev;
};
