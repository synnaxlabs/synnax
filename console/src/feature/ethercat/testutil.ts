// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ethercat, type Synnax } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";

import { type Description } from "@/feature/ethercat/device/settings";
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
  READ_CHANNEL_SCHEMAS,
  WRITE_CHANNEL_SCHEMAS,
} from "@/feature/ethercat/task/types";

/** Builds a Core-safe device identifier (2-12 chars, letter first). */
export const createIdentifier = (): string =>
  `s${id.create().replace(/[^a-zA-Z0-9]/g, "")}`.slice(0, 12);

export const STATUS_ENTRY: ethercat.PDOEntry = {
  name: "Status",
  index: 0x6000,
  subIndex: 1,
  bitLength: 16,
  dataType: "uint16",
};

export const CONTROL_ENTRY: ethercat.PDOEntry = {
  name: "Control",
  index: 0x7000,
  subIndex: 1,
  bitLength: 16,
  dataType: "uint16",
};

/** Builds a PDO set with one input ("Status") and one output ("Control") entry. */
export const createPDOs = (): PDOs => ({
  inputs: [STATUS_ENTRY],
  outputs: [CONTROL_ENTRY],
});

/**
 * Builds the PDO set of an eight-channel digital terminal pair (EL1008 and EL2008):
 * eight one-bit entries per direction that share one name, told apart by their PDO
 * names ("Channel 1" to "Channel 8") and addresses (0x6000:01 to 0x6070:01).
 */
export const createRepeatedNamePDOs = (): PDOs => {
  const channels = Array.from({ length: 8 }, (_, i) => i);
  const entry = (name: string, base: number, i: number): ethercat.PDOEntry => ({
    name,
    pdoName: `Channel ${i + 1}`,
    index: base + 0x10 * i,
    subIndex: 1,
    bitLength: 1,
    dataType: "uint8",
  });
  return {
    inputs: channels.map((i) => entry("Input", 0x6000, i)),
    outputs: channels.map((i) => entry("Output", 0x7000, i)),
  };
};

/** The input entry that {@link createSettingsDescription} describes. */
export const STRAIN_ENTRY: ethercat.PDOEntry = {
  name: "Value",
  pdoName: "Channel 1",
  index: 0x6030,
  subIndex: 1,
  bitLength: 32,
  dataType: "int32",
};

/** A buffered layout entry that reads the input of {@link STRAIN_ENTRY}. */
export const STRAIN_SAMPLE_ENTRY: ethercat.PDOEntry = {
  name: "CH 1",
  pdoName: "Buffered channel 1",
  index: 0x6040,
  subIndex: 1,
  bitLength: 32,
  dataType: "uint32",
};

/**
 * Builds the catalog of a strain input that {@link STRAIN_ENTRY} reads, whose
 * excitation polarity picks one of two excitation voltage fields at 0x8040:1C: up to 24
 * V when unipolar (0) and up to 12 V when bipolar (1).
 */
export const createExcitationDescription = (): Description => {
  const polarity = { index: 0x8040, subIndex: 0x21 };
  const voltage = {
    index: 0x8040,
    subIndex: 0x1c,
    name: "Excitation voltage",
    dataType: "float32",
    default: 0,
    unit: "V",
    min: 0,
  };
  return {
    device: [],
    channels: [
      {
        entries: [{ index: STRAIN_ENTRY.index, subIndex: STRAIN_ENTRY.subIndex }],
        settings: [
          {
            ...polarity,
            name: "Excitation polarity",
            dataType: "uint16",
            default: 0,
            unit: "",
            options: [
              { value: 0, name: "Unipolar" },
              { value: 1, name: "Bipolar" },
            ],
            visibleWhen: [],
          },
          { ...voltage, max: 24, visibleWhen: [{ ...polarity, values: [0] }] },
          { ...voltage, max: 12, visibleWhen: [{ ...polarity, values: [1] }] },
        ],
        requires: [],
      },
    ],
  };
};

/**
 * Builds the catalog fields of a strain input that {@link STRAIN_ENTRY} and
 * {@link STRAIN_SAMPLE_ENTRY} read. A voltage input picks one of five ranges, a current
 * input one of two. A bridge shows an excitation voltage from 0 to 24 V and needs one
 * above 0.
 */
