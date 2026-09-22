// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/Page";

export const PLUTO_NAV: PageNavNode = {
  key: "pluto",
  name: "Pluto components",
  icon: "Visualize",
  children: [
    {
      name: "Get started",
      key: "/reference/pluto/get-started",
      href: "/reference/pluto/get-started",
    },
    {
      name: "Provider and canvas",
      key: "/reference/pluto/provider-and-canvas",
      href: "/reference/pluto/provider-and-canvas",
    },
    {
      name: "Theming",
      key: "/reference/pluto/theming",
      href: "/reference/pluto/theming",
    },
    {
      name: "Line plot",
      key: "/reference/pluto/line-plot",
      href: "/reference/pluto/line-plot",
    },
    {
      name: "Example app",
      key: "/reference/pluto/example-app",
      href: "/reference/pluto/example-app",
    },
  ],
};
