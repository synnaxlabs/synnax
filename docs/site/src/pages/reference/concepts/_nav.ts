// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/nav/Page";

export const CONCEPTS_NAV: PageNavNode = {
  key: "concepts",
  name: "Concepts",
  icon: "Concepts",
  children: [
    {
      key: "/reference/concepts/overview",
      href: "/reference/concepts/overview",
      name: "Overview",
    },
    {
      key: "/reference/concepts/clusters-and-nodes",
      href: "/reference/concepts/clusters-and-nodes",
      name: "Clusters and Nodes",
    },
    {
      key: "/reference/concepts/channels",
      href: "/reference/concepts/channels",
      name: "Channels",
    },
    {
      key: "/reference/concepts/ranges",
      href: "/reference/concepts/ranges",
      name: "Ranges",
    },
    {
      key: "/reference/concepts/reads",
      href: "/reference/concepts/reads",
      name: "Reads",
    },
    {
      key: "/reference/concepts/writes",
      href: "/reference/concepts/writes",
      name: "Writes",
    },
    {
      key: "/reference/concepts/streams",
      href: "/reference/concepts/streams",
      name: "Streams",
    },
  ],
};
