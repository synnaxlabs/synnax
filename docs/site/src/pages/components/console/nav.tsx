// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav";

export const consoleNav: PageNavNode = {
  key: "console",
  name: "Console",
  children: [
    {
      key: "/console/get-started",
      href: "/console/get-started",
      name: "Get Started",
    },
    {
      key: "/console/connect-a-cluster",
      href: "/console/connect-a-cluster",
      name: "Connect a Cluster",
    },
    {
      key: "/console/querying-data",
      href: "/console/querying-data",
      name: "Querying Data Using Ranges",
    },
  ],
};
