// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { z } from "zod/v4";

export const PREFIX = "ethercat";

export const MAKE = PREFIX;
export type Make = typeof MAKE;

export const SLAVE_MODEL = "slave";
export type SlaveModel = typeof SLAVE_MODEL;

/** Schema for a single PDO entry from device scan. */
export const pdoEntryZ = z.object({
  name: z.string(),
  index: z.number(),
  subindex: z.number(),
  bitLength: z.number(),
  dataType: z.string(),
});
export interface PDOEntry extends z.infer<typeof pdoEntryZ> {}

/** Schema for PDO collections (inputs and outputs). */
export const pdosZ = z.object({
  inputs: z.array(pdoEntryZ),
  outputs: z.array(pdoEntryZ),
});
export interface PDOs extends z.infer<typeof pdosZ> {}

export const ZERO_PDOS: PDOs = {
  inputs: [],
  outputs: [],
};

/** Slave device properties schema. */
export const slavePropertiesZ = z.object({
  serial: z.number(),
  vendorId: z.number(),
  productCode: z.number(),
  revision: z.number(),
  name: z.string(),
  network: z.string(),
  position: z.number(),
  pdos: pdosZ,
  readIndex: z.number(),
  writeStateIndex: z.number(),
  read: z.object({
    channels: z.record(z.string(), z.number()),
  }),
  write: z.object({
    channels: z.record(z.string(), z.number()),
  }),
});
export interface SlaveProperties extends z.infer<typeof slavePropertiesZ> {}

export const ZERO_SLAVE_PROPERTIES: SlaveProperties = {
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
};

export interface SlaveDevice extends device.Device<SlaveProperties, Make, SlaveModel> {}
