// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

declare const VALUE: unique symbol;

/** What a path points at. An API that reads or writes a V takes a Target<V>. */
export interface Target<V> {
  readonly [VALUE]: V;
  [Symbol.toPrimitive](this: void): string;
}

type Primitive = string | number | boolean | bigint | symbol | null | undefined;

interface Open {
  readonly [key: string]: Path<unknown>;
}

type Children<V> = unknown extends V
  ? Open
  : [V] extends [Primitive]
    ? unknown
    : [V] extends [NonNullable<V>]
      ? [V] extends [readonly (infer E)[]]
        ? { readonly [key: string | number]: Path<E> }
        : { readonly [K in keyof V & string]-?: Path<V[K]> }
      : unknown;

/**
 * A path to a value of type V. It has exactly the fields that V has, and each one is a
 * path. `String(path)` gives the dot-separated path.
 *
 * @example
 * const path = deep.path<Values>();
 * const port = path.config.channels[key].port;
 */
export type Path<V> = Target<V> & Children<V>;

interface Node {
  path: string;
  children: Map<string, unknown> | null;
}

const HANDLER: ProxyHandler<Node> = {
  // A path shows no fields, so a walker such as `JSON.stringify` stops at it.
  ownKeys: () => [],
  get: (node, key) => {
    if (key === Symbol.toPrimitive) return () => node.path;
    if (typeof key === "symbol") return undefined;
    node.children ??= new Map();
    let child = node.children.get(key);
    if (child == null) {
      child = create(node.path === "" ? key : `${node.path}.${key}`);
      node.children.set(key, child);
    }
    return child;
  },
};

const create = (path: string): unknown =>
  new Proxy<Node>({ path, children: null }, HANDLER);

/**
 * Creates a root path for values of type V. A root returns the same object for the same
 * field, so a path is safe to compare and to memoize on.
 */
export const path = <V>(): Path<V> => create("") as Path<V>;

/**
 * Narrows a path to a union to the members with the discriminator value. Nothing checks
 * the value at run time.
 */
export const variant = <
  V,
  D extends keyof NonNullable<V>,
  const T extends NonNullable<V>[D],
>(
  target: Target<V>,
  _discriminator: D,
  _value: T,
): Path<Extract<NonNullable<V>, Record<D, T>>> =>
  target as unknown as Path<Extract<NonNullable<V>, Record<D, T>>>;

/** Narrows a path to an optional value. Nothing checks the value at run time. */
export const present = <V>(target: Target<V>): Path<NonNullable<V>> =>
  target as unknown as Path<NonNullable<V>>;

/** Reads a path to a generic value through a wider, known shape. */
export const widen = <S>(target: Target<S>): Path<S> => target as unknown as Path<S>;
