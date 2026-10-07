// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Edition } from "@/components/edition/edition";
import { type PageNavNode } from "@/components/nav/Page";
import { normalizeRoute } from "@/util/route";

/** Lists every node in depth-first order, parents before their children. */
export const flatten = (nodes: PageNavNode[]): PageNavNode[] =>
  nodes.flatMap((node) => [node, ...flatten(node.children ?? [])]);

export interface Neighbors {
  prev?: PageNavNode;
  next?: PageNavNode;
}

/**
 * Returns the pages before and after the page at path in reading order. Both are
 * undefined when no node links to path.
 */
export const neighbors = (nodes: PageNavNode[], path: string): Neighbors => {
  const pages = flatten(nodes).filter((node) => node.href != null);
  const target = normalizeRoute(path);
  const i = pages.findIndex((node) => normalizeRoute(node.href as string) === target);
  if (i === -1) return {};
  return { prev: pages[i - 1], next: pages[i + 1] };
};

/**
 * Returns the edition of the page at path: the nearest edition set on the page's node
 * or on one of its ancestors. Undefined means the page applies to every edition.
 */
export const editionOf = (nodes: PageNavNode[], path: string): Edition | undefined => {
  const target = normalizeRoute(path);
  const find = (
    level: PageNavNode[],
    inherited?: Edition,
  ): { edition?: Edition } | null => {
    for (const node of level) {
      const edition = node.edition ?? inherited;
      if (node.href != null && normalizeRoute(node.href) === target) return { edition };
      const found = find(node.children ?? [], edition);
      if (found != null) return found;
    }
    return null;
  };
  return find(nodes)?.edition;
};

/** Returns the name of the nearest group that holds the page at path, if one does. */
export const groupOf = (nodes: PageNavNode[], path: string): string | undefined => {
  const target = normalizeRoute(path);
  // Null means the page is not in level. Undefined means it is, outside every group.
  const find = (level: PageNavNode[], group?: string): string | undefined | null => {
    for (const node of level) {
      if (node.href != null && normalizeRoute(node.href) === target) return group;
      if (node.children == null) continue;
      const found = find(node.children, node.name);
      if (found !== null) return found;
    }
    return null;
  };
  return find(nodes) ?? undefined;
};
