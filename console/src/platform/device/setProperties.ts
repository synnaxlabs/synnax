// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, type Synnax } from "@synnaxlabs/client";
import { caseconv, type record } from "@synnaxlabs/x";
import { type z } from "zod";

/**
 * Replaces the properties of a device on the Core.
 * @param schemas - The schemas that the device was retrieved with. They name the
 * property keys that keep their case.
 * @throws {NotFoundError} if the device does not exist.
 */
export const setProperties = async <
  Properties extends z.ZodType<record.Unknown>,
  Make extends z.ZodType<string>,
  Model extends z.ZodType<string>,
>(
  client: Synnax,
  {
    key,
    properties,
  }: Pick<device.Device<Properties, Make, Model>, "key" | "properties">,
  schemas?: device.DeviceSchemas<Properties, Make, Model>,
): Promise<void> => {
  const value = caseconv.camelToSnake(properties, { schema: schemas?.properties });
  await client.devices.dispatch(key, device.set({ path: ["properties"], value }));
};
