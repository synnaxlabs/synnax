// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/Page";

export const RANGES_NAV: PageNavNode = {
  key: "ranges",
  name: "Ranges",
  children: [
    {
      key: "/reference/ranges",
      href: "/reference/ranges",
      name: "Overview",
    },
    {
      key: "/reference/ranges/manage",
      href: "/reference/ranges/manage",
      name: "Manage ranges",
    },
    {
      key: "/reference/ranges/export",
      href: "/reference/ranges/export",
      name: "Export data",
    },
  ],
};
