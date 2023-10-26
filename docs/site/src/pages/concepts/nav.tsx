// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { PageNavNode } from "@/components/PageNav";

export const conceptsNav: PageNavNode = {
  key: "concepts",
  name: "Concepts",
  children: [
    {
      key: "/concepts/overview",
      href: "/concepts/overview",
      name: "Overview",
    },
    {
      key: "/concepts/clusters-and-nodes",
      href: "/concepts/clusters-and-nodes",
      name: "Clusters and Nodes",
    },
    {
      key: "/concepts/channels",
      href: "/concepts/channels",
      name: "Channels",
    },
    {
      key: "/concepts/arrays-and-frames",
      href: "/concepts/arrays-and-frames",
      name: "Arrays and Frames",
    },
    {
      key: "/concepts/write-domains",
      href: "/concepts/write-domains",
      name: "Write Domains",
    },
    {
      key: "/concepts/read-ranges",
      href: "/concepts/read-ranges",
      name: "Read Ranges",
    },
  ],
};
