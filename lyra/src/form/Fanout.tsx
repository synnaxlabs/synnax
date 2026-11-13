// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement, useRef } from "react";

import {
  Context,
  type ContextValue,
  type GetFunc,
  type InsertFunc,
  type MoveFunc,
  type RemoveFunc,
  type SetFunc,
  useContext,
} from "@/form/Context";
import { type FieldState } from "@/form/state";

export interface FanoutProps extends PropsWithChildren {
  /** The subtrees to edit as one. The fields inside bind under the first. */
  paths: deep.Target<unknown>[];
}

interface Merged {
  states: (FieldState | null)[];
  state: FieldState | null;
}

const merge = (states: (FieldState | null)[]): FieldState | null => {
  const [first] = states;
  if (first == null) return null;
  const mixed = states.some((s) => s == null || !deep.equal(s.value, first.value));
  const status =
    states.find((s) => s != null && s.status.variant !== "success")?.status ??
    first.status;
  if (!mixed && status === first.status) return first;
  return { ...first, status, mixed };
};

/**
 * Edits several subtrees of the form as one. A field inside binds under the first path.
 * It reads as mixed when the subtrees hold different values there. Edits, statuses,
 * checks, and validations of a path in the first subtree go to every subtree, and `has`
 * asks every subtree. Focus stays with the fields under the first path. Paths outside
 * the first pass through unchanged.
 */
export const Fanout = ({ paths, children }: FanoutProps): ReactElement => {
  const parent = useContext(undefined, "Form.Fanout");
  const cache = useRef(new Map<string, Merged>());
  const [first, ...rest] = paths;
  const prefix = String(first);
  const targets = <V,>(target: deep.Target<V>): deep.Target<V>[] => {
    const path = String(target);
    const inside = path === prefix || path.startsWith(`${prefix}${deep.SEPARATOR}`);
    if (!inside) return [target];
    const keys =
      path === prefix ? [] : path.slice(prefix.length + 1).split(deep.SEPARATOR);
    // The subtrees share one shape, so the same keys reach a V under each.
    const under = (root: deep.Target<unknown>) =>
      keys.reduce<deep.Path<unknown>>(
        (p, k) => p[k],
        deep.widen(root),
      ) as deep.Target<V>;
    return [target, ...rest.map(under)];
  };
  const each = <V,>(target: deep.Target<V>, fn: (path: deep.Target<V>) => void): void =>
    parent.batch(() => targets(target).forEach(fn));
  const get = ((target, opts) => {
    const [path, ...others] = targets(target);
    if (others.length === 0) return parent.get(path, opts);
    const states = [
      parent.get(path, opts),
      ...others.map((p) => parent.get(p, { optional: true })),
    ];
    const key = String(path);
    const cached = cache.current.get(key);
    if (
      cached != null &&
      cached.states.length === states.length &&
      cached.states.every((s, i) => s === states[i])
    )
      return cached.state;
    const state = merge(states);
    cache.current.set(key, { states, state });
    return state;
  }) as GetFunc;
  const set: SetFunc = (target, value, opts) =>
    each(target, (p) => parent.set(p, value, opts));
  const remove: RemoveFunc = (target) => each(target, (p) => parent.remove(p));
  const insert: InsertFunc = (target, value, after) =>
    each(target, (p) => parent.insert(p, value, after));
  const move: MoveFunc = (target, key, after) =>
    each(target, (p) => parent.move(p, key, after));
  const validate: ContextValue["validate"] = (target) =>
    target == null
      ? parent.validate()
      : targets(target)
          .map((p) => parent.validate(p))
          .every(Boolean);
  const validateAsync: ContextValue["validateAsync"] = async (target) => {
    if (target == null) return await parent.validateAsync();
    const results = await Promise.all(
      targets(target).map(async (p) => await parent.validateAsync(p)),
    );
    return results.every(Boolean);
  };
  const addCheck: ContextValue["addCheck"] = (target, schema) => {
    const removes = targets(target).map((p) => parent.addCheck(p, schema));
    return () => removes.forEach((r) => r());
  };
  const ctx: ContextValue = {
    ...parent,
    get,
    set,
    remove,
    insert,
    move,
    validate,
    validateAsync,
    addCheck,
    has: (target) => targets(target).every((p) => parent.has(p)),
    setStatus: (target, status) => each(target, (p) => parent.setStatus(p, status)),
  };
  return <Context value={ctx}>{children}</Context>;
};
