// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";

import { Device } from "@/feature/labjack/device";
import { Task } from "@/feature/labjack/task";

type AnalogReadChannel = Extract<Task.ReadChannel, { type: "analog" }>;
type DigitalReadChannel = Extract<Task.ReadChannel, { type: "digital" }>;
type ThermocoupleReadChannel = Extract<Task.ReadChannel, { type: "thermocouple" }>;
type AnalogWriteChannel = Extract<Task.WriteChannel, { type: "analog" }>;
type DigitalWriteChannel = Extract<Task.WriteChannel, { type: "digital" }>;

/** Builds a Core-safe device identifier (2-12 chars, letter first). */
const createIdentifier = (): string =>
  `l${id.create().replace(/[^a-zA-Z0-9]/g, "")}`.slice(0, 12);

export interface CreateLabJackDeviceOptions {
  model?: Device.Model;
  configured?: boolean;
  properties?: Partial<Device.Properties>;
}

/**
 * Creates a rack and a LabJack device on the live Core with a unique identifier so
 * channels created from it never collide across runs.
 */
export const createLabJackDevice = async (
  client: Synnax,
  {
    model = Device.T4_MODEL,
    configured = true,
    properties = {},
  }: CreateLabJackDeviceOptions = {},
): Promise<Device.Device> => {
  const rack = await client.racks.create({ name: `lj_rack_${id.create()}` });
  return await client.devices.create(
    {
      key: id.create(),
      name: `lj_dev_${id.create()}`,
      rack: rack.key,
      location: "dev1",
      make: Device.MAKE,
      model,
      configured,
      properties: {
        ...Device.ZERO_PROPERTIES,
        identifier: createIdentifier(),
        ...properties,
      },
    },
    Device.SCHEMAS,
  );
};

/** Builds an analog read channel on the given port. */
export const createAnalogReadChannel = (
  port: string,
  overrides: Partial<AnalogReadChannel> = {},
): AnalogReadChannel => ({
  ...Task.createReadChannel("analog"),
  key: id.create(),
  port,
  ...overrides,
});

/** Builds a digital read channel on the given port. */
export const createDigitalReadChannel = (
  port: string,
  overrides: Partial<DigitalReadChannel> = {},
): DigitalReadChannel => ({
  ...Task.createReadChannel("digital"),
  key: id.create(),
  port,
  ...overrides,
});

/** Builds a thermocouple read channel on the given port. */
export const createThermocoupleReadChannel = (
  port: string,
  overrides: Partial<ThermocoupleReadChannel> = {},
): ThermocoupleReadChannel => ({
  ...Task.createReadChannel("thermocouple"),
  key: id.create(),
  port,
  ...overrides,
});

/** Builds an analog write channel on the given port. */
export const createAnalogWriteChannel = (
  port: string,
  overrides: Partial<AnalogWriteChannel> = {},
): AnalogWriteChannel => ({
  ...Task.createWriteChannel("analog"),
  key: id.create(),
  port,
  ...overrides,
  type: "analog",
});

/** Builds a digital write channel on the given port. */
export const createDigitalWriteChannel = (
  port: string,
  overrides: Partial<DigitalWriteChannel> = {},
): DigitalWriteChannel => ({
  ...Task.createWriteChannel("digital"),
  key: id.create(),
  port,
  ...overrides,
  type: "digital",
});
