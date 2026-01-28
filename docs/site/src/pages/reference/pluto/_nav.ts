// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav/PageNav";

export const PLUTO_NAV: PageNavNode = {
  key: "pluto",
  name: "Pluto Components",
  children: [
    {
      name: "Get Started",
      key: "/reference/pluto/get-started",
      href: "/reference/pluto/get-started",
    },
    {
      name: "Provider and Canvas",
      key: "/reference/pluto/provider-and-canvas",
      href: "/reference/pluto/provider-and-canvas",
    },
    {
      name: "Theming",
      key: "/reference/pluto/theming",
      href: "/reference/pluto/theming",
    },
    {
      name: "Line Plot",
      key: "/reference/pluto/line-plot",
      href: "/reference/pluto/line-plot",
    },
    {
      name: "Example App",
      key: "/reference/pluto/example-app",
      href: "/reference/pluto/example-app",
    },
  ],
};
