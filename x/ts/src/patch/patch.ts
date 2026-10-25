// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { narrow } from "@/narrow";
import {
  type Guard,
  type Insert,
  type Move,
  type Remove,
  type Set,
} from "@/patch/types.gen";
import { type record } from "@/record";

/** A path action, tagged the way a document's action union tags its variants. */
export type Action =
  | { type: "set"; set: Set }
  | { type: "remove"; remove: Remove }
  | { type: "insert"; insert: Insert }
  | { type: "move"; move: Move };

/** Options for {@link apply}. */
export interface ApplyOptions {
  /**
   * Copies a value read from the document before an action replaces or removes it. Pass
   * Immer's `current` when the document is a draft.
   */
  snapshot?: <T>(value: T) => T;
}

const identity = <T>(value: T): T => value;

/**
 * Applies a path action to the document in place. A segment of a path into an object
 * names a field, and a segment into a list names the key of an entry. An action whose
 * guards fail, or whose path goes through a missing list entry, does nothing.
 *
 * @returns the actions that undo this one, or an empty list when it did nothing.
 * @throws {Error} if the document cannot hold the action.
 */
export const apply = (
  doc: record.Unknown,
  action: Action,
  { snapshot = identity }: ApplyOptions = {},
): Action[] => {
  switch (action.type) {
    case "set":
      return applySet(doc, action.set, snapshot);
    case "remove":
      return applyRemove(doc, action.remove, snapshot);
    case "insert":
      return applyInsert(doc, action.insert);
    case "move":
      return applyMove(doc, action.move);
  }
};

/**
 * @returns the path of the value that the action changes: the path of a set, or the
 * path of a list or object followed by the key of the entry or field.
 */
export const pathOf = (action: Action): string[] => {
  switch (action.type) {
    case "set":
      return action.set.path;
    case "remove":
      return [...action.remove.path, action.remove.key];
    case "insert": {
      const key = keyOf(action.insert.value);
      return key == null ? action.insert.path : [...action.insert.path, key];
    }
    case "move":
      return [...action.move.path, action.move.key];
  }
};

/**
 * @returns the action with the path before its own path and the path of each of its
 * guards, so an action written against a part of a document applies to the document.
 */
export const prefix = (action: Action, path: string[]): Action => {
  if (path.length === 0) return action;
  const withPrefix = <P extends { path: string[]; guards: Guard[] }>(p: P): P => ({
    ...p,
    path: [...path, ...p.path],
    guards: p.guards.map((g) => ({ ...g, path: [...path, ...g.path] })),
  });
  switch (action.type) {
    case "set":
      return { type: "set", set: withPrefix(action.set) };
    case "remove":
      return { type: "remove", remove: withPrefix(action.remove) };
    case "insert":
      return { type: "insert", insert: withPrefix(action.insert) };
    case "move":
      return { type: "move", move: withPrefix(action.move) };
  }
};

type Snapshot = NonNullable<ApplyOptions["snapshot"]>;

const applySet = (doc: record.Unknown, a: Set, snapshot: Snapshot): Action[] => {
  const [parentPath, last] = split(a.path);
  if (!guardsHold(doc, a.guards)) return [];
  const found = descend(doc, parentPath, true);
  if (found == null) return [];
  const { node, created } = found;
  if (Array.isArray(node)) {
    const i = entryIndex(node, last);
    if (i < 0) return [];
    if (keyOf(a.value) !== last)
      throw new Error(
        `${format(a.path)}: the entry must be an object with the key ${last}`,
      );
    const prev = snapshot(node[i]);
    node[i] = structuredClone(a.value);
    return [set(a.path, prev, a.guards)];
  }
  if (!narrow.isObject(node)) throw notContainer(parentPath);
  if (created != null) {
    node[last] = structuredClone(a.value);
    return [undoCreate(a.path, created, a.guards)];
  }
  const existed = Object.hasOwn(node, last);
  const prev = snapshot(node[last]);
  node[last] = structuredClone(a.value);
  if (!existed) return [remove(parentPath, last, a.guards)];
  return [set(a.path, prev, a.guards)];
};

const applyRemove = (doc: record.Unknown, a: Remove, snapshot: Snapshot): Action[] => {
  if (!guardsHold(doc, a.guards)) return [];
  const found = descend(doc, a.path, false);
  if (found == null) return [];
  const { node } = found;
  if (Array.isArray(node)) {
    let inverse: Action[] = [];
    for (let i = node.length - 1; i >= 0; i--) {
      if (keyOf(node[i]) !== a.key) continue;
      const after = i > 0 ? keyOf(node[i - 1]) : undefined;
      inverse = [insert(a.path, snapshot(node[i]), after, a.guards)];
      node.splice(i, 1);
    }
    return inverse;
  }
  if (!narrow.isObject(node)) throw notContainer(a.path);
  if (!Object.hasOwn(node, a.key)) return [];
  const prev = snapshot(node[a.key]);
  delete node[a.key];
  return [set([...a.path, a.key], prev, a.guards)];
};

