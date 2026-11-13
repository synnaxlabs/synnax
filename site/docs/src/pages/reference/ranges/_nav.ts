// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Nav } from "@/components/nav";

export const RANGES_NAV: Nav.PageNavNode = {
  key: "ranges",
  name: "Ranges",
  children: [
    {
      href: "/reference/ranges",
      name: "Overview",
    },
    {
      href: "/reference/ranges/create",
      name: "Create ranges",
    },
    {
      href: "/reference/ranges/arc",
      name: "Arc ranges",
    },
    {
      href: "/reference/ranges/find",
      name: "Find ranges",
    },
    {
      href: "/reference/ranges/edit",
      name: "Edit and delete ranges",
    },
    {
      href: "/reference/ranges/metadata",
      name: "Metadata and labels",
    },
    {
      href: "/reference/ranges/aliases",
      name: "Channel aliases",
    },
    {
      href: "/reference/ranges/snapshots",
      name: "Snapshots",
    },
    {
      href: "/reference/ranges/data",
      name: "Read and write range data",
    },
    {
      href: "/reference/ranges/export",
      name: "Export data",
    },
    {
      href: "/reference/ranges/reference",
      name: "Reference",
    },
  ],
};
