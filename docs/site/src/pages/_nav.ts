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
import { CLIENT_NAV } from "@/pages/reference/client/_nav";
import { CONCEPTS_NAV } from "@/pages/reference/concepts/_nav";
import { CONSOLE_NAV } from "@/pages/reference/console/_nav";
import { CONTROL_NAV } from "@/pages/reference/control/_nav";
import { CORE_NAV } from "@/pages/reference/core/_nav";
import { DRIVER_NAV } from "@/pages/reference/driver/_nav";
import { PLUTO_NAV } from "@/pages/reference/pluto/_nav";

const visible = (nodes: PageNavNode[]): PageNavNode[] =>
  nodes
    .filter(({ flag }) => flag == null || FLAGS[flag])
    .map((node) =>
      node.children == null ? node : { ...node, children: visible(node.children) },
    );

export const REFERENCE_PAGES: PageNavNode[] = visible([
  { name: "Get started", key: "/reference/", href: "/reference/" },
  {
    name: "Installation",
    key: "/reference/installation",
    href: "/reference/installation",
  },
  CONCEPTS_NAV,
  CORE_NAV,
  CONTROL_NAV,
  CONSOLE_NAV,
  CLIENT_NAV,
  DRIVER_NAV,
  PLUTO_NAV,
]);
