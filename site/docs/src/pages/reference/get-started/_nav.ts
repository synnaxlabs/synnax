// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Nav } from "@/components/nav";

export const GET_STARTED_NAV: Nav.PageNavNode = {
  key: "get-started",
  name: "Get started",
  children: [
    {
      href: "/reference/",
      name: "Overview",
    },
    {
      href: "/reference/get-started/concepts",
      name: "Concepts",
    },
    {
      href: "/reference/get-started/editions",
      name: "Editions",
    },
    {
      href: "/reference/get-started/install-desktop",
      name: "Install Synnax Desktop",
      edition: "desktop",
    },
    {
      href: "/reference/get-started/sample-project",
      name: "Sample project",
      edition: "desktop",
    },
    {
      href: "/reference/get-started/first-run",
      name: "Your first run",
    },
    {
      href: "/reference/get-started/ui-overview",
      name: "UI overview",
    },
  ],
};
