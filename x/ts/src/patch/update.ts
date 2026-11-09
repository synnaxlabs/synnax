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
}

interface ListOps {
  set?: unknown[];
  insert?: unknown[];
  remove?: unknown[];
  update?: record.Unknown[];
  move?: { key: unknown; after?: unknown }[];
}

type StructKind = Extract<Kind, { kind: "struct" }>;

const asObject = (value: unknown): record.Unknown =>
  narrow.isObject(value) ? value : {};

const asList = (value: unknown): unknown[] => (Array.isArray(value) ? value : []);

const unionOf = (schema: $ZodType): Union | undefined => {
  const unwrapped = zod.unwrap(schema);
  const def = unwrapped._zod.def as {
    type: string;
    discriminator?: string;
    options?: $ZodType[];
  };
  if (def.type !== "union" || def.discriminator == null) return undefined;
  return {
    schema: unwrapped as $ZodDiscriminatedUnion,
    discriminator: def.discriminator,
    options: def.options ?? [],
  };
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

const recordUnionOf = (schema: $ZodType): Union => {
  const union = unionOf(schema);
  if (union == null) throw new Error("The record schema of a union is not a union");
  return union;
};

/** @returns the fields that every variant of the union holds, but its discriminator. */
const baseFields = (union: Union): string[] => {
  const [first, ...rest] = union.options.map((o) => Object.keys(shapeOf(o)));
  return first.filter(
    (k) => k !== union.discriminator && rest.every((keys) => keys.includes(k)),
  );
};

const kindAt = (shape: Shape, field: string): Kind => {
  const schema = shape[field];
  if (schema == null) throw new Error(`The update schema holds no field ${field}`);
  const kind = kindOf(schema);
  if (kind == null) throw new Error(`${field} is not an update field`);
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
  if (union == null)
    return mergeObject(shapeOf(u), shapeOf(t), asObject(current), partial);
  const records = recordUnionOf(t);
  const { discriminator } = union;
  const type = partial[discriminator];
  const fields = omit(partial, discriminator);
  const updateShape = shapeOf(optionOf(union, type));
  const recordShape = shapeOf(optionOf(records, type));
  const base = asObject(current);
  if (base[discriminator] === type)
    return mergeObject(updateShape, recordShape, base, fields);
  const start: record.Unknown = { [discriminator]: type };
  baseFields(records).forEach((k) => {
    if (base[k] !== undefined) start[k] = base[k];
  });
  return fillDefaults(
    recordShape,
    mergeObject(updateShape, recordShape, start, fields),
  );
};

const fillDefaults = (shape: Shape, value: record.Unknown): record.Unknown => {
  Object.entries(shape).forEach(([k, field]) => {
    if (value[k] !== undefined) return;
    if (field._zod.optin == null) throw new Error(`${k} is required`);
    const fallback = field.parse(undefined);
    if (fallback !== undefined) value[k] = fallback;
  });
  return value;
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
  switch (kind.kind) {
    case "change": {
      if (change === null) {
        if (field._zod.optout !== "optional")
          throw new Error("a required field cannot be cleared");
        return undefined;
      }
      const inner = kindOf(kind.value);
      return inner?.kind === "struct" ? applyStruct(inner, current, change) : change;
    }
    case "struct":
      return applyStruct(kind, current, change);
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

const applyStruct = (kind: StructKind, current: unknown, change: unknown): unknown => {
  if (replaces(change)) return change[SET];
  if (current == null) throw new StaleError("the struct to merge into is absent");
  return merge(kind.update, kind.full, current, change as record.Unknown);
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
    list[i] = mergeEntry(list[i], u);
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
 * Applies an update to a record without changing the record. A union that changes its
 * discriminator keeps the fields that every variant holds, and fills absent variant
 * fields with their defaults.
 *
 * @param schemas - The schema of the update value, and the record schema for defaults.
 * @throws {StaleError} if a guard does not hold, a list key is missing, or a merge
 * goes into an absent struct.
 * @throws {Error} if the update clears a required field, combines set with another
 * operator, inserts a key that exists, or leaves out a variant field with no default.
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
    case "change": {
      if (cur === undefined) return null;
      const inner = kindOf(kind.value);
      return inner?.kind === "struct"
        ? diffStruct(inner, prev, cur, path, guards)
        : cur;
    }
    case "struct":
      return diffStruct(kind, prev, cur, path, guards);
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

const diffStruct = (
  kind: StructKind,
  prev: unknown,
  cur: unknown,
  path: string[],
  guards: Guard[],
): unknown =>
  prev == null ? { [SET]: cur } : diffMerge(kind.update, prev, cur, path, guards);

const replaceWith = (kind: Kind, value: unknown): unknown => {
  if (kind.kind === "change" && kindOf(kind.value)?.kind !== "struct") return value;
  return { [SET]: value };
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
  const child = asObject(doc)[field];
  switch (kind.kind) {
    case "change": {
      const inner = kindOf(kind.value);
      if (inner?.kind !== "struct")
        throw new Error(`${formatPath(path.slice(0, i + 1))} holds no fields`);
      return { [field]: nestAt(inner.update, child, path, i + 1, leaf) };
    }
    case "struct":
      return { [field]: nestAt(kind.update, child, path, i + 1, leaf) };
    case "list": {
      const key = path[i + 1];
      const entry = asList(child).find((e) => String(asObject(e)[KEY]) === key);
      const inner = nestAt(kind.update, entry, path, i + 2, leaf);
      return {
        [field]: {
          update: [{ [KEY]: asObject(entry)[KEY] ?? key, ...asObject(inner) }],
        },
      };
    }
    case "valueList":
      throw new Error(`${formatPath(path.slice(0, i + 1))} holds no fields`);
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
