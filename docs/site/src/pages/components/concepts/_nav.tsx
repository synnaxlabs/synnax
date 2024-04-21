// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { PageNavNode } from "@/components/PageNav/PageNav";

export const conceptsNav: PageNavNode = {
  key: "concepts",
  name: "Concepts",
  children: [
    {
      key: "/components/concepts/overview",
      href: "/components/concepts/overview",
      name: "Overview",
    },
    {
      key: "/components/concepts/clusters-and-nodes",
      href: "/components/concepts/clusters-and-nodes",
      name: "Clusters and Nodes",
    },
    {
      key: "/components/concepts/channels",
      href: "/components/concepts/channels",
      name: "Channels",
    },
    {
      key: "/components/concepts/write-domains",
      href: "/components/concepts/write-domains",
      name: "Write Domains",
    },
    {
      key: "/components/concepts/read-ranges",
      href: "/components/concepts/read-ranges",
      name: "Read Ranges",
    },
  ],
};
