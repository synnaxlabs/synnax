// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type Synnax } from "@synnaxlabs/client";
import { patch, type record } from "@synnaxlabs/x";
import { z } from "zod";

/**
 * Writes the properties of a device that differ from their values before the caller
 * changed them, so a concurrent write to other properties survives. Sends nothing when
 * no property differs. Writes every property when the properties have a version, since
 * a read migrates older stored versions and the stored version must change too.
 * @param before - The properties before the caller changed them.
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
  before: z.infer<Properties>,
  schemas: device.DeviceSchemas<Properties, Make, Model>,
): Promise<void> => {
  const schema = z.object({ properties: schemas.properties });
  const base = { properties: before };
  let actions = patch.diff(schema, base, { properties });
  if (actions.length === 0) return;
  if ("version" in properties)
    actions = [
      { type: "set", set: { path: ["properties"], value: properties, guards: [] } },
    ];
  await client.devices.dispatch(
    key,
    actions.map((action) => patch.camelToSnake(action, { schema, data: base })),
  );
};
