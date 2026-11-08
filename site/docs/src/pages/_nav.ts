// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Nav } from "@/components/nav";
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

const visible = (nodes: Nav.PageNavNode[]): Nav.PageNavNode[] =>
  nodes
    .filter(({ flag }) => flag == null || FLAGS[flag])
    .map((node) =>
      node.children == null ? node : { ...node, children: visible(node.children) },
    );

export const REFERENCE_PAGES: Nav.PageNavNode[] = visible([
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

const LABELS = Nav.labels(REFERENCE_PAGES);

/**
 * Returns the nav name of a route. A route that the pages of a group share, such as the
 * route of its overview page, takes the name of the group.
 */
export const labelOf = (route: string): string | undefined =>
  LABELS.get(normalizeRoute(route));

/** Maps the route of each reference group without a page to the group's first page. */
export const FIRST_PAGES = Nav.firstPages(REFERENCE_PAGES, "/reference");
