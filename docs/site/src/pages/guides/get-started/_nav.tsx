// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav/PageNav";

export const getStartedNav: PageNavNode = {
  key: "get-started",
  name: "Get Started",
  children: [
    // {
    //   key: "/guides/get-started/introduction",
    //   href: "/guides/get-started/introduction",
    //   name: "Introduction",
    // },
    {
      key: "/guides/get-started/installation",
      href: "/guides/get-started/installation",
      name: "Installation",
    },
    // {
    //   key: "/guides/get-started/acquiring-data",
    //   href: "/guides/get-started/acquiring-data",
    //   name: "Acquiring Data",
    // },
    // {
    //   key: "/guides/get-started/manual-control",
    //   href: "/guides/get-started/manual-control",
    //   name: "Manual Control",
    // },
    // {
    //   key: "/guides/get-started/control-sequences",
    //   href: "/guides/get-started/control-sequences",
    //   name: "Control Sequences",
    // },
    // {
    //   key: "/guides/get-started/data-review",
    //   href: "/guides/get-started/data-review",
    //   name: "Data Review",
    // },
    // {
    //   key: "/guides/get-started/workspaces",
    //   href: "/guides/get-started/workspaces",
    //   name: "Workspaces",
    // },
  ],
};
