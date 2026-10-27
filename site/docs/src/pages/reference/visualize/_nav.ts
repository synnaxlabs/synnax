// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Nav } from "@/components/nav";

export const VISUALIZE_NAV: Nav.PageNavNode = {
  key: "visualize",
  name: "Visualize",
  children: [
    {
      href: "/reference/visualize",
      name: "Overview",
    },
    {
      href: "/reference/visualize/projects",
      name: "Projects",
    },
    {
      href: "/reference/visualize/panels",
      name: "Panels and tabs",
    },
    {
      href: "/reference/visualize/line-plots",
      name: "Line plots",
    },
    {
      href: "/reference/visualize/schematics",
      name: "Schematics",
    },
    {
      href: "/reference/visualize/custom-symbols",
      name: "Custom symbols",
    },
    {
      href: "/reference/visualize/tables",
      name: "Tables",
    },
    {
      href: "/reference/visualize/logs",
      name: "Logs",
    },
    {
      href: "/reference/visualize/share",
      name: "Share",
    },
  ],
};
