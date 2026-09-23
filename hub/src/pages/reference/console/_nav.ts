// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/Page";

export const CONSOLE_NAV: PageNavNode = {
  key: "console",
  name: "Console",
  icon: "Dashboard",
  children: [
    {
      key: "/reference/console/get-started",
      href: "/reference/console/get-started",
      name: "Get started",
    },
    {
      key: "/reference/console/ui-overview",
      href: "/reference/console/ui-overview",
      name: "UI overview",
    },
    {
      key: "/reference/console/projects",
      href: "/reference/console/projects",
      name: "Projects",
    },
    {
      key: "/reference/console/line-plots",
      href: "/reference/console/line-plots",
      name: "Line plots",
    },
    { key: "/reference/console/logs", href: "/reference/console/logs", name: "Logs" },
    {
      key: "/reference/console/schematics",
      href: "/reference/console/schematics",
      name: "Schematics",
    },
    {
      key: "/reference/console/tables",
      href: "/reference/console/tables",
      name: "Tables",
    },
    {
      key: "/reference/console/users",
      href: "/reference/console/users",
      name: "Users",
    },
  ],
};
