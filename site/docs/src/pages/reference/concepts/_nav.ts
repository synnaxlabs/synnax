// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/tree";

export const CONCEPTS_NAV: PageNavNode = {
  key: "concepts",
  name: "Concepts",
  children: [
    {
      href: "/reference/concepts/overview",
      name: "Overview",
    },
    {
      href: "/reference/concepts/clusters-and-nodes",
      name: "Clusters and nodes",
    },
    {
      href: "/reference/concepts/channels",
      name: "Channels",
    },
    {
      href: "/reference/concepts/ranges",
      name: "Ranges",
    },
    {
      href: "/reference/concepts/reads",
      name: "Reads",
    },
    {
      href: "/reference/concepts/writes",
      name: "Writes",
    },
    {
      href: "/reference/concepts/streams",
      name: "Streams",
    },
  ],
};
