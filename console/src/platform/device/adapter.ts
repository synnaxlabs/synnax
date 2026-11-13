// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { actions, device, type resource } from "@synnaxlabs/client";
import { Device, Flux } from "@synnaxlabs/pluto";
import { deep, type record, verbs } from "@synnaxlabs/x";

/** The fields of a found device that the configure form and menu items read. */
export interface Configurable {
  key: string;
  name: string;
  identifier: string;
  /** Whether the user has named the device and set its identifier. */
  configured: boolean;
}

/** A save of the name and identifier of a found device. */
export interface SaveParams {
  key: string;
  name: string;
  identifier: string;
}

/** Binds the configure form and menu items to the table that stores a device. */
export interface Adapter {
  /** The make that analytics records when the user configures a device. */
  make: string;
  resourceID: (key: string) => resource.ID;
  /** Suspends until the device loads. */
  use: (key: string) => Configurable;
  /** Saves the name and identifier and marks the device configured. */
  useSave: Flux.UseUpdate<SaveParams>;
}

export interface CreateLegacyAdapterArgs {
  make: string;
  /** The properties a device starts with when the user configures it. */
  zeroProperties: record.Unknown;
}

/** Creates an {@link Adapter} over the generic device table. */
export const createLegacyAdapter = ({
  make,
  zeroProperties,
}: CreateLegacyAdapterArgs): Adapter => {
  const useConfigurable = (key: string): Configurable => {
    const { name, configured, properties } = Device.use({ key });
    const { identifier } = properties ?? {};
    return {
      key,
      name,
      configured,
      identifier: typeof identifier === "string" ? identifier : "",
    };
  };
  return {
    make,
    resourceID: device.resourceID,
    use: useConfigurable,
    useSave: Flux.createUpdate<SaveParams>({
      name: "device",
      verbs: verbs.UPDATE,
      update: async ({ client, data }) => {
        const { key, name, identifier } = data;
        const prev = await client.devices.retrieve({ key });
        const next = {
          ...prev,
          configured: true,
          name,
          properties: { ...deep.copy(zeroProperties), ...prev.properties, identifier },
        };
        const edits = actions.diffFields(device.deviceZ(), prev, next, [
          "name",
          "configured",
          "properties",
        ]);
        if (edits.length > 0) await client.devices.dispatch(key, edits);
        return data;
      },
    }).useUpdate,
  };
};
