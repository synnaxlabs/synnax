// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type query } from "@synnaxlabs/client";
import { Device as PDevice, Flux, Form } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";

import { type Device, SCHEMAS } from "@/feature/ni/device/types";

export const { use, useResult } = PDevice.createRetrieve(SCHEMAS);

/** The device the form's config names, or undefined until it resolves or on failure. */
export const useFromConfig = (): Device | undefined => {
  const key = Form.useFieldValue<device.Key>("config.device", { optional: true });
  return useResult(primitive.isNonZero(key) ? { key } : null).data;
};

const { useResult: useResultByKeys } = Flux.createRetrieve<
  { keys: device.Key[] },
  Device[]
>({
  name: "NI devices",
  retrieve: async ({ client, query: { keys } }) =>
    await client.devices.retrieve({ keys, schemas: SCHEMAS }),
  onChange: ({ client, query }, handler) =>
    client.devices.onChange(
      query,
      handler as unknown as query.ChangeHandler<device.Device[]>,
    ),
  getCached: ({ client, query }) =>
    client.devices.getCached(query) as query.Cached<Device[]> | undefined,
});

/** The devices the keys name, or undefined until they resolve or on failure. */
export const useByKeys = (keys: device.Key[]): Device[] | undefined =>
  useResultByKeys(keys.length === 0 ? null : { keys }).data;
