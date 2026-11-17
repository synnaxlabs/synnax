// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { actions, channel, device } from "@synnaxlabs/client";
import { Flux } from "@synnaxlabs/pluto";
import { deep, type record, verbs } from "@synnaxlabs/x";

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
 * @returns the first idea that gives no taken channel name prefix. A taken idea takes
 * the lowest free number first, such as `iol_2`.
 */
const firstFree = (ideas: string[], taken: ReadonlySet<string>): string | undefined => {
  for (const idea of ideas)
    for (let n = 1; ; n++) {
      const candidate = n === 1 ? idea : `${idea}_${n}`;
      if (!identifierZ.safeParse(candidate).success) break;
      if (!taken.has(channel.escapeInvalidName(candidate))) return candidate;
    }
  return undefined;
};

export interface ConfigureManyParams {
  keys: device.Key[];
  /** The properties that each device starts with. */
  zeroProperties: record.Unknown;
  /** @returns the identifiers to try for the device, the best first. */
  suggestIdentifiers: (device: device.Device) => string[];
}

/**
 * Configures each device that is not configured yet with the first suggested
 * identifier that no other device uses.
 *
 * @throws {Error} if no suggested identifier is free for a device.
 */
export const { useUpdate: useConfigureMany } = Flux.createUpdate<ConfigureManyParams>({
  name: "devices",
  verbs: verbs.UPDATE,
  update: async ({ client, data }) => {
    const { keys, zeroProperties, suggestIdentifiers } = data;
    for (const key of keys) {
      // A fresh read sees the identifiers that other runs already saved, not their
      // pending picks.
      const all = await client.devices.retrieve({});
      const prev = all.find((d) => d.key === key);
      if (prev == null || prev.configured) continue;
      const taken = prefixesOf(all.filter((d) => d.key !== key));
      const identifier = firstFree(suggestIdentifiers(prev), taken);
      if (identifier == null)
        throw new Error(
          `Synnax cannot suggest a free identifier for ${prev.name}. Configure the ` +
            "device by hand.",
        );
      const next: device.Device = {
        ...prev,
        configured: true,
        properties: {
          ...deep.copy(zeroProperties),
          ...prev.properties,
          identifier,
        },
      };
      const edits = actions.diffFields(device.deviceZ(), prev, next, [
        "configured",
        "properties",
      ]);
      await client.devices.dispatch(key, edits);
    }
    return data;
  },
});
