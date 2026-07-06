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
  MAKE,
  type PDOs,
  SLAVE_MODEL,
  SLAVE_SCHEMAS,
  type SlaveDevice,
  type SlaveProperties,
  ZERO_SLAVE_PROPERTIES,
} from "@/feature/ethercat/device/types";
import {
  type InputChannel,
  type OutputChannel,
  ZERO_INPUT_CHANNELS,
  ZERO_OUTPUT_CHANNELS,
} from "@/feature/ethercat/task/types";

type AutoInputChannel = Extract<InputChannel, { type: "automatic" }>;
type ManualInputChannel = Extract<InputChannel, { type: "manual" }>;
type AutoOutputChannel = Extract<OutputChannel, { type: "automatic" }>;
type ManualOutputChannel = Extract<OutputChannel, { type: "manual" }>;

/** Builds a cluster-safe device identifier (2-12 chars, letter first). */
export const createIdentifier = (): string =>
  `s${id.create().replace(/[^a-zA-Z0-9]/g, "")}`.slice(0, 12);

/** Builds a PDO set with one input ("Status") and one output ("Control") entry. */
export const createPDOs = (): PDOs => ({
  inputs: [
    { name: "Status", index: 0x6000, subindex: 1, bitLength: 16, dataType: "uint16" },
  ],
  outputs: [
    { name: "Control", index: 0x7000, subindex: 1, bitLength: 16, dataType: "uint16" },
  ],
});

/** Builds an automatic (PDO-driven) input channel bound to a slave device. */
export const createAutoInputChannel = (
  device: string,
  pdo: string,
  overrides: Partial<AutoInputChannel> = {},
): AutoInputChannel => ({
  ...ZERO_INPUT_CHANNELS.automatic,
  key: id.create(),
  device,
  pdo,
  ...overrides,
});

/** Builds a manual (address-driven) input channel bound to a slave device. */
export const createManualInputChannel = (
  device: string,
  index: number,
  subindex: number,
  overrides: Partial<ManualInputChannel> = {},
): ManualInputChannel => ({
  ...ZERO_INPUT_CHANNELS.manual,
  key: id.create(),
  device,
  index,
  subindex,
  ...overrides,
});

/** Builds an automatic (PDO-driven) output channel bound to a slave device. */
export const createAutoOutputChannel = (
  device: string,
  pdo: string,
  overrides: Partial<AutoOutputChannel> = {},
): AutoOutputChannel => ({
  ...ZERO_OUTPUT_CHANNELS.automatic,
  key: id.create(),
  device,
  pdo,
  ...overrides,
});

/** Builds a manual (address-driven) output channel bound to a slave device. */
export const createManualOutputChannel = (
  device: string,
  index: number,
  subindex: number,
  overrides: Partial<ManualOutputChannel> = {},
): ManualOutputChannel => ({
  ...ZERO_OUTPUT_CHANNELS.manual,
  key: id.create(),
  device,
  index,
  subindex,
  ...overrides,
});

/**
 * Creates an EtherCAT slave device on the live cluster. `properties` are merged over
 * ZERO_SLAVE_PROPERTIES; `properties.name` doubles as the device name when set.
 */
export const createSlaveDevice = async (
  client: Synnax,
  rackKey: number,
  properties: Partial<SlaveProperties> = {},
  configured = true,
): Promise<SlaveDevice> => {
  const key = id.create();
  return await client.devices.create(
    {
      key,
      name: properties.name ?? `EtherCAT Slave ${key}`,
      rack: rackKey,
      location: "test-location",
      make: MAKE,
      model: SLAVE_MODEL,
      configured,
      properties: { ...ZERO_SLAVE_PROPERTIES, ...properties },
    },
    SLAVE_SCHEMAS,
  );
};
