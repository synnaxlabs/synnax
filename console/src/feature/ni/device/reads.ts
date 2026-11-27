// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type device, ni, type Synnax } from "@synnaxlabs/client";
import { deep, primitive } from "@synnaxlabs/x";

import { useByKeys, useTypedByKeys } from "@/feature/ni/device/queries";
import { type Device, SCHEMAS } from "@/feature/ni/device/types";
import { fromTyped, toView, type View } from "@/feature/ni/device/view";
import { FLAGS } from "@/flags";
import { Device as PlatformDevice } from "@/platform/device";

/** The key the channel of the built-in temperature sensor is stored under. */
export const BUILT_IN_TEMP_KEY = "boardTempSensor";

/**
 * The channels that the read tasks of one kind keep on a device: an index, and a
 * channel per port, keyed by port number or {@link BUILT_IN_TEMP_KEY}. Zero is no
 * channel.
 */
export interface ReadChannels {
  index: channel.Key;
  channels: Record<string, channel.Key>;
}

export type ReadKind = "analogInput" | "counterInput";

/** A device that a read task configures, with its read channels of one kind. */
export interface ReadTarget extends View {
  configured: boolean;
  channels: ReadChannels;
}

const genericTarget = (d: Device, kind: ReadKind): ReadTarget => ({
  ...toView(d),
  configured: d.configured ?? false,
  channels: deep.copy(d.properties[kind]),
});

const retrieveGeneric = async (
  client: Synnax,
  keys: device.Key[],
  kind: ReadKind,
): Promise<ReadTarget[]> =>
  (await client.devices.retrieve({ key: keys, schemas: SCHEMAS })).map((d) =>
    genericTarget(d, kind),
  );

const writeGeneric = async (
  client: Synnax,
  key: device.Key,
  kind: ReadKind,
  before: ReadChannels,
  after: ReadChannels,
): Promise<void> => {
  const { properties } = await client.devices.retrieve({ key, schemas: SCHEMAS });
  await PlatformDevice.setProperties(
    client,
    { key, properties: { ...properties, [kind]: after } },
    { ...properties, [kind]: before },
    SCHEMAS,
  );
};

/** Applies the changes from before to after onto stored, and keeps other keys. */
const merge = (
  stored: ReadChannels,
  before: ReadChannels,
  after: ReadChannels,
): ReadChannels => {
  const channels = { ...stored.channels };
  Object.keys(before.channels).forEach((k) => {
    if (!(k in after.channels)) delete channels[k];
  });
  Object.entries(after.channels).forEach(([k, c]) => {
    if (c !== before.channels[k]) channels[k] = c;
  });
  return {
    index: after.index === before.index ? stored.index : after.index,
    channels,
  };
};

const orZero = (key?: channel.Key): channel.Key => key ?? 0;

const orAbsent = (key: channel.Key): channel.Key | undefined =>
  primitive.isZero(key) ? undefined : key;

const TYPED_CHANNELS: Record<ReadKind, (d: ni.device.Device) => ReadChannels> = {
  analogInput: ({ analogInput: { index, temperature, channels } }) => ({
    index: orZero(index),
    channels: {
      ...channels,
      ...(temperature == null ? {} : { [BUILT_IN_TEMP_KEY]: temperature }),
    },
  }),
  // The counters of a device share the index that a counter read task creates.
  counterInput: ({ counterInput }) => {
    const counters = Object.entries(counterInput);
    return {
      index: orZero(counters.find(([, c]) => c.index != null)?.[1].index),
      channels: Object.fromEntries(
        counters.flatMap(([p, c]) => (c.channel == null ? [] : [[p, c.channel]])),
      ),
    };
  },
};

const TO_TYPED: Record<
  ReadKind,
  (c: ReadChannels) => Partial<ni.device.ConfigurationUpdate>
> = {
  analogInput: ({
    index,
    channels: { [BUILT_IN_TEMP_KEY]: temperature, ...ports },
  }) => ({
    analogInput: {
      index: orAbsent(index),
      temperature: temperature == null ? undefined : orAbsent(temperature),
      channels: Object.fromEntries(
        Object.entries(ports).map(([p, c]) => [Number(p), c]),
      ),
    },
  }),
  counterInput: ({ index, channels }) => ({
    counterInput: Object.fromEntries(
      Object.entries(channels).map(([p, c]) => [
        Number(p),
        { index: orAbsent(index), channel: orAbsent(c) },
      ]),
    ),
  }),
};

const typedTarget = (d: ni.device.Device, kind: ReadKind): ReadTarget => ({
  ...fromTyped(d),
  configured: d.identifier !== "",
  channels: TYPED_CHANNELS[kind](d),
});

const retrieveTyped = async (
  client: Synnax,
  keys: device.Key[],
  kind: ReadKind,
): Promise<ReadTarget[]> =>
  (await client.ni.devices.retrieve(keys)).map((d) => typedTarget(d, kind));

const writeTyped = async (
  client: Synnax,
  key: device.Key,
  kind: ReadKind,
  before: ReadChannels,
  after: ReadChannels,
): Promise<void> => {
  const stored = await client.ni.devices.retrieve({ key });
  const merged = merge(TYPED_CHANNELS[kind](stored), before, after);
  const written = await client.ni.devices.update(key, {
    ...ni.device.configurationZ.parse(stored),
    ...TO_TYPED[kind](merged),
    group: "configuration",
  });
  // A typed map without a device-level index drops an index that has no channels.
  const { index } = TYPED_CHANNELS[kind](written);
  if (primitive.isNonZero(merged.index) && index !== merged.index)
    await client.channels.delete(merged.index);
};

/**
 * Retrieves the devices that a read task configures, with their read channels of the
 * kind. Reads the typed device table when `typedNIDevices` is on.
 */
export const retrieveReadTargets = FLAGS.typedNIDevices
  ? retrieveTyped
  : retrieveGeneric;

/**
 * Applies the changes from before to after to the read channels of the kind on the
 * device, and keeps the keys that another write stored since before. Writes the typed
 * device table when `typedNIDevices` is on.
 */
export const writeReadChannels = FLAGS.typedNIDevices ? writeTyped : writeGeneric;

const useGeneric = (keys: device.Key[], kind: ReadKind): ReadTarget[] | undefined =>
  useByKeys(keys)?.map((d) => genericTarget(d, kind));

const useTyped = (keys: device.Key[], kind: ReadKind): ReadTarget[] | undefined =>
  useTypedByKeys(keys)?.map((d) => typedTarget(d, kind));

/**
 * The devices the keys name, with their read channels of the kind, or undefined until
 * they resolve or on failure. Reads the typed device table when `typedNIDevices` is on.
 */
export const useReadTargets = FLAGS.typedNIDevices ? useTyped : useGeneric;
