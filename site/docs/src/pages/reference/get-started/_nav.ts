// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/Page";

export const GET_STARTED_NAV: PageNavNode = {
  key: "get-started",
  name: "Get started",
  children: [
    {
      key: "/reference/",
      href: "/reference/",
      name: "Overview",
    },
    {
      key: "/reference/get-started/editions",
      href: "/reference/get-started/editions",
      name: "Editions",
    },
    {
      key: "/reference/get-started/install-desktop",
      href: "/reference/get-started/install-desktop",
      name: "Install Synnax Desktop",
      flag: "desktop",
      edition: "desktop",
    },
    {
      key: "/reference/get-started/first-run",
      href: "/reference/get-started/first-run",
      name: "Your first run",
    },
    {
      key: "/reference/get-started/ui-overview",
      href: "/reference/get-started/ui-overview",
      name: "UI overview",
    },
    {
      key: "/reference/get-started/manage-desktop",
      href: "/reference/get-started/manage-desktop",
      name: "Manage Synnax Desktop",
      flag: "desktop",
      edition: "desktop",
    },
  ],
};
