// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, ethercat } from "@synnaxlabs/client";
import { z } from "zod";

export const MAKE = "ethercat";

/** The name a rack lists in its integrations when its Driver runs this one. */
export const INTEGRATION_KEY = "ethercat";

export const makeZ = z.literal(MAKE);

export const SLAVE_MODEL = "slave";
export const modelZ = z.literal(SLAVE_MODEL);

/** Schema for PDO collections (inputs and outputs). */
const pdosZ = z.object({
  inputs: z.array(ethercat.pdoEntryZ).default(() => []),
  outputs: z.array(ethercat.pdoEntryZ).default(() => []),
});
export interface PDOs extends z.infer<typeof pdosZ> {}

/** A synchronization mode that the slave lists in its SII. */
const opModeZ = z.object({
  name: z.string().default(""),
  assignActivate: z.number(),
  sync0Cycle: z.number(),
  sync0Factor: z.number(),
  sync0Shift: z.number(),
  sync1Factor: z.number(),
  sync1Shift: z.number(),
});
export interface OpMode extends z.infer<typeof opModeZ> {}

/**
 * The mode that the user chose for a slave: its index in opModes, its AssignActivate,
 * and SYNC0 and SYNC1 cycles in nanoseconds. A cycle of 0 means the cycle of the mode.
 */
const opModeChoiceZ = z.object({
  index: z.number(),
  assignActivate: z.number(),
  sync0Cycle: z.number().default(0),
  sync1Cycle: z.number().default(0),
});
export interface OpModeChoice extends z.infer<typeof opModeChoiceZ> {}
/** A CoE object the Driver writes to the slave before each move to SAFE-OP. */
const startupParameterZ = z.object({
  index: z.number(),
  subIndex: z.number(),
  dataType: z.string(),
  value: z.number(),
  name: z.string().default(""),
});
export interface StartupParameter extends z.infer<typeof startupParameterZ> {}

/** The Synnax channel that a slave maps for an input entry. */
const readBindingZ = z.object({
  index: z.number(),
  subIndex: z.number(),
  /** The channel holds the scaled samples of the entry. */
  scaled: z.boolean().default(false),
  channel: z.number(),
});
export interface ReadBinding extends z.infer<typeof readBindingZ> {}

/** The Synnax channels that a slave maps for an output entry. 0 is no channel. */
const writeBindingZ = z.object({
  index: z.number(),
  subIndex: z.number(),
  command: z.number().default(0),
  state: z.number().default(0),
});
export interface WriteBinding extends z.infer<typeof writeBindingZ> {}

/** The CoE index and sub-index of a PDO entry. */
export interface Address {
  index: number;
  subIndex: number;
}

/**
 * Reads bindings stored as a map keyed by the address of the entry, such as 24592_1.
 * prefix is the optional prefix of a key for the second channel of the entry. A key
 * that names no address holds no binding and is dropped.
 */
const bindingMapZ = <B>(
  prefix: string,
  add: (bindings: B[], address: Address, prefixed: boolean, channel: number) => void,
) => {
  const keyPattern = new RegExp(`^(${prefix}_)?(\\d+)_(\\d+)$`);
  return z.record(z.string(), z.number()).transform((map) => {
    const bindings: B[] = [];
    for (const [key, channel] of Object.entries(map)) {
      const match = keyPattern.exec(key);
      if (match == null) continue;
      const address = { index: Number(match[2]), subIndex: Number(match[3]) };
      add(bindings, address, match[1] != null, channel);
    }
    return bindings;
  });
};

const readBindingsZ = z
  .array(readBindingZ)
  .or(
    bindingMapZ<ReadBinding>("scaled", (bindings, address, scaled, channel) =>
      bindings.push({ ...address, scaled, channel }),
    ),
  );

const writeBindingsZ = z.array(writeBindingZ).or(
  bindingMapZ<WriteBinding>("state", (bindings, address, state, channel) => {
    let binding = bindings.find(
      (b) => b.index === address.index && b.subIndex === address.subIndex,
    );
    if (binding == null) {
      binding = { ...address, command: 0, state: 0 };
      bindings.push(binding);
    }
    if (state) binding.state = channel;
    else binding.command = channel;
  }),
);

const ZERO_PDOS: PDOs = {
  inputs: [],
  outputs: [],
};

/** Slave device properties schema. */
export const slavePropertiesZ = z.object({
  identifier: z.string().default(""),
  serial: z.number().default(0),
  vendorId: z.number().default(0),
  productCode: z.number().default(0),
  revision: z.number().default(0),
  name: z.string().default(""),
  network: z.string().default(""),
  position: z.number().default(0),
  pdos: pdosZ.default(() => ({ inputs: [], outputs: [] })),
  readIndex: z.number().default(0),
  writeStateIndex: z.number().default(0),
  read: z
    .object({ channels: readBindingsZ.default(() => []) })
    .default(() => ({ channels: [] })),
  write: z
    .object({ channels: writeBindingsZ.default(() => []) })
    .default(() => ({ channels: [] })),
  enabled: z.boolean().optional().default(true),
  opModes: z.array(opModeZ).default(() => []),
  /** The mode the Driver sets up. Unset means the Driver picks it. */
  opMode: opModeChoiceZ.optional(),
  startupParameters: z.array(startupParameterZ).default(() => []),
});
export interface SlaveProperties extends z.infer<typeof slavePropertiesZ> {}

export const ZERO_SLAVE_PROPERTIES: SlaveProperties = {
  identifier: "",
  serial: 0,
  vendorId: 0,
  productCode: 0,
  revision: 0,
  name: "",
  network: "",
  position: 0,
  pdos: ZERO_PDOS,
  readIndex: 0,
  writeStateIndex: 0,
  read: { channels: [] },
  write: { channels: [] },
  enabled: true,
  opModes: [],
  startupParameters: [],
};

export interface SlaveDevice extends device.Device<
  typeof slavePropertiesZ,
  typeof makeZ,
  typeof modelZ
> {}

export const SLAVE_SCHEMAS = {
  properties: slavePropertiesZ,
  make: makeZ,
  model: modelZ,
} as const satisfies device.DeviceSchemas<
  typeof slavePropertiesZ,
  typeof makeZ,
  typeof modelZ
>;
