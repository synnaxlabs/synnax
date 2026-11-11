// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger, type Synnax as Client } from "@synnaxlabs/client";
import { useMemoDeepEqual } from "@synnaxlabs/lyra/memo";
import { Synnax } from "@synnaxlabs/pluto";
import { compare, TimeStamp, unique } from "@synnaxlabs/x";
import { useEffect, useState } from "react";

/** The most matches a search shows in a tree. */
export const MATCH_LIMIT = 100;

interface Placed {
  roots: ranger.Key[];
  /** The ranges above the matches that do not match, each with its children. */
  fixed: Map<ranger.Key, ranger.Key[]>;
  /** Every range that holds a match. */
  holders: ranger.Key[];
}

/** Reads the ranges above the matches, one level at a time. */
const readAbove = async (
  client: Client,
  matches: ranger.Range[],
): Promise<ranger.Range[]> => {
  const known = new Set(matches.map((r) => r.key));
  const above: ranger.Range[] = [];
  let level = matches;
  for (;;) {
    const missing = unique.unique(
      level.flatMap((r) => (r.parent == null || known.has(r.parent) ? [] : [r.parent])),
    );
    if (missing.length === 0) return above;
    missing.forEach((key) => known.add(key));
    level = await client.ranges.retrieve(missing);
    above.push(...level);
  }
};

const compareBy =
  ({ field, descending }: ranger.Order) =>
  (a: ranger.Range, b: ranger.Range): number => {
    const c =
      field === "name"
        ? compare.text(a.name, b.name)
        : TimeStamp.sort(a.timeRange.start, b.timeRange.start);
    return descending ? -c : c;
  };

/** Places each match under its parent, with the ranges of each level in order. */
const place = (
  matches: ranger.Range[],
  above: ranger.Range[],
  order: ranger.Order,
): Placed => {
  const records = new Map([...above, ...matches].map((r) => [r.key, r]));
  const roots: ranger.Key[] = [];
  const children = new Map<ranger.Key, ranger.Key[]>();
  const placed = new Set<ranger.Key>();
  const visit = (range: ranger.Range): void => {
    if (placed.has(range.key)) return;
    placed.add(range.key);
    const parent = range.parent == null ? undefined : records.get(range.parent);
    if (parent == null) {
      roots.push(range.key);
      return;
    }
    children.set(parent.key, [...(children.get(parent.key) ?? []), range.key]);
    visit(parent);
  };
  matches.forEach(visit);
  const compareRanges = compareBy(order);
  const sort = (keys: ranger.Key[]) =>
    keys.toSorted((a, b) => compareRanges(records.get(a)!, records.get(b)!));
  children.forEach((keys, key) => children.set(key, sort(keys)));
  const matched = new Set(matches.map((r) => r.key));
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
  error: Error | null;
}

interface Answer {
  links: unknown;
  /** The matches and the ranges above them, or null until the first read lands. */
  read: { matches: ranger.Range[]; above: ranger.Range[] } | null;
  error: Error | null;
}

const EMPTY: Placed = { roots: [], fixed: new Map(), holders: [] };

/**
 * Shows search matches in their tree: each match under the ranges above it, with those
 * ranges open.
 */
export const useSearchTree = ({
  matches,
  scope,
  order,
}: UseSearchTreeParams): SearchTree => {
  const client = Synnax.use();
  const links = useMemoDeepEqual(
    matches?.map(({ key, parent }) => ({ key, parent })) ?? null,
  );
  const [answer, setAnswer] = useState<Answer | null>(null);
  if (links == null && answer != null) setAnswer(null);
  useEffect(() => {
    if (client == null || matches == null) return;
    let current = true;
    readAbove(client, matches).then(
      (above) => {
        if (current) setAnswer({ links, read: { matches, above }, error: null });
      },
      (error: unknown) => {
        if (!current) return;
        const err = error instanceof Error ? error : new Error(String(error));
        setAnswer((prev) => ({ links, read: prev?.read ?? null, error: err }));
      },
    );
    return () => {
      current = false;
    };
  }, [client, links]);
  const [toggled, setToggled] = useState(() => ({
    scope,
    open: new Map<ranger.Key, boolean>(),
  }));
  if (toggled.scope !== scope) setToggled({ scope, open: new Map() });
  const placed =
    answer?.read == null ? EMPTY : place(answer.read.matches, answer.read.above, order);
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
    answered: answer?.links === links && answer.read != null,
    error: answer?.links === links ? answer.error : null,
  };
};
