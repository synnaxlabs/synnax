// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType } from "@synnaxlabs/x";

import {
  formatIndex,
  type SlaveProperties,
  type StartupParameter,
} from "@/feature/ethercat/device/types";

const DATA_TYPES: Record<number, DataType> = {
  1: DataType.UINT8,
  2: DataType.UINT16,
  4: DataType.UINT32,
  8: DataType.UINT64,
};

// The Driver writes startup parameters in PRE-OP, before the move to SAFE-OP.
const PRE_SAFE_OP_TRANSITIONS = new Set(["IP", "PS"]);

/** Parses an ESI HexDecValue, such as "#x8020" or "32800". */
const parseHexDec = (text: string | null | undefined): number | null => {
  if (text == null) return null;
  const trimmed = text.trim();
  const value = /^#x/i.test(trimmed)
    ? Number.parseInt(trimmed.slice(2), 16)
    : Number.parseInt(trimmed, 10);
  return Number.isNaN(value) ? null : value;
};

const childText = (el: Element, tag: string): string | null =>
  Array.from(el.children).find((c) => c.tagName === tag)?.textContent ?? null;

const parseCommand = (cmd: Element): StartupParameter | null => {
  const transitions = Array.from(cmd.children)
    .filter((c) => c.tagName === "Transition")
    .map((c) => c.textContent?.trim() ?? "");
  if (!transitions.some((t) => PRE_SAFE_OP_TRANSITIONS.has(t))) return null;
  if (cmd.getAttribute("CompleteAccess") === "1") return null;
  const index = parseHexDec(childText(cmd, "Index"));
  const subIndex = parseHexDec(childText(cmd, "SubIndex"));
  const data = childText(cmd, "Data")?.trim() ?? "";
  const dataType = DATA_TYPES[data.length / 2];
  if (
    index == null ||
    subIndex == null ||
    dataType == null ||
    !/^[0-9a-f]+$/i.test(data)
  )
    return null;
  // ESI data is the object's bytes in EtherCAT (little-endian) order.
  let value = 0;
  for (let i = data.length - 2; i >= 0; i -= 2)
    value = value * 256 + Number.parseInt(data.slice(i, i + 2), 16);
  return {
    index,
    subIndex,
    dataType: dataType.toString(),
    value,
    name: childText(cmd, "Comment")?.trim() ?? "",
  };
};

export interface StartupCommands {
  /** The name of the matching device in the ESI file. */
  device: string;
  /** The CoE startup commands that the Driver can write before SAFE-OP. */
  parameters: StartupParameter[];
  /** The number of startup commands that the Driver cannot write. */
  skipped: number;
}

/**
 * Reads the CoE startup commands (InitCmds) of the slave's device from an ESI file.
 * The Driver writes the commands of the INIT to PRE-OP and PRE-OP to SAFE-OP
 * transitions before SAFE-OP, so other transitions count as skipped, as do commands
 * with complete access or data that is not 1, 2, 4, or 8 bytes.
 * @throws {Error} if the file is not XML or lists no device with the slave's product
 * code and revision.
 */
export const parseStartupCommands = (
  xml: string,
  { productCode, revision }: Pick<SlaveProperties, "productCode" | "revision">,
): StartupCommands => {
  const doc = new DOMParser().parseFromString(xml, "application/xml");
  if (doc.getElementsByTagName("parsererror").length > 0)
    throw new Error("The file is not a valid ESI XML file.");
  const devices = Array.from(doc.getElementsByTagName("Device"));
  const types = devices.map((d) =>
    Array.from(d.children).find((c) => c.tagName === "Type"),
  );
  const found = devices.find((_, i) => {
    const type = types[i];
    return (
      type != null &&
      parseHexDec(type.getAttribute("ProductCode")) === productCode &&
      parseHexDec(type.getAttribute("RevisionNo")) === revision
    );
  });
  if (found == null) {
    const names = [...new Set(types.map((t) => t?.textContent?.trim() ?? ""))].filter(
      (n) => n !== "",
    );
    const listed =
      names.length > 3
        ? `${names.slice(0, 3).join(", ")}, and others`
        : names.join(", ");
    throw new Error(
      `The ESI file lists no device with product code ${formatIndex(productCode)} ` +
        `and revision 0x${revision.toString(16).toUpperCase().padStart(8, "0")}. ` +
        `It is for ${listed}.`,
    );
  }
  const commands = Array.from(found.getElementsByTagName("InitCmd")).filter(
    (cmd) => cmd.parentElement?.tagName === "CoE",
  );
  const parameters = commands
    .map(parseCommand)
    .filter((p): p is StartupParameter => p != null);
  return {
    device: types[devices.indexOf(found)]?.textContent?.trim() ?? "",
    parameters,
    skipped: commands.length - parameters.length,
  };
};

/**
 * Merges imported parameters into existing ones. An imported parameter replaces an
 * existing one with the same index and sub-index.
 */
export const mergeStartupParameters = (
  existing: StartupParameter[],
  imported: StartupParameter[],
): StartupParameter[] => {
  const key = ({ index, subIndex }: StartupParameter) => `${index}:${subIndex}`;
  const replaced = new Map(imported.map((p) => [key(p), p]));
  const kept = existing.map((p) => replaced.get(key(p)) ?? p);
  const keptKeys = new Set(existing.map(key));
  return [...kept, ...imported.filter((p) => !keptKeys.has(key(p)))];
};
