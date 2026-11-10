// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ethercat } from "@synnaxlabs/client";
import { DataType } from "@synnaxlabs/x";

import { type Field, sameAddress } from "@/feature/ethercat/device/settings";
import { type SlaveProperties } from "@/feature/ethercat/device/types";

/** The type of a command on an object that the catalog does not describe. */
const UNSIGNED: Record<number, DataType> = {
  1: DataType.UINT8,
  2: DataType.UINT16,
  4: DataType.UINT32,
  8: DataType.UINT64,
};

/** Reads a value of each type from the little-endian bytes of a command. */
const READERS: Record<string, (view: DataView) => number> = {
  [DataType.UINT8.toString()]: (v) => v.getUint8(0),
  [DataType.INT8.toString()]: (v) => v.getInt8(0),
  [DataType.UINT16.toString()]: (v) => v.getUint16(0, true),
  [DataType.INT16.toString()]: (v) => v.getInt16(0, true),
  [DataType.UINT32.toString()]: (v) => v.getUint32(0, true),
  [DataType.INT32.toString()]: (v) => v.getInt32(0, true),
  [DataType.UINT64.toString()]: (v) => Number(v.getBigUint64(0, true)),
  [DataType.INT64.toString()]: (v) => Number(v.getBigInt64(0, true)),
  [DataType.FLOAT32.toString()]: (v) => v.getFloat32(0, true),
  [DataType.FLOAT64.toString()]: (v) => v.getFloat64(0, true),
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

/** @returns the type of the catalog field at the address when its size is size. */
const catalogType = (
  fields: Field[],
  address: Pick<ethercat.StartupParameter, "index" | "subIndex">,
  size: number,
): DataType | null => {
  const field = fields.find((f) => sameAddress(f, address));
  if (field == null) return null;
  const type = new DataType(field.dataType);
  return type.density.valueOf() === size ? type : null;
};

const parseCommand = (
  cmd: Element,
  fields: Field[],
): ethercat.StartupParameter | null => {
  const transitions = Array.from(cmd.children)
    .filter((c) => c.tagName === "Transition")
    .map((c) => c.textContent?.trim() ?? "");
  if (!transitions.some((t) => PRE_SAFE_OP_TRANSITIONS.has(t))) return null;
  if (cmd.getAttribute("CompleteAccess") === "1") return null;
  const index = parseHexDec(childText(cmd, "Index"));
  const subIndex = parseHexDec(childText(cmd, "SubIndex"));
  const data = childText(cmd, "Data")?.trim() ?? "";
  if (index == null || subIndex == null || !/^([0-9a-f]{2})+$/i.test(data)) return null;
  const bytes = Uint8Array.from(data.match(/../g) ?? [], (b) => Number.parseInt(b, 16));
  const parameter = { index, subIndex, name: childText(cmd, "Comment")?.trim() ?? "" };
  const dataType =
    catalogType(fields, parameter, bytes.length) ?? UNSIGNED[bytes.length];
  if (dataType == null) return null;
  const value = READERS[dataType.toString()](new DataView(bytes.buffer));
  if (!Number.isFinite(value) || (dataType.isInteger && !Number.isSafeInteger(value)))
    return null;
  return { ...parameter, dataType: dataType.toString(), value };
};

export interface StartupCommands {
  /** The name of the matching device in the ESI file. */
  device: string;
  /** The commands to add as startup parameters. */
  parameters: ethercat.StartupParameter[];
  /** The number of commands that the Driver cannot write before SAFE-OP. */
  unwritableCount: number;
  /** The commands on the object that holds the slave's mode, which Synnax sets. */
  modeCommands: ethercat.StartupParameter[];
}

const DECLARED_ENCODING = /<\?xml[^>]*\bencoding\s*=\s*["']([^"']+)["']/;

/** @returns a decoder for the encoding, or null if the platform cannot decode it. */
const decoderOf = (encoding: string): TextDecoder | null => {
  try {
    return new TextDecoder(encoding);
  } catch {
    return null;
  }
};

const BYTE_ORDER_MARKS = [
  { bytes: [0xef, 0xbb, 0xbf], encoding: "utf-8" },
  { bytes: [0xff, 0xfe], encoding: "utf-16le" },
  { bytes: [0xfe, 0xff], encoding: "utf-16be" },
];

/**
 * @returns the text of an XML file, read in the encoding of its byte order mark, or
 * else in the encoding that its declaration names.
 * @throws {Error} if the declaration names an unknown encoding.
 */
const decode = (file: Uint8Array): string => {
  const mark = BYTE_ORDER_MARKS.find(({ bytes }) =>
    bytes.every((b, i) => file[i] === b),
  );
  if (mark != null) return new TextDecoder(mark.encoding).decode(file);
  const head = new TextDecoder("windows-1252").decode(file.subarray(0, 256));
  const encoding = DECLARED_ENCODING.exec(head)?.[1] ?? "utf-8";
  const decoder = decoderOf(encoding);
  if (decoder == null)
    throw new Error(
      `The ESI file uses ${encoding}, an encoding that Synnax cannot read.`,
    );
  return decoder.decode(file);
};

/**
 * Reads the CoE startup commands (InitCmds) of the slave's device from an ESI file.
 * The Driver writes the commands of the INIT to PRE-OP and PRE-OP to SAFE-OP
 * transitions, so other transitions are unwritable, as are commands with complete
 * access, data that is not 1, 2, 4, or 8 bytes, or a value that a number cannot hold.
 * A command on an object that the catalog of the slave describes takes the type of
 * that field.
 * @throws {Error} if the file is not XML, uses an unknown encoding, is for another
 * vendor, or lists no device with the slave's product code and revision.
 */
export const parseStartupCommands = (
  file: Uint8Array,
  props: SlaveProperties,
): StartupCommands => {
  const doc = new DOMParser().parseFromString(decode(file), "application/xml");
  if (doc.getElementsByTagName("parsererror").length > 0)
    throw new Error("The file is not a valid ESI XML file.");
  const vendor = parseHexDec(
    doc.querySelector("EtherCATInfo > Vendor > Id")?.textContent,
  );
  if (vendor !== props.vendorId)
    throw new Error(
      `The ESI file is not for vendor ${ethercat.formatHex(props.vendorId, 8)}, the ` +
        "vendor of the device.",
    );
  const devices = Array.from(doc.getElementsByTagName("Device")).flatMap((device) => {
    const type = Array.from(device.children).find((c) => c.tagName === "Type");
    return type == null ? [] : [{ device, type }];
  });
  const found = devices.find(
    ({ type }) =>
      parseHexDec(type.getAttribute("ProductCode")) === props.productCode &&
      parseHexDec(type.getAttribute("RevisionNo")) === props.revision,
  );
  if (found == null) {
    const names = [
      ...new Set(devices.map(({ type }) => type.textContent?.trim() ?? "")),
    ].filter((n) => n !== "");
    const listed =
      names.length > 3
        ? `${names.slice(0, 3).join(", ")}, and others`
        : names.join(", ");
    const lists = names.length === 0 ? "It lists no devices." : `It is for ${listed}.`;
    throw new Error(
      `The ESI file lists no device with product code ` +
        `${ethercat.formatHex(props.productCode, 4)} and revision ` +
        `${ethercat.formatHex(props.revision, 8)}. ${lists}`,
    );
  }
  const commands = Array.from(found.device.getElementsByTagName("InitCmd")).filter(
    (cmd) => cmd.parentElement?.tagName === "CoE",
  );
  const { settings } = props;
  const fields =
    settings == null
      ? []
      : [...settings.device, ...settings.channels.flatMap((c) => c.settings)];
  const writable = commands
    .map((cmd) => parseCommand(cmd, fields))
    .filter((p): p is ethercat.StartupParameter => p != null);
  const holdsMode = (p: ethercat.StartupParameter) =>
    settings?.mode != null && sameAddress(p, settings.mode);
  return {
    device: found.type.textContent?.trim() ?? "",
    parameters: writable.filter((p) => !holdsMode(p)),
    unwritableCount: commands.length - writable.length,
    modeCommands: writable.filter(holdsMode),
  };
};

/**
 * Merges imported parameters into existing ones. The imported parameters replace the
 * existing ones on the same objects, and come last in the order of the file.
 */
export const mergeStartupParameters = (
  existing: ethercat.StartupParameter[],
  imported: ethercat.StartupParameter[],
): ethercat.StartupParameter[] => [
  ...existing.filter((p) => !imported.some((i) => sameAddress(i, p))),
  ...imported,
];
