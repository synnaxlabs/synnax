// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type z } from "zod";
import {
  type $ZodDiscriminatedUnion,
  type $ZodType,
  getDiscriminatedOption,
} from "zod/v4/core";

import { deep } from "@/deep";
import { errors } from "@/errors";
import { narrow } from "@/narrow";
import { type StructSchemas } from "@/patch/field";
import { type Kind, kindOf } from "@/patch/kind";
import { StaleError } from "@/patch/stale";
import { type Guard, type Update } from "@/patch/types.gen";
import { type record } from "@/record";
import { zod } from "@/zod";

const KEY = "key";
const SET = "set";

type Shape = Record<string, z.ZodType>;

interface Union {
  schema: $ZodDiscriminatedUnion;
  discriminator: string;
  options: $ZodType[];
  /** The strict schema of an update that holds only base fields. */
  base?: $ZodType;
}

interface UnionDef {
  type: string;
  discriminator?: string;
  options?: $ZodType[];
}

interface ListOps {
  set?: unknown[];
  insert?: unknown[];
  remove?: unknown[];
  update?: record.Unknown[];
  move?: { key: unknown; after?: unknown }[];
}

const VALUE: Kind = { kind: "value" };

// Types of an unmarked field schema that must come from a field factory.
const FACTORY_TYPES = new Set(["object", "array"]);

const asObject = (value: unknown): record.Unknown =>
  narrow.isObject(value) ? value : {};

const asList = (value: unknown): unknown[] => (Array.isArray(value) ? value : []);

const taggedOf = (schema: $ZodType): Union | undefined => {
  const unwrapped = zod.unwrap(schema);
  const def = unwrapped._zod.def as UnionDef;
  if (def.type !== "union" || def.discriminator == null) return undefined;
  return {
    schema: unwrapped as $ZodDiscriminatedUnion,
    discriminator: def.discriminator,
    options: def.options ?? [],
  };
};

/**
 * @returns the discriminated union that the schema holds, or the one that it joins to
 * the strict object of an update with only base fields.
 */
const unionOf = (schema: $ZodType): Union | undefined => {
  const tagged = taggedOf(schema);
  if (tagged != null) return tagged;
  const def = zod.unwrap(schema)._zod.def as UnionDef;
  if (def.type !== "union" || def.options?.length !== 2) return undefined;
  const [base, variants] = def.options;
  const union = taggedOf(variants);
  if (union == null || zod.unwrap(base)._zod.def.type !== "object") return undefined;
  return { ...union, base };
};

const shapeOf = (schema: $ZodType): Shape => {
  const def = zod.unwrap(schema)._zod.def as { type: string; shape?: Shape };
  if (def.shape == null) throw new Error(`Expected an object schema, got ${def.type}`);
  return def.shape;
};

const optionOf = (union: Union, value: unknown): $ZodType => {
  const option = getDiscriminatedOption(union.schema, value as never) as
    $ZodType | undefined;
  if (option == null)
    throw new Error(`${union.discriminator} ${String(value)} names no variant`);
  return option;
};

const requireUnion = (schema: $ZodType): Union => {
  const union = unionOf(schema);
  if (union == null)
    throw new Error(`Expected a union schema, got ${zod.unwrap(schema)._zod.def.type}`);
  return union;
};

/** @returns the fields that every variant of the union holds, but its discriminator. */
const baseFields = (union: Union): string[] => {
  const [first, ...rest] = union.options.map((o) => Object.keys(shapeOf(o)));
  return first.filter(
    (k) => k !== union.discriminator && rest.every((keys) => keys.includes(k)),
  );
};

// An unmarked schema merges as a union, or sets a value.
const kindIn = (schema: z.ZodType): Kind => {
  const kind = kindOf(schema);
  if (kind != null) return kind;
  return unionOf(schema) == null ? VALUE : { kind: "union", update: schema };
};

const kindAt = (shape: Shape, field: string): Kind => {
  const schema = shape[field];
  if (schema == null) throw new Error(`The update schema holds no field ${field}`);
  const kind = kindIn(schema);
  if (kind.kind === "value" && FACTORY_TYPES.has(zod.unwrap(schema)._zod.def.type))
    throw new Error(`${field} is not an update field`);
  return kind;
};

