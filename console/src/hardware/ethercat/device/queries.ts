// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Device, Flux } from "@synnaxlabs/pluto";
import { array, primitive } from "@synnaxlabs/x";
import { useCallback, useEffect, useMemo, useState } from "react";

import { type SlaveProperties } from "@/hardware/ethercat/device/types";
import { type Channel } from "@/hardware/ethercat/task/types";

export const { useRetrieve: useRetrieveSlave, useRetrieveObservable } =
  Device.createRetrieve<SlaveProperties>();

export const useCommonNetwork = (channels: Channel[]) => {
  const firstDeviceKey = useMemo(() => {
    const keys = channels.map((ch) => ch.device).filter((c) => c != null);
    return keys.length > 0 ? keys[0] : "";
  }, [channels]);
  const [network, setNetwork] = useState<string>("");
  const { retrieve } = useRetrieveObservable({
    onChange: useCallback((res) => setNetwork(res.data?.properties?.network ?? ""), []),
  });
  useEffect(() => {
    if (primitive.isZero(firstDeviceKey)) return;
    retrieve({ key: firstDeviceKey });
  }, [firstDeviceKey, retrieve]);
  return network;
};

export interface ToggleEnabledParams {
  keys: device.Key | device.Key[];
  enabled?: boolean;
}

export const { useUpdate: useToggleEnabled } = Flux.createUpdate<
  ToggleEnabledParams,
  Device.FluxSubStore,
  ToggleEnabledParams
>({
  name: "Toggle Enabled",
  verbs: Flux.UPDATE_VERBS,
  update: async ({ data, client, store, rollbacks }) => {
    const keys = array.toArray(data.keys);

    const devices = await Device.retrieveMultiple<SlaveProperties>({
      client,
      store,
      query: { keys },
    });

    const enabledValue = data.enabled ?? !devices[0]?.properties?.enabled;

    const updated = devices.map((dev) => ({
      ...dev,
      properties: { ...dev.properties, enabled: enabledValue },
    }));

    rollbacks.push(store.devices.set(updated));

    await client.devices.create(updated);

    return data;
  },
});
