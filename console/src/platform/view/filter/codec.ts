// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Properties } from "@/platform/view/filter/types";

/**
 * One condition of a filter, one for each chip. A condition with no property is a node
 * of the filter that no property reads.
 */
export type Condition<W> =
  { property: string; value: unknown } | { property: null; where: W };

type Node = Record<string, unknown>;

const definedFields = (node: object): string[] =>
  Object.entries(node)
    .filter(([, value]) => value !== undefined)
    .map(([field]) => field);

/** @returns the field the node sets when it sets exactly one, else null. */
export const soleField = (node: object): string | null => {
  const fields = definedFields(node);
  return fields.length === 1 ? fields[0] : null;
};

const decodeNode = <W extends object>(
  node: W,
  properties: Properties<W>,
): Condition<W>[] => {
  const fields = definedFields(node);
  // The fields of a node combine with and, so each one reads as its own chip.
  if (fields.length > 1)
    return fields.flatMap((field) =>
      decodeNode({ [field]: (node as Node)[field] } as W, properties),
    );
  for (const [property, { decode }] of Object.entries(properties)) {
    const value = decode(node);
    if (value != null) return [{ property, value }];
  }
  return [{ property: null, where: node }];
};

/** @returns the conditions of a filter, one for each chip. */
export const decode = <W extends { and?: W[] }>(
  where: W | undefined,
  properties: Properties<W>,
): Condition<W>[] => {
  if (where == null || definedFields(where).length === 0) return [];
  if (soleField(where) === "and" && where.and != null)
    return where.and.flatMap((node) => decodeNode(node, properties));
  return decodeNode(where, properties);
};

/** @returns the filter that holds the conditions, or undefined for none. */
export const encode = <W extends { and?: W[] }>(
  conditions: Condition<W>[],
  properties: Properties<W>,
): W | undefined => {
  if (conditions.length === 0) return undefined;
  const and = conditions.map((c) =>
    c.property == null ? c.where : properties[c.property].encode(c.value),
  );
  return { and } as W;
};

/**
 * @returns the conditions with the condition at the index set to next, or without it
 * when next is null.
 */
export const replace = <W>(
  conditions: Condition<W>[],
  index: number,
  next: Condition<W> | null,
): Condition<W>[] =>
  next == null
    ? conditions.filter((_, i) => i !== index)
    : conditions.map((c, i) => (i === index ? next : c));
