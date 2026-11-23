// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// The LabJack T7 enclosure in millimeters, from the LabJack T-series datasheet. x runs
// along the length from the communication end, y runs back from the terminal face, and
// z runs up from the feet.

export const LENGTH = 188;
export const DEPTH = 74.9;
export const HEIGHT = 31.6;
/** The seam between the lid and the base. */
export const PARTING = 15.8;

/** The recess along the terminal face that holds the screw terminal blocks. */
export const POCKET = { x0: 23, x1: 164, depth: 13, floor: PARTING };

/** The extent of each terminal block. Its top sits just under the lid. */
export const BLOCK = { depth: 12, top: HEIGHT - 0.6 };

/** Where the screw of a terminal sits on the block top. */
export const SCREW = { y: 7.5, radius: 1.5 };

/** The height of the wire entry on the front of a terminal block. */
export const ENTRY = { z: PARTING + 3.8, size: 2.6 };

const PITCH = 5.08;
const BLOCK_PITCH = 23.24;
const FIRST_BLOCK = 25;

const LABELS = [
  ["200UA", "10UA", "SGND", "SPC"],
  ["FIO3", "FIO2", "GND", "VS"],
  ["FIO1", "FIO0", "GND", "VS"],
  ["DAC1", "DAC0", "GND", "VS"],
  ["AIN3", "AIN2", "GND", "VS"],
  ["AIN1", "AIN0", "GND", "VS"],
];

export interface Terminal {
  label: string;
  x: number;
}

export interface Block {
  x0: number;
  x1: number;
  terminals: Terminal[];
}

/** The six terminal blocks, from the communication end to the mounting end. */
export const BLOCKS: Block[] = LABELS.map((labels, b) => {
  const x0 = FIRST_BLOCK + b * BLOCK_PITCH;
  return {
    x0,
    x1: x0 + labels.length * PITCH,
    terminals: labels.map((label, i) => ({ label, x: x0 + (i + 0.5) * PITCH })),
  };
});

/** The terminals that the thermocouple in the post connects to. */
export const AIN0 = BLOCKS[5].terminals[1];
export const GND = BLOCKS[5].terminals[2];

/**
 * The two U-shaped mounting notches at the far end. Each one is cut from the lid down
 * to a ledge that holds a screw hole.
 */
export const NOTCHES = [18.4, 56.5].map((y) => ({
  x: LENGTH - 9.4,
  y,
  radius: 5.5,
  ledge: 10,
  hole: 2.15,
}));

/** The COMM and STATUS lights on the terminal face. */
export const LIGHTS = [11.6, 20.1].map((x) => ({ x, z: 18.6, radius: 1.5 }));
