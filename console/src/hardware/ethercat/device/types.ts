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

export const NETWORK_MODEL = "network";
export type NetworkModel = typeof NETWORK_MODEL;

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

/** Network device properties schema. */
export const networkPropertiesZ = z.object({
  interface: z.string(),
  cycleTimeUs: z.number(),
  read: z.object({
    index: z.number(),
    channels: z.record(z.string(), z.number()),
  }),
  write: z.object({
    stateIndex: z.number(),
    channels: z.record(z.string(), z.number()),
  }),
});
export interface NetworkProperties extends z.infer<typeof networkPropertiesZ> {}

export const ZERO_NETWORK_PROPERTIES: NetworkProperties = {
  interface: "",
  cycleTimeUs: 1000,
  read: { index: 0, channels: {} },
  write: { stateIndex: 0, channels: {} },
};

export interface NetworkDevice extends device.Device<
  NetworkProperties,
  Make,
  NetworkModel
> {}

/** Slave device properties schema. */
export const slavePropertiesZ = z.object({
  serial: z.number(),
  vendorId: z.number(),
  productCode: z.number(),
  revision: z.number(),
  name: z.string(),
  position: z.number(),
  pdos: pdosZ,
});
export interface SlaveProperties extends z.infer<typeof slavePropertiesZ> {}

export const ZERO_SLAVE_PROPERTIES: SlaveProperties = {
  serial: 0,
  vendorId: 0,
  productCode: 0,
  revision: 0,
  name: "",
  position: 0,
  pdos: ZERO_PDOS,
};

export interface SlaveDevice extends device.Device<
  SlaveProperties,
  Make,
  SlaveModel
> {}
