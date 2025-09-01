// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav/PageNav";

export const CLUSTER_NAV: PageNavNode = {
  key: "cluster",
  name: "Cluster",
  children: [
    {
      key: "/reference/cluster/quick-start",
      href: "/reference/cluster/quick-start",
      name: "Quick Start",
    },
    {
      key: "/reference/cluster/installation",
      href: "/reference/cluster/installation",
      name: "Installation",
    },
    {
      key: "/reference/cluster/production",
      href: "/reference/cluster/production",
      name: "Production",
    },
    {
      key: "/reference/cluster/requirements",
      href: "/reference/cluster/requirements",
      name: "Requirements",
    },
    {
      key: "/reference/cluster/cli-reference",
      href: "/reference/cluster/cli-reference",
      name: "CLI Reference",
    },
    {
      key: "/reference/cluster/systemd-service",
      href: "/reference/cluster/systemd-service",
      name: "Systemd Service",
    },
  ],
};
