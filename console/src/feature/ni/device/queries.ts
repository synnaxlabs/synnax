// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, ni, type Synnax } from "@synnaxlabs/client";
import { Device as PDevice, Flux } from "@synnaxlabs/pluto";
import { primitive, verbs } from "@synnaxlabs/x";

import { type Device, SCHEMAS } from "@/feature/ni/device/types";
import { Device as PlatformDevice } from "@/platform/device";

const { useResult } = PDevice.createRetrieve(SCHEMAS);

/** The device the form's config names, or undefined until it resolves or on failure. */
export const useFromConfig = PlatformDevice.createUseFromConfig(useResult);

/** The device with the key, or undefined for an empty key, until it resolves, or on
 * failure. */
export const useByKey = (key: device.Key): Device | undefined =>
  useResult(primitive.isNonZero(key) ? { key } : null).data;

const { useResult: useResultMultiple } = PDevice.createRetrieveMultiple(SCHEMAS);

/** The devices the keys name, or undefined until they resolve or on failure. */
export const useByKeys = (keys: device.Key[]): Device[] | undefined =>
  useResultMultiple(keys.length === 0 ? null : { where: { key: { in: keys } } }).data;

const RESOURCE_NAME = "NI device";

const BASE_QUERY = { includeStatus: true } as const;

export type RetrieveQuery = ni.device.RetrieveSingleParams;

/** The NI device with the key, with its status, from the typed device table. */
export const { use, useResult: useRetrieve } = Flux.createRetrieve<
  RetrieveQuery,
  ni.device.Device
>({
  name: RESOURCE_NAME,
  normalizeQuery: (query) => ({ ...BASE_QUERY, ...query }),
  retrieve: async ({ client, query }) => await client.ni.devices.retrieve(query),
  onChange: ({ client, query }, handler) =>
    client.ni.devices.watch(query).onChange(handler),
  getCached: ({ client, query }) => client.ni.devices.watch(query).getCached(),
});

export interface RenameParams extends Pick<ni.device.Device, "key" | "name"> {}

/**
 * Writes the fields of an NI device's configuration that change sets. The Core clears
 * each configuration field that an update leaves out, so the write sends the stored
 * configuration with the changes.
 */
const configure = async (
  client: Synnax,
  key: ni.device.Key,
  change: Partial<ni.device.Configuration>,
): Promise<void> => {
  const stored = ni.device.configurationZ.parse(
    await client.ni.devices.retrieve({ key }),
  );
  await client.ni.devices.update(key, {
    ...stored,
    ...change,
    group: "configuration",
  });
};

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data }) => {
    const { key, name } = data;
    await configure(client, key, { name });
    return data;
  },
});

export interface SaveParams extends Pick<
  ni.device.Device,
  "key" | "name" | "identifier"
> {}

/** Names an NI device and sets its identifier. */
export const { useUpdate: useSave } = Flux.createUpdate<SaveParams>({
  name: RESOURCE_NAME,
  verbs: verbs.UPDATE,
  update: async ({ client, data }) => {
    const { key, name, identifier } = data;
    await configure(client, key, { name, identifier });
    return data;
  },
});

export interface ChangeIdentifierParams extends Pick<
  ni.device.Device,
  "key" | "identifier"
> {}

export const { useUpdate: useChangeIdentifier } =
  Flux.createUpdate<ChangeIdentifierParams>({
    name: "NI device identifier",
    verbs: { present: "change", past: "changed", participle: "changing" },
    update: async ({ client, data }) => {
      const { key, identifier } = data;
      await configure(client, key, { identifier });
      return data;
    },
  });

export type DeleteParams = ni.device.Key | ni.device.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data }) => {
    await client.ni.devices.delete(data);
    return data;
  },
});
