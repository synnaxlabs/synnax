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
      key: "/reference/ranges/create",
      href: "/reference/ranges/create",
      name: "Create ranges",
    },
    {
      key: "/reference/ranges/find",
      href: "/reference/ranges/find",
      name: "Find ranges",
    },
    {
      key: "/reference/ranges/edit",
      href: "/reference/ranges/edit",
      name: "Edit and delete ranges",
    },
    {
      key: "/reference/ranges/metadata",
      href: "/reference/ranges/metadata",
      name: "Metadata and labels",
    },
    {
      key: "/reference/ranges/aliases",
      href: "/reference/ranges/aliases",
      name: "Channel aliases",
    },
    {
      key: "/reference/ranges/data",
      href: "/reference/ranges/data",
      name: "Read and write range data",
    },
    {
      key: "/reference/ranges/export",
      href: "/reference/ranges/export",
      name: "Export data",
    },
  ],
};
