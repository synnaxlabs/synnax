// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** Options of a condition on a text line. */
export interface StringOptions {
  /** caseInsensitive makes the condition ignore case. */
  caseInsensitive?: boolean;
}

declare const entity: unique symbol;

/**
 * A condition on the records of the entity E, which holds the filter tree W. Build one
 * from the lines of an entity, such as `ranger.name.eq("Hotfire")`.
 */
export class Condition<E extends string, W = unknown> {
  declare readonly [entity]: E;

  constructor(readonly tree: W) {}
}

/** An option of a retrieve, which sets the request field F. */
export class Option<F extends string = string> {
  constructor(
    readonly field: F,
    readonly value: unknown,
  ) {}
}

/** Makes a retrieve return one record. It goes after every other part. */
export class One {
  declare private readonly one: true;
}

/** A condition of the type C, or an option that sets a request field in F. */
export type Part<C extends Condition<string>, F extends string> = C | Option<F>;

/** @returns whether value is a condition, an option, or one. */
export const isPart = (value: unknown): value is Condition<string> | Option | One =>
  value instanceof Condition || value instanceof Option || value instanceof One;

interface Tree<W> {
  and?: W[];
  or?: W[];
  not?: W;
}

type Line<W> = Exclude<keyof W, keyof Tree<W>> & string;

type Member<W, L extends keyof W, M extends string> =
  NonNullable<W[L]> extends { [K in M]?: infer V } ? NonNullable<V> : never;

type Element<A> = A extends ReadonlyArray<infer T> ? T : never;

/**
 * Returns the functions that build the conditions of the entity E, whose filter tree is
 * W. Oracle generates each entity's lines from them.
 */
export const builder = <E extends string, W extends Tree<W>>() => {
  const condition = (tree: object): Condition<E, W> => new Condition(tree as W);
  const op = (line: string, member: string, value: unknown, options?: StringOptions) =>
    condition({ [line]: { [member]: value, ...options } });
  return {
    and: (...conditions: Array<Condition<E, W>>): Condition<E, W> =>
      condition({ and: conditions.map((c) => c.tree) }),
    or: (...conditions: Array<Condition<E, W>>): Condition<E, W> =>
      condition({ or: conditions.map((c) => c.tree) }),
    not: (c: Condition<E, W>): Condition<E, W> => condition({ not: c.tree }),
    where: (tree: W): Condition<E, W> => new Condition(tree),
    set: <L extends Line<W>>(line: L) => ({
      eq: (value: Member<W, L, "eq">) => op(line, "eq", value),
      in: (values: Member<W, L, "in">) => op(line, "in", values),
    }),
    bool: (line: Line<W>) => ({ eq: (value: boolean) => op(line, "eq", value) }),
    ordered: <L extends Line<W>>(line: L) => ({
      eq: (value: Member<W, L, "eq">) => op(line, "eq", value),
      in: (values: Member<W, L, "in">) => op(line, "in", values),
      gt: (value: Member<W, L, "gt">) => op(line, "gt", value),
      gte: (value: Member<W, L, "gte">) => op(line, "gte", value),
      lt: (value: Member<W, L, "lt">) => op(line, "lt", value),
      lte: (value: Member<W, L, "lte">) => op(line, "lte", value),
    }),
    string: (line: Line<W>) => ({
      eq: (value: string, options?: StringOptions) => op(line, "eq", value, options),
      in: (values: string[], options?: StringOptions) =>
        op(line, "in", values, options),
      contains: (value: string, options?: StringOptions) =>
        op(line, "contains", value, options),
      startsWith: (value: string, options?: StringOptions) =>
        op(line, "startsWith", value, options),
      regex: (pattern: string, options?: StringOptions) =>
        op(line, "regex", pattern, options),
    }),
    contains: <L extends Line<W>>(line: L) => ({
      has: (value: Element<Member<W, L, "any">>) => op(line, "any", [value]),
      hasAny: (values: Member<W, L, "any">) => op(line, "any", values),
    }),
    flag: (line: Line<W>) => ({
      eq: (value: boolean) => condition({ [line]: value }),
    }),
    list: <L extends Line<W>>(line: L) => ({
      has: (value: Element<NonNullable<W[L]>>) => condition({ [line]: [value] }),
      hasAny: (values: NonNullable<W[L]>) => condition({ [line]: values }),
    }),
    value:
      <L extends Line<W>>(line: L) =>
      (value: NonNullable<W[L]>) =>
        condition({ [line]: value }),
  };
};

/** @returns an option that returns at most n records. */
export const limit = (n: number): Option<"limit"> => new Option("limit", n);

/** @returns an option that skips the first n matching records. */
export const offset = (n: number): Option<"offset"> => new Option("offset", n);

/** @returns an option that keeps the records that match term, closest match first. */
export const search = (term: string): Option<"searchTerm"> =>
  new Option("searchTerm", term);

/** @returns a function that returns an option setting the request flag field. */
export const include =
  <F extends string>(field: F) =>
  (): Option<F> =>
    new Option(field, true);

/** @returns a part that makes a retrieve return one record. */
export const one = (): One => new One();

/**
 * @returns the retrieve request that parts make. Every condition must match, and each
 * option sets its field.
 * @throws {Error} if two options set the same field.
 */
export const toRequest = (
  parts: ReadonlyArray<Part<Condition<string>, string>>,
): Record<string, unknown> => {
  const request: Record<string, unknown> = {};
  const where: unknown[] = [];
  for (const part of parts)
    if (part instanceof Condition) where.push(part.tree);
    else if (part.field in request) throw new Error(`${part.field} is set twice`);
    else request[part.field] = part.value;

  if (where.length > 0) request.where = where.length === 1 ? where[0] : { and: where };
  return request;
};
