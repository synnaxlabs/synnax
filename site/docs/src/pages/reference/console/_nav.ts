// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/tree";

export const CONSOLE_NAV: PageNavNode = {
  key: "console",
  name: "Console",
  children: [
    {
      href: "/reference/console/get-started",
      name: "Get started",
    },
    {
      href: "/reference/console/ui-overview",
      name: "UI overview",
    },
    {
      href: "/reference/console/projects",
      name: "Projects",
    },
    {
      href: "/reference/console/line-plots",
      name: "Line plots",
    },
    { href: "/reference/console/logs", name: "Logs" },
    {
      href: "/reference/console/schematics",
      name: "Schematics",
    },
    {
      href: "/reference/console/tables",
      name: "Tables",
    },
    {
      href: "/reference/console/users",
      name: "Users",
    },
  ],
};