export const createSettingsDescription = (): Description => {
  const measurement = { index: 0x8040, subIndex: 1 };
  const ranges = ["50 V", "10 V", "3 V", "1 V", "0.1 V"];
  return {
    device: [
      {
        index: 0x8020,
        subIndex: 1,
        name: "Sample rate",
        dataType: "uint16",
        default: 4,
        unit: "",
        options: [
          { value: 4, name: "20 kS/s" },
          { value: 12, name: "1 kS/s" },
        ],
        visibleWhen: [],
      },
    ],
    channels: [
      {
        entries: [STRAIN_ENTRY, STRAIN_SAMPLE_ENTRY].map(({ index, subIndex }) => ({
          index,
          subIndex,
        })),
        settings: [
          {
            ...measurement,
            name: "Measurement",
            dataType: "uint16",
            default: 0,
            unit: "",
            options: [
              { value: 0, name: "Voltage" },
              { value: 1, name: "Current" },
              { value: 6, name: "Bridge" },
            ],
            visibleWhen: [],
          },
          {
            index: 0x8040,
            subIndex: 3,
            name: "Range",
            dataType: "uint16",
            default: 0,
            unit: "",
            options: ranges.map((name, value) => ({ value, name })),
            visibleWhen: [{ ...measurement, values: [0, 6] }],
          },
          {
            index: 0x8040,
            subIndex: 3,
            name: "Range",
            dataType: "uint16",
            default: 0,
            unit: "",
            options: [
              { value: 0, name: "20 mA" },
              { value: 1, name: "2 mA" },
            ],
            visibleWhen: [{ ...measurement, values: [1] }],
          },
          {
            index: 0x8040,
            subIndex: 0x1c,
            name: "Excitation voltage",
            dataType: "float32",
            default: 0,
            unit: "V",
            min: 0,
            max: 24,
            visibleWhen: [{ ...measurement, values: [6] }],
          },
        ],
        requires: [
          {
            when: [{ ...measurement, values: [6] }],
            then: { index: 0x8040, subIndex: 0x1c, above: 0 },
            message: "A bridge needs an excitation voltage above 0 V.",
          },
        ],
      },
    ],
  };
};

/**
 * Builds the catalog fields of {@link createSettingsDescription}, with the gain at
 * 0x8040:15 and the offset at 0x8040:14 calibrating the samples of
 * {@link STRAIN_ENTRY}.
 */
export const createCalibratedDescription = (): Description => {
  const description = createSettingsDescription();
  description.channels[0].entries[0].calibration = {
    dataType: "int32",
    index: 0x8040,
    gainSubIndex: 0x15,
    offsetSubIndex: 0x14,
    divisor: 1,
    rawOffset: false,
  };
  return description;
};

/**
 * Builds the catalog of {@link createSettingsDescription} for a card in which only
 * DewesoftX reads {@link STRAIN_SAMPLE_ENTRY}. A shared card is one that DewesoftX also
 * sets up, and a scan gives it the write that switches it to Synnax only.
 */
export const createDewesoftXSettings = (
  shared: boolean,
  bus: 1 | 2 = 1,
): NonNullable<SlaveProperties["settings"]> => {
  const description = createSettingsDescription();
  return {
    ...description,
    channels: description.channels.map((channel) => ({
      ...channel,
      entries: channel.entries.map((entry) =>
        entry.index === STRAIN_SAMPLE_ENTRY.index &&
        entry.subIndex === STRAIN_SAMPLE_ENTRY.subIndex
          ? { ...entry, onlyReadBy: "DewesoftX" }
          : entry,
      ),
    })),
    mode: {
      index: 0x8020,
      subIndex: 9,
      value: shared ? 1 : 0,
      shared,
      ...(shared ? { sharedWith: "DewesoftX" } : {}),
      bus,
      switch: shared
        ? [{ index: 0x8020, subIndex: 9, dataType: "uint16", value: 0, name: "" }]
        : [],
    },
  };
};

/** Builds an input channel whose entry name matched no single entry. */
export const createUnresolvedReadChannel = (
  device: string,
  pdo: string,
  overrides: Partial<ethercat.UnresolvedReadChannel> = {},
): ethercat.UnresolvedReadChannel =>
  READ_CHANNEL_SCHEMAS.unresolved.parse({
    type: "unresolved",
    device,
    pdo,
    ...overrides,
  });

/** Builds an output channel whose entry name matched no single entry. */
export const createUnresolvedWriteChannel = (
  device: string,
  pdo: string,
  overrides: Partial<ethercat.UnresolvedWriteChannel> = {},
): ethercat.UnresolvedWriteChannel =>
  WRITE_CHANNEL_SCHEMAS.unresolved.parse({
    type: "unresolved",
    device,
    pdo,
    ...overrides,
  });

/**
 * Creates an EtherCAT slave device on the live Core. `properties` are merged over
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

/** Creates a slave on eth0 whose eight channels repeat the same entry names. */
export const createRepeatedNameSlave = async (
  client: Synnax,
  rackKey: number,
): Promise<SlaveDevice> =>
  await createSlaveDevice(client, rackKey, {
    identifier: createIdentifier(),
    network: "eth0",
    pdos: createRepeatedNamePDOs(),
  });
