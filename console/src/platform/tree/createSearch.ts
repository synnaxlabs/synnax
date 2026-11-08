// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { resource } from "@synnaxlabs/client";
import { type Tree as Base } from "@synnaxlabs/lyra/tree";
import { unique } from "@synnaxlabs/x";
import { useEffect, useRef, useState } from "react";
import { flushSync } from "react-dom";

import { DefaultItem, type Item } from "@/platform/tree/item";
import { useItems } from "@/platform/tree/Provider";
import {
  type CreateSearchParams,
  type Match,
  type Matches,
  type Placement,
  type Search,
  type SearchAnswer,
  type SearchLayout,
  type UseSearch,
} from "@/platform/tree/search";
import { type Results } from "@/platform/tree/store";
import { type Resource } from "@/platform/tree/types";

export const LIMIT = 100;
const DEBOUNCE_MS = 120;

type FoundMatches = Extract<Matches, { variant: "success" }>;
type FailedMatches = Extract<Matches, { variant: "error" }>;

const isFound = (matches: Matches): matches is FoundMatches =>
  matches.variant === "success";

const isFailed = (matches: Matches): matches is FailedMatches =>
  matches.variant === "error";

const foldersFirst = (
  nodes: Base.Node<string>[],
  resolveItem: (type: resource.Type) => Item,
): Base.Node<string>[] => {
  const isFolder = ({ key }: Base.Node<string>): boolean =>
    resolveItem(resource.parseID(key).type).folder;
  return [...nodes.filter(isFolder), ...nodes.filter((node) => !isFolder(node))].map(
    (node) =>
      node.children == null
        ? node
        : { ...node, children: foldersFirst(node.children, resolveItem) },
  );
};

interface BuildResultsParams {
  matches: Resource[];
  placement: Placement;
  layout: SearchLayout;
  resolveItem: (type: resource.Type) => Item;
}

interface Built {
  results: Results;
  groups: Map<string, string>;
}

const buildResults = ({
  matches,
  placement: { resources, paths },
  layout,
  resolveItem,
}: BuildResultsParams): Built => {
  const names = new Map(resources.map(({ key, name }) => [key, name]));
  const ids = matches.map(({ id }) => id);
  const matched = new Set(resource.idToString(ids));
  const nodes: Base.Node<string>[] = [];
  const above = new Map<string, resource.ID[]>();
  const ancestors = new Map<string, Set<string>>();
  const groups = new Map<string, string>();
  const fixed = new Set<string>();
  const expanded = new Set<string>();
  const toNode = (id: resource.ID): Base.Node<string> => ({
    key: resource.idToString(id),
    children: resolveItem(id.type).hasChildren ? [] : undefined,
  });
  ids.forEach((id) => {
    const key = resource.idToString(id);
    const idPaths = paths[key] ?? [];
    // The tree shows only what sits under its root.
    if (idPaths.length === 0) return;
    const parent = idPaths[0].at(-1);
    above.set(key, idPaths[0].toReversed());
    ancestors.set(key, new Set(idPaths.flatMap((path) => resource.idToString(path))));
    if (layout === "flat") {
      const group = parent == null ? undefined : names.get(resource.idToString(parent));
      if (group != null) groups.set(key, group);
      nodes.push(toNode(id));
      return;
    }
    idPaths.forEach((path) => {
      let level = nodes;
      path.forEach((groupID) => {
        const groupKey = resource.idToString(groupID);
        expanded.add(groupKey);
        if (!matched.has(groupKey)) fixed.add(groupKey);
        let node = level.find((n) => n.key === groupKey);
        if (node == null) {
          node = { key: groupKey, children: [] };
          level.push(node);
        }
        node.children ??= [];
        level = node.children;
      });
      if (!level.some((n) => n.key === key)) level.push(toNode(id));
    });
  });
  return {
    results: {
      nodes: foldersFirst(nodes, resolveItem),
      fixed,
      expanded: [...expanded],
      paths: above,
      resources: new Map([...resources, ...matches].map((r) => [r.key, r])),
      ancestors,
    },
    groups,
  };
};

interface Held {
  key: string;
  layout: SearchLayout;
  results: Results;
  match: Match;
  more: boolean;
}

const nameKey = ({ key, name }: Resource): string => `${key}\t${name}`;

// Names what the results show, so a change to it rebuilds them.
const keyOf = (term: string, matches: Resource[], placement: Placement): string =>
  JSON.stringify([
    term,
    matches.map(nameKey),
    placement.resources.map(nameKey),
    placement.paths,
  ]);

const useDebounced = (value: string): string => {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const timeout = setTimeout(() => setDebounced(value), DEBOUNCE_MS);
    return () => clearTimeout(timeout);
  }, [value]);
  return debounced;
};

/**
 * Declares the search of a tree. Call it at module scope, and call the hook it returns
 * where both the search field and the tree can read its state.
 */
export const createSearch = ({
  noun,
  useMatches,
  usePlacement,
}: CreateSearchParams): UseSearch => {
  const useSearch = (root: resource.ID | null): Search => {
    const items = useItems();
    const [open, setOpen] = useState(false);
    const [term, setTerm] = useState("");
    const [layout, setLayout] = useState<SearchLayout>("flat");
    const fieldRef = useRef<HTMLInputElement>(null);
    const resultsRef = useRef<HTMLDivElement>(null);
    const treeRef = useRef<HTMLDivElement>(null);
    const searching = open && term.trim() !== "";
    const trimmed = useDebounced(term).trim();
    const sources = useMatches(
      !open || root == null || trimmed === ""
        ? null
        : { term: trimmed, root, limit: LIMIT + 1 },
    );
    const found = sources.every(isFound)
      ? unique.by(
          sources.flatMap(({ resources }) => resources),
          ({ key }) => key,
        )
      : null;
    const shown = found?.slice(0, LIMIT) ?? null;
    const placed = usePlacement(
      shown == null || root == null ? null : { matches: shown, root },
    );
    const placement = placed.variant === "success" ? placed.placement : null;
    const key =
      shown == null || placement == null ? null : keyOf(trimmed, shown, placement);
    const more = found != null && found.length > LIMIT;

    // The last answer shows until the next one is placed, so the results never blank
    // between keystrokes.
    const [held, setHeld] = useState<Held | null>(null);
    if (!searching) {
      if (held != null) setHeld(null);
    } else if (
      key != null &&
      shown != null &&
      placement != null &&
      (held?.key !== key || held.layout !== layout || held.more !== more)
    ) {
      const { results, groups: groupNames } = buildResults({
        matches: shown,
        placement,
        layout,
        resolveItem: (type) => items[type] ?? DefaultItem,
      });
      setHeld({
        key,
        layout,
        results,
        match: { term: trimmed, groups: groupNames },
        more,
      });
    }

    const failure =
      sources.find(isFailed) ?? (placed.variant === "error" ? placed : null);
    let answer: SearchAnswer | null = null;
    if (searching && failure != null)
      answer = { variant: "error", description: failure.description };
    else if (searching && held != null)
      answer = {
        variant: "success",
        results: held.results,
        match: held.match,
        more: held.more,
      };
    return {
      noun,
      open,
      term,
      layout,
      loading: searching && failure == null && (key == null || held?.key !== key),
      answer,
      start: () => setOpen(true),
      close: () => {
        // A hidden tree cannot take the focus, so the close renders first.
        flushSync(() => {
          setOpen(false);
          setTerm("");
        });
        treeRef.current?.focus({ preventScroll: true });
      },
      setTerm,
      setLayout,
      fieldRef,
      resultsRef,
      treeRef,
    };
  };
  return useSearch;
};
