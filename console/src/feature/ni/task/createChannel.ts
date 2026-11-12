// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ni } from "@synnaxlabs/client";
import { deep, id } from "@synnaxlabs/x";

import { type Device } from "@/feature/ni/device/types";
import {
  ANALOG_INPUT,
  ANALOG_OUTPUT,
  claimsOf,
  COUNTER_INPUT,
  digitalPorts,
  lineClaims,
  lineName,
  lowestFreePort,
} from "@/feature/ni/task/ports";
import {
  type AIChannel,
  type AOChannel,
  type CIChannel,
  createAIChannel,
  createAOChannel,
  createCIChannel,
  createDIChannel,
  createDOChannel,
  type DIChannel,
  type DigitalChannel,
  type DOChannel,
  isGenerated,
} from "@/feature/ni/task/types";
import { Task } from "@/platform/task";

/** Where a new channel goes. Each field it leaves out takes a default. */
export interface Placement {
  /** The channel to copy. Defaults to the first channel. */
  copy?: string;
  /** Defaults to the copied channel's device. */
  device?: string;
  /** Defaults to the lowest free port on the device. */
  port?: number;
}

const template = <C extends { key: string }>(
  channels: C[],
  zero: C,
  copy?: string,
): C => {
  if (copy == null) return deep.copy(channels[0] ?? zero);
  return deep.copy(channels.find(({ key }) => key === copy) ?? zero);
};

const hardwareOf = (devices: Device[], key: string): ni.Hardware | undefined =>
  devices.find((d) => d.key === key)?.properties.hardware;

/**
 * Creates an analog input channel. Without a port, it takes the lowest port on its
 * device that is free, along with its negative input when the channel is
 * differential.
 */
export const createNextAIChannel = (
  channels: AIChannel[],
  { copy, device, port }: Placement = {},
  devices: Device[] = [],
): AIChannel => {
  const next: AIChannel = {
    ...template(channels, createAIChannel(), copy),
    key: id.create(),
    ...Task.READ_CHANNEL_OVERRIDE,
  };
  if (device != null) next.device = device;
  if (!("port" in next)) return next;
  if (port != null) return { ...next, port };
  const hardware = hardwareOf(devices, next.device);
  const own = channels.filter((c) => c.device === next.device);
  const free = lowestFreePort(
    hardware == null ? null : ANALOG_INPUT.ports(hardware),
    claimsOf(ANALOG_INPUT, own, hardware),
    (p) => ANALOG_INPUT.holds({ ...next, port: p.port }, hardware),
  );
  return { ...next, port: free };
};

/** Creates an analog output channel on the lowest free port, or on the given one. */
export const createNextAOChannel = (
  channels: AOChannel[],
  { copy, port }: Omit<Placement, "device"> = {},
  device?: Device,
): AOChannel => {
  const hardware = device?.properties.hardware;
  const base = template(channels, createAOChannel(), copy);
  const next: AOChannel = isGenerated(base)
    ? { ...base, key: id.create() }
    : { ...base, key: id.create(), ...Task.WRITE_CHANNEL_OVERRIDE };
  next.port =
    port ??
    lowestFreePort(
      hardware == null ? null : ANALOG_OUTPUT.ports(hardware),
      claimsOf(ANALOG_OUTPUT, channels, hardware),
      (p) => ANALOG_OUTPUT.holds({ ...next, port: p.port }),
    );
  return next;
};

/** Creates a counter channel on the lowest free counter of its device. */
export const createNextCIChannel = (
  channels: CIChannel[],
  { copy, device, port }: Placement = {},
  devices: Device[] = [],
): CIChannel => {
  const next: CIChannel = {
    ...template(channels, createCIChannel(), copy),
    key: id.create(),
    ...Task.READ_CHANNEL_OVERRIDE,
  };
  if (device != null) next.device = device;
  const hardware = hardwareOf(devices, next.device);
  const own = channels.filter((c) => c.device === next.device);
  next.port =
    port ??
    lowestFreePort(
      hardware == null ? null : COUNTER_INPUT.ports(hardware),
      claimsOf(COUNTER_INPUT, own, hardware),
      (p) => COUNTER_INPUT.holds({ ...next, port: p.port }),
    );
  return next;
};

/** Where a new digital channel goes. Each field it leaves out takes a default. */
export interface LinePlacement {
  port?: number;
  line?: number;
  /** The channel whose port to fill without port data. Defaults to the last channel. */
  copy?: string;
}

const createDigitalChannel = <C extends DigitalChannel>(
  channels: C[],
  zero: C,
  direction: "input" | "output",
  placement: LinePlacement,
  device?: Device,
): C => {
  const next = { ...zero, key: id.create() };
  if (placement.port != null && placement.line != null)
    return { ...next, port: placement.port, line: placement.line };
  const taken = lineClaims(channels);
  const hardware = device?.properties.hardware;
  const ports = hardware == null ? [] : digitalPorts(hardware, direction);
  for (const { port, lines } of ports) {
    if (placement.port != null && port !== placement.port) continue;
    for (let line = 0; line < lines; line++)
      if (!taken.has(lineName(port, line))) return { ...next, port, line };
  }
  const source = channels.find(({ key }) => key === placement.copy) ?? channels.at(-1);
  const port = placement.port ?? source?.port ?? next.port;
  let line = 0;
  while (taken.has(lineName(port, line))) line++;
  return { ...next, port, line };
};

/**
 * Creates a digital input channel on the lowest free line of the device. Without port
 * data, it takes the lowest free line of the copied channel's port.
 */
export const createNextDIChannel = (
  channels: DIChannel[],
  placement: LinePlacement = {},
  device?: Device,
): DIChannel =>
  createDigitalChannel(channels, createDIChannel(), "input", placement, device);

/** Creates a digital output channel, as createNextDIChannel does. */
export const createNextDOChannel = (
  channels: DOChannel[],
  placement: LinePlacement = {},
  device?: Device,
): DOChannel =>
  createDigitalChannel(channels, createDOChannel(), "output", placement, device);
