// Copyright 2023 Synnax Labs, Inc.
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
      key: "/components/console/get-started",
      href: "/components/console/get-started",
      name: "Get Started",
    },
    {
      key: "/components/console/connect-a-cluster",
      href: "/components/console/connect-a-cluster",
      name: "Connect a Cluster",
    },
    {
      key: "/components/console/ui-overview",
      href: "/components/console/ui-overview",
      name: "UI Overview",
    },
    {
      key: "/components/console/workspaces",
      href: "/components/console/workspaces",
      name: "Workspaces",
    },
    {
      key: "/components/console/querying-data",
      href: "/components/console/querying-data",
      name: "Querying Data Using Ranges",
    },
    {
      key: "/components/console/line-plot",
      href: "/components/console/line-plot",
      name: "Line Plot Visualization",
    },
    {
      key: "/components/console/schematic",
      href: "/components/console/schematic",
      name: "Schematic Visualization",
    },
  ],
};
