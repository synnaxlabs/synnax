// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/Page";

export const VISUALIZE_NAV: PageNavNode = {
  key: "visualize",
  name: "Visualize",
  children: [
    {
      key: "/reference/visualize",
      href: "/reference/visualize",
      name: "Overview",
    },
    {
      key: "/reference/visualize/projects",
      href: "/reference/visualize/projects",
      name: "Projects",
    },
    {
      key: "/reference/visualize/panels",
      href: "/reference/visualize/panels",
      name: "Panels and tabs",
    },
    {
      key: "/reference/visualize/line-plots",
      href: "/reference/visualize/line-plots",
      name: "Line plots",
    },
    {
      key: "/reference/visualize/schematics",
      href: "/reference/visualize/schematics",
      name: "Schematics",
    },
    {
      key: "/reference/visualize/custom-symbols",
      href: "/reference/visualize/custom-symbols",
      name: "Custom symbols",
    },
    {
      key: "/reference/visualize/tables",
      href: "/reference/visualize/tables",
      name: "Tables",
    },
    {
      key: "/reference/visualize/logs",
      href: "/reference/visualize/logs",
      name: "Logs",
    },
    {
      key: "/reference/visualize/share",
      href: "/reference/visualize/share",
      name: "Share",
    },
  ],
};