const replaces = (change: unknown): change is { [SET]: unknown } =>
  narrow.isObject(change) && Object.hasOwn(change, SET);

const same = (a: unknown, b: unknown): boolean => a === b || deep.equal(a, b);

const omit = (value: record.Unknown, field: string): record.Unknown => {
  const { [field]: _, ...rest } = value;
  return rest;
};

const formatPath = (path: string[]): string => `[${path.join(", ")}]`;

const readAt = (doc: unknown, path: string[]): unknown =>
  path.reduce<unknown>((current, seg) => {
    if (Array.isArray(current))
      return current.find((e) => narrow.isObject(e) && e[KEY] === seg);
    return asObject(current)[seg];
  }, doc);

const checkGuards = (doc: unknown, guards: Guard[]): void =>
  guards.forEach(({ path, value }) => {
    if (path.length === 0) throw new Error("a guard needs a path");
    if (readAt(doc, path) !== value)
      throw new StaleError(`${formatPath(path)} is not ${value}`);
  });

const merge = (
  u: $ZodType,
  t: $ZodType,
  current: unknown,
  partial: record.Unknown,
): record.Unknown => {
  const union = unionOf(u);
  if (union == null) {
    if (current == null) return build(shapeOf(u), shapeOf(t), {}, partial);
    return mergeObject(shapeOf(u), shapeOf(t), asObject(current), partial);
  }
  const records = requireUnion(t);
  const { discriminator } = union;
  const type = partial[discriminator];
  const fields = omit(partial, discriminator);
  const base = asObject(current);
  if (type === undefined) {
    if (current == null) throw new Error(`${discriminator} is required`);
    const baseShape = union.base == null ? {} : shapeOf(union.base);
    const recordShape = shapeOf(optionOf(records, base[discriminator]));
    return mergeObject(baseShape, recordShape, base, fields);
  }
  const updateShape = shapeOf(optionOf(union, type));
  const recordShape = shapeOf(optionOf(records, type));
  if (current != null && base[discriminator] === type)
    return mergeObject(updateShape, recordShape, base, fields);
  const seed: record.Unknown = { [discriminator]: type };
  baseFields(records).forEach((k) => {
    if (base[k] !== undefined) seed[k] = base[k];
  });
  return build(updateShape, recordShape, seed, fields);
};

// Builds a value from the defaults of recordShape and seed, merged with partial.
const build = (
  updateShape: Shape,
  recordShape: Shape,
  seed: record.Unknown,
  partial: record.Unknown,
): record.Unknown => {
  const start: record.Unknown = {};
  Object.entries(recordShape).forEach(([k, field]) => {
    if (field._zod.optin == null) return;
    const fallback = field.parse(undefined);
    if (fallback !== undefined) start[k] = fallback;
  });
  const next = mergeObject(updateShape, recordShape, { ...start, ...seed }, partial);
  Object.entries(recordShape).forEach(([k, field]) => {
    if (next[k] === undefined && field._zod.optin == null)
      throw new Error(`${k} is required`);
  });
  return next;
};

const mergeObject = (
  updateShape: Shape,
  recordShape: Shape,
  current: record.Unknown,
  partial: record.Unknown,
): record.Unknown => {
  const next = { ...current };
  Object.entries(partial).forEach(([k, change]) => {
    if (change === undefined) return;
    const field = recordShape[k];
    if (field == null) throw new Error(`The record schema holds no field ${k}`);
    const value = applyField(kindAt(updateShape, k), field, current[k], change);
    if (value === undefined) delete next[k];
    else next[k] = value;
  });
  return next;
};

const applyField = (
  kind: Kind,
  field: z.ZodType,
  current: unknown,
  change: unknown,
): unknown => {
  if (change === null) {
    if (kind.kind !== "change" || field._zod.optout !== "optional")
      throw new Error("a required field cannot be cleared");
    return undefined;
  }
  switch (kind.kind) {
    case "value":
      return change;
    case "change":
      return applyField(kindIn(kind.value), field, current, change);
    case "struct":
      return replaces(change)
        ? change[SET]
        : merge(kind.update, kind.full, current, change as record.Unknown);
    case "union":
      return merge(kind.update, field, current, change as record.Unknown);
    case "list":
      return applyList(
        current,
        change as ListOps,
        (e) => asObject(e)[KEY],
        (e, u) => merge(kind.update, kind.entry, e, omit(u, KEY)),
      );
    case "valueList":
      return applyList(current, change as ListOps, (e) => e);
  }
};

