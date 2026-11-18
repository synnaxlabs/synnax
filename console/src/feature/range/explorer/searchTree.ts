// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { Store } from "@synnaxlabs/lyra/store";
import { Ranger } from "@synnaxlabs/pluto";
import { compare, deep, type destructor, TimeStamp, unique } from "@synnaxlabs/x";
import { useState } from "react";

/** The most matches a search shows in a tree. */
export const MATCH_LIMIT = 100;

/** The parent of each range, or null for a range at the top. */
type Links = Record<ranger.Key, ranger.Key | null>;

/** The matches of a search and the parent links above them. */
interface Answer {
  matches: ranger.Key[];
  links: Links;
}

interface Placed {
  roots: ranger.Key[];
  /** The ranges above the matches that do not match, each with its children. */
  fixed: Map<ranger.Key, ranger.Key[]>;
  /** Every range that holds a match. */
  holders: ranger.Key[];
}

const compareBy =
  ({ field, descending }: ranger.Order) =>
  (a: ranger.Range, b: ranger.Range): number => {
    const c =
      field === "name"
        ? compare.text(a.name, b.name)
        : TimeStamp.sort(a.timeRange.start, b.timeRange.start);
    return descending ? -c : c;
  };

/** @returns every range of the answer: the matches and the ranges above them. */
const rangesOf = ({ links }: Answer): ranger.Key[] =>
  unique.unique(
    Object.entries(links).flatMap(([key, parent]) =>
      parent == null ? [key] : [key, parent],
    ),
  );

/** Places each match under its parent, with the ranges of each level in order. */
const place = (
  { matches, links }: Answer,
  ranges: ranger.Range[],
  order: ranger.Order,
): Placed => {
  const roots: ranger.Key[] = [];
  const children = new Map<ranger.Key, ranger.Key[]>();
  const placed = new Set<ranger.Key>();
  const visit = (key: ranger.Key): void => {
    if (placed.has(key)) return;
    placed.add(key);
    const parent = links[key];
    if (parent == null) {
      roots.push(key);
      return;
    }
    children.set(parent, [...(children.get(parent) ?? []), key]);
    visit(parent);
  };
  matches.forEach(visit);
  const records = new Map(ranges.map((r) => [r.key, r]));
  const compareRanges = compareBy(order);
  // The last answer can name ranges deleted since it landed.
  const sort = (keys: ranger.Key[]) =>
    keys
      .filter((key) => records.has(key))
      .toSorted((a, b) => compareRanges(records.get(a)!, records.get(b)!));
  children.forEach((keys, key) => children.set(key, sort(keys)));
  const matched = new Set(matches);
  return {
    roots: sort(roots),
    fixed: new Map([...children].filter(([key]) => !matched.has(key))),
    holders: [...children.keys()],
  };
};

export interface UseSearchTreeParams {
  /** The matches in list order, or null outside a search. */
  matches: ranger.Range[] | null;
  /** Names the search. A new search opens every range that holds a match again. */
  scope: string;
  /** The order of the ranges at each level. */
  order: ranger.Order;
  /** Reads the live records of the matches and of the ranges above them. */
  getItem: (keys: ranger.Key[]) => ranger.Range[];
  /** Calls the listener when the range with the key changes. */
  subscribe: (listener: () => void, key: ranger.Key) => destructor.Destructor;
}

export interface SearchTree {
  roots: ranger.Key[];
  /**
   * The ranges above the matches that do not match. Each shows only the children that
   * lead to a match.
   */
  fixed: Map<ranger.Key, ranger.Key[]>;
  expanded: ranger.Key[];
  onExpand: (key: ranger.Key, expanded: boolean) => void;
  /** Whether the tree holds the current matches. Until then, it holds the last ones. */
  answered: boolean;
  /** Why the ranges above the matches could not be read. */
  error: string | null;
}

const EMPTY: Placed = { roots: [], fixed: new Map(), holders: [] };

const NO_KEYS: ranger.Key[] = [];

/**
 * Shows search matches in their tree: each match under the ranges above it, with those
 * ranges open.
 */
export const useSearchTree = ({
  matches,
  scope,
  order,
  getItem,
  subscribe,
}: UseSearchTreeParams): SearchTree => {
  const matched = new Set(matches?.map(({ key }) => key));
  const tops =
    matches == null
      ? null
      : unique.unique(
          matches.flatMap(({ parent }) =>
            parent == null || matched.has(parent) ? [] : [parent],
          ),
        );
  const above = Ranger.useResultAncestors(tops == null ? null : { keys: tops });
  const current: Answer | null =
    matches == null || above.variant !== "success"
      ? null
      : {
          matches: matches.map(({ key }) => key),
          links: {
            ...above.data,
            ...Object.fromEntries(
              matches.map(({ key, parent }) => [key, parent ?? null]),
            ),
          },
        };
  // A new search shows the last answer until the ranges above its matches land, and
  // nothing once that read fails, so the error shows.
  const [last, setLast] = useState<Answer | null>(null);
  if (matches == null ? last != null : current != null && !deep.equal(current, last))
    setLast(current);
  const shown = above.variant === "error" ? null : (current ?? last);
  const [toggled, setToggled] = useState(() => ({
    scope,
    open: new Map<ranger.Key, boolean>(),
  }));
  if (toggled.scope !== scope) setToggled({ scope, open: new Map() });
  // The tree orders the ranges in render, so it subscribes to them.
  const ranges = Store.useItems({
    keys: shown == null ? NO_KEYS : rangesOf(shown),
    subscribe,
    getItem,
  });
  const placed = shown == null ? EMPTY : place(shown, ranges, order);
  const expanded = unique.unique([
    ...placed.holders.filter((key) => toggled.open.get(key) !== false),
    ...[...toggled.open].flatMap(([key, open]) => (open ? [key] : [])),
  ]);
  return {
    roots: placed.roots,
    fixed: placed.fixed,
    expanded,
    onExpand: (key, open) =>
      setToggled((prev) => ({
        scope: prev.scope,
        open: new Map(prev.open).set(key, open),
      })),
    answered: current != null,
    error: above.variant === "error" ? (above.status.description ?? null) : null,
  };
};
