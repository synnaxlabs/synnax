// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/Page";
import { CLIENT_NAV } from "@/pages/reference/client/_nav";
import { CONCEPTS_NAV } from "@/pages/reference/concepts/_nav";
import { CONSOLE_NAV } from "@/pages/reference/console/_nav";
import { CONTROL_NAV } from "@/pages/reference/control/_nav";
import { CORE_NAV } from "@/pages/reference/core/_nav";
import { DRIVER_NAV } from "@/pages/reference/driver/_nav";
// import { PLUTO_NAV } from "@/pages/reference/pluto/_nav";

// Pluto Components nav is temporarily hidden; content remains in docs/ for future fixup.
// - Is this overall section helpful? To whom?
// - pluto/line-plot.mdx example is broken.

export const REFERENCE_PAGES: PageNavNode[] = [
  { name: "Get Started", key: "/reference/", href: "/reference/" },
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
];