const applyList = (
  current: unknown,
  ops: ListOps,
  keyOf: (entry: unknown) => unknown,
  mergeEntry?: (entry: unknown, update: record.Unknown) => unknown,
): unknown => {
  const { set, insert, remove, update, move } = ops;
  if (set != null) {
    if (insert != null || remove != null || update != null || move != null)
      throw new Error("set cannot be combined with other operators");
    return appendUnique([], set, keyOf);
  }
  if (insert == null && remove == null && update == null && move == null)
    return current;
  let list = [...asList(current)];
  const indexOf = (key: unknown): number => {
    const i = list.findIndex((e) => same(keyOf(e), key));
    if (i < 0) throw new StaleError(`${String(key)} not found`);
    return i;
  };
  remove?.forEach((key) => list.splice(indexOf(key), 1));
  update?.forEach((u) => {
    const i = indexOf(u[KEY]);
    if (mergeEntry == null) throw new Error("A value list has no update operator");
    list[i] = withKey(u[KEY], () => mergeEntry(list[i], u));
  });
  list = appendUnique(list, insert ?? [], keyOf);
  move?.forEach(({ key, after }) => {
    if (after != null && same(after, key))
      throw new Error(`${String(key)} cannot move after itself`);
    const [entry] = list.splice(indexOf(key), 1);
    list.splice(after == null ? 0 : indexOf(after) + 1, 0, entry);
  });
  return list;
};

// Runs fn, and starts the message of an error that it throws with the key of an entry.
const withKey = <R>(key: unknown, fn: () => R): R => {
  try {
    return fn();
  } catch (e) {
    const message = `${String(key)}: ${errors.fromUnknown(e).message}`;
    if (StaleError.matches(e)) throw new StaleError(message, { cause: e });
    throw new Error(message, { cause: e });
  }
};

const appendUnique = (
  list: unknown[],
  entries: unknown[],
  keyOf: (entry: unknown) => unknown,
): unknown[] =>
  entries.reduce<unknown[]>((acc, entry) => {
    const key = keyOf(entry);
    if (acc.some((e) => same(keyOf(e), key)))
      throw new Error(`${String(key)} already exists`);
    acc.push(entry);
    return acc;
  }, list);

/** The result of {@link applyUpdate}. */
export interface AppliedUpdate<U extends z.ZodType, T extends z.ZodType> {
  /** The record after the update. */
  next: z.output<T>;
  /** The update that turns next back into the record before the update. */
  inverse: Update<U>;
}

/**
 * Applies an update to a record without changing the record. A merge into an absent
 * struct starts from its defaults. A union partial with no discriminator merges base
 * fields into the current variant. A union that changes its discriminator keeps the
 * fields that every variant holds, and fills absent variant fields with their
 * defaults.
 *
 * @param schemas - The schema of the update value, and the record schema for defaults.
 * @throws {StaleError} if a guard does not hold or a list key is missing.
 * @throws {Error} if the update clears a required field, combines set with another
 * operator, inserts a key that exists, gives no discriminator for an absent union, or
 * leaves out a field with no default of a new struct or variant.
 */
export const applyUpdate = <U extends z.ZodType, T extends z.ZodType>(
  { u, t }: StructSchemas<U, T>,
  state: z.output<T>,
  update: Update<U>,
): AppliedUpdate<U, T> => {
  const { value, guards } = update as Update<z.ZodType>;
  checkGuards(state, guards);
  const next = merge(u, t, state, asObject(value)) as z.output<T>;
  return { next, inverse: diffUpdate(u, next, state) };
};