const applyInsert = (doc: record.Unknown, a: Insert): Action[] => {
  const [parentPath, last] = split(a.path);
  if (!guardsHold(doc, a.guards)) return [];
  const key = keyOf(a.value);
  if (key == null)
    throw new Error(`${format(a.path)}: the entry must be an object with a string key`);
  const found = descend(doc, parentPath, true);
  if (found == null) return [];
  const { node, created } = found;
  if (!narrow.isObject(node)) throw notList(a.path);
  let list = node[last];
  const listCreated = list == null;
  if (listCreated) {
    node[last] = [];
    list = node[last];
  }
  if (!Array.isArray(list)) throw notList(a.path);
  if (entryIndex(list, key) >= 0) return [];
  list.splice(position(list, a.after), 0, structuredClone(a.value));
  if (created != null) return [undoCreate(a.path, created, a.guards)];
  if (listCreated) return [remove(parentPath, last, a.guards)];
  return [remove(a.path, key, a.guards)];
};

const applyMove = (doc: record.Unknown, a: Move): Action[] => {
  if (!guardsHold(doc, a.guards)) return [];
  const found = descend(doc, a.path, false);
  if (found == null) return [];
  const list = found.node;
  if (!Array.isArray(list)) throw notList(a.path);
  const i = entryIndex(list, a.key);
  if (i < 0 || a.after === a.key) return [];
  const prev = i > 0 ? keyOf(list[i - 1]) : undefined;
  const [entry] = list.splice(i, 1);
  list.splice(position(list, a.after), 0, entry);
  return [move(a.path, a.key, prev, a.guards)];
};

interface Found {
  node: unknown;
  /** The first object that descend created, if it created one. */
  created?: Created;
}

interface Created {
  depth: number;
  /** Whether the replaced field held null rather than nothing. */
  wasNull: boolean;
}

/**
 * Walks the path from the document. Creates a missing object on the path when create
 * is set. Returns null when a list entry, or an object that is not created, is missing.
 */
const descend = (
  doc: record.Unknown,
  path: string[],
  create: boolean,
): Found | null => {
  let node: unknown = doc;
  let created: Created | undefined;
  for (let depth = 0; depth < path.length; depth++) {
    const seg = path[depth];
    if (Array.isArray(node)) {
      const i = entryIndex(node, seg);
      if (i < 0) return null;
      node = node[i];
      continue;
    }
    if (!narrow.isObject(node)) throw notContainer(path.slice(0, depth));
    if (node[seg] == null) {
      if (!create) return null;
      created ??= { depth, wasNull: node[seg] === null };
      node[seg] = {};
    }
    node = node[seg];
  }
  return { node, created };
};

const guardsHold = (doc: record.Unknown, guards: Guard[]): boolean =>
  guards.every(({ path, value }) => {
    if (path.length === 0) throw new Error("a guard needs a path");
    let node: unknown = doc;
    for (const seg of path)
      if (Array.isArray(node)) {
        const i = entryIndex(node, seg);
        if (i < 0) return false;
        node = node[i];
      } else if (narrow.isObject(node)) node = node[seg];
      else return false;
    return node === value;
  });

const undoCreate = (path: string[], created: Created, guards: Guard[]): Action => {
  const createdPath = path.slice(0, created.depth + 1);
  if (created.wasNull) return set(createdPath, null, guards);
  return remove(createdPath.slice(0, -1), createdPath[createdPath.length - 1], guards);
};

const split = (path: string[]): [string[], string] => {
  if (path.length === 0) throw new Error("the path is empty");
  return [path.slice(0, -1), path[path.length - 1]];
};

const position = (list: unknown[], after: string | undefined): number => {
  if (after == null) return 0;
  const i = entryIndex(list, after);
  return i < 0 ? list.length : i + 1;
};

const keyOf = (entry: unknown): string | undefined => {
  if (!narrow.isObject(entry)) return undefined;
  return typeof entry.key === "string" ? entry.key : undefined;
};

const entryIndex = (list: unknown[], key: string): number =>
  list.findIndex((e) => keyOf(e) === key);

const format = (path: string[]): string => `[${path.join(", ")}]`;

const notContainer = (path: string[]): Error =>
  new Error(`${format(path)}: the value is not an object or a list`);

const notList = (path: string[]): Error =>
  new Error(`${format(path)}: the value is not a list`);

const set = (path: string[], value: unknown, guards: Guard[]): Action => ({
  type: "set",
  set: { path, value, guards },
});

const remove = (path: string[], key: string, guards: Guard[]): Action => ({
  type: "remove",
  remove: { path, key, guards },
});

const insert = (
  path: string[],
  value: unknown,
  after: string | undefined,
  guards: Guard[],
): Action => ({ type: "insert", insert: { path, value, after, guards } });

const move = (
  path: string[],
  key: string,
  after: string | undefined,
  guards: Guard[],
): Action => ({ type: "move", move: { path, key, after, guards } });
