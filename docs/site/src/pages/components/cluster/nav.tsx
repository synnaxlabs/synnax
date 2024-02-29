// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav";

export const clusterNav: PageNavNode = {
  key: "cluster",
  name: "Cluster",
  children: [
    {
      key: "/components/cluster/quick-start",
      href: "/components/cluster/quick-start",
      name: "Quick Start",
    },
    {
      key: "/components/cluster/requirements",
      href: "/components/cluster/requirements",
      name: "Requirements",
    },
    {
      key: "/components/cluster/cli-reference",
      href: "/components/cluster/cli-reference",
      name: "CLI Reference",
    },
    {
      key: "/components/cluster/systemd-service",
      href: "/components/cluster/systemd-service",
      name: "Systemd Service",
    },
  ],
};
