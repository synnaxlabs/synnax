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
import {
  fromTyped,
  toView,
  useConfigDeviceKeys,
  type View,
} from "@/feature/ni/device/view";
import { FLAGS } from "@/flags";
import { Device as PlatformDevice } from "@/platform/device";

/** The key the channel of the built-in temperature sensor is stored under. */
export const BUILT_IN_TEMP_KEY = "boardTempSensor";

/**
 * The channels that the read tasks of one kind keep on a device: an index, and a
 * channel per port or line. Zero is no channel.
 */
export interface ReadChannels {
  index: channel.Key;
  channels: Record<string, channel.Key>;
}

/**
 * The channels that the write tasks of one kind keep on a device: a state index, and a
 * command and a state channel per port or line. Zero is no channel.
 */
export interface WriteChannels {
  stateIndex: channel.Key;
  channels: Record<string, PlatformDevice.CommandStatePair>;
}

interface ChannelsByKind {
  analogInput: ReadChannels;
  counterInput: ReadChannels;
  digitalInput: ReadChannels;
  analogOutput: WriteChannels;
  digitalOutput: WriteChannels;
}

/**
 * A kind of port whose channels a task stores on a device. Channels are keyed by port
 * number, by {@link BUILT_IN_TEMP_KEY}, or by `${port}l${line}` for a digital line.
 */
export type ChannelKind = keyof ChannelsByKind;

/** A device that a task configures, with its channels of one kind. */
export interface Target<K extends ChannelKind> extends View {
  configured: boolean;
  channels: ChannelsByKind[K];
}

const genericTarget = <K extends ChannelKind>(d: Device, kind: K): Target<K> => ({
  ...toView(d),
  configured: d.configured ?? false,
  channels: deep.copy(d.properties[kind]),
});

const retrieveGeneric = async <K extends ChannelKind>(
  client: Synnax,
  keys: device.Key[],
  kind: K,
): Promise<Target<K>[]> =>
  (await client.devices.retrieve({ key: keys, schemas: SCHEMAS })).map((d) =>
    genericTarget(d, kind),
  );

