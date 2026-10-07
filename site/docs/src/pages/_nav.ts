// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/Page";
import { FLAGS } from "@/flags";
import { ALARMS_NAV } from "@/pages/reference/alarms/_nav";
import { CHANNELS_NAV } from "@/pages/reference/channels/_nav";
import { CLIENT_NAV } from "@/pages/reference/client/_nav";
import { CONTROL_NAV } from "@/pages/reference/control/_nav";
import { DESKTOP_NAV } from "@/pages/reference/desktop/_nav";
import { ENTERPRISE_NAV } from "@/pages/reference/enterprise/_nav";
import { GET_STARTED_NAV } from "@/pages/reference/get-started/_nav";
import { HARDWARE_NAV } from "@/pages/reference/hardware/_nav";
import { RANGES_NAV } from "@/pages/reference/ranges/_nav";
import { VISUALIZE_NAV } from "@/pages/reference/visualize/_nav";
import { normalizeRoute } from "@/util/route";

const visible = (nodes: PageNavNode[]): PageNavNode[] =>
  nodes
    .filter(({ flag }) => flag == null || FLAGS[flag])
    .map((node) =>
      node.children == null ? node : { ...node, children: visible(node.children) },
    );

export const REFERENCE_PAGES: PageNavNode[] = visible([
  GET_STARTED_NAV,
  CHANNELS_NAV,
  HARDWARE_NAV,
  VISUALIZE_NAV,
  RANGES_NAV,
  CONTROL_NAV,
  ALARMS_NAV,
  CLIENT_NAV,
  DESKTOP_NAV,
  ENTERPRISE_NAV,
]);

const contains = (node: PageNavNode, route: string): boolean =>
  (node.href != null && normalizeRoute(node.href) === route) ||
  (node.children?.some((child) => contains(child, route)) ?? false);

/** Returns the name of the nav section that holds a route, if one does. */
export const sectionOf = (route: string): string | undefined =>
  REFERENCE_PAGES.find((node) => node.children != null && contains(node, route))?.name;

const routesOf = (node: PageNavNode): string[] => [
  ...(node.href == null ? [] : [normalizeRoute(node.href)]),
  ...(node.children ?? []).flatMap(routesOf),
];

const commonPath = (routes: string[]): string => {
  const [first, ...rest] = routes.map((route) => route.split("/"));
  const end = first.findIndex((part, i) => rest.some((parts) => parts[i] !== part));
  return (end === -1 ? first : first.slice(0, end)).join("/");
};

// A section names the route that its pages share, unless an enclosing section shares
// the same route. The section entry comes last, so it wins over its overview page.
const labels = (nodes: PageNavNode[], parent: string): [string, string][] =>
  nodes.flatMap((node): [string, string][] => {
    if (node.children == null)
      return node.href == null ? [] : [[normalizeRoute(node.href), node.name]];
    const shared = commonPath(routesOf(node));
    if (shared === parent) return labels(node.children, parent);
    return [...labels(node.children, shared), [shared, node.name]];
  });

const ROOT = commonPath(REFERENCE_PAGES.flatMap(routesOf));

const LABELS: ReadonlyMap<string, string> = new Map(
  labels(REFERENCE_PAGES, ROOT).filter(([route]) => route !== ROOT),
);

/**
 * Returns the nav name of a route. A route that a section's pages share, such as the
 * route of its overview page, takes the name of the section.
 */
export const labelOf = (route: string): string | undefined =>
  LABELS.get(normalizeRoute(route));
