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

import {
  type Device,
  MAKE,
  type Model,
  type Properties,
  SCHEMAS,
  T4_MODEL,
  ZERO_PROPERTIES,
} from "@/feature/labjack/device/types";
import {
  type InputChannel,
  type OutputChannel,
  ZERO_INPUT_CHANNELS,
  ZERO_OUTPUT_CHANNELS,
} from "@/feature/labjack/task/types";

type AIChannel = Extract<InputChannel, { type: "AI" }>;
type DIChannel = Extract<InputChannel, { type: "DI" }>;
type TCChannel = Extract<InputChannel, { type: "TC" }>;
type AOChannel = Extract<OutputChannel, { type: "AO" }>;
type DOChannel = Extract<OutputChannel, { type: "DO" }>;

/** Builds a cluster-safe device identifier (2-12 chars, letter first). */
export const createIdentifier = (): string =>
  `l${id.create().replace(/[^a-zA-Z0-9]/g, "")}`.slice(0, 12);

export interface CreateLabJackDeviceOptions {
  model?: Model;
  configured?: boolean;
  properties?: Partial<Properties>;
}

/**
 * Creates a rack and a LabJack device on the live cluster with a unique identifier so
 * channels created from it never collide across runs.
 */
export const createLabJackDevice = async (
  client: Synnax,
  {
    model = T4_MODEL,
    configured = true,
    properties = {},
  }: CreateLabJackDeviceOptions = {},
): Promise<Device> => {
  const rack = await client.racks.create({ name: `lj_rack_${id.create()}` });
  return await client.devices.create(
    {
      key: id.create(),
      name: `lj_dev_${id.create()}`,
      rack: rack.key,
      location: "dev1",
      make: MAKE,
      model,
      configured,
      properties: { ...ZERO_PROPERTIES, identifier: createIdentifier(), ...properties },
    },
    SCHEMAS,
  );
};

/** Builds an analog input channel on the given port. */
export const createAIChannel = (
  port: string,
  overrides: Partial<AIChannel> = {},
): AIChannel => ({ ...ZERO_INPUT_CHANNELS.AI, key: id.create(), port, ...overrides });

/** Builds a digital input channel on the given port. */
export const createDIChannel = (
  port: string,
  overrides: Partial<DIChannel> = {},
): DIChannel => ({ ...ZERO_INPUT_CHANNELS.DI, key: id.create(), port, ...overrides });

/** Builds a thermocouple input channel on the given port. */
export const createTCChannel = (
  port: string,
  overrides: Partial<TCChannel> = {},
): TCChannel => ({ ...ZERO_INPUT_CHANNELS.TC, key: id.create(), port, ...overrides });

/** Builds an analog output channel on the given port. */
export const createAOChannel = (
  port: string,
  overrides: Partial<AOChannel> = {},
): AOChannel => ({
  ...ZERO_OUTPUT_CHANNELS.AO,
  key: id.create(),
  port,
  ...overrides,
  type: "AO",
});

/** Builds a digital output channel on the given port. */
export const createDOChannel = (
  port: string,
  overrides: Partial<DOChannel> = {},
): DOChannel => ({
  ...ZERO_OUTPUT_CHANNELS.DO,
  key: id.create(),
  port,
  ...overrides,
  type: "DO",
});
