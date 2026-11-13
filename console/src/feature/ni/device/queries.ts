// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, ni } from "@synnaxlabs/client";
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
const PLURAL_RESOURCE_NAME = "NI devices";

const BASE_QUERY = { includeStatus: true } as const;

export type RetrieveQuery = ni.device.RetrieveSingleParams;

/** The NI device with the key, with its status, from the typed device table. */
export const { useResult: useRetrieve } = Flux.createRetrieve<
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

export type ListQuery = ni.device.RetrieveRequest;

/** Lists NI devices from the typed device table, with their statuses. */
export const useList = ({
  query,
  ...rest
}: Omit<
  Flux.UseListParams<ListQuery, ni.device.Device>,
  "name"
> = {}): Flux.UseListReturn<ni.device.Key, ni.device.Device> =>
  Flux.useList((client) => client.ni.devices, {
    ...rest,
    name: PLURAL_RESOURCE_NAME,
    query: query === null ? null : { ...BASE_QUERY, ...query },
  });

export interface UpdateConfigurationParams {
  key: ni.device.Key;
  /** The fields to change. The rest keep their stored values. */
  configuration: Partial<ni.device.Configuration>;
}

/**
 * Writes the configuration group of an NI device. It reads the stored group first,
 * because the Core clears each field that the update leaves out.
 */
export const { useUpdate: useUpdateConfiguration } =
  Flux.createUpdate<UpdateConfigurationParams>({
    name: RESOURCE_NAME,
    verbs: verbs.UPDATE,
    update: async ({ client, data }) => {
      const { key, configuration } = data;
      const stored = ni.device.configurationZ.parse(
        await client.ni.devices.retrieve({ key }),
      );
      await client.ni.devices.update(key, {
        ...stored,
        ...configuration,
        group: "configuration",
      });
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
