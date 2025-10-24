// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav/PageNav";

export const CONSOLE_NAV: PageNavNode = {
  key: "console",
  name: "Console",
  children: [
    {
      key: "/reference/console/get-started",
      href: "/reference/console/get-started",
      name: "Get Started",
    },
    {
      key: "/reference/console/requirements",
      href: "/reference/console/requirements",
      name: "Requirements",
    },
    {
      key: "/reference/console/cores",
      href: "/reference/console/cores",
      name: "Cores",
    },
    {
      key: "/reference/console/ui-overview",
      href: "/reference/console/ui-overview",
      name: "UI Overview",
    },
    {
      key: "/reference/console/channels",
      href: "/reference/console/channels",
      name: "Channels",
    },
    {
      key: "/reference/console/calculated-channels",
      href: "/reference/console/calculated-channels",
      name: "Calculated Channels",
    },
    {
      key: "/reference/console/workspaces",
      href: "/reference/console/workspaces",
      name: "Workspaces",
    },
    {
      key: "/reference/console/ranges",
      href: "/reference/console/ranges",
      name: "Ranges",
    },
    {
      key: "/reference/console/line-plots",
      href: "/reference/console/line-plots",
      name: "Line Plots",
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
