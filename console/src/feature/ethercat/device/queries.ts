// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type query } from "@synnaxlabs/client";
import { Device, Flux } from "@synnaxlabs/pluto";
import { array, primitive, verbs } from "@synnaxlabs/x";
import { useMemo } from "react";

import { SLAVE_SCHEMAS, type SlaveDevice } from "@/feature/ethercat/device/types";
import { type Channel } from "@/feature/ethercat/task/types";

export const { useRetrieve: useRetrieveSlave, useCached: useCachedSlave } =
  Device.createRetrieve(SLAVE_SCHEMAS);

const { useRetrieve: useRetrieveSlaves } = Flux.createRetrieve<
  { keys: device.Key[] },
  SlaveDevice[]
>({
  name: "EtherCAT slaves",
  retrieve: async ({ client, query: { keys } }) =>
    await client.devices.retrieve({ keys, schemas: SLAVE_SCHEMAS }),
  onChange: ({ client, query: { keys } }, handler) =>
    client.devices.onChange(
      { keys },
      handler as unknown as query.ChangeHandler<device.Device[]>,
    ),
  getCached: ({ client, query: { keys } }) =>
    client.devices.getCached({ keys }) as query.Cached<SlaveDevice[]> | undefined,
  // The client allocates a fresh array of stable rows per read.
  equal: (a, b) => a.length === b.length && a.every((dev, i) => dev === b[i]),
});

export interface EnabledState {
  allEnabled: boolean;
  allDisabled: boolean;
}

export interface EnabledStateParams {
  keys: device.Key[];
}

export const useEnabledState = ({ keys }: EnabledStateParams): EnabledState => {
  const slaves = useRetrieveSlaves({ keys });
  return useMemo(() => {
    const disabledCount = slaves.filter((d) => !d.properties?.enabled).length;
    return {
      allDisabled: disabledCount === slaves.length,
      allEnabled: disabledCount === 0,
    };
  }, [slaves]);
};

export const useCommonNetwork = (channels: Channel[]) => {
  const firstDeviceKey = useMemo(() => {
    const keys = channels.map((ch) => ch.device).filter((c) => c != null);
    return keys.length > 0 ? keys[0] : "";
  }, [channels]);
  const slave = useCachedSlave(
    primitive.isZero(firstDeviceKey) ? null : { key: firstDeviceKey },
  );
  return slave?.properties?.network ?? "";
};

export interface ToggleEnabledParams {
  keys: device.Key | device.Key[];
  enabled?: boolean;
}

export const { useUpdate: useToggleEnabled } = Flux.createUpdate<ToggleEnabledParams>({
  name: "Toggle Enabled",
  verbs: verbs.UPDATE,
  update: async ({ data, client }) => {
    const keys = array.toArray(data.keys);

    const devices = await client.devices.retrieve({
      keys,
      includeStatus: true,
      schemas: SLAVE_SCHEMAS,
    });

    const enabledValue = data.enabled ?? !devices[0]?.properties?.enabled;

    const updated = devices.map((dev) => ({
      ...dev,
      properties: { ...dev.properties, enabled: enabledValue },
    }));

    await client.devices.create(updated);

    return data;
  },
});
