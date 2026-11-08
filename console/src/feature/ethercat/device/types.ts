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
export const INTEGRATION = "ethercat";

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
    .object({ channels: z.record(z.string(), z.number()).default(() => ({})) })
    .default(() => ({ channels: {} })),
  write: z
    .object({ channels: z.record(z.string(), z.number()).default(() => ({})) })
    .default(() => ({ channels: {} })),
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
  read: { channels: {} },
  write: { channels: {} },
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
