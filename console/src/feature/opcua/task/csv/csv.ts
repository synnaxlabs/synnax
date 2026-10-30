// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { csv, DataType } from "@synnaxlabs/x";

import {
  type Channel,
  nodeIdentity,
  type ScannedNode,
} from "@/feature/opcua/task/types";

const COLUMNS = ["name", "node_name", "node_id", "field", "data_type", "enabled"];

const INDEX_COLUMN = "index";

const DATA_TYPES = new Set(
  DataType.ALL.filter((dt) => !dt.equals(DataType.UNKNOWN)).map((dt) => dt.toString()),
);

// The text after the identifier type, such as Motor in ns=2;s=Motor.
const NODE_ID_IDENTIFIER = /^(?:nsu?=[^;]*;)?[isgb]=(.+)$/su;

export interface EncodeParams {
  /** True when a channel can be the task's index, which adds the index column. */
  indexed?: boolean;
}

/** @returns the channels as CSV text: a header, then one row per channel. */
export const encode = (
  channels: Channel[],
  { indexed = false }: EncodeParams = {},
): string => {
  const header = indexed ? [...COLUMNS, INDEX_COLUMN] : COLUMNS;
  const rows = channels.map((ch) => {
    const row = [
      ch.name,
      ch.nodeName,
      ch.nodeId,
      "field" in ch ? ch.field : "",
      ch.dataType,
      String(!ch.disabled),
    ];
    if (indexed) row.push(String("isIndex" in ch && ch.isIndex));
    return row;
  });
  const lines = [header, ...rows].map((row) => row.map(csv.formatValue).join(","));
  return `${lines.join("\n")}\n`;
};

export interface DecodeParams<C extends Channel> extends EncodeParams {
  /** The channels the task already holds. */
  existing: C[];
  /** @returns the task's channel for a node, or null if the task cannot hold it. */
  convert: (node: ScannedNode) => C | null;
}

const parseBoolean = (value: string, column: string, row: number): boolean | null => {
  if (value === "") return null;
  const lower = value.toLowerCase();
  if (lower === "true" || lower === "1") return true;
  if (lower === "false" || lower === "0") return false;
  throw new Error(`Row ${row} has ${column} "${value}", which is not true or false.`);
};

const parseDataType = (value: string, row: number): string => {
  if (value === "") return DataType.FLOAT32.toString();
  const lower = value.toLowerCase();
  if (!DATA_TYPES.has(lower))
    throw new Error(
      `Row ${row} has data_type "${value}", which is not a Synnax data type.`,
    );
  return lower;
};

const nodeName = (nodeId: string, field: string): string => {
  const name = NODE_ID_IDENTIFIER.exec(nodeId)?.[1] ?? nodeId;
  return field === "" ? name : `${name}.${field}`;
};

/**
 * Decodes comma or tab delimited text into the channels to add to a task. Header names
 * ignore case, and unknown columns are ignored.
 * @returns a channel for each row, except rows for a node and field that the task or an
 * earlier row holds, and rows that `convert` rejects.
 * @throws {Error} if the text has no node_id column, or a row has no node ID, an
 * unknown data type, or a boolean other than true, false, 1, or 0. The message names
 * the row, with the header as row 1.
 */
export const decode = <C extends Channel>(
  text: string,
  { existing, convert, indexed = false }: DecodeParams<C>,
): C[] => {
  const [header = [], ...rows] = csv.parseBlock(text);
  const columns = header.map((name) => name.toLowerCase());
  if (!columns.includes("node_id"))
    throw new Error("The CSV file has no node_id column.");
  const identities = new Set(existing.map(nodeIdentity));
  const channels: C[] = [];
  rows.forEach((cells, i) => {
    const row = i + 2;
    const cell = (column: string) => {
      const c = columns.indexOf(column);
      return c === -1 ? "" : (cells[c] ?? "");
    };
    const nodeId = cell("node_id");
    if (nodeId === "") throw new Error(`Row ${row} has no node_id.`);
    const field = cell("field");
    const dataType = parseDataType(cell("data_type"), row);
    const enabled = parseBoolean(cell("enabled"), "enabled", row) ?? true;
    const isIndex =
      indexed && (parseBoolean(cell(INDEX_COLUMN), INDEX_COLUMN, row) ?? false);
    const node: ScannedNode = {
      key: nodeIdentity({ nodeId, field }),
      name: cell("node_name") || nodeName(nodeId, field),
      nodeId,
      field,
      dataType,
      isArray: false,
      nodeClass: "Variable",
    };
    if (identities.has(node.key)) return;
    const ch = convert(node);
    if (ch == null) return;
    identities.add(node.key);
    const imported = { ...ch, name: cell("name"), disabled: !enabled };
    channels.push(indexed ? { ...imported, isIndex } : imported);
  });
  return channels;
};
