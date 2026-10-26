// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { MultipleFoundError, NotFoundError } from "@/errors";

/** A PDO entry that a scan discovered on an EtherCAT slave. */
export const pdoEntryZ = z.object({
  /** name is the entry name, such as "Input". */
  name: z.string(),
  /** pdoName is the name of the PDO that holds the entry, such as "Channel 1". */
  pdoName: z.string().optional(),
  /** pdoIndex is the index of the PDO that holds the entry, such as 0x1A00. */
  pdoIndex: z.number().optional(),
  /** index is the CoE object index. */
  index: z.number(),
  /** subIndex is the CoE object sub-index. */
  subIndex: z.number(),
  /** bitLength is the size of the entry in bits. */
  bitLength: z.number(),
  /** dataType is the data type of the entry, such as "uint8". */
  dataType: z.string(),
});
export interface PDOEntry extends z.infer<typeof pdoEntryZ> {}

const hex = (value: number, width: number): string =>
  value.toString(16).toUpperCase().padStart(width, "0");

/** @returns the address in CoE notation, such as 0x6010:01. */
export const formatAddress = ({
  index,
  subIndex,
}: Pick<PDOEntry, "index" | "subIndex">): string =>
  `0x${hex(index, 4)}:${hex(subIndex, 2)}`;

/** @returns the address of the entry followed by its PDO name when known. */
export const formatEntry = (
  entry: Pick<PDOEntry, "index" | "subIndex" | "pdoName">,
): string =>
  entry.pdoName == null || entry.pdoName === ""
    ? formatAddress(entry)
    : `${formatAddress(entry)} (${entry.pdoName})`;

/**
 * Finds the entry with the given name. Entries that share a CoE address are the same
 * object, so they count as one match.
 *
 * @param entries - The entries of one slave in one direction.
 * @param name - The entry name.
 * @param pdoName - The name of the PDO that holds the entry. Necessary when more than
 * one PDO holds an entry with the same name.
 * @returns The first entry at the matching address.
 * @throws {NotFoundError} if no entry matches.
 * @throws {MultipleFoundError} if entries at different addresses match.
 */
export const findEntry = (
  entries: PDOEntry[],
  name: string,
  pdoName?: string,
): PDOEntry => {
  const matches = new Map<string, PDOEntry>();
  for (const entry of entries) {
    if (entry.name !== name || (pdoName != null && entry.pdoName !== pdoName)) continue;
    const key = `${entry.index}:${entry.subIndex}`;
    if (!matches.has(key)) matches.set(key, entry);
  }
  const target =
    pdoName == null ? `entry "${name}"` : `entry "${name}" in PDO "${pdoName}"`;
  if (matches.size === 0) throw new NotFoundError(`no ${target}`);
  if (matches.size > 1) {
    const found = Array.from(matches.values(), formatEntry).join(", ");
    const hint = pdoName == null ? ". Pass pdoName to pick one" : "";
    throw new MultipleFoundError(`${target} matches ${found}${hint}`);
  }
  const [entry] = matches.values();
  return entry;
};