const diffMerge = (
  u: $ZodType,
  base: unknown,
  next: unknown,
  path: string[],
  guards: Guard[],
): record.Unknown | undefined => {
  const prev = asObject(base);
  const cur = asObject(next);
  const union = unionOf(u);
  if (union == null) return diffObject(shapeOf(u), prev, cur, path, guards);
  const { discriminator } = union;
  const type = cur[discriminator];
  const shape = shapeOf(optionOf(union, type));
  if (prev[discriminator] === type) {
    const partial = diffObject(
      shape,
      omit(prev, discriminator),
      omit(cur, discriminator),
      path,
      guards,
    );
    if (partial == null) return undefined;
    if (typeof type !== "string")
      throw new Error(`The discriminator at ${path.join(".")} is not a string`);
    guards.push({ path: [...path, discriminator], value: type });
    return { [discriminator]: type, ...partial };
  }
  const shared = new Set(baseFields(union));
  const partial: record.Unknown = { [discriminator]: type };
  Object.keys(shape).forEach((k) => {
    if (k === discriminator) return;
    if (!shared.has(k)) {
      if (cur[k] !== undefined) partial[k] = replaceWith(kindAt(shape, k), cur[k]);
      return;
    }
    if (same(prev[k], cur[k])) return;
    const change = diffField(kindAt(shape, k), prev[k], cur[k], [...path, k], guards);
    if (change !== undefined) partial[k] = change;
  });
  return partial;
};

const diffObject = (
  shape: Shape,
  prev: record.Unknown,
  cur: record.Unknown,
  path: string[],
  guards: Guard[],
): record.Unknown | undefined => {
  let partial: record.Unknown | undefined;
  new Set([...Object.keys(prev), ...Object.keys(cur)]).forEach((k) => {
    if (same(prev[k], cur[k])) return;
    const change = diffField(kindAt(shape, k), prev[k], cur[k], [...path, k], guards);
    if (change !== undefined) (partial ??= {})[k] = change;
  });
  return partial;
};

const diffField = (
  kind: Kind,
  prev: unknown,
  cur: unknown,
  path: string[],
  guards: Guard[],
): unknown => {
  switch (kind.kind) {
    case "value":
      return cur;
    case "change":
      if (cur === undefined) return null;
      return diffField(kindIn(kind.value), prev, cur, path, guards);
    case "struct":
    case "union":
      if (prev == null) return replaceWith(kind, cur);
      return diffMerge(kind.update, prev, cur, path, guards);
    case "list":
      return diffList(
        asList(prev),
        asList(cur),
        (e) => asObject(e)[KEY],
        (p, c) => {
          const key = asObject(c)[KEY];
          const partial = diffMerge(kind.update, p, c, [...path, String(key)], guards);
          return partial == null ? undefined : { [KEY]: key, ...partial };
        },
      );
    case "valueList":
      return diffList(asList(prev), asList(cur), (e) => e);
  }
};

// Builds the change that turns any value of the field into value.
const replaceWith = (kind: Kind, value: unknown): unknown => {
  switch (kind.kind) {
    case "value":
      return value;
    case "change":
      return replaceWith(kindIn(kind.value), value);
    case "union":
      return replaceUnion(kind.update, asObject(value));
    case "struct":
    case "list":
    case "valueList":
      return { [SET]: value };
  }
};

// A union has no set, so the change names the variant and replaces each field.
const replaceUnion = (u: $ZodType, value: record.Unknown): record.Unknown => {
  const union = requireUnion(u);
  const { discriminator } = union;
  const type = value[discriminator];
  const shape = shapeOf(optionOf(union, type));
  const partial: record.Unknown = { [discriminator]: type };
  Object.entries(omit(value, discriminator)).forEach(([k, v]) => {
    if (v !== undefined) partial[k] = replaceWith(kindAt(shape, k), v);
  });
  return partial;
};

const diffList = (
  prev: unknown[],
  cur: unknown[],
  keyOf: (entry: unknown) => unknown,
  diffEntry?: (prev: unknown, cur: unknown) => record.Unknown | undefined,
): ListOps | undefined => {
  const indexIn = (list: unknown[], key: unknown): number =>
    list.findIndex((e) => same(keyOf(e), key));
  const ops: ListOps = {};
  const remove = prev.filter((e) => indexIn(cur, keyOf(e)) < 0).map(keyOf);
  const insert = cur.filter((e) => indexIn(prev, keyOf(e)) < 0);
  const update: record.Unknown[] = [];
  if (diffEntry != null)
    cur.forEach((c) => {
      const i = indexIn(prev, keyOf(c));
      if (i < 0 || same(prev[i], c)) return;
      const partial = diffEntry(prev[i], c);
      if (partial != null) update.push(partial);
    });
  const kept = prev.filter((e) => indexIn(cur, keyOf(e)) >= 0);
  const positions = [...kept, ...insert].map((e) => indexIn(cur, keyOf(e)));
  const ordered = new Set(longestIncreasing(positions));
  const move: ListOps["move"] = [];
  cur.forEach((c, i) => {
    if (ordered.has(i)) return;
    move.push(
      i === 0 ? { key: keyOf(c) } : { key: keyOf(c), after: keyOf(cur[i - 1]) },
    );
  });
  if (remove.length > 0) ops.remove = remove;
  if (update.length > 0) ops.update = update;
  if (insert.length > 0) ops.insert = insert;
  if (move.length > 0) ops.move = move;
  return Object.keys(ops).length > 0 ? ops : undefined;
};

