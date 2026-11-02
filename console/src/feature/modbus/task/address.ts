// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type ReadChannelType,
  type WriteChannelType,
} from "@/feature/modbus/task/types";

export type ChannelType = ReadChannelType | WriteChannelType;

// The first digit of a six-digit Modicon address names the table.
const TABLE_DIGITS: Record<ChannelType, number> = {
  coil: 0,
  discrete_input: 1,
  input_register: 3,
  input_register_bit: 3,
  holding_register: 4,
  holding_register_bit: 4,
};

const TABLE_NAMES: Record<number, string> = {
  0: "a coil",
  1: "a discrete input",
  3: "an input register",
  4: "a holding register",
};

const MAX_ADDRESS = 65535;

/** @returns the six-digit Modicon form of a zero-based address, such as 400001. */
export const toModicon = (type: ChannelType, address: number): string =>
  `${TABLE_DIGITS[type]}${(address + 1).toString().padStart(5, "0")}`;

export type ParsedAddress = { address: number } | { error: string };

/**
 * Parses an address typed as a zero-based offset or as a six-digit Modicon address. The
 * first digit of a Modicon address must name the table of the channel type.
 */
export const parseAddress = (type: ChannelType, raw: string): ParsedAddress => {
  const text = raw.trim();
  if (!/^\d+$/.test(text)) return { error: "Address must be a whole number" };
  if (text.length === 6) {
    const digit = Number(text[0]);
    const n = Number(text.slice(1));
    const table = TABLE_NAMES[digit];
    if (table == null || n < 1 || n > MAX_ADDRESS + 1)
      return { error: `${text} is not a Modicon address` };
    if (digit !== TABLE_DIGITS[type]) return { error: `${text} is ${table} address` };
    return { address: n - 1 };
  }
  const n = Number(text);
  if (n > MAX_ADDRESS) return { error: `Address must be from 0 to ${MAX_ADDRESS}` };
  return { address: n };
};
