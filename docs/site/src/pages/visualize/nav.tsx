// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { PageNavLeaf } from "@/components/PageNav";

export const visualizeNav: PageNavLeaf = {
  key: "visualize",
  name: "Visualize",
  icon: <Icon.Visualize />,
  children: [
    {
      key: "/visualize/get-started",
      url: "/visualize/get-started",
      name: "Get Started",
    },
    {
      key: "/visualize/connect-a-cluster",
      url: "/visualize/connect-a-cluster",
      name: "Connect a Cluster",
    },
    {
      key: "/visualize/define-a-range",
      url: "/visualize/define-a-range",
      name: "Define a Range",
    },
    {
      key: "/visualize/create-a-visualization",
      url: "/visualize/create-a-visualization",
      name: "Create a Visualization",
    },
  ],
};