/** @returns the values of the longest strictly increasing subsequence of values. */
const longestIncreasing = (values: number[]): number[] => {
  const tails: number[] = [];
  const previous: number[] = [];
  values.forEach((v, i) => {
    let lo = 0;
    let hi = tails.length;
    while (lo < hi) {
      const mid = (lo + hi) >> 1;
      if (values[tails[mid]] < v) lo = mid + 1;
      else hi = mid;
    }
    previous[i] = lo > 0 ? tails[lo - 1] : -1;
    tails[lo] = i;
  });
  const out: number[] = [];
  for (let i = tails.length > 0 ? tails[tails.length - 1] : -1; i >= 0; i = previous[i])
    out.push(values[i]);
  return out.reverse();
};

/**
 * Builds the update that turns base into next and touches only what differs, so a
 * concurrent edit to another field survives. A union that keeps its discriminator
 * merges under a guard on it. A union that changes it takes every variant field of
 * next.
 *
 * @param u - The schema of the update value.
 * @throws {Error} if the schema does not hold a field that differs.
 */
export const diffUpdate = <U extends z.ZodType, T>(
  u: U,
  base: T,
  next: T,
): Update<U> => {
  const guards: Guard[] = [];
  const value = diffMerge(u, base, next, [], guards) ?? {};
  const update: Update<z.ZodType> = { value, guards };
  return update;
};

const nestAt = (
  schema: $ZodType,
  doc: unknown,
  path: string[],
  i: number,
  leaf: unknown,
): unknown => {
  const union = unionOf(schema);
  if (union != null) {
    const type = asObject(doc)[union.discriminator];
    const inner = nestAt(optionOf(union, type), doc, path, i, leaf);
    return { [union.discriminator]: type, ...asObject(inner) };
  }
  if (i === path.length) return leaf;
  const field = path[i];
  const kind = kindAt(shapeOf(schema), field);
  if (i + 1 === path.length) return { [field]: leaf };
  return { [field]: nestInto(kind, asObject(doc)[field], path, i + 1, leaf) };
};

const nestInto = (
  kind: Kind,
  doc: unknown,
  path: string[],
  i: number,
  leaf: unknown,
): unknown => {
  switch (kind.kind) {
    case "change":
      return nestInto(kindIn(kind.value), doc, path, i, leaf);
    case "struct":
    case "union":
      return nestAt(kind.update, doc, path, i, leaf);
    case "list": {
      const key = path[i];
      const entry = asList(doc).find((e) => String(asObject(e)[KEY]) === key);
      const inner = nestAt(kind.update, entry, path, i + 1, leaf);
      return { update: [{ [KEY]: asObject(entry)[KEY] ?? key, ...asObject(inner) }] };
    }
    case "value":
    case "valueList":
      throw new Error(`${formatPath(path.slice(0, i))} holds no fields`);
  }
};

/**
 * Wraps the update of the value at a path into the update value of the whole record. A
 * segment into a struct nests as a merge, and a segment into a keyed list names the key
 * of an entry and nests as an update of that entry. A union that the path goes into
 * takes the discriminator that doc holds there.
 *
 * @param u - The schema of the update value.
 * @param doc - The record, which gives the discriminator of each union on the path.
 * @param leaf - The update of the value at the path.
 * @throws {Error} if the schema does not hold the path, or doc holds no variant of a
 * union on the path.
 */
export const nest = <U extends z.ZodType>(
  u: U,
  doc: unknown,
  path: string[],
  leaf: unknown,
): z.output<U> => nestAt(u, doc, path, 0, leaf) as z.output<U>;
