// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * NODE_TAG is the brand symbol carried on every {@link Node} value. Exported
 * so per-entity generated code can construct Nodes directly without a shared
 * runtime constructor, which keeps each entity's combinators fully resolved
 * at codegen time.
 */
export const NODE_TAG: unique symbol = Symbol("filter:node");

/**
 * OP_TAG is the brand symbol carried on every {@link OpNode} value. Exported
 * for the same reason as {@link NODE_TAG}: per-entity generated code inlines
 * the op constructors.
 */
export const OP_TAG: unique symbol = Symbol("filter:op");

export type OpKind = "eq" | "gt" | "lt" | "gte" | "lte" | "between";

type OpValue<T, K extends OpKind> = K extends "between" ? readonly [T, T] : T;

/**
 * OpNode is a branded wrapper around a comparison value that records its
 * operator. Constructed inline by per-entity generated code; the brand is
 * invisible to autocomplete but discriminable by {@link isOpNode} at runtime.
 */
export interface OpNode<T, K extends OpKind = OpKind> {
  readonly value: OpValue<T, K>;
  readonly [OP_TAG]: K;
}

export type Combinator = "and" | "or" | "not";

/**
 * Node is a branded combinator output that tags its entity at the type level,
 * so {@code channel.or(rack.where(...))} fails type-checking. Children may be
 * filter objects or other Nodes; the per-entity wire walker interprets them
 * via the entity descriptor.
 */
export interface Node<E extends string> {
  readonly kind: Combinator;
  readonly children: readonly unknown[];
  readonly [NODE_TAG]: E;
}

export const isOpNode = <K extends OpKind = OpKind>(
  v: unknown,
  kind?: K,
): v is OpNode<unknown, K> => {
  if (typeof v !== "object" || v === null || !(OP_TAG in v)) return false;
  return kind === undefined || (v as OpNode<unknown>)[OP_TAG] === kind;
};

export const isNode = <E extends string = string>(
  v: unknown,
  entity?: E,
): v is Node<E> => {
  if (typeof v !== "object" || v === null || !(NODE_TAG in v)) return false;
  return entity === undefined || (v as Node<string>)[NODE_TAG] === entity;
};

export type FieldKind = "string" | "bool" | "numeric";

/**
 * Descriptor names the filterable and sortable fields on an entity. It is
 * emitted per entity by oracle and consumed by {@link toWire} to drive
 * conversion from the user-facing API surface to the wire format.
 */
export interface Descriptor {
  readonly entity: string;
  readonly fields: Readonly<Record<string, FieldKind>>;
  readonly orderFields: Readonly<Record<string, FieldKind>>;
}

/**
 * BASE_OPTION_KEYS are the option fields shared by every entity's retrieve
 * request. Per-entity option sets extend this with entity-specific keys.
 */
export const BASE_OPTION_KEYS: ReadonlySet<string> = new Set([
  "limit",
  "offset",
  "orderBy",
  "searchTerm",
  "includeStatus",
]);

export type WireNode = {
  [field: string]: unknown;
  and?: WireNode[];
  or?: WireNode[];
  not?: WireNode;
};

export type FilterArg = Node<string> | object;

const convertOpNode = (op: OpNode<unknown>): Record<string, unknown> => {
  const kind = (op as { readonly [OP_TAG]: OpKind })[OP_TAG];
  if (kind === "between") {
    const [lo, hi] = op.value as readonly [unknown, unknown];
    return { gte: lo, lte: hi };
  }
  return { [kind]: op.value };
};

const convertFieldValue = (value: unknown): unknown => {
  if (value === undefined || value === null) return undefined;
  if (isOpNode(value)) return convertOpNode(value);
  if (value instanceof RegExp) return { regex: value.source };
  if (Array.isArray(value)) return { in: value };
  return { eq: value };
};

const convertFilter = (filter: object, descriptor: Descriptor): WireNode => {
  const out: WireNode = {};
  for (const [key, value] of Object.entries(filter)) {
    if (!(key in descriptor.fields)) continue;
    const wire = convertFieldValue(value);
    if (wire !== undefined) out[key] = wire;
  }
  return out;
};

const convertNode = (n: Node<string>, descriptor: Descriptor): WireNode => {
  const children = n.children.map((c) => convertArg(c as FilterArg, descriptor));
  if (n.kind === "not") return { not: children[0] };
  return { [n.kind]: children };
};

const convertArg = (arg: FilterArg, descriptor: Descriptor): WireNode => {
  if (isNode(arg)) return convertNode(arg, descriptor);
  return convertFilter(arg, descriptor);
};

/**
 * toWire converts a list of user-facing filter args into the JSON wire shape
 * accepted by the server. Multiple top-level args are implicitly ANDed.
 * Returns undefined when no constraints are present.
 */
export const toWire = (
  args: readonly FilterArg[],
  descriptor: Descriptor,
): WireNode | undefined => {
  if (args.length === 0) return undefined;
  if (args.length === 1) return convertArg(args[0], descriptor);
  return { and: args.map((arg) => convertArg(arg, descriptor)) };
};
