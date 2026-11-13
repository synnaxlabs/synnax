// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { actions, channel, device } from "@synnaxlabs/client";
import { Device, Flux } from "@synnaxlabs/pluto";
import { deep, type record, strings, unique, verbs } from "@synnaxlabs/x";

import { identifierZ } from "@/platform/device/types";

/** @returns the channel name prefixes that the identifiers of the devices give. */
const prefixesOf = (devices: device.Device[]): Set<string> =>
  new Set(
    devices.flatMap(({ properties }) =>
      typeof properties?.identifier === "string" && properties.identifier !== ""
        ? [channel.escapeInvalidName(properties.identifier)]
        : [],
    ),
  );

/**
 * @returns short identifiers for the name that give no taken channel name prefix. A
 * taken suggestion takes the lowest free number, such as `iol_2`.
 */
const suggest = (name: string, taken: ReadonlySet<string>): string[] =>
  unique.unique(
    strings.createShortIdentifiers(name).flatMap((id) => {
      for (let n = 1; ; n++) {
        const candidate = n === 1 ? id : `${id}_${n}`;
        if (!identifierZ.safeParse(candidate).success) return [];
        if (!taken.has(channel.escapeInvalidName(candidate))) return [candidate];
      }
    }),
  );

/**
 * @returns short identifiers for the name of the device that no other device uses, the
 * best first.
 */
export const useIdentifierSuggestions = (name: string, key: device.Key): string[] => {
  const devices = Device.useResultMultiple({}).data ?? [];
  return suggest(name, prefixesOf(devices.filter((d) => d.key !== key)));
};

export interface ConfigureManyParams {
  keys: device.Key[];
  /** The properties that each device starts with. */
  zeroProperties: record.Unknown;
}

/**
 * Configures each device that is not configured yet, with its name and the best
 * identifier that no other device uses.
 *
 * @throws {Error} if no suggested identifier is free for the name of a device, or the
 * name gives no suggestion.
 */
export const { useUpdate: useConfigureMany } = Flux.createUpdate<ConfigureManyParams>({
  name: "devices",
  verbs: verbs.UPDATE,
  update: async ({ client, data }) => {
    for (const key of data.keys) {
      // A fresh read before each write sees what an overlapping run already wrote.
      const all = await client.devices.retrieve({});
      const prev = all.find((d) => d.key === key);
      if (prev == null || prev.configured) continue;
      const [identifier] = suggest(
        prev.name,
        prefixesOf(all.filter((d) => d.key !== key)),
      );
      if (identifier == null)
        throw new Error(
          `Synnax cannot suggest a free identifier for ${prev.name}. Configure the ` +
            "device by hand.",
        );
      const next: device.Device = {
        ...prev,
        configured: true,
        properties: {
          ...deep.copy(data.zeroProperties),
          ...prev.properties,
          identifier,
        },
      };
      const edits = actions.diffFields(device.deviceZ(), prev, next, [
        "configured",
        "properties",
      ]);
      await client.devices.dispatch(prev.key, edits);
    }
    return data;
  },
});
