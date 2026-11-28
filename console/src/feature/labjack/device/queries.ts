// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { labjack } from "@synnaxlabs/client";
import { Device, Flux } from "@synnaxlabs/pluto";
import { verbs } from "@synnaxlabs/x";

import { SCHEMAS } from "@/feature/labjack/device/types";

export const { use } = Device.createRetrieve(SCHEMAS);

const RESOURCE_NAME = "LabJack device";

const BASE_QUERY = { includeStatus: true } as const;

export type RetrieveQuery = labjack.device.RetrieveSingleParams;

/** The LabJack device with the key, with its status, from the typed device table. */
export const { use: useSuspended, useResult: useRetrieve } = Flux.createRetrieve<
  RetrieveQuery,
  labjack.device.Device
>({
  name: RESOURCE_NAME,
  normalizeQuery: (query) => ({ ...BASE_QUERY, ...query }),
  retrieve: async ({ client, query }) => await client.labjack.devices.retrieve(query),
  onChange: ({ client, query }, handler) =>
    client.labjack.devices.watch(query).onChange(handler),
  getCached: ({ client, query }) => client.labjack.devices.watch(query).getCached(),
});

export interface RenameParams extends Pick<labjack.device.Device, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data }) => {
    const { key, name } = data;
    await client.labjack.devices.dispatch(
      key,
      labjack.device.set({ path: ["name"], value: name }),
    );
    return data;
  },
});

export interface SaveParams extends Pick<
  labjack.device.Device,
  "key" | "name" | "identifier"
> {}

/** Names a LabJack device and sets its identifier in one atomic write. */
export const { useUpdate: useSave } = Flux.createUpdate<SaveParams>({
  name: RESOURCE_NAME,
  verbs: verbs.UPDATE,
  update: async ({ client, data }) => {
    const { key, name, identifier } = data;
    await client.labjack.devices.dispatch(key, [
      labjack.device.set({ path: ["name"], value: name }),
      labjack.device.set({ path: ["identifier"], value: identifier }),
    ]);
    return data;
  },
});

export interface ChangeIdentifierParams extends Pick<
  labjack.device.Device,
  "key" | "identifier"
> {}

export const { useUpdate: useChangeIdentifier } =
  Flux.createUpdate<ChangeIdentifierParams>({
    name: "LabJack device identifier",
    verbs: { present: "change", past: "changed", participle: "changing" },
    update: async ({ client, data }) => {
      const { key, identifier } = data;
      await client.labjack.devices.dispatch(
        key,
        labjack.device.set({ path: ["identifier"], value: identifier }),
      );
      return data;
    },
  });

export type DeleteParams = labjack.device.Key | labjack.device.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data }) => {
    await client.labjack.devices.delete(data);
    return data;
  },
});
