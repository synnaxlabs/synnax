// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav/PageNav";

export const consoleNav: PageNavNode = {
  key: "console",
  name: "Console",
  children: [
    {
      key: "/reference/console/get-started",
      href: "/reference/console/get-started",
      name: "Get Started",
    },
    {
      key: "/reference/console/connect-a-cluster",
      href: "/reference/console/connect-a-cluster",
      name: "Connect a Cluster",
    },
    {
      key: "/reference/console/ui-overview",
      href: "/reference/console/ui-overview",
      name: "UI Overview",
    },
    {
      key: "/reference/console/workspaces",
      href: "/reference/console/workspaces",
      name: "Workspaces",
    },
    {
      key: "/reference/console/querying-data",
      href: "/reference/console/querying-data",
      name: "Querying Data Using Ranges",
    },
    {
      key: "/reference/console/line-plot",
      href: "/reference/console/line-plot",
      name: "Line Plot Visualization",
    },
    {
      key: "/reference/console/schematic",
      href: "/reference/console/schematic",
      name: "Schematic Visualization",
    },
  ],
};
