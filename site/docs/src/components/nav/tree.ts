// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Edition } from "@/components/edition/edition";
import { type Flag } from "@/flags";
import { normalizeRoute } from "@/util/route";

interface Base {
  name: string;
  /** Hides the node and its children while the flag is off. */
  flag?: Flag;
  /** Marks the node and its children as applying only to this edition. */
  edition?: Edition;
  children?: PageNavNode[];
}

/** A node that links to a page. Its href identifies it. */
interface PageNode extends Base {
  href: string;
  key?: never;
}

/** A node without a page that holds other nodes. Its key identifies it. */
interface GroupNode extends Base {
  key: string;
  href?: never;
  children: PageNavNode[];
}

/** A node of the reference nav. */
export type PageNavNode = PageNode | GroupNode;

const isPage = (node: PageNavNode): node is PageNode => node.href != null;

const flatten = (nodes: PageNavNode[]): PageNavNode[] =>
  nodes.flatMap((node) => [node, ...flatten(node.children ?? [])]);

/**
 * Returns the nodes from a top-level node down to the node that links to path. The list
 * is empty when no node links to path.
 */
export const ancestry = (nodes: PageNavNode[], path: string): PageNavNode[] => {
  const target = normalizeRoute(path);
  for (const node of nodes) {
    if (node.href != null && normalizeRoute(node.href) === target) return [node];
    const below = ancestry(node.children ?? [], target);
    if (below.length > 0) return [node, ...below];
  }
  return [];
};

export interface Neighbors {
  prev?: PageNavNode;
  next?: PageNavNode;
}

/**
 * Returns the pages before and after the page at path in reading order. Both are
 * undefined when no node links to path.
 */
export const neighbors = (nodes: PageNavNode[], path: string): Neighbors => {
  const pages = flatten(nodes).filter(isPage);
  const target = normalizeRoute(path);
  const i = pages.findIndex((node) => normalizeRoute(node.href) === target);
  if (i === -1) return {};
  return { prev: pages[i - 1], next: pages[i + 1] };
};

/**
 * Returns the edition of the page at path: the nearest edition set on the page's node
 * or on one of its ancestors. Undefined means the page applies to every edition.
 */
export const editionOf = (nodes: PageNavNode[], path: string): Edition | undefined =>
  ancestry(nodes, path).findLast(({ edition }) => edition != null)?.edition;

/** Returns the name of the nearest group that holds the page at path, if one does. */
export const groupOf = (nodes: PageNavNode[], path: string): string | undefined =>
  ancestry(nodes, path).at(-2)?.name;

/** Returns the name of the top-level group that holds the page at path, if one does. */
export const sectionOf = (nodes: PageNavNode[], path: string): string | undefined => {
  const [top] = ancestry(nodes, path);
  return top?.children == null ? undefined : top.name;
};

const routesOf = (node: PageNavNode): string[] => [
  ...(node.href == null ? [] : [normalizeRoute(node.href)]),
  ...(node.children ?? []).flatMap(routesOf),
];

const commonPath = (routes: string[]): string => {
  const [first, ...rest] = routes.map((route) => route.split("/"));
  const end = first.findIndex((part, i) => rest.some((parts) => parts[i] !== part));
  return (end === -1 ? first : first.slice(0, end)).join("/");
};

// A group's entry follows the entries of its descendants, so the outermost group that
// shares a route names it.
const entries = (nodes: PageNavNode[]): [string, string][] =>
  nodes.flatMap((node): [string, string][] => {
    if (node.children == null)
      return node.href == null ? [] : [[normalizeRoute(node.href), node.name]];
    return [...entries(node.children), [commonPath(routesOf(node)), node.name]];
  });

/**
 * Maps the route of each page to its name in the nav. A route that the pages of a group
 * share, such as the route of its overview page, maps to the name of the outermost such
 * group. The route that every page shares has no name.
 */
export const labels = (nodes: PageNavNode[]): ReadonlyMap<string, string> => {
  const root = commonPath(nodes.flatMap(routesOf));
  return new Map(entries(nodes).filter(([route]) => route !== root));
};
