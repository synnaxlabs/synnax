// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, query } from "@synnaxlabs/client";
import { Device, Flux, Status, Synnax } from "@synnaxlabs/pluto";
import { array, primitive, verbs } from "@synnaxlabs/x";
import { useEffect, useMemo, useState } from "react";

import { SLAVE_SCHEMAS, type SlaveDevice } from "@/feature/ethercat/device/types";
import { type Channel } from "@/feature/ethercat/task/types";

export const {
  useRetrieve: useRetrieveSlave,
  useRetrieveStateful: useRetrieveSlaveStateful,
} = Device.createRetrieve(SLAVE_SCHEMAS);

export interface EnabledState {
  allEnabled: boolean;
  allDisabled: boolean;
}

export interface SelectEnabledStateParams {
  keys: device.Key[];
}

const EMPTY_SLAVES: SlaveDevice[] = [];

export const [useSelectEnabledState] = Flux.createSelector<
  SelectEnabledStateParams,
  EnabledState,
  SlaveDevice[]
>({
  subscribe: ({ client, args: { keys } }, notify) => {
    if (client == null) return () => {};
    const destructors = keys.map((key) => client.devices.onChange(key, notify));
    return () => destructors.forEach((d) => d());
  },
  select: ({ client, args: { keys } }) => {
    if (client == null || keys.length === 0) return EMPTY_SLAVES;
    const cached = client.devices.getCached({ keys });
    if (!query.isLive(cached)) return EMPTY_SLAVES;
    return cached as unknown as SlaveDevice[];
  },
  transform: (devices) => {
    const disabledCount = devices.filter((d) => !d.properties?.enabled).length;
    return {
      allDisabled: disabledCount === devices.length,
      allEnabled: disabledCount === 0,
    };
  },
  equal: (a, b) => a.allEnabled === b.allEnabled && a.allDisabled === b.allDisabled,
});

export const useCommonNetwork = (channels: Channel[]) => {
  const firstDeviceKey = useMemo(() => {
    const keys = channels.map((ch) => ch.device).filter((c) => c != null);
    return keys.length > 0 ? keys[0] : "";
  }, [channels]);
  const [network, setNetwork] = useState<string>("");
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  useEffect(() => {
    setNetwork("");
    if (primitive.isZero(firstDeviceKey) || client == null) return;
    handleError(async () => {
      const slave = await client.devices.retrieve({
        key: firstDeviceKey,
        schemas: SLAVE_SCHEMAS,
      });
      setNetwork(slave.properties.network ?? "");
    }, "Failed to retrieve device");
    return client.devices.onChange({ key: firstDeviceKey }, (res) => {
      if (query.isLive(res))
        setNetwork((res as unknown as SlaveDevice).properties?.network ?? "");
    });
  }, [client, firstDeviceKey, handleError]);
  return network;
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
