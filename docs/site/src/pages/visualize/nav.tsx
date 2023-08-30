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

export const visualizeNav: PageNavNode = {
  key: "visualize",
  name: "Visualize",
  icon: <Icon.Visualize />,
  children: [
    {
      key: "/visualize/get-started",
      href: "/visualize/get-started",
      name: "Get Started",
    },
    {
      key: "/visualize/connect-a-cluster",
      href: "/visualize/connect-a-cluster",
      name: "Connect a Cluster",
    },
    {
      key: "/visualize/define-a-range",
      href: "/visualize/define-a-range",
      name: "Define a Range",
    },
    {
      key: "/visualize/create-a-visualization",
      href: "/visualize/create-a-visualization",
      name: "Create a Visualization",
    },
  ],
};