const writeGeneric = async <K extends ChannelKind>(
  client: Synnax,
  key: device.Key,
  kind: K,
  before: ChannelsByKind[K],
  after: ChannelsByKind[K],
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
const mergeMap = <V>(
  stored: Record<string, V>,
  before: Record<string, V>,
  after: Record<string, V>,
): Record<string, V> => {
  const merged = { ...stored };
  Object.keys(before).forEach((k) => {
    if (!(k in after)) delete merged[k];
  });
  Object.entries(after).forEach(([k, v]) => {
    if (!deep.equal(v, before[k])) merged[k] = v;
  });
  return merged;
};

const pick = (stored: channel.Key, before: channel.Key, after: channel.Key) =>
  after === before ? stored : after;

const mergeRead = (s: ReadChannels, b: ReadChannels, a: ReadChannels) => ({
  index: pick(s.index, b.index, a.index),
  channels: mergeMap(s.channels, b.channels, a.channels),
});

const mergeWrite = (s: WriteChannels, b: WriteChannels, a: WriteChannels) => ({
  stateIndex: pick(s.stateIndex, b.stateIndex, a.stateIndex),
  channels: mergeMap(s.channels, b.channels, a.channels),
});

const MERGE: {
  [K in ChannelKind]: (
    stored: ChannelsByKind[K],
    before: ChannelsByKind[K],
    after: ChannelsByKind[K],
  ) => ChannelsByKind[K];
} = {
  analogInput: mergeRead,
  counterInput: mergeRead,
  digitalInput: mergeRead,
  analogOutput: mergeWrite,
  digitalOutput: mergeWrite,
};

const indexOf = (c: ReadChannels | WriteChannels): channel.Key =>
  "index" in c ? c.index : c.stateIndex;

const orZero = (key?: channel.Key): channel.Key => key ?? 0;

const orAbsent = (key: channel.Key): channel.Key | undefined =>
  primitive.isZero(key) ? undefined : key;

const lineKey = (port: string, line: string): string => `${port}l${line}`;

const parseLineKey = (key: string): [number, number] => {
  const [port, line] = key.split("l").map(Number);
  return [port, line];
};

const toPair = ({ command, state }: ni.device.CommandStatePair) => ({
  command: orZero(command),
  state: orZero(state),
});

const fromPair = ({ command, state }: PlatformDevice.CommandStatePair) => ({
  command: orAbsent(command),
  state: orAbsent(state),
});

// The ports of a device share the index that a task creates.
const sharedIndex = (ports: Array<{ index?: channel.Key }>): channel.Key =>
  orZero(ports.find((p) => p.index != null)?.index);

const sharedStateIndex = (ports: Array<{ stateIndex?: channel.Key }>): channel.Key =>
  orZero(ports.find((p) => p.stateIndex != null)?.stateIndex);

const TYPED_CHANNELS: {
  [K in ChannelKind]: (d: ni.device.Device) => ChannelsByKind[K];
} = {
  analogInput: ({ analogInput: { index, temperature, channels } }) => ({
    index: orZero(index),
    channels: {
      ...channels,
      ...(temperature == null ? {} : { [BUILT_IN_TEMP_KEY]: temperature }),
    },
  }),
  counterInput: ({ counterInput }) => ({
    index: sharedIndex(Object.values(counterInput)),
    channels: Object.fromEntries(
      Object.entries(counterInput).flatMap(([p, c]) =>
        c.channel == null ? [] : [[p, c.channel]],
      ),
    ),
  }),
  digitalInput: ({ digitalInput }) => ({
    index: sharedIndex(Object.values(digitalInput)),
    channels: Object.fromEntries(
      Object.entries(digitalInput).flatMap(([p, { lines }]) =>
        Object.entries(lines).map(([l, c]) => [lineKey(p, l), c]),
      ),
    ),
  }),
  analogOutput: ({ analogOutput }) => ({
    stateIndex: sharedStateIndex(Object.values(analogOutput)),
    channels: Object.fromEntries(
      Object.entries(analogOutput).map(([p, pair]) => [p, toPair(pair)]),
    ),
  }),
  digitalOutput: ({ digitalOutput }) => ({
    stateIndex: sharedStateIndex(Object.values(digitalOutput)),
    channels: Object.fromEntries(
      Object.entries(digitalOutput).flatMap(([p, { lines }]) =>
        Object.entries(lines).map(([l, pair]) => [lineKey(p, l), toPair(pair)]),
      ),
    ),
  }),
};

const byPort = <V>(channels: Record<string, V>): Map<number, Record<number, V>> => {
  const ports = new Map<number, Record<number, V>>();
  Object.entries(channels).forEach(([key, value]) => {
    const [port, line] = parseLineKey(key);
    ports.set(port, { ...ports.get(port), [line]: value });
  });
  return ports;
};

const TO_TYPED: {
  [K in ChannelKind]: (c: ChannelsByKind[K]) => Partial<ni.device.ConfigurationUpdate>;
} = {
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
  digitalInput: ({ index, channels }) => ({
    digitalInput: Object.fromEntries(
      [...byPort(channels)].map(([p, lines]) => [p, { index: orAbsent(index), lines }]),
    ),
  }),
  analogOutput: ({ stateIndex, channels }) => ({
    analogOutput: Object.fromEntries(
      Object.entries(channels).map(([p, pair]) => [
        Number(p),
        { stateIndex: orAbsent(stateIndex), ...fromPair(pair) },
      ]),
    ),
  }),
  digitalOutput: ({ stateIndex, channels }) => ({
    digitalOutput: Object.fromEntries(
      [...byPort(channels)].map(([p, lines]) => [
        p,
        {
          stateIndex: orAbsent(stateIndex),
          lines: Object.fromEntries(
            Object.entries(lines).map(([l, pair]) => [l, fromPair(pair)]),
          ),
        },
      ]),
    ),
  }),
};

const typedTarget = <K extends ChannelKind>(
  d: ni.device.Device,
  kind: K,
): Target<K> => ({
  ...fromTyped(d),
  configured: d.identifier !== "",
  channels: TYPED_CHANNELS[kind](d),
});

const retrieveTyped = async <K extends ChannelKind>(
  client: Synnax,
  keys: device.Key[],
  kind: K,
): Promise<Target<K>[]> =>
  (await client.ni.devices.retrieve(keys)).map((d) => typedTarget(d, kind));

const writeTyped = async <K extends ChannelKind>(
  client: Synnax,
  key: device.Key,
  kind: K,
  before: ChannelsByKind[K],
  after: ChannelsByKind[K],
): Promise<void> => {
  const stored = await client.ni.devices.retrieve({ key });
  const merged = MERGE[kind](TYPED_CHANNELS[kind](stored), before, after);
  const written = await client.ni.devices.update(key, {
    ...ni.device.configurationZ.parse(stored),
    ...TO_TYPED[kind](merged),
    group: "configuration",
  });
  // A typed map without a device-level index drops an index that has no channels.
  const index = indexOf(merged);
  if (primitive.isNonZero(index) && indexOf(TYPED_CHANNELS[kind](written)) !== index)
    await client.channels.delete(index);
};

/**
 * Retrieves the devices that a task configures, with their channels of the kind. Reads
 * the typed device table when `typedNIDevices` is on.
 */
export const retrieveTargets = FLAGS.typedNIDevices ? retrieveTyped : retrieveGeneric;

/**
 * Applies the changes from before to after to the channels of the kind on the device,
 * and keeps the keys that another write stored since before. Writes the typed device
 * table when `typedNIDevices` is on.
 */
export const writeChannels = FLAGS.typedNIDevices ? writeTyped : writeGeneric;

const useGeneric = <K extends ChannelKind>(
  keys: device.Key[],
  kind: K,
): Target<K>[] | undefined => useByKeys(keys)?.map((d) => genericTarget(d, kind));

const useTyped = <K extends ChannelKind>(
  keys: device.Key[],
  kind: K,
): Target<K>[] | undefined => useTypedByKeys(keys)?.map((d) => typedTarget(d, kind));

/**
 * The devices the keys name, with their channels of the kind, or undefined until they
 * resolve or on failure. Reads the typed device table when `typedNIDevices` is on.
 */
export const useTargets = FLAGS.typedNIDevices ? useTyped : useGeneric;

/** The device the form's config names, with its channels of the kind. */
export const useTargetFromConfig = <K extends ChannelKind>(
  kind: K,
): Target<K> | undefined => useTargets(useConfigDeviceKeys(), kind)?.[0];
